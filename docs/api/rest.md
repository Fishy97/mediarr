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
| GET | `/catalog` | Persisted media catalog |
| PUT | `/catalog/{mediaFileId}/correction` | Apply a user-approved metadata correction |
| DELETE | `/catalog/{mediaFileId}/correction` | Clear a metadata correction |
| GET | `/scans` | Scan history |
| POST | `/scans` | Run a scan |
| GET | `/recommendations` | Open cleanup review items |
| POST | `/recommendations/{id}/ignore` | Hide an advisory recommendation |
| POST | `/recommendations/{id}/restore` | Restore an ignored advisory recommendation |
| GET | `/providers` | Metadata provider health and attribution |
| GET | `/provider-settings` | Redacted provider credential and base URL settings |
| PUT | `/provider-settings/{provider}` | Update provider base URL, API key, or clear stored key |
| GET | `/integrations` | Media-server and AI integration status |
| POST | `/integrations/{id}/refresh` | Request a Jellyfin, Plex, or Emby library refresh |
| POST | `/integrations/{id}/sync` | Import Jellyfin or Plex inventory and activity |
| GET | `/integrations/{id}/sync` | Latest media-server sync job |
| GET | `/integrations/{id}/items` | Imported media-server items; supports `?unmapped=true` |
| GET | `/activity/rollups` | Normalized media activity rollups; supports `?serverId=` |
| GET | `/path-mappings` | Path prefix mappings used to resolve server paths to Mediarr paths |
| POST | `/path-mappings` | Create a path mapping |
| PUT | `/path-mappings/{id}` | Update a path mapping |
| DELETE | `/path-mappings/{id}` | Delete a path mapping |
| GET | `/ai/status` | Optional local AI health and model availability |
| POST | `/backups` | Create a `/config` backup |
| POST | `/backups/restore` | Inspect or restore a `/config` backup |
| GET | `/audit` | Audit log lines |

No media file deletion route is provided.

`refresh` and `sync` are intentionally different. `refresh` tells a media server to refresh its own library. `sync` pulls inventory, file paths, file sizes, and activity signals into Mediarr for suggest-only cleanup recommendations.
