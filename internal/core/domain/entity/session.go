package entity

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type Session struct {
	ID             vo.SessionID
	ConversationID vo.ConversationID
	RoomID         vo.RoomID
	Status         vo.SessionStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewSession(id vo.SessionID, conversationID vo.ConversationID, roomID vo.RoomID, now time.Time) Session {
	return Session{
		ID:             id,
		ConversationID: conversationID,
		RoomID:         roomID,
		Status:         vo.SessionStatusCreated,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (s *Session) Activate(now time.Time) {
	s.Status = vo.SessionStatusActive
	s.UpdatedAt = now
}

func (s *Session) End(now time.Time) {
	s.Status = vo.SessionStatusEnded
	s.UpdatedAt = now
}

func (s *Session) Fail(now time.Time) {
	s.Status = vo.SessionStatusFailed
	s.UpdatedAt = now
}
