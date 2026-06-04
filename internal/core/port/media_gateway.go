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
	SessionID               vo.SessionID
	ParticipantID           vo.ParticipantID
	Role                    vo.ParticipantRole
	SDP                     string
	PublishAudio            bool
	OnConnectionStateChange ConnectionStateChangeHandler
	OnMediaTrackStateChange MediaTrackStateChangeHandler
}

type CreateOfferInput struct {
	SessionID               vo.SessionID
	ParticipantID           vo.ParticipantID
	Role                    vo.ParticipantRole
	DataChannelLabel        string
	InitialDataMessages     []string
	OnConnectionStateChange ConnectionStateChangeHandler
	OnMediaTrackStateChange MediaTrackStateChangeHandler
	OnDataChannelMessage    DataChannelMessageHandler
}

type ConnectionStateChangeHandler func(ConnectionStateChange)

type ConnectionStateChange struct {
	SessionID     vo.SessionID
	ParticipantID vo.ParticipantID
	Role          vo.ParticipantRole
	State         vo.ConnectionState
}

type MediaTrackStateChangeHandler func(MediaTrackStateChange)

type MediaTrackStateChange struct {
	SessionID     vo.SessionID
	ParticipantID vo.ParticipantID
	Role          vo.ParticipantRole
	Kind          vo.TrackKind
	State         vo.TrackState
}

type DataChannelMessageHandler func(DataChannelMessage)

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
	Handle           any
}
