# Stewardship Advantage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add read-only request/analytics signals, verified collection publishing, campaign templates, notifications, what-if simulation, storage ledger, and protection requests without introducing destructive media actions.

**Architecture:** Keep the modular monolith pattern. Add focused backend domain packages for stewardship features, persist new normalized tables in SQLite, expose authenticated REST APIs, and add compact MediaGlass UI panels. External services are read-only except explicit collection publish, which writes only collections and is always previewable.

**Tech Stack:** Go, SQLite, REST API, React/TypeScript/Vite, Docker Compose, existing audit/job/auth infrastructure.

---

## File Structure

- Create `backend/internal/stewardship/types.go`: shared request signal, Tautulli enrichment, collection publication, notification, storage ledger, template, what-if, and protection request types.
- Create `backend/internal/stewardship/templates.go`: built-in campaign templates and creation helpers.
- Create `backend/internal/stewardship/ledger.go`: deterministic storage ledger and what-if calculations.
- Create `backend/internal/stewardship/notifications.go`: notification validation and webhook delivery helpers.
- Create `backend/internal/stewardship/seerr.go`: Seerr/Jellyseerr/Overseerr request client and normalizer.
- Create `backend/internal/stewardship/tautulli.go`: Tautulli history client and Plex activity enrichment.
- Create `backend/internal/stewardship/publication.go`: verified Leaving Soon publish planning and Plex/Jellyfin collection write clients.
- Modify `backend/internal/database/store.go`: migrations and persistence methods for new stewardship tables.
- Modify `backend/internal/integrations/integrations.go`: add Seerr/Tautulli targets to integration defaults.
- Modify `backend/internal/api/server.go`: route and handler wiring.
- Modify `frontend/src/types.ts`: add stewardship API types.
- Modify `frontend/src/lib/api.ts` and `frontend/src/lib/api.test.ts`: add client methods and tests.
- Modify `frontend/src/App.tsx` and `frontend/src/styles/app.css`: add visible UI surfaces for templates, ledger, what-if, notifications, and protection requests.
- Modify `README.md`, `docs/api/rest.md`, `docs/release-checklist.md`, and add `docs/release-notes/v1.6.0.md`.

## Task 1: Stewardship Domain Types And Tests

- [ ] **Step 1: Write failing stewardship domain tests**

Create tests in `backend/internal/stewardship/ledger_test.go`, `templates_test.go`, `seerr_test.go`, `tautulli_test.go`, `publication_test.go`, `notifications_test.go`, and `protection_test.go` covering:

- Seerr request normalization from `/api/v1/request` style payloads.
- Tautulli history rows enriching existing Plex rollups.
- templates producing editable campaigns.
- storage ledger separating verified, mapped estimate, server estimate, blocked, protected, accepted, and requested bytes.
- what-if simulation returning matched/suppressed/blocked totals without writes.
- collection publish planning blocking unverified or missing external ids.
- notification validation rejecting unsafe webhook schemes.
- protection request state transitions.

Run:

```bash
cd backend && go test ./internal/stewardship -count=1
```

Expected: fail because package/types are missing.

- [ ] **Step 2: Implement domain package**

Add the seven stewardship files listed above. Keep them dependency-light and deterministic. No database imports except where necessary in adapters; no API/server imports.

- [ ] **Step 3: Verify domain tests pass**

Run:

```bash
cd backend && go test ./internal/stewardship -count=1
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/stewardship
git commit -m "feat: add stewardship advantage domain"
```

## Task 2: Persistence

- [ ] **Step 1: Write failing database tests**

Add tests to `backend/internal/database/stewardship_test.go` for:

- upsert/list Seerr source settings with redacted API key responses.
- replace/list request signals.
- record/list Tautulli sync jobs.
- create/list/update collection publications.
- create/list/read notifications.
- create/approve/decline protection requests.
- storage ledger reads from recommendations, request signals, and protection requests.

Run:

