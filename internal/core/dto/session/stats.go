package session

type RuntimeStatsResponse struct {
	Rooms           int                     `json:"rooms"`
	Sessions        int                     `json:"sessions"`
	Participants    int                     `json:"participants"`
	Tracks          int                     `json:"tracks"`
	ByStatus        map[string]int          `json:"by_status"`
	ByConnection    map[string]int          `json:"by_connection"`
	ByMedia         map[string]int          `json:"by_media"`
	ByRole          map[string]int          `json:"by_role"`
	ByAudioMode     map[string]int          `json:"by_audio_mode"`
	ByRealtimeEvent map[string]int          `json:"by_realtime_event"`
	RoomsDetail     []RuntimeRoomStatDetail `json:"rooms_detail"`
}
