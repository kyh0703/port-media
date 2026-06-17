package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	"github.com/redis/go-redis/v9"
)

const defaultConversationStreamName = "media:conversation-events:v1"

type RedisConversationEventPublisher struct {
	client *redis.Client
	stream string
	maxLen int64
}

type NoopConversationEventPublisher struct{}

func NewConversationEventPublisher(client *redis.Client, cfg *configs.Config) port.ConversationEventPublisher {
	if cfg != nil && !cfg.Events.ConversationStreamEnabled {
		return NoopConversationEventPublisher{}
	}

	stream := defaultConversationStreamName
	maxLen := int64(0)
	if cfg != nil {
		if configuredStream := strings.TrimSpace(cfg.Events.ConversationStreamName); configuredStream != "" {
			stream = configuredStream
		}
		maxLen = cfg.Events.ConversationStreamMaxLen
	}

	return &RedisConversationEventPublisher{
		client: client,
		stream: stream,
		maxLen: maxLen,
	}
}

func (p *RedisConversationEventPublisher) Publish(ctx context.Context, event port.ConversationEvent) error {
	args := &redis.XAddArgs{
		Stream: p.stream,
		Values: []any{
			"schema_version", fmt.Sprintf("%d", event.SchemaVersion),
			"event_id", event.EventID,
			"conversation_id", string(event.ConversationID),
			"session_id", string(event.SessionID),
			"room_id", string(event.RoomID),
			"provider_call_id", event.ProviderCallID,
			"event_type", string(event.Type),
			"provider_event_type", string(event.ProviderEventType),
			"provider_item_id", event.ProviderItemID,
			"previous_provider_item_id", event.PreviousProviderItemID,
			"provider_response_id", event.ProviderRespID,
			"occurred_at", event.OccurredAt.UTC().Format(time.RFC3339Nano),
			"payload", event.Payload,
		},
	}
	if p.maxLen > 0 {
		args.MaxLen = p.maxLen
		args.Approx = true
	}

	if err := p.client.XAdd(ctx, args).Err(); err != nil {
		return fmt.Errorf("publish conversation event %s: %w", event.EventID, err)
	}
	return nil
}

func (NoopConversationEventPublisher) Publish(ctx context.Context, event port.ConversationEvent) error {
	_ = ctx
	_ = event
	return nil
}
