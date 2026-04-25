# Security Policy

## Supported Versions

Security fixes target the latest released branch and the main development branch. Self-hosters should keep Docker images and source checkouts current.

## Reporting

Please report security issues privately to the maintainers. Do not open public issues for credential leaks, auth bypasses, path traversal, or filesystem write vulnerabilities.

## Baseline Guarantees

- No permanent media deletion endpoint
- First-run admin setup, session auth, and optional bearer-token automation
- Read-only media mounts in Docker Compose
- Config and backups isolated under `/config`
- Provider keys must not be written to logs
- Backup restore rejects archive entries that escape `/config`
