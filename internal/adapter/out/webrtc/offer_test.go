package webrtc

import (
	"context"
	"strings"
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

func TestEngineCreatesAgentOffer(t *testing.T) {
	engine, err := NewEngine(&configs.Config{})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	offer, err := engine.CreateOffer(context.Background(), CreateOfferInput{
		SessionID:     vo.SessionID("session-1"),
		ParticipantID: vo.ParticipantID("agent-1"),
		Role:          vo.ParticipantRoleAgent,
	})
	if err != nil {
		t.Fatalf("CreateOffer() error = %v", err)
	}
	defer func() {
		_ = offer.Close()
	}()

	if offer.SDPOffer == "" {
		t.Fatal("CreateOffer() returned empty SDP offer")
	}
	if offer.Role != vo.ParticipantRoleAgent {
		t.Fatalf("Role = %q, want %q", offer.Role, vo.ParticipantRoleAgent)
	}
}

func TestEngineCreatesAgentDataChannelOffer(t *testing.T) {
	engine, err := NewEngine(&configs.Config{})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	offer, err := engine.CreateOffer(context.Background(), CreateOfferInput{
		SessionID:        vo.SessionID("session-1"),
		ParticipantID:    vo.ParticipantID("agent-1"),
		Role:             vo.ParticipantRoleAgent,
		DataChannelLabel: "oai-events",
	})
	if err != nil {
		t.Fatalf("CreateOffer() error = %v", err)
	}
	defer func() {
		_ = offer.Close()
	}()

	if offer.DataChannelLabel != "oai-events" {
		t.Fatalf("DataChannelLabel = %q, want oai-events", offer.DataChannelLabel)
	}
	if !strings.Contains(offer.SDPOffer, "m=application") {
		t.Fatalf("SDP offer does not include data channel application section: %s", offer.SDPOffer)
	}
}
