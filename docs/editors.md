<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Editors

passmcp-lsp is a stdio language server: an editor starts the `passmcp-lsp`
program and exchanges Language Server Protocol messages with it over stdin
and stdout. Any editor that can do that can use it, for JSON and
JSON-with-comments files. The server recognises the MCP artefacts among
those files and says nothing about the rest.

## VS Code

The extension in `editors/vscode` starts the server for every JSON
document. It does not bundle the server; install `passmcp-lsp` first, then
build and install the extension from a checkout:

```sh
make vscode
code --install-extension editors/vscode/passmcp-lsp.vsix
```

| Setting | Default | Meaning |
|---|---|---|
| `passmcpLsp.path` | `passmcp-lsp` | The server program: a path, or a name on `PATH` |
| `passmcpLsp.passmcpPath` | empty | The passmcp program hover asks for guidance |
| `passmcpLsp.trace.server` | `off` | Trace the protocol in the output panel |

The two paths are machine-scoped settings. A workspace's
`.vscode/settings.json` cannot set them, so opening a repository cannot
choose which program the extension starts. Changing either restarts the
server.

The extension is not published to a marketplace.

## Neovim

Neovim 0.11 or later has the configuration built in:

```lua
vim.lsp.config('passmcp_lsp', {
  cmd = { 'passmcp-lsp' },
  filetypes = { 'json', 'jsonc' },
  root_markers = { '.git' },
})
vim.lsp.enable('passmcp_lsp')
```

To name the passmcp program for hover, add
`init_options = { passmcpPath = '/path/to/passmcp' }`, or start the server
with `cmd = { 'passmcp-lsp', '--passmcp', '/path/to/passmcp' }`.

## Any other editor

| What the editor needs | Value |
|---|---|
| Command | `passmcp-lsp` (it accepts `--stdio`, which some clients add) |
| Transport | stdio |
| Languages | `json`, `jsonc` |
| Initialization options | optional: `{"passmcpPath": "/path/to/passmcp"}` |

The server asks for whole-document sync and applies incremental changes as
well when a client sends them. It advertises diagnostics (pushed with
`textDocument/publishDiagnostics`) and hover.
