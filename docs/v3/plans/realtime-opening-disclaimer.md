# Realtime Opening Disclaimer

## Goal

- Trigger a Korean opening disclaimer from the OpenAI Realtime participant when
  a media call connects.

## References

- docs/STATE.md
- docs/ROADMAP.md
- docs/ARCHITECTURE.md
- docs/v3/designs/2026-06-24-v3-realtime-opening-disclaimer.md
- internal/core/service/session/service.go
- internal/core/service/session/service_test.go

## Workspace

- Branch: feat/v3-realtime-opening-disclaimer
- Base: main
- Isolation: required
- Created by: exec-plan via git-worktree

## Task Graph

### Task T1

- Goal: Add the default OpenAI Realtime opening disclaimer event and tests for
  default behavior plus configured override behavior.
- Depends on:
  - none
- Write Scope:
  - internal/core/service/session/service.go
  - internal/core/service/session/service_test.go
- Read Context:
  - docs/v3/designs/2026-06-24-v3-realtime-opening-disclaimer.md
  - internal/core/service/session/service.go
  - internal/core/service/session/service_test.go
- Checks:
  - rtk go test ./internal/core/service/session
  - rtk go test ./...
- Parallel-safe: no

## Notes

- Keep this slice to the initial Realtime event. Browser microphone gating and
  crisis handling belong in later web/API/provider-policy slices.
