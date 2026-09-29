<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Comparison

The other ways to check the files passmcp-lsp reads, and what each one
covers. Every "yes" for another tool is a property of that tool as it
describes itself; every "yes" for passmcp-lsp names the test behind it.

| | passmcp-lsp | An editor's JSON schema support | `check-jsonschema` | `passmcp verify --policy` |
|---|---|---|---|---|
| `server.json` against the registry schema | yes, embedded (`TestServerJSON`) | when the file's `$schema` names it and it can be fetched | yes, given the schema | no |
| Client configurations | yes (`TestClientConfig`) | only with a schema someone supplies | only with a schema someone supplies | no |
| MCP tool definitions | yes (`TestToolsProblems`) | no | no | no |
| passmcp policies | yes (`TestPolicyCrosscheck`) | no | no | yes, when a run applies one |
| passmcp attestations | yes, with passmcp-reporting's verifier (`TestAttestation`) | no | against the published schema, without the digest check | yes |
| passmcp guidance for a check id | yes, on hover (`TestHover`) | no | no | no |
| While editing | yes | yes | no | no |
| In CI | `passmcp-lsp check` | no | yes | yes |

## Evidence

- **Agreement with `check-jsonschema` on `server.json`.** On 2026-09-29,
  five listings (a minimal valid one, one with four errors, passmcp-server's
  own listing, one with remotes and icons, one with three invalid values)
  were checked by both `check-jsonschema --schemafile` and
  `passmcp-lsp check` against the embedded schema; both accepted the same
  three and refused the same two.
- **Agreement with passmcp on policies.** `TestPolicyCrosscheck` applies
  every fixture with `passmcp verify --policy`; passmcp 0.0.2 refuses
  exactly the 25 under `refuse/` and accepts the 5 under `accept/`.
- **Editor schema support.** VS Code and other editors built on its JSON
  language service validate a document against the schema its `$schema`
  names. That covers `server.json`, which names one, and nothing else here,
  because the other artefacts have no published schema that says what
  passmcp or the MCP specification require.
