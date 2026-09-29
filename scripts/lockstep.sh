#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# The version rule: every repository in the passmcp family carries passmcp's
# version. This one reads passmcp's formats and asks passmcp for its guidance,
# and its CI checks both against passmcp at that tag, so its version is
# passmcp's latest release, exactly.
#
#   scripts/lockstep.sh
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

mine=$(grep -Eo '^## \[[0-9]+\.[0-9]+\.[0-9]+\]' CHANGELOG.md | head -1 | tr -d '#[] ')
[ -n "$mine" ] || { echo "lockstep: CHANGELOG.md has no released version heading" >&2; exit 1; }

api="https://api.github.com/repos/sebastienrousseau/passmcp/releases/latest"
auth=()
[ -n "${GITHUB_TOKEN:-}" ] && auth=(-H "Authorization: Bearer ${GITHUB_TOKEN}")
# Fetched to a file and parsed from it, never piped into an interpreter:
# that shape reads as download-then-run to a supply-chain scanner.
release=$(mktemp)
trap 'rm -f "$release"' EXIT
curl -fsSL "${auth[@]}" -H "Accept: application/vnd.github+json" "$api" -o "$release"
theirs=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["tag_name"].lstrip("v"))' "$release")

if [ "$mine" = "$theirs" ]; then
  echo "lockstep: $mine matches passmcp's latest release"
else
  echo "lockstep: this repository is at $mine and passmcp's latest release is $theirs; release passmcp-lsp at $theirs" >&2
  exit 1
fi
