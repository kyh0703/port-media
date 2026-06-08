package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/redis/go-redis/v9"
)

func TestRedisMediaServerStateRepositorySavesState(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() {
		_ = client.Close()
	}()

	repo := NewRedisMediaServerStateRepository(client, &configs.Config{
		MediaServer: configs.MediaServerConfig{
			HeartbeatTTL: 15 * time.Second,
		},
	})
	updatedAt := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

	err = repo.Save(context.Background(), entity.MediaServerState{
		ID:                 1,
		URL:                "http://media-a.internal:8080",
		Status:             entity.MediaServerStatusHealthy,
		ActiveRooms:        2,
		ActiveSessions:     2,
		ActiveParticipants: 5,
		ActiveTracks:       3,
		MaxSessions:        10,
		UpdatedAt:          updatedAt,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	members, err := client.SMembers(context.Background(), mediaServersSetKey).Result()
	if err != nil {
		t.Fatalf("SMembers() error = %v", err)
	}
	if len(members) != 1 || members[0] != "1" {
		t.Fatalf("members = %v, want [1]", members)
	}

	raw, err := client.Get(context.Background(), mediaServerStateKey(1)).Bytes()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	var payload mediaServerStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.ID != 1 {
		t.Fatalf("payload.ID = %d, want 1", payload.ID)
	}
	if payload.URL != "http://media-a.internal:8080" {
		t.Fatalf("payload.URL = %q", payload.URL)
	}
	if payload.Status != "healthy" {
		t.Fatalf("payload.Status = %q, want healthy", payload.Status)
	}
	if payload.ActiveSessions != 2 {
		t.Fatalf("payload.ActiveSessions = %d, want 2", payload.ActiveSessions)
	}
	if payload.ActiveParticipants != 5 {
		t.Fatalf("payload.ActiveParticipants = %d, want 5", payload.ActiveParticipants)
	}
	if payload.MaxSessions != 10 {
		t.Fatalf("payload.MaxSessions = %d, want 10", payload.MaxSessions)
	}
	if !payload.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("payload.UpdatedAt = %s", payload.UpdatedAt)
	}

	ttl, err := client.TTL(context.Background(), mediaServerStateKey(1)).Result()
	if err != nil {
		t.Fatalf("TTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 15*time.Second {
		t.Fatalf("ttl = %s, want between 0s and 15s", ttl)
	}
}

func TestRedisMediaServerStateRepositorySavesOfflineState(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() {
		_ = client.Close()
	}()

	repo := NewRedisMediaServerStateRepository(client, &configs.Config{
		MediaServer: configs.MediaServerConfig{
			HeartbeatTTL: 15 * time.Second,
		},
	})
	state := entity.MediaServerState{
		ID:        1,
		URL:       "http://media-a.internal:8080",
		Status:    entity.MediaServerStatusHealthy,
		UpdatedAt: time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC),
	}

	if err := repo.Save(context.Background(), state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := repo.SaveOffline(context.Background(), state); err != nil {
		t.Fatalf("SaveOffline() error = %v", err)
	}

	members, err := client.SMembers(context.Background(), mediaServersSetKey).Result()
	if err != nil {
		t.Fatalf("SMembers() error = %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("members = %v, want empty", members)
	}

	raw, err := client.Get(context.Background(), mediaServerStateKey(1)).Bytes()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	var payload mediaServerStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Status != "offline" {
		t.Fatalf("payload.Status = %q, want offline", payload.Status)
	}
}
