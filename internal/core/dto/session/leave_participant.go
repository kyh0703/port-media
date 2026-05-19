package session

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
