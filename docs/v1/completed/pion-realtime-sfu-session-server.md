# Pion Realtime SFU Session Server

completed_at: 2026-06-08

## Summary

- Implemented the first audio-only realtime media server vertical slice.
- The media server now owns WebRTC peer handling, session room runtime, OpenAI Realtime call setup, Redis-backed live media state, and lifecycle cleanup.
- The API server remains the source of truth for authenticated session creation and durable persistence; this repository exposes the media runtime and media-token-authorized signaling surface.

## Completed Scope

- Added a Go service runtime with `fx` wiring, configuration loading, HTTP routing, health handling, panic recovery, CORS, structured logging, Redis, SQLite/Bun persistence, and Docker/Taskfile build support.
- Added session endpoints under `/api/v1`:
  - `POST /sessions`
  - `POST /sessions/{sessionId}/join`
  - `GET /sessions/{sessionId}/status`
  - `POST /sessions/{sessionId}/participants/{participantId}/leave`
  - `POST /sessions/{sessionId}/end`
- Added Redis-backed media token verification using `media:token:<token>` values with `session_id`, `conversation_id`, and `user_id`.
- Added room, participant, track, media session record, media session state, and runtime event domain models.
- Added in-memory room runtime storage and lifecycle operations for create, join, leave, end, idle cleanup, and shutdown cleanup.
- Added Pion WebRTC offer/answer handling, ICE gathering timeout behavior, audio-only peer setup, connection state reporting, media track state reporting, and audio bridge tracks for client-to-OpenAI and OpenAI-to-client paths.
- Added OpenAI Realtime call adapter for `/v1/realtime/calls`, provider call id extraction, SDP answer handling, and provider call hangup.
- Added OpenAI Realtime data channel event observation through the configured `oai-events` label and projected recent event metadata into live session state.
- Added Redis `media:session:<session_id>` live-state projection with TTL, participant snapshots, connection/media state, provider call id, latest realtime event, recent realtime events, and `last_active_at`.
- Added Redis media-server heartbeat state for node-level liveness.
- Added Redis conversation event publishing for media lifecycle events while keeping durable history ownership outside this repository.
- Refactored the codebase into hexagonal boundaries with inbound HTTP/lifecycle adapters, outbound Redis/OpenAI/WebRTC/persistence adapters, core usecases, ports, domain entities, and query projection types.

## Verification

- `rtk go test ./...`
  - Passed: 104 tests across 36 packages.
- `rtk go build -o /tmp/portfoilo-media-check ./cmd/api`
  - Passed.

## Deferred

- End-to-end smoke test with the real web client, API server, Redis, browser WebRTC, and OpenAI Realtime credentials.
- Production monitoring backend or dashboard.
- TURN production hardening and SFU node selection.
- Horizontal SFU clustering.
- Video, screen sharing, recording, playback, and monitor participant UI.
