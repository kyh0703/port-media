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

type IdleRoomCleanupScheduler struct {
	svc      sessionservice.Service
	log      *zap.Logger
	timeout  time.Duration
	interval time.Duration
	enabled  bool
}

func NewIdleRoomCleanupScheduler(
	cfg *configs.Config,
	svc sessionservice.Service,
	log *zap.Logger,
) *IdleRoomCleanupScheduler {
	if log == nil {
		log = zap.NewNop()
	}
	timeout := idleRoomCleanupTimeout(cfg)
	return &IdleRoomCleanupScheduler{
		svc:      svc,
		log:      log,
		timeout:  timeout,
		interval: idleRoomCleanupInterval(timeout),
		enabled:  timeout > 0,
	}
}

func RegisterIdleRoomCleanup(lc fx.Lifecycle, scheduler *IdleRoomCleanupScheduler) {
	if scheduler == nil || !scheduler.enabled {
		return
	}

	done := make(chan struct{})
	stopped := make(chan struct{})
	var stopOnce sync.Once
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ticker := time.NewTicker(scheduler.interval)
			go func() {
				defer ticker.Stop()
				defer close(stopped)
				for {
					select {
					case <-ticker.C:
						cleaned, err := scheduler.svc.CleanupIdleRooms(context.Background(), scheduler.timeout)
						if err != nil {
							scheduler.log.Error("idle_room_cleanup_failed", zap.Error(err))
							continue
						}
						if cleaned > 0 {
							scheduler.log.Info("idle_rooms_cleaned", zap.Int("count", cleaned))
						}
					case <-done:
						return
					}
				}
			}()
			_ = ctx
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopOnce.Do(func() { close(done) })
			<-stopped
			cleaned, err := scheduler.svc.ShutdownActiveRooms(ctx)
			if err != nil {
				scheduler.log.Error("shutdown_room_cleanup_failed", zap.Error(err))
				return err
			}
			if cleaned > 0 {
				scheduler.log.Info("shutdown_rooms_cleaned", zap.Int("count", cleaned))
			}
			return nil
		},
	})
}

func idleRoomCleanupTimeout(cfg *configs.Config) time.Duration {
	if cfg == nil {
		return 0
	}
	return cfg.Realtime.RoomIdleTimeout
}

func idleRoomCleanupInterval(timeout time.Duration) time.Duration {
	interval := timeout / 2
	if interval < time.Second {
		return time.Second
	}
	if interval > 30*time.Second {
		return 30 * time.Second
	}
	return interval
}
