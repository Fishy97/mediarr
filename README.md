# Mediarr

Mediarr is a self-hosted control plane for movies, series, and anime that already exist on disk. It scans read-only media mounts, builds a catalog, enriches metadata, syncs with playback servers, and creates safe review recommendations such as duplicate cleanup and oversized-file review.

It deliberately does not search for, download, torrent, index, or acquire media.

## Project Status

Mediarr 1.1 is a Docker-hosted library scanner, catalog, and review dashboard for already-downloaded media. It can:

- scan movie, series, and anime folders mounted read-only
- parse common movie, series, and anime filename patterns
- persist catalog rows, user metadata corrections, subtitle sidecars, and review recommendations in SQLite
- detect duplicate catalog items and estimate recoverable space
- create, inspect, and restore `/config` backups with a pre-restore backup
- protect the API with first-run admin setup, password login, sessions, and bearer-token automation
- configure provider credentials without returning secrets through the API
- request Jellyfin, Plex, and Emby library refreshes as sync targets
- sync Jellyfin and Plex inventory, file evidence, and user activity into a normalized activity model
- show durable background-job telemetry for filesystem scans and Jellyfin/Plex syncs, including phase, counters, current item, and recent events
- create activity-aware cleanup recommendations for inactive and never-watched media
- attach optional local AI rationales to deterministic recommendations when the Ollama sidecar is enabled
- expose a web UI and REST API

Mediarr remains deliberately conservative: it does not delete media, does not download media, and does not treat provider or AI output as catalog truth unless a user applies a correction.

## Current Capabilities

- Docker Compose-first deployment on port `8080`
- Go backend with SQLite, WAL mode, audit logging, backup creation, scanner, parser, and recommendation engine
- React/TypeScript frontend for dashboard, libraries, catalog, review queue, integrations, provider health, settings, and backups
- Read-only media mounts by default
- Suggest-only cleanup recommendations with affected paths, confidence, source, and recoverable storage
- Provider health and credential surfaces for TMDb, AniList, TheTVDB, OpenSubtitles, and local sidecars
- Integration status, refresh actions, and inventory/activity sync for Jellyfin and Plex
- Path evidence labels that distinguish locally verified, path-mapped, and server-reported savings
- Catalog correction workflow with user overrides taking precedence over scan guesses

## Quick Start

For a local test with the included fixture library:

```bash
cp .env.example .env
docker compose up --build -d
```

Open [http://localhost:8080](http://localhost:8080).

Mediarr ships with no default password. On a fresh `/config` volume, the first browser visit is forced through **First run setup**, where you create the local admin account. The setup endpoint is disabled after the first admin exists.

### Launch Modes

Run Mediarr without the optional local AI sidecar:

```bash
docker compose up -d
```

Run Mediarr with the optional AI sidecar:

```bash
docker compose --profile ai up -d
```

The AI sidecar is optional. When enabled, Compose starts Ollama and a one-shot model initializer that pulls `MEDIARR_AI_MODEL` into the Docker volume. Mediarr remains fully usable without it, and AI output is treated as advisory rather than catalog truth.

Set `MOVIES_DIR`, `SERIES_DIR`, and `ANIME_DIR` in `.env` to point at your media folders. The compose file mounts them read-only.

For a full Ubuntu server walkthrough, see [Docker Compose Deployment Guide](docs/deployment/docker-compose.md).
Provider behavior is documented in [Provider Guide](docs/providers.md), and optional local AI behavior is documented in [Local AI Guide](docs/ai.md).

Run a scan from the UI or with:

```bash
curl -X POST http://localhost:8080/api/v1/scans
```

Scan and sync requests return a background job immediately. Track progress from the dashboard or with:

```bash
curl "http://localhost:8080/api/v1/jobs?active=true"
curl http://localhost:8080/api/v1/jobs/<job-id>
```

Connect Jellyfin or Plex from **Integrations** in the web UI by entering the server URL and API key/token. Mediarr stores those credentials in `/config/mediarr.db` and only returns redacted key status to the browser.

You can also configure integrations through the REST API:

```bash
curl -X PUT http://localhost:8080/api/v1/integration-settings/jellyfin \
  -H "Content-Type: application/json" \
  -d '{"baseUrl":"http://jellyfin:8096","apiKey":"your-jellyfin-api-key"}'

curl -X PUT http://localhost:8080/api/v1/integration-settings/plex \
  -H "Content-Type: application/json" \
  -d '{"baseUrl":"http://plex:32400","apiKey":"your-plex-token"}'
```

Then sync from the UI Integrations screen, or with:

```bash
curl -X POST http://localhost:8080/api/v1/integrations/jellyfin/sync
curl -X POST http://localhost:8080/api/v1/integrations/plex/sync
```

`refresh` asks the media server to rescan its own libraries. `sync` imports inventory and activity into Mediarr so it can create cleanup suggestions. Activity data can reveal household viewing behavior, so Mediarr stores only the normalized fields needed for recommendations and never returns media-server tokens through the API.

## Ubuntu Server Deployment

Most users should run Mediarr on an Ubuntu VM, NAS, or home server with Docker Compose.

### 1. Install Docker Engine

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

Log out and back in so your shell can use Docker without `sudo`.

### 2. Clone And Configure

```bash
git clone https://github.com/Fishy97/mediarr.git
cd mediarr
cp .env.example .env
id -u
id -g
```

Edit `.env`:

```env
PUID=1000
PGID=1000
MOVIES_DIR=/srv/media/movies
SERIES_DIR=/srv/media/series
ANIME_DIR=/srv/media/anime
MEDIARR_ADMIN_TOKEN=change-this-long-random-token
```

Create the config directory and make it writable by the configured user:

```bash
mkdir -p config
sudo chown -R "$(id -u):$(id -g)" config
```

### 3. Start The App

```bash
docker compose up --build -d
docker compose ps
```

Open `http://<server-ip>:8080`.

The media folders are mounted read-only. Mediarr writes durable state only to `./config`.

### 4. Upgrade

```bash
git pull
docker compose up --build -d
```

### 5. Back Up

Use the Settings screen or:

```bash
curl -X POST http://localhost:8080/api/v1/backups
```

Backups are written under `./config/backups`.

Restore dry-run:

```bash
curl -X POST http://localhost:8080/api/v1/backups/restore \
  -H 'Content-Type: application/json' \
  -d '{"path":"/config/backups/mediarr-example.zip","dryRun":true}'
```

Restore execution creates a new pre-restore backup first:

```bash
curl -X POST http://localhost:8080/api/v1/backups/restore \
  -H 'Content-Type: application/json' \
  -d '{"path":"/config/backups/mediarr-example.zip","dryRun":false}'
```

## Local Development

```bash
npm --prefix frontend install
npm --prefix frontend run dev
```

Backend development requires Go 1.23:

```bash
cd backend
go test ./...
go run ./cmd/mediarr
```

## Safety Model

Mediarr does not expose a permanent delete endpoint. Recommendations are review items only. A future quarantine workflow can be added behind explicit write-path configuration, but the default filesystem posture is read-only media and writable `/config`.

Recommended production defaults:

- mount media folders read-only
- set `MEDIARR_ADMIN_TOKEN`
- put the app behind a trusted reverse proxy for TLS
- back up `./config` regularly
- use backup restore dry-run before restoring an archive
- do not expose port `8080` directly to the public internet

## License

AGPL-3.0-or-later. See [LICENSE](./LICENSE).
