package dto

type CreateSessionRequest struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
}

type GetSessionStatusResponse struct {
	SessionID             string                     `json:"session_id"`
	ConversationID        string                     `json:"conversation_id"`
	UserID                string                     `json:"user_id"`
	RoomID                string                     `json:"room_id"`
	Status                string                     `json:"status"`
	ConnectionState       string                     `json:"connection_state"`
	MediaState            string                     `json:"media_state"`
	Participants          int                        `json:"participants"`
	ParticipantStates     []ParticipantStateResponse `json:"participant_states"`
	LastRealtimeEventType string                     `json:"last_realtime_event_type"`
	LastRealtimeEventAt   string                     `json:"last_realtime_event_at"`
	RecentRealtimeEvents  []RealtimeEventResponse    `json:"recent_realtime_events"`
	StartedAt             string                     `json:"started_at"`
	LastActiveAt          string                     `json:"last_active_at"`
}

type ParticipantStateResponse struct {
	ID              string `json:"id"`
	Role            string `json:"role"`
	AudioMode       string `json:"audio_mode"`
	ConnectionState string `json:"connection_state"`
	Tracks          int    `json:"tracks"`
}

type RealtimeEventResponse struct {
	Type string `json:"type"`
	At   string `json:"at"`
}

type CreateSessionResponse struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	RoomID         string `json:"room_id"`
	Status         string `json:"status"`
}

type LeaveParticipantResponse struct {
	SessionID     string `json:"session_id"`
	RoomID        string `json:"room_id"`
	ParticipantID string `json:"participant_id"`
	Status        string `json:"status"`
}

type EndSessionResponse struct {
	SessionID string `json:"session_id"`
	RoomID    string `json:"room_id"`
	Status    string `json:"status"`
}
