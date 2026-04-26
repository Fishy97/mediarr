# Recommendation Proof UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Mediarr’s Jellyfin/Plex ingestion and review queue understandable, trustworthy, and production-safe by clearly separating sync telemetry, activity evidence, estimated savings, verified savings, and grouped affected media.

**Architecture:** Keep deterministic ingestion and recommendations as the source of truth, then expose clearer proof models to the UI. Backend changes should improve persisted recommendation metadata and job telemetry without introducing destructive behavior. Frontend changes should make uncertainty explicit, collapse large file lists by default, and explain why a recommendation exists.

**Tech Stack:** Go backend, SQLite store, React/TypeScript frontend, Vitest, Go tests, Docker Compose, existing Mediarr REST API.

---

## Production Principles

- Mediarr must never imply deletion is automatic or safe without proof.
- Server-reported storage is an estimate from Jellyfin/Plex/Emby, not a guaranteed local disk saving.
- Path-mapped savings are stronger than server-reported savings, but still not as strong as local file verification.
- Locally verified savings are the only values that may be described as confirmed on disk.
- Per-user watch data must be aggregated by default. Names may appear only in short-lived job telemetry when privacy settings allow it.
- Long lists of affected episodes/files must be collapsed and grouped by media hierarchy.
- Every recommendation must answer: what is this, why was it suggested, who/what activity supports it, how confident is Mediarr, and how certain is the storage number?

## File Structure

### Backend

- Modify `backend/internal/integrations/integrations.go`
  - Normalize Jellyfin/Plex/Emby sync progress into explicit phases: connection, user discovery, inventory import, activity import, snapshot persistence, recommendation generation.
  - Avoid using a media title as the main label while reading user activity.
  - Add privacy-safe user labels for telemetry.

- Modify `backend/internal/recommendations/engine.go`
  - Add subject-aware recommendation titles for movies, series, and anime.
  - Add structured evidence fields for series title, item count, category, storage basis, savings certainty, and local verification status.
  - Preserve deterministic confidence scoring.

- Modify `backend/internal/database/store.go`
  - Persist any new evidence fields through the existing recommendation evidence JSON.
  - Avoid schema migrations unless a field must be queryable; prefer evidence JSON for proof details.

- Modify `backend/internal/api/server.go`
  - Ensure recommendation list and evidence endpoints expose enough proof data for the new UI.
  - Keep list payloads bounded where possible and reserve full path details for expanded proof views if the existing payload becomes too large.

- Add or modify tests:
  - `backend/internal/integrations/jellyfin_test.go`
  - `backend/internal/integrations/plex_test.go` if equivalent Plex coverage exists
  - `backend/internal/recommendations/engine_test.go`
  - `backend/internal/api/integration_sync_test.go`
  - `backend/internal/database/integration_activity_test.go`

### Frontend

- Modify `frontend/src/types.ts`
  - Add typed evidence helpers for storage certainty and grouped affected paths if returned by API.

- Modify `frontend/src/App.tsx`
  - Replace raw path walls with grouped/collapsed affected media views.
  - Add storage certainty UI: estimated savings, verified savings, confidence percentage, and evidence label.
  - Clarify activity proof: last played, total plays, watched user count, favorite/protected count.
  - Clarify sync telemetry so user names and media titles cannot be confused.

- Modify `frontend/src/lib/format.ts`
  - Add labels for storage certainty and verification states.

- Modify frontend tests:
  - `frontend/src/lib/api.test.ts`
  - Add component-level tests if the project introduces a testable recommendation card module.

### Documentation

- Modify `README.md`
- Modify `docs/api/rest.md`
- Modify `docs/deployment/docker-compose.md` only if settings or environment variables change.
- Add `docs/recommendation-proof.md` explaining storage certainty and activity evidence.

---

## Task 1: Normalize Sync Telemetry

**Intent:** The user should understand whether Mediarr is reading users, importing inventory, importing activity, or generating recommendations. Names like `Hugo` must not look like random media titles.

**Files:**
- Modify: `backend/internal/integrations/integrations.go`
- Modify: `backend/internal/api/server.go`
- Test: `backend/internal/integrations/jellyfin_test.go`
- Test: `backend/internal/api/integration_sync_test.go`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Write backend tests for phase-specific Jellyfin progress**

