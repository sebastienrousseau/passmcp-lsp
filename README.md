<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

<p align="center">
  <img src="https://raw.githubusercontent.com/sebastienrousseau/passmcp/main/.github/logo.svg" alt="passmcp-lsp logo" width="128" />
</p>

<h1 align="center">passmcp-lsp</h1>

<p align="center">
  A language server for MCP artefacts: it checks server.json, MCP client configurations, MCP tool definitions, and passmcp policies and attestations as you type, and shows passmcp's guidance when you hover a check id.
</p>

<p align="center">
  <a href="https://github.com/sebastienrousseau/passmcp-lsp/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/sebastienrousseau/passmcp-lsp/ci.yml?branch=main&style=for-the-badge&logo=github&label=Build" alt="Build" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-lsp/blob/main/DEVELOPMENT.md#coverage"><img src="https://img.shields.io/endpoint?url=https%3A%2F%2Fsebastienrousseau.com%2Fpassmcp-lsp%2Fcoverage.json&style=for-the-badge&logo=codecov&logoColor=white" alt="Coverage" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-lsp/releases"><img src="https://img.shields.io/github/v/release/sebastienrousseau/passmcp-lsp?style=for-the-badge&color=fc8d62&logo=github&label=Release" alt="Release" /></a>
  <a href="https://pkg.go.dev/satellion.com/passmcp-lsp"><img src="https://img.shields.io/badge/go.dev-reference-007d9c?style=for-the-badge&labelColor=555555&logo=go&logoColor=white" alt="Docs" /></a>
  <a href="https://scorecard.dev/viewer/?uri=github.com/sebastienrousseau/passmcp-lsp"><img src="https://img.shields.io/ossf-scorecard/github.com/sebastienrousseau/passmcp-lsp?style=for-the-badge&label=OpenSSF%20Scorecard&logo=openssf" alt="OpenSSF Scorecard" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-blue.svg?style=for-the-badge" alt="License: Apache-2.0" /></a>
  <a href="https://github.com/sebastienrousseau/passmcp-lsp/blob/main/DEVELOPMENT.md#requirements"><img src="https://img.shields.io/badge/go-1.26.8%2B-93450a.svg?style=for-the-badge&logo=go" alt="Go 1.26.8+" /></a>
</p>

---

## Contents

**Getting started**

