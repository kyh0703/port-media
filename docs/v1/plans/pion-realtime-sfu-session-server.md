# Pion Realtime SFU Session Server

## Goal

- Implement the first audio-only Pion SFU session-server vertical slice so `dubu-api` creates sessions and media tokens, clients send SDP signaling directly to `portfoilo-media` with that token, client WebRTC media connects to `portfoilo-media`, `portfoilo-media` can represent a client and OpenAI Realtime as SFU participants, and `dubu-api` remains the session creation and persistence owner.

## References

- docs/STATE.md
- docs/ROADMAP.md
- docs/ARCHITECTURE.md
- docs/v1/designs/2026-05-19-v1-pion-realtime-sfu-session-server.md
- ../dubu-api/docs/ARCHITECTURE.md
- ../dubu-web/docs/ARCHITECTURE.md
- OpenAI Realtime WebRTC guide: https://platform.openai.com/docs/guides/realtime-webrtc
- OpenAI Realtime API reference: https://platform.openai.com/docs/api-reference/realtime?api-mode=responses
- ion-sfu reference shape: https://github.com/ionorg/ion-sfu

## Workspace

- Branch: feat/v1-pion-realtime-sfu-session-server
- Base: main
- Isolation: required
- Created by: exec-plan via git-worktree

## Task Graph

### Task T1

- Goal: Create the Go service scaffold, root commands, config loading, and baseline test/build scripts.
- Depends on:
  - none
- Write Scope:
  - go.mod
  - go.sum
  - cmd/portfoilo-media/**
  - internal/config/**
  - Makefile
  - .gitignore
  - README.md
- Read Context:
  - docs/ARCHITECTURE.md
  - docs/v1/designs/2026-05-19-v1-pion-realtime-sfu-session-server.md
- Checks:
  - rtk go test ./...
  - rtk go test ./internal/config/...
- Parallel-safe: no

### Task T2

- Goal: Add shared session, room, participant, track, lifecycle, and live media state contract types.
- Depends on:
  - T1
- Write Scope:
  - internal/contracts/**
- Read Context:
  - docs/ARCHITECTURE.md
  - docs/v1/designs/2026-05-19-v1-pion-realtime-sfu-session-server.md
- Checks:
  - rtk go test ./internal/contracts/...
- Parallel-safe: no

### Task T3

- Goal: Implement the in-memory SFU room registry and audio-only room/participant/track lifecycle model.
- Depends on:
  - T2
- Write Scope:
  - internal/sfu/**
- Read Context:
  - internal/contracts/**
  - docs/ARCHITECTURE.md
- Checks:
  - rtk go test ./internal/sfu/...
- Parallel-safe: no

### Task T4

- Goal: Implement media token validation and client signaling endpoints for session join and SDP offer/answer exchange.
- Depends on:
  - T3
- Write Scope:
  - internal/auth/**
  - internal/signaling/**
- Read Context:
  - internal/contracts/**
  - internal/sfu/**
  - docs/v1/designs/2026-05-19-v1-pion-realtime-sfu-session-server.md
- Checks:
  - rtk go test ./internal/auth/... ./internal/signaling/...
- Parallel-safe: no

### Task T5

- Goal: Implement the OpenAI Realtime WebRTC call adapter that creates provider calls from SDP offers and extracts provider call ids.
- Depends on:
  - T2
- Write Scope:
  - internal/openai/**
- Read Context:
  - internal/contracts/**
  - docs/v1/designs/2026-05-19-v1-pion-realtime-sfu-session-server.md
- Checks:
  - rtk go test ./internal/openai/...
- Parallel-safe: no

### Task T6

- Goal: Implement Redis-backed live media session state storage for `media:session:<session_id>`.
- Depends on:
  - T2
- Write Scope:
  - internal/core/repository/**
  - internal/core/domain/**
- Read Context:
  - internal/contracts/**
  - docs/ARCHITECTURE.md
- Checks:
  - rtk go test ./internal/core/repository/...
- Parallel-safe: no

### Task T7

- Goal: Wire signaling, SFU room lifecycle, OpenAI adapter, and Redis media state into one local server runtime.
- Depends on:
  - T4
  - T5
  - T6
- Write Scope:
  - internal/server/**
  - cmd/portfoilo-media/**
- Read Context:
  - internal/auth/**
  - internal/signaling/**
  - internal/sfu/**
  - internal/openai/**
  - internal/core/repository/**
- Checks:
  - rtk go test ./internal/server/...
  - rtk go test ./...
- Parallel-safe: no

### Task T8

- Goal: Add monitoring-ready structured logs and health endpoints for session, participant, track, and connection state.
- Depends on:
  - T7
- Write Scope:
  - internal/server/**
  - README.md
- Read Context:
  - docs/ROADMAP.md
  - internal/server/**
  - internal/contracts/**
- Checks:
  - rtk go test ./internal/server/...
  - rtk go test ./...
- Parallel-safe: no

## Notes

- The plan intentionally avoids implementing multi-client rooms, video tracks, raw recording, SFU clustering, and a monitoring dashboard.
- The first implementation may use a mocked or contract-tested OpenAI Realtime response path before real provider smoke testing, but the adapter contract must match the `/v1/realtime/calls` SDP answer flow.
- If the repository has no initial commit, `exec-plan` must stop before creating a worktree and request an initial base commit or an explicit non-worktree execution decision.