Add a Jellyfin sync test that captures `integrations.Progress` events and verifies:
- user discovery emits `phase=users`
- inventory/activity reads emit explicit messages such as `Reading Jellyfin activity for profile 1 of 5`
- media import emits media labels only during item import
- final generation emits a recommendation phase before completion

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/integrations -run TestSyncJellyfinReportsClearProgressPhases -count=1
```

Expected before implementation: fail because progress phases/messages are not yet specific enough.

- [ ] **Step 2: Implement privacy-safe progress labels**

In `backend/internal/integrations/integrations.go`, update Jellyfin/Plex progress generation so:
- `Phase: "users"` means reading server profiles.
- `Phase: "inventory"` means importing media items.
- `Phase: "activity"` means reading user-specific watch state.
- `CurrentLabel` for activity uses `Profile 3 of 5` by default.
- The media title is used only for inventory item progress.

The implementation should avoid persisting household member names in job events by default.

- [ ] **Step 3: Add recommendation-generation job phase**

In `backend/internal/api/server.go`, before regenerating recommendations after sync, update the job:

```text
Phase: recommendations
Message: Generating evidence-based recommendations
CurrentLabel: Deterministic review rules
```

This prevents completed ingestion from appearing stuck while recommendations are being generated.

- [ ] **Step 4: Update frontend labels**

In `frontend/src/App.tsx`, update job telemetry rendering so:
- `users` is shown as `Reading profiles`
- `inventory` is shown as `Importing media`
- `activity` is shown as `Importing watch activity`
- `recommendations` is shown as `Building review queue`

- [ ] **Step 5: Validate**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager
make ci
docker compose --profile ai up -d --build mediarr
```

Acceptance:
- The dashboard never shows a user profile name as if it were a movie.
- The integration tab shows the current operation clearly.
- Sync completion means ingestion and recommendation generation are both complete.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/integrations/integrations.go backend/internal/api/server.go backend/internal/integrations/jellyfin_test.go backend/internal/api/integration_sync_test.go frontend/src/App.tsx
git commit -m "fix: clarify media-server sync telemetry"
```

---

## Task 2: Add Storage Certainty And Savings Semantics

**Intent:** The review queue must make it impossible to mistake server-reported savings for guaranteed disk savings.

**Files:**
- Modify: `backend/internal/recommendations/engine.go`
- Modify: `backend/internal/database/store.go`
- Modify: `backend/internal/api/server.go`
- Test: `backend/internal/recommendations/engine_test.go`
- Test: `backend/internal/database/integration_activity_test.go`
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/lib/format.ts`

- [ ] **Step 1: Write recommendation evidence tests**

Add tests proving:
- `server_reported` recommendations expose `estimatedSavingsBytes` equal to the provider-reported total and `verifiedSavingsBytes` equal to `0`.
- `path_mapped` recommendations expose estimated savings but not fully verified savings unless local file proof exists.
- `local_verified` recommendations expose verified savings equal to the confirmed local size.
- confidence remains deterministic and is not changed by UI labels.

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/recommendations -run TestStorageCertaintyEvidence -count=1
```

Expected before implementation: fail because explicit estimated/verified savings evidence does not exist.

- [ ] **Step 2: Extend evidence JSON**

In `backend/internal/recommendations/engine.go`, add evidence entries for activity recommendations:

```text
storageBasis=server_reported|path_mapped|local_verified
estimatedSavingsBytes=<space_saved_bytes>
verifiedSavingsBytes=<0 or confirmed local size>
storageCertainty=estimate|mapped_estimate|verified
```

Keep `SpaceSavedBytes` as the backward-compatible estimated/reclaimable number.

- [ ] **Step 3: Preserve evidence through persistence**

Confirm `backend/internal/database/store.go` persists the new evidence JSON fields unchanged. Add a database regression test that writes and reads a recommendation with the new fields.

- [ ] **Step 4: Add frontend certainty rendering**

In `frontend/src/App.tsx`, each recommendation card should show:
- `Estimated savings`
- `Verified savings`
- `Confidence`
- `Evidence`

For server-reported recommendations, show a warning copy:

```text
Estimated from media-server data. Verify a path mapping before treating this as guaranteed disk savings.
```

For locally verified recommendations, show:

```text
Confirmed from read-only media mount.
```

- [ ] **Step 5: Add formatting helpers**

In `frontend/src/lib/format.ts`, add stable labels:

```text
server_reported -> Server estimate
path_mapped -> Path mapped estimate
local_verified -> Locally verified
unmapped -> Unmapped
```

- [ ] **Step 6: Validate**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager
go test ./backend/internal/recommendations ./backend/internal/database ./backend/internal/api
npm --prefix frontend run test -- --run
npm --prefix frontend run build
make ci
```

