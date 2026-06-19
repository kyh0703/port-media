package session

import (
	"context"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
)

func TestMediaServerStateReporterReportsRuntimeState(t *testing.T) {
	states := &fakeMediaServerStateRepository{}
	reporter := NewMediaServerStateReporter(states, fakeStateReporterService{
		stats: sessionio.RuntimeStatsResponse{
			Rooms:        2,
			Sessions:     2,
			Participants: 5,
			Tracks:       3,
		},
	}, MediaServerStateReporterOptions{
		ID:          1,
		URL:         "http://media-a.internal:8080",
		Status:      entity.MediaServerStatusHealthy,
		MaxSessions: 10,
	})
	reporter.now = func() time.Time {
		return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	}

	if err := reporter.Report(context.Background()); err != nil {
		t.Fatalf("Report() error = %v", err)
	}

	if states.saved.ID != 1 {
		t.Fatalf("state.ID = %d, want 1", states.saved.ID)
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
	reporter := NewMediaServerStateReporter(states, fakeStateReporterService{}, MediaServerStateReporterOptions{
		ID:  1,
		URL: "http://media-a.internal:8080",
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
	stats sessionio.RuntimeStatsResponse
}

func (f fakeStateReporterService) CreateSession(ctx context.Context, req sessionio.CreateSessionRequest) (sessionio.CreateSessionResponse, error) {
	return sessionio.CreateSessionResponse{}, nil
}

func (f fakeStateReporterService) LeaveParticipant(ctx context.Context, req sessionio.LeaveParticipantRequest) (sessionio.LeaveParticipantResponse, error) {
	return sessionio.LeaveParticipantResponse{}, nil
}

func (f fakeStateReporterService) EndSession(ctx context.Context, req sessionio.EndSessionRequest) (sessionio.EndSessionResponse, error) {
	return sessionio.EndSessionResponse{}, nil
}

func (f fakeStateReporterService) CleanupIdleRooms(ctx context.Context, idleTimeout time.Duration) (int, error) {
	return 0, nil
}

func (f fakeStateReporterService) ShutdownActiveRooms(ctx context.Context) (int, error) {
	return 0, nil
}

func (f fakeStateReporterService) GetSessionStatus(ctx context.Context, req sessionio.GetSessionStatusRequest) (sessionio.GetSessionStatusResult, bool, error) {
	return sessionio.GetSessionStatusResult{}, false, nil
}

func (f fakeStateReporterService) GetRuntimeStats(ctx context.Context) (sessionio.RuntimeStatsResponse, error) {
	return f.stats, nil
}

func (f fakeStateReporterService) GetHealth(ctx context.Context) error {
	return nil
}
