package sessionio

type RuntimeRoomStatDetail struct {
	RoomID               string
	SessionID            string
	ConversationID       string
	Status               string
	ConnectionState      string
	MediaState           string
	Participants         int
	Publishers           int
	Listeners            int
	LastRuntimeEventType string
	LastRuntimeEventAt   string
	Tracks               int
}
