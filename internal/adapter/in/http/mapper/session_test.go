package mapper

import (
	"testing"
	"time"

	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
)

func TestToGetSessionStatusResponseMapsRuntimeEvents(t *testing.T) {
	eventAt := time.Date(2026, 7, 1, 10, 1, 0, 0, time.UTC)

	response := ToGetSessionStatusResponse(sessionio.GetSessionStatusResult{
		SessionID:            "session-1",
		LastRuntimeEventType: "data_channel.message",
		LastRuntimeEventAt:   eventAt,
		RecentRuntimeEvents: []sessionio.RuntimeEventResult{
			{Type: "data_channel.message", At: eventAt},
		},
	})

	if response.LastRuntimeEventType != "data_channel.message" {
		t.Fatalf("LastRuntimeEventType = %q, want data_channel.message", response.LastRuntimeEventType)
	}
	if response.LastRuntimeEventAt != eventAt.Format(time.RFC3339Nano) {
		t.Fatalf("LastRuntimeEventAt = %q, want %s", response.LastRuntimeEventAt, eventAt.Format(time.RFC3339Nano))
	}
	if len(response.RecentRuntimeEvents) != 1 {
		t.Fatalf("RecentRuntimeEvents len = %d, want 1", len(response.RecentRuntimeEvents))
	}
	if response.RecentRuntimeEvents[0].Type != "data_channel.message" {
		t.Fatalf("RecentRuntimeEvents[0].Type = %q, want data_channel.message", response.RecentRuntimeEvents[0].Type)
	}
	if response.RecentRuntimeEvents[0].At != eventAt.Format(time.RFC3339Nano) {
		t.Fatalf("RecentRuntimeEvents[0].At = %q, want %s", response.RecentRuntimeEvents[0].At, eventAt.Format(time.RFC3339Nano))
	}
}
