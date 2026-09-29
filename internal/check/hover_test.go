// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIDAtPolicy(t *testing.T) {
	src := `{"version": 1, "name": "net.tls", "must_pass": ["net.tls"], "must_not_fail": ["catalog.tools.descriptions"],
	 "exemptions": [{"check": "perf.latency", "reason": "auth.scheme", "expires": "2099-01-01"}]}`
	r := Analyze("x.policy.json", []byte(src))
	cases := []struct {
		at, id string
		ok     bool
	}{
		{`"net.tls"]`, "net.tls", true},
		{`"catalog.tools`, "catalog.tools.descriptions", true},
		{`"perf.latency"`, "perf.latency", true},
		{`"net.tls", "must`, "", false}, // the policy's name
		{`"auth.scheme"`, "", false},    // a reason
		{`"must_pass"`, "", false},      // a key
		{`1, "name"`, "", false},        // a number
	}
	for _, c := range cases {
		off := strings.Index(src, c.at) + 1
		ref, ok := IDAt(r, off)
		if ok != c.ok || ref.ID != c.id {
			t.Errorf("at %q: %+v %v", c.at, ref, ok)
		}
		if ok && src[ref.Start:ref.End] != `"`+c.id+`"` {
			t.Errorf("range %q", src[ref.Start:ref.End])
		}
	}
}

func TestIDAtAttestation(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "attestation", "mcp-statement.json"))
	if err != nil {
		t.Fatal(err)
	}
	withDoc := strings.Replace(string(src), `"phase": "protocol",`, `"phase": "protocol", "doc": "https://satellion.com/passmcp/docs/checks/#check-protocol-origin",`, 1)
	r := Analyze("a.json", []byte(withDoc))
	ref, ok := IDAt(r, strings.Index(withDoc, `"protocol.origin"`)+2)
	if !ok || ref.ID != "protocol.origin" || !strings.HasSuffix(ref.Doc, "#check-protocol-origin") {
		t.Fatalf("%+v %v", ref, ok)
	}
	if _, ok := IDAt(r, strings.Index(withDoc, `"auth"`)+2); ok {
		t.Fatal("a phase is not a check id")
	}
	if _, ok := IDAt(Analyze("server.json", []byte(`{"name": "a.b"}`)), 10); ok {
		t.Fatal("only policies and attestations carry check ids")
	}
	if _, ok := IDAt(Result{Kind: Policy}, 0); ok {
		t.Fatal("no tree, no id")
	}
}
