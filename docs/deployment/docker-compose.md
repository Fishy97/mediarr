# Docker Compose Deployment

Media Steward is designed for NAS and home-server deployments.

1. Copy `.env.example` to `.env`.
2. Set `MOVIES_DIR`, `SERIES_DIR`, and `ANIME_DIR`.
3. Run `docker compose up --build`.
4. Open `http://localhost:8080`.

Media mounts are read-only by default:

```yaml
- ${MOVIES_DIR:-./fixtures/media/movies}:/media/movies:ro
```

Durable application state is stored in `./config`, mounted at `/config` in the container.

