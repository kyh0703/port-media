package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/redis/go-redis/v9"
)

func TestRedisMediaSessionStateRepositorySavesStateWithTTL(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	repo := NewRedisMediaSessionStateRepository(client, &configs.Config{
		Realtime: configs.RealtimeConfig{RoomIdleTimeout: 2 * time.Minute},
	})

	startedAt := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	lastActiveAt := time.Date(2026, 5, 19, 10, 1, 0, 0, time.UTC)
	err = repo.Save(context.Background(), entity.MediaSessionState{
		SessionID:            vo.SessionID("session-1"),
		ConversationID:       vo.ConversationID("conversation-1"),
		UserID:               "user-1",
		RoomID:               vo.RoomID("room-1"),
		Status:               vo.RoomStatusActive,
		ConnectionState:      vo.ConnectionStateConnected,
		MediaState:           vo.TrackStateActive,
		OpenAIProviderCallID: "rtc_123",
		Participants:         2,
		ParticipantStates: []entity.MediaSessionParticipantState{
			{
				ID:              vo.ParticipantID("client-1"),
				Role:            vo.ParticipantRoleClient,
				AudioMode:       "publisher",
				ConnectionState: vo.ConnectionStateConnected,
				Tracks:          1,
			},
		},
		LastRealtimeEventType: "response.done",
		LastRealtimeEventAt:   lastActiveAt,
		RecentRealtimeEvents: []entity.RealtimeEvent{
			{Type: "session.updated", At: startedAt},
			{Type: "response.done", At: lastActiveAt},
		},
		StartedAt: startedAt,
		UpdatedAt: lastActiveAt,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := server.Get("media:session:session-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v raw=%s", err, raw)
	}

	if payload["status"] != string(vo.RoomStatusActive) {
		t.Fatalf("status = %v, want %q", payload["status"], vo.RoomStatusActive)
	}
	if payload["connection_state"] != string(vo.ConnectionStateConnected) {
		t.Fatalf("connection_state = %v, want %q", payload["connection_state"], vo.ConnectionStateConnected)
	}
	if payload["media_state"] != string(vo.TrackStateActive) {
		t.Fatalf("media_state = %v, want %q", payload["media_state"], vo.TrackStateActive)
	}
	if payload["user_id"] != "user-1" {
		t.Fatalf("user_id = %v, want user-1", payload["user_id"])
	}
	if payload["last_active_at"] != lastActiveAt.Format(time.RFC3339Nano) {
		t.Fatalf("last_active_at = %v, want %s", payload["last_active_at"], lastActiveAt.Format(time.RFC3339Nano))
	}
	participantStates, ok := payload["participant_states"].([]any)
	if !ok || len(participantStates) != 1 {
		t.Fatalf("participant_states = %#v, want one item", payload["participant_states"])
	}
	participant, ok := participantStates[0].(map[string]any)
	if !ok {
		t.Fatalf("participant state = %#v, want object", participantStates[0])
	}
	if participant["audio_mode"] != "publisher" {
		t.Fatalf("participant audio_mode = %v, want publisher", participant["audio_mode"])
	}
	if payload["last_realtime_event_type"] != "response.done" {
		t.Fatalf("last_realtime_event_type = %v, want response.done", payload["last_realtime_event_type"])
	}
	if payload["last_realtime_event_at"] != lastActiveAt.Format(time.RFC3339Nano) {
		t.Fatalf("last_realtime_event_at = %v, want %s", payload["last_realtime_event_at"], lastActiveAt.Format(time.RFC3339Nano))
	}
	realtimeEvents, ok := payload["recent_realtime_events"].([]any)
	if !ok || len(realtimeEvents) != 2 {
		t.Fatalf("recent_realtime_events = %#v, want two items", payload["recent_realtime_events"])
	}
	lastRealtimeEvent, ok := realtimeEvents[1].(map[string]any)
	if !ok {
		t.Fatalf("recent realtime event = %#v, want object", realtimeEvents[1])
	}
	if lastRealtimeEvent["type"] != "response.done" {
		t.Fatalf("recent realtime event type = %v, want response.done", lastRealtimeEvent["type"])
	}
	if server.TTL("media:session:session-1") != 2*time.Minute {
		t.Fatalf("ttl = %v, want 2m", server.TTL("media:session:session-1"))
	}
}

func TestRedisMediaSessionStateRepositoryFindsState(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	repo := NewRedisMediaSessionStateRepository(client, &configs.Config{
		Realtime: configs.RealtimeConfig{RoomIdleTimeout: 2 * time.Minute},
	})

	startedAt := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 19, 10, 1, 0, 0, time.UTC)
	err = repo.Save(context.Background(), entity.MediaSessionState{
		SessionID:            vo.SessionID("session-1"),
		ConversationID:       vo.ConversationID("conversation-1"),
		UserID:               "user-1",
		RoomID:               vo.RoomID("room-1"),
		Status:               vo.RoomStatusActive,
		ConnectionState:      vo.ConnectionStateConnected,
		MediaState:           vo.TrackStateActive,
		OpenAIProviderCallID: "rtc_123",
		Participants:         2,
		ParticipantStates: []entity.MediaSessionParticipantState{
			{
				ID:              vo.ParticipantID("client-1"),
				Role:            vo.ParticipantRoleClient,
				AudioMode:       "listener",
				ConnectionState: vo.ConnectionStateConnected,
				Tracks:          1,
			},
		},
		LastRealtimeEventType: "session.updated",
		LastRealtimeEventAt:   updatedAt,
		RecentRealtimeEvents: []entity.RealtimeEvent{
			{Type: "session.updated", At: updatedAt},
		},
		StartedAt: startedAt,
		UpdatedAt: updatedAt,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	state, found, err := repo.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if state.SessionID != vo.SessionID("session-1") {
		t.Fatalf("SessionID = %q, want session-1", state.SessionID)
	}
	if state.UpdatedAt != updatedAt {
		t.Fatalf("UpdatedAt = %v, want %v", state.UpdatedAt, updatedAt)
	}
	if len(state.ParticipantStates) != 1 {
		t.Fatalf("ParticipantStates len = %d, want 1", len(state.ParticipantStates))
	}
	if state.ParticipantStates[0].AudioMode != "listener" {
		t.Fatalf("AudioMode = %q, want listener", state.ParticipantStates[0].AudioMode)
	}
	if state.LastRealtimeEventType != "session.updated" {
		t.Fatalf("LastRealtimeEventType = %q, want session.updated", state.LastRealtimeEventType)
	}
	if state.LastRealtimeEventAt != updatedAt {
		t.Fatalf("LastRealtimeEventAt = %v, want %v", state.LastRealtimeEventAt, updatedAt)
	}
	if len(state.RecentRealtimeEvents) != 1 {
		t.Fatalf("RecentRealtimeEvents len = %d, want 1", len(state.RecentRealtimeEvents))
	}
	if state.RecentRealtimeEvents[0].Type != "session.updated" {
		t.Fatalf("RecentRealtimeEvents[0].Type = %q, want session.updated", state.RecentRealtimeEvents[0].Type)
	}
}