Acceptance:
- A Sopranos-style card with server-reported proof says `Estimated savings: 940 GB`, `Verified savings: 0 B`, and `Confidence: 68%`.
- No server-reported card implies the saving is guaranteed.
- Locally verified cards can truthfully say disk savings are confirmed.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/recommendations/engine.go backend/internal/database/store.go backend/internal/api/server.go backend/internal/recommendations/engine_test.go backend/internal/database/integration_activity_test.go frontend/src/types.ts frontend/src/App.tsx frontend/src/lib/format.ts
git commit -m "feat: expose recommendation storage certainty"
```

---

## Task 3: Group And Collapse Review Queue Media

**Intent:** A series recommendation must not render every episode as a wall of text. The card should summarize the series and reveal grouped file detail only when requested.

**Files:**
- Modify: `backend/internal/recommendations/engine.go`
- Test: `backend/internal/recommendations/engine_test.go`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/types.ts`
- Optional create: `frontend/src/lib/pathGroups.ts`
- Optional test: `frontend/src/lib/pathGroups.test.ts`

- [ ] **Step 1: Write backend tests for subject titles**

Add tests proving:
- abandoned series recommendation title is the series name, such as `The Sopranos`
- action label remains available through `action`
- evidence includes `itemCount`, `category`, and `seriesTitle`
- movie recommendations use the movie title

Run:

```bash
cd /Users/mat/Desktop/media-library-manager/backend
go test ./internal/recommendations -run TestActivityRecommendationsUseSubjectTitles -count=1
```

Expected before implementation: fail because titles are generic.

- [ ] **Step 2: Update recommendation titles and evidence**

In `backend/internal/recommendations/engine.go`:
- set `Title` to `item.Title` for movie recommendations
- set `Title` to `group.title` for series/anime recommendations
- store action/rule meaning in `Action` and `Source`
- add evidence fields:

```text
subjectKind=movie|series|anime
subjectTitle=<title>
itemCount=<number of files>
```

- [ ] **Step 3: Add path grouping helper**

Create `frontend/src/lib/pathGroups.ts` if the logic becomes too large for `App.tsx`.

Grouping rules:
- Movies: one group named `File`
- Series/anime: group by season directory when paths contain `/Season N/`
- Fallback: group by immediate parent directory
- Each group exposes `label`, `count`, `paths`, and `estimatedBytes` if per-path size is available later

- [ ] **Step 4: Render compact cards**

In `frontend/src/App.tsx`, change recommendation cards to:
- card title: subject title
- action badge: `Abandoned series`, `Never-watched movie`, `Inactive series`
- summary row: `86 files`, `6 seasons`, `Server estimate`, `68% confidence`
- collapsed file preview: first 3 paths only
- dropdown button: `Show 86 affected files`
- expanded view grouped by season/parent folder

- [ ] **Step 5: Add keyboard/accessibility behavior**

The dropdown must:
- use a real button
- expose `aria-expanded`
- not shift layout unpredictably
- remain usable on mobile

- [ ] **Step 6: Validate**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager
npm --prefix frontend run test -- --run
npm --prefix frontend run build
make ci
```

Browser acceptance:
- The Sopranos appears as `The Sopranos`, not `Review abandoned series`.
- The card does not show all paths by default.
- Expanding the card groups files by season.
- The page remains performant with hundreds of affected paths.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/recommendations/engine.go backend/internal/recommendations/engine_test.go frontend/src/App.tsx frontend/src/types.ts frontend/src/lib/pathGroups.ts frontend/src/lib/pathGroups.test.ts
git commit -m "feat: group affected media in review queue"
```

---

## Task 4: Add Plain-English Proof Language

