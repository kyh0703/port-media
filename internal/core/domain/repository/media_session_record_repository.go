package repository

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

//go:generate go tool counterfeiter -generate
//counterfeiter:generate . MediaSessionRecordRepository

type MediaSessionRecordRepository interface {
	Save(ctx context.Context, record entity.MediaSessionRecord) error
	FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionRecord, bool, error)
	Delete(ctx context.Context, roomID vo.RoomID) error
}
