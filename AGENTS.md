<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Working on passmcp-lsp as an AI agent

passmcp-lsp is the family's editing surface: a language server that checks
MCP artefacts (server.json, client configurations, tool definitions,
passmcp policies and attestations) and shows passmcp's guidance on hover.
Its value is that its verdicts match the sources that define the formats.
These are the constraints a plausible change breaks for a reason the code
does not state. Read [DEVELOPMENT.md](DEVELOPMENT.md) for the toolchain
and the local form of every CI gate.

## Hard gates

| Gate | Command |
|---|---|
| 85% statement coverage, every package | `make coverage` (ci.yml's Coverage Gate checks each package) |
| Race detector, randomised order | `make test-race` |
| Lint at zero findings, complexity ceilings included | `make lint` |
| SPDX header on every source file | `make spdx-check` |
| The built server answers over stdio | `make smoke` |
| The fuzz targets still run | `make fuzz` |
| Policy rules and hover agree with the released passmcp | `make crosscheck` |
| The embedded schema is the published one | `make schema-check` |
| The VS Code extension tests and packages | `make vscode` |
| `make install` under `DESTDIR` is correct | `make install-smoke` |
| Every version-bearing file agrees | `make versions` |
| The version is passmcp's latest release | `make lockstep` |
| The family manifest's row is true | `make family` |
| The README follows the portfolio template | `make readme-check` |
| No retired product name in the tree | `make name-guard` |

## Commits

- **Every commit must be cryptographically signed and carry a DCO
  `Signed-off-by` trailer, added by the human who certifies it.** An
  agent's shell usually cannot reach the maintainer's ssh-agent; hand the
  commits over as a script rather than producing unsigned history.
- Conventional Commits for the subject line.
- Never rewrite published history.

## Versioning

- **The version is passmcp's latest release, exactly.** It lives in the
  newest `## [x.y.z]` heading in `CHANGELOG.md`, and
  `scripts/verify-release-versions.sh` checks every other place that names
  it: install lines, the README's ecosystem sentence, `CITATION.cff`, the
  extension's `package.json` and lock, and the passmcp-reporting release
  `go.mod` requires.
- **A capability's status names its release**: "Released in X.Y.Z" linking
  that release, or "Not yet released" linking the CHANGELOG entry. Never
  "Shipping", "Available" or "Planned".
- `main.Version` is stamped by the release build; never hard-code it.

## Things that look like bugs and are not

- **The guidance catalogue is not in this repository.** It is passmcp's,
  in an internal package; hover asks the installed passmcp through
  `passmcp explain`. Copying the catalogue here would give it two sources
  ([ADR-0001](docs/adr/0001-ask-passmcp-for-guidance.md)).
- **Attestation rules are not written here.** passmcp-reporting's
  `attestation.Parse` and `a2a.Parse` are the verifier; this code only
  places their problems on the right key.
- **The policy rules are written here, and checked against passmcp.** The
  format is passmcp's; `make crosscheck` applies every fixture with the
  released passmcp. A new rule without a fixture both agree on is a guess.
- **The JSON Schema validator implements a subset, on purpose.** It
  refuses to compile a schema that uses anything else, so a new keyword in
  the registry's schema fails a test rather than being skipped
  ([ADR-0002](docs/adr/0002-own-parser-and-schema-subset.md)).
- **An unrecognised JSON file gets no diagnostics, not even syntax
  errors.** The VS Code extension sends every JSON file; the ones that are
  not MCP artefacts belong to other tools.
- **`passmcp explain` runs with an empty `PASSMCP_CONFIG`.** Do not pass
  the operator's configuration through: a configured model would make a
  hover send data.

## Things that are load-bearing

- **Stdout is the protocol stream in server mode**, and the selected
  format in `check` mode. Diagnostics about the program go to stderr.
- **Documents are touched only by the read loop.** Hover copies what it
  needs before its goroutine starts.
- **Diagnostic codes are the stable interface**; messages are not.

## Scope

- Do not couple a structure or documentation cleanup to a behaviour change.
- Do not add a dependency without saying why in the commit and in
  DEVELOPMENT.md.
- Do not add a CI gate that does not currently pass.
