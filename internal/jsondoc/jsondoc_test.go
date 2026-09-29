// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package jsondoc

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseAgreesWithEncodingJSON(t *testing.T) {
	docs := []string{
		`{}`, `[]`, `{"a":1,"b":[true,false,null],"c":{"d":"e"}}`,
		` "x" `, `-1.5e3`, `0`, `"é😀\n"`,
		`{"a":1,}`, `[1,]`, `{"a" 1}`, `{"a":}`, `[1 2]`, `01`, `tr`, `nul`,
		`"unterminated`, `{"a":1} x`, `{`, `[`, `{1:2}`, `"\x"`, "\"a\tb\"", ``, `/`,
	}
	for _, d := range docs {
		_, err := Parse([]byte(d), Options{})
		if got, want := err == nil, json.Valid([]byte(d)); got != want {
			t.Errorf("Parse(%q) ok=%v, json.Valid=%v (err %v)", d, got, want, err)
		}
	}
}

func TestParseValues(t *testing.T) {
	n, err := Parse([]byte(`{"s":"x","n":2.5,"i":3,"t":true,"f":false,"z":null,"a":[1,"b"],"s":"dup"}`), Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"s": "dup", "n": 2.5, "i": 3.0, "t": true, "f": false, "z": nil, "a": []any{1.0, "b"}}
	if got := n.Value(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Value() = %#v, want %#v", got, want)
	}
	if len(n.Members) != 8 {
		t.Fatalf("duplicates must be kept: %d members", len(n.Members))
	}
}

func TestAccessors(t *testing.T) {
	n, _ := Parse([]byte(`{"s":"x","n":2.5,"i":3,"a":[1],"s":"dup"}`), Options{})
	if n.Get("s").Str != "x" {
		t.Error("Get returns the first of duplicate members")
	}
	if n.Get("missing") != nil || n.Get("a").Get("x") != nil || (*Node)(nil).Member("a") != nil {
		t.Error("Get on a missing key or a non-object must be nil")
	}
	if !n.Get("i").IsInteger() || n.Get("n").IsInteger() || n.Get("s").IsInteger() {
		t.Error("IsInteger")
	}
	if _, ok := n.Get("s").Float(); ok {
		t.Error("Float on a string")
	}
}

func TestKindString(t *testing.T) {
	for k, want := range map[Kind]string{Object: "object", Array: "array", String: "string", Number: "number", Bool: "boolean", Null: "null", Invalid: "invalid", Kind(99): "invalid"} {
		if k.String() != want {
			t.Errorf("%d.String() = %q, want %q", k, k.String(), want)
		}
	}
}

