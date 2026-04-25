# Docker Compose Deployment Guide

Mediarr is designed to be deployed with Docker Compose on Ubuntu servers, NAS boxes, and self-hosted VMs. The repository includes a production-oriented `docker-compose.yml` that builds the app image locally and runs the web UI plus API on port `8080`.

The default deployment is intentionally conservative:

- media folders are mounted read-only
- app state is stored under `./config`
- cleanup recommendations are suggest-only
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

Then view the catalog:

```bash
curl http://localhost:8080/api/v1/catalog
```

## 7. Backups

Backups include the SQLite database, settings, audit log, provider cache, artwork cache, and user review state.

Create a backup from the UI, or use:

```bash
curl -X POST http://localhost:8080/api/v1/backups
```

Backups are written to:

```text
./config/backups
```

For host-level backups, back up the whole `config` directory:

```bash
tar -czf mediarr-config-$(date +%Y%m%d).tar.gz config
```

## 8. Upgrades

```bash
cd mediarr
git pull
docker compose up --build -d
docker compose ps
```

The app stores durable state in `./config`, so rebuilding the image does not remove catalog data.

## 9. Optional Local AI

Ollama is included as an optional Compose profile. Start Mediarr with local AI enabled:

```bash
docker compose --profile ai up -d
```

The AI profile starts two additional services:

- `ollama`, the local model runtime
- `mediarr-ai-init`, a one-shot initializer that waits for Ollama and pulls `MEDIARR_AI_MODEL`

The default model is `qwen3:0.6b`. Change `MEDIARR_AI_MODEL` in `.env` if you want a different local Ollama model.

Mediarr treats local AI as advisory only. Core scanning and recommendations do not require AI.

## 10. Reverse Proxy

For a production server, put Mediarr behind a reverse proxy such as Caddy, Nginx Proxy Manager, Traefik, or Nginx. Do not expose port `8080` directly to the public internet.

Minimum reverse proxy expectations:

- terminate HTTPS
- restrict access to trusted users or networks
- forward to `http://127.0.0.1:8080` or `http://<server-ip>:8080`
- keep `MEDIARR_ADMIN_TOKEN` set

## 11. Troubleshooting

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
