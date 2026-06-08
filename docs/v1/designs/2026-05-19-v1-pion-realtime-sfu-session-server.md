---
feature: pion-realtime-sfu-session-server
status: completed
created_at: 2026-05-19T14:19:33+09:00
completed_at: 2026-06-08
---

# Pion Realtime SFU Session Server

## Goal

Create the first executable plan for a Pion-based realtime media server that sits between browsers and OpenAI Realtime. The media server should use an SFU room/participant/track model from v1, while the API server remains the source of truth for authenticated session creation and persistence.

## Context / Inputs

- Source docs:
  - `docs/STATE.md`
  - `docs/ROADMAP.md`
  - `docs/ARCHITECTURE.md`
  - API server architecture docs
  - Web client architecture docs
- Existing system facts:
  - The web client currently creates a browser `RTCPeerConnection`, posts SDP to the API server, and sets the OpenAI SDP answer as the remote description.
  - The API server currently brokers OpenAI Realtime call creation, stores provider call ids, opens sideband monitoring, and persists conversation/session events.
  - The new direction removes direct browser-to-OpenAI media connectivity.
  - The API server should create sessions, own user authorization, and persist lifecycle/history without directly managing WebRTC peer connections.
  - `portfoilo-media` should own Pion peer connections, room runtime, participant runtime, track forwarding, OpenAI Realtime connectivity, and connection lifecycle.
- External constraints:
  - OpenAI Realtime supports WebRTC call creation through `/v1/realtime/calls`, where a server sends an SDP offer and receives an SDP answer.
  - OpenAI Realtime WebRTC sessions exchange application events over data channels.
  - Pion and ion-sfu demonstrate the Go/Pion SFU shape, including room/participant/media forwarding abstractions.

## Problem Statement

The product needs to stop letting clients connect directly to OpenAI Realtime and instead route realtime audio through a first-party media server. The first version must preserve the API server as the authenticated session and persistence owner, while introducing a Pion SFU runtime that can manage client participants, an OpenAI Realtime participant, media forwarding, lifecycle events, and monitoring-ready telemetry. This needs to be narrow enough for v1, but not modeled as a throwaway gateway because later versions will add more sessions, monitor participants, and operational monitoring.

## Decision Drivers

- Client must never receive OpenAI credentials or directly manage OpenAI Realtime connectivity.
- The API server must remain the source of truth for user ownership, session creation, durable state, and history.
- The media server must own WebRTC runtime details and avoid pushing connection management back into the API server.
- The model must support later monitoring participants and multi-session operation.
- v1 must stay executable by limiting media scope to audio-only and one OpenAI agent participant per room while allowing multiple client participants with per-offer publisher/listener direction.
- Operational telemetry must be designed early enough that later monitoring work does not require changing every event shape.

## Options Considered

### Option A

- Summary: Build a thin Pion WebRTC gateway with client peer connections and one OpenAI peer connection per session.
- Pros:
  - Smallest first implementation.
  - Directly matches the immediate browser-to-OpenAI replacement.
  - Less abstraction and fewer room-management concerns.
- Cons:
  - Does not naturally support monitor participants or future multi-party rooms.
  - Later SFU conversion would likely rewrite session and forwarding internals.
  - Monitoring dimensions such as room, participant, and track would be bolted on later.
- Risks:
  - Short-term speed creates a structural dead end for the stated roadmap.

### Option B

- Summary: Build a Pion SFU session server with room, participant, and track abstractions, but constrain v1 to audio-only and one OpenAI agent participant.
- Pros:
  - Matches the future need for more sessions, monitoring, and participant-level visibility.
  - Keeps the API server cleanly separated from WebRTC runtime.
  - Allows monitor participants, multi-user rooms, and routing policies to be added later without replacing the core model.
  - Provides natural telemetry labels: room, participant role, track type, connection state, and failure reason.
- Cons:
  - More v1 design and implementation work than a thin gateway.
  - Requires careful scope control to avoid building full conferencing in v1.
- Risks:
  - Room/participant abstractions can become too broad if video, multi-user UI, or clustering is pulled into v1.

### Option C

- Summary: Keep current direct browser-to-OpenAI WebRTC and improve API server sideband/session management only.
- Pros:
  - Lowest implementation cost.
  - Preserves existing working direction in the web client and API server.
- Cons:
  - Does not satisfy the requirement that browser media flows through the media server.
  - Cannot centralize media runtime monitoring in first-party infrastructure.
  - Keeps WebRTC connection control split between browser and provider.
- Risks:
  - Blocks future SFU-based monitoring and participant expansion.

## Recommended Option

- Choice: Option B, Pion SFU session server with constrained v1 scope.
- Why now:
  - The user explicitly wants SFU capability because more sessions and monitoring are planned later.
  - The API/media boundary is clear: the API server creates and persists sessions; `portfoilo-media` owns realtime connections.
  - The extra abstraction is justified by concrete future requirements, not speculative scaling alone.
- Rejected alternatives:
  - Thin gateway rejected because it creates a near-term rewrite risk.
  - Current direct OpenAI browser connection rejected because it contradicts the target architecture.

## Scope Decision

- In:
  - Define the service boundary for `portfoilo-media`.
  - Define API server session creation contract inputs/outputs.
  - Define media token verification responsibilities between the API server and `portfoilo-media`.
  - Define Pion SFU room/participant/track model for v1.
  - Define client-to-media-server signaling flow authorized by an API-server-minted media token.
  - Define OpenAI Realtime participant connection flow from the media server.
  - Define audio forwarding path client -> OpenAI and OpenAI -> client.
  - Define Redis key-value media session state for API server status lookup.
  - Define monitoring-ready log/metric dimensions.
