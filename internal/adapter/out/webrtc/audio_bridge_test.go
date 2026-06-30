package webrtc

import (
	"context"
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestEnginePreparesAudioBridgeTracks(t *testing.T) {
	engine, err := NewEngine(&configs.Config{})
	if err != nil {
		t.Fatalf("NewEngine() error = %v", err)
	}

	offerer, err := pionwebrtc.NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		t.Fatalf("NewPeerConnection() error = %v", err)
	}
	defer func() {
		_ = offerer.Close()
	}()

	if _, err := offerer.AddTransceiverFromKind(pionwebrtc.RTPCodecTypeAudio); err != nil {
		t.Fatalf("AddTransceiverFromKind(audio) error = %v", err)
	}
	offer, err := offerer.CreateOffer(nil)
	if err != nil {
		t.Fatalf("CreateOffer() error = %v", err)
	}
	if err := offerer.SetLocalDescription(offer); err != nil {
		t.Fatalf("SetLocalDescription() error = %v", err)
	}

	clientPeer, err := engine.AcceptOffer(context.Background(), OfferInput{
		SessionID:     vo.SessionID("session-1"),
		ParticipantID: vo.ParticipantID("client-1"),
		Role:          vo.ParticipantRoleUser,
		SDP:           offer.SDP,
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}
	defer func() {
		_ = clientPeer.Close()
	}()

	agentOffer, err := engine.CreateOffer(context.Background(), CreateOfferInput{
		SessionID:     vo.SessionID("session-1"),
		ParticipantID: vo.ParticipantID("agent-1"),
		Role:          vo.ParticipantRoleAgent,
	})
	if err != nil {
		t.Fatalf("CreateOffer() error = %v", err)
	}
	defer func() {
		_ = agentOffer.Close()
	}()

	bridge := engine.bridgeForSession(vo.SessionID("session-1"))
	if bridge == nil {
		t.Fatal("bridge not found")
	}
	if bridge.userToAgent == nil {
		t.Fatal("userToAgent local track is nil")
	}
	if bridge.agentToUser == nil {
		t.Fatal("agentToUser local track is nil")
	}
	if bridge.destinationFor(vo.ParticipantRoleUser) != bridge.userToAgent {
		t.Fatal("user remote track destination is not agent local track")
	}
	if bridge.destinationFor(vo.ParticipantRoleAgent) != bridge.agentToUser {
		t.Fatal("agent remote track destination is not user local track")
	}
}
