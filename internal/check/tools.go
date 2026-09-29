// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"regexp"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// The rules below are the Tool definition of the MCP specification,
// revision 2025-11-25 (schema.ts and server/tools): a name, an inputSchema
// whose root is type "object", an optional outputSchema restricted the same
// way, boolean annotation hints, and the naming guidance of "Tool Names".
// A MUST becomes an error, a SHOULD a warning.

// SpecRevision is the MCP revision the tool rules come from.
const SpecRevision = "2025-11-25"

// isToolsDocument recognises a single tool, an array of tools, or a
// tools/list result.
func isToolsDocument(root *jsondoc.Node) bool {
	return len(toolsIn(root)) > 0
}

func toolsIn(root *jsondoc.Node) []*jsondoc.Node {
	switch {
	case root.Kind == jsondoc.Object && root.Get("inputSchema") != nil:
		return []*jsondoc.Node{root}
	case root.Kind == jsondoc.Object && root.Get("tools") != nil:
		return toolsIn(root.Get("tools"))
	case root.Kind == jsondoc.Array && len(root.Items) > 0 && root.Items[0].Get("inputSchema") != nil:
		return root.Items
	}
	return nil
}

// toolName is the character set the specification says tool names SHOULD
// keep to, at the length it says they SHOULD have.
var toolName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

// analyzeTools checks MCP tool definitions.
func analyzeTools(root *jsondoc.Node, _ []byte, _ string) []Diagnostic {
	var out []Diagnostic
	names := map[string]bool{}
	for _, t := range toolsIn(root) {
		out = append(out, checkTool(t, names)...)
	}
	return out
}

func checkTool(t *jsondoc.Node, names map[string]bool) []Diagnostic {
	if t.Kind != jsondoc.Object {
		return []Diagnostic{at(t, Error, "tool/shape", "a tool must be an object")}
	}
	out := checkToolName(t, names)
	in := t.Get("inputSchema")
	if in == nil {
		out = append(out, at(head(t), Error, "tool/input-schema", "the tool has no inputSchema; a tool with no parameters uses {\"type\": \"object\", \"additionalProperties\": false}"))
	} else {
		out = append(out, rootSchema(in, "inputSchema")...)
	}
	if o := t.Get("outputSchema"); o != nil {
		out = append(out, rootSchema(o, "outputSchema")...)
	}
	if d := t.Get("description"); d == nil {
		out = append(out, at(head(t), Information, "tool/description", "the tool has no description; an agent chooses tools by their descriptions"))
	}
	out = append(out, checkAnnotations(t)...)
	return append(out, checkExecution(t.Get("execution"))...)
}

func checkToolName(t *jsondoc.Node, names map[string]bool) []Diagnostic {
	n := t.Get("name")
	switch {
	case n == nil:
		return []Diagnostic{at(head(t), Error, "tool/name", "the tool has no name")}
	case n.Kind != jsondoc.String:
		return []Diagnostic{at(n, Error, "tool/name", "name must be a string")}
	case names[n.Str]:
		return []Diagnostic{at(n, Warning, "tool/name", "another tool is already named %q; tool names should be unique within a server", n.Str)}
	}
	names[n.Str] = true
	if !toolName.MatchString(n.Str) {
		return []Diagnostic{at(n, Warning, "tool/name",
			"tool names should be 1 to 128 characters of A-Z, a-z, 0-9, underscore, hyphen and dot (MCP %s)", SpecRevision)}
	}
	return nil
}

// rootSchema checks the root of an input or output schema: an object
// schema, whose required names are declared properties.
func rootSchema(s *jsondoc.Node, field string) []Diagnostic {
	if s.Kind != jsondoc.Object {
		return []Diagnostic{at(s, Error, "tool/"+schemaCode(field), "%s must be a JSON Schema object", field)}
	}
	var out []Diagnostic
	if ty := s.Get("type"); ty == nil || ty.Kind != jsondoc.String || ty.Str != "object" {
		target := head(s)
		if ty != nil {
			target = ty
		}
		out = append(out, at(target, Error, "tool/"+schemaCode(field), "%s must have \"type\": \"object\" at its root (MCP %s)", field, SpecRevision))
	}
	out = append(out, checkRequired(s, field)...)
	return append(out, schemaTypes(s, jsondoc.Path{}.Key(field))...)
}

func schemaCode(field string) string {
	if field == "outputSchema" {
		return "output-schema"
	}
	return "input-schema"
}

