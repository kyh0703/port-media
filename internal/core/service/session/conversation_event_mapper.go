package session

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type conversationEventMapper struct {
	realtime realtimeEventPolicy
}

func (m conversationEventMapper) Map(
	room entity.Room,
	eventType string,
	payload string,
	occurredAt time.Time,
) (entity.ConversationEvent, bool) {
	if !m.realtime.IsPublishable(eventType) {
		return entity.ConversationEvent{}, false
	}

	eventID := m.realtime.ID(payload)
	sanitizedPayload := m.realtime.SanitizePayload(payload)
	providerCallID := openAIProviderCallID(room)
	if eventID == "" {
		eventID = fallbackConversationEventID(room.SessionID, providerCallID, eventType, sanitizedPayload)
	}

	return entity.ConversationEvent{
		SchemaVersion:     1,
		EventID:           eventID,
		ConversationID:    room.ConversationID,
		SessionID:         room.SessionID,
		RoomID:            room.ID,
		ProviderCallID:    providerCallID,
		ProviderEventType: eventType,
		OccurredAt:        occurredAt,
		Payload:           sanitizedPayload,
	}, true
}

func openAIProviderCallID(room entity.Room) string {
	for _, participant := range room.Participants {
		if participant.Role == vo.ParticipantRoleOpenAIAgent {
			return participant.ProviderCallID
		}
	}
	return ""
}