**Intent:** Users should understand the difference between activity evidence, storage evidence, and recommendation confidence without reading docs.

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/lib/format.ts`
- Modify: `frontend/src/types.ts`
- Modify: `docs/api/rest.md`
- Create: `docs/recommendation-proof.md`
- Modify: `README.md`

- [ ] **Step 1: Define canonical proof copy**

Use these labels consistently:

```text
Server estimate: Jellyfin/Plex/Emby reports this path and size. Mediarr has not verified it on disk.
Path mapped estimate: Mediarr translated the server path to a local mount, but size still needs local confirmation.
Locally verified: Mediarr found the file on a read-only mount and confirmed the size.
Unmapped: Mediarr cannot connect the server path to a local file path yet.
```

- [ ] **Step 2: Add proof explainer components**

In `frontend/src/App.tsx`, add compact explanation rows:
- `Why suggested`
- `Storage certainty`
- `Activity proof`
- `Safety`

The safety row must always say:

```text
Mediarr will not delete this. Accepting marks it for manual action only.
```

- [ ] **Step 3: Improve proof panel**

The existing proof panel should show:
- rule source
- thresholds, such as `older than 180 days`
- last played date or `Never watched by imported users`
- watched user count
- favorite/protection suppression state
- storage certainty definition

- [ ] **Step 4: Document the model**

Create `docs/recommendation-proof.md` with:
- what Mediarr imports from media servers
- how per-user Jellyfin/Plex data is aggregated
- what confidence means
- what estimated savings means
- how path mapping upgrades proof
- why recommendations stay suggest-only

- [ ] **Step 5: Validate**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager
npm --prefix frontend run test -- --run
npm --prefix frontend run build
make ci
```

Acceptance:
- A non-technical admin can tell whether a storage number is estimated or verified.
- The review queue clearly explains why a movie/series is suggested.
- The UI never suggests Mediarr will delete anything automatically.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/App.tsx frontend/src/lib/format.ts frontend/src/types.ts docs/recommendation-proof.md docs/api/rest.md README.md
git commit -m "docs: explain recommendation proof model"
```

---

## Task 5: End-To-End NAS Acceptance Test

**Intent:** Validate this against a real, frequently used Jellyfin instance without modifying media.

**Files:**
- Modify: `docs/deployment/docker-compose.md`
- Modify: `docs/recommendation-proof.md`
- Optional modify: `backend/cmd/mediarr-acceptance`

- [ ] **Step 1: Run local production build**

```bash
cd /Users/mat/Desktop/media-library-manager
make ci
docker compose --profile ai up -d --build
```

Expected:
- Mediarr healthy on `http://localhost:8080`
- Ollama sidecar available if AI profile is enabled
- no debug bearer token enabled

- [ ] **Step 2: Sync Jellyfin from UI**

From the Integrations tab:
- click Jellyfin `Sync`
- verify progress moves through users, inventory, activity, recommendations, completed
- verify no random user names appear as media titles

- [ ] **Step 3: Check recommendation proof**

In Review Queue:
- confirm confidence percentage is visible
- confirm estimated and verified savings are both visible
- confirm server-reported recommendations warn that savings are not guaranteed
- expand a series recommendation and confirm paths are grouped

- [ ] **Step 4: Configure path mapping**

For the current NAS example:
- server path should suggest `/Volume1/Media`
- local path should be the mounted container path, such as `/media`
- save and verify the mapping
- rerun sync or verification

Expected:
- server-reported proof upgrades where local files are visible
- verified savings increase only for files Mediarr can actually confirm

- [ ] **Step 5: Confirm no destructive capability**

Run:

```bash
cd /Users/mat/Desktop/media-library-manager
scripts/verify-no-delete.sh
```

Expected:

```text
No-delete invariant verified.
```

- [ ] **Step 6: Commit acceptance docs**

```bash
git add docs/deployment/docker-compose.md docs/recommendation-proof.md backend/cmd/mediarr-acceptance
git commit -m "docs: add recommendation proof acceptance flow"
```

---

## Release Gate

Before merging:

```bash
cd /Users/mat/Desktop/media-library-manager
make ci
docker compose config --quiet
docker compose --profile ai config --quiet
docker compose --profile ai up -d --build
curl -fsS http://localhost:8080/api/v1/health
scripts/verify-no-delete.sh
```

Required outcomes:
- backend tests pass
- frontend tests pass
- frontend production build passes
- Go vet passes
- Docker build passes
- no-delete invariant passes
- Integrations tab renders without console errors
- live Jellyfin sync completes and produces recommendations
- Review Queue clearly separates estimated savings from verified savings
- PR GitHub Actions pass

## Self-Review

- Spec coverage: all four requested pillars are represented as tasks: sync telemetry, savings certainty, grouped review queue, and proof language.
- Placeholder scan: no TBD/TODO placeholders remain.
- Type consistency: plan uses existing domain names where possible: `Recommendation`, `Evidence`, `Verification`, `SpaceSavedBytes`, `AffectedPaths`, `Progress`, `JobUpdate`.
- Scope check: this is one cohesive UX/proof hardening project, not a destructive-action project.

