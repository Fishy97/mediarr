# Security Policy

## Supported Versions

The project is pre-1.0. Security fixes target the main development branch until formal release channels exist.

## Reporting

Please report security issues privately to the maintainers. Do not open public issues for credential leaks, auth bypasses, path traversal, or filesystem write vulnerabilities.

## Baseline Guarantees

- No permanent media deletion endpoint
- Optional bearer-token protection for API routes
- Read-only media mounts in Docker Compose
- Config and backups isolated under `/config`
- Provider keys must not be written to logs

