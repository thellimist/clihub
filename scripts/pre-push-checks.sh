#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "[pre-push] gofmt check"
if [[ "${CHECK_ALL_GO_FILES:-1}" == "0" ]]; then
  go_files="${CHANGED_GO_FILES:-}"
else
  go_files="$(git ls-files "*.go")"
fi

if [[ -n "$go_files" ]]; then
  unformatted=$(printf "%s\n" "$go_files" | awk 'NF > 0' | xargs gofmt -l)
  if [[ -n "$unformatted" ]]; then
    echo "Unformatted Go files detected:"
    echo "$unformatted"
    echo "Run: gofmt -w <files>"
    exit 1
  fi
fi

echo "[pre-push] go vet ./..."
go vet ./...

echo "[pre-push] go test ./..."
go test ./...

if command -v golangci-lint >/dev/null 2>&1; then
  echo "[pre-push] golangci-lint run"
  golangci-lint run
fi
