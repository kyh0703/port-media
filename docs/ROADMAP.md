# Roadmap

## Current Track

- Active version: `v1`
- Exit criteria:
  - `dubu-web` can create an authenticated realtime session through `dubu-api`.
  - `dubu-web` can establish WebRTC audio with `portfoilo-media`.
  - `portfoilo-media` can connect a matching OpenAI Realtime participant for the session.
  - Client audio is forwarded to OpenAI Realtime and OpenAI audio is forwarded back to the client.
  - `portfoilo-media` stores live media session status in Redis for `dubu-api` lookup.
  - The system exposes enough logs and `/api/v1/metrics` runtime fields to support later monitoring.

## Upcoming Versions

- `v2`:
  - Goal: Add monitoring-oriented participant/session views and operational metrics.
  - Dependencies: v1 room/session lifecycle events and stable media server metrics.
- `v3`:
  - Goal: Support multiple client or monitor participants in a room.
  - Dependencies: v1 SFU participant/track abstraction and v2 monitoring visibility.
- `v4`:
  - Goal: Harden production scaling with TURN, SFU node selection, reconnect handling, and horizontal coordination.
  - Dependencies: stable session routing and production traffic observations.

## Deferred

- Candidate versions:
  - Read-only supervisor or monitor participant.
  - Multi-client room UI.
  - Raw media recording or archival.
  - Redis-backed SFU cluster coordination beyond single-node live state.
  - Alerting dashboard and long-term metrics storage.
- Open sequencing questions:
  - Whether monitor participants should receive live audio, metadata only, or delayed transcripts.
  - Whether session routing should stay API-driven or move behind a dedicated media coordinator.
