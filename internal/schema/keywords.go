// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// checkValue applies enum and const.
func (v *validator) checkValue(m map[string]any, n *jsondoc.Node, path jsondoc.Path) []Violation {
	var out []Violation
	if c, ok := m["const"]; ok && !equal(n.Value(), c) {
		out = append(out, violation(n, path, "must be "+show(c)))
	}
	if e, ok := m["enum"].([]any); ok && !inEnum(n.Value(), e) {
		shown := make([]string, len(e))
		for i, x := range e {
			shown[i] = show(x)
		}
		out = append(out, violation(n, path, "must be one of "+strings.Join(shown, ", ")))
	}
	return out
}

func inEnum(val any, e []any) bool {
	for _, x := range e {
		if equal(val, x) {
			return true
		}
	}
	return false
}

// equal compares two decoded JSON values. Both sides hold numbers as
// float64, so 1 and 1.0 are the same value, as JSON Schema says.
func equal(a, b any) bool { return reflect.DeepEqual(a, b) }

func show(x any) string {
	b, err := json.Marshal(x)
	if err != nil {
		return fmt.Sprint(x)
	}
	return string(b)
}

// checkString applies minLength, maxLength, pattern and format.
func (v *validator) checkString(m map[string]any, n *jsondoc.Node, path jsondoc.Path) []Violation {
	if n.Kind != jsondoc.String {
		return nil
	}
	var out []Violation
	runes := utf8.RuneCountInString(n.Str)
	if lim, ok := m["minLength"].(float64); ok && float64(runes) < lim {
		out = append(out, violation(n, path, fmt.Sprintf("must be at least %d characters long; it is %d", int(lim), runes)))
	}
	if lim, ok := m["maxLength"].(float64); ok && float64(runes) > lim {
		out = append(out, violation(n, path, fmt.Sprintf("must be at most %d characters long; it is %d", int(lim), runes)))
	}
	return append(out, v.checkShape(m, n, path)...)
}

// checkShape applies pattern and format.
func (v *validator) checkShape(m map[string]any, n *jsondoc.Node, path jsondoc.Path) []Violation {
	var out []Violation
	if p, ok := m["pattern"].(string); ok && !v.s.patterns[p].MatchString(n.Str) {
		out = append(out, violation(n, path, fmt.Sprintf("must match the pattern %s", p)))
	}
	if f, ok := m["format"].(string); ok && f == "uri" && !isURI(n.Str) {
		out = append(out, violation(n, path, "must be an absolute URI"))
	}
	return out
}

// isURI accepts an absolute URI: a scheme, and a parse the standard
// library agrees with. Other formats are annotations under draft-07 and
// are not asserted.
func isURI(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && !strings.ContainsAny(s, " \t\n")
}

// checkObject applies required, properties and additionalProperties.
func (v *validator) checkObject(m map[string]any, n *jsondoc.Node, path jsondoc.Path, depth int) []Violation {
	if n.Kind != jsondoc.Object {
		return nil
	}
	var out []Violation
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if name, _ := r.(string); n.Member(name) == nil {
				out = append(out, violation(n, path, fmt.Sprintf("is missing the required property %q", name)))
			}
		}
	}
	props, _ := m["properties"].(map[string]any)
	extra, hasExtra := m["additionalProperties"]
	for _, mem := range n.Members {
		sub, declared := props[mem.Key]
		switch {
		case declared:
			out = append(out, v.check(sub, mem.Value, path.Key(mem.Key), depth)...)
		case extra == false:
			out = append(out, Violation{Start: mem.KeyStart, End: mem.KeyEnd, Path: path.Key(mem.Key).String(),
				Msg: fmt.Sprintf("%q is not a property this object may have", mem.Key)})
		case hasExtra:
			out = append(out, v.check(extra, mem.Value, path.Key(mem.Key), depth)...)
		}
	}
	return out
}

// checkItems applies items, in its single-schema form.
func (v *validator) checkItems(m map[string]any, n *jsondoc.Node, path jsondoc.Path, depth int) []Violation {
	items, ok := m["items"]
	if !ok || n.Kind != jsondoc.Array {
		return nil
	}
	var out []Violation
	for i, it := range n.Items {
		out = append(out, v.check(items, it, path.Index(i), depth)...)
	}
	return out
}

// checkCombinators applies allOf, anyOf and not.
func (v *validator) checkCombinators(m map[string]any, n *jsondoc.Node, path jsondoc.Path, depth int) []Violation {
	var out []Violation
	if all, ok := m["allOf"].([]any); ok {
		for _, sub := range all {
			out = append(out, v.check(sub, n, path, depth)...)
		}
	}
	if anyOf, ok := m["anyOf"].([]any); ok {
		out = append(out, v.anyOf(anyOf, n, path, depth)...)
	}
	if not, ok := m["not"]; ok && len(v.check(not, n, path, depth)) == 0 {
		msg := "is a value this field does not allow"
		if nm, ok := not.(map[string]any); ok {
			if c, ok := nm["const"]; ok {
				msg = "must not be " + show(c)
			}
		}
		out = append(out, violation(n, path, msg))
	}
	return out
}

// anyOf passes when one branch passes. When none does, one branch is
// reported: a branch whose type matched before one that did not, then the
// one with the fewest violations. It is the shape the author most likely
// meant, and its complaints are the ones that fix the document.
func (v *validator) anyOf(branches []any, n *jsondoc.Node, path jsondoc.Path, depth int) []Violation {
	var best []Violation
	for i, sub := range branches {
		got := v.check(sub, n, path, depth)
		if len(got) == 0 {
			return nil
		}
		if i == 0 || rank(got) < rank(best) {
			best = got
		}
	}
	return best
}

// rank orders a branch's failures for anyOf: lower is a closer match.
func rank(vs []Violation) int {
	if len(vs) == 1 && vs[0].wrongType {
		return 1 << 20
	}
	return len(vs)
}
