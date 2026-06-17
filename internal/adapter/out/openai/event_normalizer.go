package openai

import (
	"encoding/json"
	"strings"

	"github.com/kyh0703/portfoilo-media/internal/core/port"
)

type eventNormalizer struct{}

func NewEventNormalizer() port.ProviderEventNormalizer {
	return eventNormalizer{}
}

func (n eventNormalizer) Normalize(payload string) port.ConversationSignal {
	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return port.ConversationSignal{
			Type:    port.ConversationEventTypeUnknown,
			Payload: payload,
		}
	}

	providerEventType, providerEventID := eventFields(decoded)
	return port.ConversationSignal{
		Type:                   commonEventType(providerEventType),
		ProviderEventType:      providerEventType,
		ProviderEventID:        providerEventID,
		ProviderItemID:         eventString(decoded, []string{"item_id", "item.id"}),
		PreviousProviderItemID: eventString(decoded, []string{"previous_item_id", "previous_item.id", "item.previous_item_id", "item.previous_item.id"}),
		ProviderRespID:         eventString(decoded, []string{"response_id"}),
		Payload:                n.sanitizePayload(payload, decoded),
		Publishable:            n.isPublishable(providerEventType),
	}
}

func (eventNormalizer) sanitizePayload(payload string, decoded any) string {
	sanitized := sanitizeValue(decoded)
	body, err := json.Marshal(sanitized)
	if err != nil {
		return payload
	}
	return string(body)
}

func (eventNormalizer) isPublishable(eventType port.ProviderEventType) bool {
	_, ok := conversationEvents[eventType]
	return ok
}

func eventFields(decoded any) (port.ProviderEventType, string) {
	eventType, _ := pathValue(decoded, "type").(string)
	eventID, _ := pathValue(decoded, "event_id").(string)
	if strings.TrimSpace(eventType) == "" {
		return "", strings.TrimSpace(eventID)
	}
	return port.ProviderEventType(strings.TrimSpace(eventType)), strings.TrimSpace(eventID)
}

func eventString(decoded any, paths []string) string {
	for _, path := range paths {
		value := pathValue(decoded, path)
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func pathValue(value any, path string) any {
	current := value
	for _, segment := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			current = typed[segment]
		case []any:
			return nil
		default:
			return nil
		}
	}
	return current
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSecretPayloadField(key) {
				continue
			}
			out[key] = sanitizeValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeValue(item))
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

const (
	openAIEventItemCreated               port.ProviderEventType = "conversation.item.created"
	openAIEventSessionCreated            port.ProviderEventType = "session.created"
	openAIEventSessionUpdated            port.ProviderEventType = "session.updated"
	openAIEventInputTranscriptDone       port.ProviderEventType = "conversation.item.input_audio_transcription.completed"
	openAIEventInputTranscriptFailed     port.ProviderEventType = "conversation.item.input_audio_transcription.failed"
	openAIEventOutputAudioTranscriptDone port.ProviderEventType = "response.output_audio_transcript.done"
	openAIEventRespCreated               port.ProviderEventType = "response.created"
	openAIEventRespDone                  port.ProviderEventType = "response.done"
	openAIEventOutputTextDone            port.ProviderEventType = "response.output_text.done"
	openAIEventOutputItemDone            port.ProviderEventType = "response.output_item.done"
	openAIEventFunctionArgsDone          port.ProviderEventType = "response.function_call_arguments.done"
	openAIEventMCPArgsDone               port.ProviderEventType = "response.mcp_call_arguments.done"
	openAIEventMCPCallCompleted          port.ProviderEventType = "response.mcp_call.completed"
	openAIEventMCPCallFailed             port.ProviderEventType = "response.mcp_call.failed"
	openAIEventMCPToolsCompleted         port.ProviderEventType = "mcp_list_tools.completed"
	openAIEventMCPToolsFailed            port.ProviderEventType = "mcp_list_tools.failed"
	openAIEventError                     port.ProviderEventType = "error"
)

var conversationEvents = map[port.ProviderEventType]struct{}{
	openAIEventItemCreated:               {},
	openAIEventInputTranscriptDone:       {},
	openAIEventInputTranscriptFailed:     {},
	openAIEventOutputAudioTranscriptDone: {},
	openAIEventOutputTextDone:            {},
	openAIEventFunctionArgsDone:          {},
	openAIEventMCPArgsDone:               {},
	openAIEventMCPCallCompleted:          {},
	openAIEventMCPCallFailed:             {},
	openAIEventMCPToolsCompleted:         {},
	openAIEventMCPToolsFailed:            {},
	openAIEventOutputItemDone:            {},
	openAIEventError:                     {},
}

var commonTypes = map[port.ProviderEventType]port.ConversationEventType{
	openAIEventItemCreated:    port.ConversationEventTypeConversationItemCreated,
	openAIEventSessionCreated: port.ConversationEventTypeSessionProviderCreated,
	openAIEventSessionUpdated: port.ConversationEventTypeSessionProviderUpdated,

	openAIEventInputTranscriptDone:       port.ConversationEventTypeSpeechInputTranscriptCompleted,
	openAIEventInputTranscriptFailed:     port.ConversationEventTypeSpeechInputTranscriptFailed,
	openAIEventOutputAudioTranscriptDone: port.ConversationEventTypeSpeechOutputTranscriptCompleted,

	openAIEventRespCreated:    port.ConversationEventTypeAssistantRespCreated,
	openAIEventRespDone:       port.ConversationEventTypeAssistantRespCompleted,
	openAIEventOutputTextDone: port.ConversationEventTypeAssistantOutputTextCompleted,
	openAIEventOutputItemDone: port.ConversationEventTypeAssistantOutputItemCompleted,

	openAIEventFunctionArgsDone:  port.ConversationEventTypeToolArgumentsCompleted,
	openAIEventMCPArgsDone:       port.ConversationEventTypeToolArgumentsCompleted,
	openAIEventMCPCallCompleted:  port.ConversationEventTypeToolCallCompleted,
	openAIEventMCPCallFailed:     port.ConversationEventTypeToolCallFailed,
	openAIEventMCPToolsCompleted: port.ConversationEventTypeToolListCompleted,
	openAIEventMCPToolsFailed:    port.ConversationEventTypeToolListFailed,

	openAIEventError: port.ConversationEventTypeError,
}

func commonEventType(providerEventType port.ProviderEventType) port.ConversationEventType {
	eventType := port.ProviderEventType(strings.TrimSpace(string(providerEventType)))
	if eventType == "" {
		return port.ConversationEventTypeUnknown
	}
	if commonType, ok := commonTypes[eventType]; ok {
		return commonType
	}
	return port.ConversationEventTypeProviderEvent
}
