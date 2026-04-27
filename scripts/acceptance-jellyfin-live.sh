#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"

if [ -z "${MEDIARR_ACCEPTANCE_JELLYFIN_URL:-}" ]; then
  echo "Set MEDIARR_ACCEPTANCE_JELLYFIN_URL to your Jellyfin URL, for example http://nas:8096" >&2
  exit 2
fi

if [ -z "${MEDIARR_ACCEPTANCE_JELLYFIN_API_KEY:-}" ]; then
  echo "Set MEDIARR_ACCEPTANCE_JELLYFIN_API_KEY to a Jellyfin API key" >&2
  exit 2
fi

export MEDIARR_ACCEPTANCE_REPORT_DIR="${MEDIARR_ACCEPTANCE_REPORT_DIR:-$ROOT_DIR/acceptance-reports}"
RUNNER="${MEDIARR_ACCEPTANCE_RUNNER:-docker}"
mkdir -p "$MEDIARR_ACCEPTANCE_REPORT_DIR"

if [ "$RUNNER" = "go" ]; then
  cd "$ROOT_DIR/backend"
  go run ./cmd/mediarr-acceptance
  exit 0
fi

if [ "$RUNNER" != "docker" ]; then
  echo "MEDIARR_ACCEPTANCE_RUNNER must be docker or go" >&2
  exit 2
fi

cd "$ROOT_DIR"
docker compose build mediarr
docker compose run --rm --no-deps \
  --entrypoint /app/mediarr-acceptance \
  -e MEDIARR_ACCEPTANCE_JELLYFIN_URL \
  -e MEDIARR_ACCEPTANCE_JELLYFIN_API_KEY \
  -e MEDIARR_ACCEPTANCE_PATH_MAPS \
  -e MEDIARR_ACCEPTANCE_REDACT_TITLES \
  -e MEDIARR_ACCEPTANCE_REQUIRE_LOCAL_VERIFY \
  -e MEDIARR_ACCEPTANCE_TIMEOUT \
  -e MEDIARR_ACCEPTANCE_REPORT_DIR=/reports \
  -v "$MEDIARR_ACCEPTANCE_REPORT_DIR:/reports" \
  mediarr
