// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package lsp is the language server: the protocol's lifecycle, the open
// documents, diagnostics published on every change, and hover over passmcp
// check ids.
//
// Everything runs on the read loop except hover, which may wait on the
// passmcp program and so answers from its own goroutine; documents are
// only ever touched by the loop.
package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"satellion.com/passmcp-lsp/internal/check"
	"satellion.com/passmcp-lsp/internal/guidance"
	"satellion.com/passmcp-lsp/internal/jsondoc"
	"satellion.com/passmcp-lsp/internal/jsonrpc"
)

// Looker looks up passmcp's guidance for a check id.
type Looker interface {
	Lookup(ctx context.Context, id string) (guidance.Guidance, bool, error)
}

// Options configure a server.
type Options struct {
	// Name and Version are reported to the client in serverInfo.
	Name, Version string
	// Guidance builds the guidance source once the client has initialised,
	// given the passmcp program the client named ("" when it named none).
	Guidance func(program string) Looker
	// Log receives diagnostics about the protocol itself. Never stdout,
	// which is the protocol's own stream.
	Log io.Writer
}

// ErrExitWithoutShutdown is returned when the client ends the session
// without asking the server to shut down first; the protocol says the
// process then exits with status 1.
var ErrExitWithoutShutdown = errors.New("lsp: the client exited without a shutdown request")

type lifecycle int

const (
	created lifecycle = iota
	running
	shuttingDown
)

// Server is one session with one client.
type Server struct {
	opt    Options
	out    *jsonrpc.Writer
	state  lifecycle
	docs   map[string]*document
	looker Looker
	ctx    context.Context
	wg     sync.WaitGroup
}

// document is an open text and its latest analysis.
type document struct {
	uri     string
	text    []byte
	lines   *jsondoc.Lines
	result  check.Result
	version int
}

// Serve runs a session: it reads from in and writes to out until the
// client sends exit or closes the stream. It returns nil after an orderly
// shutdown and exit.
func Serve(ctx context.Context, in io.Reader, out io.Writer, opt Options) error {
	if opt.Log == nil {
		opt.Log = io.Discard
	}
	s := &Server{opt: opt, out: jsonrpc.NewWriter(out), docs: map[string]*document{}, ctx: ctx}
	defer s.wg.Wait()
	r := jsonrpc.NewReader(in)
	for {
		body, err := r.Read()
		if errors.Is(err, io.EOF) {
			return s.exitStatus()
		}
		if err != nil {
			return err
		}
		if s.dispatch(body) {
			return s.exitStatus()
		}
	}
}

func (s *Server) exitStatus() error {
	if s.state == shuttingDown {
		return nil
	}
	return ErrExitWithoutShutdown
}

// dispatch handles one message and reports whether the session is over.
func (s *Server) dispatch(body []byte) bool {
	m, bad := jsonrpc.Decode(body)
	if bad != nil {
		s.logf("refused a message: %s", bad.Message)
		if m == nil || len(m.ID) > 0 {
			s.replyError(idOf(m), bad.Code, bad.Message)
		}
		return false
	}
	switch {
	case m.Method == "exit":
		return true
	case m.IsRequest():
		s.request(m)
	case m.IsNotification():
		s.notification(m)
	}
	return false
}

func idOf(m *jsonrpc.Message) json.RawMessage {
	if m == nil {
		return nil
	}
	return m.ID
}

type requestHandler func(s *Server, id, params json.RawMessage)

var requests = map[string]requestHandler{
	"initialize":         (*Server).initialize,
	"shutdown":           (*Server).shutdown,
	"textDocument/hover": (*Server).hover,
}

func (s *Server) request(m *jsonrpc.Message) {
	switch {
	case s.state == created && m.Method != "initialize":
		s.replyError(m.ID, jsonrpc.CodeServerNotInitialized, "the server is not initialised")
		return
	case s.state == shuttingDown:
		s.replyError(m.ID, jsonrpc.CodeInvalidRequest, "the server is shutting down")
		return
	}
	h, ok := requests[m.Method]
	if !ok {
		s.replyError(m.ID, jsonrpc.CodeMethodNotFound, "passmcp-lsp does not implement "+m.Method)
		return
	}
	h(s, m.ID, m.Params)
}

type notificationHandler func(s *Server, params json.RawMessage)

var notifications = map[string]notificationHandler{
	"textDocument/didOpen":   (*Server).didOpen,
	"textDocument/didChange": (*Server).didChange,
	"textDocument/didClose":  (*Server).didClose,
}

// notification handles a notification. Unknown ones, "initialized" and
// "$/" ones included, are ignored, as the protocol requires.
func (s *Server) notification(m *jsonrpc.Message) {
	if s.state != running {
		return
	}
	if h, ok := notifications[m.Method]; ok {
		h(s, m.Params)
	}
}

func (s *Server) initialize(id, params json.RawMessage) {
	if s.state != created {
		s.replyError(id, jsonrpc.CodeInvalidRequest, "initialize was already received")
		return
	}
	var p struct {
		InitializationOptions struct {
			PassmcpPath string `json:"passmcpPath"`
		} `json:"initializationOptions"`
	}
	// Options are optional; an unreadable block is the same as none.
	_ = json.Unmarshal(params, &p)
	if s.opt.Guidance != nil {
		s.looker = s.opt.Guidance(p.InitializationOptions.PassmcpPath)
	}
	s.state = running
	s.reply(id, map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync": map[string]any{"openClose": true, "change": 1},
			"hoverProvider":    true,
		},
		"serverInfo": map[string]string{"name": s.opt.Name, "version": s.opt.Version},
	})
}

func (s *Server) shutdown(id, _ json.RawMessage) {
	s.state = shuttingDown
	s.reply(id, nil)
}

func (s *Server) reply(id json.RawMessage, result any) {
	if err := s.out.Reply(id, result); err != nil {
		s.logf("writing a response: %v", err)
	}
}

func (s *Server) replyError(id json.RawMessage, code int, msg string) {
	if err := s.out.ReplyError(id, code, msg); err != nil {
		s.logf("writing an error response: %v", err)
	}
}

func (s *Server) logf(format string, a ...any) {
	_, _ = fmt.Fprintf(s.opt.Log, "passmcp-lsp: "+format+"\n", a...)
}
