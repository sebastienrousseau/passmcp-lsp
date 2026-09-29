<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# passmcp-lsp

A language server for MCP artefacts. It checks `server.json`, MCP client
configurations, MCP tool definitions, and passmcp policies and attestations
as you type, and shows passmcp's guidance when you hover a check id. The
same checks run from a terminal or CI with `passmcp-lsp check`.

| Page | Covers |
|---|---|
| [Diagnostics](diagnostics.md) | How a file is recognised, and every diagnostic code |
| [Editors](editors.md) | VS Code, Neovim and any other stdio client |
| [Architecture](ARCHITECTURE.md) | How a document flows from the editor to a diagnostic |
| [Comparison](COMPARISON.md) | passmcp-lsp beside the other ways to check these files |
| [Benchmarks](BENCHMARKS.md) | What an analysis and a hover cost |
| [Decision records](adr/README.md) | Why it is built the way it is |

Install, requirements and the command line are in the
[README](https://github.com/sebastienrousseau/passmcp-lsp/blob/main/README.md).
What each passmcp check means is in
[passmcp's manual](https://satellion.com/passmcp/docs/).

The README's coverage badge reads
[coverage.json](https://sebastienrousseau.com/passmcp-lsp/coverage.json),
published with this site by the Manual workflow from the statement coverage
CI measures on `main`.
