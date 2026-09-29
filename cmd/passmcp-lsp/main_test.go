// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"satellion.com/passmcp-lsp/internal/jsonrpc"
)

func runArgs(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := run(context.Background(), args, strings.NewReader(stdin), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestVersionAndUsage(t *testing.T) {
	if code, out, _ := runArgs(t, "", "--version"); code != 0 || out != "passmcp-lsp dev\n" {
		t.Fatalf("%d %q", code, out)
	}
	if code, _, errOut := runArgs(t, "", "--help"); code != 0 || !strings.Contains(errOut, "passmcp-lsp check") {
		t.Fatalf("help goes to stderr: %d %q", code, errOut)
	}
	if code, _, _ := runArgs(t, "", "--nope"); code != 2 {
		t.Fatal("an unknown flag is a usage error")
	}
	if code, _, errOut := runArgs(t, "", "serve"); code != 2 || !strings.Contains(errOut, `unexpected argument "serve"`) {
		t.Fatalf("%d %q", code, errOut)
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("PASSMCP_LSP_TEST", "")
	if envOr("PASSMCP_LSP_TEST", "d") != "d" {
		t.Fatal("empty is unset")
	}
	t.Setenv("PASSMCP_LSP_TEST", "v")
	if envOr("PASSMCP_LSP_TEST", "d") != "v" {
		t.Fatal("set wins")
	}
}

func frames(msgs ...string) string {
	var b bytes.Buffer
	w := jsonrpc.NewWriter(&b)
	for _, m := range msgs {
		_ = w.Write([]byte(m))
	}
	return b.String()
}

func TestServe(t *testing.T) {
	in := frames(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"initializationOptions":{"passmcpPath":"/nowhere/passmcp"}}}`,
		`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{"textDocument":{"uri":"file:///r/x.policy.json","version":1,"text":"{\"version\":1,\"name\":\"x\",\"must_pass\":[\"net.tls\"]}"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{"textDocument":{"uri":"file:///r/x.policy.json"},"position":{"line":0,"character":38}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"shutdown"}`, `{"jsonrpc":"2.0","method":"exit"}`)
	code, out, errOut := runArgs(t, in, "--stdio")
	if code != 0 {
		t.Fatalf("orderly exit: %d %s", code, errOut)
	}
	if !strings.Contains(out, `"name":"passmcp-lsp"`) || !strings.Contains(out, "Install passmcp") {
		t.Fatalf("the client's passmcpPath is used, and a missing passmcp is said so: %s", out)
	}
	if code, _, errOut := runArgs(t, frames(`{"jsonrpc":"2.0","method":"exit"}`)); code != 1 || !strings.Contains(errOut, "without a shutdown") {
		t.Fatalf("exit without shutdown: %d %q", code, errOut)
	}
}

func TestCompletion(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		code, out, _ := runArgs(t, "", "--completion", shell)
		if code != 0 || !strings.Contains(out, "passmcp-lsp") || !strings.Contains(out, "check") {
			t.Errorf("%s: %d %q", shell, code, out)
		}
		for _, flag := range []string{"version", "stdio", "passmcp", "completion"} {
			if !strings.Contains(out, flag) {
				t.Errorf("%s script lacks %s", shell, flag)
			}
		}
		syntaxCheck(t, shell, out)
	}
	if code, _, errOut := runArgs(t, "", "--completion", "tcsh"); code != 2 || !strings.Contains(errOut, "bash, zsh or fish") {
		t.Fatal("an unknown shell is a usage error")
	}
}

// syntaxCheck parses a script with its shell, when the shell is installed.
// Not on Windows, where a bash on PATH is usually Git's or WSL's and reads
// paths differently.
func syntaxCheck(t *testing.T, shell, script string) {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err != nil || runtime.GOOS == "windows" {
		return
	}
	f := filepath.Join(t.TempDir(), "c")
	_ = os.WriteFile(f, []byte(script), 0o600)
	if b, err := exec.Command(path, "-n", f).CombinedOutput(); err != nil {
		t.Errorf("%s -n: %v %s", shell, err, b)
	}
}

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheckText(t *testing.T) {
	dir := t.TempDir()
	good := writeFile(t, dir, "ok.policy.json", `{"version": 1, "name": "x", "max_fail": 0}`)
	bad := writeFile(t, dir, "bad.policy.json", "{\"version\": 1,\n \"name\": \"x\", \"max_fail\": -1}")
	warn := writeFile(t, dir, ".vscode/mcp.json", `{"mcpServers": {}}`)
	code, out, _ := runArgs(t, "", "check", good, bad, warn)
	if code != 1 {
		t.Fatalf("an error fails the check: %d", code)
	}
	for _, want := range []string{good + ": ok (passmcp-policy)", bad + ":2:27: error: max_fail is -1", "[policy/limit]", warn + ":1:2: warning:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if code, _, _ := runArgs(t, "", "check", good, warn); code != 0 {
		t.Fatal("warnings do not fail the check")
	}
}

func TestCheckJSON(t *testing.T) {
	dir := t.TempDir()
	f := writeFile(t, dir, "server.json", `{"name": "a/b", "description": "d"}`)
	u := writeFile(t, dir, "package.json", `{"name": "x"}`)
	code, out, _ := runArgs(t, "", "check", "--output", "json", f, u)
	var reports []fileReport
	if err := json.Unmarshal([]byte(out), &reports); err != nil || code != 1 {
		t.Fatalf("%d %v %s", code, err, out)
	}
	if reports[0].Kind != "server.json" || reports[1].Kind != "unrecognised" || reports[1].Diagnostics == nil {
		t.Fatalf("%+v", reports)
	}
	if !hasDiagnostic(reports[0].Diagnostics, "server-json/schema", `"version"`, 1) {
		t.Fatalf("%+v", reports[0].Diagnostics)
	}
}

func hasDiagnostic(ds []diagnosticJS, code, text string, line int) bool {
	for _, d := range ds {
		if d.Code == code && strings.Contains(d.Message, text) && d.Line == line {
			return true
		}
	}
	return false
}

func TestCheckUsage(t *testing.T) {
	for _, args := range [][]string{{"check"}, {"check", "--output", "xml", "f"}, {"check", "--bad"}, {"check", filepath.Join(t.TempDir(), "missing.json")}} {
		if code, out, _ := runArgs(t, "", args...); code != 2 || out != "" {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }

func TestCheckWriteFailure(t *testing.T) {
	f := writeFile(t, t.TempDir(), "x.json", `{}`)
	for _, format := range []string{"text", "json"} {
		var errOut bytes.Buffer
		if code := run(context.Background(), []string{"check", "--output", format, f}, nil, failWriter{}, &errOut); code != 2 {
			t.Errorf("%s: %d", format, code)
		}
	}
	bad := writeFile(t, t.TempDir(), "y.policy.json", `{}`)
	var errOut bytes.Buffer
	if code := run(context.Background(), []string{"check", bad}, nil, failWriter{}, &errOut); code != 2 {
		t.Errorf("a failed write of a diagnostic: %d", code)
	}
}
