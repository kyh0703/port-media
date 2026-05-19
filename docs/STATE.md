# State

current_version: v1

## Active Track

- Version: `v1`
- Goal: Build a Pion-based realtime SFU session server so `dubu-api` creates authenticated sessions, clients use a short-lived media token to signal with the media server, and client media connects to the media server instead of directly with OpenAI Realtime.
- First feature: `pion-realtime-sfu-session-server`

## Project Context

- `../dubu-web` owns the browser app.
- `../dubu-api` owns authentication, conversation/session creation, durable persistence, and user-facing history APIs.
- This repository owns the realtime media server runtime.

## Current Decisions

- `dubu-api` remains the source of truth for authenticated session creation, ownership, conversation state, and persistence.
- `portfoilo-media` owns WebRTC peer connections, Pion SFU rooms, participants, tracks, OpenAI Realtime connectivity, and media runtime lifecycle.
- `dubu-api` creates sessions and returns a short-lived media token.
- Clients use the media token to send SDP signaling directly to `portfoilo-media`.
- Client WebRTC media must connect to `portfoilo-media`, not directly to OpenAI Realtime.
- The media server connects to OpenAI Realtime as a server-side participant.
- `portfoilo-media` writes live media session state to Redis under `media:session:<session_id>` with TTL instead of calling back into `dubu-api`.
- v1 is audio-first with multiple client participants and one OpenAI agent participant per room. Client offers choose `publisher` or `listener` media direction, while publisher-count policy remains outside the media server.

## Open Follow-ups

- Decide whether production should scrape Prometheus `/metrics` directly or route it through OpenTelemetry Collector.
