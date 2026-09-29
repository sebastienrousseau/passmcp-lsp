// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package check

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicyCrosscheck holds this package's policy rules to passmcp's own.
// With $PASSMCP naming a passmcp program (CI's crosscheck job installs the
// release this repository is in lockstep with), every fixture is applied
// by `passmcp verify --policy`: passmcp must refuse exactly the fixtures
// under refuse/, which are exactly the ones this package reports errors in.
func TestPolicyCrosscheck(t *testing.T) {
	bin := os.Getenv("PASSMCP")
	if bin == "" {
		t.Skip("PASSMCP is not set")
	}
	statement := filepath.Join("testdata", "attestation", "mcp-statement.json")
	files, _ := filepath.Glob(filepath.Join("testdata", "policy", "*", "*.json"))
	for _, f := range files {
		refused, why := passmcpRefuses(t, bin, statement, f)
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		ours := errorsIn(Analyze("fixture.policy.json", src).Diagnostics) > 0
		if refused != ours || refused != strings.Contains(f, "refuse") {
			t.Errorf("%s: passmcp refuses=%v (%s), passmcp-lsp reports errors=%v", f, refused, why, ours)
		}
	}
}

// passmcpRefuses applies a policy with passmcp. passmcp exits 1 when it
// cannot read the policy; 0 and 2 are a policy met and not met.
func passmcpRefuses(t *testing.T, bin, statement, policy string) (bool, string) {
	t.Helper()
	var stderr bytes.Buffer
	cmd := exec.CommandContext(context.Background(), bin, "verify", statement, "--policy", policy)
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return false, ""
	case errors.As(err, &exit) && exit.ExitCode() == 2:
		return false, "policy not met"
	case errors.As(err, &exit) && exit.ExitCode() == 1 && strings.Contains(stderr.String(), "policy"):
		return true, strings.TrimSpace(strings.SplitN(stderr.String(), "\n", 2)[0])
	}
	t.Fatalf("passmcp verify --policy %s: %v %s", policy, err, stderr.String())
	return false, ""
}
