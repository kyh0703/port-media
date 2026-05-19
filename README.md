# portfoilo-media

Pion-based realtime SFU session server for Dubu.

## Role

- `dubu-api` creates authenticated sessions and owns persistence.
- `portfoilo-media` owns WebRTC runtime, SFU rooms, participants, tracks, and OpenAI Realtime connectivity.
- `dubu-api` creates authenticated sessions and returns a short-lived media token.
- Clients send SDP signaling directly to this service with that media token.
- Client WebRTC media flows to this media server instead of directly to OpenAI Realtime.
- The media server opens the OpenAI Realtime WebRTC data channel for control/events.
- Live media session state is written to Redis for `dubu-api` lookup.

## Media Token Contract

`dubu-api` issues an opaque token and stores the session binding in Redis with TTL.
The API creation endpoint for the new media-server path is:

```text
POST /api/v1/conversations/media-sessions
Authorization: Bearer <jwt>
```

It returns `sessionId`, `conversationId`, `mediaServerUrl`, `mediaToken`, and `expiresInMs`.

Redis key:

```text
media:token:<token>
```

Redis value:

```json
{"session_id":"...","conversation_id":"...","user_id":"..."}
```

The media server validates the client bearer token by reading that Redis key.

SDP signaling endpoint:

```text
POST /api/v1/sessions/<session_id>/offer?mode=publisher
POST /api/v1/sessions/<session_id>/offer?mode=listener
Authorization: Bearer <media_token>
Content-Type: application/sdp
```

The server returns an `application/sdp` answer after local ICE gathering completes, so clients can use non-trickle HTTP offer/answer signaling.
v1 allows multiple client participants per room. A client using `mode=publisher` can send audio toward OpenAI, and a client using `mode=listener` receives OpenAI audio only. Publisher count is not hard-coded in the media server; that remains a policy decision for the session layer.

JSON API responses follow the Dubu common response shape:

```json
{"statusCode":200,"message":"OK","data":{}}
```

Errors use the same shape with `data: null` and an `error` code. The SDP offer endpoint is the only successful raw response exception because it must return `Content-Type: application/sdp`; its error responses still use the common JSON shape.

SDP answer response headers:

```text
X-Room-Id: <room_id>
X-Participant-Id: <participant_id>
```

Clients should keep `X-Participant-Id` and use it when leaving only their own peer connection without ending the room:

```text
POST /api/v1/sessions/<session_id>/participants/<participant_id>/leave
Authorization: Bearer <media_token>
```

## Media Session State

`portfoilo-media` stores live state in Redis with `realtime.room_idle_timeout` as TTL.
OpenAI setup failure, WebRTC connection failure, and audio relay failure are recorded as failed state. If a provider call already exists, failure cleanup also hangs it up before the room is removed from runtime state.
Runtime rooms that have not changed for `realtime.room_idle_timeout` are closed automatically by the idle cleanup loop.
On process shutdown, active runtime rooms are also closed and provider calls are hung up.

Redis key:

```text
media:session:<session_id>
```

Redis value:

```json
{
  "session_id": "...",
  "conversation_id": "...",
  "user_id": "...",
  "room_id": "...",
  "status": "active",
  "connection_state": "connected",
  "media_state": "active",
  "openai_provider_call_id": "...",
  "participants": 2,
  "participant_states": [
    {
      "id": "...",
      "role": "client",
      "audio_mode": "publisher",
      "connection_state": "connected",
      "tracks": 1
    }
  ],
  "last_realtime_event_type": "response.done",
  "last_realtime_event_at": "...",
  "recent_realtime_events": [
    {
      "type": "session.updated",
      "at": "..."
    },
    {
      "type": "response.done",
      "at": "..."
    }
  ],
  "started_at": "...",
  "last_active_at": "..."
}
```

## Commands

```bash
make tidy
make test
make run
```

Smoke the HTTP SDP offer endpoint with a local Pion client:

```bash
go test ./internal/core/handler -run TestOfferEndpointSmokeWithPionClient -v
```

## Runtime Metrics

Readiness is exposed as JSON using the common response shape:

```text
GET /api/v1/health
```

It checks SQLite and Redis. Healthy responses return HTTP 200 with `data.status: "ok"`; degraded responses return HTTP 503 with `data.status: "degraded"` and per-dependency check details.

Live runtime stats are exposed as JSON:

```text
GET /api/v1/metrics
```

The response includes active room/session count, participant count, track count, and grouped counts by room status, connection state, media state, participant role, client audio mode, and latest OpenAI Realtime event type.

When `observability.metrics_enabled` is true, the same runtime stats are also exposed in Prometheus text format:

```text
GET /metrics
```

The service also emits structured lifecycle logs for joins, leaves, room close/fail, connection state, media track state, and Realtime data-channel events. Monitoring fields use stable keys such as `session_id`, `conversation_id`, `room_id`, `participant_id`, `participant_role`, `connection_state`, `track_type`, `media_state`, `close_reason`, and `failure_reason`.

Single-session live status is available with a valid media token:

```text
GET /api/v1/sessions/<session_id>/status
Authorization: Bearer <media_token>
```

The response is read from Redis `media:session:<session_id>`.
When OpenAI Realtime emits data channel events, the media server records the latest realtime event and a bounded recent realtime event list in the same live state.

## Configuration

Default local config is `dev.yaml`. Environment variables can override values through Viper.
`realtime.ice_gathering_timeout` controls how long SDP offer/answer creation waits for local ICE candidates.
`realtime.room_idle_timeout` controls both Redis live-state TTL and stale runtime room cleanup.
`realtime.realtime_event_history_limit` controls how many recent OpenAI Realtime data channel event summaries are retained in live state.
`server.cors.allowed_origins`, `server.cors.allowed_methods`, and `server.cors.allowed_headers` control browser preflight access for direct SDP signaling from `dubu-web`.
`server.cors.expose_headers` must include `X-Room-Id` and `X-Participant-Id` if browser JavaScript needs to read the offer response metadata.
`openai.realtime_data_channel_label` defaults to `oai-events`, matching OpenAI Realtime WebRTC control-event examples.
`openai.realtime_initial_events` can contain raw JSON Realtime client events that are sent when the provider data channel opens.
