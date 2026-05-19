package session

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
