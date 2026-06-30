package sessionio

import "time"

type GetSessionStatusRequest struct {
	SessionID string
}

type GetSessionStatusResult struct {
	SessionID             string
	ConversationID        string
	UserID                string
	RoomID                string
	Status                string
	ConnectionState       string
	MediaState            string
	Participants          int
	ParticipantStates     []ParticipantStateResult
	LastRealtimeEventType string
	LastRealtimeEventAt   time.Time
	RecentRealtimeEvents  []RealtimeEventResult
	StartedAt             time.Time
	LastActiveAt          time.Time
}
