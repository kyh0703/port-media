package session

import (
	sessionquery "github.com/kyh0703/portfoilo-media/internal/core/query/session"
	"time"
)

const runtimeEventTypeDataChannelMessage = "data_channel.message"

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func appendRuntimeEvent(
	events []sessionquery.RuntimeEvent,
	event sessionquery.RuntimeEvent,
	limit int,
) []sessionquery.RuntimeEvent {
	if limit <= 0 {
		return nil
	}
	next := append(append([]sessionquery.RuntimeEvent(nil), events...), event)
	if len(next) <= limit {
		return next
	}
	return next[len(next)-limit:]
}
