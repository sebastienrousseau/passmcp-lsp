// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// codePattern finds diagnostic codes as the source writes them.
var codePattern = regexp.MustCompile(`"((?:server-json|client|policy|tool|attestation)/[a-z-]+|syntax|duplicate-key)"`)

// TestEveryCodeIsDocumented fails when a code in the source is missing
// from docs/diagnostics.md, the page the stability guarantee points at.
func TestEveryCodeIsDocumented(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "diagnostics.md"))
	if err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob("*.go")
	codes := map[string]bool{"policy/version": true, "policy/name": true, "tool/input-schema": true, "tool/output-schema": true}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range codePattern.FindAllStringSubmatch(string(src), -1) {
			codes[m[1]] = true
		}
	}
	if len(codes) < 30 {
		t.Fatalf("found only %d codes; the pattern has stopped matching the source", len(codes))
	}
	for c := range codes {
		if !strings.Contains(string(doc), "| `"+c+"` |") {
			t.Errorf("docs/diagnostics.md has no row for %s", c)
		}
	}
}
