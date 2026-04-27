# Stewardship Campaigns Design

## Summary

Mediarr will add production-grade stewardship campaigns: saved, auditable, rule-backed review workflows that let an admin define which media should be reviewed, preview the impact, and generate evidence-first recommendations without enabling destructive automation.

This closes the main gap with Maintainerr-style tools while preserving Mediarr's differentiator. Maintainerr is strong at rules, collections, countdowns, and cleanup workflows. Mediarr should become stronger at proof, simulation, path certainty, and safe review operations before any future media-server writeback or deletion workflow exists.

## Scope

Included in this phase:

- Saved campaign definitions with name, description, enabled state, schedule metadata, target media kinds, target libraries, and rule groups.
- A deterministic rule engine over normalized media-server inventory/activity plus existing recommendation evidence.
- Campaign simulation that shows matched, suppressed, and excluded items before a campaign generates recommendations.
- Campaign recommendation generation that writes normal non-destructive recommendations with campaign evidence attached.
- UI surfaces for campaign list, editor, simulation results, and run history.
- API routes for CRUD, simulation, run, and history.
- Documentation and tests proving campaigns cannot delete media.

Excluded from this phase:

- Publishing Plex/Jellyfin collections.
- Countdown collection sync.
- Delete, unmonitor, remove-from-request-service, quarantine, or file mutation.
- Overseerr/Jellyseerr and Tautulli ingestion. Those can be follow-up phases once campaigns exist.

## Product Principles

- Campaigns must be explainable before they are runnable.
- Simulation is first-class; admins should see impact before any recommendation state changes.
- A campaign can only create or update suggest-only recommendations.
- Suppression is as important as matching. Favorites, low confidence, unmapped paths, recent activity, and protected items must be visible.
- Rules should be powerful but not a DSL maze. V1 uses a typed JSON rule model with UI controls for common fields.

## Data Model

### Campaign

- `id`
- `name`
- `description`
- `enabled`
- `targetKinds`: movie, series, anime
- `targetLibraryNames`
- `rules`: array of typed rule conditions
- `requireAllRules`
- `minimumConfidence`
- `minimumStorageBytes`
- `createdAt`
- `updatedAt`
- `lastRunAt`

### Rule Condition

Supported fields:

- `kind`
- `libraryName`
- `verification`
- `storageBytes`
- `estimatedSavingsBytes`
- `verifiedSavingsBytes`
- `lastPlayedDays`
- `addedDays`
- `playCount`
- `uniqueUsers`
- `favoriteCount`
- `confidence`

Supported operators:

- `equals`
- `not_equals`
- `in`
- `not_in`
- `greater_than`
- `greater_or_equal`
- `less_than`
- `less_or_equal`
- `is_empty`
- `is_not_empty`

### Campaign Result

Simulation and run results should expose:

- `matched`
- `suppressed`
- `totalEstimatedSavingsBytes`
- `totalVerifiedSavingsBytes`
- `confidenceMin`
- `confidenceAverage`
- `confidenceMax`
- `items`: compact evidence rows containing title, kind, path count, storage certainty, confidence, reasons, and affected paths

## Backend Architecture

Create a focused `backend/internal/campaigns` package. It owns campaign types, rule evaluation, simulation output, suppression reasons, and recommendation conversion.

The database layer persists campaign definitions and run records in SQLite:

- `stewardship_campaigns`
- `stewardship_campaign_runs`

Rules are stored as JSON because rule definitions need versioning but are not yet query-critical. Campaign recommendations attach campaign evidence to the existing recommendation evidence JSON.

The API layer exposes:

- `GET /api/v1/campaigns`
- `POST /api/v1/campaigns`
- `GET /api/v1/campaigns/{id}`
- `PUT /api/v1/campaigns/{id}`
- `DELETE /api/v1/campaigns/{id}`
- `POST /api/v1/campaigns/{id}/simulate`
- `POST /api/v1/campaigns/{id}/run`
- `GET /api/v1/campaigns/{id}/runs`

`DELETE /api/v1/campaigns/{id}` deletes only the campaign definition. It must never delete media files or recommendations for accepted manual actions.

## Frontend Architecture

Add a `Campaigns` view to the existing app shell. Keep the first UI production-focused rather than decorative:

- Campaign list with enabled state, last run, matched count, estimated savings, verified savings.
- Campaign editor with compact rule rows, target kinds, target libraries, confidence/storage thresholds.
- Simulation panel with summary cards and matched/suppressed tables.
- Run button that generates recommendations and shows a run audit row.

The Review Queue should show campaign evidence when present: campaign name, matched rules, suppression status, and run ID.

## Recommendation Behavior

Campaign-generated recommendations use existing recommendation actions where possible:

- `review_campaign_match` is added for campaign-specific review items that do not fit duplicate, oversized, inactive, or never-watched actions.

All campaign recommendations must have:

- `destructive=false`
- `source=campaign:<campaign-id>`
- `Evidence["campaignId"]`
- `Evidence["campaignName"]`
- `Evidence["campaignRunId"]`
- `Evidence["matchedRules"]`
- `Evidence["suppressionReasons"]`

Campaign generation should not erase unrelated recommendations. It should replace only active, non-protected, non-ignored recommendations for the same campaign source.

## Safety

- No permanent delete endpoint is introduced.
- Campaign runs never call media-server delete APIs.
- Campaign deletes remove campaign config only.
- All media paths remain read-only in Docker Compose.
- Audit events record campaign create, update, delete, simulate, and run.

## Testing

Backend:

- Rule engine tests for each operator.
- Simulation tests for matched and suppressed items.
- Persistence tests for campaign CRUD and run records.
- API tests for CRUD, simulate, run, and no-delete invariant.
- Recommendation tests proving campaign recommendations are non-destructive and attach evidence.

Frontend:

- API client tests for campaign endpoints.
- Formatting/helper tests for campaign summaries.
- Production build and existing tests.

Release gate:

- `make ci`
- `docker compose config --quiet`
- `docker compose --profile ai config --quiet`
- `docker compose --profile ai up -d --build`
- `curl -fsS http://localhost:8080/api/v1/health`
- `scripts/verify-no-delete.sh`

## Follow-Up Phases

1. Optional Jellyfin/Plex collection publishing for campaign result sets.
2. Tautulli activity provider.
3. Overseerr/Jellyseerr request-aware protection.
4. Optional quarantine workflow with explicit writable path, dry-run, undo window, and audit trail.
