# Architecture

## Purpose

- Records structural principles that are common to all versions.
- Detailed designs for each version are left in `docs/vN/designs/`.
- Defines this repository as the media plane/SFU. AI provider and agent logic
  must run outside this repository as an agent participant.

## Shared Boundaries

- Core domains:
  - Web client: owns user interaction, microphone capture, session UI, and
    history UI.
  - API server: owns authentication, authorization, conversation creation, room
    id issuance, participant token/permission issuance, agent dispatch, durable
    persistence, and history APIs.
  - Media server: this repository owns Pion WebRTC runtime, SFU rooms,
    participant joins, signaling, participant registry, tracks,
    publish/subscribe, media forwarding, and live media state.
  - Agent worker: owns AI provider connectivity, speech pipeline/provider
    lifecycle, transcript/tool/audit event generation, and joins media rooms as
    an agent participant.
- External integrations:
  - `../port-api` for room creation requests, participant token minting, durable
    history, and authenticated SSE fanout.
  - Agent worker for AI provider bridge behavior.
  - Optional STUN/TURN infrastructure for client, agent, and media server
    connectivity.
  - Shared Redis key-value storage for short-lived participant tokens and live
    media session state.
- Data boundaries:
  - Client receives API URL, media signaling URL, room/session identifiers, and
    a short-lived participant token from the API server.
  - Client sends conversation creation to the API server.
  - Client and agent send room join/signaling directly to `port-media` with
    participant tokens.
  - Client never receives AI provider credentials.
  - Media server never receives OpenAI API keys or other provider credentials.
  - The API server remains the durable source of truth for user ownership and
    conversation history.

## Shared Constraints

- Security:
  - All user-owned conversation creation starts in the API server after JWT
    verification.
  - Media server accepts participant joins only with a token minted by the API
    server.
  - Participant tokens are opaque bearer tokens stored in Redis under
    `media:token:<token>` with a TTL owned by the API server.
  - Redis token values include `session_id`, `conversation_id`, `room_id`,
    `participant_id`, `participant_role`, owner identity when applicable, and
    media permissions.
  - AI provider credentials stay in the agent worker, not in this repository.
- Reliability:
  - Media runtime lifecycle is reflected in Redis under
    `media:session:<session_id>` with a TTL.
  - Individual participant connection or track failure must update that
    participant state without failing the whole room.
  - Runtime rooms idle longer than `room_idle_timeout` must close media peers
    and write closed live state.
  - Service shutdown must close active runtime rooms before exit.
  - Durable history, transcript, tool, provider, and audit persistence remain
    owned by the API server and agent event paths.
- Performance:
  - v1 is audio-first.
  - Media forwarding should avoid transcoding in the normal path.
  - Room and participant abstractions should support later multi-session,
    monitor participants, and multiple agent participants without replacing the
    core model.
- Operational limits:
  - v1 does not need horizontal SFU clustering.
  - v1 allows multiple participants per room. Publisher/subscriber capability is
    token-permission based, not hard-coded by participant role.
  - v1 should expose structured logs for session id, room id, participant id,
    participant role, track type, connection state, and failure reason.
  - v1 exposes authenticated single-session live status at
    `/api/v1/sessions/:sessionId/status`, backed by Redis state.

## Recommended Stack

- Language/runtime: Go.
- WebRTC stack: Pion.
- Server shape: standalone media service in this repository.
- Client-facing conversation creation: owned by the API server.
- Client/agent-facing media signaling: owned by `port-media`, authorized by
  short-lived participant tokens minted by the API server.
- API integration: shared Redis state lookup plus explicit room creation/control
  endpoints.
- Provider integration: owned by an external agent worker, not by this service.

## Primary Flow

1. Client asks the API server to create a conversation.
2. The API server verifies the user's JWT and creates a durable conversation
   record.
3. The API server creates or assigns `sessionId`, `conversationId`, `roomId`,
   user `participantId`, `mediaSignalingUrl`, and a short-lived user
   participant token.
4. The API server calls `port-media` to create or reserve the runtime room.
5. The API server dispatches an external agent job and mints a separate agent
   participant token.
6. Client connects to `port-media` with the user participant token and joins the
   room.
7. Client publishes microphone audio into the room.
8. Agent worker connects to `port-media` with the agent participant token and
   joins the same room.
9. Agent worker subscribes to user audio, connects to its AI provider, and
   publishes assistant audio back into the room.
10. `port-media` forwards audio tracks between subscribed participants.
11. `port-media` writes live media state to Redis under
    `media:session:<session_id>`.
12. Agent worker sends transcript/tool/provider/audit events to API-owned
    persistence and fanout paths.
13. Client reads live transcript/status and durable history from API-owned
    endpoints.

## Core Model

- Room:
  - Represents one realtime conversation media room.
  - Has stable `sessionId`, `conversationId`, and `roomId`.
- Participant:
  - `user`: browser or user-owned client participant.
  - `agent`: external AI worker participant.
  - `monitor`: deferred read-only participant role.
- Track:
  - v1 supports audio tracks.
  - Video and screen tracks are deferred.
- Permission:
  - `publish_audio`: participant may publish audio tracks.
  - `subscribe_audio`: participant may subscribe to audio tracks.
  - Future permissions should be added without role-specific branching where
    possible.
- Media session state:
  - Live key-value state for `session_id`, `conversation_id`, `room_id`, room
    status, WebRTC connection state, audio media state, participant count,
    participant role/audio-mode snapshots, track snapshots, and
    `last_active_at`.
  - Stored in Redis with TTL so stale sessions expire without callback cleanup.

## v1 Architectural Decision

- Build a Pion SFU session server, not a thin one-off WebRTC gateway.
- Keep `port-media` limited to room join, signaling, participant registry,
  track publish/subscribe, and media forwarding.
- Keep OpenAI Realtime, STT/LLM/TTS, tool calls, transcript generation, and
  provider lifecycle out of this repository.
- Model the AI runtime as an external `agent` participant that joins the same
  room as the user.
- Keep the API server out of direct WebRTC connection management.
