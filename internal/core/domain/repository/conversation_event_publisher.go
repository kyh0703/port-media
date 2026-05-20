package repository

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
)

//counterfeiter:generate . ConversationEventPublisher

type ConversationEventPublisher interface {
	Publish(ctx context.Context, event entity.ConversationEvent) error
}