- [Install](#install) — `make install` from source, `go install` and release archives from the first tag, the VS Code extension, Neovim
- [Requirements](#requirements) — the Go floor, and passmcp for hover
- [Quick Start](#quick-start) — check a file from the terminal, then in the editor

**The passmcp-lsp ecosystem**

- [The passmcp-lsp ecosystem](#the-passmcp-lsp-ecosystem) — `passmcp`, `passmcp-reporting`, `passmcp-server`, `passmcp-action`, `passmcp-graph`, `passmcp-registry`, `passmcp-lsp`, `passmcp-census`, `satellion.com`

**Library reference**

- [Capabilities at a glance](#capabilities-at-a-glance) — the current surface by theme
- [Ecosystem comparison](#ecosystem-comparison) — short matrix; full table at [`docs/COMPARISON.md`](docs/COMPARISON.md)
- [Benchmarks](#benchmarks) — headline numbers; full table at [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md)
- [Features](#features) — the five artefacts and the hover
- [Configuration](#configuration) — three flags, one environment variable, one initialization option
- [Examples](#examples) — a policy, a client configuration, a CI step

**Operational**

- [When not to use passmcp-lsp](#when-not-to-use-passmcp-lsp) — limitations
- [Development](#development) — make targets, fuzzing, CI
- [Security](#security) — what it reads, what it runs, what it never sends
- [Documentation](#documentation) — all reference docs
- [Stability guarantees](#stability-guarantees) — diagnostic codes, the command line, the protocol surface
- [License](#license)

---

## Install

passmcp-lsp is one binary, `passmcp-lsp`. Editors start it and talk to it
over stdio; the same binary checks files from a terminal or a CI job with
`passmcp-lsp check`.

### As a Go program

From source, today:

```sh
git clone https://github.com/sebastienrousseau/passmcp-lsp
cd passmcp-lsp
make install PREFIX="$HOME/.local"
```

`make install` builds the binary and its bash, zsh and fish completions and
installs them under `PREFIX` (default `/usr/local`), staged under `DESTDIR`
when a packager sets it; `make uninstall` removes them. The
[GNUmakefile](GNUmakefile) holds the contract, and CI checks the staged tree
on every push.

Once `v0.0.2` is tagged, `go install` and the release archives work too:

```sh
go install satellion.com/passmcp-lsp/cmd/passmcp-lsp@v0.0.2
```

The Release workflow builds archives for Linux, macOS and Windows on amd64
and arm64, with a signed checksums file and SLSA build provenance, on every
tag. 0.0.2 is not tagged yet, so the
[releases page](https://github.com/sebastienrousseau/passmcp-lsp/releases)
is empty until it is.

### In VS Code

The extension in [`editors/vscode`](editors/vscode/README.md) starts
`passmcp-lsp` for every JSON document and stays silent on the ones that are
not MCP artefacts. It is not on the Marketplace; build and install it from
this repository (Node 22 or later):

```sh
make vscode
code --install-extension editors/vscode/passmcp-lsp.vsix
```

CI builds the same `.vsix` on every push and keeps it as a workflow
artefact, and the Release workflow attaches it to each release, covered
by the signed checksums.

### In Neovim

Neovim 0.11 or later, with `passmcp-lsp` on `PATH`:

```lua
vim.lsp.config('passmcp_lsp', {
  cmd = { 'passmcp-lsp' },
  filetypes = { 'json', 'jsonc' },
  root_markers = { '.git' },
})
vim.lsp.enable('passmcp_lsp')
```

Any other editor that starts a stdio language server works the same way:
the command is `passmcp-lsp`, for JSON and JSON-with-comments files.

---

## Requirements

| Requirement | Floor | Enforced by |
|---|---|---|
| Go (building from source) | the `go` directive in [`go.mod`](go.mod), 1.26.8 | CI tests on that version and on latest stable, on Linux, macOS and Windows |
| passmcp (hover guidance only) | the version this repository is in lockstep with, on `PATH` or named with `--passmcp` | the `crosscheck` job asks passmcp at that tag for guidance, and for its verdict on every policy fixture |
| An editor | any that starts a stdio language server and sends whole or incremental text changes | the protocol tests in `internal/lsp` and `make smoke` |
| Node (building the VS Code extension only) | 22 | the `vscode` job builds and tests the extension on Node 22 |

The Go floor is raised only when a release needs a language feature, on a
patch release like everything else pre-1.0, and the changelog says so.

---

## Quick Start

```sh
make build
printf '{"version": 1, "name": "platform", "must_pas": ["net.tls"]}\n' > company.policy.json
build/passmcp-lsp check company.policy.json
```

```text
company.policy.json:1:1: info: the policy states no rule, so nothing will be judged [policy/no-rules]
company.policy.json:1:36: error: "must_pas" is not a key of a policy; passmcp refuses a policy with a field it does not know, because a misspelled rule would otherwise silently not apply [policy/unknown-key]
```

`check` prints one line per problem, `file:line:column`, and exits 1 when
any is an error. In an editor the same diagnostics appear as you type, and
hovering `net.tls` in a corrected `must_pass` list shows what passmcp says
the check means and how to fix it.

---

## The passmcp-lsp ecosystem

Every component is released at **0.0.2** and moves in lockstep: one version across the family, released together ([docs/ecosystem.md](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)).

| Component | Purpose | Use case |
| :--- | :--- | :--- |
| [passmcp](https://github.com/sebastienrousseau/passmcp) | The MCP server diagnostic: checks in nine phases, every finding tied to the request that showed it, signed attestations | Test a server before your agents trust it, and gate it in CI |
| [passmcp-reporting](https://github.com/sebastienrousseau/passmcp-reporting) | The attestation format, its JSON Schemas and offline verifier, the graph model, and the agentgateway processor | Verify an attestation in a gateway, registry or pipeline |
| [passmcp-server](https://github.com/sebastienrousseau/passmcp-server) | passmcp's diagnostics as read-only MCP tools | Evaluate a server, or check an attestation, from inside the agent |
| [passmcp-action](https://github.com/sebastienrousseau/passmcp-action) | passmcp in GitHub Actions and GitLab CI, the image pinned by digest | Fail a build on the findings you choose |
| [passmcp-graph](https://github.com/sebastienrousseau/passmcp-graph) | A local graph of agents, servers, tools and identities built from attestations | Find inherited risk and over-privilege, and gate on policy |
| [passmcp-registry](https://github.com/sebastienrousseau/passmcp-registry) | A signed public scorecard of the MCP Registry's remote servers | Check a public server's standing before connecting to it |
| [passmcp-lsp](https://github.com/sebastienrousseau/passmcp-lsp) | A language server for MCP artefacts, with check-id hover from the guidance catalogue | Catch mistakes in server.json, tool schemas and client configuration while editing |
| [passmcp-census](https://github.com/sebastienrousseau/passmcp-census) | The published reliability census: dataset, methodology, disclosure log and reproduction command | Cite ecosystem-wide reliability figures, and reproduce them |
| [satellion.com](https://github.com/sebastienrousseau/satellion.github.io) | The website, the Go module paths and the format URIs | Read the manual, and resolve `satellion.com/...` imports |

This repository is the editing surface: it reads the formats the rest of
the family defines and never chooses its own rules for them. Attestations
are verified by passmcp-reporting's own verifier, hover guidance comes from
the installed passmcp, and the policy rules are checked against passmcp's
verdicts in CI. `make lockstep` checks that the version is passmcp's latest
release, `make family` checks this repository's row in the family
manifest, and `make versions` checks that every file naming the version,
and the passmcp-reporting module in `go.mod`, agree on it.

---

## Capabilities at a glance

Nothing here is released yet: 0.0.2 is the first version, and its entry is
in the [CHANGELOG](CHANGELOG.md#002--2026-09-29).

| Area | Capability | Status |
| :--- | :--- | :--- |
| Registry listings | `server.json` validated against the MCP Registry's 2025-12-11 schema, embedded and checked byte for byte against the published copy | [Not yet released](CHANGELOG.md#002--2026-09-29) |
| Client configuration | `claude_desktop_config.json`, `.mcp.json`, `.cursor/mcp.json` and `.vscode/mcp.json`: entry shape, URLs, literal credentials | [Not yet released](CHANGELOG.md#002--2026-09-29) |
| Tool definitions | The MCP 2025-11-25 `Tool` rules: names, `inputSchema` and `outputSchema` roots, JSON Schema types, annotation hints | [Not yet released](CHANGELOG.md#002--2026-09-29) |
| passmcp policies | Every rule passmcp 0.0.2 applies when it reads a policy, held to passmcp's own verdicts by the `crosscheck` job | [Not yet released](CHANGELOG.md#002--2026-09-29) |
| passmcp attestations | Structure and integrity, verified offline by passmcp-reporting 0.0.2 | [Not yet released](CHANGELOG.md#002--2026-09-29) |
| Hover | passmcp's guidance for a check id in a policy or an attestation, from the installed passmcp | [Not yet released](CHANGELOG.md#002--2026-09-29) |
| Terminal and CI | `passmcp-lsp check`, text or JSON, exit 1 on any error | [Not yet released](CHANGELOG.md#002--2026-09-29) |
| Editors | A VS Code extension built into a `.vsix` in CI; a verified Neovim configuration | [Not yet released](CHANGELOG.md#002--2026-09-29) |

---

## Ecosystem comparison

An editor's built-in JSON support can validate `server.json` against the
schema its `$schema` names, and `check-jsonschema` can do the same from a
terminal. Neither knows what a client configuration, a passmcp policy or an
attestation is, and neither can explain a passmcp check.

| Tool | `server.json` against the registry schema | Client configs, policies and attestations | passmcp guidance on hover |
| :--- | :---: | :---: | :---: |
| **passmcp-lsp** | yes, embedded | yes | yes, with passmcp installed |
| An editor's JSON schema support | when `$schema` is set and reachable | no | no |
| `check-jsonschema` | yes, from a terminal | no | no |
| `passmcp verify --policy` | no | policies and attestations, when run | no |

See [`docs/COMPARISON.md`](docs/COMPARISON.md) for the evidence and complete matrix.

---

## Benchmarks

Analysis runs on every change to an open document, so it has to be well
under a keystroke. Measured on this tree with `go test -bench` and
[hyperfine](https://github.com/sharkdp/hyperfine), on a machine that was
running other builds at the time (load average about 42), so the spread
is wide and the lower figures are the better guide.

| Scenario | Result | Environment |
| :--- | ---: | :--- |
| Analyse a 1 KB `server.json` against the registry schema | 51–71 µs per analysis (3 runs) | Apple A18 Pro, Go 1.27.1, 2026-09-29 |
| Analyse the worked-example policy | 16–18 µs per analysis (3 runs) | same |
| `passmcp-lsp check` on that `server.json`, process start to exit | 10.9 ms ± 2.9 ms mean, 7.4 ms min (30 runs) | same |
| One hover lookup: `passmcp explain` for one check id | 69.5 ms ± 73.3 ms mean, 23.2 ms min (30 runs) | same, passmcp 0.0.1 |

See [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md) for methodology and full results.

---

## Features

**Five artefacts, recognised by name and by content.** A file named
`server.json`, `claude_desktop_config.json`, `.mcp.json`, `.cursor/mcp.json`,
`.vscode/mcp.json`, `*.policy.json` or `passmcp-policy.json` is what its
name says. Any other JSON file is recognised by what it holds: an in-toto
statement with a `https://satellion.com/attestation/` predicate, a
`$schema` naming the registry's `server.schema.json`, an `mcpServers`
block, a passmcp policy rule, or a tool with an `inputSchema`. Every other
JSON file gets no diagnostics at all.

**`server.json` against the registry's schema.** The 2025-12-11 schema the
MCP Registry publishes is embedded, and `make schema-check` fails when the
embedded copy differs from the published one. The validator implements the
part of JSON Schema that schema uses and refuses to load one that uses
more, rather than skip a rule it does not know.

**Client configurations as passmcp reads them.** The `mcpServers` layout
(Claude Desktop, Claude Code, Cursor) and VS Code's `servers` layout, with
comments and trailing commas allowed. Each server needs a command or an
`http(s)` URL; `args`, `env` and `headers` must be strings; a credential
written into `env` or `headers`, rather than referenced as `${...}`, is a
warning.

**Tool definitions as the MCP specification states them.** A single tool,
an array, or a `tools/list` result, checked against revision 2025-11-25:
a MUST is an error and a SHOULD a warning. A tool without `readOnlyHint`
is noted, because passmcp does not invoke it by default.

**Policies passmcp will accept.** Unknown keys, versions, names, targets,
limits, severities and exemptions, each with the reason passmcp gives, plus
a warning for an exemption that has expired. The fixtures in
`internal/check/testdata/policy` are the evidence: CI applies each one with
the released passmcp and fails if the two disagree.

**Attestations verified offline.** An attestation is checked by
passmcp-reporting's own verifier, one diagnostic per problem, placed on the
field it concerns. Who signed it is the envelope's business, not this
check's.

**Guidance on hover, from passmcp.** Hovering a check id in `must_pass`,
`must_not_fail`, an exemption or an attestation verdict shows what the check
means and the steps to fix it, as the installed passmcp states them. The
catalogue is passmcp's and is not copied here; without passmcp, the hover
says how to install it.

---

## Configuration

| Flag | Environment | Default | Meaning |
|---|---|---|---|
| `--passmcp` | `PASSMCP_LSP_PASSMCP` | `passmcp` on `PATH` | The passmcp program hover asks for guidance |
| `--stdio` | — | on | Serve over stdin and stdout; the only transport, accepted because clients pass it |
| `--version` | — | — | Print the version and exit |
| `--completion` | — | — | Print a completion script for `bash`, `zsh` or `fish`, and exit |

A flag overrides its environment variable. A client may also name the
passmcp program in the `passmcpPath` initialization option, which overrides
both; the VS Code extension sends its `passmcpLsp.passmcpPath` setting
there.

`passmcp-lsp check [--output text|json] FILE...` checks files and exits 0
when none has an error, 1 when one does, and 2 when it could not run.

Shell completions come from the flag set:

```sh
passmcp-lsp --completion bash > /etc/bash_completion.d/passmcp-lsp
passmcp-lsp --completion zsh > "${fpath[1]}/_passmcp-lsp"
passmcp-lsp --completion fish > ~/.config/fish/completions/passmcp-lsp.fish
```

---

## Examples

A VS Code configuration with a token written into it:

```json
{
  "servers": {
    "weather": {
      "type": "http",
      "url": "https://mcp.example.com/weather",
      "headers": { "Authorization": "Bearer sk-live-1234" }
    }
  }
}
```

```text
.vscode/mcp.json:6:37: warning: server "weather": headers.Authorization holds a literal credential; anyone who can read this file has it. Reference it instead, for example ${env:NAME} or ${input:name} where the host supports them [client/literal-credential]
```

The same check as a CI step, with the results as JSON for an annotator:

```sh
passmcp-lsp check --output json server.json .vscode/mcp.json company.policy.json
```

Every diagnostic and the code it carries is listed in
[`docs/diagnostics.md`](docs/diagnostics.md).

---

## When not to use passmcp-lsp

- **To judge a running server.** passmcp-lsp reads files. Whether a server
  behaves is [passmcp](https://github.com/sebastienrousseau/passmcp)'s
  question, asked of the server itself.
- **For a server's source code.** It reads MCP artefacts, not the Go,
  TypeScript or Python that implements a server.
- **For every client's configuration format.** It knows the layouts
  passmcp reads: `mcpServers` and VS Code's `servers`. Zed's
  `context_servers` and other hosts' files are not recognised by name.
- **To check who signed an attestation.** It checks structure and
  integrity; `cosign` or `gh attestation verify` checks the signature.
- **For a policy file with no rule and an unconventional name.** A policy
  is recognised by a rule key or by a `.policy.json` name; one holding only
  `version` and `name` under another name is left alone.

---

## Development

```bash
make              # format, vet, lint, headers, tests, stdio smoke test
make test-race    # race detector, randomised order
make fuzz         # the JSON-RPC framing and JSON parser fuzz targets
make crosscheck   # policy rules and hover against the released passmcp
make schema-check # the embedded server.json schema is the published one
make vscode       # test and package the VS Code extension
make install-smoke  # install and uninstall under a staged DESTDIR
make coverage-json  # build/coverage.json, the document behind the badge
```

Every gate CI runs has a local form; [DEVELOPMENT.md](DEVELOPMENT.md) maps
them, and says why the module's one dependency is passmcp-reporting and
nothing else.

---

## Security

passmcp-lsp reads the documents an editor sends it and writes diagnostics
back; it makes no network request. The one program it runs is passmcp, for
hover, as `passmcp explain --output json` with a one-finding report on
stdin and `PASSMCP_CONFIG` pointing at an empty file, so no default in the
operator's own passmcp configuration can make a hover send anything
anywhere. The check id is JSON-encoded into that report, never placed on a
command line. In VS Code, both program paths are machine settings, so a
repository's workspace settings cannot choose what runs. Every input is
bounded: a protocol message at 16 MiB, JSON nesting at 256 levels, passmcp's
output at 1 MiB. The JSON-RPC framing and the JSON parser are fuzzed; CI runs
`govulncheck` on every push.

Report vulnerabilities according to [`SECURITY.md`](SECURITY.md).

---

## Documentation

The four entry points, identical across every repo in the family:

- **[User Manual](https://sebastienrousseau.com/passmcp-lsp/)** — this repository's manual: the artefacts, every diagnostic, editor setup
- **[API reference](https://pkg.go.dev/satellion.com/passmcp-lsp)** — this module's packages
- **[Developer docs](DEVELOPMENT.md)** — toolchain, task map, reproducing every CI gate locally
- **[Ecosystem map](https://github.com/sebastienrousseau/passmcp/blob/main/docs/ecosystem.md)** — the family, the published artefacts, the lockstep version rule

| Document | Covers |
|---|---|
| [`docs/diagnostics.md`](docs/diagnostics.md) | Every diagnostic code, its severity and what triggers it |
| [`docs/editors.md`](docs/editors.md) | VS Code, Neovim and any other stdio client |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | How a document flows from the editor to a diagnostic |
| [`docs/adr/`](docs/adr/README.md) | Decision records for this repository |
| [`docs/COMPARISON.md`](docs/COMPARISON.md) | passmcp-lsp beside the other ways to check these files |
| [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md) | What an analysis and a hover cost, and how it was measured |
| [`docs/releases/`](docs/releases/v0.0.2.md) | Release highlights, one file per release |
| [`SECURITY.md`](SECURITY.md) | Disclosure policy, supported versions, what is guaranteed |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Signed-commit and DCO policy, what a change needs |
| [`CHANGELOG.md`](CHANGELOG.md) | Per-release notes, and the lockstep version rule |
| [`SUPPORT.md`](SUPPORT.md) | Where to ask, and what to expect |

---

## Stability guarantees

passmcp-lsp is pre-1.0, carries passmcp's version, and follows SemVer with
the patch digit moving for everything until 1.0.

**The breaking axis is what an editor, a CI job or a reader relies on.**
These are breaking:

- Removing or renaming a diagnostic code, or changing its severity
- Reporting an error for a document the previous release accepted, other
  than to follow passmcp, the MCP specification or the registry schema
- Removing or renaming a flag, the `check` subcommand, its output fields or
  its exit statuses
- Dropping a protocol capability the server advertises

New diagnostic codes, new recognised artefacts, new output fields and a
newer registry schema are **not** breaking. A diagnostic's message text is
for people and may change in any release; match on the code.

**Deprecation window.** A deprecated code, flag or output field keeps
working for at least one release after the release that announces it.

---

## License

Licensed under the **[Apache License, Version 2.0](LICENSE)**.

Apache-2.0 because passmcp-lsp ships inside editors and extension
marketplaces whose licensing is not passmcp's. The embedded MCP Registry
schema is its upstream's, under the terms stated in
[`REUSE.toml`](REUSE.toml).

<p align="right"><a href="#contents">Back to Top</a></p>
