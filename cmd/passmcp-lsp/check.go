// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"satellion.com/passmcp-lsp/internal/check"
	"satellion.com/passmcp-lsp/internal/jsondoc"
)

// fileReport is one file's result in --output json.
type fileReport struct {
	File        string         `json:"file"`
	Kind        string         `json:"kind"`
	Diagnostics []diagnosticJS `json:"diagnostics"`
}

type diagnosticJS struct {
	Line      int    `json:"line"`
	Character int    `json:"character"`
	EndLine   int    `json:"endLine"`
	EndChar   int    `json:"endCharacter"`
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

// runCheck checks files the way the server checks open documents. It
// exits 1 when any file has an error, 2 when it could not run, and 0
// otherwise; warnings and information do not fail it.
func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("passmcp-lsp check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	output := fs.String("output", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *output != "text" && *output != "json" {
		_, _ = fmt.Fprintf(stderr, "passmcp-lsp check: --output %q: text or json\n", *output)
		return 2
	}
	if fs.NArg() == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: passmcp-lsp check [--output text|json] FILE...")
		return 2
	}
	reports, failed, err := checkFiles(fs.Args())
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "passmcp-lsp check:", err)
		return 2
	}
	if *output == "json" {
		err = writeJSON(stdout, reports)
	} else {
		err = writeText(stdout, reports)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "passmcp-lsp check:", err)
		return 2
	}
	if failed {
		return 1
	}
	return 0
}

func checkFiles(files []string) ([]fileReport, bool, error) {
	var reports []fileReport
	failed := false
	for _, f := range files {
		src, err := os.ReadFile(f) // #nosec G304 -- the operator named the file
		if err != nil {
			return nil, false, err
		}
		r := check.Analyze(f, src)
		lines := jsondoc.NewLines(src)
		rep := fileReport{File: f, Kind: kindName(r.Kind), Diagnostics: []diagnosticJS{}}
		for _, d := range r.Diagnostics {
			s, e := lines.Position(d.Start), lines.Position(d.End)
			rep.Diagnostics = append(rep.Diagnostics, diagnosticJS{
				Line: s.Line + 1, Character: s.Character + 1, EndLine: e.Line + 1, EndChar: e.Character + 1,
				Severity: d.Severity.String(), Code: d.Code, Message: d.Message,
			})
			failed = failed || d.Severity == check.Error
		}
		reports = append(reports, rep)
	}
	return reports, failed, nil
}

func kindName(k check.Kind) string {
	if k == check.Unknown {
		return "unrecognised"
	}
	return string(k)
}

func writeJSON(w io.Writer, reports []fileReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(reports)
}

// writeText prints one line per diagnostic, file:line:column, the way
// compilers do, so editors and CI annotators pick it up; a clean file
// gets a line saying what it was recognised as.
func writeText(w io.Writer, reports []fileReport) error {
	for _, r := range reports {
		if len(r.Diagnostics) == 0 {
			if _, err := fmt.Fprintf(w, "%s: ok (%s)\n", r.File, r.Kind); err != nil {
				return err
			}
			continue
		}
		for _, d := range r.Diagnostics {
			if _, err := fmt.Fprintf(w, "%s:%d:%d: %s: %s [%s]\n", r.File, d.Line, d.Character, d.Severity, d.Message, d.Code); err != nil {
				return err
			}
		}
	}
	return nil
}
