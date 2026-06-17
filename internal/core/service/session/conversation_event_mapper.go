package session

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
)

type conversationEventBuilder struct{}

func (conversationEventBuilder) Build(
	room entity.Room,
	signal port.ConversationSignal,
	occurredAt time.Time,
) (port.ConversationEvent, bool) {
	if !signal.Publishable {
		return port.ConversationEvent{}, false
	}

	eventID := signal.ProviderEventID
	providerCallID := openAIProviderCallID(room)
	if eventID == "" {
		eventID = fallbackConversationEventID(room.SessionID, providerCallID, string(signal.Type), signal.Payload)
	}

	return port.ConversationEvent{
		SchemaVersion:          1,
		EventID:                eventID,
		ConversationID:         room.ConversationID,
		SessionID:              room.SessionID,
		RoomID:                 room.ID,
		ProviderCallID:         providerCallID,
		Type:                   signal.Type,
		ProviderEventType:      signal.ProviderEventType,
		ProviderItemID:         signal.ProviderItemID,
		PreviousProviderItemID: signal.PreviousProviderItemID,
		ProviderRespID:         signal.ProviderRespID,
		OccurredAt:             occurredAt,
		Payload:                signal.Payload,
	}, true
}

func openAIProviderCallID(room entity.Room) string {
	for _, participant := range room.Participants() {
		if participant.Role == vo.ParticipantRoleOpenAIAgent {
			return participant.ProviderCallID
		}
	}
	return ""
}
