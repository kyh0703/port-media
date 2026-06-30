package sessionio

import "time"

type GetSessionStatusRequest struct {
	SessionID string
}

type GetSessionStatusResult struct {
	SessionID            string
	ConversationID       string
	UserID               string
	RoomID               string
	Status               string
	ConnectionState      string
	MediaState           string
	Participants         int
	ParticipantStates    []ParticipantStateResult
	LastRuntimeEventType string
	LastRuntimeEventAt   time.Time
	RecentRuntimeEvents  []RuntimeEventResult
	StartedAt            time.Time
	LastActiveAt         time.Time
}
