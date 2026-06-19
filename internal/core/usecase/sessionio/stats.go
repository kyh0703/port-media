package sessionio

type RuntimeStatsResponse struct {
	Rooms           int
	Sessions        int
	Participants    int
	Tracks          int
	ByStatus        map[string]int
	ByConnection    map[string]int
	ByMedia         map[string]int
	ByRole          map[string]int
	ByAudioMode     map[string]int
	ByRealtimeEvent map[string]int
	RoomsDetail     []RuntimeRoomStatDetail
}
