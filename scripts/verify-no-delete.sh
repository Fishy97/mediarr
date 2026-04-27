#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

search_regex() {
  local pattern="$1"
  if command -v rg >/dev/null 2>&1; then
    rg -n \
      --glob '!frontend/node_modules/**' \
      --glob '!frontend/dist/**' \
      --glob '!config/**' \
      --glob '!acceptance-reports/**' \
      --glob '!.superpowers/**' \
      --glob '!tmp/**' \
      --glob '!*verify-no-delete.sh' \
      "$pattern" .
  else
    grep -RInE \
      --exclude-dir=.git \
      --exclude-dir=node_modules \
      --exclude-dir=dist \
      --exclude-dir=config \
      --exclude-dir=acceptance-reports \
      --exclude-dir=.superpowers \
      --exclude-dir=tmp \
      --exclude=verify-no-delete.sh \
      "$pattern" .
  fi
}

search_backend_regex_without_tests() {
  local pattern="$1"
  if command -v rg >/dev/null 2>&1; then
    rg -n --glob '!**/*_test.go' "$pattern" backend
  else
    grep -RInE --include='*.go' --exclude='*_test.go' "$pattern" backend
  fi
}

file_contains_fixed() {
  local needle="$1"
  local file="$2"
  if command -v rg >/dev/null 2>&1; then
    rg -n --fixed-strings "$needle" "$file" >/dev/null
  else
    grep -nF "$needle" "$file" >/dev/null
  fi
}

deny_patterns=(
  'os\.Remove(All)?\('
  'syscall\.Unlink'
  'unix\.Unlink'
  '/api/v1/media/.*/delete'
  'rm -rf .*/media'
)

for pattern in "${deny_patterns[@]}"; do
  if search_regex "$pattern"; then
    echo "No-delete invariant failed: found forbidden pattern '$pattern'." >&2
    exit 1
  fi
done

if search_backend_regex_without_tests 'http\.MethodDelete.*media'; then
  echo "No-delete invariant failed: production media DELETE handler detected." >&2
  exit 1
fi

required_mounts=(
  '/media/movies:ro'
  '/media/series:ro'
  '/media/anime:ro'
)

for mount in "${required_mounts[@]}"; do
  if ! file_contains_fixed "$mount" docker-compose.yml; then
    echo "No-delete invariant failed: docker-compose.yml must mount $mount." >&2
    exit 1
  fi
done

if ! file_contains_fixed 'server.mux.HandleFunc("/api/v1/media/files/", methodNotAllowed)' backend/internal/api/server.go; then
  echo "No-delete invariant failed: media file routes must remain explicitly non-destructive." >&2
  exit 1
fi

echo "No-delete invariant verified."
