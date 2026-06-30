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
	SessionID       string `json:"session_id"`
	ConversationID  string `json:"conversation_id"`
	RoomID          string `json:"room_id"`
	ParticipantID   string `json:"participant_id"`
	ParticipantRole string `json:"participant_role"`
	UserID          string `json:"user_id"`
	Permissions     struct {
		PublishAudio   bool `json:"publish_audio"`
		SubscribeAudio bool `json:"subscribe_audio"`
	} `json:"permissions"`
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
