# Release Checklist

Use this checklist before tagging a public Mediarr release.

- `go test ./...` passes in `backend`
- `go vet ./...` passes in `backend`
- `npm --prefix frontend run test -- --run` passes
- `npm --prefix frontend run build` passes
- `docker compose config --quiet` passes
- `docker compose --profile ai config --quiet` passes
- `docker compose build mediarr` passes
- `docker compose up -d mediarr` starts the no-AI service
- `/api/v1/health` returns `status: ok`
- `/api/v1/setup/status` returns setup state
- unauthenticated private API routes return `401`
- no endpoint permanently deletes media files
- recommendation evidence, protect, and accept-manual routes work
- path mapping unmapped and verify routes work
- background job cancel, retry, and stale recovery routes work
- GHCR image workflow is present for tagged releases
- `docker compose --profile ai up -d` starts the optional Ollama sidecar and model init service
- README and deployment docs include both launch modes:

```bash
# No AI
docker compose up -d

# With AI sidecar
docker compose --profile ai up -d
```

- repository contains `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`, Docker Compose guide, provider guide, AI guide, and REST API docs
- `/config` backup creation and restore dry-run work from the UI
- media mounts are read-only in the default Compose file
