package session

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

//go:generate go tool counterfeiter -generate
//counterfeiter:generate . MediaSessionStateRepository

type MediaSessionStateRepository interface {
	Save(ctx context.Context, state MediaSessionState) error
	FindBySessionID(ctx context.Context, sessionID vo.SessionID) (MediaSessionState, bool, error)
	Delete(ctx context.Context, sessionID vo.SessionID) error
}
