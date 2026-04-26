# REST API

All API routes are rooted at `/api/v1`.

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Service health |
| GET | `/setup/status` | First-run setup state |
| POST | `/setup/admin` | Create the first local admin account |
| POST | `/auth/login` | Create an authenticated session |
| POST | `/auth/logout` | Revoke the current session |
| GET | `/auth/me` | Current authenticated user |
| GET | `/libraries` | Configured libraries |
| POST | `/libraries` | Add a library |
| GET | `/jobs` | Recent background jobs; supports `?active=true`, `?kind=`, `?targetId=`, and `?limit=` |
| GET | `/jobs/{id}` | Job progress plus recent job events |
| POST | `/jobs/{id}/cancel` | Cancel a queued or running background job |
| POST | `/jobs/{id}/retry` | Queue a new job from a completed, failed, canceled, or stale job |
| GET | `/catalog` | Persisted media catalog |
| PUT | `/catalog/{mediaFileId}/correction` | Apply a user-approved metadata correction |
| DELETE | `/catalog/{mediaFileId}/correction` | Clear a metadata correction |
| GET | `/scans` | Scan history |
| POST | `/scans` | Queue a background filesystem scan job |
| GET | `/scans/active` | Active filesystem scan jobs |
| GET | `/recommendations` | Open cleanup review items |
| GET | `/recommendations/{id}/evidence` | Structured recommendation proof with storage, activity, source, and risk evidence |
| POST | `/recommendations/{id}/ignore` | Hide an advisory recommendation |
| POST | `/recommendations/{id}/restore` | Restore an ignored advisory recommendation |
| POST | `/recommendations/{id}/protect` | Protect the item and remove it from the open recommendation queue |
| POST | `/recommendations/{id}/accept-manual` | Mark the suggestion accepted for manual action |
| GET | `/providers` | Metadata provider health and attribution |
| GET | `/provider-settings` | Redacted provider credential and base URL settings |
| PUT | `/provider-settings/{provider}` | Update provider base URL, API key, or clear stored key |
| GET | `/integration-settings` | Redacted Jellyfin, Plex, and Emby connection settings |
| PUT | `/integration-settings/{id}` | Update media-server base URL, API key/token, auto-sync settings, or clear stored key |
| GET | `/integrations` | Media-server and AI integration status |
| POST | `/integrations/{id}/refresh` | Request a Jellyfin, Plex, or Emby library refresh |
| POST | `/integrations/{id}/sync` | Queue a background Jellyfin, Plex, or Emby inventory/activity sync job |
| GET | `/integrations/{id}/sync` | Active or latest media-server sync job |
| GET | `/integrations/{id}/diagnostics` | Imported inventory/activity proof summary, warnings, storage verification, and top suggestions |
| GET | `/integrations/{id}/items` | Imported media-server items; supports `?unmapped=true` |
| GET | `/activity/rollups` | Normalized media activity rollups; supports `?serverId=` |
| GET | `/path-mappings` | Path prefix mappings used to resolve server paths to Mediarr paths |
| GET | `/path-mappings/unmapped` | Imported server items that still lack usable local path evidence; supports `?serverId=` |
| POST | `/path-mappings` | Create a path mapping |
| PUT | `/path-mappings/{id}` | Update a path mapping |
| POST | `/path-mappings/{id}/verify` | Verify mapped server files against local paths and update evidence labels |
| DELETE | `/path-mappings/{id}` | Delete a path mapping |
| GET | `/ai/status` | Optional local AI health and model availability |
| GET | `/backups` | List `/config` backups |
| POST | `/backups` | Create a `/config` backup |
| GET | `/backups/{name}` | Download one backup by safe archive name |
| POST | `/backups/restore` | Inspect or restore a `/config` backup |
| GET | `/support/bundles` | List redacted support bundles under `/config/support` |
| POST | `/support/bundles` | Create a redacted diagnostics archive under `/config/support` |
| GET | `/support/bundles/{name}` | Download one support bundle by safe archive name |
| GET | `/audit` | Audit log lines |

No media file deletion route is provided.

`refresh` and `sync` are intentionally different. `refresh` tells a media server to refresh its own library. `sync` pulls inventory, file paths, file sizes, and activity signals into Mediarr for suggest-only cleanup recommendations. Plex sync stores a watch-history cursor and uses it on later runs while preserving existing normalized activity rollups.

Long-running scan and sync requests return a job object immediately. Poll `/jobs/{id}` for `status`, `phase`, `message`, `processed`, `total`, `currentLabel`, imported counts, and recent events. Jobs support `queued`, `running`, `completed`, `failed`, `canceled`, and `stale` states. Listing jobs marks old queued/running rows stale after 24 hours without progress.

Recommendation evidence is intentionally verbose. Clients should display storage verification separately from confidence: `local_verified` means Mediarr found a matching local file, `path_mapped` means a mapping resolved the server path, `server_reported` means savings came from the media server, and `unmapped` is blocked from cleanup recommendations.

Provider and media-server API calls use bounded retry behavior for `429` and `5xx` responses. `Retry-After` is honored when present, capped to avoid wedging background jobs indefinitely.

Integration settings include `autoSyncEnabled` and `autoSyncIntervalMinutes`. Auto-sync defaults to enabled at 360 minutes. Saving a valid URL and API key/token queues the first sync immediately, and the backend scheduler keeps due integrations fresh while avoiding duplicate active jobs.

Integration diagnostics reuse the same evidence model as the live Jellyfin acceptance suite. They summarize persisted imported data, not raw provider payloads: movie/series/episode counts, file counts, server-reported bytes, locally verified bytes, unmapped files, activity rollups, warning messages, and top recommendations.

Support bundles package redacted provider and integration settings, path mappings, recent jobs, recommendation state, ingestion diagnostics, and safety proof into a zip archive. They intentionally exclude media files, the raw SQLite database, raw provider payloads, and API keys. Download paths only accept generated `mediarr-support-*.zip` archive names and reject path traversal. The archive may still include media titles and paths because those are necessary to troubleshoot ingestion evidence.

Backup downloads and restores only accept generated `mediarr-*.zip` archive names or existing paths under `/config/backups`. Non-dry-run restore requests must send `confirmRestore: true`; Mediarr creates a pre-restore backup before replacing files under `/config`.
