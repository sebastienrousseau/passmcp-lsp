// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"net/url"
	"path"
	"regexp"
	"strings"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// The layouts recognised here are the ones passmcp itself reads
// (internal/clientconf and internal/discover at v0.0.1): an "mcpServers"
// object for Claude Desktop, Claude Code and Cursor, and a "servers" object
// in VS Code's .vscode/mcp.json. Each entry starts a program ("command",
// "args", "env") or names a remote server ("url", "headers").

// analyzeClientConfig checks an MCP client configuration.
func analyzeClientConfig(root *jsondoc.Node, _ []byte, name string) []Diagnostic {
	if root.Kind != jsondoc.Object {
		return []Diagnostic{at(root, Error, "client/no-servers", "a client configuration is a JSON object")}
	}
	key := "mcpServers"
	if strings.HasSuffix(strings.ReplaceAll(name, "\\", "/"), ".vscode/mcp.json") {
		key = "servers"
	}
	block := root.Member(key)
	if block == nil {
		return []Diagnostic{missingBlock(root, key, name)}
	}
	if block.Value.Kind != jsondoc.Object {
		return []Diagnostic{at(block.Value, Error, "client/no-servers", "%q must be an object of named servers", key)}
	}
	var out []Diagnostic
	for _, m := range block.Value.Members {
		out = append(out, checkServerEntry(m)...)
	}
	return out
}

// missingBlock explains which key the host reads, naming the other layout
// when the file has it instead.
func missingBlock(root *jsondoc.Node, key, name string) Diagnostic {
	other := "servers"
	if key == "servers" {
		other = "mcpServers"
	}
	if m := root.Member(other); m != nil {
		return atKey(m, Warning, "client/wrong-layout",
			"%s names its servers under %q, not %q", hostOf(name), key, other)
	}
	return at(head(root), Error, "client/no-servers",
		"names no MCP servers: expected %q (Claude Desktop, Claude Code, Cursor) or \"servers\" (VS Code's .vscode/mcp.json)", "mcpServers")
}

func hostOf(name string) string {
	if strings.HasSuffix(strings.ReplaceAll(name, "\\", "/"), ".vscode/mcp.json") {
		return "VS Code's .vscode/mcp.json"
	}
	return path.Base(strings.ReplaceAll(name, "\\", "/"))
}

// checkServerEntry checks one named server.
func checkServerEntry(m *jsondoc.Member) []Diagnostic {
	e := m.Value
	if e.Kind != jsondoc.Object {
		return []Diagnostic{at(e, Error, "client/entry", "server %q must be an object", m.Key)}
	}
	var out []Diagnostic
	cmd, url := e.Get("command"), e.Get("url")
	switch {
	case cmd == nil && url == nil:
		out = append(out, atKey(m, Error, "client/entry", "server %q has neither a command nor a url", m.Key))
	case nonBlank(cmd) && nonBlank(url):
		out = append(out, atKey(m, Warning, "client/command-and-url",
			"server %q has both a command and a url; passmcp reads it as a program and ignores the url", m.Key))
	}
	out = append(out, checkEntryFields(m.Key, e)...)
	return out
}

// checkEntryFields checks the type of each field a host reads.
func checkEntryFields(name string, e *jsondoc.Node) []Diagnostic {
	var out []Diagnostic
	if c := e.Get("command"); c != nil && (c.Kind != jsondoc.String || strings.TrimSpace(c.Str) == "") {
		out = append(out, at(c, Error, "client/entry", "server %q: command must be a non-empty string", name))
	}
	if u := e.Get("url"); u != nil && !isHTTPURL(u) {
		out = append(out, at(u, Error, "client/url", "server %q: url must be an absolute http or https URL", name))
	}
	if a := e.Get("args"); a != nil {
		out = append(out, stringArray(a, "client/entry", "server "+quote(name)+": args")...)
	}
	for _, field := range []string{"env", "headers"} {
		if f := e.Get(field); f != nil {
			out = append(out, stringMap(f, field, name)...)
		}
	}
	return out
}

// nonBlank reports a string node with something in it; a command of spaces
// is no command.
func nonBlank(n *jsondoc.Node) bool {
	return n != nil && n.Kind == jsondoc.String && strings.TrimSpace(n.Str) != ""
}

func quote(s string) string { return `"` + s + `"` }

func isHTTPURL(n *jsondoc.Node) bool {
	if n.Kind != jsondoc.String {
		return false
	}
	u, err := url.Parse(n.Str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// stringArray reports a value that is not an array of strings.
func stringArray(n *jsondoc.Node, code, what string) []Diagnostic {
	if n.Kind != jsondoc.Array {
		return []Diagnostic{at(n, Error, code, "%s must be an array of strings", what)}
	}
	var out []Diagnostic
	for _, it := range n.Items {
		if it.Kind != jsondoc.String {
			out = append(out, at(it, Error, code, "%s must hold only strings, not %s", what, it.Kind))
		}
	}
	return out
}

// secretName matches the names under which credentials usually travel.
var secretName = regexp.MustCompile(`(?i)(authorization|token|secret|password|passwd|api[_-]?key|credential)`)

// variableRef matches a value that names a secret rather than holding it:
// ${input:...}, ${env:...} or ${VAR}.
var variableRef = regexp.MustCompile(`\$\{[^}]+\}`)

// stringMap checks env and headers: an object of string values, and none
// of them a credential written into the file.
func stringMap(n *jsondoc.Node, field, server string) []Diagnostic {
	if n.Kind != jsondoc.Object {
		return []Diagnostic{at(n, Error, "client/entry", "server %q: %s must be an object of strings", server, field)}
	}
	var out []Diagnostic
	for _, m := range n.Members {
		v := m.Value
		switch {
		case v.Kind != jsondoc.String:
			out = append(out, at(v, Error, "client/entry", "server %q: %s.%s must be a string", server, field, m.Key))
		case secretName.MatchString(m.Key) && strings.TrimSpace(v.Str) != "" && !variableRef.MatchString(v.Str):
			out = append(out, at(v, Warning, "client/literal-credential",
				"server %q: %s.%s holds a literal credential; anyone who can read this file has it. Reference it instead, for example ${env:NAME} or ${input:name} where the host supports them", server, field, m.Key))
		}
	}
	return out
}
