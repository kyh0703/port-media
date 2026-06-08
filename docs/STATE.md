# State

current_version: none

## Completed Versions

### v1

- Completed: `2026-06-08`
- Feature: `pion-realtime-sfu-session-server`
- Built the first audio-only Pion realtime media server vertical slice.
- Added media-token-authorized session signaling, Pion WebRTC offer/answer handling, OpenAI Realtime server-side call creation, and Redis-backed live session state.
- Added room, participant, track, lifecycle, and runtime event models so future monitor and multi-participant work can extend the core shape.
- Added `/api/v1` session create, join, status, participant leave, and end endpoints.
- Added Redis media token verification, `media:session:<session_id>` live state projection, media-server heartbeat, and Redis conversation event publishing.
- Added lifecycle cleanup for participant leave, session end, idle rooms, provider hangup, and service shutdown.
- Verification passed with `rtk go test ./...` and `rtk go build -o /tmp/portfoilo-media-check ./cmd/api`.

## Project Context

- The web client owns the browser app.
- The API server owns authentication, conversation/session creation, durable persistence, and user-facing history APIs.
- This repository owns the realtime media server runtime.

## Current Decisions

- The API server remains the source of truth for authenticated session creation, ownership, conversation state, and persistence.
- `portfoilo-media` owns WebRTC peer connections, Pion SFU rooms, participants, tracks, OpenAI Realtime connectivity, and media runtime lifecycle.
- The API server creates sessions and returns a short-lived media token.
- Clients use the media token to send SDP signaling directly to `portfoilo-media`.
- Client WebRTC media must connect to `portfoilo-media`, not directly to OpenAI Realtime.
- The media server connects to OpenAI Realtime as a server-side participant.
- `portfoilo-media` writes live media session state to Redis under `media:session:<session_id>` with TTL instead of calling back into the API server.
- v1 is audio-first with multiple client participants and one OpenAI agent participant per room. Client offers choose `publisher` or `listener` media direction, while publisher-count policy remains outside the media server.

## Open Follow-ups

- Decide whether production monitoring should read Redis live state, structured logs, or a future metrics backend.
- Run a full cross-service smoke test with the web client, API server, Redis, browser WebRTC, and OpenAI Realtime credentials.
