// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// codes returns the diagnostics' codes, severity-tagged: "error:policy/name".
func codes(ds []Diagnostic) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = d.Severity.String() + ":" + d.Code
	}
	return out
}

func has(ds []Diagnostic, sev Severity, code, text string) bool {
	for _, d := range ds {
		if d.Severity == sev && d.Code == code && strings.Contains(d.Message, text) {
			return true
		}
	}
	return false
}

func errorsIn(ds []Diagnostic) int {
	n := 0
	for _, d := range ds {
		if d.Severity == Error {
			n++
		}
	}
	return n
}

func TestSeverityString(t *testing.T) {
	for s, want := range map[Severity]string{Error: "error", Warning: "warning", Information: "info", Hint: "hint", 9: "unknown"} {
		if s.String() != want {
			t.Errorf("%d: %q", s, s.String())
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name, src string
		want      Kind
	}{
		{"/x/server.json", `{}`, ServerJSON},
		{"listing.json", `{"$schema": "` + ServerSchemaURL + `"}`, ServerJSON},
		{"/home/a/Library/claude_desktop_config.json", `{}`, ClientConfig},
		{"/repo/.mcp.json", `{}`, ClientConfig},
		{"C:\\repo\\.vscode\\mcp.json", `{}`, ClientConfig},
		{"/repo/.cursor/mcp.json", `{}`, ClientConfig},
		{"/repo/mcp.json", `{"mcpServers": {}}`, ClientConfig},
		{"/repo/other/mcp.json", `{}`, Unknown},
		{"company.policy.json", `{}`, Policy},
		{"passmcp-policy.json", `{}`, Policy},
		{"company.json", `{"version": 1, "must_pass": []}`, Policy},
		{"a.json", `{"_type": "x", "predicateType": "https://satellion.com/attestation/mcp-evaluation/v1"}`, Attestation},
		{"a.json", `{"predicateType": "https://satellion.com/attestation/mcp-evaluation/v1"}`, Unknown},
		{"t.json", `{"name": "t", "inputSchema": {"type": "object"}}`, Tools},
		{"t.json", `[{"name": "t", "inputSchema": {"type": "object"}}]`, Tools},
		{"t.json", `{"tools": [{"name": "t", "inputSchema": {"type": "object"}}]}`, Tools},
		{"t.json", `[]`, Unknown},
		{"package.json", `{"name": "x"}`, Unknown},
	}
	for _, c := range cases {
		if got := Analyze(c.name, []byte(c.src)).Kind; got != c.want {
			t.Errorf("Analyze(%q, %s).Kind = %q, want %q", c.name, c.src, got, c.want)
		}
	}
}

func TestSyntaxErrors(t *testing.T) {
	r := Analyze("server.json", []byte(`{"name": "x",}`))
	if len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "syntax" || r.Root != nil || r.Diagnostics[0].Start != 13 {
		t.Fatalf("trailing comma in server.json: %+v", r)
	}
	// A comment is fine in a client configuration, and an error elsewhere.
	jsonc := "{\n  // mine\n  \"servers\": {\"a\": {\"command\": \"x\"},},\n}"
	if r := Analyze("/r/.vscode/mcp.json", []byte(jsonc)); len(r.Diagnostics) != 0 || r.Kind != ClientConfig {
		t.Fatalf("JSONC client configuration: %+v", r.Diagnostics)
	}
}

func TestSyntaxErrorsByDialect(t *testing.T) {
	if r := Analyze("/r/x.policy.json", []byte("// no\n{}")); !has(r.Diagnostics, Error, "syntax", "comments are not allowed") {
		t.Fatalf("comment in a policy: %+v", r.Diagnostics)
	}
	if r := Analyze("/r/.mcp.json", []byte("{ /* open")); !has(r.Diagnostics, Error, "syntax", "never closed") {
		t.Fatalf("broken JSONC: %+v", r.Diagnostics)
	}
	if r := Analyze("x.json", []byte("{")); len(r.Diagnostics) != 0 || r.Kind != Unknown {
		t.Fatalf("an unrecognised file's syntax is not this tool's business: %+v", r)
	}
	if r := Analyze("server.json", []byte("")); r.Diagnostics[0].Start != 0 || r.Diagnostics[0].End != 0 {
		t.Fatalf("an empty document's error range must stay inside it: %+v", r.Diagnostics)
	}
	if d := syntax(errors.New("plain"), 3); d.Message != "plain" {
		t.Fatal(d)
	}
}

func TestDuplicateKeys(t *testing.T) {
	if r := Analyze("x.json", []byte(`{"a": 1, "a": 2}`)); len(r.Diagnostics) != 0 {
		t.Fatalf("an unrecognised file gets nothing: %v", r.Diagnostics)
	}
	r := Analyze("/r/.mcp.json", []byte(`{"a": 1, "b": [{"c": 1, "c": 2}], "a": 3, "mcpServers": {}}`))
	if len(r.Diagnostics) != 2 || !has(r.Diagnostics, Warning, "duplicate-key", `"c"`) || !has(r.Diagnostics, Warning, "duplicate-key", `"a"`) {
		t.Fatalf("%v", codes(r.Diagnostics))
	}
}

func TestServerJSON(t *testing.T) {
	valid := `{
  "$schema": "` + ServerSchemaURL + `",
  "name": "io.github.example/weather",
  "description": "Weather forecasts.",
  "version": "1.0.2",
  "packages": [{"registryType": "npm", "identifier": "@example/weather", "version": "1.0.2", "transport": {"type": "stdio"}}]
}`
	if r := Analyze("server.json", []byte(valid)); len(r.Diagnostics) != 0 {
		t.Fatalf("valid listing: %+v", r.Diagnostics)
	}
	r := Analyze("server.json", []byte(`{"name": "no-slash", "description": "d", "version": 1,
	  "packages": [{"registryType": "npm", "identifier": "x", "version": "latest", "transport": {"type": "stdio"}}]}`))
	if !has(r.Diagnostics, Information, "server-json/schema-version", "no $schema") ||
		!has(r.Diagnostics, Error, "server-json/schema", "name: must match the pattern") ||
		!has(r.Diagnostics, Error, "server-json/schema", "version: must be string, not number") ||
		!has(r.Diagnostics, Error, "server-json/schema", `packages[0].version: must not be "latest"`) {
		t.Fatalf("%+v", r.Diagnostics)
	}
	r = Analyze("server.json", []byte(`{"$schema": "https://static.modelcontextprotocol.io/schemas/2025-09-29/server.schema.json", "name": "a/b", "description": "d", "version": "1"}`))
	if !has(r.Diagnostics, Information, "server-json/schema-version", "names a schema other than") {
		t.Fatalf("%+v", r.Diagnostics)
	}
}

func TestEmbeddedSchemaIsTheRegistrys(t *testing.T) {
	root, err := jsondoc.Parse(ServerSchema(), jsondoc.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if id := root.Get("$id").Str; id != ServerSchemaURL || !strings.Contains(id, ServerSchemaVersion) {
		t.Fatalf("the embedded schema's $id is %q", id)
	}
	if _, err := serverSchema(); err != nil {
		t.Fatalf("every keyword of the embedded schema must be one the validator enforces: %v", err)
	}
}

func TestClientConfig(t *testing.T) {
	src := `{"mcpServers": {
  "ok": {"command": "npx", "args": ["-y", "pkg"], "env": {"API_KEY": "${env:API_KEY}", "MODE": "x"}},
  "remote": {"url": "https://mcp.example.com/mcp", "headers": {"Authorization": "Bearer abc123"}},
  "none": {},
  "both": {"command": "x", "url": "https://a.example/mcp"},
  "badurl": {"url": "ftp://a.example"},
  "badargs": {"command": "x", "args": "one", "env": [1]},
  "badarg": {"command": "x", "args": [1], "headers": {"X": 2}},
  "blank": {"command": "  "},
  "str": "nope"
}}`
	r := Analyze("/u/claude_desktop_config.json", []byte(src))
	d := r.Diagnostics
	for _, want := range []struct {
		sev        Severity
		code, text string
	}{
		{Warning, "client/literal-credential", "headers.Authorization"},
		{Error, "client/entry", `server "none" has neither`},
		{Warning, "client/command-and-url", `server "both"`},
		{Error, "client/url", `server "badurl"`},
		{Error, "client/entry", `args must be an array`},
		{Error, "client/entry", `env must be an object`},
		{Error, "client/entry", `args must hold only strings`},
		{Error, "client/entry", `headers.X must be a string`},
		{Error, "client/entry", `server "blank": command must be a non-empty string`},
		{Error, "client/entry", `server "str" must be an object`},
	} {
		if !has(d, want.sev, want.code, want.text) {
			t.Errorf("missing %s %s %q in %v", want.sev, want.code, want.text, d)
		}
	}
	if has(d, Warning, "client/literal-credential", "API_KEY") || has(d, Error, "", `"ok"`) {
		t.Errorf("a variable reference is not a literal credential: %v", d)
	}
}

func TestClientConfigLayouts(t *testing.T) {
	cases := []struct {
		name, src, code, text string
	}{
		{"/r/.vscode/mcp.json", `{"mcpServers": {}}`, "client/wrong-layout", `VS Code's .vscode/mcp.json names its servers under "servers"`},
		{"/r/.cursor/mcp.json", `{"servers": {}}`, "client/wrong-layout", `mcp.json names its servers under "mcpServers"`},
		{"/r/.mcp.json", `{"other": 1}`, "client/no-servers", "names no MCP servers"},
		{"/r/.mcp.json", `{"mcpServers": []}`, "client/no-servers", "must be an object"},
		{"/r/.mcp.json", `[]`, "client/no-servers", "is a JSON object"},
	}
	for _, c := range cases {
		r := Analyze(c.name, []byte(c.src))
		if len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != c.code || !strings.Contains(r.Diagnostics[0].Message, c.text) {
			t.Errorf("%s %s: %+v", c.name, c.src, r.Diagnostics)
		}
	}
	if r := Analyze("/r/.vscode/mcp.json", []byte(`{"servers": {"a": {"type": "http", "url": "http://127.0.0.1:3000/mcp"}}}`)); len(r.Diagnostics) != 0 {
		t.Errorf("VS Code layout: %+v", r.Diagnostics)
	}
}

func TestPolicyFixturesAgreeWithPassmcp(t *testing.T) {
	// The fixtures' verdicts were read from passmcp 0.0.1 itself
	// (make crosscheck repeats that): accept/ is every policy passmcp
	// applies, refuse/ every policy it refuses.
	for dir, wantErrors := range map[string]bool{"accept": false, "refuse": true} {
		files, _ := filepath.Glob(filepath.Join("testdata", "policy", dir, "*.json"))
		if len(files) == 0 {
			t.Fatalf("no fixtures in %s", dir)
		}
		for _, f := range files {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			r := Analyze("fixture.policy.json", src)
			if got := errorsIn(r.Diagnostics) > 0; got != wantErrors {
				t.Errorf("%s: errors=%v, want %v: %v", f, got, wantErrors, r.Diagnostics)
			}
		}
	}
}

func TestPolicyMessages(t *testing.T) {
	for file, want := range map[string]string{
		"unknown-key":           `"must_pas" is not a key of a policy`,
		"later-version":         "passmcp 0.0.2 implements policy format 1",
		"float-version":         "version must be a whole number",
		"no-name":               "the policy has no name",
		"blank-name":            "the name is empty",
		"no-version":            "the policy has no version",
		"bad-transport":         `transport "websocket" is neither http nor stdio`,
		"empty-target":          "constrains nothing",
		"target-unknown-key":    `"host" is not a key of target`,
		"negative-allowance":    "a negative allowance",
		"fractional-allowance":  "max_fail must be a whole number",
		"score-range":           "min_score is 101",
		"category-range":        "min_category_score for authorization is -1",
		"bad-severity":          `forbid_severity "high"`,
		"exemption-no-reason":   "has no reason",
		"exemption-no-expiry":   "has no expiry",
		"exemption-bad-date":    "not a date in YYYY-MM-DD form",
		"exemption-no-check":    "names no check",
		"exemption-twice":       "exempted twice",
		"exemption-unknown-key": `"owner" is not a key of an exemption`,
		"required-and-exempted": "both required and exempted",
		"empty-check-id":        "empty check id",
		"check-id-twice":        "names a.b twice",
		"wrong-type":            "must_pass must be an array",
		"not-an-object":         "a policy is a JSON object",
	} {
		src, err := os.ReadFile(filepath.Join("testdata", "policy", "refuse", file+".json"))
		if err != nil {
			t.Fatal(err)
		}
		r := Analyze("x.policy.json", src)
		found := false
		for _, d := range r.Diagnostics {
			found = found || (d.Severity == Error && strings.Contains(d.Message, want))
		}
		if !found {
			t.Errorf("%s: no error containing %q in %v", file, want, r.Diagnostics)
		}
	}
}

func TestPolicyEdges(t *testing.T) {
	r := Analyze("x.policy.json", []byte(`{"version": 1, "name": "x", "exemptions": [{"check": "perf.latency", "reason": "r", "expires": "2020-01-31"}]}`))
	if !has(r.Diagnostics, Warning, "policy/expired", "expired on 2020-01-31") || !has(r.Diagnostics, Information, "policy/no-rules", "states no rule") {
		t.Fatalf("%+v", r.Diagnostics)
	}
	for _, src := range []string{
		`{"version": -1, "name": "x"}`, `{"version": 1, "name": 5}`, `{"version": 1, "name": "x", "target": 1}`,
		`{"version": 1, "name": "x", "target": {"endpoint": 5}}`, `{"version": 1, "name": "x", "min_score": "high"}`,
		`{"version": 1, "name": "x", "min_category_score": {"": 50}}`, `{"version": 1, "name": "x", "min_category_score": []}`,
		`{"version": 1, "name": "x", "forbid_severity": 1}`, `{"version": 1, "name": "x", "exemptions": {}}`,
		`{"version": 1, "name": "x", "exemptions": [1]}`, `{"version": 1, "name": "x", "must_pass": [1]}`,
	} {
		if errorsIn(Analyze("x.policy.json", []byte(src)).Diagnostics) == 0 {
			t.Errorf("%s must be refused", src)
		}
	}
	for _, src := range []string{
		`{"version": 1, "name": "x", "min_category_score": {"auth": 50}}`, `{"version": 1, "name": "x", "forbid_severity": ""}`,
		`{"version": 1, "name": "x", "target": {"endpoint": "https://a.example/mcp", "transport": null}}`,
	} {
		if r := Analyze("x.policy.json", []byte(src)); errorsIn(r.Diagnostics) != 0 {
			t.Errorf("%s must be accepted: %v", src, r.Diagnostics)
		}
	}
}

func TestAttestation(t *testing.T) {
	for _, f := range []string{"mcp-statement.json", "a2a-statement.json"} {
		src, err := os.ReadFile(filepath.Join("testdata", "attestation", f))
		if err != nil {
			t.Fatal(err)
		}
		r := Analyze(f, src)
		if r.Kind != Attestation || len(r.Diagnostics) != 0 {
			t.Errorf("%s: kind %q, %+v", f, r.Kind, r.Diagnostics)
		}
		// Editing the endpoint the predicate names, and not the subject,
		// breaks the digest.
		tampered := strings.Replace(string(src), `"endpoint": "https://`, `"endpoint": "https://tampered.`, 1)
		if r := Analyze(f, []byte(tampered)); !has(r.Diagnostics, Error, "attestation/invalid", "digest does not cover") {
			t.Errorf("%s tampered: %+v", f, r.Diagnostics)
		}
	}
}

func TestAttestationProblems(t *testing.T) {
	r := Analyze("a.json", []byte(`{"_type": "https://in-toto.io/Statement/v1", "predicateType": "https://satellion.com/attestation/other/v1"}`))
	if !has(r.Diagnostics, Warning, "attestation/predicate-type", "not one passmcp-reporting 0.0.2 verifies") {
		t.Fatalf("%+v", r.Diagnostics)
	}
	src := `{"_type": "x", "subject": [], "predicateType": "https://satellion.com/attestation/mcp-evaluation/v1", "predicate": {"ranAt": "2026-01-01T00:00:00Z"}}`
	r = Analyze("a.json", []byte(src))
	if errorsIn(r.Diagnostics) < 5 {
		t.Fatalf("each problem is its own diagnostic: %v", r.Diagnostics)
	}
	for _, d := range r.Diagnostics {
		if strings.HasPrefix(d.Message, "_type") && src[d.Start:d.End] != `"_type"` {
			t.Errorf("the _type problem must underline the _type key, got %q", src[d.Start:d.End])
		}
	}
}

func TestProblems(t *testing.T) {
	got := problems("attestation: a; b;  ; c")
	if strings.Join(got, "|") != "a|b|c" {
		t.Fatal(got)
	}
	root, _ := jsondoc.Parse([]byte(`{"x": 1}`), jsondoc.Options{})
	if n := locate(root, "nothing matches this"); n.Start != 0 || n.End != 1 {
		t.Fatal("fallback is the first character")
	}
	if n := locate(root, "the target has no endpoint"); n.Start != 0 || n.End != 1 {
		t.Fatal("a missing anchor falls back too")
	}
}
