<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Diagnostics

Every diagnostic passmcp-lsp reports, by code. The code is the stable part:
match on it in CI, not on the message, which is written for people and may
be reworded in any release. A test fails when a code in the source is
missing from this page.

Severities follow the Language Server Protocol: **error** is something the
format's owner refuses, **warning** something it advises against,
**info** something worth knowing. `passmcp-lsp check` exits 1 on an error
and 0 otherwise.

## How a file is recognised

By name first, then by content. A file that is none of these gets no
diagnostics at all.

| Artefact | Recognised by |
|---|---|
| `server.json` | the name `server.json`, or a `$schema` naming the registry's `server.schema.json` |
| Client configuration | the names `claude_desktop_config.json`, `.mcp.json`, `.cursor/mcp.json`, `.vscode/mcp.json`, or an `mcpServers` block |
| Tool definitions | an object with an `inputSchema`, an array of them, or a `{"tools": [...]}` result |
| passmcp policy | a name ending `.policy.json`, the name `passmcp-policy.json`, or any policy rule key |
| passmcp attestation | an in-toto `_type` with a `https://satellion.com/attestation/` predicate type |

## Any recognised file

| Code | Severity | When |
|---|---|---|
| `syntax` | error | The text is not JSON. A client configuration may use comments and trailing commas; nothing else may. |
| `duplicate-key` | warning | A key appears twice in one object; JSON readers disagree about which value wins. |

## `server.json`

Checked against the MCP Registry schema revision 2025-12-11, embedded from
<https://static.modelcontextprotocol.io/schemas/2025-12-11/server.schema.json>.

| Code | Severity | When |
|---|---|---|
| `server-json/schema` | error | The listing breaks the schema: a missing required property, a wrong type, a pattern, a length, an enumeration or a URI. The message names the location, like `packages[0].transport.type`. |
| `server-json/schema-version` | info | The listing names no `$schema`, or names a revision other than the embedded one. |

## Client configurations

The layouts passmcp itself reads: `mcpServers` (Claude Desktop, Claude
Code, Cursor) and `servers` (VS Code's `.vscode/mcp.json`).

| Code | Severity | When |
|---|---|---|
| `client/no-servers` | error | There is no server block, or it is not an object of named servers. |
| `client/wrong-layout` | warning | The file uses the other host's key: `mcpServers` in `.vscode/mcp.json`, or `servers` where `mcpServers` is read. |
| `client/entry` | error | A server is not an object, has neither `command` nor `url`, or has a `command`, `args`, `env` or `headers` of the wrong type. |
| `client/url` | error | A `url` is not an absolute `http` or `https` URL. |
| `client/command-and-url` | warning | A server has both; passmcp reads it as a program and ignores the URL. |
| `client/literal-credential` | warning | An `env` or `headers` entry whose name suggests a credential holds a literal value rather than a `${...}` reference. |

## Tool definitions

The `Tool` rules of the MCP specification, revision 2025-11-25. A MUST is an
error, a SHOULD a warning.

| Code | Severity | When |
|---|---|---|
| `tool/shape` | error | A tool is not an object. |
| `tool/name` | error, warning | No name, or not a string (error); outside 1–128 characters of `A-Z a-z 0-9 _ - .`, or a duplicate (warning). |
| `tool/input-schema` | error | No `inputSchema`, or one whose root is not `"type": "object"`, or whose `required` is not an array. |
| `tool/output-schema` | error | An `outputSchema` whose root is not an object schema. |
| `tool/schema-type` | error | A `type` keyword anywhere in either schema names something that is not a JSON Schema type. |
| `tool/required-undeclared` | warning | A `required` name that `properties` does not declare. |
| `tool/annotations` | error, warning | `annotations` is not an object or a hint is not a boolean (error); `destructiveHint` set on a read-only tool, where it means nothing (warning). |
| `tool/read-only-hint` | info | No `readOnlyHint`: it defaults to false, and passmcp does not invoke the tool by default. |
| `tool/execution` | error | `execution.taskSupport` is not `forbidden`, `optional` or `required`. |
| `tool/description` | info | No description; an agent chooses tools by their descriptions. |

## passmcp policies

The policy format passmcp 0.0.2 reads, with the reasons passmcp gives. Every
fixture in `internal/check/testdata/policy/` is applied with the released
passmcp in CI, which must refuse exactly the ones this page calls errors.

| Code | Severity | When |
|---|---|---|
| `policy/unknown-key` | error | A key passmcp does not know, at the top, in `target` or in an exemption. |
| `policy/version` | error | No version, not a whole number, not positive, or later than format 1. |
| `policy/name` | error | No name, or an empty one. |
| `policy/type` | error | A field of the wrong JSON type. |
| `policy/target` | error | A transport other than `http` or `stdio`, or a target block that constrains nothing. |
| `policy/limit` | error | A negative `max_fail` or `max_warn`, a score outside 0–100, or an unnamed category. |
| `policy/severity` | error | A `forbid_severity` other than `critical`, `major` or `minor`. |
| `policy/check-id` | error | An empty or repeated check id, or one both required and exempted. |
| `policy/exemption` | error | An exemption with no check, no reason, no expiry, a date not in `YYYY-MM-DD` form, or a check exempted twice. |
| `policy/expired` | warning | An exemption whose date has passed; passmcp counts it as a failed rule. |
| `policy/no-rules` | info | The policy states no rule, so nothing will be judged. |

Hovering a check id in `must_pass`, `must_not_fail` or an exemption's
`check` shows passmcp's guidance for it.

## passmcp attestations

Verified offline by passmcp-reporting 0.0.2's `attestation.Parse` (the
`mcp-evaluation/v1` predicate) and `a2a.Parse` (`a2a-evaluation/v1`).

| Code | Severity | When |
|---|---|---|
| `attestation/invalid` | error | One per problem the verifier reports, placed on the key it concerns. |
| `attestation/predicate-type` | warning | A `https://satellion.com/attestation/` predicate type the verifier does not know. |

Hovering a verdict's `id` shows passmcp's guidance for it, with the
verdict's own `doc` link when the attestation carries one.
