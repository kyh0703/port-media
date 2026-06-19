package sessionio

type ParticipantStateResult struct {
	ID              string
	Role            string
	AudioMode       string
	ConnectionState string
	Tracks          int
}
