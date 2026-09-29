// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package jsondoc parses JSON, and the JSON with comments that editors
// write, into a tree that remembers where every value and every key sits in
// the text. A diagnostic is only useful when it points at the characters
// that caused it, and encoding/json forgets them.
//
// The parser is strict: anything encoding/json refuses, it refuses, except
// that Options.Comments admits // and /* */ comments and trailing commas,
// which is the dialect VS Code's configuration files use.
package jsondoc

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// MaxDepth bounds nesting, so a hostile document of ten thousand opening
// brackets is a syntax error rather than a stack overflow.
const MaxDepth = 256

// Kind is the JSON type of a node.
type Kind uint8

// The kinds of JSON value.
const (
	Invalid Kind = iota
	Object
	Array
	String
	Number
	Bool
	Null
)

var kindNames = [...]string{"invalid", "object", "array", "string", "number", "boolean", "null"}

// String names the kind the way JSON Schema's "type" keyword does.
func (k Kind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return "invalid"
}

// Node is one JSON value and the byte range [Start, End) it occupies.
type Node struct {
	Kind       Kind
	Start, End int
	// Str is the decoded text of a String.
	Str string
	// Raw is the literal text of a Number.
	Raw string
	// Bool is the value of a Bool.
	Bool bool
	// Members are an Object's members in document order, duplicates kept.
	Members []*Member
	// Items are an Array's elements.
	Items []*Node
}

// Member is one key of an object and the value it holds.
type Member struct {
	Key              string
	KeyStart, KeyEnd int
	Value            *Node
}

// Get returns the value of the first member named key, or nil when n is
// not an object or has no such member.
func (n *Node) Get(key string) *Node {
	if m := n.Member(key); m != nil {
		return m.Value
	}
	return nil
}

// Member returns the first member named key, or nil.
func (n *Node) Member(key string) *Member {
	if n == nil || n.Kind != Object {
		return nil
	}
	for _, m := range n.Members {
		if m.Key == key {
			return m
		}
	}
	return nil
}

// Float returns a Number's value.
func (n *Node) Float() (float64, bool) {
	if n == nil || n.Kind != Number {
		return 0, false
	}
	f, err := strconv.ParseFloat(n.Raw, 64)
	return f, err == nil
}

// IsInteger reports whether n is a Number with no fractional part.
func (n *Node) IsInteger() bool {
	f, ok := n.Float()
	return ok && f == float64(int64(f))
}

// Value converts the node to the value encoding/json would decode it to,
// with numbers as float64.
func (n *Node) Value() any {
	switch n.Kind {
	case Object:
		m := make(map[string]any, len(n.Members))
		for _, mem := range n.Members {
			m[mem.Key] = mem.Value.Value()
		}
		return m
	case Array:
		a := make([]any, len(n.Items))
		for i, it := range n.Items {
			a[i] = it.Value()
		}
		return a
	case String:
		return n.Str
	case Number:
		f, _ := n.Float()
		return f
	case Bool:
		return n.Bool
	}
	return nil
}

// Options select the dialect.
type Options struct {
	// Comments admits // and /* */ comments and trailing commas.
	Comments bool
}

// SyntaxError is a document that is not JSON, and where it stops being so.
type SyntaxError struct {
	Offset int
	Msg    string
}

func (e *SyntaxError) Error() string { return fmt.Sprintf("offset %d: %s", e.Offset, e.Msg) }

// Parse reads one JSON document. Anything after it but whitespace (and, in
// the comment dialect, comments) is an error.
func Parse(src []byte, opt Options) (*Node, error) {
	p := &parser{src: src, opt: opt}
	n, err := p.value(0)
	if err != nil {
		return nil, err
	}
	if err := p.skip(); err != nil {
		return nil, err
	}
	if p.pos < len(p.src) {
		return nil, p.fail("unexpected text after the document")
	}
	return n, nil
}

type parser struct {
	src []byte
	pos int
	opt Options
}

func (p *parser) fail(format string, a ...any) *SyntaxError {
	return &SyntaxError{Offset: p.pos, Msg: fmt.Sprintf(format, a...)}
}

// skip moves past whitespace and, in the comment dialect, comments.
func (p *parser) skip() error {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		case '/':
			if err := p.comment(); err != nil {
				return err
			}
		default:
			return nil
		}
	}
	return nil
}

// comment moves past one comment at p.pos.
func (p *parser) comment() error {
	if !p.opt.Comments {
		return p.fail("comments are not allowed in this file")
	}
	if p.pos+1 >= len(p.src) {
		return p.fail("a lone '/' is not a comment")
	}
	switch p.src[p.pos+1] {
	case '/':
		for p.pos < len(p.src) && p.src[p.pos] != '\n' {
			p.pos++
		}
		return nil
	case '*':
		for i := p.pos + 2; i+1 < len(p.src); i++ {
			if p.src[i] == '*' && p.src[i+1] == '/' {
				p.pos = i + 2
				return nil
			}
		}
		return p.fail("the comment is never closed")
	}
	return p.fail("a lone '/' is not a comment")
}

