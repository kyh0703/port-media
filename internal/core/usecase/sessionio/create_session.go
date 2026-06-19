package sessionio

type CreateSessionRequest struct {
	SessionID      string
	ConversationID string
}

type CreateSessionResponse struct {
	SessionID      string
	ConversationID string
	RoomID         string
	Status         string
}