func TestComments(t *testing.T) {
	src := "// lead\n{ /* a */ \"a\": [1, 2,], // trailing\n \"b\": {\"c\": 1,}, }\n/* end */"
	n, err := Parse([]byte(src), Options{Comments: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Get("a").Items) != 2 || n.Get("b").Get("c") == nil {
		t.Fatalf("bad tree: %#v", n.Value())
	}
	if _, err := Parse([]byte(src), Options{}); err == nil || !strings.Contains(err.Error(), "comments are not allowed") {
		t.Fatalf("strict mode must refuse comments, got %v", err)
	}
	for _, bad := range []string{"{} /* open", "{} / x", "{}/", "[,]", "{,}"} {
		if _, err := Parse([]byte(bad), Options{Comments: true}); err == nil {
			t.Errorf("Parse(%q) with comments should fail", bad)
		}
	}
}

func TestDepthLimit(t *testing.T) {
	deep := strings.Repeat("[", MaxDepth+2) + strings.Repeat("]", MaxDepth+2)
	_, err := Parse([]byte(deep), Options{})
	var se *SyntaxError
	if !errors.As(err, &se) || !strings.Contains(se.Msg, "nested more than") {
		t.Fatalf("want a depth error, got %v", err)
	}
	ok := strings.Repeat("[", MaxDepth) + strings.Repeat("]", MaxDepth)
	if _, err := Parse([]byte(ok), Options{}); err != nil {
		t.Fatalf("%d levels must parse: %v", MaxDepth, err)
	}
}

func TestSyntaxErrorOffset(t *testing.T) {
	_, err := Parse([]byte(`{"a": 1 "b": 2}`), Options{})
	var se *SyntaxError
	if !errors.As(err, &se) || se.Offset != 8 {
		t.Fatalf("want an error at offset 8, got %v", err)
	}
	if !strings.HasPrefix(se.Error(), "offset 8: ") {
		t.Errorf("Error() = %q", se.Error())
	}
}

func TestRanges(t *testing.T) {
	src := `{"key": "value", "list": [10, 20]}`
	n, _ := Parse([]byte(src), Options{})
	m := n.Member("key")
	if src[m.KeyStart:m.KeyEnd] != `"key"` || src[m.Value.Start:m.Value.End] != `"value"` {
		t.Fatalf("ranges: key %q value %q", src[m.KeyStart:m.KeyEnd], src[m.Value.Start:m.Value.End])
	}
	if it := n.Get("list").Items[1]; src[it.Start:it.End] != "20" {
		t.Fatalf("item range %q", src[it.Start:it.End])
	}
	if n.Start != 0 || n.End != len(src) {
		t.Fatal("root range")
	}
}

func TestFind(t *testing.T) {
	src := `{"a": {"b": [1, "two"]}, "c": 3}`
	n, _ := Parse([]byte(src), Options{})
	cases := []struct {
		at    string
		path  string
		onKey bool
		kind  Kind
	}{
		{`"two"`, "a.b[1]", false, String},
		{`"b"`, "a.b", true, Array},
		{`3}`, "c", false, Number},
		{`{"b"`, "a", false, Object},
	}
	for _, c := range cases {
		off := strings.Index(src, c.at)
		node, path, onKey := n.Find(off)
		if node == nil || path.String() != c.path || onKey != c.onKey || node.Kind != c.kind {
			t.Errorf("Find(%q) = %v %q key=%v", c.at, node, path, onKey)
		}
	}
	if node, path, _ := n.Find(0); node != n || path.String() != "(document)" {
		t.Error("Find at the root")
	}
	if node, _, _ := n.Find(len(src)); node != nil {
		t.Error("Find past the end must be nil")
	}
}

func TestPath(t *testing.T) {
	p := Path{}.Key("packages").Index(0).Key("transport")
	if p.String() != "packages[0].transport" || p.Last() != "transport" {
		t.Fatalf("got %q last %q", p, p.Last())
	}
	q := p.Key("type")
	if p.String() != "packages[0].transport" || q.String() != "packages[0].transport.type" {
		t.Fatal("Key must not alias the parent path")
	}
	if root, idx := (Path{}), (Path{}).Index(2); root.Last() != "" || idx.String() != "[2]" || idx.Last() != "" {
		t.Fatal("root and index paths")
	}
}

func TestLines(t *testing.T) {
	src := []byte("ab\r\né\U0001F600x\nlast")
	l := NewLines(src)
	x := strings.Index(string(src), "x")
	cases := []struct {
		off int
		pos Position
	}{
		{0, Position{0, 0}}, {2, Position{0, 2}}, {4, Position{1, 0}},
		{x, Position{1, 3}}, {len(src), Position{2, 4}}, {-5, Position{0, 0}}, {999, Position{2, 4}},
	}
	for _, c := range cases {
		if got := l.Position(c.off); got != c.pos {
			t.Errorf("Position(%d) = %v, want %v", c.off, got, c.pos)
		}
	}
	for _, c := range cases[:5] {
		if got := l.Offset(c.pos); got != c.off {
			t.Errorf("Offset(%v) = %d, want %d", c.pos, got, c.off)
		}
	}
	if l.Offset(Position{Line: -1}) != 0 || l.Offset(Position{Line: 9}) != len(src) {
		t.Error("out-of-range lines clamp")
	}
	if got := l.Offset(Position{Line: 0, Character: 99}); got != 3 {
		t.Errorf("a character past the line end clamps to it, got %d", got)
	}
}