- Out:
  - Multi-client room UI.
  - Video or screen tracks.
  - Raw media recording.
  - Full monitoring dashboard.
  - Horizontal SFU clustering.
  - Redis-backed distributed SFU cluster coordination.
  - TURN production hardening beyond config placeholders.
- Deferred:
  - Read-only monitor participant.
  - Multi-user session support.
  - SFU node selection and autoscaling.
  - Callback/event delivery for durable audit streams.
  - Alerting and metrics dashboard.
  - Recording pipeline.

## Open Questions

- Media token verification uses Redis lookup of an opaque token stored by the API server with TTL.
- Redis token values must include `session_id`, `conversation_id`, and `user_id`.
- Live media session state uses Redis key `media:session:<session_id>` with TTL, room status, WebRTC connection state, audio media state, and `last_active_at`.
- Failure cleanup writes failed room, connection, and media state before leaving the session for TTL expiry.
- Runtime monitoring starts with Redis live state and structured logs, including role and publisher/listener audio-mode counts.
- Single-session live status is available at `/api/v1/sessions/:sessionId/status` with a valid media token and includes participant role/audio-mode snapshots.
- HTTP SDP offer/answer signaling uses non-trickle ICE by waiting for local ICE gathering before returning SDP.
- Multiple client offers can join the same active session; media direction is selected with publisher/listener mode.
- OpenAI Realtime provider peers create the configured WebRTC data channel, default `oai-events`, for control/event messages.
- The latest and recent OpenAI Realtime data channel event types and timestamps are reflected in Redis live session state for status/monitoring.
- Stale runtime rooms are cleaned up automatically after `room_idle_timeout`.
- Service shutdown closes active runtime rooms and hangs up provider calls.
- Should v1 client-to-media signaling be HTTP offer/answer endpoints, WebSocket signaling, or a hybrid where join is HTTP and later ICE/control is WebSocket?
- Should OpenAI Realtime be connected from the media server through WebRTC only in v1, or should a later WebSocket audio bridge remain a documented alternative?
- Which monitoring backend should be targeted first: Redis live state, OpenTelemetry metrics, or structured logs only?

## Plan Handoff

### Source of Truth Docs

- `docs/STATE.md`
- `docs/ROADMAP.md`
- `docs/ARCHITECTURE.md`
- `docs/v1/designs/2026-05-19-v1-pion-realtime-sfu-session-server.md`

### Scope for Planning

- Create one executable v1 plan for the media server architecture and minimal scaffolding path.
- The plan should cover API contracts and service boundaries even if implementation changes in the API server and web client are deferred or represented as integration contracts.
- The plan should keep the first vertical slice focused on audio-only client-to-SFU-to-OpenAI session establishment and Redis-backed live session state.

### Fixed Constraints

- The API server owns authenticated session creation and durable state.
- `portfoilo-media` owns all WebRTC peer connections and OpenAI Realtime connectivity.
- Client gets `mediaServerUrl` and `mediaToken` from the API server.
- Client sends SDP signaling directly to `portfoilo-media` with the media token.
- Client media connects to `portfoilo-media`, not OpenAI Realtime.
- v1 uses a Pion SFU room/participant/track model.
- v1 is audio-only.
- v1 has multiple client participants and one OpenAI agent participant per room.
- v1 stores live media state in Redis for API lookup instead of direct media-server callbacks to the API server.
- Monitoring dashboard, multi-user rooms, video, recording, and clustering are non-goals for this feature.

### Success Criteria

- A plan reader can see exactly how the web client, API server, `portfoilo-media`, and OpenAI Realtime interact.
- The media server has a clear v1 room/participant/track model that can later add monitor participants.
- The planned implementation keeps the API server out of direct WebRTC connection management.
- Redis live session state is part of the first media-server contract.
- The first implementation can be validated locally with one client session, one media-token-authorized signaling request, and one OpenAI Realtime participant.

### Non-Goals

- Building a full conferencing SFU.
- Replacing API server persistence.
- Exposing OpenAI credentials to browsers.
- Implementing a production monitoring dashboard in v1.
- Implementing media recording or playback.

### Open Questions

- Token validation style.
- Signaling transport style.
- Metrics backend.
- Exact OpenAI Realtime event subset to relay to the API server in the first slice.

### Suggested Validation

- Unit tests for session token parsing/validation boundaries.
- Unit tests for room, participant, and track lifecycle state transitions.
- Integration test or local harness for client SDP offer -> media server SDP answer.
- Provider-adapter test with mocked OpenAI `/v1/realtime/calls` response.
- Redis state repository test for `media:session:<session_id>` payload and TTL.
- Manual local smoke test with one client session once implementation exists.

### Parallelization Hints

- Candidate write boundaries:
  - Go module/service scaffold and config.
  - SFU domain model: room, participant, track, lifecycle.
  - Signaling API and media token validation.
  - OpenAI Realtime adapter.
  - Redis live media session state repository.
  - Observability/logging primitives.
- Shared files to avoid touching in parallel:
  - `go.mod`
  - primary application entrypoint
  - central config package
  - shared state type definitions
- Likely sequential dependencies:
  - Go module scaffold before package implementation.
  - Domain model before signaling and provider adapter integration.
  - State contract before Redis repository.
  - Signaling and OpenAI adapter before local end-to-end smoke validation.
