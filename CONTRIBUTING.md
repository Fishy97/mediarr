# Contributing

Mediaar is intended to be community-run infrastructure for self-hosters.

## Development Rules

- Keep acquisition/download automation out of scope.
- Preserve the suggest-only cleanup safety model.
- Add tests for parser, scanner, recommendation, API, backup, and provider behavior.
- Keep media paths and provider credentials out of logs.
- Treat local metadata and user overrides as the source of truth.

## Commands

```bash
npm --prefix frontend install
npm --prefix frontend run build
cd backend && go test ./...
```

## Provider Policy

Metadata providers must include attribution, caching, rate limiting, and graceful failure behavior. No provider should be required for catalog integrity.

