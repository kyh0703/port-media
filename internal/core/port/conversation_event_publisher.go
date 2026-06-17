package port

import "context"

type ConversationEventPublisher interface {
	Publish(ctx context.Context, event ConversationEvent) error
}
