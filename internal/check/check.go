// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package check recognises MCP artefacts and reports what is wrong with
// them: server.json registry listings, MCP client configurations, MCP tool
// definitions, passmcp acceptance policies and passmcp attestations.
//
// Every diagnostic has a stable code (docs/diagnostics.md lists them all)
// and a byte range in the text, so an editor can underline the cause and a
// CI job can match on the code.
package check

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// Severity follows the Language Server Protocol's numbering.
type Severity int

// The severities, most serious first.
const (
	Error       Severity = 1
	Warning     Severity = 2
	Information Severity = 3
	Hint        Severity = 4
)

// String names the severity in lower case.
func (s Severity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Information:
		return "info"
	case Hint:
		return "hint"
	}
	return "unknown"
}

// Diagnostic is one problem, and the byte range [Start, End) it concerns.
type Diagnostic struct {
	Start, End int
	Severity   Severity
	Code       string
	Message    string
}

// Kind is the artefact a document was recognised as.
type Kind string

// The artefacts this package understands. Unknown documents get only the
// syntax and duplicate-key checks.
const (
	Unknown      Kind = ""
	ServerJSON   Kind = "server.json"
	ClientConfig Kind = "client-config"
	Tools        Kind = "mcp-tools"
	Policy       Kind = "passmcp-policy"
	Attestation  Kind = "passmcp-attestation"
)

// Result is the analysis of one document.
type Result struct {
	Kind Kind
	// Root is the parsed document, nil when it is not JSON.
	Root        *jsondoc.Node
	Diagnostics []Diagnostic
}

type analyzer func(root *jsondoc.Node, src []byte, name string) []Diagnostic

var analyzers = map[Kind]analyzer{
	ServerJSON:   analyzeServerJSON,
	ClientConfig: analyzeClientConfig,
	Tools:        analyzeTools,
	Policy:       analyzePolicy,
	Attestation:  analyzeAttestation,
}

// Analyze recognises and checks one document. name is its file name or
// path, which recognition uses first; the content decides the rest.
//
// A document that is none of the artefacts gets no diagnostics at all,
// not even for its syntax: it is some other JSON file, and some other tool
// owns it.
func Analyze(name string, src []byte) Result {
	byName := classifyName(name)
	root, err := jsondoc.Parse(src, jsondoc.Options{})
	if err != nil {
		return analyzeLoose(byName, name, src, err)
	}
	return analyzeTree(classify(byName, root), root, src, name)
}

// analyzeLoose handles a document that is not strict JSON: a client
// configuration may carry comments and trailing commas, anything else is a
// syntax error.
func analyzeLoose(byName Kind, name string, src []byte, strictErr error) Result {
	loose, err := jsondoc.Parse(src, jsondoc.Options{Comments: true})
	kind := byName
	if err == nil {
		kind = classify(byName, loose)
	}
	switch {
	case kind == Unknown:
		return Result{}
	case err == nil && kind == ClientConfig:
		return analyzeTree(kind, loose, src, name)
	case kind != ClientConfig:
		err = strictErr
	}
	return Result{Kind: kind, Diagnostics: []Diagnostic{syntax(err, len(src))}}
}

func analyzeTree(kind Kind, root *jsondoc.Node, src []byte, name string) Result {
	if kind == Unknown {
		return Result{Root: root}
	}
	ds := append(duplicates(root), analyzers[kind](root, src, name)...)
	// In document order, as a reader and a compiler-style listing expect.
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].Start < ds[j].Start })
	return Result{Kind: kind, Root: root, Diagnostics: ds}
}

func syntax(err error, size int) Diagnostic {
	var se *jsondoc.SyntaxError
	if !errors.As(err, &se) {
		return Diagnostic{Severity: Error, Code: "syntax", Message: err.Error()}
	}
	end := min(se.Offset+1, size)
	return Diagnostic{Start: se.Offset, End: max(end, se.Offset), Severity: Error, Code: "syntax", Message: "not valid JSON: " + se.Msg}
}

// classifyName recognises the files whose name alone says what they are.
func classifyName(name string) Kind {
	name = strings.ReplaceAll(name, "\\", "/")
	base := path.Base(name)
	switch {
	case base == "server.json":
		return ServerJSON
	case base == "claude_desktop_config.json" || base == ".mcp.json":
		return ClientConfig
	case base == "mcp.json" && (strings.HasSuffix(name, ".vscode/mcp.json") || strings.HasSuffix(name, ".cursor/mcp.json")):
		return ClientConfig
	case strings.HasSuffix(base, ".policy.json") || base == "passmcp-policy.json":
		return Policy
	}
	return Unknown
}

// policyKeys are the rule keys only a passmcp policy has.
var policyKeys = []string{"must_pass", "must_not_fail", "max_fail", "max_warn", "min_score", "min_category_score", "forbid_severity", "exemptions"}

// classify recognises a document by its content when its name did not.
func classify(byName Kind, root *jsondoc.Node) Kind {
	if byName != Unknown {
		return byName
	}
	switch {
	case isAttestation(root):
		return Attestation
	case isRegistrySchema(root.Get("$schema")):
		return ServerJSON
	case root.Get("mcpServers") != nil:
		return ClientConfig
	case hasAny(root, policyKeys):
		return Policy
	case isToolsDocument(root):
		return Tools
	}
	return Unknown
}

func isAttestation(root *jsondoc.Node) bool {
	pt := root.Get("predicateType")
	return root.Get("_type") != nil && pt != nil && strings.HasPrefix(pt.Str, predicatePrefix)
}

func isRegistrySchema(n *jsondoc.Node) bool {
	return n != nil && strings.Contains(n.Str, "modelcontextprotocol.io/schemas/") && strings.HasSuffix(n.Str, "/server.schema.json")
}

func hasAny(root *jsondoc.Node, keys []string) bool {
	for _, k := range keys {
		if root.Member(k) != nil {
			return true
		}
	}
	return false
}

// duplicates reports every key that appears twice in one object. JSON
// decoders disagree about which value wins, so the file means different
// things to different readers.
func duplicates(n *jsondoc.Node) []Diagnostic {
	var out []Diagnostic
	seen := map[string]bool{}
	for _, m := range n.Members {
		if seen[m.Key] {
			out = append(out, Diagnostic{Start: m.KeyStart, End: m.KeyEnd, Severity: Warning, Code: "duplicate-key",
				Message: fmt.Sprintf("%q appears more than once in this object; readers disagree about which value wins", m.Key)})
		}
		seen[m.Key] = true
		out = append(out, duplicates(m.Value)...)
	}
	for _, it := range n.Items {
		out = append(out, duplicates(it)...)
	}
	return out
}

// at builds a diagnostic over a node's range.
func at(n *jsondoc.Node, sev Severity, code, format string, a ...any) Diagnostic {
	return Diagnostic{Start: n.Start, End: n.End, Severity: sev, Code: code, Message: fmt.Sprintf(format, a...)}
}

// atKey builds a diagnostic over a member's key.
func atKey(m *jsondoc.Member, sev Severity, code, format string, a ...any) Diagnostic {
	return Diagnostic{Start: m.KeyStart, End: m.KeyEnd, Severity: sev, Code: code, Message: fmt.Sprintf(format, a...)}
}

// head is the first character of a node: where to point when the whole
// node would underline the entire document.
func head(n *jsondoc.Node) *jsondoc.Node {
	return &jsondoc.Node{Start: n.Start, End: n.Start + 1}
}
