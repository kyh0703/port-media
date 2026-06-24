---
feature: realtime-opening-disclaimer
status: plan_ready
created_at: 2026-06-24T00:00:00+09:00
---

# Realtime Opening Disclaimer

## Goal

Make the OpenAI Realtime participant speak a natural Korean opening disclaimer
as soon as the provider data channel opens for a new media call.

## Context / Inputs

- Source docs:
  - `docs/STATE.md`
  - `docs/ROADMAP.md`
  - `docs/ARCHITECTURE.md`
  - `docs/v3/plans/realtime-vad-no-interrupt.md`
- Existing system facts:
  - `portfoilo-media` owns OpenAI Realtime connectivity and data-channel
    initial messages.
  - `ServiceOptions.RealtimeInitialEvents` already supports configured
    OpenAI Realtime client events sent when the data channel opens.
  - OpenAI Realtime `response.create` starts model inference and can include
    per-response `instructions`.
- User brief:
  - Use the AI emotional support phone-agent concept and have the call opening
    disclaimer spoken naturally when the call connects.

## Plan Handoff

### Scope for Planning

- Add a default OpenAI Realtime initial client event that triggers an opening
  disclaimer response in Korean.
- Keep explicit `OPENAI_REALTIME_INITIAL_EVENTS` behavior as an override for
  deployments that want custom initial events.
- Test that the default event is present and that explicit initial events still
  override the default.

### Success Criteria

- A default session join creates an OpenAI offer with one initial
  `response.create` event.
- The event includes instructions for the AI to speak the agreed Korean
  non-medical emotional-support disclaimer.
- Blank configured initial events are ignored.
- Explicit configured initial events still replace the default.

### Non-Goals

- Web-client microphone mute/unmute behavior during the opening disclaimer.
- Crisis keyword detection, escalation flows, or prompt-wide counseling persona.
- New runtime config fields or admin UI.

### Open Questions

- None for this slice.

### Suggested Validation

- `rtk go test ./internal/core/service/session`
- `rtk go test ./...`

### Parallelization Hints

- Candidate write boundaries:
  - `internal/core/service/session/service.go`
  - `internal/core/service/session/service_test.go`
- Shared files to avoid touching in parallel:
  - none beyond the session service files.
- Likely sequential dependencies:
  - Add default-event tests before implementation.
