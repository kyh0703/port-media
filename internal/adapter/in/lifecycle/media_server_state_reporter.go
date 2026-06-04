package lifecycle

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MediaServerStateReporter struct {
	states      repository.MediaServerStateRepository
	svc         sessionservice.Service
	log         *zap.Logger
	id          string
	url         string
	status      entity.MediaServerStatus
	maxSessions int
	interval    time.Duration
	now         func() time.Time
}

func NewMediaServerStateReporter(
	states repository.MediaServerStateRepository,
	svc sessionservice.Service,
	cfg *configs.Config,
	log *zap.Logger,
) *MediaServerStateReporter {
	if log == nil {
		log = zap.NewNop()
	}
	return &MediaServerStateReporter{
		states:      states,
		svc:         svc,
		log:         log,
		id:          mediaServerID(cfg),
		url:         mediaServerURL(cfg),
		status:      mediaServerStatus(cfg),
		maxSessions: mediaServerMaxSessions(cfg),
		interval:    mediaServerStateInterval(cfg),
		now:         time.Now,
	}
}

func RegisterMediaServerStateReporter(lc fx.Lifecycle, cfg *configs.Config, reporter *MediaServerStateReporter) {
	if cfg == nil || !cfg.MediaServer.HeartbeatEnabled {
		return
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	var stopOnce sync.Once
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := reporter.Report(ctx); err != nil {
				return err
			}

			ticker := time.NewTicker(reporter.interval)
			go func() {
				defer ticker.Stop()
				defer close(stopped)
				for {
					select {
					case <-ticker.C:
						if err := reporter.Report(context.Background()); err != nil {
							reporter.log.Error("media_server_state_report_failed", zap.Error(err))
						}
					case <-done:
						return
					}
				}
			}()
			reporter.log.Info("media_server_state_reporter_started",
				zap.String("id", reporter.id),
				zap.String("url", reporter.url),
				zap.Duration("interval", reporter.interval),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopOnce.Do(func() { close(done) })
			<-stopped
			if err := reporter.ReportOffline(ctx); err != nil {
				reporter.log.Error("media_server_offline_state_report_failed", zap.Error(err))
				return err
			}
			return nil
		},
	})
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

func mediaServerStateInterval(cfg *configs.Config) time.Duration {
	if cfg == nil || cfg.MediaServer.HeartbeatInterval <= 0 {
		return 10 * time.Second
	}
	return cfg.MediaServer.HeartbeatInterval
}

func mediaServerMaxSessions(cfg *configs.Config) int {
	if cfg == nil {
		return 0
	}
	return cfg.MediaServer.MaxSessions
}
