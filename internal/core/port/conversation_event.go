package port

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type ConversationEvent struct {
	SchemaVersion          int
	EventID                string
	ConversationID         vo.ConversationID
	SessionID              vo.SessionID
	RoomID                 vo.RoomID
	ProviderCallID         string
	Type                   ConversationEventType
	ProviderEventType      ProviderEventType
	ProviderItemID         string
	PreviousProviderItemID string
	ProviderRespID         string
	OccurredAt             time.Time
	Payload                string
}

type ConversationEventType string

const (
	ConversationEventTypeUnknown                         ConversationEventType = "unknown"
	ConversationEventTypeProviderEvent                   ConversationEventType = "provider.event"
	ConversationEventTypeConversationItemCreated         ConversationEventType = "conversation.item.created"
	ConversationEventTypeSessionProviderCreated          ConversationEventType = "session.provider.created"
	ConversationEventTypeSessionProviderUpdated          ConversationEventType = "session.provider.updated"
	ConversationEventTypeSpeechInputTranscriptCompleted  ConversationEventType = "speech.input_transcript.completed"
	ConversationEventTypeSpeechInputTranscriptFailed     ConversationEventType = "speech.input_transcript.failed"
	ConversationEventTypeSpeechOutputTranscriptCompleted ConversationEventType = "speech.output_transcript.completed"
	ConversationEventTypeAssistantRespCreated            ConversationEventType = "assistant.response.created"
	ConversationEventTypeAssistantRespCompleted          ConversationEventType = "assistant.response.completed"
	ConversationEventTypeAssistantOutputTextCompleted    ConversationEventType = "assistant.output_text.completed"
	ConversationEventTypeAssistantOutputItemCompleted    ConversationEventType = "assistant.output_item.completed"
	ConversationEventTypeToolArgumentsCompleted          ConversationEventType = "tool.arguments.completed"
	ConversationEventTypeToolCallCompleted               ConversationEventType = "tool.call.completed"
	ConversationEventTypeToolCallFailed                  ConversationEventType = "tool.call.failed"
	ConversationEventTypeToolListCompleted               ConversationEventType = "tool.list.completed"
	ConversationEventTypeToolListFailed                  ConversationEventType = "tool.list.failed"
	ConversationEventTypeError                           ConversationEventType = "error"
)

type ProviderEventType string
