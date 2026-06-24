---
feature: realtime-vad-no-interrupt
status: plan_ready
created_at: 2026-06-24T00:00:00+09:00
---

# Realtime VAD No Interrupt

## Goal

Configure OpenAI Realtime sessions created by the media server so user speech
detected during assistant audio does not automatically cancel the assistant's
ongoing response.

## Context / Inputs

- Source docs:
  - `docs/STATE.md`
  - `docs/ROADMAP.md`
  - `docs/ARCHITECTURE.md`
- Existing system facts:
  - This repository owns OpenAI Realtime call creation through
    `internal/adapter/out/openai/realtime_client.go`.
  - The existing call creation request sends a multipart `session` JSON field
    containing `type` and `model`.
  - Existing data-channel initial messages remain available for later explicit
    session or response events.
- User brief:
  - Add Realtime VAD settings with `server_vad`, `create_response: true`, and
    `interrupt_response: false`.

## Plan Handoff

### Scope for Planning

- Add the VAD turn detection settings to the OpenAI Realtime session payload
  used when creating calls.
- Keep the existing model, base URL, API key, SDP, provider call id extraction,
  data channel label, and initial data message behavior unchanged.
- Add a focused test that decodes the multipart `session` field and verifies the
  nested VAD configuration.

### Success Criteria

- Realtime call creation sends:
  - `audio.input.turn_detection.type = server_vad`
  - `audio.input.turn_detection.create_response = true`
  - `audio.input.turn_detection.interrupt_response = false`
- Existing OpenAI Realtime call tests pass.
- The full Go test suite passes.

### Non-Goals

- Implementing semantic VAD or new runtime configuration switches.
- Muting, buffering, or dropping client audio while assistant audio is playing.
- Changing media forwarding, Pion, Redis, API, or web-client behavior.

### Open Questions

- None for this slice.

### Suggested Validation

- `rtk go test ./internal/adapter/out/openai`
- `rtk go test ./...`

### Parallelization Hints

- Candidate write boundaries:
  - `internal/adapter/out/openai/realtime_client.go`
  - `internal/adapter/out/openai/realtime_client_test.go`
- Shared files to avoid touching in parallel:
  - none beyond the two OpenAI adapter files.
- Likely sequential dependencies:
  - Add failing payload test before implementation.
