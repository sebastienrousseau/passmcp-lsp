<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Development

The single entry point for working on passmcp-lsp: toolchain, how to
reproduce every CI gate locally, and how a release is cut. If a gate fails
in CI and you cannot reproduce it from this file, that is a bug in this
file.

## Requirements

| Tool | Version | Why |
|---|---|---|
| Go | 1.26.8 or later, the `go` directive in `go.mod` | `GOTOOLCHAIN=auto` downloads it; CI tests on that version and on latest stable |
| make | any | Task runner for everything below |
| passmcp | the version in `CHANGELOG.md` | hover, and `make crosscheck`; `go install satellion.com/passmcp/cmd/passmcp@v0.0.2` |
| Node | 22 or later | only for `make vscode`, the VS Code extension |

Optional, only for the gate that uses it: `golangci-lint` (`make lint`),
`markdownlint-cli2`, `codespell` and `lychee` (the Docs Lint workflow and
`pre-commit`), `zsh` and `fish` (`make completions` syntax-checks their
scripts when present), `curl` and `python3` (`make family`, `make lockstep`,
`make schema-check`, `make versions`), MkDocs from `docs/requirements.txt`
(`make docs`), `goreleaser` (`goreleaser check`).

## Dependencies

One direct Go dependency, **satellion.com/passmcp-reporting**, for its
attestation verifiers: the family's single source of what a valid
attestation is ([ADR-0003](docs/adr/0003-attestations-verified-by-passmcp-reporting.md)).
It has no dependencies of its own.

Everything else is the standard library, deliberately. The LSP layer is
the three messages a server needs, framed by one header; a general LSP
library would bring a dependency tree for a surface this small. The JSON
parser keeps positions, which encoding/json does not, and the JSON Schema
validator implements exactly what the embedded registry schema uses and
refuses anything more ([ADR-0002](docs/adr/0002-own-parser-and-schema-subset.md)).

The VS Code extension depends on `vscode-languageclient`, Microsoft's
client for the protocol, which is how every VS Code language extension
talks to its server; its development dependencies are TypeScript, the
`vscode` and `node` type definitions, and `@vscode/vsce`, which packages
the `.vsix`. All are pinned exactly in `editors/vscode/package.json` and
locked in `package-lock.json`.

## Reproducing every CI gate

| CI job | Local command |
|---|---|
| Test (three OSes × two Go versions) | `make test` |
| Race & Shuffled Tests | `make test-race` |
| Coverage Gate (85% per package) | `make coverage` |
| Lint, with the complexity ceilings | `gofmt -l .` and `make lint` |
| Fuzz | `make fuzz FUZZTIME=30s` |
| Vulnerability Scan | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` |
| Crosscheck against passmcp | `make crosscheck` with passmcp at the lockstep version on `PATH` |
| Repository Checks | `make smoke completions install-smoke versions schema-check family lockstep` |
| VS Code extension | `make vscode` |
| Licence Headers | `make spdx-check` and `make name-guard` |
| Markdown & Spelling | `make readme-check`, `markdownlint-cli2 '**/*.md'` and `codespell` |
| Link Check | `lychee --offline --include-fragments '**/*.md'` |
| DCO check | `git log --format=%B origin/main.. \| grep Signed-off-by` |
| Manual (build, and on `main` deploy with `coverage.json`) | `make docs` after `pip install --require-hashes -r docs/requirements.txt`, and `make coverage-json` |
| OpenSSF Scorecard | not reproducible locally; runs on push to `main` and weekly, and publishes to [scorecard.dev](https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-lsp) |

`make` with no target runs the gates that need no network, in the order
they fail fastest.

`make family` reads the family manifest from passmcp's `main` branch; set
`PASSMCP_ECOSYSTEM_URL=file:///path/to/passmcp/ecosystem.json` to check
against a local passmcp checkout instead. It reads schema versions 1 and
2 of the manifest (`shipping`/`planned` and `released`/`unreleased`), and
refuses any other.

## Complexity

