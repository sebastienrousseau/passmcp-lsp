// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Command passmcp-lsp is a language server for MCP artefacts: server.json,
// MCP client configurations, MCP tool definitions, passmcp policies and
// passmcp attestations.
//
//	passmcp-lsp                      serve the Language Server Protocol over stdio
//	passmcp-lsp check [--output json] FILE...
//	                                 check files and print what is wrong
//	passmcp-lsp --version
//	passmcp-lsp --completion bash|zsh|fish
//
// The selected output goes to stdout; diagnostics about the program itself
// go to stderr. In server mode stdout is the protocol stream.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"satellion.com/passmcp-lsp/internal/guidance"
	"satellion.com/passmcp-lsp/internal/lsp"
)

// Version is stamped by the release build with -ldflags "-X main.Version=...".
var Version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

// flags is the command line of server mode.
type flags struct {
	version    bool
	stdio      bool
	passmcp    string
	completion string
}

func newFlagSet(stderr io.Writer, f *flags) *flag.FlagSet {
	fs := flag.NewFlagSet("passmcp-lsp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.BoolVar(&f.version, "version", false, "print the version and exit")
	fs.BoolVar(&f.stdio, "stdio", false, "serve over stdin and stdout (the default, and the only transport)")
	fs.StringVar(&f.passmcp, "passmcp", envOr("PASSMCP_LSP_PASSMCP", "passmcp"), "the passmcp program hover asks for guidance (env PASSMCP_LSP_PASSMCP)")
	fs.StringVar(&f.completion, "completion", "", "print a completion script for bash, zsh or fish, and exit")
	fs.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "usage: passmcp-lsp [flags]\n       passmcp-lsp check [--output text|json] FILE...\n\n")
		fs.PrintDefaults()
	}
	return fs
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// run is the program without the process around it.
func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "check" {
		return runCheck(args[1:], stdout, stderr)
	}
	var f flags
	fs := newFlagSet(stderr, &f)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	switch {
	case fs.NArg() > 0:
		_, _ = fmt.Fprintf(stderr, "passmcp-lsp: unexpected argument %q\n", fs.Arg(0))
		return 2
	case f.version:
		_, _ = fmt.Fprintln(stdout, "passmcp-lsp "+Version)
		return 0
	case f.completion != "":
		return completion(fs, f.completion, stdout, stderr)
	}
	return serve(ctx, f.passmcp, stdin, stdout, stderr)
}

func serve(ctx context.Context, program string, stdin io.Reader, stdout, stderr io.Writer) int {
	opt := lsp.Options{
		Name:    "passmcp-lsp",
		Version: Version,
		Log:     stderr,
		Guidance: func(fromClient string) lsp.Looker {
			if fromClient != "" {
				program = fromClient
			}
			return &guidance.Passmcp{Program: program, Timeout: 10 * time.Second}
		},
	}
	err := lsp.Serve(ctx, stdin, stdout, opt)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "passmcp-lsp:", err)
		return 1
	}
	return 0
}
