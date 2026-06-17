package openai

import (
	"encoding/json"
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/core/port"
)

func TestEventNormalizerNormalizesRealtimeEvent(t *testing.T) {
	normalizer := NewEventNormalizer()

	signal := normalizer.Normalize(`{"type":"conversation.item.input_audio_transcription.completed","event_id":"evt_1","item_id":"item-1","previous_item_id":"item-0","response_id":"resp-1","transcript":"hello","token":"drop","nested":{"authorization":"drop","value":"keep"}}`)

	if signal.Type != port.ConversationEventTypeSpeechInputTranscriptCompleted {
		t.Fatalf("Type = %q, want %s", signal.Type, port.ConversationEventTypeSpeechInputTranscriptCompleted)
	}
	if signal.ProviderEventType != port.ProviderEventType("conversation.item.input_audio_transcription.completed") {
		t.Fatalf("ProviderEventType = %q", signal.ProviderEventType)
	}
	if signal.ProviderEventID != "evt_1" {
		t.Fatalf("ProviderEventID = %q, want evt_1", signal.ProviderEventID)
	}
	if signal.ProviderItemID != "item-1" {
		t.Fatalf("ProviderItemID = %q, want item-1", signal.ProviderItemID)
	}
	if signal.PreviousProviderItemID != "item-0" {
		t.Fatalf("PreviousProviderItemID = %q, want item-0", signal.PreviousProviderItemID)
	}
	if signal.ProviderRespID != "resp-1" {
		t.Fatalf("ProviderRespID = %q, want resp-1", signal.ProviderRespID)
	}
	if !signal.Publishable {
		t.Fatal("Publishable = false, want true")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(signal.Payload), &payload); err != nil {
		t.Fatalf("payload Unmarshal() error = %v", err)
	}
	if _, ok := payload["token"]; ok {
		t.Fatal("payload token was not removed")
	}
	nested, ok := payload["nested"].(map[string]any)
	if !ok {
		t.Fatalf("payload nested = %#v, want object", payload["nested"])
	}
	if _, ok := nested["authorization"]; ok {
		t.Fatal("payload nested authorization was not removed")
	}
	if nested["value"] != "keep" {
		t.Fatalf("payload nested value = %v, want keep", nested["value"])
	}
}

func TestEventNormalizerFallsBackToUnknown(t *testing.T) {
	normalizer := NewEventNormalizer()

	if got := normalizer.Normalize(`{"event":"missing"}`).Type; got != port.ConversationEventTypeUnknown {
		t.Fatalf("Type = %q, want %s", got, port.ConversationEventTypeUnknown)
	}
	if got := normalizer.Normalize(`not-json`).Type; got != port.ConversationEventTypeUnknown {
		t.Fatalf("invalid Type = %q, want %s", got, port.ConversationEventTypeUnknown)
	}
}
