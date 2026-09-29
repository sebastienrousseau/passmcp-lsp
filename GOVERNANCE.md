<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Governance

passmcp-lsp is one repository in the passmcp family and is governed the way
passmcp is. This document says what is specific to this repository and points
at passmcp's for the rest.

## Roles

**Maintainer:** Sebastien Rousseau (<sebastian.rousseau@gmail.com>,
GitHub `@sebastienrousseau`), with commit access, responsible for the
diagnostics, the editor extensions, the releases and the security response.

**Contributor:** anyone who opens an issue or a pull request. Mechanics
are in [CONTRIBUTING.md](CONTRIBUTING.md).

## What is decided here, and what is not

This repository decides **how the artefacts are presented while they are
edited**: which files are recognised, the diagnostic codes and their
severities, the hover, the command line and the editor extensions. A
decision that will be questioned later is recorded in
[docs/adr/](docs/adr/README.md).

It does not decide **the formats**. The policy and attestation formats are
passmcp's and passmcp-reporting's, the tool rules are the MCP
specification's, and `server.json` is the MCP Registry's. When this
repository and its source disagree, the source is right and this repository
changes.

## Version and release

The version is passmcp's. This repository never chooses its own; see the
lockstep rule in [CHANGELOG.md](CHANGELOG.md). Releases are signed tags cut
by the Maintainer. Publishing the VS Code extension to a marketplace is the
Maintainer's decision and is done by hand.

## Continuity

The single-Maintainer model is a real bus-factor risk, stated rather than
hidden. The succession procedure — hand-off, community fork after six
months of unresponsiveness, and compromise response — is passmcp's, in
[passmcp's GOVERNANCE.md](https://github.com/sebastienrousseau/passmcp/blob/main/GOVERNANCE.md),
and applies to this repository as one of the family. Apache-2.0 lets anyone
fork without further permission.

The family manifest names a kill criterion for this repository: if the
guidance hover goes unused, it is archived.

## Changes to this document

Through the usual pull request process, with a week's notice.
