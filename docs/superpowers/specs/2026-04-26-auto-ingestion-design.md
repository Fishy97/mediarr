# Automatic Media Server Ingestion Design

## Goal

Mediarr should pull media-server inventory and activity automatically after a user connects Jellyfin, Plex, or Emby. Users should not have to remember to press Sync before recommendations become useful.

## Behavior

- Auto-sync is enabled by default for every configured media-server integration.
- Saving a valid server URL and API key/token immediately queues the first sync.
- On app startup, Mediarr checks configured integrations and queues sync jobs that have never run or are stale.
- A backend scheduler checks for due integrations every minute.
- The default sync interval is 360 minutes.
- Users can disable auto-sync per integration or change the interval in the Integrations UI.
- Manual Sync remains available as a force-refresh control.
- The scheduler must not queue duplicate jobs when a sync is already queued or running.

## Safety

Automatic ingestion remains read-only. It only calls media-server APIs, persists normalized inventory/activity, regenerates advisory recommendations, and records audit events. It never deletes, moves, or modifies media.

## Data Model

`integration_settings` stores:

- `auto_sync_enabled`
- `auto_sync_interval_minutes`

Existing settings default to enabled with a 360-minute interval during migration.

## Scheduling

The API server exposes an internal `QueueDueAutoSyncs` method for startup and scheduler use. It:

1. Loads stored integration settings and merged environment settings.
2. Skips unconfigured integrations.
3. Skips disabled integrations.
4. Skips integrations with an active sync job.
5. Skips integrations whose last successful sync is still fresh.
6. Queues due sync jobs through the same job runner used by manual sync.

Plex continues using its watch-history cursor during automatic sync.

## UI

Each media-server card shows:

- whether auto-sync is enabled
- the configured interval
- the computed next sync time when a previous sync exists
- the existing progress panel while sync is running

The connection form includes an auto-sync checkbox and interval input.

## Testing

- Database tests cover default auto-sync values, updates, interval persistence, and credential clearing preserving auto-sync settings.
- API tests cover immediate sync after saving credentials, startup due-sync queueing, disabled integrations, and duplicate active job protection.
- Frontend tests cover the new auto-sync fields in integration settings requests.
