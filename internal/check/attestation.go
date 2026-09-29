// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"strings"

	"satellion.com/passmcp-lsp/internal/jsondoc"
	"satellion.com/passmcp-reporting/a2a"
	"satellion.com/passmcp-reporting/attestation"
)

// predicatePrefix is shared by every predicate type passmcp publishes.
const predicatePrefix = "https://satellion.com/attestation/"

// verifiers are passmcp-reporting's own structural verifiers, one per
// predicate type. This package decides nothing about an attestation
// itself: the verifier is the single source of what a valid one is.
var verifiers = map[string]func([]byte) error{
	attestation.PredicateType: func(b []byte) error { _, err := attestation.Parse(b); return err },
	a2a.PredicateType:         func(b []byte) error { _, err := a2a.Parse(b); return err },
}

// analyzeAttestation verifies an attestation's structure and integrity
// offline with passmcp-reporting. Who signed it is the envelope's
// business, not the statement's.
func analyzeAttestation(root *jsondoc.Node, src []byte, _ string) []Diagnostic {
	pt := root.Get("predicateType")
	verify, ok := verifiers[pt.Str]
	if !ok {
		return []Diagnostic{at(pt, Warning, "attestation/predicate-type",
			"predicate type %q is not one passmcp-reporting 0.0.2 verifies", pt.Str)}
	}
	err := verify(src)
	if err == nil {
		return nil
	}
	var out []Diagnostic
	for _, problem := range problems(err.Error()) {
		out = append(out, at(locate(root, problem), Error, "attestation/invalid", "%s", problem))
	}
	return out
}

// problems splits a verifier error into its individual findings. The
// verifiers join them with "; " after a package prefix.
func problems(msg string) []string {
	for _, prefix := range []string{"attestation: ", "a2a: "} {
		msg = strings.TrimPrefix(msg, prefix)
	}
	var out []string
	for _, p := range strings.Split(msg, "; ") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// problemAnchors maps words a verifier uses to the part of the statement
// the problem is about, most specific first, so the underline lands on it.
var problemAnchors = []struct {
	word string
	path []string
}{
	{"predicateType", []string{"predicateType"}},
	{"_type", []string{"_type"}},
	{"subjectKind", []string{"predicate", "subjectKind"}},
	{"subject", []string{"subject"}},
	{"judgedAgainst", []string{"predicate", "judgedAgainst"}},
	{"rubric", []string{"predicate", "judgedAgainst"}},
	{"instrument", []string{"predicate", "instrument"}},
	{"run time", []string{"predicate", "ranAt"}},
	{"verdict", []string{"predicate", "verdicts"}},
	{"count", []string{"predicate", "counts"}},
	{"score", []string{"predicate", "score"}},
	{"transport", []string{"predicate", "target"}},
	{"endpoint", []string{"predicate", "target"}},
	{"target", []string{"predicate", "target"}},
}

// locate finds the key a problem is about, so the underline is one word
// rather than a whole block, falling back to the first character of the
// document.
func locate(root *jsondoc.Node, problem string) *jsondoc.Node {
	for _, a := range problemAnchors {
		if !strings.Contains(problem, a.word) {
			continue
		}
		if m := memberAt(root, a.path); m != nil {
			return &jsondoc.Node{Start: m.KeyStart, End: m.KeyEnd}
		}
	}
	return head(root)
}

// memberAt follows keys from root and returns the last one's member.
func memberAt(root *jsondoc.Node, keys []string) *jsondoc.Member {
	n := root
	for _, k := range keys[:len(keys)-1] {
		if n = n.Get(k); n == nil {
			return nil
		}
	}
	return n.Member(keys[len(keys)-1])
}
