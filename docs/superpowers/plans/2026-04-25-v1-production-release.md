# Mediarr V1 Production Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete Mediarr V1 as a production-grade, open-source, Docker Compose-first media library manager for already-downloaded movies, series, and anime.

**Architecture:** Keep the modular Go monolith serving a React UI, backed by SQLite in `/config`. Add real setup/auth, provider and integration clients behind typed interfaces, resilient scan/enrichment jobs, optional local AI through an Ollama sidecar with model initialization, and release-grade Docker/docs/CI.

**Tech Stack:** Go 1.23, SQLite via `modernc.org/sqlite`, React/TypeScript/Vite, Docker Compose, Ollama optional sidecar, GitHub Actions.

---

## File Map

- `backend/internal/auth`: session/API-token authentication, password hashing, auth middleware.
- `backend/internal/database`: migrations and persistence for users, sessions, settings, provider cache, metadata matches, scan jobs, recommendations, backups.
- `backend/internal/api`: REST routes for setup, auth, settings, providers, integrations, scans, catalog, recommendations, backups, and AI status.
- `backend/internal/config`: typed runtime config for provider keys, AI model, paths, and safety defaults.
- `backend/internal/filescan`, `backend/internal/catalog`, `backend/internal/probe`: resumable scanning, stronger parsing, stale-file cleanup, ffprobe metadata.
- `backend/internal/metadata`: TMDb, AniList, TheTVDB, OpenSubtitles, local sidecar adapters with cache/rate-limit handling.
- `backend/internal/integrations`: Jellyfin, Plex, and Emby clients as sync targets.
- `backend/internal/ai`: Ollama health and advisory JSON suggestion client.
- `backend/internal/recommendations`: deterministic recommendation rules plus optional AI rationale attachment.
- `frontend/src`: setup/login, dashboard, libraries, scan status, catalog detail, recommendation queue, provider/integration settings, backups/restore.
- `docker-compose.yml`, `.env.example`, `Dockerfile`, `.github/workflows/ci.yml`: V1 deployment and validation.
- `docs`: Docker Compose guide, provider guide, AI guide, release checklist, API docs, security docs.

## Commit Track 1: Release Packaging And Auth

- [x] Write failing Go tests for admin setup, password login, session validation, bearer-token fallback, and unauthenticated API rejection.
- [x] Add users, sessions, settings, and API token tables with idempotent migrations.
- [x] Implement bcrypt password hashing, session token generation, token hashing-at-rest, setup status, admin creation, login, logout, and `me` routes.
- [x] Protect all mutating and private API routes while keeping health/setup/login public.
- [x] Add frontend first-run setup and login screens.
- [x] Add Docker AI model-init service, `MEDIARR_AI_MODEL`, pinned defaults where feasible, and docs for no-AI vs AI startup.
- [x] Run `go test ./...`, `go vet ./...`, `npm --prefix frontend run build`, `npm --prefix frontend run test -- --run`, `docker compose config --quiet`, `docker compose --profile ai config --quiet`, `docker compose build mediarr`.
- [ ] Commit as `feat: add v1 setup auth and ai packaging` and push `codex/v1-production-release`.

## Commit Track 2: Scanner, Catalog, And Metadata Providers

- [ ] Write failing tests for stale catalog removal, multi-episode parsing, anime absolute episodes, edition detection, provider cache expiry, and provider health statuses.
- [ ] Store scan jobs with resumable status and stale file pruning by library.
- [ ] Improve parsing for editions, specials, multi-episode series, anime groups, quality/source/codec tags, and unknown confidence.
- [ ] Add real provider clients for TMDb, AniList, TheTVDB, OpenSubtitles, and local sidecars behind a common interface.
- [ ] Add provider config storage, cache with expiry, `Retry-After` handling, attribution, and health surfaces.
- [ ] Add metadata match persistence with confidence, provider IDs, candidate lists, and user override precedence.
- [ ] Add frontend provider settings and catalog metadata correction flow.
- [ ] Run the full validation suite and commit as `feat: add v1 scanner metadata providers`.

## Commit Track 3: Recommendations, AI, And Integrations

- [ ] Write failing tests for non-destructive recommendation invariants, duplicate scoring, oversized scoring, missing subtitles, AI JSON validation, and integration connection checks.
- [ ] Expand recommendations with rule source, severity, affected paths, expected savings, keep/remove candidates, ignore/restore workflow, and audit events.
- [ ] Implement Ollama health, model availability checks, advisory suggestion prompts, strict JSON decoding, timeout handling, and confidence scoring.
- [ ] Attach AI explanations only to deterministic recommendations and never allow AI to alter catalog truth automatically.
- [ ] Implement Jellyfin, Plex, and Emby connection tests and sync-target refresh actions with typed clients and audit results.
- [ ] Add frontend recommendation action screens, AI status, and integration configuration screens.
- [ ] Run the full validation suite and commit as `feat: add v1 recommendations ai integrations`.

## Commit Track 4: Backups, Release Hardening, And Public Docs

- [ ] Write failing tests for backup contents, restore validation, no-delete endpoints, API auth regression, Docker smoke, and UI setup flow.
- [ ] Add restore dry-run, restore execution with pre-restore backup, backup manifest, and config inclusion.
- [ ] Add real tiny media fixtures for ffprobe validation and integration fixture libraries.
- [ ] Harden Docker runtime with read-only media mounts, `/config` ownership guidance, healthchecks, labels, and release image docs.
- [ ] Expand CI with backend, frontend, Docker, Compose no-AI, Compose AI config, smoke test, and security/no-delete regression.
- [ ] Update README, deployment guide, provider guide, AI guide, security policy, contributing guide, release checklist, and API docs.
- [ ] Verify GitHub branch CI, then merge/push to `main` after V1 validation is green.

## V1 Definition Of Done

- [ ] No README language describes V1 features as scaffolded.
- [ ] No configured provider/integration route returns placeholder-only data.
- [ ] No endpoint can permanently delete media.
- [ ] No AI output is accepted as catalog truth without user approval.
- [ ] Docker Compose no-AI and AI modes both validate.
- [ ] The app can be installed on a clean Ubuntu Docker host from documented commands.
- [ ] CI is green on the release branch and `main`.
- [ ] Repository is ready to make public, with AGPL license, security policy, and contributor docs.
