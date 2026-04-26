#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

deny_patterns=(
  'os\.Remove(All)?\('
  'syscall\.Unlink'
  'unix\.Unlink'
  '/api/v1/media/.*/delete'
  'rm -rf .*/media'
)

for pattern in "${deny_patterns[@]}"; do
  if rg -n --glob '!frontend/node_modules/**' --glob '!frontend/dist/**' --glob '!*verify-no-delete.sh' "$pattern" .; then
    echo "No-delete invariant failed: found forbidden pattern '$pattern'." >&2
    exit 1
  fi
done

if rg -n --glob '!**/*_test.go' 'http\.MethodDelete.*media' backend; then
  echo "No-delete invariant failed: production media DELETE handler detected." >&2
  exit 1
fi

required_mounts=(
  '/media/movies:ro'
  '/media/series:ro'
  '/media/anime:ro'
)

for mount in "${required_mounts[@]}"; do
  if ! rg -n --fixed-strings "$mount" docker-compose.yml >/dev/null; then
    echo "No-delete invariant failed: docker-compose.yml must mount $mount." >&2
    exit 1
  fi
done

if ! rg -n --fixed-strings 'server.mux.HandleFunc("/api/v1/media/files/", methodNotAllowed)' backend/internal/api/server.go >/dev/null; then
  echo "No-delete invariant failed: media file routes must remain explicitly non-destructive." >&2
  exit 1
fi

echo "No-delete invariant verified."
