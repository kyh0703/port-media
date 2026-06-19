package sessionio

type EndSessionRequest struct {
	SessionID      string
	ConversationID string
	UserID         string
}

type EndSessionResponse struct {
	SessionID string
	RoomID    string
	Status    string
}
