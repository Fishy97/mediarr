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
| GET | `/scans` | Scan history |
| POST | `/scans` | Run a scan |
| GET | `/recommendations` | Open cleanup review items |
| POST | `/recommendations/{id}/ignore` | Hide an advisory recommendation |
| POST | `/recommendations/{id}/restore` | Restore an ignored advisory recommendation |
| GET | `/providers` | Metadata provider health and attribution |
| GET | `/integrations` | Media-server and AI integration status |
| GET | `/ai/status` | Optional local AI health and model availability |
| POST | `/backups` | Create a `/config` backup |
| POST | `/backups/restore` | Inspect or restore a `/config` backup |
| GET | `/audit` | Audit log lines |

No media file deletion route is provided.
