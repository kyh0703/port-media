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
