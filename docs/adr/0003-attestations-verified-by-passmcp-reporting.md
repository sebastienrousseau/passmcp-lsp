<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0003 — Attestations are verified by passmcp-reporting, not reimplemented

**Status:** Accepted · **Decided:** 2026-09-29

## Context

passmcp-reporting is the family's Apache-2.0 module for the attestation
format and its offline verifier. Its `attestation.Parse` and `a2a.Parse`
check the envelope, the predicate type, that the subject's digest still
covers the target the predicate names, the verdicts and the counts. That
is the definition of a valid attestation; anything this repository wrote
instead would be a second definition.

## Decision

An attestation is verified by calling passmcp-reporting's verifier for its
predicate type. This repository adds only placement: the verifier's
problems, which it joins with `"; "`, are split and each is placed on the
key it mentions, or on the first character of the document when none
matches. `go.mod` requires passmcp-reporting at the family's version, and
`make versions` fails when it does not.

## Consequences

- An attestation is valid here exactly when it is valid to every other
  consumer of passmcp-reporting.
- A new predicate type needs passmcp-reporting to verify it first; until
  then it is a `attestation/predicate-type` warning.
- If the verifier changes how it joins its problems, the placement falls
  back to one diagnostic at the start of the document, and still reports
  every problem.
