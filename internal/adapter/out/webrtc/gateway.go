package webrtc

import (
	"context"
	"fmt"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
)

type Gateway struct {
	engine *Engine
}

func NewGateway(engine *Engine) port.MediaGateway {
	return &Gateway{engine: engine}
}

func (g *Gateway) AcceptOffer(ctx context.Context, input port.OfferInput) (*port.Peer, error) {
	peer, err := g.engine.AcceptOffer(ctx, OfferInput{
		SessionID:               input.SessionID,
		ParticipantID:           input.ParticipantID,
		Role:                    input.Role,
		SDP:                     input.SDP,
		PublishAudio:            input.PublishAudio,
		OnConnectionStateChange: wrapConnectionStateHandler(input.OnConnectionStateChange),
		OnMediaTrackStateChange: wrapMediaTrackStateHandler(input.OnMediaTrackStateChange),
	})
	if err != nil {
		return nil, err
	}
	return toPortPeer(peer), nil
}

func (g *Gateway) CreateOffer(ctx context.Context, input port.CreateOfferInput) (*port.PeerOffer, error) {
	offer, err := g.engine.CreateOffer(ctx, CreateOfferInput{
		SessionID:               input.SessionID,
		ParticipantID:           input.ParticipantID,
		Role:                    input.Role,
		DataChannelLabel:        input.DataChannelLabel,
		InitialDataMessages:     input.InitialDataMessages,
		OnConnectionStateChange: wrapConnectionStateHandler(input.OnConnectionStateChange),
		OnMediaTrackStateChange: wrapMediaTrackStateHandler(input.OnMediaTrackStateChange),
		OnDataChannelMessage:    wrapDataChannelMessageHandler(input.OnDataChannelMessage),
	})
	if err != nil {
		return nil, err
	}
	return &port.PeerOffer{
		SessionID:        offer.SessionID,
		ParticipantID:    offer.ParticipantID,
		Role:             offer.Role,
		SDPOffer:         offer.SDPOffer,
		DataChannelLabel: offer.DataChannelLabel,
		Handle:           offer,
	}, nil
}

func (g *Gateway) ApplyAnswer(ctx context.Context, offer *port.PeerOffer, answerSDP string) (*port.Peer, error) {
	if offer == nil {
		return nil, fmt.Errorf("apply answer: nil offer")
	}
	webrtcOffer, ok := offer.Handle.(*PeerOffer)
	if !ok || webrtcOffer == nil {
		return nil, fmt.Errorf("apply answer: invalid offer handle")
	}
	peer, err := g.engine.ApplyAnswer(ctx, webrtcOffer, answerSDP)
	if err != nil {
		return nil, err
	}
	return toPortPeer(peer), nil
}

func (g *Gateway) CloseSession(ctx context.Context, sessionID vo.SessionID) error {
	return g.engine.CloseSession(ctx, sessionID)
}

func (g *Gateway) CloseParticipant(ctx context.Context, sessionID vo.SessionID, participantID vo.ParticipantID) error {
	return g.engine.CloseParticipant(ctx, sessionID, participantID)
}

func toPortPeer(peer *Peer) *port.Peer {
	if peer == nil {
		return nil
	}
	return &port.Peer{
		SessionID:     peer.SessionID,
		ParticipantID: peer.ParticipantID,
		Role:          peer.Role,
		AnswerSDP:     peer.AnswerSDP,
	}
}

func wrapConnectionStateHandler(handler port.ConnectionStateChangeHandler) ConnectionStateChangeHandler {
	if handler == nil {
		return nil
	}
	return func(change ConnectionStateChange) {
		handler(port.ConnectionStateChange{
			SessionID:     change.SessionID,
			ParticipantID: change.ParticipantID,
			Role:          change.Role,
			State:         change.State,
		})
	}
}

func wrapMediaTrackStateHandler(handler port.MediaTrackStateChangeHandler) MediaTrackStateChangeHandler {
	if handler == nil {
		return nil
	}
	return func(change MediaTrackStateChange) {
		handler(port.MediaTrackStateChange{
			SessionID:     change.SessionID,
			ParticipantID: change.ParticipantID,
			Role:          change.Role,
			Kind:          change.Kind,
			State:         change.State,
		})
	}
}

func wrapDataChannelMessageHandler(handler port.DataChannelMessageHandler) DataChannelMessageHandler {
	if handler == nil {
		return nil
	}
	return func(message DataChannelMessage) {
		handler(port.DataChannelMessage{
			SessionID:     message.SessionID,
			ParticipantID: message.ParticipantID,
			Role:          message.Role,
			Label:         message.Label,
			Payload:       message.Payload,
		})
	}
}
