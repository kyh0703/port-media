package session

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type realtimeEventPolicy struct{}

func (realtimeEventPolicy) Type(payload string) string {
	eventType, _ := realtimeEventEnvelopeFields(payload)
	if eventType == "" {
		return "unknown"
	}
	return eventType
}

func (realtimeEventPolicy) ID(payload string) string {
	_, eventID := realtimeEventEnvelopeFields(payload)
	return eventID
}

func (realtimeEventPolicy) SanitizePayload(payload string) string {
	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return payload
	}
	sanitized := sanitizeRealtimeValue(decoded)
	body, err := json.Marshal(sanitized)
	if err != nil {
		return payload
	}
	return string(body)
}

func (realtimeEventPolicy) IsPublishable(eventType string) bool {
	_, ok := conversationEventAllowlist[eventType]
	return ok
}

func realtimeEventType(payload string) string {
	return realtimeEventPolicy{}.Type(payload)
}

func realtimeEventEnvelopeFields(payload string) (string, string) {
	var event struct {
		Type    string `json:"type"`
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return "", ""
	}
	if strings.TrimSpace(event.Type) == "" {
		return "", strings.TrimSpace(event.EventID)
	}
	return strings.TrimSpace(event.Type), strings.TrimSpace(event.EventID)
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func appendRealtimeEvent(
	events []entity.RealtimeEvent,
	event entity.RealtimeEvent,
	limit int,
) []entity.RealtimeEvent {
	if limit <= 0 {
		return nil
	}
	next := append(append([]entity.RealtimeEvent(nil), events...), event)
	if len(next) <= limit {
		return next
	}
	return next[len(next)-limit:]
}

func fallbackConversationEventID(
	sessionID vo.SessionID,
	providerCallID string,
	eventType string,
	payload string,
) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(sessionID),
		providerCallID,
		eventType,
		payload,
	}, "\x00")))
	return fmt.Sprintf("media-%x", sum[:])
}

func sanitizeRealtimeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSecretPayloadField(key) {
				continue
			}
			out[key] = sanitizeRealtimeValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeRealtimeValue(item))
		}
		return out
	default:
		return typed
	}
}

func isSecretPayloadField(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "api_key", "authorization", "token", "secret", "client_secret":
		return true
	default:
		return false
	}
}

var conversationEventAllowlist = map[string]struct{}{
	"conversation.item.created":                             {},
	"conversation.item.input_audio_transcription.completed": {},
	"conversation.item.input_audio_transcription.failed":    {},
	"response.output_audio_transcript.done":                 {},
	"response.output_text.done":                             {},
	"response.function_call_arguments.done":                 {},
	"response.mcp_call_arguments.done":                      {},
	"response.mcp_call.completed":                           {},
	"response.mcp_call.failed":                              {},
	"mcp_list_tools.completed":                              {},
	"mcp_list_tools.failed":                                 {},
	"response.output_item.done":                             {},
	"error":                                                 {},
}
