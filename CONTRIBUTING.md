<!-- SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com> -->
<!-- SPDX-License-Identifier: Apache-2.0 -->

# Contributing

passmcp-lsp tells people what is wrong with files they are about to trust:
a registry listing, a client configuration that starts programs, a policy
that gates a deployment. A diagnostic that is wrong in either direction
costs someone a decision, so every rule here needs a source and a test.

## Getting started

1. Fork and clone the repository.
2. Install **Go** at the version the `go` directive in `go.mod` names and
   **make**; install [passmcp](https://github.com/sebastienrousseau/passmcp)
   to try hover and to run `make crosscheck`; install **Node 22** only to
   work on the VS Code extension.
3. Create a branch from `main` and open the pull request against `main`.
   Every workflow filters on `pull_request: branches: [main]`, so a PR
   aimed elsewhere runs no CI.
4. Make the change.
5. Verify:

   ```bash
   make            # format, vet, lint, headers, tests, stdio smoke test
   make test-race
   ```

## Commits

**Sign your commits cryptographically and add a DCO sign-off trailer.**
Both are required and both are enforced: signing proves who authored the
commit; the sign-off (`git commit -s`) certifies the
[Developer Certificate of Origin](https://developercertificate.org).
Merge commits are exempt from the DCO check.

Use [Conventional Commits](https://www.conventionalcommits.org/) with an
imperative subject: `feat(check): flag a tool name over 128 characters`,
not `Added check`.

## What a change needs

- **A source for every rule.** A diagnostic cites what it enforces: the
  registry schema, the MCP specification revision, passmcp's own behaviour,
  or passmcp-reporting's verifier. A rule that is only an opinion is a
  hint at most, and says so.
- **A test that fails without it**, and for a policy rule a fixture under
  `internal/check/testdata/policy/` that `make crosscheck` agrees with.
- **85% statement coverage in every package.**
- **A new diagnostic code in [`docs/diagnostics.md`](docs/diagnostics.md).**
  Codes are part of the stability guarantee.
- **An entry in `CHANGELOG.md`.**
- **A reason in the commit for any new dependency.** The module has one.

## Pull request checklist

- [ ] `make` and `make test-race` pass
- [ ] The change is covered by a test that fails without it
- [ ] `CHANGELOG.md` has an entry
- [ ] Commits are signed and carry a DCO sign-off
