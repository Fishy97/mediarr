# Recommendation Proof Model

Mediarr recommendations are designed for human review. They explain what Mediarr knows, where the evidence came from, and how certain the storage number is. Mediarr does not delete media files.

## What Mediarr Imports

When Jellyfin, Plex, or Emby is connected, Mediarr imports normalized inventory and activity signals:

- media title, kind, server item ID, library name, path, and size when the server reports them
- parent series identifiers for episodes
- aggregate play count, last played date, watched user count, and favorite/protection count
- path mapping and local verification state

Mediarr stores the normalized fields needed for recommendations. It does not store raw provider payloads during normal operation, and API keys or tokens are never returned to the browser.

## Activity Aggregation

Playback data is aggregated before it is used by rules. The review queue should show signals such as total plays, watched users, last played, and favorite/protection count, not a confusing stream of household profile names.

For example, a series can be suggested when all known episodes are older than the configured threshold and no imported user has watched them. A previously watched movie can be suggested when the latest imported play is older than the inactivity threshold.

## Storage Certainty

Storage values are separated into estimated savings and verified savings.

- **Server estimate:** Jellyfin/Plex/Emby reports this path and size. Mediarr has not verified it on disk.
- **Path mapped estimate:** Mediarr translated the server path to a local mount, but size still needs local confirmation.
- **Locally verified:** Mediarr found the file on a read-only mount and confirmed the size.
- **Unmapped:** Mediarr cannot connect the server path to a local file path yet.

Only locally verified storage should be treated as confirmed disk savings. Server estimates can still be useful, but they are not a guarantee that deleting a file manually will reclaim that exact amount on the host.

## Confidence

Confidence is a deterministic rule score. It combines the media-server match confidence with the storage evidence level. It is not an AI score, and it is not a promise that the media is safe to remove.

Low-confidence, favorite/protected, recently watched, active-series, or unmapped items should be suppressed or kept in a blocking review state instead of appearing as normal cleanup candidates.

## Path Mapping

Media servers often see NAS paths such as `/volume1/media`, while Mediarr sees Docker paths such as `/media`. Path mappings translate one prefix into the other.

After a mapping is saved, Mediarr can verify whether the translated files exist under the read-only mount. If file sizes match, storage proof can be upgraded to locally verified.

## Suggest-Only Safety

Mediarr will not delete this. Accepting marks it for manual action only.

Recommendation actions record intent and audit history. They do not move, delete, quarantine, or overwrite media files. Future writable workflows must remain opt-in, dry-run first, and auditable.

## Acceptance Checklist

For a real Jellyfin NAS sync, a production operator should be able to confirm:

- job telemetry moves through profile discovery, inventory import, activity import, recommendation generation, and completion
- no household profile name is presented as a movie or series title
- a series recommendation is grouped by series, with affected paths collapsed until expanded
- the card shows confidence, estimated savings, verified savings, activity proof, and storage certainty
- server-reported savings are labeled as estimates, while locally verified savings are the only confirmed disk values
- manual acceptance, protection, and ignore actions create audit history without deleting media
