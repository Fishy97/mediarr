# Docker Compose Deployment Guide

Mediarr is designed to be deployed with Docker Compose on Ubuntu servers, NAS boxes, and self-hosted VMs. The repository includes a production-oriented `docker-compose.yml` that builds the app image locally and runs the web UI plus API on port `8080`.

The default deployment is intentionally conservative:

- media folders are mounted read-only
- app state is stored under `./config`
- cleanup recommendations are suggest-only
- there is no default password; first launch requires creating the local admin account in the web UI
- the container has a healthcheck
- the container runs as `PUID:PGID`

## Requirements

- Ubuntu 22.04 LTS or newer
- Docker Engine with the Compose plugin
- Git
- media folders already present on the host
- at least one writable config directory for Mediarr

## 1. Install Docker On Ubuntu

```bash
sudo apt update
sudo apt install -y ca-certificates curl git
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker "$USER"
```

Log out and back in before running Docker without `sudo`.

Verify:

```bash
docker version
docker compose version
```

## 2. Clone The Repository

```bash
git clone https://github.com/Fishy97/mediarr.git
cd mediarr
```

## 3. Prepare Host Folders

Use your real media paths. This example assumes a common `/srv/media` layout:

```bash
sudo mkdir -p /srv/media/movies /srv/media/series /srv/media/anime
mkdir -p config
sudo chown -R "$(id -u):$(id -g)" config
```

Mediarr only needs write access to `./config`. The media mounts should stay read-only.

## 4. Configure `.env`

```bash
cp .env.example .env
nano .env
```

Recommended Ubuntu server values:

```env
MEDIARR_ADMIN_TOKEN=change-this-to-a-long-random-token
MEDIARR_OVERSIZED_BYTES=60000000000
MEDIARR_OLLAMA_URL=http://ollama:11434
MEDIARR_AI_MODEL=qwen3:0.6b
MEDIARR_TMDB_TOKEN=
MEDIARR_THETVDB_API_KEY=
MEDIARR_OPENSUBTITLES_API_KEY=
MEDIARR_JELLYFIN_URL=
MEDIARR_JELLYFIN_API_KEY=
MEDIARR_PLEX_URL=
MEDIARR_PLEX_TOKEN=
MEDIARR_EMBY_URL=
MEDIARR_EMBY_API_KEY=
PUID=1000
PGID=1000
MOVIES_DIR=/srv/media/movies
SERIES_DIR=/srv/media/series
ANIME_DIR=/srv/media/anime
```

Find your `PUID` and `PGID` with:

```bash
id -u
id -g
```

## 5. Start Mediarr

Run Mediarr without the optional local AI sidecar:

```bash
docker compose up -d
```

Run Mediarr with the optional AI sidecar:

```bash
docker compose --profile ai up -d
```

When building directly from a fresh source checkout, add `--build`:

```bash
docker compose up --build -d
```

Check status:

```bash
docker compose ps
docker compose logs -f mediarr
```

Open:

```text
http://<server-ip>:8080
```

For local testing on the server itself:

```bash
curl http://localhost:8080/api/v1/health
```

Expected response:

```json
{"service":"mediarr","status":"ok"}
```

## 6. Run A Scan

Run a scan from the web UI, or use the API:

```bash
curl -X POST http://localhost:8080/api/v1/scans
```

The scan runs as a background job. Track active work from the web UI or with:

```bash
curl "http://localhost:8080/api/v1/jobs?active=true"
curl http://localhost:8080/api/v1/jobs/<job-id>
```

Jobs can be canceled or retried from the web UI. The same controls are available through the API:

```bash
curl -X POST http://localhost:8080/api/v1/jobs/<job-id>/cancel
curl -X POST http://localhost:8080/api/v1/jobs/<job-id>/retry
```

When the job completes, view the catalog:

```bash
curl http://localhost:8080/api/v1/catalog
```

## 7. Sync Jellyfin, Plex, Or Emby Activity

Open the Integrations screen, choose Jellyfin, Plex, or Emby, and enter the media-server URL plus API key/token. Mediarr stores the settings in `/config/mediarr.db` and shows only redacted credential status in the browser.

Environment variables such as `MEDIARR_JELLYFIN_URL`, `MEDIARR_JELLYFIN_API_KEY`, `MEDIARR_PLEX_URL`, `MEDIARR_PLEX_TOKEN`, `MEDIARR_EMBY_URL`, and `MEDIARR_EMBY_API_KEY` are still supported for automation, but they are not required for normal setup.

