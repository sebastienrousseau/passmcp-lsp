// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package jsondoc

import (
	"strconv"
	"strings"
)

// Step is one move from a container into a child: a member's key, or an
// array index when Key is empty and Index is set.
type Step struct {
	Key   string
	Index int
	// IsIndex distinguishes index 0 from "no index".
	IsIndex bool
}

// Path is the route from the root to a value.
type Path []Step

// String renders the path the way the diagnostics name a location:
// packages[0].transport.type. The root is "(document)".
func (p Path) String() string {
	if len(p) == 0 {
		return "(document)"
	}
	var b strings.Builder
	for i, s := range p {
		switch {
		case s.IsIndex:
			b.WriteString("[" + strconv.Itoa(s.Index) + "]")
		case i == 0:
			b.WriteString(s.Key)
		default:
			b.WriteString("." + s.Key)
		}
	}
	return b.String()
}

// Key returns a copy of p extended by a member key.
func (p Path) Key(k string) Path { return append(p[:len(p):len(p)], Step{Key: k}) }

// Index returns a copy of p extended by an array index.
func (p Path) Index(i int) Path {
	return append(p[:len(p):len(p)], Step{Index: i, IsIndex: true})
}

// Last returns the final step's key, or "" for the root or an index.
func (p Path) Last() string {
	if len(p) == 0 {
		return ""
	}
	return p[len(p)-1].Key
}

// Find returns the innermost value whose range contains off, and the path
// to it. OnKey is true when off falls on a member's key rather than its
// value; the node is then that member's value.
func (n *Node) Find(off int) (node *Node, path Path, onKey bool) {
	if n == nil || off < n.Start || off >= n.End {
		return nil, nil, false
	}
	cur := n
	for {
		next, step, key, ok := cur.child(off)
		if !ok {
			return cur, path, false
		}
		path = append(path, step)
		if key {
			return next, path, true
		}
		cur = next
	}
}

// child returns the direct child containing off, if any.
func (n *Node) child(off int) (*Node, Step, bool, bool) {
	for _, m := range n.Members {
		if off >= m.KeyStart && off < m.KeyEnd {
			return m.Value, Step{Key: m.Key}, true, true
		}
		if off >= m.Value.Start && off < m.Value.End {
			return m.Value, Step{Key: m.Key}, false, true
		}
	}
	for i, it := range n.Items {
		if off >= it.Start && off < it.End {
			return it, Step{Index: i, IsIndex: true}, false, true
		}
	}
	return nil, Step{}, false, false
}