```bash
cd backend && go test ./internal/database -run Stewardship -count=1
```

Expected: fail because methods and tables are missing.

- [ ] **Step 2: Add migrations and store methods**

Extend `migrate()` with new tables and implement focused store methods. Reuse existing secret-redaction helpers where possible and preserve prior data on sync failure by replacing source-specific rows only after successful fetch.

- [ ] **Step 3: Verify database tests pass**

Run:

```bash
cd backend && go test ./internal/database -run Stewardship -count=1
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/database
git commit -m "feat: persist stewardship advantage data"
```

## Task 3: API And Jobs

- [ ] **Step 1: Write failing API tests**

Add tests to `backend/internal/api/stewardship_test.go` for:

- request source settings and sync.
- Tautulli sync.
- campaign templates listing and creation.
- campaign what-if.
- publish preview and publish request validation.
- storage ledger endpoint.
- notifications list/read.
- protection request create/approve/decline.

Run:

```bash
cd backend && go test ./internal/api -run Stewardship -count=1
```

Expected: fail because routes are missing.

- [ ] **Step 2: Wire handlers**

Add authenticated `/api/v1` routes. Sync handlers use short-lived contexts, bounded pagination, existing audit logging, and persisted notifications for important outcomes. Do not add any delete, unmonitor, or request-clear route.

- [ ] **Step 3: Verify API tests pass**

Run:

```bash
cd backend && go test ./internal/api -run Stewardship -count=1
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/api backend/internal/integrations
git commit -m "feat: expose stewardship advantage api"
```

## Task 4: Frontend API And UI

- [ ] **Step 1: Write failing frontend API tests**

Extend `frontend/src/lib/api.test.ts` to assert client methods call:

- `/api/v1/request-sources`
- `/api/v1/request-sources/seerr/sync`
- `/api/v1/integrations/tautulli/sync`
- `/api/v1/campaign-templates`
- `/api/v1/campaigns/{id}/what-if`
- `/api/v1/campaigns/{id}/publish-preview`
- `/api/v1/storage-ledger`
- `/api/v1/notifications`
- `/api/v1/protection-requests`

Run:

```bash
npm --prefix frontend run test -- --run src/lib/api.test.ts
```

Expected: fail because types/client methods are missing.

- [ ] **Step 2: Add frontend types and API methods**

Extend `frontend/src/types.ts` and `frontend/src/lib/api.ts`.

- [ ] **Step 3: Add UI panels**

Modify `frontend/src/App.tsx` and `frontend/src/styles/app.css`:

- Dashboard storage ledger card.
- Campaign template gallery and what-if/publish preview controls.
- Integrations cards for Seerr and Tautulli.
- Notifications list.
- Protection request queue.

- [ ] **Step 4: Verify frontend tests/build**

Run:

```bash
npm --prefix frontend run test -- --run src/lib/api.test.ts
npm --prefix frontend run build
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat: add stewardship advantage ui"
```

## Task 5: Documentation And Release Gate

- [ ] **Step 1: Update docs**

Update public docs with deployment and safety notes for Seerr, Tautulli, Leaving Soon publishing, notifications, storage ledger, and protection requests.

- [ ] **Step 2: Add release notes**

Create `docs/release-notes/v1.6.0.md` describing the phase.

- [ ] **Step 3: Run full validation**

Run:

```bash
make ci
docker compose config --quiet
docker compose --profile ai config --quiet
docker compose build mediarr
scripts/verify-no-delete.sh
```

Expected: all pass.

- [ ] **Step 4: Commit and push**

```bash
git add README.md docs
git commit -m "docs: document stewardship advantage release"
git push origin codex/media-server-ingestion
```

## Self-Review

- Spec coverage: all seven requested features map to tasks above.
- Placeholder scan: no deferred requirements are left in task text.
- Safety consistency: no automatic deletion, no request clearing, and no Radarr/Sonarr destructive actions are included.
- Type consistency: route names and domain names match the design document.
