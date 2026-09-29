<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Architecture Decision Records

Decisions about this repository that will be questioned later, with the
reasoning that produced them. Records are immutable once merged; a
decision that changes gets a new record superseding the old one.

Decisions about the checks, the policy and attestation formats and the
version rule are passmcp's and passmcp-reporting's, in their own ADRs.
This directory records what is decided here.

| # | Decision | Status |
|---|---|---|
| [0001](0001-ask-passmcp-for-guidance.md) | Ask the installed passmcp for guidance; do not copy the catalogue | Accepted |
| [0002](0002-own-parser-and-schema-subset.md) | A position-keeping JSON parser and a fail-closed JSON Schema subset, no dependencies | Accepted |
| [0003](0003-attestations-verified-by-passmcp-reporting.md) | Attestations are verified by passmcp-reporting, not reimplemented | Accepted |
| [0004](0004-silent-on-unrecognised-json.md) | Unrecognised JSON gets no diagnostics, and editors send every JSON file | Accepted |
| [0005](0005-version-in-lockstep-with-passmcp.md) | The version is passmcp's latest release | Accepted |
