package webrtc

import (
	"context"
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
)

func TestGatewayRuntimeEventsUseConfiguredContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	gateway := NewGatewayWithEventContext(nil, ctx)
	handler := &captureRuntimeEventHandler{}
	gateway.SubscribeRuntimeEvents(handler)

	gateway.handleConnectionStateChange()(ConnectionStateChange{
		SessionID:     vo.SessionID("session-1"),
		ParticipantID: vo.ParticipantID("participant-1"),
		Role:          vo.ParticipantRoleUser,
		State:         vo.ConnectionStateClosed,
	})

	if handler.connectionContextErr != context.Canceled {
		t.Fatalf("runtime event context err = %v, want context.Canceled", handler.connectionContextErr)
	}
}

type captureRuntimeEventHandler struct {
	connectionContextErr error
}

func (h *captureRuntimeEventHandler) HandleConnectionStateChange(ctx context.Context, change port.ConnectionStateChange) {
	_ = change
	h.connectionContextErr = ctx.Err()
}

func (h *captureRuntimeEventHandler) HandleMediaTrackStateChange(ctx context.Context, change port.MediaTrackStateChange) {
	_ = ctx
	_ = change
}

func (h *captureRuntimeEventHandler) HandleDataChannelMessage(ctx context.Context, message port.DataChannelMessage) {
	_ = ctx
	_ = message
}
