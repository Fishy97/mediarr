# Release Checklist

Use this checklist before tagging a public Mediarr release.

- `make ci` passes locally
- `go test ./...` passes in `backend`
- `go vet ./...` passes in `backend`
- `npm --prefix frontend run test -- --run` passes
- `npm --prefix frontend run build` passes
- `scripts/verify-no-delete.sh` passes
- `docker compose config --quiet` passes
- `docker compose --profile ai config --quiet` passes
- `docker compose build mediarr` passes
- `docker compose up -d mediarr` starts the no-AI service
- `/api/v1/health` returns `status: ok`
- `/api/v1/setup/status` returns setup state
- unauthenticated private API routes return `401`
- no endpoint permanently deletes media files
- CI has a dedicated no-delete safety job
- recommendation evidence, protect, and accept-manual routes work
- integration diagnostics route and in-app ingestion proof panel work after media-server sync
- path mapping unmapped and verify routes work
- support bundle creation, listing, and download work from the UI and API, and redaction/path-safety tests prove provider/media-server API keys are excluded
- background job cancel, retry, and stale recovery routes work
- `scripts/acceptance-jellyfin-live.sh` builds and documents the opt-in live Jellyfin acceptance workflow
- GHCR image workflow is present for tagged releases
- tagged GHCR images generate GitHub artifact provenance attestations with `actions/attest`
- `docker compose --profile ai up -d` starts the optional Ollama sidecar and model init service
- README and deployment docs include both launch modes:

```bash
# No AI
docker compose up -d

# With AI sidecar
docker compose --profile ai up -d
```

- repository contains `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`, Docker Compose guide, provider guide, AI guide, and REST API docs
- `/config` backup create/list/download, restore dry-run, confirmed restore, and support bundle create/list/download work from the UI
- media mounts are read-only in the default Compose file
- release notes and `docs/threat-model.md` are updated for the release
