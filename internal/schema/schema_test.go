// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"strings"
	"testing"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

const testSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$ref": "#/definitions/Root",
  "definitions": {
    "Root": {
      "type": "object",
      "required": ["name", "version"],
      "properties": {
        "name": {"type": "string", "minLength": 3, "maxLength": 8, "pattern": "^[a-z]+$"},
        "version": {"type": "string", "not": {"const": "latest"}},
        "count": {"type": "integer"},
        "ratio": {"type": ["number", "null"]},
        "kind": {"enum": ["a", "b"]},
        "fixed": {"const": 1},
        "site": {"type": "string", "format": "uri"},
        "tags": {"type": "array", "items": {"type": "string"}},
        "strict": {"type": "object", "additionalProperties": false, "properties": {"ok": {"type": "boolean"}}},
        "loose": {"type": "object", "additionalProperties": {"type": "number"}},
        "either": {"anyOf": [{"type": "string"}, {"type": "object", "required": ["x", "y"]}]},
        "both": {"allOf": [{"required": ["p"]}, {"required": ["q"]}]},
        "never": false,
        "always": true,
        "shape": {"not": {"type": "string"}},
        "a~b/c": {"type": "boolean"}
      }
    }
  }
}`

func mustCompile(t *testing.T, s string) *Schema {
	t.Helper()
	c, err := Compile([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func validate(t *testing.T, s *Schema, doc string) []Violation {
	t.Helper()
	n, err := jsondoc.Parse([]byte(doc), jsondoc.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return s.Validate(n)
}

func TestValidDocument(t *testing.T) {
	s := mustCompile(t, testSchema)
	doc := `{"name":"abc","version":"1.0","count":3,"ratio":null,"kind":"a","fixed":1.0,"site":"https://x.example/a",
	  "tags":["t"],"strict":{"ok":true},"loose":{"n":1},"either":{"x":1,"y":2},"both":{"p":1,"q":2},"always":[],"shape":1,"a~b/c":true}`
	if got := validate(t, s, doc); len(got) != 0 {
		t.Fatalf("want no violations, got %+v", got)
	}
}

func TestEveryKeywordReports(t *testing.T) {
	s := mustCompile(t, testSchema)
	cases := []struct {
		doc, path, msg string
	}{
		{`{"version":"1"}`, "(document)", `missing the required property "name"`},
		{`{"name":"ab","version":"1"}`, "name", "at least 3 characters"},
		{`{"name":"abcdefghi","version":"1"}`, "name", "at most 8 characters"},
		{`{"name":"ABC","version":"1"}`, "name", "must match the pattern"},
		{`{"name":"abc","version":"latest"}`, "version", `must not be "latest"`},
		{`{"name":"abc","version":"1","count":1.5}`, "count", "must be integer, not number"},
		{`{"name":"abc","version":"1","ratio":"x"}`, "ratio", "must be number or null, not string"},
		{`{"name":"abc","version":"1","kind":"c"}`, "kind", `must be one of "a", "b"`},
		{`{"name":"abc","version":"1","fixed":2}`, "fixed", "must be 1"},
		{`{"name":"abc","version":"1","site":"not a uri"}`, "site", "absolute URI"},
		{`{"name":"abc","version":"1","tags":["a",2]}`, "tags[1]", "must be string, not number"},
		{`{"name":"abc","version":"1","strict":{"no":1}}`, "strict.no", `"no" is not a property`},
		{`{"name":"abc","version":"1","loose":{"n":"x"}}`, "loose.n", "must be number"},
		{`{"name":"abc","version":"1","either":{"x":1}}`, "either", `missing the required property "y"`},
		{`{"name":"abc","version":"1","both":{"p":1}}`, "both", `missing the required property "q"`},
		{`{"name":"abc","version":"1","never":1}`, "never", "no value is allowed"},
		{`{"name":"abc","version":"1","shape":"s"}`, "shape", "does not allow"},
		{`{"name":"abc","version":"1","a~b/c":1}`, "a~b/c", "must be boolean"},
		{`[]`, "(document)", "must be object, not array"},
	}
	for _, c := range cases {
		got := validate(t, s, c.doc)
		if len(got) != 1 || got[0].Path != c.path || !strings.Contains(got[0].Msg, c.msg) {
			t.Errorf("%s: want one violation at %s containing %q, got %+v", c.doc, c.path, c.msg, got)
		}
	}
}

func TestViolationRanges(t *testing.T) {
	s := mustCompile(t, testSchema)
	doc := `{"name":"abc","version":"1","strict":{"no":1},"kind":"z"}`
	got := validate(t, s, doc)
	if len(got) != 2 {
		t.Fatalf("want 2, got %+v", got)
	}
	if doc[got[0].Start:got[0].End] != `"no"` || doc[got[1].Start:got[1].End] != `"z"` {
		t.Fatalf("ranges %q %q, sorted by position", doc[got[0].Start:got[0].End], doc[got[1].Start:got[1].End])
	}
}

func TestCompileRefusals(t *testing.T) {
	for doc, want := range map[string]string{
		`{"properties":{"a":{"minimum":1}}}`:       `uses "minimum"`,
		`{"anyOf":[{"uniqueItems":true}]}`:         `#/anyOf/0 uses "uniqueItems"`,
		`{"pattern":"("}`:                          "pattern",
		`{"pattern":1}`:                            "not a string",
		`not json`:                                 "schema:",
		`{"definitions":{"x":{"if":{}}}}`:          `uses "if"`,
		`{"items":{"additionalItems":false}}`:      `uses "additionalItems"`,
		`{"not":{"not":{"patternProperties":{}}}}`: "patternProperties",
	} {
		if _, err := Compile([]byte(doc)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Compile(%s) = %v, want an error containing %q", doc, err, want)
		}
	}
}

func TestBrokenReferences(t *testing.T) {
	loop := mustCompile(t, `{"$ref":"#/definitions/a","definitions":{"a":{"$ref":"#/definitions/a"}}}`)
	if got := validate(t, loop, `1`); len(got) != 1 || !strings.Contains(got[0].Msg, "loop") {
		t.Fatalf("loop: %+v", got)
	}
	for _, ref := range []string{"#/definitions/missing", "other.json#/x", "#/definitions/a/deeper"} {
		s := mustCompile(t, `{"$ref":"`+ref+`","definitions":{"a":1}}`)
		if got := validate(t, s, `1`); len(got) != 1 || !strings.Contains(got[0].Msg, "does not define") {
			t.Errorf("%s: %+v", ref, got)
		}
	}
}

func TestShowFallsBack(t *testing.T) {
	if show(func() {}) == "" {
		t.Fatal("show must render something for an unmarshalable value")
	}
}
