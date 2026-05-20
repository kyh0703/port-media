package session

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kyh0703/portfoilo-media/configs"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestMediaServerHeartbeatWritesRedisState(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() {
		_ = client.Close()
	}()

	cfg := &configs.Config{
		NodeID: "media-a",
		Server: configs.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		MediaServer: configs.MediaServerConfig{
			URL:               "http://media-a.internal:8080",
			Status:            "healthy",
			HeartbeatInterval: 5 * time.Second,
			HeartbeatTTL:      15 * time.Second,
			MaxSessions:       10,
		},
	}
	svc := fakeHeartbeatService{
		stats: sessiondto.RuntimeStatsResponse{
			Rooms:        2,
			Sessions:     2,
			Participants: 5,
			Tracks:       3,
		},
	}
	heartbeat := NewMediaServerHeartbeat(client, svc, cfg, zap.NewNop())
	heartbeat.now = func() time.Time {
		return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	}

	if err := heartbeat.write(context.Background(), "healthy"); err != nil {
		t.Fatalf("write() error = %v", err)
	}

	members, err := client.SMembers(context.Background(), mediaServersSetKey).Result()
	if err != nil {
		t.Fatalf("SMembers() error = %v", err)
	}
	if len(members) != 1 || members[0] != "media-a" {
		t.Fatalf("members = %v, want [media-a]", members)
	}

	raw, err := client.Get(context.Background(), mediaServerStateKey("media-a")).Bytes()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	var payload mediaServerHeartbeatPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.ID != "media-a" {
		t.Fatalf("payload.ID = %q, want media-a", payload.ID)
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
	if !payload.UpdatedAt.Equal(time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("payload.UpdatedAt = %s", payload.UpdatedAt)
	}

	ttl, err := client.TTL(context.Background(), mediaServerStateKey("media-a")).Result()
	if err != nil {
		t.Fatalf("TTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 15*time.Second {
		t.Fatalf("ttl = %s, want between 0s and 15s", ttl)
	}
}

func TestMediaServerHeartbeatWriteOfflineRemovesServerFromDiscoverySet(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() {
		_ = client.Close()
	}()

	cfg := &configs.Config{
		NodeID: "media-a",
		Server: configs.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		MediaServer: configs.MediaServerConfig{
			URL:          "http://media-a.internal:8080",
			HeartbeatTTL: 15 * time.Second,
		},
	}
	heartbeat := NewMediaServerHeartbeat(client, fakeHeartbeatService{}, cfg, zap.NewNop())

	if err := heartbeat.write(context.Background(), "healthy"); err != nil {
		t.Fatalf("write() error = %v", err)
	}
	if err := heartbeat.writeOffline(context.Background()); err != nil {
		t.Fatalf("writeOffline() error = %v", err)
	}

	members, err := client.SMembers(context.Background(), mediaServersSetKey).Result()
	if err != nil {
		t.Fatalf("SMembers() error = %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("members = %v, want empty", members)
	}

	raw, err := client.Get(context.Background(), mediaServerStateKey("media-a")).Bytes()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	var payload mediaServerHeartbeatPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Status != "offline" {
		t.Fatalf("payload.Status = %q, want offline", payload.Status)
	}
}

func TestMediaServerHeartbeatDefaultsURLFromServerConfig(t *testing.T) {
	cfg := &configs.Config{
		Server: configs.ServerConfig{
			Host: "0.0.0.0",
			Port: 9090,
		},
	}

	if got := mediaServerURL(cfg); got != "http://localhost:9090" {
		t.Fatalf("mediaServerURL() = %q, want http://localhost:9090", got)
	}
}

func TestMediaServerHeartbeatUsesConfigNodeID(t *testing.T) {
	cfg := &configs.Config{
		NodeID: "node-a",
		Server: configs.ServerConfig{
			Port: 8080,
		},
	}

	if got := mediaServerID(cfg); got != "node-a" {
		t.Fatalf("mediaServerID() = %q, want node-a", got)
	}
}

type fakeHeartbeatService struct {
	stats sessiondto.RuntimeStatsResponse
}

func (f fakeHeartbeatService) CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error) {
	return sessiondto.CreateSessionResponse{}, nil
}

func (f fakeHeartbeatService) AcceptOffer(ctx context.Context, req sessiondto.AcceptOfferRequest) (sessiondto.AcceptOfferResponse, error) {
	return sessiondto.AcceptOfferResponse{}, nil
}

func (f fakeHeartbeatService) LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error) {
	return sessiondto.LeaveParticipantResponse{}, nil
}

func (f fakeHeartbeatService) EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error) {
	return sessiondto.EndSessionResponse{}, nil
}

func (f fakeHeartbeatService) CleanupIdleRooms(ctx context.Context, idleTimeout time.Duration) (int, error) {
	return 0, nil
}

func (f fakeHeartbeatService) ShutdownActiveRooms(ctx context.Context) (int, error) {
	return 0, nil
}

func (f fakeHeartbeatService) GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error) {
	return sessiondto.GetSessionStatusResponse{}, false, nil
}

func (f fakeHeartbeatService) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	return f.stats, nil
}

func (f fakeHeartbeatService) GetHealth(ctx context.Context) error {
	return nil
}
