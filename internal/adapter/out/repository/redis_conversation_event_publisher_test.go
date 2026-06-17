package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	"github.com/redis/go-redis/v9"
)

func TestRedisConversationEventPublisherWritesStreamEntry(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	publisher := NewConversationEventPublisher(client, &configs.Config{
		Events: configs.EventsConfig{
			ConversationStreamEnabled: true,
			ConversationStreamName:    "test:conversation-events",
		},
	})
	occurredAt := time.Date(2026, 5, 19, 10, 1, 2, 3, time.UTC)

	err = publisher.Publish(context.Background(), port.ConversationEvent{
		SchemaVersion:          1,
		EventID:                "evt_123",
		ConversationID:         vo.ConversationID("conversation-1"),
		SessionID:              vo.SessionID("session-1"),
		RoomID:                 vo.RoomID("room-1"),
		ProviderCallID:         "rtc_123",
		Type:                   "assistant.output_text.completed",
		ProviderEventType:      "response.output_text.done",
		ProviderItemID:         "item-2",
		PreviousProviderItemID: "item-1",
		ProviderRespID:         "resp-1",
		OccurredAt:             occurredAt,
		Payload:                `{"type":"response.output_text.done","text":"hello"}`,
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	entries, err := client.XRange(context.Background(), "test:conversation-events", "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}

	fields := entries[0].Values
	assertStreamField(t, fields, "schema_version", "1")
	assertStreamField(t, fields, "event_id", "evt_123")
	assertStreamField(t, fields, "conversation_id", "conversation-1")
	assertStreamField(t, fields, "session_id", "session-1")
	assertStreamField(t, fields, "room_id", "room-1")
	assertStreamField(t, fields, "provider_call_id", "rtc_123")
	assertStreamField(t, fields, "event_type", "assistant.output_text.completed")
	assertStreamField(t, fields, "provider_event_type", "response.output_text.done")
	assertStreamField(t, fields, "provider_item_id", "item-2")
	assertStreamField(t, fields, "previous_provider_item_id", "item-1")
	assertStreamField(t, fields, "provider_response_id", "resp-1")
	assertStreamField(t, fields, "occurred_at", occurredAt.Format(time.RFC3339Nano))
	assertStreamField(t, fields, "payload", `{"type":"response.output_text.done","text":"hello"}`)
}

func TestConversationEventPublisherDisabledIsNoop(t *testing.T) {
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer server.Close()

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	publisher := NewConversationEventPublisher(client, &configs.Config{
		Events: configs.EventsConfig{
			ConversationStreamEnabled: false,
			ConversationStreamName:    "test:conversation-events",
		},
	})

	err = publisher.Publish(context.Background(), port.ConversationEvent{
		SchemaVersion:     1,
		EventID:           "evt_123",
		ConversationID:    vo.ConversationID("conversation-1"),
		SessionID:         vo.SessionID("session-1"),
		RoomID:            vo.RoomID("room-1"),
		ProviderCallID:    "rtc_123",
		ProviderEventType: "response.output_text.done",
		OccurredAt:        time.Now(),
		Payload:           `{"type":"response.output_text.done"}`,
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	exists, err := client.Exists(context.Background(), "test:conversation-events").Result()
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists != 0 {
		t.Fatalf("stream exists = %d, want 0", exists)
	}
}

func assertStreamField(t *testing.T, fields map[string]any, key string, want string) {
	t.Helper()
	if fields[key] != want {
		t.Fatalf("%s = %v, want %q", key, fields[key], want)
	}
}
