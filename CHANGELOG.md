<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Changelog

All notable changes to passmcp-lsp are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions are
[Semantic Versioning](https://semver.org/) shaped.

**This repository carries passmcp's version.** It is in lockstep with
[passmcp](https://github.com/sebastienrousseau/passmcp): every repository in
the family carries one version, and a release here with nothing in it is the
version rule working.

## [0.0.4] — 2026-09-30

The family's fourth release. Nothing in the language server changed; the
version moves with passmcp 0.0.4.

### Added

- **A README demo**, rendered from `.github/demo.tape` by `make demo`: the
  check command flagging a misspelled key in a passmcp policy, then
  passing once it is fixed.

### Changed

- **Release pages are published in the family layout** by the release
  workflow itself (Highlights, What's Changed, Checksums, Full Changelog),
  so no page is rewritten by hand after a release.
- **In lockstep with passmcp 0.0.4.** passmcp-lsp requires
  passmcp-reporting v0.0.4, whose attestation API is unchanged, and its
  policy rules and hover are cross-checked against passmcp v0.0.4.

### Fixed

- **The README's install section**: it said `go install` waits on
  satellion.com serving the module's import path. The site has served it
  since its 0.0.3 deploy, and `go install
  satellion.com/passmcp-lsp/cmd/passmcp-lsp@v0.0.3` works.

## [0.0.3] — 2026-09-30

The family's third release. Nothing in the language server changed; the
version moves with passmcp, whose family manifest now lists passmcp-lsp
as released.

### Changed

- **The VS Code extension builds with TypeScript 7.0** and the Node 26
  type definitions. The VS Code API types stay at 1.91, the oldest VS
  Code the extension supports, so `vsce` still packages it for the same
  editors.

### Fixed

- **The README says 0.0.2 is released.** It still called every capability
  "Not yet released" and the tag missing after 0.0.2 shipped; it also
  said `go install` works, which it does not until satellion.com serves
  the module's import path.

## [0.0.2] — 2026-09-29

The first version. passmcp-lsp was never tagged at 0.0.1, so it first
ships at the family's version, 0.0.2.

### Added

- **A language server for MCP artefacts**, over stdio: diagnostics on open
  and on every change, whole or incremental, and hover.
- **`server.json`** validated against the MCP Registry's 2025-12-11 schema,
  embedded byte for byte, by a JSON Schema validator that refuses to load a
  schema using a keyword it does not implement.
- **MCP client configurations**: the `mcpServers` layout (Claude Desktop,
  Claude Code, Cursor) and VS Code's `servers` layout, with comments and
  trailing commas; a literal credential in `env` or `headers` is a warning.
- **MCP tool definitions**, against the Tool rules of MCP 2025-11-25.
- **passmcp policies**, with the rules passmcp 0.0.2 applies and its
  reasons; a CI job holds them to passmcp's own verdicts.
- **passmcp attestations**, verified offline by passmcp-reporting 0.0.2.
- **Hover on check ids** in policies and attestations, with the guidance
  the installed passmcp gives; without passmcp the hover says how to get it.
- **`passmcp-lsp check`**, the same analysis from a terminal or CI, as text
  or JSON, exiting 1 on any error.
- **A VS Code extension** in `editors/vscode`, built into a `.vsix` in CI
  and attached to each release, and a Neovim configuration.
- Release archives for Linux, macOS and Windows on amd64 and arm64 with
  signed checksums and SLSA provenance; `make install` with bash, zsh and
  fish completions.

[0.0.4]: https://github.com/sebastienrousseau/passmcp-lsp/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/sebastienrousseau/passmcp-lsp/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/sebastienrousseau/passmcp-lsp/commits/main
