# Media Steward

Media Steward is a self-hosted control plane for movies, series, and anime that already exist on disk. It scans read-only media mounts, builds a catalog, enriches metadata, syncs with playback servers, and creates safe review recommendations such as duplicate cleanup and oversized-file review.

It deliberately does not search for, download, torrent, index, or acquire media.

## Current Capabilities

- Docker Compose-first deployment on port `8080`
- Go backend with SQLite, WAL mode, audit logging, backup creation, scanner, parser, and recommendation engine
- React/TypeScript frontend for dashboard, libraries, catalog, review queue, integrations, provider health, settings, and backups
- Read-only media mounts by default
- Suggest-only cleanup recommendations with affected paths, confidence, source, and recoverable storage
- Provider health surfaces for TMDb, AniList, TheTVDB, OpenSubtitles, and local sidecars
- Integration surfaces for Jellyfin, Plex, Emby, and optional local Ollama

## Quick Start

```bash
cp .env.example .env
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080).

Set `MOVIES_DIR`, `SERIES_DIR`, and `ANIME_DIR` in `.env` to point at your media folders. The compose file mounts them read-only.

## Local Development

```bash
npm --prefix frontend install
npm --prefix frontend run dev
```

Backend development requires Go 1.23:

```bash
cd backend
go test ./...
go run ./cmd/medi-steward
```

## Safety Model

Media Steward does not expose a permanent delete endpoint. Recommendations are review items only. A future quarantine workflow can be added behind explicit write-path configuration, but the default filesystem posture is read-only media and writable `/config`.

## License

AGPL-3.0-or-later. See [LICENSE](./LICENSE).

