package lifecycle

import (
	"context"
	"sync"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MediaServerStateScheduler struct {
	reporter *sessionservice.MediaServerStateReporter
	log      *zap.Logger
	interval time.Duration
}

func NewMediaServerStateScheduler(
	reporter *sessionservice.MediaServerStateReporter,
	cfg *configs.Config,
	log *zap.Logger,
) *MediaServerStateScheduler {
	if log == nil {
		log = zap.NewNop()
	}
	return &MediaServerStateScheduler{
		reporter: reporter,
		log:      log,
		interval: mediaServerStateInterval(cfg),
	}
}

func RegisterMediaServerStateScheduler(lc fx.Lifecycle, cfg *configs.Config, scheduler *MediaServerStateScheduler) {
	if cfg == nil || !cfg.MediaServer.HeartbeatEnabled {
		return
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	var stopOnce sync.Once
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := scheduler.reporter.Report(ctx); err != nil {
				return err
			}

			ticker := time.NewTicker(scheduler.interval)
			go func() {
				defer ticker.Stop()
				defer close(stopped)
				for {
					select {
					case <-ticker.C:
						if err := scheduler.reporter.Report(context.Background()); err != nil {
							scheduler.log.Error("media_server_state_report_failed", zap.Error(err))
						}
					case <-done:
						return
					}
				}
			}()
			scheduler.log.Info("media_server_state_scheduler_started",
				zap.Duration("interval", scheduler.interval),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopOnce.Do(func() { close(done) })
			<-stopped
			if err := scheduler.reporter.ReportOffline(ctx); err != nil {
				scheduler.log.Error("media_server_offline_state_report_failed", zap.Error(err))
				return err
			}
			return nil
		},
	})
}

func mediaServerStateInterval(cfg *configs.Config) time.Duration {
	if cfg == nil || cfg.MediaServer.HeartbeatInterval <= 0 {
		return 10 * time.Second
	}
	return cfg.MediaServer.HeartbeatInterval
}
