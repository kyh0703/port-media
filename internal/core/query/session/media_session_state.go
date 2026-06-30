package session

import (
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type MediaSessionState struct {
	SessionID            vo.SessionID
	ConversationID       vo.ConversationID
	UserID               string
	RoomID               vo.RoomID
	Status               vo.RoomStatus
	ConnectionState      vo.ConnectionState
	MediaState           vo.TrackState
	Participants         int
	ParticipantStates    []MediaSessionParticipantState
	LastRuntimeEventType string
	LastRuntimeEventAt   time.Time
	RecentRuntimeEvents  []RuntimeEvent
	StartedAt            time.Time
	UpdatedAt            time.Time
}

type RuntimeEvent struct {
	Type string
	At   time.Time
}

type MediaSessionParticipantState struct {
	ID              vo.ParticipantID
	Role            vo.ParticipantRole
	AudioMode       string
	ConnectionState vo.ConnectionState
	Tracks          int
}
