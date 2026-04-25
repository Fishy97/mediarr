# Media Server Ingestion And Activity Recommendations Design

## Goal

Add production-grade Jellyfin and Plex ingestion to Mediarr so connected media servers can report what media they know about and how people use it. Mediarr will normalize that inventory and activity into its own database, then generate suggest-only cleanup recommendations such as "no one has watched this movie in 18 months; removing it would save 42 GB."

This feature does not add download, acquisition, torrent, Usenet, or permanent deletion behavior.

## Product Position

Mediarr remains the storage steward and recommendation engine. Jellyfin and Plex become inventory and activity sources. Local filesystem scans remain valuable for verification, hashing, probing, and exact file-size checks, but the media server can be the first source of catalog truth when a user has already curated their library there.

The first implementation will support Jellyfin and Plex. The design keeps provider boundaries general enough to add Emby later without changing recommendation rules.

## UX Direction

The selected visual direction is MediaGlass: a cinematic, premium, data-rich operations console.

The UI should use a dark cinematic base, glass-like surfaces, restrained cyan/magenta/yellow accents, and polished typography. It should prioritize useful operational data over decorative copy:

- integration sync health and last successful sync
- total items imported from each server
- unmapped paths and match confidence
- activity recency, play counts, and unique user counts
- reclaimable storage by rule and confidence level
- recommendation explanations with exact server/source signals

The first screen remains a usable dashboard, not a landing page.

## Scope

### In Scope

- Configure Jellyfin and Plex credentials.
- Test connection health for each server.
- Pull media libraries and items from Jellyfin.
- Pull media libraries, items, media parts, and viewing history from Plex.
- Normalize inventory, file, user, and activity data into provider-neutral tables.
- Match server-reported files to Mediarr catalog files using path mapping and provider IDs.
- Generate activity-aware cleanup recommendations.
- Show sync state, imported items, unmapped items, and recommendation evidence through the API.
- Keep all recommendations advisory and non-destructive.

### Out Of Scope

- Deleting media files.
- Moving files to quarantine.
- Writing changes back to Jellyfin or Plex except explicit library refresh requests that already exist.
- Full Jellyfin Playback Reporting plugin ingestion in the first implementation.
- Emby ingestion in the first implementation.
- Replacing the local scanner entirely.

## Source Capabilities

### Jellyfin

Mediarr will call Jellyfin with `X-Emby-Token`.

Initial endpoints:

- `/System/Info` for connection health.
- `/Users` for local Jellyfin users.
- `/Items` with recursive library queries for movies, series, seasons, episodes, and videos.
- Item fields should request path, provider IDs, media sources, genres, production year, overview, date created, and user data where available.
- Per-user item data should be collected so recommendations can use play count, last played date, played state, and favorite state.

Jellyfin Playback Reporting may be added later as an optional richer activity source. Until then, user data rollups are enough for the first activity-aware recommendations.

### Plex

Mediarr will call Plex with `X-Plex-Token`.

Initial endpoints:

- `/identity` for connection health.
- `/library/sections` for library definitions.
- `/library/sections/{sectionKey}/all` for movies, shows, seasons, episodes, and videos.
- Media part data for file paths and sizes.
- `/status/sessions/history/all` for playback history where available.

Plex activity ingestion must be paginated and incremental because playback history can be large.

## Normalized Data Model

The database will add provider-neutral tables:

- `media_servers`: configured server records, kind, base URL, status, last sync, and redacted metadata.
- `media_server_users`: external users, display names, provider user IDs, disabled state, and last seen time.
- `media_server_libraries`: external libraries/sections with kind, name, provider IDs, and item counts.
- `media_server_items`: external movies, shows, seasons, episodes, and generic videos with provider IDs, title, year, runtime, path, parent external IDs, server source, and match confidence.
- `media_server_files`: external item file paths, reported sizes, container info, local path match, and local catalog file ID when resolved.
- `media_activity_events`: provider activity events such as playback start/completion/history rows when available.
- `media_activity_rollups`: per-item and per-file rollups containing last played date, play count, unique users, favorite count, watched user count, and stale/cold flags.
- `integration_path_mappings`: explicit path prefix mappings from server paths to Mediarr-visible paths, for example `/media/movies` to `/media/movies` or `/mnt/media` to `/media`.

Existing recommendation persistence should be extended instead of replaced. Recommendation rows should store source server IDs, activity evidence, verification state, and match confidence.

## Matching Strategy

Mediarr will resolve server items to local catalog files using a confidence ladder:

1. Exact normalized path match.
2. Explicit path mapping prefix match.
3. Provider ID match such as IMDb, TMDb, TVDb, or AniDB plus media type.
4. Title/year/runtime match.
5. File-size and filename similarity match.

