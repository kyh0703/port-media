package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kyh0703/portfoilo-media/internal/core/port"
	"github.com/redis/go-redis/v9"
)

const MediaTokenKeyPrefix = "media:token:"

type Verifier struct {
	store port.MediaTokenStore
}

func NewMediaTokenVerifier(store port.MediaTokenStore) port.MediaTokenVerifier {
	return &Verifier{store: store}
}

func (v *Verifier) Verify(ctx context.Context, raw string) (port.MediaToken, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return port.MediaToken{}, port.ErrMissingMediaToken
	}

	claims, err := v.store.Get(ctx, token)
	if err != nil {
		return port.MediaToken{}, fmt.Errorf("%w: %v", port.ErrInvalidMediaToken, err)
	}
	if claims.SessionID == "" || claims.ConversationID == "" || claims.UserID == "" {
		return port.MediaToken{}, fmt.Errorf("%w: missing required token fields", port.ErrInvalidMediaToken)
	}

	return claims, nil
}

type RedisMediaTokenStore struct {
	client *redis.Client
}

func NewRedisMediaTokenStore(client *redis.Client) *RedisMediaTokenStore {
	return &RedisMediaTokenStore{client: client}
}

func (s *RedisMediaTokenStore) Get(ctx context.Context, token string) (port.MediaToken, error) {
	payload, err := s.client.Get(ctx, MediaTokenKeyPrefix+token).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return port.MediaToken{}, port.ErrMediaTokenNotFound
		}
		return port.MediaToken{}, err
	}

	var mediaToken port.MediaToken
	if err := json.Unmarshal([]byte(payload), &mediaToken); err != nil {
		return port.MediaToken{}, fmt.Errorf("decode media token: %w", err)
	}

	return mediaToken, nil
}
