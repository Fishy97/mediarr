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
| GET | `/campaigns` | Saved stewardship campaigns |
| POST | `/campaigns` | Create a stewardship campaign |
| GET | `/campaigns/{id}` | Fetch one campaign definition |
| PUT | `/campaigns/{id}` | Update one campaign definition |
| DELETE | `/campaigns/{id}` | Delete the campaign definition and run history without deleting media |
| POST | `/campaigns/{id}/simulate` | Evaluate a campaign against imported media-server activity without changing recommendations |
| POST | `/campaigns/{id}/what-if` | Evaluate a campaign and include request/protection conflicts plus unmapped blockers |
| POST | `/campaigns/{id}/run` | Record a campaign run and create suggest-only campaign recommendations |
| POST | `/campaigns/{id}/publish-preview` | Create a dry-run Leaving Soon collection publication plan from verified campaign matches |
| POST | `/campaigns/{id}/publish` | Publish a verified Jellyfin collection plan; requires `confirmPublish: true` and does not delete media |
| GET | `/campaigns/{id}/runs` | Campaign run history |
| GET | `/campaign-templates` | Built-in stewardship campaign templates |
| POST | `/campaign-templates/{id}/create` | Create an editable campaign from a built-in template |
| GET | `/providers` | Metadata provider health and attribution |
| GET | `/provider-settings` | Redacted provider credential and base URL settings |
| PUT | `/provider-settings/{provider}` | Update provider base URL, API key, or clear stored key |
| GET | `/integration-settings` | Redacted Jellyfin, Plex, Emby, and Tautulli connection settings |
| PUT | `/integration-settings/{id}` | Update media-server or analytics base URL, API key/token, auto-sync settings, or clear stored key |
| GET | `/integrations` | Media-server and AI integration status |
| POST | `/integrations/{id}/refresh` | Request a Jellyfin, Plex, or Emby library refresh |
| POST | `/integrations/{id}/sync` | Queue a background Jellyfin, Plex, Emby, or Tautulli sync job |
| GET | `/integrations/{id}/sync` | Active or latest media-server or Tautulli sync job |
| GET | `/integrations/{id}/diagnostics` | Imported inventory/activity proof summary, warnings, storage verification, and top suggestions |
| GET | `/integrations/{id}/items` | Imported media-server items; supports `?unmapped=true` |
| GET | `/activity/rollups` | Normalized media activity rollups; supports `?serverId=` |
| GET | `/request-sources` | Redacted request-source settings |
| PUT | `/request-sources/{id}` | Update Seerr-compatible request source URL, API key, and enabled state |
| POST | `/request-sources/{id}/sync` | Import Seerr-compatible request signals |
| GET | `/request-signals` | Imported request signals; supports `?sourceId=` |
| GET | `/storage-ledger` | Storage ledger separating verified savings, estimates, blocked bytes, protected bytes, accepted manual bytes, and requested media |
| GET | `/notifications` | Unread stewardship notifications; supports `?includeRead=true` |
| POST | `/notifications/{id}/read` | Mark a stewardship notification read |
| GET | `/protection-requests` | Protection requests; supports `?status=` |
| POST | `/protection-requests` | Create a protection request |
| POST | `/protection-requests/{id}/approve` | Approve a protection request and protect the linked recommendation when present |
| POST | `/protection-requests/{id}/decline` | Decline a protection request |
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

Recommendation evidence is intentionally verbose. Clients should display storage verification separately from confidence and separate estimated savings from verified savings. Evidence responses include `storage.estimatedSavingsBytes`, `storage.verifiedSavingsBytes`, `storage.verification`, `storage.certainty`, activity rollups, source rule, and proof points. Activity recommendation evidence may include `confidenceBasis`, which explains the deterministic inputs behind the displayed confidence percentage.

Storage certainty labels have fixed meanings:

- `server_reported` / `estimate`: Jellyfin/Plex/Emby reports this path and size. Mediarr has not verified it on disk.
- `path_mapped` / `mapped_estimate`: Mediarr translated the server path to a local mount, but size still needs local confirmation.
- `local_verified` / `verified`: Mediarr found the file on a read-only mount and confirmed the size.
- `unmapped`: Mediarr cannot connect the server path to a local file path yet.

Recommendation actions are suggest-only. `accept-manual` records that an administrator accepted the suggestion for manual action; it does not delete, move, quarantine, or overwrite media files.

Campaign actions are also suggest-only. Campaign simulation returns matched items, suppressed items, estimated savings, verified savings, and confidence ranges without writing recommendations. Campaign runs create recommendations with action `review_campaign_match`, source `campaign:{id}`, `destructive=false`, and evidence fields such as `campaignId`, `campaignName`, `campaignRunId`, `matchedRules`, `estimatedSavingsBytes`, and `verifiedSavingsBytes`. Re-running a campaign replaces only open recommendations from the same campaign source and preserves ignored, protected, and accepted-for-manual decisions.

Campaign what-if responses add the operational conflicts an admin needs before acting: `requestConflicts`, `protectionConflicts`, and `blockedUnmapped` counts, plus estimated and verified bytes. Publication preview responses are dry-runs by default and include every candidate with `publishable` and `blockedReason`. The publish endpoint currently writes Jellyfin collections only and requires `confirmPublish: true`; Plex plans can be previewed but are blocked from write execution until a Plex collection adapter is added. Publication still performs no delete, unmonitor, move, or quarantine action.

Request-source sync is read-only. Seerr-compatible imports store normalized request state, availability, requester, provider IDs, and timestamps. Tautulli sync is also read-only; it enriches Plex rollups after Plex inventory has been imported. Raw provider payloads are not stored by default.

The storage ledger intentionally separates trustworthy reclaimable space from estimates:

- `locallyVerifiedBytes`: Mediarr found the file on a read-only mount and confirmed size.
- `mappedEstimateBytes`: the server path maps to a Mediarr-visible prefix but local size still needs confirmation.
- `serverReportedBytes`: the media server reported size, with no local proof yet.
- `blockedUnmappedBytes`: path evidence is missing and the item should not be treated as cleanup-ready.
- `protectedBytes`: protected media removed from the open queue.
- `acceptedManualBytes`: suggestions accepted for human action.
- `requestedMediaBytes`: imported request signals, used as context rather than cleanup savings.

Provider and media-server API calls use bounded retry behavior for `429` and `5xx` responses. `Retry-After` is honored when present, capped to avoid wedging background jobs indefinitely.

Integration settings include `autoSyncEnabled` and `autoSyncIntervalMinutes`. Auto-sync defaults to enabled at 360 minutes. Saving a valid URL and API key/token queues the first sync immediately, and the backend scheduler keeps due integrations fresh while avoiding duplicate active jobs.

Integration diagnostics reuse the same evidence model as the live Jellyfin acceptance suite. They summarize persisted imported data, not raw provider payloads: movie/series/episode counts, file counts, server-reported bytes, locally verified bytes, unmapped files, activity rollups, warning messages, and top recommendations.

Support bundles package redacted provider and integration settings, path mappings, recent jobs, recommendation state, ingestion diagnostics, and safety proof into a zip archive. They intentionally exclude media files, the raw SQLite database, raw provider payloads, and API keys. Download paths only accept generated `mediarr-support-*.zip` archive names and reject path traversal. The archive may still include media titles and paths because those are necessary to troubleshoot ingestion evidence.

Backup downloads and restores only accept generated `mediarr-*.zip` archive names or existing paths under `/config/backups`. Non-dry-run restore requests must send `confirmRestore: true`; Mediarr creates a pre-restore backup before replacing files under `/config`.
