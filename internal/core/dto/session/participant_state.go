package session

type ParticipantStateResponse struct {
	ID              string `json:"id"`
	Role            string `json:"role"`
	AudioMode       string `json:"audio_mode"`
	ConnectionState string `json:"connection_state"`
	Tracks          int    `json:"tracks"`
}
