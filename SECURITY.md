<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Security Policy

passmcp-lsp runs inside an editor, on every JSON file the editor opens,
including files in repositories nobody has reviewed yet. Its security
posture is about three things: it reads documents and never acts on them,
the only program it starts is the operator's own passmcp, and a hover never
sends anything off the machine.

## Reporting a Vulnerability

Report security issues through [GitHub's private vulnerability reporting](https://github.com/sebastienrousseau/passmcp-lsp/security/advisories/new). Do not open a public issue.

You will receive an acknowledgement within **72 hours**. A confirmed
vulnerability is fixed and released within **90 days** of the report, or
sooner when a fix is straightforward; if the window cannot be met you will
be told why and given a revised date.

A way to make passmcp-lsp start a program other than the configured
passmcp, pass it arguments derived from a document, make a network
request, crash or hang on a document, or report a document valid that its
source says is not, is a vulnerability.

## Supported Versions

Only the latest release is supported. The version is passmcp's; see the
lockstep rule in [CHANGELOG.md](CHANGELOG.md).

## Security Measures

Each item names the test that enforces it, so the claim can be checked
rather than taken on trust.

- **No network.** Nothing in the module opens a connection; the schema it
  validates against is embedded. `make schema-check` is the only step that
  fetches anything, and it runs in CI, not in the server.
- **One program, fixed arguments.** Hover runs `passmcp explain --output
  json` with no shell. The check id travels JSON-encoded in the report on
  stdin, never on the command line. `TestReportEncodesTheID` and
  `TestExecRunner` in `internal/guidance`.
- **No operator configuration.** passmcp runs with `PASSMCP_CONFIG`
  pointing at an empty file, so no default in the operator's passmcp
  configuration, such as a model for `explain` to send findings to, applies
  to a hover. `TestExecRunner`.
- **Workspaces cannot choose the program.** The VS Code extension's
  `passmcpLsp.path` and `passmcpLsp.passmcpPath` are machine-scoped, so a
  repository's `.vscode/settings.json` cannot set them.
- **Bounded input.** A protocol message is at most 16 MiB and a header line
  4 KiB (`TestReadErrors`), JSON nesting at most 256 levels
  (`TestDepthLimit`), passmcp's output at most 1 MiB (`TestExecRunner`),
  and a hover lookup at most 10 seconds.
- **Fuzzed parsers.** `FuzzFraming` holds the JSON-RPC reader to its
  documented errors; `FuzzParse` holds the JSON parser to encoding/json's
  verdict on every input. `make fuzz` runs both, and CI runs them on every
  push.
- **Fail-closed schema validation.** The validator refuses a schema that
  uses a keyword it does not implement. `TestCompileRefusals`, and
  `TestEmbeddedSchemaIsTheRegistrys` for the embedded schema.
- **Untrusted links stay links.** A documentation URL taken from an
  attestation is percent-encoded before it becomes a Markdown link.
  `TestMarkdown`.
- **Supply chain.** One direct dependency, passmcp-reporting, which has
  none. CI runs `govulncheck` on every push. Release checksums are signed
  with cosign keyless and carry SLSA build provenance.
