#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ ! -d "$ROOT/.git" ]]; then
  echo "Not a git repository: $ROOT" >&2
  exit 1
fi

chmod +x "$ROOT/.githooks/pre-push" "$ROOT/scripts/pre-push-checks.sh" "$ROOT/scripts/bump-version.sh"
git -C "$ROOT" config --local core.hooksPath .githooks

echo "Installed repo hooks (core.hooksPath=.githooks)"
echo "Active pre-push hook: $ROOT/.githooks/pre-push"
