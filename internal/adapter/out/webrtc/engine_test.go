package webrtc

import (
	"context"
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	pionwebrtc "github.com/pion/webrtc/v4"
)

func TestEngineHandlesClientAudioOffer(t *testing.T) {
	engine, err := NewEngine(&configs.Config{
		Realtime: configs.RealtimeConfig{
			STUNURLs: []string{"stun:stun.l.google.com:19302"},
		},
	})
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
		t.Fatalf("SetLocalDescription(offer) error = %v", err)
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
	defer func() {
		_ = peer.Close()
	}()

	if peer.AnswerSDP == "" {
		t.Fatal("AcceptOffer() returned empty answer SDP")
	}
	if peer.Role != vo.ParticipantRoleUser {
		t.Fatalf("peer role = %q, want %q", peer.Role, vo.ParticipantRoleUser)
	}
}
