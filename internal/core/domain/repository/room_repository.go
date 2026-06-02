package repository

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

//counterfeiter:generate . RoomRepository

type RoomRepository interface {
	Save(ctx context.Context, room entity.Room) error
	FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.Room, bool, error)
	Delete(ctx context.Context, roomID vo.RoomID) error
}
