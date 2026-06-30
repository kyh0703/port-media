package sessionio

type JoinSessionCommand struct {
	SessionID       string
	ConversationID  string
	ParticipantID   string
	ParticipantRole string
	UserID          string
	SDP             string
	AudioMode       AudioMode
}

type JoinSessionResult struct {
	SDPAnswer     string
	RoomID        string
	ParticipantID string
}

func (r JoinSessionCommand) PublishesAudio() bool {
	return r.AudioMode != AudioModeListener
}
