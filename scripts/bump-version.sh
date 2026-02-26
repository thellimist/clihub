#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$ROOT/VERSION"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/bump-version.sh [patch|minor|major]
  bash scripts/bump-version.sh --verify
EOF
}

read_version() {
  if [[ -f "$VERSION_FILE" ]]; then
    tr -d '[:space:]' < "$VERSION_FILE"
  else
    echo "0.1.0"
  fi
}

validate_version() {
  local v="$1"
  if [[ ! "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Invalid VERSION value: $v (expected MAJOR.MINOR.PATCH)" >&2
    exit 1
  fi
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

current="$(read_version)"
validate_version "$current"

if [[ "${1:-}" == "--verify" ]]; then
  echo "VERSION is valid: $current"
  exit 0
fi

part="${1:-patch}"
IFS='.' read -r major minor patch <<< "$current"

case "$part" in
  patch)
    patch=$((patch + 1))
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  *)
    usage
    exit 1
    ;;
esac

next="${major}.${minor}.${patch}"
printf '%s\n' "$next" > "$VERSION_FILE"
echo "VERSION bumped: $current -> $next"
