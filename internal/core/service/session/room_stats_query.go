package session

import (
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
)

type roomStatsQuery struct{}

func (roomStatsQuery) Build(rooms []entity.Room) sessiondto.RuntimeStatsResponse {
	stats := sessiondto.RuntimeStatsResponse{
		Rooms:           len(rooms),
		Sessions:        len(rooms),
		ByStatus:        make(map[string]int),
		ByConnection:    make(map[string]int),
		ByMedia:         make(map[string]int),
		ByRole:          make(map[string]int),
		ByAudioMode:     make(map[string]int),
		ByRealtimeEvent: make(map[string]int),
		RoomsDetail:     make([]sessiondto.RuntimeRoomStatDetail, 0, len(rooms)),
	}

	for _, room := range rooms {
		connectionState := roomConnectionState(room)
		mediaState := roomMediaState(room)
		trackCount := countTracks(room)

		stats.Participants += room.ParticipantCount()
		stats.Tracks += trackCount
		stats.ByStatus[string(room.Status)]++
		stats.ByConnection[string(connectionState)]++
		stats.ByMedia[string(mediaState)]++
		if room.LastRealtimeEventType != "" {
			stats.ByRealtimeEvent[room.LastRealtimeEventType]++
		}
		publishers, listeners := countClientAudioModes(room)
		for _, participant := range room.Participants() {
			stats.ByRole[string(participant.Role)]++
			if participant.Role == vo.ParticipantRoleClient {
				stats.ByAudioMode[participantAudioMode(participant)]++
			}
		}
		stats.RoomsDetail = append(stats.RoomsDetail, sessiondto.RuntimeRoomStatDetail{
			RoomID:                string(room.ID),
			SessionID:             string(room.SessionID),
			ConversationID:        string(room.ConversationID),
			Status:                string(room.Status),
			ConnectionState:       string(connectionState),
			MediaState:            string(mediaState),
			Participants:          room.ParticipantCount(),
			Publishers:            publishers,
			Listeners:             listeners,
			LastRealtimeEventType: room.LastRealtimeEventType,
			LastRealtimeEventAt:   formatOptionalTime(room.LastRealtimeEventAt),
			Tracks:                trackCount,
		})
	}

	return stats
}
