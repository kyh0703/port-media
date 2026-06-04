package repository

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
)

//counterfeiter:generate . MediaServerStateRepository
type MediaServerStateRepository interface {
	Save(ctx context.Context, state entity.MediaServerState) error
	SaveOffline(ctx context.Context, state entity.MediaServerState) error
}
