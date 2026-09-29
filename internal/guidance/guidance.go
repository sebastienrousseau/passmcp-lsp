// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Package guidance fetches passmcp's remediation guidance for a check id
// from the passmcp program itself.
//
// The guidance catalogue lives in passmcp, in an internal package, and is
// not published as data. Copying it here would give it two sources that
// drift apart, so this package asks the installed passmcp instead: `passmcp
// explain --output json` reads a report on stdin and prints, for each
// failing finding, passmcp's own guidance for its check. A one-finding
// report naming the check id gets exactly that entry. Without --model,
// explain sends nothing anywhere; it only reads the report and prints.
//
// When passmcp is not installed the hover says so and nothing else fails.
package guidance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Step is one action in the guidance.
type Step struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// Guidance is passmcp's remediation for one check.
type Guidance struct {
	ID    string `json:"id"`
	Means string `json:"means,omitempty"`
	Steps []Step `json:"steps,omitempty"`
}

// ErrNotInstalled is returned when the passmcp program cannot be found.
var ErrNotInstalled = errors.New("passmcp is not installed")

// Runner runs passmcp with arguments and stdin and returns its stdout.
type Runner func(ctx context.Context, program string, args []string, stdin []byte) ([]byte, error)

// Passmcp looks guidance up through the passmcp program, caching each
// answer for the life of the process: the catalogue belongs to the
// installed passmcp and does not change under it.
type Passmcp struct {
	// Program is the passmcp executable, a path or a name on PATH.
	Program string
	// Timeout bounds one lookup.
	Timeout time.Duration
	// Run executes the program; nil means os/exec.
	Run Runner

	mu    sync.Mutex
	cache map[string]lookup
}

type lookup struct {
	g     Guidance
	found bool
}

// maxOutput bounds what is read from passmcp: one entry of guidance is a
// few kilobytes.
const maxOutput = 1 << 20

// Lookup returns passmcp's guidance for id. found is false when passmcp has
// no guidance for it. An error means passmcp could not be asked.
func (p *Passmcp) Lookup(ctx context.Context, id string) (Guidance, bool, error) {
	p.mu.Lock()
	if l, ok := p.cache[id]; ok {
		p.mu.Unlock()
		return l.g, l.found, nil
	}
	p.mu.Unlock()

	g, found, err := p.ask(ctx, id)
	if err != nil {
		return Guidance{}, false, err
	}
	p.mu.Lock()
	if p.cache == nil {
		p.cache = map[string]lookup{}
	}
	p.cache[id] = lookup{g, found}
	p.mu.Unlock()
	return g, found, nil
}

func (p *Passmcp) ask(ctx context.Context, id string) (Guidance, bool, error) {
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	run := p.Run
	if run == nil {
		run = execRunner
	}
	out, err := run(ctx, p.program(), []string{"explain", "--output", "json"}, report(id))
	if err != nil {
		return Guidance{}, false, err
	}
	return parse(out, id)
}

func (p *Passmcp) program() string {
	if strings.TrimSpace(p.Program) == "" {
		return "passmcp"
	}
	return p.Program
}

// report is the smallest report explain accepts: one phase with one
// failing finding whose id is the check asked about. The id is encoded by
// encoding/json, so nothing in it can change the document's shape.
func report(id string) []byte {
	type finding struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	type phase struct {
		Name     string    `json:"name"`
		Findings []finding `json:"findings"`
	}
	b, _ := json.Marshal(struct {
		Phases []phase `json:"phases"`
	}{[]phase{{Name: phaseOf(id), Findings: []finding{{ID: id, Title: id, Status: "fail"}}}}})
	return b
}

func phaseOf(id string) string {
	before, _, _ := strings.Cut(id, ".")
	return before
}

// parse reads explain's JSON document and returns the entry for id.
func parse(out []byte, id string) (Guidance, bool, error) {
	var doc struct {
		Findings []Guidance `json:"findings"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		return Guidance{}, false, fmt.Errorf("passmcp explain printed something other than its JSON document: %w", err)
	}
	for _, f := range doc.Findings {
		if f.ID == id {
			return f, f.Means != "" || len(f.Steps) > 0, nil
		}
	}
	return Guidance{}, false, nil
}

// execRunner runs the program with no shell, stdin as given, a bound on
// what it may print, and PASSMCP_CONFIG pointing at an empty configuration
// file: a default in the operator's own passmcp configuration, such as a
// model for explain to send findings to, must not apply to a lookup an
// editor made on hover.
func execRunner(ctx context.Context, program string, args []string, stdin []byte) ([]byte, error) {
	path, err := exec.LookPath(program)
	if err != nil {
		return nil, ErrNotInstalled
	}
	dir, err := os.MkdirTemp("", "passmcp-lsp-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	cfg := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfg, []byte("{}\n"), 0o600); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, args...) // #nosec G204 -- the operator's own passmcp, fixed arguments
	cmd.Env = append(os.Environ(), "PASSMCP_CONFIG="+cfg)
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout limited
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("passmcp explain: %w: %s", err, lastLine(stderr.String()))
	}
	if stdout.over {
		return nil, fmt.Errorf("passmcp explain printed more than %d bytes", maxOutput)
	}
	return stdout.Bytes(), nil
}

// limited is a buffer that stops growing at maxOutput. The buffer is a
// field, not embedded, so that io.Copy cannot reach bytes.Buffer's
// ReadFrom and go around the limit.
type limited struct {
	buf  bytes.Buffer
	over bool
}

func (l *limited) Write(p []byte) (int, error) {
	if l.buf.Len()+len(p) > maxOutput {
		l.over = true
		return len(p), nil
	}
	return l.buf.Write(p)
}

// Bytes returns what was kept.
func (l *limited) Bytes() []byte { return l.buf.Bytes() }

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}
