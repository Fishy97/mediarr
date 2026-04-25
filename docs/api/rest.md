# REST API

All API routes are rooted at `/api/v1`.

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Service health |
| GET | `/libraries` | Configured libraries |
| POST | `/libraries` | Add a library |
| GET | `/catalog` | Persisted media catalog |
| GET | `/scans` | Scan history |
| POST | `/scans` | Run a scan |
| GET | `/recommendations` | Open cleanup review items |
| GET | `/providers` | Metadata provider health and attribution |
| GET | `/integrations` | Media-server and AI integration status |
| POST | `/backups` | Create a `/config` backup |
| GET | `/audit` | Audit log lines |

No media file deletion route is provided.
