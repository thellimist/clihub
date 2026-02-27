#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$ROOT/VERSION"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/bump-version.sh [patch|minor|major]
  bash scripts/bump-version.sh --verify
  bash scripts/bump-version.sh --prepare-push [patch|minor|major]
  bash scripts/bump-version.sh --sync-tag
EOF
}

read_version() {
  if [[ -f "$VERSION_FILE" ]]; then
    tr -d '[:space:]' < "$VERSION_FILE"
  else
    echo "0.0.0"
  fi
}

validate_version() {
  local v="$1"
  if [[ ! "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Invalid VERSION value: $v (expected MAJOR.MINOR.PATCH)" >&2
    exit 1
  fi
}

require_git_repo() {
  if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
    echo "Not inside a git repository." >&2
    exit 1
  fi
}

ensure_matching_tag() {
  local version="$1"
  local target_sha="$2"
  local tag="v$version"
  local tag_obj_type
  local tag_target_sha

  tag_obj_type=$(git for-each-ref "refs/tags/$tag" --format='%(objecttype)' | head -1 || true)
  if [[ -n "$tag_obj_type" ]]; then
    tag_target_sha=$(git rev-list -n 1 "$tag" 2>/dev/null || true)
    if [[ "$tag_obj_type" == "tag" && "$tag_target_sha" == "$target_sha" ]]; then
      echo "Annotated tag already present: $tag -> $target_sha"
      return 0
    fi
    git tag -d "$tag" >/dev/null
    echo "Replaced existing tag $tag"
  fi

  git tag -a "$tag" "$target_sha" -m "$tag"
  echo "Annotated tag created: $tag -> $target_sha"
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

if [[ "${1:-}" == "--sync-tag" ]]; then
  require_git_repo
  if ! git rev-parse --verify HEAD >/dev/null 2>&1; then
    echo "No commits found (HEAD is missing)." >&2
    exit 1
  fi
  head_sha=$(git rev-parse HEAD)
  ensure_matching_tag "$current" "$head_sha"
  echo "Done. Push with: git push --follow-tags origin main"
  exit 0
fi

prepare_push=0
part="${1:-patch}"
if [[ "${1:-}" == "--prepare-push" ]]; then
  prepare_push=1
  part="${2:-patch}"
fi
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
echo "Next tag: v$next"

if [[ "$prepare_push" == "1" ]]; then
  require_git_repo
  if ! git rev-parse --verify HEAD >/dev/null 2>&1; then
    echo "No commits found (HEAD is missing)." >&2
    exit 1
  fi

  staged_other=$(git diff --cached --name-only --diff-filter=ACMRTUXB | grep -v '^VERSION$' || true)
  if [[ -n "$staged_other" ]]; then
    echo "Cannot run --prepare-push: staged changes exist outside VERSION:" >&2
    echo "$staged_other" >&2
    echo "Commit/stash those first, then rerun." >&2
    exit 1
  fi

  git add VERSION
  git commit --amend --no-edit

  head_sha=$(git rev-parse HEAD)
  ensure_matching_tag "$next" "$head_sha"
  echo "Done. Push with: git push --follow-tags origin main"
fi
