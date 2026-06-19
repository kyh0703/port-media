package sessionio

type LeaveParticipantRequest struct {
	SessionID      string
	ConversationID string
	UserID         string
	ParticipantID  string
}

type LeaveParticipantResponse struct {
	SessionID     string
	RoomID        string
	ParticipantID string
	Status        string
}
