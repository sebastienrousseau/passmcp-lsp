<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0001 — Ask the installed passmcp for guidance; do not copy the catalogue

**Status:** Accepted · **Decided:** 2026-09-29

## Context

Hovering a check id should show what the check means and how to fix it.
That text is passmcp's guidance catalogue: in passmcp 0.0.1 it is the
`remediations` map in `internal/report/remediation.go`, keyed by check id,
in an internal package no other module can import, and it is not published
as data.

Copying it here would give the catalogue two sources that drift apart the
first time either changes, and passmcp is GPL-3.0-only while this
repository is Apache-2.0.

passmcp 0.0.1 does expose the catalogue through a command: `passmcp
explain --output json` reads a report and prints, for each failing or
warning finding, the id and passmcp's guidance for it (`means` and
`steps`). With no `--model` it sends nothing anywhere. This was verified
against the released binary: a report with one failing finding,
`handshake.protocol_era`, returns that check's guidance; an unknown id
returns the finding with no guidance.

## Decision

`internal/guidance` builds a one-finding report for the hovered id, runs
the installed passmcp with `explain --output json`, and reads the entry
back. The id is JSON-encoded into the report on stdin, never placed on the
command line. passmcp runs with `PASSMCP_CONFIG` pointing at an empty file,
so no default in the operator's configuration, such as a model, applies.
Answers are cached for the life of the process. When passmcp is not
installed, the hover says how to install it.

## Consequences

- The guidance a hover shows is exactly what the installed passmcp says,
  including after a passmcp upgrade.
- Hover needs passmcp installed; diagnostics do not.
- A change in `explain`'s output shape breaks hover. `TestRealPassmcp` runs
  against the released passmcp in CI, so it breaks a build rather than an
  editor.
- The first hover on an id costs a process start (see
  [BENCHMARKS.md](../BENCHMARKS.md)).
- If passmcp one day publishes the catalogue as data, or in an importable
  package, this record is superseded by one that reads that instead.