`make lint` enforces the portfolio's per-function ceilings through
golangci-lint: cyclomatic complexity 10 (`gocyclo`), cognitive complexity
15 (`gocognit`) and 60 lines (`funlen`), on tests as well as code. Every
function is under them, so there is no baseline of existing offenders; a
new one fails the Lint job. Halstead difficulty has no golangci-lint
analyser and is not gated.

## Coverage

The gate is 85% statement coverage in every package. `make coverage`
writes `coverage.out`; ci.yml's Coverage Gate checks each package against
the threshold. The VS Code extension's testable module, `settings.ts`, is
held to 100% line, branch and function coverage by `npm test`.

The README's coverage badge is a separate, whole-module number:
`make coverage-json` runs the tests with a cover profile and
`scripts/coveragebadge` turns it into a
[shields.io endpoint document](https://shields.io/badges/endpoint-badge),
`build/coverage.json`. It is brightgreen from 90%, green from 85%, yellow
from 70% and red below, truncated rather than rounded. The Manual workflow
publishes it on every push to `main` at
<https://sebastienrousseau.com/passmcp-lsp/coverage.json>, which the badge
reads.

## Install contract

`GNUmakefile` includes `Makefile` and adds `install`, `uninstall` and
`install-smoke`. `make install` builds the binary and its completions and
installs them under `PREFIX` (default `/usr/local`), staged under
`DESTDIR`: the binary in `bin`, completions in
`share/bash-completion/completions`, `share/zsh/site-functions` and
`share/fish/vendor_completions.d`, and the README, changelog, licence and
security policy in `share/doc/passmcp-lsp`. The binary generates no manual
page, so none is installed. `make install-smoke` stages an install in a
temporary directory, runs the installed binary, uninstalls, and fails if
anything is missing or left behind.

## Test layout

| Package | What its tests hold it to |
|---|---|
| `internal/jsondoc` | encoding/json's verdict on every document (`FuzzParse` differentially), ranges, UTF-16 positions |
| `internal/schema` | every supported keyword, the refusal of every other one, broken references |
| `internal/check` | every diagnostic code; the policy fixtures; the embedded schema; `TestPolicyCrosscheck` against a real passmcp when `PASSMCP` is set |
| `internal/guidance` | the lookup with a fake runner, a shell stand-in for passmcp (not on Windows), and `TestRealPassmcp` when `PASSMCP` is set |
| `internal/jsonrpc` | framing, every documented error, `FuzzFraming` |
| `internal/lsp` | the lifecycle, every protocol error, open/change/close, hover in each outcome |
| `cmd/passmcp-lsp` | flags, `check` in both formats, completions parsed by their shells, a stdio session |

## Generated artefacts

None are committed. Release archives and checksums are built by goreleaser
into `dist/`; `make build`, `make completions`, `make coverage-json` and
`make schema-check` write to `build/`; `make vscode` writes
`editors/vscode/out/` and `editors/vscode/passmcp-lsp.vsix`. All are
ignored.

## Release model

The version is passmcp's latest release, exactly, and a release is cut
after passmcp's, on a `feat/vX.Y.Z` branch:

1. Date the `## [X.Y.Z]` heading in `CHANGELOG.md`, write
   `docs/releases/vX.Y.Z.md`, and update the version in the install lines,
   the README's ecosystem sentence, `CITATION.cff` (`version`, and add
   `date-released`), and `editors/vscode/package.json` (then
   `npm install --package-lock-only` to move the lock). Move `go.mod` to
   passmcp-reporting's `vX.Y.Z`.
2. `make lockstep` and `make versions`.
3. `goreleaser check`, and the Release workflow's dry run.
4. Push a signed annotated tag `vX.Y.Z` with the message
   `passmcp-lsp vX.Y.Z`. The Release workflow builds the archives and the
   `.vsix`, signs the checksums, and attests them.
5. Read the tag, the release page and the checksums back before calling it
   done. Publishing the extension to a marketplace is a separate, manual
   step.

## Conventions

- Stdout is the protocol, or the selected `check` output; nothing else is
  written there.
- Every exported identifier is documented.
- Every document an editor sends is untrusted input.
