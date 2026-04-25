# Media Server Ingestion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build production-grade Jellyfin and Plex ingestion so Mediarr can import media-server inventory, normalize user activity, and generate safe cleanup recommendations from real viewing signals.

**Architecture:** Add a provider-neutral activity model in `backend/internal/integrations` and persist it in SQLite through `backend/internal/database`. Extend the existing API server with sync/status/items/activity endpoints, then feed normalized rollups into the recommendation engine without allowing any destructive media operations.

**Tech Stack:** Go 1.23, `net/http`, `encoding/json`, `encoding/xml`, SQLite via `modernc.org/sqlite`, React/TypeScript/Vite, Docker Compose.

---

## File Map

- Modify `backend/internal/integrations/integrations.go`: keep existing health and refresh behavior, add normalized model types, sync result types, path mapping helpers, Jellyfin client, Plex client, and activity-aware recommendation input conversion.
- Add `backend/internal/integrations/jellyfin_test.go`: fixture-backed Jellyfin sync tests for users, items, user data, media sources, paths, and rollups.
- Add `backend/internal/integrations/plex_test.go`: fixture-backed Plex sync tests for sections, items, media parts, and playback history.
- Add `backend/internal/integrations/activity_test.go`: provider-neutral rollup and path mapping tests.
- Modify `backend/internal/recommendations/engine.go`: add activity recommendation actions, evidence fields, and generation rules.
- Modify `backend/internal/recommendations/engine_test.go`: assert inactive/never-watched recommendations, favorites suppression, and non-destructive invariants.
- Modify `backend/internal/database/store.go`: add migrations for media servers, users, libraries, items, files, rollups, sync jobs, path mappings, and extended recommendation evidence; add store methods.
- Add `backend/internal/database/integration_activity_test.go`: persistence tests for sync snapshots, rollups, mappings, and recommendation evidence.
- Modify `backend/internal/api/server.go`: add sync/status/items/activity/path-mapping routes; regenerate recommendations after successful sync.
- Add `backend/internal/api/integration_sync_test.go`: API auth, sync action, activity rollup, and no-delete regression tests.
- Modify `backend/cmd/mediarr/main.go`: pass integration options and store-backed sync dependencies already available from config.
- Modify `frontend/src/types.ts`: add integration sync, imported item, activity rollup, path mapping, and recommendation evidence types.
- Modify `frontend/src/lib/api.ts`: add sync, item, rollup, and path mapping API methods.
- Modify `frontend/src/App.tsx`: add MediaGlass integration sync panels, activity cleanup view, path mapping status, and richer recommendation evidence.
- Modify `frontend/src/styles/app.css`: apply MediaGlass visual direction while preserving responsive operational UI.
- Modify `frontend/src/lib/api.test.ts`: cover new API methods.
- Modify `README.md`, `docs/api/rest.md`, and `docs/deployment/docker-compose.md`: document Jellyfin/Plex ingestion, permissions, activity privacy, and deployment.

## Task 1: Normalized Activity Schema And Store

**Files:**
- Modify: `backend/internal/database/store.go`
- Test: `backend/internal/database/integration_activity_test.go`

- [ ] **Step 1: Write failing migration and persistence tests**

Create `backend/internal/database/integration_activity_test.go` with tests that open a temp store, call new methods, and verify these behaviors:

