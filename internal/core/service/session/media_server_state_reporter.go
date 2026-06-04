package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
)

type MediaServerStateReporter struct {
	states      repository.MediaServerStateRepository
	svc         Service
	id          string
	url         string
	status      entity.MediaServerStatus
	maxSessions int
	now         func() time.Time
}

func NewMediaServerStateReporter(
	states repository.MediaServerStateRepository,
	svc Service,
	cfg *configs.Config,
) *MediaServerStateReporter {
	return &MediaServerStateReporter{
		states:      states,
		svc:         svc,
		id:          mediaServerID(cfg),
		url:         mediaServerURL(cfg),
		status:      mediaServerStatus(cfg),
		maxSessions: mediaServerMaxSessions(cfg),
		now:         time.Now,
	}
}

func (r *MediaServerStateReporter) Report(ctx context.Context) error {
	state, err := r.state(ctx, r.status)
	if err != nil {
		return err
	}
	if err := r.states.Save(ctx, state); err != nil {
		return fmt.Errorf("save media server state %s: %w", state.ID, err)
	}
	return nil
}

func (r *MediaServerStateReporter) ReportOffline(ctx context.Context) error {
	state, err := r.state(ctx, entity.MediaServerStatusOffline)
	if err != nil {
		return err
	}
	if err := r.states.SaveOffline(ctx, state); err != nil {
		return fmt.Errorf("save offline media server state %s: %w", state.ID, err)
	}
	return nil
}

func (r *MediaServerStateReporter) state(ctx context.Context, status entity.MediaServerStatus) (entity.MediaServerState, error) {
	stats, err := r.svc.GetRuntimeStats(ctx)
	if err != nil {
		return entity.MediaServerState{}, fmt.Errorf("get runtime stats for media server state: %w", err)
	}

	return entity.MediaServerState{
		ID:                 r.id,
		URL:                r.url,
		Status:             status,
		ActiveRooms:        stats.Rooms,
		ActiveSessions:     stats.Sessions,
		ActiveParticipants: stats.Participants,
		ActiveTracks:       stats.Tracks,
		MaxSessions:        r.maxSessions,
		UpdatedAt:          r.now().UTC(),
	}, nil
}

func mediaServerID(cfg *configs.Config) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.NodeID)
}

func mediaServerURL(cfg *configs.Config) string {
	if cfg == nil {
		return "http://localhost:8080"
	}
	if url := strings.TrimSpace(cfg.MediaServer.URL); url != "" {
		return url
	}
	host := strings.TrimSpace(cfg.Server.Host)
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}
	port := cfg.Server.Port
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}

func mediaServerStatus(cfg *configs.Config) entity.MediaServerStatus {
	if cfg == nil {
		return entity.MediaServerStatusHealthy
	}
	status := strings.TrimSpace(cfg.MediaServer.Status)
	if status == "" {
		return entity.MediaServerStatusHealthy
	}
	return entity.MediaServerStatus(status)
}

func mediaServerMaxSessions(cfg *configs.Config) int {
	if cfg == nil {
		return 0
	}
	return cfg.MediaServer.MaxSessions
}
