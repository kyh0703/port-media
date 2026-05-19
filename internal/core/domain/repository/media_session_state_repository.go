package repository

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type MediaSessionStateRepository interface {
	Save(ctx context.Context, state entity.MediaSessionState) error
	FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionState, bool, error)
	Delete(ctx context.Context, sessionID vo.SessionID) error
}