You can also configure integrations through the API:

```bash
curl -X PUT http://localhost:8080/api/v1/integration-settings/jellyfin \
  -H "Content-Type: application/json" \
  -d '{"baseUrl":"http://jellyfin:8096","apiKey":"your-jellyfin-api-key"}'

curl -X PUT http://localhost:8080/api/v1/integration-settings/plex \
  -H "Content-Type: application/json" \
  -d '{"baseUrl":"http://plex:32400","apiKey":"your-plex-token"}'

curl -X PUT http://localhost:8080/api/v1/integration-settings/emby \
  -H "Content-Type: application/json" \
  -d '{"baseUrl":"http://emby:8096","apiKey":"your-emby-api-key"}'
```

From the API:

```bash
curl -X POST http://localhost:8080/api/v1/integrations/jellyfin/sync
curl -X POST http://localhost:8080/api/v1/integrations/plex/sync
curl -X POST http://localhost:8080/api/v1/integrations/emby/sync
```

Syncs also run as background jobs. The Integrations screen shows the active phase, current item/title, imported counts, unmapped count, retry policy, auto-sync interval, next sync estimate, and recent events while a media server is being read.

Mediarr imports media-server inventory, file paths, file sizes, and activity rollups such as play count and last played date. It uses those signals to create suggest-only cleanup recommendations for inactive or never-watched media.

Auto-sync is enabled by default. Saving a valid Jellyfin, Plex, or Emby connection queues the first sync immediately, and Mediarr checks for due integrations on startup and on a background schedule. The default interval is 6 hours. You can disable auto-sync or change the interval per integration from the UI.

Use path mappings when Jellyfin, Plex, or Emby sees a different path than the Mediarr container. For example, if Plex reports `/mnt/media/movies` but Mediarr sees `/media/movies`, create a mapping from `/mnt/media` to `/media` in the Integrations screen.

After saving a mapping, run **Verify**. Mediarr checks mapped files against the local filesystem and updates the evidence label:

- `local_verified`: mapped path exists and the size matches the server-reported file
- `path_mapped`: prefix mapping resolved the path but the file could not be fully verified
- `server_reported`: no usable local mapping exists yet
- `unmapped`: the item remains in the path review queue and is blocked from cleanup recommendations

The unmapped queue is deliberate. Mediarr should not tell an admin to remove something unless it can explain exactly where the file is and how the savings were calculated.

Activity data can expose household viewing behavior. Keep Mediarr behind authentication, avoid exposing it directly to the public internet, and treat imported activity as local operational data.

Plex syncs store the most recent imported watch-history cursor. Later syncs request history at or after that cursor and preserve prior rollups, so large Plex libraries do not need to rebuild playback activity from scratch on every run.

## 8. Validate A Live Jellyfin NAS Library

Mediarr includes an opt-in live acceptance suite for production Jellyfin validation. It is designed for real libraries, not mocked fixture libraries. The suite reads Jellyfin inventory and user activity, imports the data into an isolated scratch Mediarr database, generates advisory recommendations, and writes reports to `acceptance-reports/`.

It is read-only:

- no Jellyfin library refresh is requested
- no media file is deleted, moved, renamed, or edited
- no normal `/config/mediarr.db` state is changed
- scratch database files stay under `acceptance-reports/` unless `MEDIARR_ACCEPTANCE_STORE_DIR` is set
- API keys are read from the environment and are not written to the report

Run it from a checkout:

```bash
MEDIARR_ACCEPTANCE_JELLYFIN_URL="http://nas:8096" \
MEDIARR_ACCEPTANCE_JELLYFIN_API_KEY="your-jellyfin-api-key" \
MEDIARR_ACCEPTANCE_PATH_MAPS="/volume1/media=/media" \
MEDIARR_ACCEPTANCE_REQUIRE_LOCAL_VERIFY=true \
scripts/acceptance-jellyfin-live.sh
```

The script builds and runs the Mediarr Docker image by default, then executes `/app/mediarr-acceptance` inside a one-off container. This keeps the workflow aligned with normal Ubuntu Compose deployment and does not require Go on the server. Developers can set `MEDIARR_ACCEPTANCE_RUNNER=go` to run the same command with `go run`.

Use `MEDIARR_ACCEPTANCE_PATH_MAPS` when the host running the suite can see the same NAS media read-only under a different path. Multiple mappings are separated with semicolons:

```bash
MEDIARR_ACCEPTANCE_PATH_MAPS="/volume1/movies=/media/movies;/volume1/anime=/media/anime"
```

