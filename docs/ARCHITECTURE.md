# Architecture

## Purpose

- Records structural principles that are common to all versions.
- Detailed designs for each version are left in `docs/vN/designs/`.

## Shared Boundaries

- Core domains:
  - Web client: `../dubu-web` owns user interaction, microphone capture, session UI, and history UI.
  - API server: `../dubu-api` owns authentication, authorization, session creation, conversation ownership, durable persistence, and history APIs.
  - Media server: this repository owns Pion WebRTC runtime, SFU rooms, participants, tracks, OpenAI Realtime connectivity, and media lifecycle.
  - AI provider: OpenAI Realtime provides the speech-to-speech model session.
- External integrations:
  - OpenAI Realtime `/v1/realtime/calls` for server-side WebRTC call creation.
  - OpenAI Realtime WebRTC data channel for control and lifecycle events.
  - Optional STUN/TURN infrastructure for client and media server connectivity.
  - Shared Redis key-value storage for short-lived media tokens and live media session state.
- Data boundaries:
  - Client receives Dubu API URL, media server URL, session identifiers, and a short-lived media token from `dubu-api`.
  - Client sends session creation to `dubu-api`.
  - Clients send SDP signaling directly to `portfoilo-media` with the media token.
  - Client never receives `OPENAI_API_KEY` or direct OpenAI Realtime credentials.
  - Media server may receive a short-lived session token and OpenAI API configuration.
  - `dubu-api` remains the durable source of truth for user ownership and conversation history.

## Shared Constraints

- Security:
  - All user-owned session creation starts in `dubu-api` after JWT verification.
  - Media server accepts client joins only with a token minted by `dubu-api`.
  - Media tokens are opaque bearer tokens stored in Redis under `media:token:<token>` with a TTL owned by `dubu-api`.
  - The Redis token value contains `session_id`, `conversation_id`, and `user_id`.
  - OpenAI credentials stay server-side.
- Reliability:
  - Media runtime lifecycle is reflected in Redis under `media:session:<session_id>` with a TTL.
  - OpenAI Realtime data channel events update the live session state's latest realtime event type, timestamp, and bounded recent event list.
  - OpenAI setup failure, OpenAI participant WebRTC failure, and OpenAI audio relay failure must hang up provider calls when a call exists, close media peers, and write failed live state.
  - Individual client participant connection or track failure must update that participant state without failing the whole room.
  - Runtime rooms idle longer than `room_idle_timeout` must close media peers, hang up provider calls, and write closed live state.
  - Service shutdown must close active runtime rooms and hang up provider calls before exit.
  - `dubu-api` can read the Redis session state when it needs current media status.
  - Durable history, transcript, and audit persistence remain owned by `dubu-api`.
- Performance:
  - v1 is audio-first.
  - Media forwarding should avoid transcoding in the normal path.
  - Room and participant abstractions should support later multi-session and monitor participants without replacing the core model.
- Operational limits:
  - v1 does not need horizontal SFU clustering.
  - v1 allows multiple client participants per room and one OpenAI agent participant. Client media direction is selected per offer with `publisher` or `listener`; publisher count is not hard-coded in the media server.
  - v1 should expose structured logs for session id, room id, participant role, track type, connection state, and failure reason.
  - v1 exposes authenticated single-session live status at `/api/v1/sessions/:sessionId/status`, backed by Redis state.
  - v1 emits structured lifecycle logs for participant join/leave, room close/fail, connection state, media track state, and OpenAI Realtime event observations using stable monitoring keys.

## Recommended Stack

- Language/runtime: Go.
- WebRTC stack: Pion.
- Server shape: standalone media service in this repository.
- Client-facing session creation: owned by `dubu-api`.
- Client-facing media signaling: owned by `portfoilo-media`, authorized by a short-lived media token minted by `dubu-api`.
- API integration: shared Redis state lookup, not direct WebRTC or lifecycle callback ownership.
- Provider integration: OpenAI Realtime WebRTC call from the media server.

## Primary Flow

1. Client asks `dubu-api` to create a realtime session.
2. `dubu-api` verifies the user's JWT and creates a session/conversation record.
3. `dubu-api` returns `sessionId`, `conversationId`, `mediaServerUrl`, and a short-lived `mediaToken` to the client.
4. Client creates an SDP offer and posts it to `/api/v1/sessions/:sessionId/join?mode=publisher` or `/api/v1/sessions/:sessionId/join?mode=listener` with `Authorization: Bearer <mediaToken>`.
5. `portfoilo-media` validates the media token and creates or joins one SFU room for the session.
6. `portfoilo-media` waits for local ICE gathering and returns an SDP answer to the client.
7. Client sets the SDP answer and establishes WebRTC media with `portfoilo-media`.
8. Client joins as a `client` participant. `publisher` clients send audio toward OpenAI; `listener` clients receive audio only.
9. `portfoilo-media` connects to OpenAI Realtime as an `openai_agent` participant.
10. `portfoilo-media` opens the configured OpenAI Realtime data channel, default `oai-events`, for provider control/events.
11. `portfoilo-media` forwards publisher client audio to OpenAI and OpenAI audio back to all connected clients.
12. `portfoilo-media` writes live media state to Redis under `media:session:<session_id>`.
13. `dubu-api` reads live media state from Redis when it needs current status and persists durable history through its own workflows.
14. Client reads history and final state from existing API-owned history endpoints.

## Core Model

- Room:
  - Represents one Dubu realtime session.
  - Has a stable `sessionId` and `conversationId`.
- Participant:
  - `client`: user client participant.
  - `openai_agent`: OpenAI Realtime participant controlled by media server.
  - `monitor`: deferred read-only participant role.
- Track:
  - v1 supports audio tracks.
  - Video and screen tracks are deferred.
- Media session state:
  - Live key-value state for `session_id`, `conversation_id`, `user_id`, `room_id`, room status, WebRTC connection state, audio media state, OpenAI call id, participant count, participant role/audio-mode snapshots, latest/recent realtime event metadata, and `last_active_at`.
  - Stored in Redis with TTL so stale sessions expire without callback cleanup.

## v1 Architectural Decision

- Build a Pion SFU session server, not a thin one-off WebRTC gateway.
- Limit v1 runtime to one OpenAI participant per room, while allowing multiple client participants with per-offer `publisher` or `listener` media direction.
- Keep SFU room/participant/track abstractions now because future monitoring and multi-session work depends on them.
- Keep `dubu-api` out of direct WebRTC connection management.
