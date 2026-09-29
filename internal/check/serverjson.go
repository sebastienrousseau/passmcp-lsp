// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	_ "embed"
	"sync"

	"satellion.com/passmcp-lsp/internal/jsondoc"
	"satellion.com/passmcp-lsp/internal/schema"
)

// ServerSchemaURL is where the embedded server.json schema is published.
// The file under schemas/ is that document, byte for byte; `make
// schema-check` fetches the URL and fails when the two differ.
const ServerSchemaURL = "https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json"

// ServerSchemaVersion is the schema revision, as its URL names it.
const ServerSchemaVersion = "2025-12-11"

//go:embed schemas/server.schema.json
var serverSchemaJSON []byte

// ServerSchema returns the embedded schema's text.
func ServerSchema() []byte { return serverSchemaJSON }

var serverSchema = sync.OnceValues(func() (*schema.Schema, error) {
	return schema.Compile(serverSchemaJSON)
})

// analyzeServerJSON validates a registry listing against the schema the
// MCP Registry publishes.
func analyzeServerJSON(root *jsondoc.Node, _ []byte, _ string) []Diagnostic {
	var out []Diagnostic
	switch s := root.Get("$schema"); {
	case s == nil:
		out = append(out, at(head(root), Information, "server-json/schema-version",
			"no $schema: checked against the %s registry schema", ServerSchemaVersion))
	case s.Str != ServerSchemaURL:
		out = append(out, at(s, Information, "server-json/schema-version",
			"this listing names a schema other than %s; passmcp-lsp checks against the %s revision", ServerSchemaURL, ServerSchemaVersion))
	}
	sch, err := serverSchema()
	if err != nil {
		// The embedded schema is fixed at build time and a test compiles
		// it, so this is unreachable in a released binary.
		return append(out, at(head(root), Error, "server-json/schema", "the embedded schema does not compile: %v", err))
	}
	for _, v := range sch.Validate(root) {
		out = append(out, Diagnostic{Start: v.Start, End: v.End, Severity: Error, Code: "server-json/schema", Message: v.Path + ": " + v.Msg})
	}
	return out
}
