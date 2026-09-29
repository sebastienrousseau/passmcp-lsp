// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package guidance

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stub writes a shell script standing in for passmcp.
func stub(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "passmcp")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExecRunner(t *testing.T) {
	echo := stub(t, `test "$1 $2 $3" = "explain --output json" || exit 9; cat`)
	out, err := execRunner(context.Background(), echo, []string{"explain", "--output", "json"}, []byte("in"))
	if err != nil || string(out) != "in" {
		t.Fatalf("stdin must reach the program and stdout come back: %q %v", out, err)
	}
	cfg := stub(t, `cat "$PASSMCP_CONFIG"`)
	if out, err := execRunner(context.Background(), cfg, nil, nil); err != nil || string(out) != "{}\n" {
		t.Fatalf("passmcp runs with an empty configuration: %q %v", out, err)
	}
	fail := stub(t, `echo "first" >&2; echo "passmcp: the last line" >&2; exit 1`)
	if _, err := execRunner(context.Background(), fail, nil, nil); err == nil || !strings.Contains(err.Error(), "the last line") {
		t.Fatalf("a failure carries passmcp's last error line: %v", err)
	}
	big := stub(t, `head -c 2000000 /dev/zero`)
	if _, err := execRunner(context.Background(), big, nil, nil); err == nil || !strings.Contains(err.Error(), "more than") {
		t.Fatalf("output is bounded: %v", err)
	}
	if _, err := execRunner(context.Background(), filepath.Join(t.TempDir(), "absent"), nil, nil); !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("a missing program is ErrNotInstalled: %v", err)
	}
}

// TestRealPassmcp asks the passmcp named by $PASSMCP, when set: CI's
// crosscheck job installs the released passmcp and runs this against it,
// so a change in explain's output format fails here rather than in an
// editor.
func TestRealPassmcp(t *testing.T) {
	bin := os.Getenv("PASSMCP")
	if bin == "" {
		t.Skip("PASSMCP is not set")
	}
	if _, err := exec.LookPath(bin); err != nil {
		t.Fatalf("PASSMCP=%s: %v", bin, err)
	}
	p := &Passmcp{Program: bin}
	g, found, err := p.Lookup(context.Background(), "handshake.protocol_era")
	if err != nil || !found || g.Means == "" || len(g.Steps) == 0 {
		t.Fatalf("passmcp's guidance for handshake.protocol_era: %+v %v %v", g, found, err)
	}
	if _, found, err := p.Lookup(context.Background(), "no_such.check"); err != nil || found {
		t.Fatalf("an unknown id has no guidance: %v %v", found, err)
	}
}
