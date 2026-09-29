// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"strings"
	"testing"
)

func TestToolsValid(t *testing.T) {
	src := `{"tools": [
  {"name": "get_weather", "description": "Forecast", "inputSchema": {"type": "object", "properties": {"city": {"type": "string"}}, "required": ["city"]},
   "outputSchema": {"type": "object", "properties": {"t": {"type": ["number", "null"]}}}, "annotations": {"readOnlyHint": true}},
  {"name": "admin.tools.list", "description": "d", "inputSchema": {"type": "object", "additionalProperties": false},
   "annotations": {"readOnlyHint": false, "destructiveHint": false}, "execution": {"taskSupport": "optional"}}
]}`
	if r := Analyze("tools.json", []byte(src)); r.Kind != Tools || len(r.Diagnostics) != 0 {
		t.Fatalf("%q %+v", r.Kind, r.Diagnostics)
	}
}

func TestToolsProblems(t *testing.T) {
	src := `[
  {"inputSchema": {"type": "object"}},
  {"name": 5, "inputSchema": {"type": "object"}},
  {"name": "has space", "inputSchema": {"type": "array"}, "annotations": {"readOnlyHint": "yes"}},
  {"name": "dup", "inputSchema": {"type": "object"}},
  {"name": "dup", "inputSchema": {"properties": {}}},
  {"name": "noinput"},
  {"name": "req", "inputSchema": {"type": "object", "properties": {"a": {"type": "text"}, "b": {"items": {"type": ["string", 3]}}}, "required": ["a", "z"]}},
  {"name": "reqtype", "inputSchema": {"type": "object", "required": "a"}},
  {"name": "out", "inputSchema": {"type": "object"}, "outputSchema": [], "annotations": {"readOnlyHint": true, "destructiveHint": true}},
  {"name": "ann", "inputSchema": {"type": "object"}, "annotations": 1},
  {"name": "exec", "inputSchema": {"type": "object"}, "execution": {"taskSupport": "sometimes"}},
  {"name": "execn", "inputSchema": {"type": "object"}, "execution": {"taskSupport": 1}},
  {"name": "nested", "inputSchema": {"type": "object", "anyOf": [{"type": "x"}], "$defs": {"d": {"type": "y"}}, "not": {"type": "z"}}},
  "str"
]`
	r := Analyze("tools.json", []byte(src))
	for _, want := range []struct {
		sev        Severity
		code, text string
	}{
		{Error, "tool/name", "has no name"},
		{Error, "tool/name", "name must be a string"},
		{Warning, "tool/name", "should be 1 to 128 characters"},
		{Error, "tool/input-schema", `must have "type": "object"`},
		{Error, "tool/annotations", "readOnlyHint must be true or false"},
		{Warning, "tool/name", `already named "dup"`},
		{Error, "tool/input-schema", "has no inputSchema"},
		{Error, "tool/schema-type", `inputSchema.properties.a.type: "text"`},
		{Error, "tool/schema-type", `inputSchema.properties.b.items.type: number`},
		{Warning, "tool/required-undeclared", `"z" is required`},
		{Error, "tool/input-schema", "required must be an array"},
		{Error, "tool/output-schema", "outputSchema must be a JSON Schema object"},
		{Warning, "tool/annotations", "destructiveHint is meaningful only"},
		{Error, "tool/annotations", "annotations must be an object"},
		{Error, "tool/execution", "taskSupport"},
		{Error, "tool/schema-type", `inputSchema.anyOf[0].type`},
		{Error, "tool/schema-type", `inputSchema.$defs.d.type`},
		{Error, "tool/schema-type", `inputSchema.not.type`},
		{Error, "tool/shape", "a tool must be an object"},
		{Information, "tool/read-only-hint", "no readOnlyHint"},
		{Information, "tool/description", "no description"},
	} {
		if !has(r.Diagnostics, want.sev, want.code, want.text) {
			t.Errorf("missing %s %s %q", want.sev, want.code, want.text)
		}
	}
	execErrors := 0
	for _, d := range r.Diagnostics {
		if d.Code == "tool/execution" {
			execErrors++
		}
	}
	if execErrors != 2 {
		t.Errorf("both bad taskSupport values: %d", execErrors)
	}
	if t.Failed() {
		t.Log(strings.Join(codes(r.Diagnostics), "\n"))
	}
}
