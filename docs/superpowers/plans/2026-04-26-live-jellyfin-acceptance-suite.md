# Live Jellyfin Acceptance Suite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an opt-in production acceptance suite that reads a live Jellyfin server, verifies inventory/activity ingestion, and produces safe recommendation reports without modifying Jellyfin or media files.

**Architecture:** Add series/anime-safe activity aggregation to the existing recommendation engine, then add a dedicated acceptance package and CLI that reuse the production Jellyfin connector, SQLite persistence, and recommendation rules. The runner writes JSON and Markdown reports to a gitignored local directory and fails on unsafe or useless ingestion signals.

**Tech Stack:** Go backend, existing Jellyfin integration client, existing SQLite store, existing recommendation engine, shell wrapper for Ubuntu/macOS operators.

---

### Task 1: Series And Anime Activity Recommendations

**Files:**
- Modify: `backend/internal/recommendations/engine.go`
- Modify: `backend/internal/recommendations/engine_test.go`
- Modify: `backend/internal/database/store.go`
- Modify: `backend/internal/database/integration_activity_test.go`

- [ ] **Step 1: Write failing recommendation tests**

Add tests proving episode files aggregate into one inactive-series recommendation, that never-watched series/anime produce abandoned-series suggestions, and that recently added episodes suppress active series.

- [ ] **Step 2: Run recommendation tests and confirm failure**

Run: `go test ./internal/recommendations`

Expected: tests fail because `ActivityMedia` does not expose parent/library fields and `GenerateActivity` only creates movie recommendations.

- [ ] **Step 3: Implement aggregation**

Extend `ActivityMedia` with parent and library fields. Update `GenerateActivity` to group episode files by series parent, sum size, preserve affected paths, aggregate play count, use the latest last-played date, suppress favorites and recently added active series, and emit `review_inactive_series` or `review_abandoned_series`.

- [ ] **Step 4: Update database activity query**

Join parent item and library rows in `ListActivityRecommendationMedia` so the engine receives series title, parent ID, and library name.

- [ ] **Step 5: Run targeted tests**

Run: `go test ./internal/recommendations ./internal/database`

Expected: pass.

### Task 2: Acceptance Report Builder

**Files:**
- Create: `backend/internal/acceptance/jellyfin.go`
- Create: `backend/internal/acceptance/jellyfin_test.go`

- [ ] **Step 1: Write failing report tests**

Add tests for path-map parsing, title redaction, warning generation for missing activity/size/local verification, and recommendation summarization.

- [ ] **Step 2: Run tests and confirm failure**

Run: `go test ./internal/acceptance`

Expected: fail because the package does not exist.

- [ ] **Step 3: Implement report builder**

Create a `RunJellyfin` function that calls `integrations.SyncJellyfin`, persists the snapshot to a temporary store, generates activity recommendations, validates safety invariants, and returns a structured report. Create helpers to write JSON and Markdown reports.

- [ ] **Step 4: Run acceptance package tests**

Run: `go test ./internal/acceptance`

Expected: pass.

### Task 3: Live CLI And Operator Script

**Files:**
- Create: `backend/cmd/mediarr-acceptance/main.go`
- Create: `scripts/acceptance-jellyfin-live.sh`
- Modify: `.gitignore`

- [ ] **Step 1: Add CLI tests where practical through package tests**

Keep CLI thin and test parsing/reporting in `backend/internal/acceptance`.

- [ ] **Step 2: Implement CLI**

Read `MEDIARR_ACCEPTANCE_JELLYFIN_URL`, `MEDIARR_ACCEPTANCE_JELLYFIN_API_KEY`, optional `MEDIARR_ACCEPTANCE_PATH_MAPS`, optional `MEDIARR_ACCEPTANCE_REPORT_DIR`, optional `MEDIARR_ACCEPTANCE_REDACT_TITLES`, and optional `MEDIARR_ACCEPTANCE_REQUIRE_LOCAL_VERIFY`.

- [ ] **Step 3: Implement wrapper script**

Run the CLI from the backend module, fail early if required env vars are missing, and keep generated reports under `acceptance-reports/` by default.

- [ ] **Step 4: Ignore local reports**

Add `acceptance-reports/` to `.gitignore`.

### Task 4: Documentation And Validation

**Files:**
- Modify: `README.md`
- Modify: `docs/deployment/docker-compose.md`
- Modify: `docs/release-checklist.md`

- [ ] **Step 1: Document the live acceptance workflow**

Explain that the suite is opt-in, read-only, uses a temporary DB, and can run against a real Jellyfin server.

- [ ] **Step 2: Run full validation**

Run:

```bash
cd backend && go test ./...
cd backend && go vet ./...
npm --prefix frontend run test -- --run
npm --prefix frontend run build
scripts/verify-no-delete.sh
docker compose config --quiet
docker compose --profile ai config --quiet
docker compose build mediarr
```

Expected: all pass.

- [ ] **Step 3: Commit and push**

Commit message: `feat: add live jellyfin acceptance suite`
