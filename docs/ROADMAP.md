# Roadmap

## Current Track

- Active version: `none`
- Latest completed version: `v2`
- v1 exit criteria:
  - The web client can create an authenticated realtime session through the API server.
  - The web client can establish WebRTC audio with `portfoilo-media`.
  - `portfoilo-media` can connect a matching OpenAI Realtime participant for the session.
  - Client audio is forwarded to OpenAI Realtime and OpenAI audio is forwarded back to the client.
  - `portfoilo-media` stores live media session status in Redis for API server lookup.
  - The system exposes enough logs and Redis live-state fields to support later monitoring.
- v2 exit criteria:
  - Durable SQL persistence uses PostgreSQL through `DATABASE_URL`.
  - SQLite driver, dialect, schema bootstrap, and health labels are removed or renamed.
  - Redis-backed media tokens, live state, heartbeat, and conversation events remain unchanged.
  - Repository tests, health tests, full test suite, and API build pass.

## Upcoming Versions

- `v3`:
  - Goal: Add monitoring-oriented participant/session views and operational metrics.
  - Dependencies: v1 room/session lifecycle events and stable media server live state.
- `v4`:
  - Goal: Support multiple client or monitor participants in a room.
  - Dependencies: v1 SFU participant/track abstraction and v3 monitoring visibility.
- `v5`:
  - Goal: Harden production scaling with TURN, SFU node selection, reconnect handling, and horizontal coordination.
  - Dependencies: stable session routing and production traffic observations.

## Deferred

- Candidate versions:
  - Read-only supervisor or monitor participant.
  - Multi-client room UI.
  - Raw media recording or archival.
  - Redis-backed SFU cluster coordination beyond single-node live state.
  - Alerting dashboard and long-term operational telemetry storage.
- Open sequencing questions:
  - Whether monitor participants should receive live audio, metadata only, or delayed transcripts.
  - Whether session routing should stay API-driven or move behind a dedicated media coordinator.
