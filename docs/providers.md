# Provider Guide

Mediarr keeps provider output advisory until a user applies a catalog correction. User corrections always win over scan guesses, provider candidates, and local AI rationale.

## Supported Providers

| Provider | Current V1 Use | Configuration |
| --- | --- | --- |
| TMDb | Health checks and movie/series candidate search client | `MEDIARR_TMDB_TOKEN` or Integrations screen |
| AniList | Anime provider health surface | no key required |
| TheTVDB | Health checks for configured API credentials | `MEDIARR_THETVDB_API_KEY` or Integrations screen |
| OpenSubtitles | Health checks for configured API credentials | `MEDIARR_OPENSUBTITLES_API_KEY` or Integrations screen |
| Local sidecar | Local NFO/artwork path awareness | mounted media libraries |

## Credential Handling

Provider credentials are stored in `/config/mediarr.db` when entered through the UI. API responses only return `apiKeyConfigured` and `apiKeyLast4`; full secrets are not returned by the REST API.

Backups include provider settings, so protect `/config/backups` the same way you protect the live config directory.

## Attribution

Provider health responses include attribution text. Any future provider-backed metadata display must preserve provider attribution and rate-limit behavior.

## Failure Behavior

Provider outages, invalid credentials, and rate limits are shown as degraded health states. They do not corrupt the catalog and do not block scanning local media.
