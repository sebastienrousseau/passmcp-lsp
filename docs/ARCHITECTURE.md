<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture

How a document gets from the editor to a diagnostic, and a hover to
passmcp's guidance.

## The packages

| Package | Owns |
|---|---|
| `cmd/passmcp-lsp` | Flags, server mode, `check` mode, completions |
| `internal/jsonrpc` | Content-Length framing, message envelopes, bounded reads |
| `internal/lsp` | The session: lifecycle, open documents, publishing, hover |
| `internal/check` | Recognising an artefact and every diagnostic about it |
| `internal/schema` | The JSON Schema subset `server.json` needs, fail-closed |
| `internal/jsondoc` | A JSON and JSON-with-comments parser that keeps byte ranges, and UTF-16 positions |
| `internal/guidance` | Asking the installed passmcp for a check's guidance |

The dependency direction is one way: `lsp` uses `check`, `guidance` and
`jsonrpc`; `check` uses `schema` and `jsondoc`; nothing below `lsp` knows
the protocol exists, which is why `check` mode needs no server.

## A change to a document

1. The read loop takes one framed message from stdin
   (`jsonrpc.Reader`), decodes the envelope and dispatches it.
2. `didOpen` stores the text; `didChange` applies each change, whole or by
   range; either calls `update`.
3. `update` runs `check.Analyze(path, text)`. Recognition is by file name
   first and content second; an unrecognised document ends here with no
   diagnostics.
4. The document is parsed strictly, and, for a client configuration, again
   with comments and trailing commas allowed. A parse failure is one
   `syntax` diagnostic.
5. The artefact's analyzer walks the tree. Every node knows its byte range,
   so each diagnostic carries one.
6. `publish` turns byte ranges into line and UTF-16 character positions and
   sends `textDocument/publishDiagnostics`.

The read loop is the only goroutine that touches documents.

## A hover

1. `IDAt` finds the string under the cursor and checks where it is: a
   policy's `must_pass`, `must_not_fail` or exemption `check`, or an
   attestation verdict's `id`.
2. The lookup runs in its own goroutine, with the id and range copied out
   of the document, because it waits on a program.
3. `guidance.Passmcp` writes a one-finding report, encodes the id into it,
   and runs `passmcp explain --output json` with the report on stdin and
   `PASSMCP_CONFIG` pointing at an empty file. The entry for the id is
   the guidance; the answer is cached for the life of the process.
4. The hover is Markdown: what the check means, the steps, the verdict's
   documentation link when the attestation gave one.

## Where each rule comes from

| Artefact | Source of truth | How this repository follows it |
|---|---|---|
| `server.json` | The MCP Registry's published schema | Embedded byte for byte; `make schema-check` compares it with the URL |
| Client configuration | passmcp's `internal/clientconf` and `internal/discover` at 0.0.1 | The same layouts and fields, reimplemented; see the file's comments |
| Tool definitions | MCP specification 2025-11-25, `schema.ts` and `server/tools` | Each rule's MUST or SHOULD sets its severity |
| Policies | passmcp 0.0.1's policy reader, documented in its manual | `make crosscheck` applies every fixture with the released passmcp |
| Attestations | passmcp-reporting 0.0.1 | Its verifiers are called; nothing is reimplemented |
| Guidance | passmcp's catalogue | Asked of the installed passmcp at hover time |