func (p *parser) value(depth int) (*Node, error) {
	if depth > MaxDepth {
		return nil, p.fail("nested more than %d levels deep", MaxDepth)
	}
	if err := p.skip(); err != nil {
		return nil, err
	}
	if p.pos >= len(p.src) {
		return nil, p.fail("unexpected end of the document")
	}
	switch c := p.src[p.pos]; {
	case c == '{':
		return p.object(depth)
	case c == '[':
		return p.array(depth)
	case c == '"':
		return p.stringNode()
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	default:
		return p.literal()
	}
}

func (p *parser) object(depth int) (*Node, error) {
	n := &Node{Kind: Object, Start: p.pos}
	p.pos++ // {
	for {
		if err := p.skip(); err != nil {
			return nil, err
		}
		if p.pos < len(p.src) && p.src[p.pos] == '}' && (len(n.Members) == 0 || p.opt.Comments) {
			p.pos++
			n.End = p.pos
			return n, nil
		}
		m, err := p.member(depth)
		if err != nil {
			return nil, err
		}
		n.Members = append(n.Members, m)
		done, err := p.separator('}')
		if err != nil {
			return nil, err
		}
		if done {
			n.End = p.pos
			return n, nil
		}
	}
}

func (p *parser) member(depth int) (*Member, error) {
	if p.pos >= len(p.src) || p.src[p.pos] != '"' {
		return nil, p.fail("expected a quoted key")
	}
	key, err := p.stringNode()
	if err != nil {
		return nil, err
	}
	if err := p.skip(); err != nil {
		return nil, err
	}
	if p.pos >= len(p.src) || p.src[p.pos] != ':' {
		return nil, p.fail("expected ':' after the key %q", key.Str)
	}
	p.pos++
	v, err := p.value(depth + 1)
	if err != nil {
		return nil, err
	}
	return &Member{Key: key.Str, KeyStart: key.Start, KeyEnd: key.End, Value: v}, nil
}

// separator reads the ',' between elements or the closing bracket, and
// reports whether the container ended.
func (p *parser) separator(closing byte) (bool, error) {
	if err := p.skip(); err != nil {
		return false, err
	}
	if p.pos >= len(p.src) {
		return false, p.fail("unexpected end of the document: expected ',' or '%c'", closing)
	}
	switch p.src[p.pos] {
	case ',':
		p.pos++
		return false, nil
	case closing:
		p.pos++
		return true, nil
	}
	return false, p.fail("expected ',' or '%c'", closing)
}

func (p *parser) array(depth int) (*Node, error) {
	n := &Node{Kind: Array, Start: p.pos}
	p.pos++ // [
	for {
		if err := p.skip(); err != nil {
			return nil, err
		}
		if p.pos < len(p.src) && p.src[p.pos] == ']' && (len(n.Items) == 0 || p.opt.Comments) {
			p.pos++
			n.End = p.pos
			return n, nil
		}
		v, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		n.Items = append(n.Items, v)
		done, err := p.separator(']')
		if err != nil {
			return nil, err
		}
		if done {
			n.End = p.pos
			return n, nil
		}
	}
}

// stringNode reads the string opening at p.pos. It finds the closing quote
// itself and lets encoding/json decode the escapes, so what counts as a
// valid string is exactly what the standard library says.
func (p *parser) stringNode() (*Node, error) {
	start := p.pos
	esc := false
	for i := start + 1; i < len(p.src); i++ {
		switch c := p.src[i]; {
		case esc:
			esc = false
		case c == '\\':
			esc = true
		case c == '"':
			var s string
			if err := json.Unmarshal(p.src[start:i+1], &s); err != nil {
				return nil, p.fail("invalid string: %v", err)
			}
			p.pos = i + 1
			return &Node{Kind: String, Start: start, End: p.pos, Str: s}, nil
		}
	}
	return nil, p.fail("the string is never closed")
}

func (p *parser) number() (*Node, error) {
	start := p.pos
	for p.pos < len(p.src) && isNumberByte(p.src[p.pos]) {
		p.pos++
	}
	raw := p.src[start:p.pos]
	if !json.Valid(raw) {
		p.pos = start
		return nil, p.fail("invalid number %q", raw)
	}
	return &Node{Kind: Number, Start: start, End: p.pos, Raw: string(raw)}, nil
}

func isNumberByte(c byte) bool {
	return (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E'
}

func (p *parser) literal() (*Node, error) {
	for _, l := range []struct {
		text string
		node Node
	}{
		{"true", Node{Kind: Bool, Bool: true}},
		{"false", Node{Kind: Bool}},
		{"null", Node{Kind: Null}},
	} {
		end := p.pos + len(l.text)
		if end <= len(p.src) && string(p.src[p.pos:end]) == l.text {
			n := l.node
			n.Start, n.End = p.pos, end
			p.pos = end
			return &n, nil
		}
	}
	return nil, p.fail("unexpected character %q", p.src[p.pos])
}
