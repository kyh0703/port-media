---
feature: postgresql-database
status: plan_ready
created_at: 2026-06-23T13:49:14+09:00
---

# PostgreSQL Database

## Goal

Move the repository's durable SQL database from SQLite to PostgreSQL while leaving Redis responsibilities unchanged.

## Context / Inputs
- Source docs:
  - docs/STATE.md
  - docs/ROADMAP.md
  - docs/ARCHITECTURE.md
  - docs/v1/completed/pion-realtime-sfu-session-server.md
- Existing system facts:
  - SQLite is currently configured through `DB_PATH` and `DATABASE_URL`.
  - Bun uses `sqlitedialect` and `sqliteshim`.
  - Schema bootstrap is implemented in `internal/adapter/out/persistence/sqlite_schema.go`.
  - Redis still owns media tokens, live media session state, media server heartbeat, and conversation event publishing.
- User brief:
  - "우리 지금 디비쓰고있는거 postgresql로 바꾸자"

## Plan Handoff

### Scope for Planning
- Replace SQLite SQL connection setup with PostgreSQL connection setup.
- Replace SQLite dialect/driver dependencies with PostgreSQL equivalents.
- Rename SQLite-specific schema and health labels to PostgreSQL-specific names.
- Update config defaults and examples to use `DATABASE_URL` for PostgreSQL.
- Update focused tests that currently assume SQLite.

### Success Criteria
- Application SQL DB opens through a PostgreSQL DSN.
- Room schema creation and repository upsert still work against Bun.
- SQLite-specific code, env defaults, and health check names are removed or renamed.
- Redis behavior remains unchanged.
- `rtk go test ./...` passes.

### Non-Goals
- Do not migrate existing SQLite data files.
- Do not replace Redis token/live-state/event stream storage.
- Do not introduce a new migration framework in this change.
- Do not change session API behavior.

### Open Questions
- Production PostgreSQL DSN, credentials, and provisioning are outside this repository and must be supplied through `DATABASE_URL`.

### Suggested Validation
- `rtk go test ./...`
- `rtk go build -o /tmp/portfoilo-media-postgres-check ./cmd/api`

### Parallelization Hints
- Candidate write boundaries:
  - `configs/`
  - `internal/adapter/out/persistence/`
  - `internal/pkg/health/`
  - `.env.example`, `go.mod`, `go.sum`
- Shared files to avoid touching in parallel:
  - `go.mod`
  - `go.sum`
  - `internal/adapter/out/persistence/module wiring`
- Likely sequential dependencies:
  - Update config defaults before DB constructor tests.
  - Update persistence constructor/schema before repository and health tests.
