package webrtc

import (
	"fmt"
	"sync"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	pionwebrtc "github.com/pion/webrtc/v4"
)

var opusCapability = pionwebrtc.RTPCodecCapability{
	MimeType:    pionwebrtc.MimeTypeOpus,
	ClockRate:   48000,
	Channels:    2,
	SDPFmtpLine: "minptime=10;useinbandfec=1",
}

type audioBridge struct {
	sessionID vo.SessionID

	mu             sync.RWMutex
	clientToOpenAI *pionwebrtc.TrackLocalStaticRTP
	openAIToClient *pionwebrtc.TrackLocalStaticRTP
}

func (b *audioBridge) ensureClientToOpenAITrack() (*pionwebrtc.TrackLocalStaticRTP, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.clientToOpenAI != nil {
		return b.clientToOpenAI, nil
	}

	track, err := pionwebrtc.NewTrackLocalStaticRTP(
		opusCapability,
		fmt.Sprintf("%s-client-to-openai-audio", b.sessionID),
		string(b.sessionID),
	)
	if err != nil {
		return nil, fmt.Errorf("create client-to-openai audio track: %w", err)
	}

	b.clientToOpenAI = track
	return track, nil
}

func (b *audioBridge) ensureOpenAIToClientTrack() (*pionwebrtc.TrackLocalStaticRTP, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.openAIToClient != nil {
		return b.openAIToClient, nil
	}

	track, err := pionwebrtc.NewTrackLocalStaticRTP(
		opusCapability,
		fmt.Sprintf("%s-openai-to-client-audio", b.sessionID),
		string(b.sessionID),
	)
	if err != nil {
		return nil, fmt.Errorf("create openai-to-client audio track: %w", err)
	}

	b.openAIToClient = track
	return track, nil
}

func (b *audioBridge) destinationFor(sourceRole vo.ParticipantRole) *pionwebrtc.TrackLocalStaticRTP {
	b.mu.RLock()
	defer b.mu.RUnlock()

	switch sourceRole {
	case vo.ParticipantRoleClient:
		return b.clientToOpenAI
	case vo.ParticipantRoleOpenAIAgent:
		return b.openAIToClient
	default:
		return nil
	}
}
