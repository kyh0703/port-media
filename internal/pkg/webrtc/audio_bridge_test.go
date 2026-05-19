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
		Role:          vo.ParticipantRoleClient,
		SDP:           offer.SDP,
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}
	defer func() {
		_ = clientPeer.Close()
	}()

	openAIOffer, err := engine.CreateOffer(context.Background(), CreateOfferInput{
		SessionID:     vo.SessionID("session-1"),
		ParticipantID: vo.ParticipantID("openai-1"),
		Role:          vo.ParticipantRoleOpenAIAgent,
	})
	if err != nil {
		t.Fatalf("CreateOffer() error = %v", err)
	}
	defer func() {
		_ = openAIOffer.Close()
	}()

	bridge := engine.bridgeForSession(vo.SessionID("session-1"))
	if bridge == nil {
		t.Fatal("bridge not found")
	}
	if bridge.clientToOpenAI == nil {
		t.Fatal("clientToOpenAI local track is nil")
	}
	if bridge.openAIToClient == nil {
		t.Fatal("openAIToClient local track is nil")
	}
	if bridge.destinationFor(vo.ParticipantRoleClient) != bridge.clientToOpenAI {
		t.Fatal("client remote track destination is not OpenAI local track")
	}
	if bridge.destinationFor(vo.ParticipantRoleOpenAIAgent) != bridge.openAIToClient {
		t.Fatal("OpenAI remote track destination is not client local track")
	}
}
