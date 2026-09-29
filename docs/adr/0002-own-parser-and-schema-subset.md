<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0002 — A position-keeping JSON parser and a fail-closed JSON Schema subset

**Status:** Accepted · **Decided:** 2026-09-29

## Context

A diagnostic is useful only when it underlines the text that caused it.
`encoding/json` decodes values and forgets where they were. Client
configurations are also JSON with comments and trailing commas, which
`encoding/json` refuses.

`server.json` is validated against the MCP Registry's schema, a draft-07
document. General JSON Schema libraries exist, but they report locations
as JSON Pointers into a decoded value, which then have to be mapped back to
text, and they bring a dependency tree to a module that otherwise has one
dependency.

The registry schema (revision 2025-12-11) uses fifteen assertion and
applicator keywords: `$ref`, `additionalProperties`, `allOf`, `anyOf`,
`const`, `enum`, `format` (only `uri`), `items`, `maxLength`, `minLength`,
`not`, `pattern`, `properties`, `required` and `type`.

## Decision

- `internal/jsondoc` parses JSON into a tree where every value, and every
  key, carries its byte range. Strings are decoded by `encoding/json`
  itself, and a differential fuzz target holds the strict dialect to
  `encoding/json`'s verdict on every input. A comment dialect adds
  comments and trailing commas. Nesting is capped at 256 levels.
- `internal/schema` implements exactly those fifteen keywords, with local
  references, and treats the rest of draft-07's annotations as
  annotations. `Compile` refuses a schema that uses any other keyword,
  naming it, so a new keyword in a future registry schema fails a test
  rather than being skipped.

## Consequences

- Every diagnostic has a precise range, and no dependency is added.
- A registry schema that adopts a new keyword needs this validator
  extended before it can be embedded; `TestEmbeddedSchemaIsTheRegistrys`
  makes that impossible to miss.
- The validator is not a general JSON Schema implementation and is not
  offered as one. It was cross-checked against `check-jsonschema` on five
  listings, which agreed on all five
  ([COMPARISON.md](../COMPARISON.md)).
