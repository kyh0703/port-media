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

	mu          sync.RWMutex
	userToAgent *pionwebrtc.TrackLocalStaticRTP
	agentToUser *pionwebrtc.TrackLocalStaticRTP
}

func (b *audioBridge) ensureUserToAgentTrack() (*pionwebrtc.TrackLocalStaticRTP, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.userToAgent != nil {
		return b.userToAgent, nil
	}

	track, err := pionwebrtc.NewTrackLocalStaticRTP(
		opusCapability,
		fmt.Sprintf("%s-user-to-agent-audio", b.sessionID),
		string(b.sessionID),
	)
	if err != nil {
		return nil, fmt.Errorf("create user-to-agent audio track: %w", err)
	}

	b.userToAgent = track
	return track, nil
}

func (b *audioBridge) ensureAgentToUserTrack() (*pionwebrtc.TrackLocalStaticRTP, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.agentToUser != nil {
		return b.agentToUser, nil
	}

	track, err := pionwebrtc.NewTrackLocalStaticRTP(
		opusCapability,
		fmt.Sprintf("%s-agent-to-user-audio", b.sessionID),
		string(b.sessionID),
	)
	if err != nil {
		return nil, fmt.Errorf("create agent-to-user audio track: %w", err)
	}

	b.agentToUser = track
	return track, nil
}

func (b *audioBridge) ensureSubscriberTrack(role vo.ParticipantRole) (*pionwebrtc.TrackLocalStaticRTP, error) {
	switch role {
	case vo.ParticipantRoleUser:
		return b.ensureAgentToUserTrack()
	case vo.ParticipantRoleAgent:
		return b.ensureUserToAgentTrack()
	default:
		return nil, fmt.Errorf("unsupported subscriber role %q", role)
	}
}

func (b *audioBridge) destinationFor(sourceRole vo.ParticipantRole) *pionwebrtc.TrackLocalStaticRTP {
	b.mu.RLock()
	defer b.mu.RUnlock()

	switch sourceRole {
	case vo.ParticipantRoleUser:
		return b.userToAgent
	case vo.ParticipantRoleAgent:
		return b.agentToUser
	default:
		return nil
	}
}
