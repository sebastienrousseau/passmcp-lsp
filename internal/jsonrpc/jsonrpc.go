// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package jsonrpc is the JSON-RPC 2.0 layer of the Language Server
// Protocol's base protocol: messages framed by a Content-Length header,
// read one at a time from the client and written, whole and one at a time,
// back to it.
//
// It is deliberately small. The server needs to read requests and
// notifications and to write responses and notifications; it never sends a
// request of its own, so there is no call tracking here.
package jsonrpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// MaxMessage bounds one message's body. An editor sends a whole document
// on every change, and 16 MiB is far beyond any artefact this server
// reads; a larger length is a broken or hostile client.
const MaxMessage = 16 << 20

// maxHeaderLine bounds one header line, so a client cannot make the reader
// buffer an unbounded line before the body.
const maxHeaderLine = 4096

// The error codes of JSON-RPC 2.0 and the Language Server Protocol.
const (
	CodeParseError           = -32700
	CodeInvalidRequest       = -32600
	CodeMethodNotFound       = -32601
	CodeInvalidParams        = -32602
	CodeInternalError        = -32603
	CodeServerNotInitialized = -32002
)

// Message is one decoded message: a request (ID and Method), a
// notification (Method only) or a response (ID and Result or Error).
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// IsRequest reports whether the message expects a response.
func (m *Message) IsRequest() bool { return len(m.ID) > 0 && m.Method != "" }

// IsNotification reports whether the message is a notification.
func (m *Message) IsNotification() bool { return len(m.ID) == 0 && m.Method != "" }

// Error is a JSON-RPC error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("jsonrpc %d: %s", e.Code, e.Message) }

// ErrFraming is wrapped by every error about the header block. The
// connection cannot be resynchronised after one.
var ErrFraming = errors.New("jsonrpc: bad framing")

// Reader reads framed messages.
type Reader struct{ r *bufio.Reader }

// NewReader reads messages from r.
func NewReader(r io.Reader) *Reader { return &Reader{r: bufio.NewReaderSize(r, maxHeaderLine)} }

// Read returns the next message's body. io.EOF means the client closed
// the stream between messages; any other end is io.ErrUnexpectedEOF.
func (r *Reader) Read() ([]byte, error) {
	length, err := r.header()
	if err != nil {
		return nil, err
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(r.r, body); err != nil {
		return nil, io.ErrUnexpectedEOF
	}
	return body, nil
}

// header reads the header block and returns the body length.
func (r *Reader) header() (int, error) {
	length, lines := -1, 0
	for {
		line, err := r.line()
		switch {
		case errors.Is(err, io.EOF) && lines == 0:
			return 0, io.EOF
		case errors.Is(err, io.EOF):
			return 0, io.ErrUnexpectedEOF
		case err != nil:
			return 0, err
		}
		lines++
		if line == "" {
			break
		}
		if n, ok, err := contentLength(line); err != nil {
			return 0, err
		} else if ok {
			length = n
		}
	}
	if length < 0 {
		return 0, fmt.Errorf("%w: no Content-Length header", ErrFraming)
	}
	return length, nil
}

// line reads one header line without its line ending.
func (r *Reader) line() (string, error) {
	b, err := r.r.ReadSlice('\n')
	switch {
	case errors.Is(err, bufio.ErrBufferFull):
		return "", fmt.Errorf("%w: a header line longer than %d bytes", ErrFraming, maxHeaderLine)
	case errors.Is(err, io.EOF) && len(b) == 0:
		return "", io.EOF
	case err != nil:
		return "", io.ErrUnexpectedEOF
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

// contentLength parses a Content-Length header; other headers are ignored,
// as the protocol allows.
func contentLength(line string) (int, bool, error) {
	name, value, ok := strings.Cut(line, ":")
	if !ok {
		return 0, false, fmt.Errorf("%w: %q is not a header", ErrFraming, line)
	}
	if !strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
		return 0, false, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 0 {
		return 0, false, fmt.Errorf("%w: Content-Length %q", ErrFraming, strings.TrimSpace(value))
	}
	if n > MaxMessage {
		return 0, false, fmt.Errorf("%w: a %d-byte message is over the %d-byte limit", ErrFraming, n, MaxMessage)
	}
	return n, true, nil
}

// Writer writes framed messages. It is safe for concurrent use: each
// message is written whole before the next begins.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

// NewWriter writes messages to w.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// Write frames and writes one body.
func (w *Writer) Write(body []byte) error {
	var b bytes.Buffer
	b.WriteString("Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n")
	b.Write(body)
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.w.Write(b.Bytes())
	return err
}

// Reply writes a successful response. A nil result is written as null,
// which is how the protocol spells "no result".
func (w *Writer) Reply(id json.RawMessage, result any) error {
	return w.send(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
	}{"2.0", id, result})
}

// ReplyError writes an error response. A message whose id could not be
// read is answered with a null id.
func (w *Writer) ReplyError(id json.RawMessage, code int, msg string) error {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return w.send(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   Error           `json:"error"`
	}{"2.0", id, Error{Code: code, Message: msg}})
}

// Notify writes a notification.
func (w *Writer) Notify(method string, params any) error {
	return w.send(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}{"2.0", method, params})
}

func (w *Writer) send(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.Write(b)
}

// Decode parses a body into a Message and checks the envelope.
func Decode(body []byte) (*Message, *Error) {
	var m Message
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, &Error{Code: CodeParseError, Message: "the message is not JSON: " + err.Error()}
	}
	if m.JSONRPC != "2.0" {
		return &m, &Error{Code: CodeInvalidRequest, Message: `the message is not JSON-RPC 2.0: "jsonrpc" must be "2.0"`}
	}
	if len(m.ID) > 0 && !validID(m.ID) {
		return &m, &Error{Code: CodeInvalidRequest, Message: "a request id must be a number or a string"}
	}
	return &m, nil
}

// validID accepts the id kinds the protocol allows: number or string.
func validID(id json.RawMessage) bool {
	var v any
	if json.Unmarshal(id, &v) != nil {
		return false
	}
	switch v.(type) {
	case float64, string:
		return true
	}
	return false
}
