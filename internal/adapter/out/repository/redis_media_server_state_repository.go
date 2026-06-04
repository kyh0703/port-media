package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	domainrepo "github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/redis/go-redis/v9"
)

const (
	mediaServersSetKey        = "media:servers"
	mediaServerStateKeyPrefix = "media:server:"
)

type RedisMediaServerStateRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisMediaServerStateRepository(client *redis.Client, cfg *configs.Config) domainrepo.MediaServerStateRepository {
	return &RedisMediaServerStateRepository{
		client: client,
		ttl:    mediaServerStateTTL(cfg),
	}
}

func (r *RedisMediaServerStateRepository) Save(ctx context.Context, state entity.MediaServerState) error {
	return r.writeState(ctx, state, true)
}

func (r *RedisMediaServerStateRepository) SaveOffline(ctx context.Context, state entity.MediaServerState) error {
	state.Status = entity.MediaServerStatusOffline
	return r.writeState(ctx, state, false)
}

func (r *RedisMediaServerStateRepository) writeState(ctx context.Context, state entity.MediaServerState, listed bool) error {
	body, err := json.Marshal(mediaServerStatePayload{
		ID:                 state.ID,
		URL:                state.URL,
		Status:             string(state.Status),
		ActiveRooms:        state.ActiveRooms,
		ActiveSessions:     state.ActiveSessions,
		ActiveParticipants: state.ActiveParticipants,
		ActiveTracks:       state.ActiveTracks,
		MaxSessions:        state.MaxSessions,
		UpdatedAt:          state.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal media server state: %w", err)
	}

	pipe := r.client.TxPipeline()
	if listed {
		pipe.SAdd(ctx, mediaServersSetKey, state.ID)
	} else {
		pipe.SRem(ctx, mediaServersSetKey, state.ID)
	}
	pipe.Set(ctx, mediaServerStateKey(state.ID), body, r.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("write media server state %s: %w", state.ID, err)
	}
	return nil
}

func mediaServerStateKey(id string) string {
	return mediaServerStateKeyPrefix + id
}

func mediaServerStateTTL(cfg *configs.Config) time.Duration {
	if cfg == nil || cfg.MediaServer.HeartbeatTTL <= 0 {
		return 3 * mediaServerStateInterval(cfg)
	}
	return cfg.MediaServer.HeartbeatTTL
}

func mediaServerStateInterval(cfg *configs.Config) time.Duration {
	if cfg == nil || cfg.MediaServer.HeartbeatInterval <= 0 {
		return 10 * time.Second
	}
	return cfg.MediaServer.HeartbeatInterval
}

type mediaServerStatePayload struct {
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
