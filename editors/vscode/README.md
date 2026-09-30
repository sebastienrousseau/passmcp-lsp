<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# passmcp: MCP artefacts

Diagnostics for the files an MCP project edits by hand, from
[passmcp-lsp](https://github.com/sebastienrousseau/passmcp-lsp):

- **`server.json`**, checked against the MCP Registry's published schema
- **MCP client configurations**: `claude_desktop_config.json`, `.mcp.json`,
  `.cursor/mcp.json` and `.vscode/mcp.json`
- **MCP tool definitions**: a tool, an array of tools, or a `tools/list` result
- **passmcp acceptance policies** and **passmcp attestations**

Hover a check id in a policy or an attestation to read passmcp's guidance
for it.

## Requirements

This extension starts the `passmcp-lsp` program; it does not bundle it.
Install it from the
[releases page](https://github.com/sebastienrousseau/passmcp-lsp/releases)
or with Go:

```sh
go install satellion.com/passmcp-lsp/cmd/passmcp-lsp@v0.0.4
```

Hover guidance needs [passmcp](https://github.com/sebastienrousseau/passmcp)
on `PATH`; without it the hover says so and everything else works.

## Settings

| Setting | Default | Meaning |
|---|---|---|
| `passmcpLsp.path` | `passmcp-lsp` | The passmcp-lsp program: a path, or a name on `PATH` |
| `passmcpLsp.passmcpPath` | empty | The passmcp program hover asks for guidance |
| `passmcpLsp.trace.server` | `off` | Trace the protocol in the output panel |

Both paths are machine settings: a workspace cannot set them, so opening a
repository cannot choose which program the extension runs.

## Licence

Apache-2.0.
