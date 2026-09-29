// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package jsondoc

import (
	"sort"
	"unicode/utf8"
)

// Position is a zero-based line and a character offset within it, counted
// in UTF-16 code units: the Language Server Protocol's default encoding,
// and the one every client supports.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Lines converts between byte offsets and positions for one text.
type Lines struct {
	src    []byte
	starts []int
}

// NewLines indexes the line starts of src. A line ends at '\n'; a '\r'
// before it is part of the line's text, which is how the protocol counts.
func NewLines(src []byte) *Lines {
	starts := []int{0}
	for i, c := range src {
		if c == '\n' {
			starts = append(starts, i+1)
		}
	}
	return &Lines{src: src, starts: starts}
}

// Position returns the position of byte offset off, clamped to the text.
func (l *Lines) Position(off int) Position {
	off = max(0, min(off, len(l.src)))
	line := sort.SearchInts(l.starts, off+1) - 1
	return Position{Line: line, Character: utf16Len(l.src[l.starts[line]:off])}
}

// Offset returns the byte offset of p, clamped to the end of its line and
// to the text.
func (l *Lines) Offset(p Position) int {
	if p.Line < 0 {
		return 0
	}
	if p.Line >= len(l.starts) {
		return len(l.src)
	}
	start := l.starts[p.Line]
	end := len(l.src)
	if p.Line+1 < len(l.starts) {
		end = l.starts[p.Line+1] - 1 // the '\n'
	}
	units := 0
	for i := start; i < end; {
		if units >= p.Character {
			return i
		}
		r, size := utf8.DecodeRune(l.src[i:end])
		units += runeUnits(r)
		i += size
	}
	return end
}

func utf16Len(b []byte) int {
	n := 0
	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		n += runeUnits(r)
		b = b[size:]
	}
	return n
}

// runeUnits is the rune's length in UTF-16: two for a supplementary-plane
// rune, which is a surrogate pair.
func runeUnits(r rune) int {
	if r >= 0x10000 {
		return 2
	}
	return 1
}