If you cannot mount the NAS media locally, omit `MEDIARR_ACCEPTANCE_PATH_MAPS` and leave `MEDIARR_ACCEPTANCE_REQUIRE_LOCAL_VERIFY=false`. The report will still validate Jellyfin-reported inventory, file sizes, last-used data, and recommendations, but it will mark reclaimable storage as server-reported rather than locally verified.

Optional controls:

- `MEDIARR_ACCEPTANCE_REDACT_TITLES=true` redacts titles, paths, and current-item progress labels in reports.
- `MEDIARR_ACCEPTANCE_REPORT_DIR=/path/to/reports` changes the report directory.
- `MEDIARR_ACCEPTANCE_TIMEOUT=8h` changes the maximum runtime for very large libraries.

Read the generated Markdown report first. It summarizes movies, series, episodes, anime-library items, file sizes, local verification coverage, unmapped paths, last-used activity rollups, and top suggestions.

## Reverse Proxy And TLS

For internet-adjacent access, place Mediarr behind a trusted reverse proxy such as Caddy, Traefik, or Nginx and terminate TLS there. Do not expose port `8080` directly to the public internet.

Minimum proxy guidance:

- forward `Host`, `X-Forwarded-For`, and `X-Forwarded-Proto`
- keep Mediarr on a private Docker or LAN network
- require HTTPS at the proxy
- keep first-run admin setup private until the admin account exists
- back up `./config` before upgrades

## 8. Backups

Backups include the SQLite database, settings, audit log, provider cache, artwork cache, and user review state.

Create a backup from the UI, or use:

```bash
curl -X POST http://localhost:8080/api/v1/backups
```

Backups are written to:

```text
./config/backups
```

Inspect a backup before restoring it:

```bash
curl -X POST http://localhost:8080/api/v1/backups/restore \
  -H "Authorization: Bearer $MEDIARR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/config/backups/mediarr-example.zip","dryRun":true}'
```

Restore creates a fresh pre-restore backup before replacing files under `/config`:

```bash
curl -X POST http://localhost:8080/api/v1/backups/restore \
  -H "Authorization: Bearer $MEDIARR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"path":"/config/backups/mediarr-example.zip","dryRun":false}'
```

For host-level backups, back up the whole `config` directory:

```bash
tar -czf mediarr-config-$(date +%Y%m%d).tar.gz config
```

## 9. Upgrades

```bash
cd mediarr
git pull
docker compose up --build -d
docker compose ps
```

The app stores durable state in `./config`, so rebuilding the image does not remove catalog data.

## 10. Optional Local AI

Ollama is included as an optional Compose profile. Start Mediarr with local AI enabled:

```bash
docker compose --profile ai up -d
```

The AI profile starts two additional services:

- `ollama`, the local model runtime
- `mediarr-ai-init`, a one-shot initializer that waits for Ollama and pulls `MEDIARR_AI_MODEL`

The default model is `qwen3:0.6b`. Change `MEDIARR_AI_MODEL` in `.env` if you want a different local Ollama model.

Mediarr treats local AI as advisory only. Core scanning and recommendations do not require AI.

The two supported launch modes are:

```bash
# No AI
docker compose up -d

# With AI sidecar
docker compose --profile ai up -d
```

## 11. Reverse Proxy

For a production server, put Mediarr behind a reverse proxy such as Caddy, Nginx Proxy Manager, Traefik, or Nginx. Do not expose port `8080` directly to the public internet.

Minimum reverse proxy expectations:

- terminate HTTPS
- restrict access to trusted users or networks
- forward to `http://127.0.0.1:8080` or `http://<server-ip>:8080`
- keep `MEDIARR_ADMIN_TOKEN` set

## 12. Troubleshooting

### Container Is Not Healthy

```bash
docker compose ps
docker compose logs --tail=200 mediarr
curl http://localhost:8080/api/v1/health
```

### Permission Errors In `/config`

```bash
sudo chown -R "$(id -u):$(id -g)" config
docker compose restart mediarr
```

### Media Folders Are Empty In The UI

Check `.env` paths:

```bash
cat .env
docker compose config
```

Confirm the host folders contain media:

```bash
find "$MOVIES_DIR" -maxdepth 2 -type f | head
```

### Port `8080` Is Already In Use

Change the published port in `docker-compose.yml`:

```yaml
ports:
  - "8090:8080"
```

Then run:

```bash
docker compose up -d
```

Open `http://<server-ip>:8090`.
