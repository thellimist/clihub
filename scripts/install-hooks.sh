#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ ! -d "$ROOT/.git" ]]; then
  echo "Not a git repository: $ROOT" >&2
  exit 1
fi

chmod +x "$ROOT/.githooks/pre-push" "$ROOT/scripts/pre-push-checks.sh" "$ROOT/scripts/bump-version.sh"
git -C "$ROOT" config --local core.hooksPath .githooks
git -C "$ROOT" config --local push.followTags true

echo "Installed repo hooks (core.hooksPath=.githooks)"
echo "Active pre-push hook: $ROOT/.githooks/pre-push"
echo "Enabled git push.followTags=true (annotated tags will be pushed automatically)"
