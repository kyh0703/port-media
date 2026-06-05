package entity

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type MediaSessionRecord struct {
	ID                    vo.RoomID
	SessionID             vo.SessionID
	ConversationID        vo.ConversationID
	UserID                string
	Status                vo.RoomStatus
	LastRealtimeEventType string
	LastRealtimeEventAt   time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func NewMediaSessionRecordFromRoom(room Room) MediaSessionRecord {
	return MediaSessionRecord{
		ID:                    room.ID,
		SessionID:             room.SessionID,
		ConversationID:        room.ConversationID,
		UserID:                room.UserID,
		Status:                room.Status,
		LastRealtimeEventType: room.LastRealtimeEventType,
		LastRealtimeEventAt:   room.LastRealtimeEventAt,
		CreatedAt:             room.CreatedAt,
		UpdatedAt:             room.UpdatedAt,
	}
}

func (r MediaSessionRecord) RuntimeRoom() Room {
	return Room{
		ID:                    r.ID,
		SessionID:             r.SessionID,
		ConversationID:        r.ConversationID,
		UserID:                r.UserID,
		Status:                r.Status,
		Participants:          make(map[vo.ParticipantID]Participant),
		LastRealtimeEventType: r.LastRealtimeEventType,
		LastRealtimeEventAt:   r.LastRealtimeEventAt,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}
}
