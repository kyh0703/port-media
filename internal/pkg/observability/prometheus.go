package observability

import (
	"fmt"
	"sort"
	"strings"

	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
)

func RuntimeStatsPrometheus(stats sessiondto.RuntimeStatsResponse) string {
	var b strings.Builder

	writeGauge(&b, "dubu_media_rooms", "Active runtime rooms.", nil, stats.Rooms)
	writeGauge(&b, "dubu_media_sessions", "Active runtime sessions.", nil, stats.Sessions)
	writeGauge(&b, "dubu_media_participants", "Active runtime participants.", nil, stats.Participants)
	writeGauge(&b, "dubu_media_tracks", "Active runtime tracks.", nil, stats.Tracks)

	writeGroupedGauges(&b, "dubu_media_rooms_by_status", "Runtime rooms grouped by lifecycle status.", "status", stats.ByStatus)
	writeGroupedGauges(&b, "dubu_media_rooms_by_connection_state", "Runtime rooms grouped by aggregate WebRTC connection state.", "state", stats.ByConnection)
	writeGroupedGauges(&b, "dubu_media_rooms_by_media_state", "Runtime rooms grouped by aggregate media state.", "state", stats.ByMedia)
	writeGroupedGauges(&b, "dubu_media_participants_by_role", "Runtime participants grouped by role.", "role", stats.ByRole)
	writeGroupedGauges(&b, "dubu_media_clients_by_audio_mode", "Client participants grouped by audio mode.", "mode", stats.ByAudioMode)
	writeGroupedGauges(&b, "dubu_media_rooms_by_realtime_event", "Runtime rooms grouped by latest OpenAI Realtime event type.", "event_type", stats.ByRealtimeEvent)

	writeRoomGauges(&b, stats.RoomsDetail)

	return b.String()
}

func writeGroupedGauges(b *strings.Builder, name string, help string, labelName string, values map[string]int) {
	if len(values) == 0 {
		return
	}

	writeHelpAndType(b, name, help)
	for _, key := range sortedKeys(values) {
		writeSample(b, name, map[string]string{labelName: key}, values[key])
	}
}

func writeRoomGauges(b *strings.Builder, rooms []sessiondto.RuntimeRoomStatDetail) {
	if len(rooms) == 0 {
		return
	}

	sort.Slice(rooms, func(i, j int) bool {
		if rooms[i].SessionID == rooms[j].SessionID {
			return rooms[i].RoomID < rooms[j].RoomID
		}
		return rooms[i].SessionID < rooms[j].SessionID
	})

	writeHelpAndType(b, "dubu_media_room_participants", "Participant count per runtime room.")
	for _, room := range rooms {
		writeSample(b, "dubu_media_room_participants", roomLabels(room), room.Participants)
	}

	writeHelpAndType(b, "dubu_media_room_publishers", "Publisher client count per runtime room.")
	for _, room := range rooms {
		writeSample(b, "dubu_media_room_publishers", roomLabels(room), room.Publishers)
	}

	writeHelpAndType(b, "dubu_media_room_listeners", "Listener client count per runtime room.")
	for _, room := range rooms {
		writeSample(b, "dubu_media_room_listeners", roomLabels(room), room.Listeners)
	}

	writeHelpAndType(b, "dubu_media_room_tracks", "Track count per runtime room.")
	for _, room := range rooms {
		writeSample(b, "dubu_media_room_tracks", roomLabels(room), room.Tracks)
	}
}

func roomLabels(room sessiondto.RuntimeRoomStatDetail) map[string]string {
	return map[string]string{
		"room_id":          room.RoomID,
		"session_id":       room.SessionID,
		"conversation_id":  room.ConversationID,
		"status":           room.Status,
		"connection_state": room.ConnectionState,
		"media_state":      room.MediaState,
	}
}

func writeGauge(b *strings.Builder, name string, help string, labels map[string]string, value int) {
	writeHelpAndType(b, name, help)
	writeSample(b, name, labels, value)
}

func writeHelpAndType(b *strings.Builder, name string, help string) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s gauge\n", name)
}

func writeSample(b *strings.Builder, name string, labels map[string]string, value int) {
	b.WriteString(name)
	if len(labels) > 0 {
		b.WriteString("{")
		keys := sortedKeys(labels)
		for i, key := range keys {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(b, `%s="%s"`, key, escapeLabelValue(labels[key]))
		}
		b.WriteString("}")
	}
	fmt.Fprintf(b, " %d\n", value)
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
