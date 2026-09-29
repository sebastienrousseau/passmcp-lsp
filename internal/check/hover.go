// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"regexp"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// Ref is a passmcp check id at a place in a document.
type Ref struct {
	ID         string
	Start, End int
	// Doc is the documentation link the document itself gives for the
	// check, when it gives one (an attestation verdict's "doc").
	Doc string
}

// checkID is the shape of a passmcp check id: a phase, a dot, and a name.
var checkID = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[A-Za-z0-9_.:/-]{1,200}$`)

// IDAt returns the check id under byte offset off, if the document
// names one there: a policy's must_pass, must_not_fail or exemption check,
// or an attestation verdict's id.
func IDAt(r Result, off int) (Ref, bool) {
	if r.Root == nil {
		return Ref{}, false
	}
	n, path, onKey := r.Root.Find(off)
	if n == nil || onKey || n.Kind != jsondoc.String || !checkID.MatchString(n.Str) {
		return Ref{}, false
	}
	ref := Ref{ID: n.Str, Start: n.Start, End: n.End}
	switch {
	case r.Kind == Policy && isPolicyCheckPath(path):
		return ref, true
	case r.Kind == Attestation && isVerdictIDPath(path):
		ref.Doc = strOrEmpty(r.Root.Get("predicate").Get("verdicts").Items[path[2].Index].Get("doc"))
		return ref, true
	}
	return Ref{}, false
}

// isPolicyCheckPath matches must_pass[i], must_not_fail[i] and
// exemptions[i].check.
func isPolicyCheckPath(p jsondoc.Path) bool {
	switch {
	case len(p) == 2 && p[1].IsIndex:
		return p[0].Key == "must_pass" || p[0].Key == "must_not_fail"
	case len(p) == 3:
		return p[0].Key == "exemptions" && p[1].IsIndex && p[2].Key == "check"
	}
	return false
}

// isVerdictIDPath matches predicate.verdicts[i].id.
func isVerdictIDPath(p jsondoc.Path) bool {
	return len(p) == 4 && p[0].Key == "predicate" && p[1].Key == "verdicts" && p[2].IsIndex && p[3].Key == "id"
}
