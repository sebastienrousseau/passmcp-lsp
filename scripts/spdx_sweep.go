// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// spdx_sweep fails when a source file lacks a machine-readable license
// header, so REUSE compliance is a CI gate rather than a habit.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var skipDirs = map[string]bool{".git": true, "build": true, "dist": true, "vendor": true, "node_modules": true, "site": true, "out": true, "public": true}

func main() {
	missing, err := sweep(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(missing) > 0 {
		// REUSE-IgnoreStart
		fmt.Fprintln(os.Stderr, "files without an SPDX-License-Identifier in their first 5 lines:")
		// REUSE-IgnoreEnd
		for _, m := range missing {
			fmt.Fprintln(os.Stderr, "  "+m)
		}
		os.Exit(1)
	}
	fmt.Println("spdx-check: every source file carries a license header")
}

// sweep walks root and returns every wanted file without a header.
func sweep(root string) ([]string, error) {
	var missing []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && skipDirs[d.Name()] && path != root:
			return filepath.SkipDir
		case !d.IsDir() && wants(path) && !hasHeader(path):
			missing = append(missing, path)
		}
		return nil
	})
	return missing, err
}

// exempt are files whose format takes no header; REUSE.toml covers them.
var exempt = map[string]bool{"go.mod": true, "go.sum": true, "LICENSE": true, "flake.lock": true, "CODEOWNERS": true, "CITATION.cff": true, ".gitignore": true, ".gitattributes": true, ".DS_Store": true}

// checkedDotfiles are the dotfile prefixes that are checked; every other
// dotfile is not.
var checkedDotfiles = []string{".golangci", ".goreleaser", ".pre-commit", ".editorconfig", ".markdownlint", ".gitleaks"}

// checkedExts are the extensions whose files must carry a header.
var checkedExts = map[string]bool{".go": true, ".sh": true, ".yml": true, ".yaml": true, ".md": true, ".toml": true, ".jsonc": true, ".nix": true, ".svg": true, ".ts": true}

// checkedNames are extensionless files that must carry a header.
var checkedNames = map[string]bool{"Makefile": true, "GNUmakefile": true, "Dockerfile": true}

func wants(path string) bool {
	base := filepath.Base(path)
	if exempt[base] {
		return false
	}
	if strings.HasPrefix(base, ".") && !hasAnyPrefix(base, checkedDotfiles) {
		return false
	}
	return checkedExts[filepath.Ext(path)] || checkedNames[base] || strings.HasPrefix(base, ".editorconfig")
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func hasHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for i := 0; i < 5 && sc.Scan(); i++ {
		// REUSE-IgnoreStart
		if strings.Contains(sc.Text(), "SPDX-License-Identifier:") {
			return true
		}
		// REUSE-IgnoreEnd
	}
	return false
}