Recommendations based on exact or mapped paths can show locally verified savings. Recommendations based only on server-reported file data must label savings as server-reported. Low-confidence matches should appear in an unmapped review surface instead of cleanup recommendations.

User overrides always win. Ignored recommendations remain ignored across syncs when the stable recommendation ID still maps to the same source item and file path.

## Sync Flow

1. User configures Jellyfin or Plex credentials.
2. Mediarr tests connection and stores redacted settings.
3. User starts a manual sync from the UI or API.
4. A sync job records progress, pages through provider data, and writes an external snapshot inside a transaction.
5. Mediarr resolves server files to local catalog files.
6. Activity rollups are recalculated.
7. Recommendation generation runs with deterministic rules.
8. API responses and audit events expose job status, imported item counts, failures, and recommendations created.

Sync jobs must be resumable enough to recover from provider outages and pagination failures without corrupting the previous successful snapshot. A failed sync should preserve old activity data and mark provider health degraded.

## Recommendation Rules

Initial activity-aware rules:

- `review_inactive_movie`: a movie has not been watched by any user for a configurable number of days.
- `review_never_watched_movie`: a movie has no play count after a configurable age.
- `review_inactive_series`: no episode in a series has been watched for a configurable number of days.
- `review_abandoned_series`: early episodes were watched but later seasons/episodes have no activity for a configurable number of days.
- `review_unwatched_duplicate`: duplicate files exist, and one copy has activity while another has none.

Rules must suppress or lower confidence for favorites, recently watched items, manually pinned items, poor matches, missing file paths, and active series.

Every recommendation must include:

- affected paths
- local or server-reported space savings
- media server source
- last played date
- play count
- unique users watched
- favorite count
- match confidence
- rule source
- explanation
- non-destructive flag

## REST API

Add or extend routes under `/api/v1`:

- `GET /integrations` returns connection, sync, and ingestion health.
- `POST /integrations/{id}/refresh` remains the explicit media-server library refresh action.
- `POST /integrations/{id}/sync` starts inventory and activity ingestion.
- `GET /integrations/{id}/sync` returns latest sync status.
- `GET /integrations/{id}/items` returns imported items with filters for unmapped, kind, library, and confidence.
- `GET /activity/rollups` returns normalized activity rollups.
- `GET /path-mappings` lists path mappings.
- `PUT /path-mappings/{id}` creates or updates a mapping.
- `DELETE /path-mappings/{id}` removes a mapping.

All routes except health/setup/login remain authenticated.

## Frontend

The first UI pass should add:

- MediaGlass integration configuration cards for Jellyfin and Plex.
- Sync buttons separate from refresh buttons.
- Sync progress, last successful sync, imported item counts, and failure reason.
- A path mapping review view for unmapped server paths.
- Recommendation cards that show activity evidence and savings confidence.
- Dashboard panels for cold storage, never watched media, inactive series, and server-reported versus locally verified savings.

## Error Handling And Safety

- Provider outages, bad credentials, rate limits, malformed responses, and pagination errors degrade integration health but do not clear existing catalog data.
- Sync writes must be transactional by server and job.
- Activity recommendations are never destructive.
- No endpoint may permanently delete media.
- Secrets are stored redacted in API responses.
- Logs must avoid printing provider tokens.
- Timeout and pagination limits must be configurable enough for large libraries.

## Testing

Backend tests:

- Jellyfin fixture ingestion for users, items, paths, provider IDs, and user data.
- Plex fixture ingestion for sections, items, media parts, and history rows.
- Path mapping and confidence ladder tests.
- Activity rollup tests.
- Recommendation tests for inactive, never watched, inactive series, abandoned series, favorites suppression, and duplicate activity.
- Auth tests for new sync and activity endpoints.
- Regression test proving no media deletion route exists.

Frontend tests:

- Integration configuration renders Jellyfin and Plex sync state.
- Sync actions call the expected endpoints.
- Recommendation cards display activity evidence.
- Empty and error states for no activity, provider outage, and unmapped paths.

Validation:

- `go test ./...`
- `go vet ./...`
- `npm --prefix frontend run test -- --run`
- `npm --prefix frontend run build`
- `docker compose config --quiet`
- `docker compose --profile ai config --quiet`
- Docker Compose smoke test when Docker is available.

## Rollout

Implement in small commits:

1. Normalized activity schema and domain interfaces.
2. Jellyfin connector and sync API.
3. Plex connector and sync API.
4. Activity rollups and recommendation rules.
5. MediaGlass integration/recommendation UI updates.
6. Documentation, validation, cleanup, and GitHub push.
