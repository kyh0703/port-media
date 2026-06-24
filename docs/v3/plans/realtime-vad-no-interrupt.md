# Realtime VAD No Interrupt

## Goal

- Configure OpenAI Realtime call creation to prevent VAD speech start events
  from automatically interrupting an in-progress assistant response.

## References

- docs/STATE.md
- docs/ROADMAP.md
- docs/ARCHITECTURE.md
- docs/v3/designs/2026-06-24-v3-realtime-vad-no-interrupt.md
- internal/adapter/out/openai/realtime_client.go
- internal/adapter/out/openai/realtime_client_test.go

## Workspace

- Branch: feat/v3-realtime-vad-no-interrupt
- Base: main
- Isolation: required
- Created by: exec-plan via git-worktree

## Task Graph

### Task T1

- Goal: Add a focused OpenAI Realtime call payload test and implement the
  default VAD turn detection session payload.
- Depends on:
  - none
- Write Scope:
  - internal/adapter/out/openai/realtime_client.go
  - internal/adapter/out/openai/realtime_client_test.go
- Read Context:
  - docs/v3/designs/2026-06-24-v3-realtime-vad-no-interrupt.md
  - internal/adapter/out/openai/realtime_client.go
  - internal/adapter/out/openai/realtime_client_test.go
- Checks:
  - rtk go test ./internal/adapter/out/openai
  - rtk go test ./...
- Parallel-safe: no

## Notes

- Keep this as a fixed default behavior. Do not add semantic VAD, buffering, or
  environment-driven policy unless a later slice requests it.
