package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

//go:generate go tool counterfeiter -generate

const MediaTokenKeyPrefix = "media:token:"

var (
	ErrMissingMediaToken  = errors.New("missing media token")
	ErrInvalidMediaToken  = errors.New("invalid media token")
	ErrMediaTokenNotFound = errors.New("media token not found")
)

type MediaToken struct {
	SessionID      string `json:"session_id"`
	ConversationID string `json:"conversation_id"`
	UserID         string `json:"user_id"`
}

//counterfeiter:generate . MediaTokenStore
type MediaTokenStore interface {
	Get(ctx context.Context, token string) (MediaToken, error)
}

//counterfeiter:generate . MediaTokenVerifier
type MediaTokenVerifier interface {
	Verify(ctx context.Context, raw string) (MediaToken, error)
}

type Verifier struct {
	store MediaTokenStore
}

func NewMediaTokenVerifier(store MediaTokenStore) MediaTokenVerifier {
	return &Verifier{store: store}
}

func (v *Verifier) Verify(ctx context.Context, raw string) (MediaToken, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return MediaToken{}, ErrMissingMediaToken
	}

	claims, err := v.store.Get(ctx, token)
	if err != nil {
		return MediaToken{}, fmt.Errorf("%w: %v", ErrInvalidMediaToken, err)
	}
	if claims.SessionID == "" || claims.ConversationID == "" || claims.UserID == "" {
		return MediaToken{}, fmt.Errorf("%w: missing required token fields", ErrInvalidMediaToken)
	}

	return claims, nil
}

type RedisMediaTokenStore struct {
	client *redis.Client
}

func NewRedisMediaTokenStore(client *redis.Client) *RedisMediaTokenStore {
	return &RedisMediaTokenStore{client: client}
}

func (s *RedisMediaTokenStore) Get(ctx context.Context, token string) (MediaToken, error) {
	payload, err := s.client.Get(ctx, MediaTokenKeyPrefix+token).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return MediaToken{}, ErrMediaTokenNotFound
		}
		return MediaToken{}, err
	}

	var mediaToken MediaToken
	if err := json.Unmarshal([]byte(payload), &mediaToken); err != nil {
		return MediaToken{}, fmt.Errorf("decode media token: %w", err)
	}

	return mediaToken, nil
}
