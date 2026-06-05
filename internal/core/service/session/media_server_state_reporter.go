package session

import (
	"context"
	"fmt"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
)

type MediaServerStateReporterOptions struct {
	ID          string
	URL         string
	Status      entity.MediaServerStatus
	MaxSessions int
}

type MediaServerStateReporter struct {
	states      repository.MediaServerStateRepository
	stats       usecase.GetRuntimeStatsQuery
	id          string
	url         string
	status      entity.MediaServerStatus
	maxSessions int
	now         func() time.Time
}

func NewMediaServerStateReporter(
	states repository.MediaServerStateRepository,
	stats usecase.GetRuntimeStatsQuery,
	options MediaServerStateReporterOptions,
) *MediaServerStateReporter {
	if options.URL == "" {
		options.URL = "http://localhost:8080"
	}
	if options.Status == "" {
		options.Status = entity.MediaServerStatusHealthy
	}
	return &MediaServerStateReporter{
		states:      states,
		stats:       stats,
		id:          options.ID,
		url:         options.URL,
		status:      options.Status,
		maxSessions: options.MaxSessions,
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
	stats, err := r.stats.GetRuntimeStats(ctx)
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