// checkRequired flags a required name no property declares: valid JSON
// Schema, but a client building arguments from the properties cannot
// supply it.
func checkRequired(s *jsondoc.Node, field string) []Diagnostic {
	req := s.Get("required")
	if req == nil {
		return nil
	}
	if req.Kind != jsondoc.Array {
		return []Diagnostic{at(req, Error, "tool/"+schemaCode(field), "%s.required must be an array of property names", field)}
	}
	props := s.Get("properties")
	var out []Diagnostic
	for _, it := range req.Items {
		if it.Kind == jsondoc.String && props.Member(it.Str) == nil {
			out = append(out, at(it, Warning, "tool/required-undeclared", "%q is required but %s.properties does not declare it", it.Str, field))
		}
	}
	return out
}

var jsonTypes = map[string]bool{"string": true, "number": true, "integer": true, "boolean": true, "object": true, "array": true, "null": true}

// schemaTypes checks every "type" keyword in a schema names a JSON Schema
// type, descending through the keywords that hold subschemas.
func schemaTypes(s *jsondoc.Node, path jsondoc.Path) []Diagnostic {
	if s.Kind != jsondoc.Object {
		return nil
	}
	var out []Diagnostic
	for _, m := range s.Members {
		switch m.Key {
		case "type":
			out = append(out, typeNames(m.Value, path)...)
		case "properties", "$defs", "definitions", "patternProperties":
			for _, p := range m.Value.Members {
				out = append(out, schemaTypes(p.Value, path.Key(m.Key).Key(p.Key))...)
			}
		case "items", "additionalProperties", "not", "contains":
			out = append(out, schemaTypes(m.Value, path.Key(m.Key))...)
		case "anyOf", "oneOf", "allOf", "prefixItems":
			for i, it := range m.Value.Items {
				out = append(out, schemaTypes(it, path.Key(m.Key).Index(i))...)
			}
		}
	}
	return out
}

func typeNames(v *jsondoc.Node, path jsondoc.Path) []Diagnostic {
	vals := []*jsondoc.Node{v}
	if v.Kind == jsondoc.Array {
		vals = v.Items
	}
	var out []Diagnostic
	for _, t := range vals {
		if t.Kind != jsondoc.String || !jsonTypes[t.Str] {
			out = append(out, at(t, Error, "tool/schema-type",
				"%s.type: %s is not a JSON Schema type (string, number, integer, boolean, object, array, null)", path, show(t)))
		}
	}
	return out
}

func show(n *jsondoc.Node) string {
	if n.Kind == jsondoc.String {
		return `"` + n.Str + `"`
	}
	return n.Kind.String()
}

var hints = []string{"readOnlyHint", "destructiveHint", "idempotentHint", "openWorldHint"}

// checkAnnotations checks the behaviour hints. A tool that does not say it
// is read-only is one passmcp, by default, never invokes.
func checkAnnotations(t *jsondoc.Node) []Diagnostic {
	a := t.Get("annotations")
	if a != nil && a.Kind != jsondoc.Object {
		return []Diagnostic{at(a, Error, "tool/annotations", "annotations must be an object")}
	}
	var out []Diagnostic
	for _, h := range hints {
		if v := a.Get(h); v != nil && v.Kind != jsondoc.Bool {
			out = append(out, at(v, Error, "tool/annotations", "%s must be true or false", h))
		}
	}
	ro, de := a.Get("readOnlyHint"), a.Get("destructiveHint")
	switch {
	case ro == nil:
		out = append(out, at(head(t), Information, "tool/read-only-hint",
			"no readOnlyHint: it defaults to false, so a client treats the tool as one that modifies its environment, and passmcp does not invoke it by default"))
	case ro.Bool && de != nil && de.Kind == jsondoc.Bool:
		out = append(out, at(de, Warning, "tool/annotations", "destructiveHint is meaningful only when readOnlyHint is false"))
	}
	return out
}

func checkExecution(e *jsondoc.Node) []Diagnostic {
	if e == nil {
		return nil
	}
	ts := e.Get("taskSupport")
	if ts == nil {
		return nil
	}
	switch ts.Str {
	case "forbidden", "optional", "required":
		if ts.Kind == jsondoc.String {
			return nil
		}
	}
	return []Diagnostic{at(ts, Error, "tool/execution", "execution.taskSupport must be \"forbidden\", \"optional\" or \"required\"")}
}