```go
func TestMediaServerSnapshotPersistsNormalizedActivity(t *testing.T) {
	store := openTestStore(t)
	snapshot := MediaServerSnapshot{
		Server: MediaServer{ID: "srv_jellyfin", Kind: "jellyfin", Name: "Jellyfin", BaseURL: "http://jellyfin.local"},
		Users: []MediaServerUser{{ServerID: "srv_jellyfin", ExternalID: "u1", DisplayName: "Alex"}},
		Libraries: []MediaServerLibrary{{ServerID: "srv_jellyfin", ExternalID: "lib_movies", Name: "Movies", Kind: "movie"}},
		Items: []MediaServerItem{{ServerID: "srv_jellyfin", ExternalID: "item_1", LibraryExternalID: "lib_movies", Kind: "movie", Title: "Arrival", Year: 2016, Path: "/media/movies/Arrival (2016).mkv"}},
		Files: []MediaServerFile{{ServerID: "srv_jellyfin", ItemExternalID: "item_1", Path: "/media/movies/Arrival (2016).mkv", SizeBytes: 42_000_000_000, Verification: "server_reported"}},
		Rollups: []MediaActivityRollup{{ServerID: "srv_jellyfin", ItemExternalID: "item_1", PlayCount: 2, UniqueUsers: 1, LastPlayedAt: parseTestTime("2025-01-02T03:04:05Z")}},
		Job: MediaSyncJob{ServerID: "srv_jellyfin", Status: "completed", ItemsImported: 1, CompletedAt: parseTestTime("2026-04-26T10:00:00Z")},
	}
	if err := store.ReplaceMediaServerSnapshot(snapshot); err != nil {
		t.Fatal(err)
	}
	rollups, err := store.ListMediaActivityRollups("")
	if err != nil {
		t.Fatal(err)
	}
	if len(rollups) != 1 || rollups[0].PlayCount != 2 || rollups[0].UniqueUsers != 1 {
		t.Fatalf("rollups = %#v", rollups)
	}
	items, err := store.ListMediaServerItems(MediaServerItemFilter{ServerID: "srv_jellyfin"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Arrival" {
		t.Fatalf("items = %#v", items)
	}
}
```

