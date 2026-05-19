package session

type GetSessionStatusRequest struct {
	SessionID string
}

type GetSessionStatusResponse struct {
	SessionID             string                     `json:"session_id"`
	ConversationID        string                     `json:"conversation_id"`
	UserID                string                     `json:"user_id"`
	RoomID                string                     `json:"room_id"`
	Status                string                     `json:"status"`
	ConnectionState       string                     `json:"connection_state"`
	MediaState            string                     `json:"media_state"`
	OpenAIProviderCallID  string                     `json:"openai_provider_call_id"`
	Participants          int                        `json:"participants"`
	ParticipantStates     []ParticipantStateResponse `json:"participant_states"`
	LastRealtimeEventType string                     `json:"last_realtime_event_type"`
	LastRealtimeEventAt   string                     `json:"last_realtime_event_at"`
	RecentRealtimeEvents  []RealtimeEventResponse    `json:"recent_realtime_events"`
	StartedAt             string                     `json:"started_at"`
	LastActiveAt          string                     `json:"last_active_at"`
}
