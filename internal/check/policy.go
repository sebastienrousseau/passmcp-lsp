// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"strings"
	"time"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// The rules below are the policy format passmcp v0.0.1 reads, as its
// manual documents it (https://satellion.com/passmcp/docs/policy/): a
// document passmcp would refuse is an error here, and the reason is the
// one passmcp gives. passmcp remains the authority; `make crosscheck` runs
// the released passmcp over this package's fixtures and fails when the two
// disagree about which policies are accepted.

// PolicyFormat is the policy format version passmcp v0.0.1 implements.
const PolicyFormat = 1

// now is the clock that decides whether an exemption has expired.
var now = time.Now

type policyField func(v *jsondoc.Node, key string) []Diagnostic

var policyFields = map[string]policyField{
	"version":            policyVersion,
	"name":               policyName,
	"description":        wantString,
	"target":             policyTarget,
	"must_pass":          checkIDList,
	"must_not_fail":      checkIDList,
	"max_fail":           allowance,
	"max_warn":           allowance,
	"min_score":          score,
	"min_category_score": categoryScores,
	"forbid_severity":    severity,
	"exemptions":         exemptions,
}

// analyzePolicy checks a passmcp acceptance policy.
func analyzePolicy(root *jsondoc.Node, _ []byte, _ string) []Diagnostic {
	if root.Kind != jsondoc.Object {
		return []Diagnostic{at(root, Error, "policy/type", "a policy is a JSON object")}
	}
	var out []Diagnostic
	for _, m := range root.Members {
		f, ok := policyFields[m.Key]
		switch {
		case !ok:
			out = append(out, unknownKey(m, "a policy"))
		case m.Value.Kind != jsondoc.Null:
			// null decodes to the zero value, which passmcp reads as absent.
			out = append(out, f(m.Value, m.Key)...)
		}
	}
	for _, req := range []string{"version", "name"} {
		if v := root.Get(req); v == nil || v.Kind == jsondoc.Null {
			out = append(out, at(head(root), Error, "policy/"+req, "the policy has no %s: %s", req, missingWhy[req]))
		}
	}
	out = append(out, crossRules(root)...)
	return out
}

var missingWhy = map[string]string{
	"version": "a policy with no version cannot be read safely by a later passmcp",
	"name":    "a failure has to be able to say which policy said no",
}

func unknownKey(m *jsondoc.Member, in string) Diagnostic {
	return atKey(m, Error, "policy/unknown-key",
		"%q is not a key of %s; passmcp refuses a policy with a field it does not know, because a misspelled rule would otherwise silently not apply", m.Key, in)
}

// isInt reports a number passmcp can decode into a Go int: no fraction and
// no exponent in the literal, as encoding/json requires.
func isInt(v *jsondoc.Node) bool {
	return v.Kind == jsondoc.Number && !strings.ContainsAny(v.Raw, ".eE")
}

func policyVersion(v *jsondoc.Node, _ string) []Diagnostic {
	f, _ := v.Float()
	switch {
	case !isInt(v):
		return []Diagnostic{at(v, Error, "policy/version", "version must be a whole number")}
	case f <= 0:
		return []Diagnostic{at(v, Error, "policy/version", "version %s is not a version", v.Raw)}
	case f > PolicyFormat:
		return []Diagnostic{at(v, Error, "policy/version",
			"version %s: passmcp 0.0.1 implements policy format %d and refuses a later one rather than apply it partly", v.Raw, PolicyFormat)}
	}
	return nil
}

func policyName(v *jsondoc.Node, key string) []Diagnostic {
	if d := wantString(v, key); d != nil {
		return d
	}
	if strings.TrimSpace(v.Str) == "" {
		return []Diagnostic{at(v, Error, "policy/name", "the name is empty: %s", missingWhy["name"])}
	}
	return nil
}

func wantString(v *jsondoc.Node, key string) []Diagnostic {
	if v.Kind != jsondoc.String {
		return []Diagnostic{at(v, Error, "policy/type", "%s must be a string", key)}
	}
	return nil
}

