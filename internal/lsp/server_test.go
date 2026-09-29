// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"satellion.com/passmcp-lsp/internal/guidance"
	"satellion.com/passmcp-lsp/internal/jsonrpc"
)

type fakeLooker struct{}

func (fakeLooker) Lookup(_ context.Context, id string) (guidance.Guidance, bool, error) {
	switch id {
	case "net.tls":
		return guidance.Guidance{ID: id, Means: "TLS protects the token.", Steps: []guidance.Step{{Title: "Serve https", Body: "Terminate TLS."}}}, true, nil
	case "absent.check":
		return guidance.Guidance{}, false, guidance.ErrNotInstalled
	case "broken.check":
		return guidance.Guidance{}, false, errors.New("exit status 1")
	}
	return guidance.Guidance{}, false, nil
}

// session sends the messages, in order, and returns every message the
// server wrote and Serve's error.
func session(t *testing.T, opt Options, msgs ...string) ([]map[string]any, error) {
	t.Helper()
	var in bytes.Buffer
	w := jsonrpc.NewWriter(&in)
	for _, m := range msgs {
		if err := w.Write([]byte(m)); err != nil {
			t.Fatal(err)
		}
	}
	var out, log bytes.Buffer
	if opt.Guidance == nil {
		opt.Guidance = func(string) Looker { return fakeLooker{} }
	}
	opt.Log = &log
	err := Serve(context.Background(), &in, &out, opt)
	r := jsonrpc.NewReader(&out)
	var got []map[string]any
	for {
		body, rerr := r.Read()
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			t.Fatal(rerr)
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatal(err)
		}
		got = append(got, m)
	}
	return got, err
}

const (
	initReq  = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}`
	inited   = `{"jsonrpc":"2.0","method":"initialized","params":{}}`
	shutdown = `{"jsonrpc":"2.0","id":99,"method":"shutdown"}`
	exit     = `{"jsonrpc":"2.0","method":"exit"}`
)

func byID(msgs []map[string]any, id float64) map[string]any {
	for _, m := range msgs {
		if m["id"] == id {
			return m
		}
	}
	return nil
}

func notificationsFor(msgs []map[string]any, method string) []map[string]any {
	var out []map[string]any
	for _, m := range msgs {
		if m["method"] == method {
			out = append(out, m)
		}
	}
	return out
}

func open(uri, text string) string {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "textDocument/didOpen",
		"params": map[string]any{"textDocument": map[string]any{"uri": uri, "languageId": "json", "version": 1, "text": text}}})
	return string(b)
}

func TestLifecycle(t *testing.T) {
	msgs, err := session(t, Options{Name: "passmcp-lsp", Version: "9.9.9"}, initReq, inited, shutdown, exit)
	if err != nil {
		t.Fatalf("an orderly exit returns nil: %v", err)
	}
	res := byID(msgs, 1)["result"].(map[string]any)
	caps := res["capabilities"].(map[string]any)
	if caps["hoverProvider"] != true || caps["textDocumentSync"].(map[string]any)["change"] != 1.0 {
		t.Fatalf("capabilities: %v", caps)
	}
	if res["serverInfo"].(map[string]any)["version"] != "9.9.9" {
		t.Fatal("serverInfo")
	}
	if r := byID(msgs, 99); r == nil || r["result"] != nil {
		t.Fatalf("shutdown answers null: %v", r)
	}
}

func TestExitWithoutShutdown(t *testing.T) {
	if _, err := session(t, Options{}, initReq, exit); !errors.Is(err, ErrExitWithoutShutdown) {
		t.Fatalf("exit before shutdown: %v", err)
	}
	if _, err := session(t, Options{}, initReq); !errors.Is(err, ErrExitWithoutShutdown) {
		t.Fatalf("the stream ending before shutdown: %v", err)
	}
	if _, err := session(t, Options{}, initReq, shutdown); err != nil {
		t.Fatalf("the stream ending after shutdown: %v", err)
	}
}

func TestProtocolErrors(t *testing.T) {
	msgs, _ := session(t, Options{},
		`{"jsonrpc":"2.0","id":2,"method":"textDocument/hover","params":{}}`,
		initReq,
		`{"jsonrpc":"2.0","id":3,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":4,"method":"workspace/symbol","params":{}}`,
		`not json`,
		`{"jsonrpc":"1.0","id":5,"method":"x"}`,
		`{"jsonrpc":"1.0","method":"x"}`,
		`{"jsonrpc":"2.0","id":6,"method":"textDocument/hover","params":"nope"}`,
		`{"jsonrpc":"2.0","method":"$/cancelRequest","params":{"id":1}}`,
		shutdown,
		`{"jsonrpc":"2.0","id":7,"method":"textDocument/hover","params":{}}`,
		exit)
	codes := map[float64]float64{2: jsonrpc.CodeServerNotInitialized, 3: jsonrpc.CodeInvalidRequest, 4: jsonrpc.CodeMethodNotFound,
		5: jsonrpc.CodeInvalidRequest, 6: jsonrpc.CodeInvalidParams, 7: jsonrpc.CodeInvalidRequest}
	for id, code := range codes {
		m := byID(msgs, id)
		if m == nil || m["error"].(map[string]any)["code"] != code {
			t.Errorf("id %v: want code %v, got %v", id, code, m)
		}
	}
	parseErrors := 0
	for _, m := range msgs {
		if e, ok := m["error"].(map[string]any); ok && e["code"] == float64(jsonrpc.CodeParseError) && m["id"] == nil {
			parseErrors++
		}
	}
	if parseErrors != 1 {
		t.Errorf("an unparsable message gets one error with a null id, got %d", parseErrors)
	}
}

func TestFramingErrorEndsTheSession(t *testing.T) {
	var out bytes.Buffer
	err := Serve(context.Background(), strings.NewReader("Content-Length: x\r\n\r\n"), &out, Options{})
	if !errors.Is(err, jsonrpc.ErrFraming) {
		t.Fatalf("got %v", err)
	}
}

func TestNotificationsBeforeInitializeAreDropped(t *testing.T) {
	msgs, _ := session(t, Options{}, open("file:///r/server.json", `{}`), initReq, shutdown, exit)
	if len(notificationsFor(msgs, "textDocument/publishDiagnostics")) != 0 {
		t.Fatal("a document opened before initialize is not analysed")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestWriteFailuresAreLogged(t *testing.T) {
	var in, log bytes.Buffer
	w := jsonrpc.NewWriter(&in)
	for _, m := range []string{initReq, `{"jsonrpc":"2.0","id":2,"method":"nope"}`, open("file:///r/server.json", "{}"), shutdown, exit} {
		_ = w.Write([]byte(m))
	}
	if err := Serve(context.Background(), &in, failWriter{}, Options{Log: &log}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"writing a response", "writing an error response", "publishing diagnostics"} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log lacks %q: %s", want, log.String())
		}
	}
}
