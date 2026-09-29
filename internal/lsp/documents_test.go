// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package lsp

import (
	"encoding/json"
	"strings"
	"testing"
)

func change(uri string, version int, changes ...map[string]any) string {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didChange",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "version": version}, "contentChanges": changes}})
	return string(b)
}

func closeDoc(uri string) string {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didClose",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri}}})
	return string(b)
}

func diagnosticsOf(m map[string]any) []map[string]any {
	var out []map[string]any
	for _, d := range m["params"].(map[string]any)["diagnostics"].([]any) {
		out = append(out, d.(map[string]any))
	}
	return out
}

func TestDiagnosticsFollowTheText(t *testing.T) {
	uri := "file:///repo/company.policy.json"
	msgs, err := session(t, Options{}, initReq, inited,
		open(uri, "{\n  \"version\": 1,\n  \"name\": \"x\",\n  \"must_pas\": [],\n  \"max_fail\": 0\n}"),
		change(uri, 2, map[string]any{"text": `{"version": 1, "name": "x", "must_pass": ["net.tls"]}`}),
		change(uri, 3, map[string]any{"range": map[string]any{"start": map[string]int{"line": 0, "character": 12}, "end": map[string]int{"line": 0, "character": 13}}, "text": "2"}),
		closeDoc(uri),
		shutdown, exit)
	if err != nil {
		t.Fatal(err)
	}
	pubs := notificationsFor(msgs, "textDocument/publishDiagnostics")
	if len(pubs) != 4 {
		t.Fatalf("one publish per open, change and close: %d", len(pubs))
	}
	checkFirstPublish(t, diagnosticsOf(pubs[0]))
	if len(diagnosticsOf(pubs[1])) != 0 {
		t.Fatalf("a full-text change that fixes it clears it: %v", diagnosticsOf(pubs[1]))
	}
	third := diagnosticsOf(pubs[2])
	if len(third) != 1 || !strings.Contains(third[0]["message"].(string), "version 2") {
		t.Fatalf("an incremental edit is applied: %v", third)
	}
	if pubs[3]["params"].(map[string]any)["uri"] != uri || len(diagnosticsOf(pubs[3])) != 0 {
		t.Fatal("close clears the diagnostics")
	}
}

// checkFirstPublish asserts the one diagnostic the opened text has, and
// that its range is the misspelled key on line 4.
func checkFirstPublish(t *testing.T, ds []map[string]any) {
	t.Helper()
	if len(ds) != 1 || ds[0]["code"] != "policy/unknown-key" || ds[0]["source"] != "passmcp-lsp" || ds[0]["severity"] != 1.0 {
		t.Fatalf("open: %v", ds)
	}
	start := ds[0]["range"].(map[string]any)["start"].(map[string]any)
	if start["line"] != 3.0 || start["character"] != 2.0 {
		t.Fatalf("the range points at the key: %v", start)
	}
}

func TestDocumentEdgeCases(t *testing.T) {
	msgs, _ := session(t, Options{}, initReq,
		`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":{}}`,
		change("file:///not/open.json", 2, map[string]any{"text": "{}"}),
		`{"jsonrpc":"2.0","method":"textDocument/didChange","params":"x"}`,
		closeDoc("file:///not/open.json"),
		`{"jsonrpc":"2.0","method":"textDocument/didClose","params":"x"}`,
		open("untitled:Untitled-1", `{"mcpServers": {"a": {}}}`),
		shutdown, exit)
	pubs := notificationsFor(msgs, "textDocument/publishDiagnostics")
	if len(pubs) != 1 {
		t.Fatalf("only the untitled document publishes: %v", pubs)
	}
	if d := diagnosticsOf(pubs[0]); len(d) != 1 || d[0]["code"] != "client/entry" {
		t.Fatalf("an untitled buffer is recognised by content: %v", d)
	}
}

func TestApplyReversedRange(t *testing.T) {
	r := &lspRange{}
	r.Start.Character, r.End.Character = 3, 1
	if got := string(apply([]byte("abcdef"), contentChange{Range: r, Text: "X"})); got != "aXdef" {
		t.Fatalf("got %q", got)
	}
}

func TestNameOf(t *testing.T) {
	for uri, want := range map[string]string{
		"file:///home/a/.vscode/mcp.json": "/home/a/.vscode/mcp.json",
		"file:///c%3A/repo/server.json":   "/c:/repo/server.json",
		"untitled:Untitled-1":             "",
		"%zz":                             "",
	} {
		if got := nameOf(uri); got != want {
			t.Errorf("nameOf(%q) = %q, want %q", uri, got, want)
		}
	}
}