func policyTarget(v *jsondoc.Node, _ string) []Diagnostic {
	if v.Kind != jsondoc.Object {
		return []Diagnostic{at(v, Error, "policy/type", "target must be an object")}
	}
	out := targetFields(v)
	tr, ep := v.Get("transport"), v.Get("endpoint")
	if tr != nil && tr.Kind == jsondoc.String && tr.Str != "" && tr.Str != "http" && tr.Str != "stdio" {
		out = append(out, at(tr, Error, "policy/target", "transport %q is neither http nor stdio", tr.Str))
	}
	if strOrEmpty(tr) == "" && strings.TrimSpace(strOrEmpty(ep)) == "" {
		out = append(out, at(v, Error, "policy/target", "the target block constrains nothing; remove it or fill it in"))
	}
	return out
}

// targetFields checks the target block's keys and their types.
func targetFields(v *jsondoc.Node) []Diagnostic {
	var out []Diagnostic
	for _, m := range v.Members {
		switch {
		case m.Key != "transport" && m.Key != "endpoint":
			out = append(out, unknownKey(m, "target"))
		case m.Value.Kind != jsondoc.Null:
			out = append(out, wantString(m.Value, "target."+m.Key)...)
		}
	}
	return out
}

func strOrEmpty(n *jsondoc.Node) string {
	if n == nil {
		return ""
	}
	return n.Str
}

// checkIDList checks must_pass and must_not_fail.
func checkIDList(v *jsondoc.Node, key string) []Diagnostic {
	if v.Kind != jsondoc.Array {
		return []Diagnostic{at(v, Error, "policy/type", "%s must be an array of check ids", key)}
	}
	var out []Diagnostic
	seen := map[string]bool{}
	for _, it := range v.Items {
		id := strings.TrimSpace(it.Str)
		switch {
		case it.Kind != jsondoc.String:
			out = append(out, at(it, Error, "policy/type", "%s must hold only strings", key))
		case id == "":
			out = append(out, at(it, Error, "policy/check-id", "%s has an empty check id", key))
		case seen[id]:
			out = append(out, at(it, Error, "policy/check-id", "%s names %s twice", key, id))
		}
		seen[id] = true
	}
	return out
}

func allowance(v *jsondoc.Node, key string) []Diagnostic {
	f, _ := v.Float()
	switch {
	case !isInt(v):
		return []Diagnostic{at(v, Error, "policy/type", "%s must be a whole number", key)}
	case f < 0:
		return []Diagnostic{at(v, Error, "policy/limit", "%s is %s; a negative allowance cannot be met by any run", key, v.Raw)}
	}
	return nil
}

func score(v *jsondoc.Node, key string) []Diagnostic {
	f, ok := v.Float()
	switch {
	case !ok:
		return []Diagnostic{at(v, Error, "policy/type", "%s must be a number", key)}
	case f < 0 || f > 100:
		return []Diagnostic{at(v, Error, "policy/limit", "%s is %s, and a score is 0 to 100", key, v.Raw)}
	}
	return nil
}

func categoryScores(v *jsondoc.Node, key string) []Diagnostic {
	if v.Kind != jsondoc.Object {
		return []Diagnostic{at(v, Error, "policy/type", "%s must be an object of category scores", key)}
	}
	var out []Diagnostic
	for _, m := range v.Members {
		if strings.TrimSpace(m.Key) == "" {
			out = append(out, atKey(m, Error, "policy/limit", "a min_category_score entry has no category name"))
		}
		out = append(out, score(m.Value, key+" for "+m.Key)...)
	}
	return out
}

func severity(v *jsondoc.Node, key string) []Diagnostic {
	if d := wantString(v, key); d != nil {
		return d
	}
	switch v.Str {
	case "critical", "major", "minor", "":
		return nil
	}
	return []Diagnostic{at(v, Error, "policy/severity", "forbid_severity %q is not critical, major or minor", v.Str)}
}
