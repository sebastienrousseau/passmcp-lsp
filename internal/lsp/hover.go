// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package lsp

import (
	"encoding/json"
	"errors"

	"satellion.com/passmcp-lsp/internal/check"
	"satellion.com/passmcp-lsp/internal/guidance"
	"satellion.com/passmcp-lsp/internal/jsondoc"
	"satellion.com/passmcp-lsp/internal/jsonrpc"
)

// hover answers with passmcp's guidance for the check id under the cursor,
// or null when there is none there. The lookup may wait on the passmcp
// program, so it runs off the read loop, with what it needs copied.
func (s *Server) hover(id, params json.RawMessage) {
	var p struct {
		TextDocument docID            `json:"textDocument"`
		Position     jsondoc.Position `json:"position"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		s.replyError(id, jsonrpc.CodeInvalidParams, "hover: "+err.Error())
		return
	}
	d, ok := s.docs[p.TextDocument.URI]
	if !ok {
		s.reply(id, nil)
		return
	}
	ref, ok := check.IDAt(d.result, d.lines.Offset(p.Position))
	if !ok {
		s.reply(id, nil)
		return
	}
	rng := lspRange{Start: d.lines.Position(ref.Start), End: d.lines.Position(ref.End)}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.reply(id, map[string]any{
			"contents": map[string]string{"kind": "markdown", "value": s.explain(ref)},
			"range":    rng,
		})
	}()
}

// explain renders the hover for one check id, saying why when there is no
// guidance to show.
func (s *Server) explain(ref check.Ref) string {
	if s.looker == nil {
		return guidance.Unavailable(ref.ID, "Guidance lookups are switched off.")
	}
	g, found, err := s.looker.Lookup(s.ctx, ref.ID)
	switch {
	case errors.Is(err, guidance.ErrNotInstalled):
		return guidance.Unavailable(ref.ID, "Install passmcp to see its guidance for this check (https://github.com/sebastienrousseau/passmcp#install), or name it with --passmcp or the passmcpPath initialization option.")
	case err != nil:
		s.logf("guidance for %s: %v", ref.ID, err)
		return guidance.Unavailable(ref.ID, "passmcp could not be asked for guidance: "+err.Error())
	case !found:
		return guidance.Unavailable(ref.ID, "The installed passmcp has no guidance for this check id.")
	}
	return guidance.Markdown(g, ref.Doc)
}
