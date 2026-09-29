// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package jsonrpc

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// FuzzFraming feeds arbitrary bytes to the reader. It must never panic or
// return a body longer than MaxMessage, and every error must be one of the
// three it documents. A body written by Writer must read back unchanged.
func FuzzFraming(f *testing.F) {
	for _, s := range []string{
		"Content-Length: 2\r\n\r\n{}",
		"Content-Length: 5\r\nContent-Type: x\r\n\r\nhello",
		"content-length:0\n\n",
		"Content-Length: 99999999999\r\n\r\n",
		"garbage",
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		readSome(t, in)
		var buf bytes.Buffer
		if err := NewWriter(&buf).Write(in); err != nil {
			t.Fatal(err)
		}
		got, err := NewReader(&buf).Read()
		if err != nil || !bytes.Equal(got, in) {
			t.Fatalf("round trip: %q, %v", got, err)
		}
	})
}

// readSome reads up to eight messages from in, checking each result.
func readSome(t *testing.T, in []byte) {
	r := NewReader(bytes.NewReader(in))
	for i := 0; i < 8; i++ {
		body, err := r.Read()
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, ErrFraming) {
				t.Fatalf("undocumented error %v", err)
			}
			return
		}
		if len(body) > MaxMessage {
			t.Fatalf("a %d-byte body passed the limit", len(body))
		}
	}
}
