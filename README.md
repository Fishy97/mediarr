# Mediarr

Mediarr is a self-hosted control plane for movies, series, and anime that already exist on disk. It scans read-only media mounts, builds a catalog, enriches metadata, syncs with playback servers, and creates safe review recommendations such as duplicate cleanup and oversized-file review.

It deliberately does not search for, download, torrent, index, or acquire media.

## Project Status

Mediarr is usable today as a Docker-hosted library scanner and review dashboard. It can:

- scan movie, series, and anime folders mounted read-only
- parse common movie, series, and anime filename patterns
- persist catalog rows, subtitle sidecars, and review recommendations in SQLite
- detect duplicate catalog items and estimate recoverable space
- create `/config` backups
- expose a web UI and REST API

The current codebase is a production-grade foundation, not a complete public release. These pieces are intentionally scaffolded but not fully connected yet:

- live TMDb, AniList, TheTVDB, and OpenSubtitles provider calls
- Jellyfin, Plex, and Emby metadata sync
- optional Ollama-backed local AI suggestions
- real authentication setup flow beyond optional bearer-token protection
- broader filename/edition handling for large mixed libraries

## Current Capabilities

- Docker Compose-first deployment on port `8080`
- Go backend with SQLite, WAL mode, audit logging, backup creation, scanner, parser, and recommendation engine
- React/TypeScript frontend for dashboard, libraries, catalog, review queue, integrations, provider health, settings, and backups
- Read-only media mounts by default
- Suggest-only cleanup recommendations with affected paths, confidence, source, and recoverable storage
- Provider health surfaces for TMDb, AniList, TheTVDB, OpenSubtitles, and local sidecars
- Integration surfaces for Jellyfin, Plex, Emby, and optional local Ollama

## Quick Start

For a local test with the included fixture library:

```bash
cp .env.example .env
docker compose up --build -d
```

Open [http://localhost:8080](http://localhost:8080).

Set `MOVIES_DIR`, `SERIES_DIR`, and `ANIME_DIR` in `.env` to point at your media folders. The compose file mounts them read-only.

Run a scan from the UI or with:

```bash
curl -X POST http://localhost:8080/api/v1/scans
```

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
git clone <repo-url> mediarr
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
- do not expose port `8080` directly to the public internet

## License

AGPL-3.0-or-later. See [LICENSE](./LICENSE).
