// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package jsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

func frame(body string) string {
	return "Content-Length: " + itoa(len(body)) + "\r\n\r\n" + body
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestReadMessages(t *testing.T) {
	in := frame(`{"a":1}`) + "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\ncontent-length: 2\r\n\r\n{}" + "Content-Length: 3\n\n[1]"
	r := NewReader(strings.NewReader(in))
	for _, want := range []string{`{"a":1}`, `{}`, `[1]`} {
		got, err := r.Read()
		if err != nil || string(got) != want {
			t.Fatalf("Read = %q, %v; want %q", got, err, want)
		}
	}
	if _, err := r.Read(); !errors.Is(err, io.EOF) {
		t.Fatalf("a clean end is io.EOF, got %v", err)
	}
}

func TestReadErrors(t *testing.T) {
	cases := map[string]error{
		"Content-Length: 10\r\n\r\n{}":             io.ErrUnexpectedEOF,
		"Content-Length: 2\r\n":                    io.ErrUnexpectedEOF,
		"Content-Length: 2":                        io.ErrUnexpectedEOF,
		"Content-Type: x\r\n\r\n{}":                ErrFraming,
		"no colon here\r\n\r\n":                    ErrFraming,
		"Content-Length: -1\r\n\r\n":               ErrFraming,
		"Content-Length: abc\r\n\r\n":              ErrFraming,
		"Content-Length: 999999999999\r\n\r\n":     ErrFraming,
		"X: " + strings.Repeat("a", 5000) + "\r\n": ErrFraming,
	}
	for in, want := range cases {
		if _, err := NewReader(strings.NewReader(in)).Read(); !errors.Is(err, want) {
			t.Errorf("%.40q: got %v, want %v", in, err, want)
		}
	}
}

func TestWriterFramesAndSerialises(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = w.Notify("n", map[string]int{"i": 1}) }()
	}
	wg.Wait()
	r := NewReader(&buf)
	for i := 0; i < 20; i++ {
		body, err := r.Read()
		if err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if string(body) != `{"jsonrpc":"2.0","method":"n","params":{"i":1}}` {
			t.Fatalf("interleaved or malformed: %s", body)
		}
	}
}

func TestReplies(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	_ = w.Reply(json.RawMessage(`1`), nil)
	_ = w.Reply(json.RawMessage(`"a"`), map[string]bool{"ok": true})
	_ = w.ReplyError(nil, CodeParseError, "bad")
	_ = w.ReplyError(json.RawMessage(`2`), CodeMethodNotFound, "no")
	r := NewReader(&buf)
	for _, want := range []string{
		`{"jsonrpc":"2.0","id":1,"result":null}`,
		`{"jsonrpc":"2.0","id":"a","result":{"ok":true}}`,
		`{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"bad"}}`,
		`{"jsonrpc":"2.0","id":2,"error":{"code":-32601,"message":"no"}}`,
	} {
		if got, _ := r.Read(); string(got) != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
	if err := w.Reply(json.RawMessage(`1`), func() {}); err == nil {
		t.Error("an unmarshalable result is an error")
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

func TestWriteError(t *testing.T) {
	if err := NewWriter(failWriter{}).Notify("x", nil); err == nil {
		t.Fatal("a failed write is returned")
	}
}

func TestDecode(t *testing.T) {
	m, e := Decode([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if e != nil || !m.IsRequest() || m.IsNotification() {
		t.Fatalf("request: %+v %v", m, e)
	}
	m, e = Decode([]byte(`{"jsonrpc":"2.0","method":"initialized"}`))
	if e != nil || m.IsRequest() || !m.IsNotification() {
		t.Fatalf("notification: %+v %v", m, e)
	}
}

func TestDecodeRefusals(t *testing.T) {
	for body, code := range map[string]int{
		`not json`:                                CodeParseError,
		`{"jsonrpc":"1.0","method":"x"}`:          CodeInvalidRequest,
		`{"jsonrpc":"2.0","id":{},"method":"x"}`:  CodeInvalidRequest,
		`{"jsonrpc":"2.0","id":[1],"method":"x"}`: CodeInvalidRequest,
	} {
		if _, e := Decode([]byte(body)); e == nil || e.Code != code {
			t.Errorf("%s: got %v, want code %d", body, e, code)
		}
	}
	if (&Error{Code: 1, Message: "m"}).Error() != "jsonrpc 1: m" {
		t.Error("Error()")
	}
	if validID(json.RawMessage(`{`)) {
		t.Error("an unparsable id is not valid")
	}
}
