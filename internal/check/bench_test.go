// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"os"
	"path/filepath"
	"testing"
)

// The benchmarks behind docs/BENCHMARKS.md: one analysis of each artefact,
// which is what the server does on every keystroke in that document.

func benchFile(b *testing.B, name, path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		Analyze(name, src)
	}
}

func BenchmarkServerJSON(b *testing.B) {
	benchFile(b, "server.json", filepath.Join("testdata", "bench", "server.json"))
}

func BenchmarkPolicy(b *testing.B) {
	benchFile(b, "p.policy.json", filepath.Join("testdata", "policy", "accept", "worked-example.json"))
}

func BenchmarkAttestation(b *testing.B) {
	benchFile(b, "a.json", filepath.Join("testdata", "attestation", "mcp-statement.json"))
}
