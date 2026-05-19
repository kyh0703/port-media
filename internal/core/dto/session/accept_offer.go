package session

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

func (r AcceptOfferRequest) PublishesAudio() bool {
	return r.AudioMode != AudioModeListener
}
