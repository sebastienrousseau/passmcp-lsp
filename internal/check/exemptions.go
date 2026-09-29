// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"strings"
	"time"

	"satellion.com/passmcp-lsp/internal/jsondoc"
)

var exemptionKeys = map[string]bool{"check": true, "reason": true, "expires": true, "ticket": true}

// exemptions checks each exemption: a check, a reason and an expiry date,
// each required, because an exception with no reason or no date is a
// permanent, undocumented hole.
func exemptions(v *jsondoc.Node, key string) []Diagnostic {
	if v.Kind != jsondoc.Array {
		return []Diagnostic{at(v, Error, "policy/type", "%s must be an array", key)}
	}
	var out []Diagnostic
	seen := map[string]bool{}
	for _, e := range v.Items {
		out = append(out, exemption(e, seen)...)
	}
	return out
}

func exemption(e *jsondoc.Node, seen map[string]bool) []Diagnostic {
	if e.Kind != jsondoc.Object {
		return []Diagnostic{at(e, Error, "policy/type", "an exemption must be an object")}
	}
	var out []Diagnostic
	for _, m := range e.Members {
		switch {
		case !exemptionKeys[m.Key]:
			out = append(out, unknownKey(m, "an exemption"))
		case m.Value.Kind != jsondoc.Null:
			out = append(out, wantString(m.Value, "exemption "+m.Key)...)
		}
	}
	id := strings.TrimSpace(strOrEmpty(e.Get("check")))
	switch {
	case id == "":
		return append(out, at(head(e), Error, "policy/exemption", "an exemption names no check"))
	case seen[id]:
		out = append(out, at(e.Get("check"), Error, "policy/exemption", "%s is exempted twice, so which reason applies is undecided", id))
	}
	seen[id] = true
	if strings.TrimSpace(strOrEmpty(e.Get("reason"))) == "" {
		out = append(out, at(head(e), Error, "policy/exemption",
			"the exemption for %s has no reason, which makes it an undocumented hole rather than a decision", id))
	}
	return append(out, expiry(e, id)...)
}

// expiry checks the date. The expiry day itself is still covered: "expires
// 2027-03-31" reads as good through the 31st.
func expiry(e *jsondoc.Node, id string) []Diagnostic {
	x := e.Get("expires")
	if x == nil || strings.TrimSpace(x.Str) == "" {
		return []Diagnostic{at(head(e), Error, "policy/exemption",
			"the exemption for %s has no expiry: an exception with no date on it is a permanent hole", id)}
	}
	d, err := time.Parse("2006-01-02", x.Str)
	if err != nil {
		return []Diagnostic{at(x, Error, "policy/exemption", "%q is not a date in YYYY-MM-DD form", x.Str)}
	}
	if now().UTC().Truncate(24 * time.Hour).After(d) {
		return []Diagnostic{at(x, Warning, "policy/expired",
			"the exemption for %s expired on %s; passmcp now counts it as a failed rule", id, x.Str)}
	}
	return nil
}

// crossRules are the rules that span fields: a check both required and
// exempted, and a policy that states no rule at all.
func crossRules(root *jsondoc.Node) []Diagnostic {
	exempt := map[string]bool{}
	if ex := root.Get("exemptions"); ex != nil {
		for _, e := range ex.Items {
			exempt[strings.TrimSpace(strOrEmpty(e.Get("check")))] = true
		}
	}
	var out []Diagnostic
	for _, list := range []string{"must_pass", "must_not_fail"} {
		for _, it := range itemsOf(root.Get(list)) {
			if id := strings.TrimSpace(it.Str); it.Kind == jsondoc.String && id != "" && exempt[id] {
				out = append(out, at(it, Error, "policy/check-id", "%s is both required and exempted; remove one", id))
			}
		}
	}
	if !statesRule(root) {
		out = append(out, at(head(root), Information, "policy/no-rules",
			"the policy states no rule, so nothing will be judged"))
	}
	return out
}

// statesRule reports whether the policy states any rule, counted the way
// passmcp counts them: a non-empty check list or category map, or any other
// rule that is set. Exemptions are not rules.
func statesRule(root *jsondoc.Node) bool {
	for _, k := range []string{"must_pass", "must_not_fail"} {
		if len(itemsOf(root.Get(k))) > 0 {
			return true
		}
	}
	if c := root.Get("min_category_score"); c != nil && len(c.Members) > 0 {
		return true
	}
	for _, k := range []string{"max_fail", "max_warn", "min_score", "target"} {
		if v := root.Get(k); v != nil && v.Kind != jsondoc.Null {
			return true
		}
	}
	return strOrEmpty(root.Get("forbid_severity")) != ""
}

func itemsOf(n *jsondoc.Node) []*jsondoc.Node {
	if n == nil {
		return nil
	}
	return n.Items
}
