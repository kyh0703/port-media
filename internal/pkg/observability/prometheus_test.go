package observability

import (
	"strings"
	"testing"

	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
)

func TestRuntimeStatsPrometheusRendersStableMetrics(t *testing.T) {
	output := RuntimeStatsPrometheus(sessiondto.RuntimeStatsResponse{
		Rooms:           1,
		Sessions:        1,
		Participants:    2,
		Tracks:          3,
		ByStatus:        map[string]int{"active": 1},
		ByConnection:    map[string]int{"connected": 1},
		ByMedia:         map[string]int{"active": 1},
		ByRole:          map[string]int{"client": 1, "openai_agent": 1},
		ByAudioMode:     map[string]int{"publisher": 1},
		ByRealtimeEvent: map[string]int{"response.done": 1},
		RoomsDetail: []sessiondto.RuntimeRoomStatDetail{
			{
				RoomID:          `room"1`,
				SessionID:       "session-1",
				ConversationID:  "conversation-1",
				Status:          "active",
				ConnectionState: "connected",
				MediaState:      "active",
				Participants:    2,
				Publishers:      1,
				Listeners:       1,
				Tracks:          3,
			},
		},
	})

	assertContains(t, output, "# TYPE dubu_media_rooms gauge")
	assertContains(t, output, "dubu_media_rooms 1")
	assertContains(t, output, `dubu_media_participants_by_role{role="openai_agent"} 1`)
	assertContains(t, output, `dubu_media_rooms_by_realtime_event{event_type="response.done"} 1`)
	assertContains(t, output, `dubu_media_room_participants{connection_state="connected",conversation_id="conversation-1",media_state="active",room_id="room\"1",session_id="session-1",status="active"} 2`)
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("output missing %q:\n%s", needle, haystack)
	}
}
