package port

import (
	"context"
	"errors"
)

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

//go:generate go tool counterfeiter -generate
//counterfeiter:generate . MediaTokenStore
type MediaTokenStore interface {
	Get(ctx context.Context, token string) (MediaToken, error)
}

//counterfeiter:generate . MediaTokenVerifier
type MediaTokenVerifier interface {
	Verify(ctx context.Context, raw string) (MediaToken, error)
}
