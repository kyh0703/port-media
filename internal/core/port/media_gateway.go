package port

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
)

type OfferAcceptor interface {
	AcceptOffer(ctx context.Context, input OfferInput) (*Peer, error)
}

type OfferCreator interface {
	CreateOffer(ctx context.Context, input CreateOfferInput) (*PeerOffer, error)
}

type MediaGateway interface {
	OfferAcceptor
	OfferCreator
	AnswerApplier
	SessionCloser
	ParticipantCloser
}

type MediaRuntimeEventSubscriber interface {
	SubscribeRuntimeEvents(handler MediaRuntimeEventHandler)
}

type MediaRuntimeEventHandler interface {
	HandleConnectionStateChange(ctx context.Context, change ConnectionStateChange)
	HandleMediaTrackStateChange(ctx context.Context, change MediaTrackStateChange)
	HandleDataChannelMessage(ctx context.Context, message DataChannelMessage)
}

type AnswerApplier interface {
	ApplyAnswer(ctx context.Context, offer *PeerOffer, answerSDP string) (*Peer, error)
}

type SessionCloser interface {
	CloseSession(ctx context.Context, sessionID vo.SessionID) error
}

type ParticipantCloser interface {
	CloseParticipant(ctx context.Context, sessionID vo.SessionID, participantID vo.ParticipantID) error
}

type OfferInput struct {
	SessionID     vo.SessionID
	ParticipantID vo.ParticipantID
	Role          vo.ParticipantRole
	SDP           string
	PublishAudio  bool
}

type CreateOfferInput struct {
	SessionID           vo.SessionID
	ParticipantID       vo.ParticipantID
	Role                vo.ParticipantRole
	DataChannelLabel    string
	InitialDataMessages []string
}

type ConnectionStateChange struct {
	SessionID     vo.SessionID
	ParticipantID vo.ParticipantID
	Role          vo.ParticipantRole
	State         vo.ConnectionState
}

type MediaTrackStateChange struct {
	SessionID     vo.SessionID
	ParticipantID vo.ParticipantID
	Role          vo.ParticipantRole
	Kind          vo.TrackKind
	State         vo.TrackState
}

type DataChannelMessage struct {
	SessionID     vo.SessionID
	ParticipantID vo.ParticipantID
	Role          vo.ParticipantRole
	Label         string
	Payload       string
}

type Peer struct {
	SessionID     vo.SessionID
	ParticipantID vo.ParticipantID
	Role          vo.ParticipantRole
	AnswerSDP     string
}

type PeerOffer struct {
	SessionID        vo.SessionID
	ParticipantID    vo.ParticipantID
	Role             vo.ParticipantRole
	SDPOffer         string
	DataChannelLabel string
}