Add a second test for `UpsertPathMapping`, `ListPathMappings`, and `DeletePathMapping`.

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./backend/internal/database -run 'TestMediaServerSnapshot|TestPathMapping'
```

Expected: fail because new types and methods are undefined.

- [ ] **Step 3: Implement database types and migrations**

In `backend/internal/database/store.go`, add typed structs for `MediaServer`, `MediaServerUser`, `MediaServerLibrary`, `MediaServerItem`, `MediaServerFile`, `MediaActivityRollup`, `MediaSyncJob`, `PathMapping`, `MediaServerSnapshot`, and `MediaServerItemFilter`.

Add migrations for:

```sql
CREATE TABLE IF NOT EXISTS media_servers (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	name TEXT NOT NULL,
	base_url TEXT NOT NULL,
	status TEXT NOT NULL,
	last_synced_at TEXT,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS media_server_users (
	server_id TEXT NOT NULL,
	external_id TEXT NOT NULL,
	display_name TEXT NOT NULL,
	last_seen_at TEXT,
	PRIMARY KEY (server_id, external_id)
);
CREATE TABLE IF NOT EXISTS media_server_libraries (
	server_id TEXT NOT NULL,
	external_id TEXT NOT NULL,
	name TEXT NOT NULL,
	kind TEXT NOT NULL,
	item_count INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (server_id, external_id)
);
CREATE TABLE IF NOT EXISTS media_server_items (
	server_id TEXT NOT NULL,
	external_id TEXT NOT NULL,
	library_external_id TEXT NOT NULL DEFAULT '',
	parent_external_id TEXT NOT NULL DEFAULT '',
	kind TEXT NOT NULL,
	title TEXT NOT NULL,
	year INTEGER NOT NULL DEFAULT 0,
	path TEXT NOT NULL DEFAULT '',
	provider_ids TEXT NOT NULL DEFAULT '{}',
	runtime_seconds INTEGER NOT NULL DEFAULT 0,
	date_created TEXT,
	match_confidence REAL NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (server_id, external_id)
);
CREATE TABLE IF NOT EXISTS media_server_files (
	server_id TEXT NOT NULL,
	item_external_id TEXT NOT NULL,
	path TEXT NOT NULL,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	container TEXT NOT NULL DEFAULT '',
	local_path TEXT NOT NULL DEFAULT '',
	local_media_file_id TEXT NOT NULL DEFAULT '',
	verification TEXT NOT NULL DEFAULT 'server_reported',
	match_confidence REAL NOT NULL DEFAULT 0,
	PRIMARY KEY (server_id, item_external_id, path)
);
CREATE TABLE IF NOT EXISTS media_activity_rollups (
	server_id TEXT NOT NULL,
	item_external_id TEXT NOT NULL,
	play_count INTEGER NOT NULL DEFAULT 0,
	unique_users INTEGER NOT NULL DEFAULT 0,
	watched_users INTEGER NOT NULL DEFAULT 0,
	favorite_count INTEGER NOT NULL DEFAULT 0,
	last_played_at TEXT,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (server_id, item_external_id)
);
CREATE TABLE IF NOT EXISTS media_sync_jobs (
	id TEXT PRIMARY KEY,
	server_id TEXT NOT NULL,
	status TEXT NOT NULL,
	items_imported INTEGER NOT NULL DEFAULT 0,
	rollups_imported INTEGER NOT NULL DEFAULT 0,
	unmapped_items INTEGER NOT NULL DEFAULT 0,
	cursor TEXT NOT NULL DEFAULT '',
	error TEXT NOT NULL DEFAULT '',
	started_at TEXT NOT NULL,
	completed_at TEXT
);
CREATE TABLE IF NOT EXISTS integration_path_mappings (
	id TEXT PRIMARY KEY,
	server_id TEXT NOT NULL DEFAULT '',
	server_path_prefix TEXT NOT NULL,
	local_path_prefix TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
```

Add columns to `recommendations`: `server_id`, `external_item_id`, `last_played_at`, `play_count`, `unique_users`, `favorite_count`, `verification`, and `evidence`.

- [ ] **Step 4: Implement store methods**

Implement:

```go
func (store *Store) ReplaceMediaServerSnapshot(snapshot MediaServerSnapshot) error
func (store *Store) ListMediaServerItems(filter MediaServerItemFilter) ([]MediaServerItem, error)
func (store *Store) ListMediaActivityRollups(serverID string) ([]MediaActivityRollup, error)
func (store *Store) LatestMediaSyncJob(serverID string) (MediaSyncJob, error)
func (store *Store) UpsertPathMapping(mapping PathMapping) (PathMapping, error)
func (store *Store) ListPathMappings() ([]PathMapping, error)
func (store *Store) DeletePathMapping(id string) error
```

Use a transaction for snapshot replacement and preserve old snapshots if any write fails.

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./backend/internal/database
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/database/store.go backend/internal/database/integration_activity_test.go
git commit -m "feat: add media server activity storage"
git push
```

## Task 2: Jellyfin Connector And Sync API

**Files:**
- Modify: `backend/internal/integrations/integrations.go`
- Add tests: `backend/internal/integrations/jellyfin_test.go`
- Modify: `backend/internal/api/server.go`
- Add tests: `backend/internal/api/integration_sync_test.go`

- [ ] **Step 1: Write failing Jellyfin connector test**

Create `backend/internal/integrations/jellyfin_test.go` using `httptest.Server`. The fake server must support `/System/Info`, `/Users`, and `/Items`.

Assert:

- request includes `X-Emby-Token`
- users are imported
- a movie item is imported with path, provider IDs, media source size, and user data
- rollup uses `PlayCount`, `LastPlayedDate`, and `IsFavorite`

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./backend/internal/integrations -run Jellyfin
```

Expected: fail because `SyncJellyfin` is undefined.

- [ ] **Step 3: Implement Jellyfin sync**

Add:

```go
func SyncJellyfin(ctx context.Context, options Options, mappings []database.PathMapping) (database.MediaServerSnapshot, error)
```

Use pagination with `StartIndex` and `Limit=200`. Request item fields needed for path/provider/user data. Keep the HTTP client timeout bounded. Do not log or return the API token.

- [ ] **Step 4: Write failing API sync test**

In `backend/internal/api/integration_sync_test.go`, assert:

- `POST /api/v1/integrations/jellyfin/sync` returns `202`
- the store receives one imported item and one rollup
- unauthenticated requests are rejected when auth is configured
- `GET /api/v1/activity/rollups` returns imported activity

- [ ] **Step 5: Implement API routes**

Extend `integrationActionHandler` to support `/sync` and add:

```go
server.mux.HandleFunc("/api/v1/activity/rollups", server.activityRollupsHandler)
server.mux.HandleFunc("/api/v1/path-mappings", server.pathMappingsHandler)
server.mux.HandleFunc("/api/v1/path-mappings/", server.pathMappingHandler)
```

After successful Jellyfin sync, persist the snapshot, regenerate recommendations, and audit `integration.sync_completed`.

- [ ] **Step 6: Run tests**

```bash
go test ./backend/internal/integrations ./backend/internal/api
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/integrations/integrations.go backend/internal/integrations/jellyfin_test.go backend/internal/api/server.go backend/internal/api/integration_sync_test.go
git commit -m "feat: add jellyfin activity ingestion"
git push
```

## Task 3: Plex Connector And Sync API

**Files:**
- Modify: `backend/internal/integrations/integrations.go`
- Test: `backend/internal/integrations/plex_test.go`
- Modify: `backend/internal/api/server.go`

- [ ] **Step 1: Write failing Plex connector test**

Create `backend/internal/integrations/plex_test.go` with XML fixtures for `/identity`, `/library/sections`, `/library/sections/1/all`, and `/status/sessions/history/all`.

Assert:

- token is passed as `X-Plex-Token`
- library sections import as libraries
- movie media parts import file path and size
- history rows import `viewedAt`, `accountID`, and roll up last played and play count

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./backend/internal/integrations -run Plex
```

Expected: fail because `SyncPlex` is undefined.

- [ ] **Step 3: Implement Plex sync**

Add:

```go
func SyncPlex(ctx context.Context, options Options, mappings []database.PathMapping) (database.MediaServerSnapshot, error)
```

Use XML decoding for Plex responses. Import movies and episodes from library sections. Import playback history with pagination controls where available. Prefer locally matched file sizes when path mapping succeeds; otherwise use Plex part size as server-reported.

- [ ] **Step 4: Wire Plex through API sync route**

Update sync dispatch to call Jellyfin or Plex based on integration ID. Return `400` for unsupported sync targets such as Emby until Emby ingestion exists.

- [ ] **Step 5: Run tests**

```bash
go test ./backend/internal/integrations ./backend/internal/api
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/integrations/integrations.go backend/internal/integrations/plex_test.go backend/internal/api/server.go
git commit -m "feat: add plex activity ingestion"
git push
```

## Task 4: Activity Recommendation Rules

**Files:**
- Modify: `backend/internal/recommendations/engine.go`
- Test: `backend/internal/recommendations/engine_test.go`
- Modify: `backend/internal/database/store.go`
- Modify: `backend/internal/api/server.go`

- [ ] **Step 1: Write failing recommendation tests**

Add tests for:

- `review_never_watched_movie` when `PlayCount == 0`, item is old enough, and not favorite.
- `review_inactive_movie` when `LastPlayedAt` is older than threshold.
- favorite suppression.
- every activity recommendation has `Destructive == false`.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./backend/internal/recommendations -run 'Activity|Inactive|Never'
```

Expected: fail because activity inputs/actions do not exist.

- [ ] **Step 3: Extend recommendation model and engine**

Add actions:

```go
ActionReviewInactiveMovie Action = "review_inactive_movie"
ActionReviewNeverWatchedMovie Action = "review_never_watched_movie"
ActionReviewInactiveSeries Action = "review_inactive_series"
ActionReviewAbandonedSeries Action = "review_abandoned_series"
ActionReviewUnwatchedDuplicate Action = "review_unwatched_duplicate"
```

Add `ActivityMedia` and `ActivityEvidence` types. Add:

```go
func (engine Engine) GenerateActivity(items []ActivityMedia, now time.Time) []Recommendation
```

Use conservative defaults: never-watched age 180 days, inactive threshold 540 days, minimum confidence 0.72, suppress favorites and zero-size items.

- [ ] **Step 4: Persist evidence fields**

Update recommendation insert/list methods for server ID, external item ID, last played, play count, unique users, favorite count, verification, and evidence JSON.

- [ ] **Step 5: Regenerate recommendations from catalog and activity**

Update API regeneration paths so catalog rules and activity rules are combined before AI enrichment and persistence.

- [ ] **Step 6: Run tests**

```bash
go test ./backend/internal/recommendations ./backend/internal/database ./backend/internal/api
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/recommendations/engine.go backend/internal/recommendations/engine_test.go backend/internal/database/store.go backend/internal/api/server.go backend/internal/database/integration_activity_test.go backend/internal/api/integration_sync_test.go
git commit -m "feat: add activity cleanup recommendations"
git push
```

## Task 5: MediaGlass Integration And Cleanup UI

**Files:**
- Modify: `frontend/src/types.ts`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Write failing frontend API tests**

Add tests for `api.syncIntegration("jellyfin")`, `api.activityRollups()`, `api.pathMappings()`, and `api.integrationItems("jellyfin")`.

- [ ] **Step 2: Run test to verify it fails**

```bash
npm --prefix frontend run test -- --run
```

Expected: fail because API methods are missing.

- [ ] **Step 3: Implement frontend API types and methods**

Add TypeScript interfaces matching backend JSON fields and API methods for sync, rollups, integration items, and path mappings.

- [ ] **Step 4: Update UI**

Update the integrations view to show Jellyfin/Plex sync cards with:

- connection status
- sync button
- last sync status
- imported item count
- unmapped item count
- activity rollup count

Update recommendation cards with:

- server source
- last played
- play count
- unique users
- verification state
- confidence
- affected paths

Add dashboard panels for cold storage, never watched, inactive media, and locally verified savings.

- [ ] **Step 5: Apply MediaGlass theme**

Use dark cinematic surfaces, translucent panels, cyan/magenta/yellow accents, and readable typography. Keep controls compact and data-dense. Do not add a landing page.

- [ ] **Step 6: Run frontend validation**

```bash
npm --prefix frontend run test -- --run
npm --prefix frontend run build
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/types.ts frontend/src/lib/api.ts frontend/src/lib/api.test.ts frontend/src/App.tsx frontend/src/styles/app.css
git commit -m "feat: add media server cleanup ui"
git push
```

## Task 6: Docs, Validation, And Cleanup

**Files:**
- Modify: `README.md`
- Modify: `docs/api/rest.md`
- Modify: `docs/deployment/docker-compose.md`
- Modify: `docs/superpowers/plans/2026-04-26-media-server-ingestion.md`

- [ ] **Step 1: Update docs**

Document:

- Jellyfin and Plex ingestion setup.
- Difference between refresh and sync.
- Activity privacy controls.
- Path mappings.
- Server-reported versus locally verified savings.
- No-delete safety rule.

- [ ] **Step 2: Run backend validation**

```bash
go test ./...
go vet ./...
```

Expected: both pass.

- [ ] **Step 3: Run frontend validation**

```bash
npm --prefix frontend run test -- --run
npm --prefix frontend run build
```

Expected: both pass.

- [ ] **Step 4: Run Compose validation**

```bash
docker compose config --quiet
docker compose --profile ai config --quiet
docker compose build mediarr
```

Expected: all pass.

- [ ] **Step 5: Clean local artifacts**

Run:

```bash
git status --short
```

Expected: only intentional source/docs changes before the final commit. `.superpowers/` must remain ignored.

- [ ] **Step 6: Commit and push**

```bash
git add README.md docs/api/rest.md docs/deployment/docker-compose.md docs/superpowers/plans/2026-04-26-media-server-ingestion.md
git commit -m "docs: document media server ingestion"
git push
```

- [ ] **Step 7: Open or update pull request**

Open a pull request from `codex/media-server-ingestion` to `main`, include validation evidence, and keep it ready for review once CI is green.
