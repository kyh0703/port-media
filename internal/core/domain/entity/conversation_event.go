package entity

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type ConversationEvent struct {
	SchemaVersion     int
	EventID           string
	ConversationID    vo.ConversationID
	SessionID         vo.SessionID
	RoomID            vo.RoomID
	ProviderCallID    string
	ProviderEventType string
	OccurredAt        time.Time
	Payload           string
}
