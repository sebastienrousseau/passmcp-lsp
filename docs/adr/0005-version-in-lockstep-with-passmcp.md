<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# 0005 — The version is passmcp's latest release

**Status:** Accepted · **Decided:** 2026-09-29, by the owner, for the
whole family

## Context

passmcp's ecosystem map records that passmcp-lsp was first planned outside
the version lockstep, because editor marketplaces keep their own cadence.
On 29 Sep 2026 the owner decided that every repository in the family
carries one version, this one included, so that nobody has to ask which
version of which piece they are looking at.

## Decision

The version is passmcp's latest release. It lives in the newest
`## [x.y.z]` heading of `CHANGELOG.md`; `make lockstep` compares it with
passmcp's latest release, and `make versions` checks every other file that
names it: install lines, the README's ecosystem sentence, `CITATION.cff`,
the VS Code extension's `package.json` and lock, and the passmcp-reporting
release `go.mod` requires. The first version is 0.0.2: passmcp-lsp was
never tagged at 0.0.1, and a family member that has never been tagged
first ships at the family's current version.

## Consequences

- A fix to the extension alone waits for, or causes, a family release.
  That cost was accepted when the rule was made.
- The extension's marketplace version, when it is published, is the
  family's version.
