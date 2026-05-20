package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	mediaServersSetKey        = "media:servers"
	mediaServerStateKeyPrefix = "media:server:"
	defaultMediaServerStatus  = "healthy"
)

type MediaServerHeartbeat struct {
	client   *redis.Client
	svc      Service
	cfg      *configs.Config
	log      *zap.Logger
	id       string
	url      string
	status   string
	interval time.Duration
	ttl      time.Duration
	now      func() time.Time
}

func NewMediaServerHeartbeat(client *redis.Client, svc Service, cfg *configs.Config, log *zap.Logger) *MediaServerHeartbeat {
	if log == nil {
		log = zap.NewNop()
	}
	return &MediaServerHeartbeat{
		client:   client,
		svc:      svc,
		cfg:      cfg,
		log:      log,
		id:       mediaServerID(cfg),
		url:      mediaServerURL(cfg),
		status:   mediaServerStatus(cfg),
		interval: mediaServerHeartbeatInterval(cfg),
		ttl:      mediaServerHeartbeatTTL(cfg),
		now:      time.Now,
	}
}

func RegisterMediaServerHeartbeat(lc fx.Lifecycle, cfg *configs.Config, heartbeat *MediaServerHeartbeat) {
	if cfg == nil || !cfg.MediaServer.HeartbeatEnabled {
		return
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	var stopOnce sync.Once
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := heartbeat.write(ctx, heartbeat.status); err != nil {
				return err
			}

			ticker := time.NewTicker(heartbeat.interval)
			go func() {
				defer ticker.Stop()
				defer close(stopped)
				for {
					select {
					case <-ticker.C:
						if err := heartbeat.write(context.Background(), heartbeat.status); err != nil {
							heartbeat.log.Error("media_server_heartbeat_failed", zap.Error(err))
						}
					case <-done:
						return
					}
				}
			}()
			heartbeat.log.Info("media_server_heartbeat_started",
				zap.String("id", heartbeat.id),
				zap.String("url", heartbeat.url),
				zap.Duration("interval", heartbeat.interval),
				zap.Duration("ttl", heartbeat.ttl),
			)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopOnce.Do(func() { close(done) })
			<-stopped
			if err := heartbeat.writeOffline(ctx); err != nil {
				heartbeat.log.Error("media_server_offline_heartbeat_failed", zap.Error(err))
				return err
			}
			return nil
		},
	})
}

func (h *MediaServerHeartbeat) write(ctx context.Context, status string) error {
	return h.writeState(ctx, status, true)
}

func (h *MediaServerHeartbeat) writeOffline(ctx context.Context) error {
	return h.writeState(ctx, "offline", false)
}

func (h *MediaServerHeartbeat) writeState(ctx context.Context, status string, listed bool) error {
	stats, err := h.svc.GetRuntimeStats(ctx)
	if err != nil {
		return fmt.Errorf("get runtime stats for media server heartbeat: %w", err)
	}

	payload := mediaServerHeartbeatPayload{
		ID:                 h.id,
		URL:                h.url,
		Status:             status,
		ActiveRooms:        stats.Rooms,
		ActiveSessions:     stats.Sessions,
		ActiveParticipants: stats.Participants,
		ActiveTracks:       stats.Tracks,
		MaxSessions:        mediaServerMaxSessions(h.cfg),
		UpdatedAt:          h.now().UTC(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal media server heartbeat: %w", err)
	}

	pipe := h.client.TxPipeline()
	if listed {
		pipe.SAdd(ctx, mediaServersSetKey, h.id)
	} else {
		pipe.SRem(ctx, mediaServersSetKey, h.id)
	}
	pipe.Set(ctx, mediaServerStateKey(h.id), body, h.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("write media server heartbeat %s: %w", h.id, err)
	}
	return nil
}

func mediaServerStateKey(id string) string {
	return mediaServerStateKeyPrefix + id
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

func mediaServerStatus(cfg *configs.Config) string {
	if cfg == nil {
		return defaultMediaServerStatus
	}
	status := strings.TrimSpace(cfg.MediaServer.Status)
	if status == "" {
		return defaultMediaServerStatus
	}
	return status
}

func mediaServerHeartbeatInterval(cfg *configs.Config) time.Duration {
	if cfg == nil || cfg.MediaServer.HeartbeatInterval <= 0 {
		return 10 * time.Second
	}
	return cfg.MediaServer.HeartbeatInterval
}

func mediaServerHeartbeatTTL(cfg *configs.Config) time.Duration {
	if cfg == nil || cfg.MediaServer.HeartbeatTTL <= 0 {
		return 3 * mediaServerHeartbeatInterval(cfg)
	}
	return cfg.MediaServer.HeartbeatTTL
}

func mediaServerMaxSessions(cfg *configs.Config) int {
	if cfg == nil {
		return 0
	}
	return cfg.MediaServer.MaxSessions
}

type mediaServerHeartbeatPayload struct {
	ID                 string    `json:"id"`
	URL                string    `json:"url"`
	Status             string    `json:"status"`
	ActiveRooms        int       `json:"active_rooms"`
	ActiveSessions     int       `json:"active_sessions"`
	ActiveParticipants int       `json:"active_participants"`
	ActiveTracks       int       `json:"active_tracks"`
	MaxSessions        int       `json:"max_sessions"`
	UpdatedAt          time.Time `json:"updated_at"`
}
