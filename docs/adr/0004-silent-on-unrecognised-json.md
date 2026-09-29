<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0004 — Unrecognised JSON gets no diagnostics, and editors send every JSON file

**Status:** Accepted · **Decided:** 2026-09-29

## Context

Policies and attestations follow no file-naming convention: passmcp's own
examples call them `company.json` and `attestation.json`. They can only be
recognised by content, so the server has to see every JSON file an editor
opens. Most of those files are nothing to do with MCP: `package.json`,
`tsconfig.json`, fixtures.

## Decision

The VS Code extension, and the recommended Neovim configuration, start the
server for every JSON and JSON-with-comments document. The server
recognises an artefact by name first and content second, and a document
that is none of them gets no diagnostics at all — not even a syntax error
or a duplicate key — because another tool owns it.

## Consequences

- A policy or attestation is checked whatever it is called.
- No false positives on unrelated JSON; the server's cost on such a file is
  one parse.
- A policy that holds only `version` and `name`, under a name that does not
  end in `.policy.json`, is not recognised. The README states this.
