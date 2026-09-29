#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
# SPDX-License-Identifier: Apache-2.0
#
# Every repository in the passmcp family verifies its own row against the
# manifest passmcp publishes, so the map and the territory cannot drift. This
# checks the facts the row states about this repository — its licence,
# language and lockstep — against what the working tree actually is.
#
#   scripts/family.sh
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

manifest="${PASSMCP_ECOSYSTEM_URL:-https://raw.githubusercontent.com/sebastienrousseau/passmcp/main/ecosystem.json}"
# Fetched to a file and parsed from it, never piped into an interpreter:
# that shape reads as download-then-run to a supply-chain scanner.
mf=$(mktemp)
trap 'rm -f "$mf"' EXIT
curl -fsSL "$manifest" -o "$mf"
row=$(python3 - "$mf" <<'PYEOF'
import json, sys
m = json.load(open(sys.argv[1]))
# Schema 1 names a row by "name" and calls statuses shipping and planned;
# schema 2 adds "repository" and calls them released and unreleased. passmcp's
# main serves schema 1 until the release that moves it, so both are read, and
# a schema this script has never seen is refused rather than guessed at.
schema = m.get("schema_version", 1)
if schema not in (1, 2):
    sys.exit("family: the manifest is schema_version %r; this script reads 1 and 2" % schema)
rows = [r for r in m["repositories"] if r.get("repository", r["name"]) == "passmcp-lsp"]
if not rows:
    sys.exit("family: passmcp-lsp has no row in the family manifest")
r = rows[0]
status = {"shipping": "released", "planned": "unreleased"}.get(r["status"], r["status"])
print(status, r["license"], r["language"], str(r["lockstep"]).lower())
PYEOF
)
read -r status licence language lockstep <<<"$row"

fail=0
# LICENSES/ holds one file per licence in use, named by SPDX identifier; the
# project's own licence is one of them (the embedded registry schema brings
# its upstream's too).
if [ ! -f "LICENSES/${licence}.txt" ]; then
  echo "family: the manifest says $licence and LICENSES/ has no ${licence}.txt" >&2; fail=1
fi
if [ "$language" != "go" ] || [ ! -f go.mod ]; then
  echo "family: the manifest says $language and this is a Go module" >&2; fail=1
fi
if [ "$lockstep" != "true" ] || [ ! -x scripts/lockstep.sh ]; then
  echo "family: the manifest says lockstep=$lockstep and this repository carries passmcp's version" >&2; fail=1
fi
if [ "$status" = "rejected" ]; then
  echo "family: the manifest lists passmcp-lsp as rejected, and this repository exists" >&2; fail=1
elif [ "$status" != "released" ]; then
  # The row moves to released in the passmcp release after this repository's
  # first tag; until then the facts above are what can be checked.
  echo "::warning::family: the manifest lists passmcp-lsp as $status"
fi
[ "$fail" -eq 0 ] && echo "family: the manifest's row for passmcp-lsp is true of this tree ($licence, $language, lockstep=$lockstep, $status)"
exit "$fail"
