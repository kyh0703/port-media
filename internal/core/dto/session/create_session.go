package session

type CreateSessionRequest struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
}

type CreateSessionResponse struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	RoomID         string `json:"room_id"`
	Status         string `json:"status"`
}

type AcceptOfferRequest struct {
	SessionID      string
	ConversationID string
	UserID         string
	SDP            string
	AudioMode      AudioMode
}

type AcceptOfferResponse struct {
	SDPAnswer     string
	RoomID        string
	ParticipantID string
}

type AudioMode string

const (
	AudioModePublisher AudioMode = "publisher"
	AudioModeListener  AudioMode = "listener"
)

func (r AcceptOfferRequest) PublishesAudio() bool {
	return r.AudioMode != AudioModeListener
}

type EndSessionRequest struct {
	SessionID      string
	ConversationID string
	UserID         string
}

type EndSessionResponse struct {
	SessionID string `json:"session_id"`
	RoomID    string `json:"room_id"`
	Status    string `json:"status"`
}

type LeaveParticipantRequest struct {
	SessionID      string
	ConversationID string
	UserID         string
	ParticipantID  string
}

type LeaveParticipantResponse struct {
	SessionID     string `json:"session_id"`
	RoomID        string `json:"room_id"`
	ParticipantID string `json:"participant_id"`
	Status        string `json:"status"`
}

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

type RealtimeEventResponse struct {
	Type string `json:"type"`
	At   string `json:"at"`
}

type ParticipantStateResponse struct {
	ID              string `json:"id"`
	Role            string `json:"role"`
	AudioMode       string `json:"audio_mode"`
	ConnectionState string `json:"connection_state"`
	Tracks          int    `json:"tracks"`
}
