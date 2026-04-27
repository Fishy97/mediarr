# Stewardship Advantage Design

## Summary

Mediarr should move from evidence-first recommendations into evidence-first stewardship operations. The phase adds request history, richer Plex analytics, verified "Leaving Soon" publishing, templates, notifications, what-if simulation, a storage ledger, and household protection requests while preserving the core safety promise: Mediarr does not delete, move, quarantine, unmonitor, or clear external requests.

This differentiates Mediarr from deletion-first cleanup tools by making every action explainable, reversible at the Mediarr state layer, and separated from destructive media operations.

## Source References

- Overseerr's OpenAPI document supports API-key authentication with `X-Api-Key` and paginated request reads under `/api/v1/request`.
- Jellyseerr exposes the same request shape for `/api/v1/request`, with `take`, `skip`, `filter`, `sort`, and `sortDirection`.
- Tautulli's API uses `/api/v2?apikey=...&cmd=...`; `get_history`, `get_libraries`, and related stats commands are read-only enrichment sources.
- Plex exposes collection mutation endpoints under library collections, including adding items to a collection by collection id.
- Jellyfin's generated SDK exposes collection creation and add-to-collection operations with `name`, `ids`, `parentId`, and `isLocked`.

## Product Scope

### Seerr Request Ingestion

Mediarr will support a read-only request source named `seerr`. It should work with modern Seerr, Jellyseerr, and legacy Overseerr-compatible APIs by calling `/api/v1/request` with `X-Api-Key`. Imported records are normalized into request signals with:

- source kind and source id
- request id
- media type
- TMDb/TVDb/IMDb provider ids when present
- request status
- requesting user display name
- request, approval, availability, and update timestamps

Request signals influence evidence, campaign rules, storage ledger segmentation, and protection suppressions. Mediarr will not approve, decline, delete, retry, or clear requests in this phase.

### Tautulli Enrichment

Mediarr will support a read-only `tautulli` signal source. It will call Tautulli's `/api/v2` API with an API key and import watch-history rows into the normalized activity model. Tautulli data should enrich Plex rollups without replacing Plex inventory:

- match Tautulli rating keys to imported Plex item external ids
- increment play count and unique users
- improve last-played timestamps
- mark evidence source as Tautulli-enriched

The connector must handle pagination and degraded failures without corrupting existing Plex/Jellyfin/Emby data.

### Verified Leaving Soon Publishing

Campaign result sets can be published as "Leaving Soon" collections in Plex or Jellyfin. Publishing is opt-in, explicit, and non-destructive. It requires:

- a campaign run or simulation result
- a target media server
- only items with provider/server item ids available
- a minimum storage verification threshold
- a dry-run preview by default

Actual publish writes only a media-server collection. It does not delete files, remove requests, unmonitor in Radarr/Sonarr, or alter recommendations. Every publish records an audit event and a persisted publication record.

### Campaign Templates

Mediarr will ship built-in campaign templates that create editable campaign records:

- Cold Movies
- Abandoned Series
- Anime Backlog
- Verified High Storage
- Never Watched Large Files
- Requested But Never Watched

Templates are deterministic JSON campaign definitions, not hidden behavior.

### Notifications

Mediarr will add a notification domain for operational events:

- new high-confidence recommendations
- campaign run completed
- collection publish dry-run ready or publish completed
- protection request created
- integration sync failed
- path mapping verification failed or found missing files

The first implementation supports stored in-app notifications and optional generic webhook delivery. Webhook failures must be retried only by explicit admin action or later scheduled work; they must not block ingestion or campaign runs.

### What-If Simulator And Storage Ledger

The storage ledger summarizes Mediarr's storage truth:

- locally verified reclaimable bytes
- mapped estimate bytes
- server-reported estimate bytes
- blocked/unmapped bytes
- protected bytes
- accepted-for-manual-action bytes
- requested media bytes

The what-if simulator evaluates selected campaigns and recommendation states without writing recommendations or external collections. It reports expected matched items, suppressions, verified savings, estimated savings, blocked items, and protection/request risks.

### Protection Request Workflow

Household users or admins can create protection requests for media. A protection request can target a recommendation, a media-server item, or a title/path. Admins can approve, decline, or leave the request pending. Approved protection creates or reinforces recommendation protection; pending requests suppress destructive-looking language and lower cleanup confidence. This remains internal to Mediarr and does not write back to Plex/Jellyfin/Seerr.

## Data Model

Add focused tables instead of overloading recommendation evidence:

- `request_sources`: source configuration metadata for Seerr-compatible services.
- `media_request_signals`: normalized request records.
- `tautulli_sync_jobs`: cursor and job summary for Tautulli enrichment.
- `collection_publications`: dry-run and publish records.
- `notifications`: stored in-app notifications and webhook delivery status.
- `protection_requests`: household/admin protection workflow.

Existing tables remain authoritative for catalog, media-server inventory, rollups, recommendations, campaigns, audit, jobs, and integration settings.

## API Surface

Add routes under `/api/v1`:

- `GET /request-sources`
- `PUT /request-sources/{id}`
- `POST /request-sources/{id}/sync`
- `GET /request-signals`
- `POST /integrations/tautulli/sync`
- `GET /campaign-templates`
- `POST /campaign-templates/{id}/create`
- `POST /campaigns/{id}/what-if`
- `POST /campaigns/{id}/publish-preview`
- `POST /campaigns/{id}/publish`
- `GET /storage-ledger`
- `GET /notifications`
- `POST /notifications/{id}/read`
- `GET /protection-requests`
- `POST /protection-requests`
- `POST /protection-requests/{id}/approve`
- `POST /protection-requests/{id}/decline`

All private routes remain authenticated. Secrets are stored server-side and redacted from API responses.

## UI Scope

Add MediaGlass surfaces without hiding safety detail:

- Integrations: Seerr and Tautulli cards with URL, API key, test/sync, last sync, and source notes.
- Campaigns: template gallery, what-if button, publish preview, and publish action gated by explicit confirmation.
- Dashboard: storage ledger panel with verified versus estimated values.
- Review Queue: request/protection badges and notification-driven status.
- Settings: notifications list and webhook settings.
- Protection Requests: compact review queue with approve/decline actions.

## Safety Rules

- No deletion endpoints.
- No Radarr/Sonarr unmonitor or delete actions in this phase.
- No Seerr request clearing in this phase.
- Collection publish is the only external write, and only after explicit publish.
- Publish preview is the default and is persisted.
- Recommendation confidence must not increase solely because AI or a request source says something.
- Tautulli enriches activity only when matched to known Plex items.
- Failed external source syncs must leave prior good data intact.

## Test Plan

- Unit tests for Seerr response normalization, Tautulli history normalization, collection publication planning, template creation, storage ledger calculation, webhook validation, notification persistence, and protection state transitions.
- API tests for each new route, including auth-compatible behavior through existing server tests.
- Regression tests proving no delete/unmonitor/request-clear endpoints are introduced.
- Frontend tests for new API client methods and key formatting helpers.
- Docker/CI validation remains `make ci`, Compose config checks, Docker build, and no-delete safety scan.
