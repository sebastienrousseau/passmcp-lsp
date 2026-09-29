#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# Fail unless every place that names the version being released names it:
# the CHANGELOG heading, the release notes file, the install lines in the
# READMEs and docs, the README's ecosystem sentence, CITATION.cff, the VS
# Code extension's manifest and lock, and the passmcp-reporting release
# go.mod requires. The family moves in lockstep, so every one of them is
# the same version.
#
#   scripts/verify-release-versions.sh vX.Y.Z
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
tag="${1:-${GITHUB_REF_NAME:-}}"
[ -n "$tag" ] || { echo "usage: $0 vX.Y.Z" >&2; exit 2; }
ver="${tag#v}"
fail=0

grep -Eq "^## \[$ver\]" CHANGELOG.md || { echo "CHANGELOG.md has no '## [$ver]' heading" >&2; fail=1; }
[ -f "docs/releases/v$ver.md" ] || { echo "docs/releases/v$ver.md does not exist" >&2; fail=1; }

# Every go install line for a family module; at least one must be there, so
# a path that stops matching cannot pass silently.
installs=$(grep -Eoh 'satellion\.com/passmcp(-lsp)?/cmd/passmcp(-lsp)?@v[0-9]+\.[0-9]+\.[0-9]+' README.md docs/*.md editors/vscode/README.md || true)
if [ -z "$installs" ]; then
  echo "no go install line found to check" >&2; fail=1
elif grep -v "@v$ver\$" <<<"$installs"; then
  echo "the docs pin a go install version other than $ver" >&2; fail=1
fi

if ! grep -Fq "Every component is released at **$ver**" README.md; then
  echo "README.md's ecosystem section does not state $ver" >&2; fail=1
fi
if ! grep -Eq "^version: \"?$ver\"?\$" CITATION.cff; then
  echo "CITATION.cff does not say version $ver" >&2; fail=1
fi
if ! grep -Eq "satellion\.com/passmcp-reporting v$ver\$" go.mod; then
  echo "go.mod does not require satellion.com/passmcp-reporting v$ver" >&2; fail=1
fi

# The extension's manifest and its lock's root package.
ext=$(python3 - "$ver" <<'PYEOF'
import json, sys
ver = sys.argv[1]
bad = []
m = json.load(open("editors/vscode/package.json"))
if m.get("version") != ver:
    bad.append("package.json is %r" % m.get("version"))
lock = json.load(open("editors/vscode/package-lock.json"))
for where, v in (("version", lock.get("version")), ('packages[""].version', lock.get("packages", {}).get("", {}).get("version"))):
    if v != ver:
        bad.append("package-lock.json %s is %r" % (where, v))
print("; ".join(bad))
PYEOF
)
if [ -n "$ext" ]; then
  echo "the VS Code extension disagrees with $ver: $ext" >&2; fail=1
fi

[ "$fail" -eq 0 ] && echo "release versions agree on $ver"
exit "$fail"
