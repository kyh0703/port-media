package session

type RuntimeRoomStatDetail struct {
	RoomID                string `json:"room_id"`
	SessionID             string `json:"session_id"`
	ConversationID        string `json:"conversation_id"`
	Status                string `json:"status"`
	ConnectionState       string `json:"connection_state"`
	MediaState            string `json:"media_state"`
	Participants          int    `json:"participants"`
	Publishers            int    `json:"publishers"`
	Listeners             int    `json:"listeners"`
	LastRealtimeEventType string `json:"last_realtime_event_type"`
	LastRealtimeEventAt   string `json:"last_realtime_event_at"`
	Tracks                int    `json:"tracks"`
}
