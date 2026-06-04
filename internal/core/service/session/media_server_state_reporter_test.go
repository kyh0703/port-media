package session

import (
	"context"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
)

func TestMediaServerStateReporterReportsRuntimeState(t *testing.T) {
	cfg := &configs.Config{
		NodeID: "media-a",
		Server: configs.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		MediaServer: configs.MediaServerConfig{
			URL:         "http://media-a.internal:8080",
			Status:      "healthy",
			MaxSessions: 10,
		},
	}
	states := &fakeMediaServerStateRepository{}
	reporter := NewMediaServerStateReporter(states, fakeStateReporterService{
		stats: sessiondto.RuntimeStatsResponse{
			Rooms:        2,
			Sessions:     2,
			Participants: 5,
			Tracks:       3,
		},
	}, cfg)
	reporter.now = func() time.Time {
		return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	}

	if err := reporter.Report(context.Background()); err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	if states.saved.ID != "media-a" {
		t.Fatalf("state.ID = %q, want media-a", states.saved.ID)
	}
	if states.saved.URL != "http://media-a.internal:8080" {
		t.Fatalf("state.URL = %q", states.saved.URL)
	}
	if states.saved.Status != entity.MediaServerStatusHealthy {
		t.Fatalf("state.Status = %q, want healthy", states.saved.Status)
	}
	if states.saved.ActiveParticipants != 5 {
		t.Fatalf("state.ActiveParticipants = %d, want 5", states.saved.ActiveParticipants)
	}
	if states.saved.MaxSessions != 10 {
		t.Fatalf("state.MaxSessions = %d, want 10", states.saved.MaxSessions)
	}
}

func TestMediaServerStateReporterReportsOfflineState(t *testing.T) {
	states := &fakeMediaServerStateRepository{}
	reporter := NewMediaServerStateReporter(states, fakeStateReporterService{}, &configs.Config{
		NodeID: "media-a",
		MediaServer: configs.MediaServerConfig{
			URL: "http://media-a.internal:8080",
		},
	})

	if err := reporter.ReportOffline(context.Background()); err != nil {
		t.Fatalf("ReportOffline() error = %v", err)
	}

	if !states.offlineSaved {
		t.Fatal("offline state was not saved")
	}
	if states.saved.Status != entity.MediaServerStatusOffline {
		t.Fatalf("state.Status = %q, want offline", states.saved.Status)
	}
}

func TestMediaServerStateReporterDefaultsURLFromServerConfig(t *testing.T) {
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

func TestMediaServerStateReporterUsesConfigNodeID(t *testing.T) {
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

type fakeMediaServerStateRepository struct {
	saved        entity.MediaServerState
	offlineSaved bool
}

func (f *fakeMediaServerStateRepository) Save(ctx context.Context, state entity.MediaServerState) error {
	_ = ctx
	f.saved = state
	return nil
}

func (f *fakeMediaServerStateRepository) SaveOffline(ctx context.Context, state entity.MediaServerState) error {
	_ = ctx
	f.saved = state
	f.offlineSaved = true
	return nil
}

type fakeStateReporterService struct {
	stats sessiondto.RuntimeStatsResponse
}

func (f fakeStateReporterService) CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error) {
	return sessiondto.CreateSessionResponse{}, nil
}

func (f fakeStateReporterService) AcceptOffer(ctx context.Context, req sessiondto.AcceptOfferRequest) (sessiondto.AcceptOfferResponse, error) {
	return sessiondto.AcceptOfferResponse{}, nil
}

func (f fakeStateReporterService) LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error) {
	return sessiondto.LeaveParticipantResponse{}, nil
}

func (f fakeStateReporterService) EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error) {
	return sessiondto.EndSessionResponse{}, nil
}

func (f fakeStateReporterService) CleanupIdleRooms(ctx context.Context, idleTimeout time.Duration) (int, error) {
	return 0, nil
}

func (f fakeStateReporterService) ShutdownActiveRooms(ctx context.Context) (int, error) {
	return 0, nil
}

func (f fakeStateReporterService) GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error) {
	return sessiondto.GetSessionStatusResponse{}, false, nil
}

func (f fakeStateReporterService) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	return f.stats, nil
}

func (f fakeStateReporterService) GetHealth(ctx context.Context) error {
	return nil
}
