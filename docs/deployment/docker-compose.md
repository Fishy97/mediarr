# Docker Compose Deployment

Mediarr is designed for NAS and home-server deployments.

1. Copy `.env.example` to `.env`.
2. Set `MOVIES_DIR`, `SERIES_DIR`, and `ANIME_DIR`.
3. Run `docker compose up --build`.
4. Open `http://localhost:8080`.

Media mounts are read-only by default:

```yaml
- ${MOVIES_DIR:-./fixtures/media/movies}:/media/movies:ro
```

Durable application state is stored in `./config`, mounted at `/config` in the container.

The container runs as `PUID:PGID` from `.env`, defaulting to `1000:1000`.
On Linux hosts, make sure the config folder is writable by that user:

```bash
mkdir -p config
chown -R 1000:1000 config
```
