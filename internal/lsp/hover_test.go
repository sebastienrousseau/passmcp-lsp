// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

func hoverAt(id int, uri string, line, char int) string {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "textDocument/hover",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}, "position": map[string]int{"line": line, "character": char}}})
	return string(b)
}

func hoverText(t *testing.T, m map[string]any) string {
	t.Helper()
	if m == nil {
		t.Fatal("no response")
	}
	res, ok := m["result"].(map[string]any)
	if !ok {
		return ""
	}
	return res["contents"].(map[string]any)["value"].(string)
}

func TestHover(t *testing.T) {
	uri := "file:///repo/company.policy.json"
	text := `{"version": 1, "name": "x", "must_pass": ["net.tls", "unknown.check", "absent.check", "broken.check"]}`
	col := func(s string) int { return strings.Index(text, s) + 1 }
	msgs, err := session(t, Options{}, initReq, open(uri, text),
		hoverAt(10, uri, 0, col(`"net.tls"`)),
		hoverAt(11, uri, 0, col(`"unknown.check"`)),
		hoverAt(12, uri, 0, col(`"absent.check"`)),
		hoverAt(13, uri, 0, col(`"broken.check"`)),
		hoverAt(14, uri, 0, col(`"version"`)),
		hoverAt(15, "file:///closed.json", 0, 0),
		shutdown, exit)
	if err != nil {
		t.Fatal(err)
	}
	found := byID(msgs, 10)
	if md := hoverText(t, found); !strings.Contains(md, "TLS protects the token.") || !strings.Contains(md, "1. **Serve https**") {
		t.Fatalf("guidance: %s", md)
	}
	rng := found["result"].(map[string]any)["range"].(map[string]any)["start"].(map[string]any)
	if rng["character"] != float64(col(`"net.tls"`)-1) {
		t.Fatalf("the hover range is the id: %v", rng)
	}
	for id, want := range map[float64]string{11: "no guidance for this check id", 12: "Install passmcp", 13: "could not be asked"} {
		if md := hoverText(t, byID(msgs, id)); !strings.Contains(md, want) {
			t.Errorf("hover %v: %q lacks %q", id, md, want)
		}
	}
	for _, id := range []float64{14, 15} {
		if m := byID(msgs, id); m == nil || m["result"] != nil {
			t.Errorf("hover %v off a check id answers null: %v", id, m)
		}
	}
}

func TestHoverWithoutGuidance(t *testing.T) {
	uri := "file:///repo/company.policy.json"
	text := `{"version": 1, "name": "x", "must_pass": ["net.tls"]}`
	msgs, _ := session(t, Options{Guidance: func(string) Looker { return nil }}, initReq, open(uri, text),
		hoverAt(10, uri, 0, strings.Index(text, `"net.tls"`)+1), shutdown, exit)
	if md := hoverText(t, byID(msgs, 10)); !strings.Contains(md, "switched off") {
		t.Fatal(md)
	}
}

func TestInitializationOptionsNameThePassmcp(t *testing.T) {
	var got string
	opt := Options{Guidance: func(program string) Looker { got = program; return fakeLooker{} }}
	_, _ = session(t, opt, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"initializationOptions":{"passmcpPath":"/opt/passmcp"}}}`, shutdown, exit)
	if got != "/opt/passmcp" {
		t.Fatalf("got %q", got)
	}
}
