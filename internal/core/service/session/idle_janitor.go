package session

import (
	"context"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func RegisterIdleCleanup(lc fx.Lifecycle, cfg *configs.Config, svc Service, log *zap.Logger) {
	timeout := cfg.Realtime.RoomIdleTimeout
	if timeout <= 0 {
		return
	}

	done := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ticker := time.NewTicker(idleCleanupInterval(timeout))

			go func() {
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						cleaned, err := svc.CleanupIdleRooms(context.Background(), timeout)
						if err != nil {
							log.Error("idle_room_cleanup_failed", zap.Error(err))
							continue
						}
						if cleaned > 0 {
							log.Info("idle_rooms_cleaned", zap.Int("count", cleaned))
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
			close(done)
			cleaned, err := svc.ShutdownActiveRooms(ctx)
			if err != nil {
				log.Error("shutdown_room_cleanup_failed", zap.Error(err))
				return err
			}
			if cleaned > 0 {
				log.Info("shutdown_rooms_cleaned", zap.Int("count", cleaned))
			}
			return nil
		},
	})
}

func idleCleanupInterval(timeout time.Duration) time.Duration {
	interval := timeout / 2
	if interval < time.Second {
		return time.Second
	}
	if interval > 30*time.Second {
		return 30 * time.Second
	}
	return interval
}
