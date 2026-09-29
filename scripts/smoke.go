// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// smoke drives a built passmcp-lsp over stdio the way an editor does:
// initialize, open a broken server.json, shut down, exit. It fails unless
// the server advertises hover, publishes a schema diagnostic for the
// document, answers shutdown, and exits 0.
//
//	go run scripts/smoke.go build/passmcp-lsp
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"satellion.com/passmcp-lsp/internal/jsonrpc"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/smoke.go PASSMCP-LSP")
		os.Exit(2)
	}
	if err := smoke(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "smoke:", err)
		os.Exit(1)
	}
	fmt.Println("smoke: initialize, diagnostics for server.json, shutdown and exit all answered")
}

func smoke(bin string) error {
	doc, _ := json.Marshal(map[string]string{"name": "no-slash", "description": "d", "version": "1"})
	var in bytes.Buffer
	w := jsonrpc.NewWriter(&in)
	for _, m := range []any{
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]any{"capabilities": map[string]any{}}},
		map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}},
		map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen", "params": map[string]any{"textDocument": map[string]any{
			"uri": "file:///smoke/server.json", "languageId": "json", "version": 1, "text": string(doc)}}},
		map[string]any{"jsonrpc": "2.0", "id": 2, "method": "shutdown"},
		map[string]any{"jsonrpc": "2.0", "method": "exit"},
	} {
		b, _ := json.Marshal(m)
		_ = w.Write(b)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--stdio")
	cmd.Stdin = &in
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s did not exit 0: %w", bin, err)
	}
	return expect(out.Bytes())
}

// expect reads every message the server wrote and checks the three that
// matter.
func expect(out []byte) error {
	r := jsonrpc.NewReader(bytes.NewReader(out))
	var sawHover, sawDiagnostic, sawShutdown bool
	for {
		body, err := r.Read()
		if err != nil {
			break
		}
		s := string(body)
		sawHover = sawHover || strings.Contains(s, `"hoverProvider":true`)
		sawDiagnostic = sawDiagnostic || (strings.Contains(s, "publishDiagnostics") && strings.Contains(s, "server-json/schema"))
		sawShutdown = sawShutdown || s == `{"jsonrpc":"2.0","id":2,"result":null}`
	}
	switch {
	case !sawHover:
		return fmt.Errorf("initialize did not advertise hover")
	case !sawDiagnostic:
		return fmt.Errorf("no server-json/schema diagnostic was published for the broken server.json")
	case !sawShutdown:
		return fmt.Errorf("shutdown was not answered with null")
	}
	return nil
}
