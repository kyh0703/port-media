# PostgreSQL Database

## Goal
- Move durable SQL persistence from SQLite to PostgreSQL and make the change verifiable.

## Summary
- Replaced SQLite `DB_PATH`, `sqliteshim`, `sqlitedialect`, and SQLite PRAGMA setup with PostgreSQL `DATABASE_URL`, `pgx`, and `pgdialect`.
- Replaced SQLite schema bootstrap with PostgreSQL room table/index bootstrap.
- Renamed health dependency reporting from `sqlite` to `postgres`.
- Kept Redis-backed media token, live session state, heartbeat, and conversation event behavior unchanged.
- Did not migrate existing SQLite file data because existing media SQL data is only session/room metadata and is not needed.

## Verification
- PASS: `rtk go test ./configs`
- PASS: `rtk go test ./internal/adapter/out/persistence ./internal/adapter/out/repository`
- PASS: `rtk go test ./internal/pkg/health`
- PASS: `rtk go test ./...`
- PASS: `rtk go build -o /tmp/portfoilo-media-postgres-check ./cmd/api`

## References
- docs/STATE.md
- docs/ROADMAP.md
- docs/ARCHITECTURE.md
- docs/v2/designs/2026-06-23-v2-postgresql-database.md

## Workspace
- Branch: feat/v2-postgresql-database
- Base: main
- Isolation: required
- Merged locally into main.

## Notes
- Redis repositories remain out of scope.
- Production PostgreSQL provisioning and credentials must be supplied through `DATABASE_URL`.
