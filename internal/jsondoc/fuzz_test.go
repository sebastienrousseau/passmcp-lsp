// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package jsondoc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// FuzzParse holds the parser to encoding/json: in the strict dialect a
// document parses exactly when the standard library says it is valid (bar
// the depth limit), and every node's range lies inside the text.
func FuzzParse(f *testing.F) {
	for _, s := range []string{`{"a":[1,2,{"b":null}]}`, `"😀"`, `[1,]`, "// c\n{}", `{"a":1e5}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		n, err := Parse(src, Options{})
		var se *SyntaxError
		if errors.As(err, &se) && strings.Contains(se.Msg, "nested more than") {
			return
		}
		if (err == nil) != json.Valid(src) {
			t.Fatalf("Parse ok=%v but json.Valid=%v for %q", err == nil, json.Valid(src), src)
		}
		if n != nil {
			checkRanges(t, n, len(src))
		}
		if c, err := Parse(src, Options{Comments: true}); err == nil {
			checkRanges(t, c, len(src))
		}
	})
}

func checkRanges(t *testing.T, n *Node, size int) {
	t.Helper()
	if n.Start < 0 || n.End > size || n.Start >= n.End {
		t.Fatalf("node range [%d,%d) outside [0,%d)", n.Start, n.End, size)
	}
	for _, m := range n.Members {
		checkRanges(t, m.Value, size)
	}
	for _, it := range n.Items {
		checkRanges(t, it, size)
	}
}
