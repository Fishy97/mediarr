# Threat Model

Mediarr is a self-hosted stewardship service for already-downloaded media. It is designed for trusted home servers, homelabs, and private VMs, not for anonymous public multi-tenant use.

## Security Goals

- Keep media mounts read-only by default.
- Keep cleanup recommendations advisory and non-destructive.
- Protect `/config`, because it contains the SQLite database, sessions, provider settings, integration settings, audit logs, backups, and user corrections.
- Avoid returning provider or media-server secrets to the browser.
- Make support exports useful for operations without including API keys, media files, raw provider payloads, or the raw database.
- Make long-running scan and sync behavior observable, cancelable, and recoverable.
- Preserve user decisions such as protected or ignored recommendations across regeneration.

## Non-Goals

- Mediarr does not provide torrent, Usenet, indexer, downloader, or acquisition automation.
- Mediarr does not permanently delete, move, or quarantine media in the current release.
- Mediarr does not replace a hardened identity provider for internet-facing deployments.
- Mediarr does not store raw Jellyfin, Plex, or Emby provider payloads as normal catalog state.

## Trust Boundaries

| Boundary | Trusted Side | Untrusted Or Less Trusted Side | Controls |
| --- | --- | --- | --- |
| Browser to Mediarr API | Authenticated admin session | Network clients and browsers | First-run admin setup, HTTP-only session cookie, bearer automation token, reverse proxy TLS guidance |
| Mediarr to media filesystem | `/config` write path | `/media/*` mounts | Docker Compose read-only media mounts, no delete endpoint, CI no-delete gate |
| Mediarr to media servers | Mediarr integration client | Jellyfin, Plex, Emby API responses | Redacted credentials, bounded retries, path mapping verification, server-reported evidence labels |
| Mediarr to metadata providers | Provider adapters | Provider API responses and outages | Provider cache, health status, graceful failure, user corrections override provider guesses |
| Mediarr to local AI | Deterministic recommendation engine | Ollama model output | JSON validation, confidence score, advisory-only storage, no destructive authority |
| Backup restore | Admin-approved `/config` archive | Zip file contents | Dry-run inspect, pre-restore backup, zip-slip path validation |
| Support bundle export | Redacted operational snapshot | Support recipients and external issue trackers | Secrets redacted, raw DB and media files excluded, operators warned that media titles and paths can be present |

## Key Risks And Mitigations

### Media Deletion Or Data Loss

Risk: A bug or future feature could expose destructive media operations.

Mitigations:

- No permanent media delete route exists.
- `DELETE /api/v1/media/files/*` is explicitly blocked.
- Compose mounts `/media/movies`, `/media/series`, and `/media/anime` read-only.
- `scripts/verify-no-delete.sh` scans for filesystem deletion primitives and enforces read-only media mounts in CI.
- Recommendations can be protected, ignored, or accepted for manual action, but not executed as destructive actions.

### Secret Disclosure

Risk: Provider tokens, media-server tokens, sessions, or backups leak through API responses or logs.

Mitigations:

- Provider and integration settings return only configured status and key suffixes.
- Tokens are stored server-side in `/config/mediarr.db`.
- Job telemetry uses titles and basenames instead of full raw paths when possible.
- Support bundles redact stored provider and media-server API keys and exclude raw provider payloads, media files, and the raw SQLite database.
- Admin sessions use HTTP-only cookies, SameSite Lax, and Secure when the request is HTTPS or arrives through an HTTPS reverse proxy.
- Operators are instructed to protect `/config`, `/config/backups`, and `/config/support`.

### Incorrect Recommendations

Risk: Mediarr recommends removing media based on partial, stale, or unmapped data.

Mitigations:

- Recommendation evidence separates `local_verified`, `path_mapped`, `server_reported`, and `unmapped`.
- Unmapped items stay in a review queue and are blocked from cleanup recommendations.
- Path mappings can be verified against local files before confidence is raised.
- Favorites, recent activity, protected recommendations, and user overrides suppress unsafe suggestions.
- AI output is advisory and cannot invent authoritative statistics or override deterministic evidence.

### Provider Outages And Rate Limits

Risk: Jellyfin, Plex, Emby, or metadata providers are unavailable or rate limited.

Mitigations:

- Provider calls use bounded retries for `429` and `5xx` responses.
- `Retry-After` is honored with a short cap so jobs do not stall indefinitely.
- Background jobs expose phase, current label, counters, recent events, errors, cancel, retry, and stale recovery.
- Failed providers do not corrupt existing catalog state.

### Public Exposure

Risk: An admin exposes Mediarr directly to the internet.

Mitigations:

- Documentation recommends running behind a reverse proxy with TLS.
- First-run setup creates an admin account; there is no default password.
- Private API routes require authentication when auth is configured.
- The project does not claim public multi-tenant hardening.

## Release Verification

Public releases should pass:

- backend tests and vet
- frontend tests and build
- Docker Compose no-AI and AI profile config validation
- Docker image build and smoke health check
- `scripts/verify-no-delete.sh`
- GHCR image publication with GitHub artifact provenance attestation

Consumers can verify release image provenance with GitHub artifact attestations for the published GHCR image.
