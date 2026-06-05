package webrtc

import (
	"context"
	"fmt"
	"sync"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
)

type Gateway struct {
	engine  *Engine
	mu      sync.RWMutex
	handler port.MediaRuntimeEventHandler
	offers  map[offerKey]*PeerOffer
}

type offerKey struct {
	sessionID     vo.SessionID
	participantID vo.ParticipantID
}

func NewGateway(engine *Engine) port.MediaGateway {
	return &Gateway{
		engine: engine,
		offers: make(map[offerKey]*PeerOffer),
	}
}

func (g *Gateway) SubscribeRuntimeEvents(handler port.MediaRuntimeEventHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.handler = handler
}

func (g *Gateway) AcceptOffer(ctx context.Context, input port.OfferInput) (*port.Peer, error) {
	peer, err := g.engine.AcceptOffer(ctx, OfferInput{
		SessionID:               input.SessionID,
		ParticipantID:           input.ParticipantID,
		Role:                    input.Role,
		SDP:                     input.SDP,
		PublishAudio:            input.PublishAudio,
		OnConnectionStateChange: g.handleConnectionStateChange(),
		OnMediaTrackStateChange: g.handleMediaTrackStateChange(),
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
		OnConnectionStateChange: g.handleConnectionStateChange(),
		OnMediaTrackStateChange: g.handleMediaTrackStateChange(),
		OnDataChannelMessage:    g.handleDataChannelMessage(),
	})
	if err != nil {
		return nil, err
	}
	g.storeOffer(offer)
	return &port.PeerOffer{
		SessionID:        offer.SessionID,
		ParticipantID:    offer.ParticipantID,
		Role:             offer.Role,
		SDPOffer:         offer.SDPOffer,
		DataChannelLabel: offer.DataChannelLabel,
	}, nil
}

func (g *Gateway) ApplyAnswer(ctx context.Context, offer *port.PeerOffer, answerSDP string) (*port.Peer, error) {
	if offer == nil {
		return nil, fmt.Errorf("apply answer: nil offer")
	}
	webrtcOffer, ok := g.offer(offer.SessionID, offer.ParticipantID)
	if !ok {
		return nil, fmt.Errorf("apply answer: offer not found")
	}
	peer, err := g.engine.ApplyAnswer(ctx, webrtcOffer, answerSDP)
	if err != nil {
		return nil, err
	}
	g.deleteParticipantOffer(offer.SessionID, offer.ParticipantID)
	return toPortPeer(peer), nil
}

func (g *Gateway) CloseSession(ctx context.Context, sessionID vo.SessionID) error {
	g.deleteSessionOffers(sessionID)
	return g.engine.CloseSession(ctx, sessionID)
}

func (g *Gateway) CloseParticipant(ctx context.Context, sessionID vo.SessionID, participantID vo.ParticipantID) error {
	g.deleteParticipantOffer(sessionID, participantID)
	return g.engine.CloseParticipant(ctx, sessionID, participantID)
}

func (g *Gateway) storeOffer(offer *PeerOffer) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.offers[offerKey{sessionID: offer.SessionID, participantID: offer.ParticipantID}] = offer
}

func (g *Gateway) offer(sessionID vo.SessionID, participantID vo.ParticipantID) (*PeerOffer, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	offer, ok := g.offers[offerKey{sessionID: sessionID, participantID: participantID}]
	return offer, ok
}

func (g *Gateway) deleteParticipantOffer(sessionID vo.SessionID, participantID vo.ParticipantID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.offers, offerKey{sessionID: sessionID, participantID: participantID})
}

func (g *Gateway) deleteSessionOffers(sessionID vo.SessionID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for key := range g.offers {
		if key.sessionID == sessionID {
			delete(g.offers, key)
		}
	}
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

func (g *Gateway) eventHandler() port.MediaRuntimeEventHandler {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.handler
}

func (g *Gateway) handleConnectionStateChange() ConnectionStateChangeHandler {
	return func(change ConnectionStateChange) {
		handler := g.eventHandler()
		if handler == nil {
			return
		}
		handler.HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
			SessionID:     change.SessionID,
			ParticipantID: change.ParticipantID,
			Role:          change.Role,
			State:         change.State,
		})
	}
}

func (g *Gateway) handleMediaTrackStateChange() MediaTrackStateChangeHandler {
	return func(change MediaTrackStateChange) {
		handler := g.eventHandler()
		if handler == nil {
			return
		}
		handler.HandleMediaTrackStateChange(context.Background(), port.MediaTrackStateChange{
			SessionID:     change.SessionID,
			ParticipantID: change.ParticipantID,
			Role:          change.Role,
			Kind:          change.Kind,
			State:         change.State,
		})
	}
}

func (g *Gateway) handleDataChannelMessage() DataChannelMessageHandler {
	return func(message DataChannelMessage) {
		handler := g.eventHandler()
		if handler == nil {
			return
		}
		handler.HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
			SessionID:     message.SessionID,
			ParticipantID: message.ParticipantID,
			Role:          message.Role,
			Label:         message.Label,
			Payload:       message.Payload,
		})
	}
}
