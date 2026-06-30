package webrtc

import (
	"context"
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestEngineClosesSessionPeersAndBridge(t *testing.T) {
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

	peer, err := engine.AcceptOffer(context.Background(), OfferInput{
		SessionID:     vo.SessionID("session-1"),
		ParticipantID: vo.ParticipantID("client-1"),
		Role:          vo.ParticipantRoleUser,
		SDP:           offer.SDP,
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	if engine.bridgeForSession(vo.SessionID("session-1")) == nil {
		t.Fatal("bridge not created")
	}

	if err := engine.CloseSession(context.Background(), vo.SessionID("session-1")); err != nil {
		t.Fatalf("CloseSession() error = %v", err)
	}

	if engine.bridgeForSession(vo.SessionID("session-1")) != nil {
		t.Fatal("bridge still present after CloseSession")
	}
	if peer.connection.ConnectionState() != pionwebrtc.PeerConnectionStateClosed {
		t.Fatalf("peer state = %q, want closed", peer.connection.ConnectionState())
	}
}

func TestEngineClosesParticipantPeer(t *testing.T) {
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

	peer, err := engine.AcceptOffer(context.Background(), OfferInput{
		SessionID:     vo.SessionID("session-1"),
		ParticipantID: vo.ParticipantID("client-1"),
		Role:          vo.ParticipantRoleUser,
		SDP:           offer.SDP,
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	if err := engine.CloseParticipant(context.Background(), vo.SessionID("session-1"), vo.ParticipantID("client-1")); err != nil {
		t.Fatalf("CloseParticipant() error = %v", err)
	}

	if peer.connection.ConnectionState() != pionwebrtc.PeerConnectionStateClosed {
		t.Fatalf("peer state = %q, want closed", peer.connection.ConnectionState())
	}
	if engine.bridgeForSession(vo.SessionID("session-1")) != nil {
		t.Fatal("bridge still present after closing final participant")
	}
}
