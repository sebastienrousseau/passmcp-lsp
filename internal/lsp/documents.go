// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package lsp

import (
	"encoding/json"
	"net/url"

	"satellion.com/passmcp-lsp/internal/check"
	"satellion.com/passmcp-lsp/internal/jsondoc"
)

type textDocumentItem struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
	Text    string `json:"text"`
}

type docID struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type lspRange struct {
	Start jsondoc.Position `json:"start"`
	End   jsondoc.Position `json:"end"`
}

type contentChange struct {
	Range *lspRange `json:"range,omitempty"`
	Text  string    `json:"text"`
}

func (s *Server) didOpen(params json.RawMessage) {
	var p struct {
		TextDocument textDocumentItem `json:"textDocument"`
	}
	if json.Unmarshal(params, &p) != nil || p.TextDocument.URI == "" {
		s.logf("didOpen without a document")
		return
	}
	d := &document{uri: p.TextDocument.URI, version: p.TextDocument.Version}
	s.docs[d.uri] = d
	s.update(d, []byte(p.TextDocument.Text))
}

// didChange applies the changes in order. The server asks for whole-text
// sync, but a change with a range is applied as an edit all the same.
func (s *Server) didChange(params json.RawMessage) {
	var p struct {
		TextDocument   docID           `json:"textDocument"`
		ContentChanges []contentChange `json:"contentChanges"`
	}
	if json.Unmarshal(params, &p) != nil {
		s.logf("didChange with unreadable parameters")
		return
	}
	d, ok := s.docs[p.TextDocument.URI]
	if !ok {
		s.logf("didChange for %s, which is not open", p.TextDocument.URI)
		return
	}
	text := d.text
	for _, c := range p.ContentChanges {
		text = apply(text, c)
	}
	d.version = p.TextDocument.Version
	s.update(d, text)
}

// apply returns text with one change made.
func apply(text []byte, c contentChange) []byte {
	if c.Range == nil {
		return []byte(c.Text)
	}
	l := jsondoc.NewLines(text)
	start, end := l.Offset(c.Range.Start), l.Offset(c.Range.End)
	if end < start {
		start, end = end, start
	}
	out := make([]byte, 0, len(text)-(end-start)+len(c.Text))
	out = append(out, text[:start]...)
	out = append(out, c.Text...)
	return append(out, text[end:]...)
}

func (s *Server) didClose(params json.RawMessage) {
	var p struct {
		TextDocument docID `json:"textDocument"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	if _, ok := s.docs[p.TextDocument.URI]; !ok {
		return
	}
	delete(s.docs, p.TextDocument.URI)
	// Clearing the document's diagnostics is the server's job on close.
	s.publish(p.TextDocument.URI, nil, nil)
}

// update re-analyses a document and publishes what was found.
func (s *Server) update(d *document, text []byte) {
	d.text = text
	d.lines = jsondoc.NewLines(text)
	d.result = check.Analyze(nameOf(d.uri), text)
	s.publish(d.uri, d.lines, d.result.Diagnostics)
}

// nameOf is the file path a URI names, which recognition reads; an
// untitled buffer has none and is recognised by content alone.
func nameOf(uri string) string {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return ""
	}
	return u.Path
}

type wireDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Code     string   `json:"code"`
	Source   string   `json:"source"`
	Message  string   `json:"message"`
}

func (s *Server) publish(uri string, lines *jsondoc.Lines, ds []check.Diagnostic) {
	wire := make([]wireDiagnostic, 0, len(ds))
	for _, d := range ds {
		wire = append(wire, wireDiagnostic{
			Range:    lspRange{Start: lines.Position(d.Start), End: lines.Position(d.End)},
			Severity: int(d.Severity),
			Code:     d.Code,
			Source:   "passmcp-lsp",
			Message:  d.Message,
		})
	}
	err := s.out.Notify("textDocument/publishDiagnostics", map[string]any{"uri": uri, "diagnostics": wire})
	if err != nil {
		s.logf("publishing diagnostics: %v", err)
	}
}
