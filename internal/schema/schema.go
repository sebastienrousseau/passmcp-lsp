// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package schema validates a jsondoc tree against a JSON Schema, so that
// every violation carries the range of the text that caused it.
//
// It implements the subset of draft-07 that the schemas this server embeds
// actually use, and no more. It is fail-closed: Compile refuses a schema
// that uses a keyword outside that subset, because a validator that skips
// what it does not understand reports a document valid that is not. The
// supported keywords are listed in Supported.
package schema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// Supported lists the assertion and applicator keywords this validator
// implements.
var Supported = []string{
	"$ref", "additionalProperties", "allOf", "anyOf", "const", "enum", "format",
	"items", "maxLength", "minLength", "not", "pattern", "properties", "required", "type",
}

// annotations are keywords that assert nothing, so ignoring them is
// correct rather than lenient.
var annotations = []string{
	"$comment", "$id", "$schema", "default", "definitions", "description", "example", "examples", "title",
}

// Violation is one way a document fails its schema.
type Violation struct {
	// Start and End are the byte range to underline.
	Start, End int
	// Path names the location, like packages[0].transport.type.
	Path string
	Msg  string
	// wrongType marks a value of the wrong JSON type, which anyOf ranks
	// below every other failure when it picks a branch to report.
	wrongType bool
}

// Schema is a compiled schema document.
type Schema struct {
	root     any
	patterns map[string]*regexp.Regexp
}

// Compile reads a schema and checks that this validator can enforce all of
// it.
func Compile(b []byte) (*Schema, error) {
	var root any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	s := &Schema{root: root, patterns: map[string]*regexp.Regexp{}}
	if err := s.prepare(root, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// prepare walks every subschema, refusing unknown keywords and compiling
// patterns once.
func (s *Schema) prepare(v any, at string) error {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	for k, sub := range m {
		if err := s.prepareKeyword(k, sub, at); err != nil {
			return err
		}
	}
	return nil
}

// holders says how each keyword that holds subschemas holds them.
var holders = map[string]string{
	"properties": "map", "definitions": "map",
	"allOf": "list", "anyOf": "list",
	"items": "one", "not": "one", "additionalProperties": "one",
}

func (s *Schema) prepareKeyword(k string, sub any, at string) error {
	switch holders[k] {
	case "map":
		return s.prepareMap(sub, at+"/"+k)
	case "list":
		return s.prepareList(sub, at+"/"+k)
	case "one":
		return s.prepare(sub, at+"/"+k)
	}
	switch {
	case k == "pattern":
		return s.compilePattern(sub, at)
	case contains(Supported, k) || contains(annotations, k):
		return nil
	}
	return fmt.Errorf("schema: %s uses %q, which this validator does not implement", orRoot(at), k)
}

func (s *Schema) prepareMap(v any, at string) error {
	m, _ := v.(map[string]any)
	for name, sub := range m {
		if err := s.prepare(sub, at+"/"+name); err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) prepareList(v any, at string) error {
	l, _ := v.([]any)
	for i, sub := range l {
		if err := s.prepare(sub, fmt.Sprintf("%s/%d", at, i)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) compilePattern(v any, at string) error {
	p, ok := v.(string)
	if !ok {
		return fmt.Errorf("schema: %s has a pattern that is not a string", orRoot(at))
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return fmt.Errorf("schema: %s: pattern %q: %w", orRoot(at), p, err)
	}
	s.patterns[p] = re
	return nil
}

func orRoot(at string) string {
	if at == "" {
		return "#"
	}
	return "#" + at
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Validate returns every violation of the schema by n, sorted by position.
func (s *Schema) Validate(n *jsondoc.Node) []Violation {
	v := &validator{s: s}
	out := v.check(s.root, n, nil, 0)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// maxRefDepth stops a schema whose references loop.
const maxRefDepth = 64

type validator struct{ s *Schema }

func (v *validator) check(sch any, n *jsondoc.Node, path jsondoc.Path, depth int) []Violation {
	if b, ok := sch.(bool); ok {
		if b {
			return nil
		}
		return []Violation{violation(n, path, "no value is allowed here")}
	}
	m, _ := sch.(map[string]any)
	if ref, ok := m["$ref"].(string); ok {
		return v.ref(ref, n, path, depth)
	}
	if out := v.checkType(m, n, path); out != nil {
		// A value of the wrong type fails every other keyword too; one
		// message says it.
		return out
	}
	var out []Violation
	out = append(out, v.checkValue(m, n, path)...)
	out = append(out, v.checkString(m, n, path)...)
	out = append(out, v.checkObject(m, n, path, depth)...)
	out = append(out, v.checkItems(m, n, path, depth)...)
	out = append(out, v.checkCombinators(m, n, path, depth)...)
	return out
}

func (v *validator) ref(ref string, n *jsondoc.Node, path jsondoc.Path, depth int) []Violation {
	if depth >= maxRefDepth {
		return []Violation{violation(n, path, "the schema's references loop at "+ref)}
	}
	target, ok := resolve(v.s.root, ref)
	if !ok {
		return []Violation{violation(n, path, "the schema refers to "+ref+", which it does not define")}
	}
	return v.check(target, n, path, depth+1)
}

// resolve follows a local JSON Pointer reference, "#/definitions/X".
func resolve(root any, ref string) (any, bool) {
	if !strings.HasPrefix(ref, "#") {
		return nil, false
	}
	cur := root
	for _, tok := range strings.Split(strings.TrimPrefix(ref, "#"), "/")[1:] {
		tok = strings.ReplaceAll(strings.ReplaceAll(tok, "~1", "/"), "~0", "~")
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = m[tok]; !ok {
			return nil, false
		}
	}
	return cur, true
}

func violation(n *jsondoc.Node, path jsondoc.Path, msg string) Violation {
	return Violation{Start: n.Start, End: n.End, Path: path.String(), Msg: msg}
}

func (v *validator) checkType(m map[string]any, n *jsondoc.Node, path jsondoc.Path) []Violation {
	var types []string
	switch t := m["type"].(type) {
	case string:
		types = []string{t}
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok {
				types = append(types, s)
			}
		}
	default:
		return nil
	}
	for _, t := range types {
		if hasType(n, t) {
			return nil
		}
	}
	out := violation(n, path, fmt.Sprintf("must be %s, not %s", strings.Join(types, " or "), n.Kind))
	out.wrongType = true
	return []Violation{out}
}

func hasType(n *jsondoc.Node, t string) bool {
	switch t {
	case "integer":
		return n.IsInteger()
	case "number":
		return n.Kind == jsondoc.Number
	}
	return n.Kind.String() == t
}
