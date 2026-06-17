package session

import (
	"crypto/sha256"
	"fmt"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessionquery "github.com/kyh0703/portfoilo-media/internal/core/query/session"
	"strings"
	"time"
)

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func appendRealtimeEvent(
	events []sessionquery.RealtimeEvent,
	event sessionquery.RealtimeEvent,
	limit int,
) []sessionquery.RealtimeEvent {
	if limit <= 0 {
		return nil
	}
	next := append(append([]sessionquery.RealtimeEvent(nil), events...), event)
	if len(next) <= limit {
		return next
	}
	return next[len(next)-limit:]
}

func fallbackConversationEventID(
	sessionID vo.SessionID,
	providerCallID string,
	eventType string,
	payload string,
) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(sessionID),
		providerCallID,
		eventType,
		payload,
	}, "\x00")))
	return fmt.Sprintf("media-%x", sum[:])
}
