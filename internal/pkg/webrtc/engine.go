package webrtc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	pionwebrtc "github.com/pion/webrtc/v4"
)

//go:generate go tool counterfeiter -generate
//counterfeiter:generate . PeerConnectionFactory

type OfferAcceptor interface {
	AcceptOffer(ctx context.Context, input OfferInput) (*Peer, error)
}

type OfferCreator interface {
	CreateOffer(ctx context.Context, input CreateOfferInput) (*PeerOffer, error)
}

type PeerConnectionFactory interface {
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

type Engine struct {
	api           *pionwebrtc.API
	configuration pionwebrtc.Configuration
	iceTimeout    time.Duration
	mu            sync.RWMutex
	bridges       map[vo.SessionID]*audioBridge
	connections   map[vo.SessionID]map[vo.ParticipantID]*pionwebrtc.PeerConnection
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

	connection *pionwebrtc.PeerConnection
}

type PeerOffer struct {
	SessionID        vo.SessionID
	ParticipantID    vo.ParticipantID
	Role             vo.ParticipantRole
	SDPOffer         string
	DataChannelLabel string

	connection  *pionwebrtc.PeerConnection
	dataChannel *pionwebrtc.DataChannel
}

func NewEngine(cfg *configs.Config) (*Engine, error) {
	mediaEngine := &pionwebrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:    pionwebrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: 111,
	}, pionwebrtc.RTPCodecTypeAudio); err != nil {
		return nil, fmt.Errorf("register opus codec: %w", err)
	}

	interceptorRegistry := &interceptor.Registry{}
	if err := pionwebrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		return nil, fmt.Errorf("register default interceptors: %w", err)
	}

	return &Engine{
		api: pionwebrtc.NewAPI(
			pionwebrtc.WithMediaEngine(mediaEngine),
			pionwebrtc.WithInterceptorRegistry(interceptorRegistry),
		),
		configuration: pionwebrtc.Configuration{
			ICEServers: toICEServers(cfg.Realtime.STUNURLs),
		},
		iceTimeout:  iceGatheringTimeout(cfg.Realtime.ICEGatheringTimeout),
		bridges:     make(map[vo.SessionID]*audioBridge),
		connections: make(map[vo.SessionID]map[vo.ParticipantID]*pionwebrtc.PeerConnection),
	}, nil
}

func (e *Engine) AcceptOffer(ctx context.Context, input OfferInput) (*Peer, error) {
	_ = ctx
	if input.SDP == "" {
		return nil, fmt.Errorf("accept offer: empty SDP")
	}
	if input.Role != vo.ParticipantRoleClient {
		return nil, fmt.Errorf("accept offer: unsupported role %q", input.Role)
	}

	pc, err := e.api.NewPeerConnection(e.configuration)
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}

	bridge := e.ensureBridge(input.SessionID)
	localTrack, err := bridge.ensureOpenAIToClientTrack()
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	sender, err := pc.AddTrack(localTrack)
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("add client outbound audio track: %w", err)
	}
	go drainSenderRTCP(sender)
	if input.PublishAudio {
		pc.OnTrack(e.forwardRemoteTrack(MediaTrackStateChange{
			SessionID:     input.SessionID,
			ParticipantID: input.ParticipantID,
			Role:          input.Role,
		}, input.OnMediaTrackStateChange))
	}
	e.observeConnectionState(pc, ConnectionStateChange{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
	}, input.OnConnectionStateChange)

	if err := pc.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeOffer,
		SDP:  input.SDP,
	}); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("set remote offer: %w", err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("create answer: %w", err)
	}

	gatherComplete := pionwebrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(answer); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("set local answer: %w", err)
	}
	if err := e.waitForLocalICEGathering(ctx, gatherComplete); err != nil {
		_ = pc.Close()
		return nil, err
	}
	localDescription := pc.LocalDescription()
	if localDescription == nil {
		_ = pc.Close()
		return nil, fmt.Errorf("local answer missing after ICE gathering")
	}

	e.registerConnection(input.SessionID, input.ParticipantID, pc)
	return &Peer{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
		AnswerSDP:     localDescription.SDP,
		connection:    pc,
	}, nil
}

func (e *Engine) CreateOffer(ctx context.Context, input CreateOfferInput) (*PeerOffer, error) {
	_ = ctx
	pc, err := e.api.NewPeerConnection(e.configuration)
	if err != nil {
		return nil, fmt.Errorf("create peer connection: %w", err)
	}
	if input.Role != vo.ParticipantRoleOpenAIAgent {
		_ = pc.Close()
		return nil, fmt.Errorf("create offer: unsupported role %q", input.Role)
	}

	bridge := e.ensureBridge(input.SessionID)
	localTrack, err := bridge.ensureClientToOpenAITrack()
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	sender, err := pc.AddTrack(localTrack)
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("add openai outbound audio track: %w", err)
	}
	go drainSenderRTCP(sender)
	pc.OnTrack(e.forwardRemoteTrack(MediaTrackStateChange{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
	}, input.OnMediaTrackStateChange))
	dataChannel, err := e.createDataChannel(pc, DataChannelInput{
		SessionID:       input.SessionID,
		ParticipantID:   input.ParticipantID,
		Role:            input.Role,
		Label:           input.DataChannelLabel,
		InitialMessages: input.InitialDataMessages,
		OnMessage:       input.OnDataChannelMessage,
	})
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	e.observeConnectionState(pc, ConnectionStateChange{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
	}, input.OnConnectionStateChange)

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("create offer: %w", err)
	}

	gatherComplete := pionwebrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		_ = pc.Close()
		return nil, fmt.Errorf("set local offer: %w", err)
	}
	if err := e.waitForLocalICEGathering(ctx, gatherComplete); err != nil {
		_ = pc.Close()
		return nil, err
	}
	localDescription := pc.LocalDescription()
	if localDescription == nil {
		_ = pc.Close()
		return nil, fmt.Errorf("local offer missing after ICE gathering")
	}

	e.registerConnection(input.SessionID, input.ParticipantID, pc)
	return &PeerOffer{
		SessionID:        input.SessionID,
		ParticipantID:    input.ParticipantID,
		Role:             input.Role,
		SDPOffer:         localDescription.SDP,
		DataChannelLabel: strings.TrimSpace(input.DataChannelLabel),
		connection:       pc,
		dataChannel:      dataChannel,
	}, nil
}

func (e *Engine) ApplyAnswer(ctx context.Context, offer *PeerOffer, answerSDP string) (*Peer, error) {
	_ = ctx
	if offer == nil || offer.connection == nil {
		return nil, fmt.Errorf("apply answer: missing peer offer")
	}
	if answerSDP == "" {
		return nil, fmt.Errorf("apply answer: empty SDP answer")
	}

	if err := offer.connection.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeAnswer,
		SDP:  answerSDP,
	}); err != nil {
		return nil, fmt.Errorf("set remote answer: %w", err)
	}

	return &Peer{
		SessionID:     offer.SessionID,
		ParticipantID: offer.ParticipantID,
		Role:          offer.Role,
		AnswerSDP:     answerSDP,
		connection:    offer.connection,
	}, nil
}

func (p *PeerOffer) Close() error {
	if p.dataChannel != nil {
		_ = p.dataChannel.Close()
	}
	if p.connection == nil {
		return nil
	}

	return p.connection.Close()
}

type DataChannelInput struct {
	SessionID       vo.SessionID
	ParticipantID   vo.ParticipantID
	Role            vo.ParticipantRole
	Label           string
	InitialMessages []string
	OnMessage       DataChannelMessageHandler
}

func (e *Engine) createDataChannel(
	connection *pionwebrtc.PeerConnection,
	input DataChannelInput,
) (*pionwebrtc.DataChannel, error) {
	label := strings.TrimSpace(input.Label)
	if label == "" {
		return nil, nil
	}

	dataChannel, err := connection.CreateDataChannel(label, nil)
	if err != nil {
		return nil, fmt.Errorf("create data channel %q: %w", label, err)
	}

	initialMessages := compactDataChannelMessages(input.InitialMessages)
	dataChannel.OnOpen(func() {
		for _, message := range initialMessages {
			_ = dataChannel.SendText(message)
		}
	})
	if input.OnMessage != nil {
		dataChannel.OnMessage(func(message pionwebrtc.DataChannelMessage) {
			if !message.IsString {
				return
			}
			input.OnMessage(DataChannelMessage{
				SessionID:     input.SessionID,
				ParticipantID: input.ParticipantID,
				Role:          input.Role,
				Label:         label,
				Payload:       string(message.Data),
			})
		})
	}

	return dataChannel, nil
}

func compactDataChannelMessages(messages []string) []string {
	compacted := make([]string, 0, len(messages))
	for _, message := range messages {
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}
		compacted = append(compacted, message)
	}
	return compacted
}

func (p *Peer) Close() error {
	if p.connection == nil {
		return nil
	}

	return p.connection.Close()
}

func (e *Engine) CloseSession(ctx context.Context, sessionID vo.SessionID) error {
	_ = ctx
	e.mu.Lock()
	connectionsByParticipant := e.connections[sessionID]
	delete(e.connections, sessionID)
	delete(e.bridges, sessionID)
	e.mu.Unlock()

	var closeErr error
	for _, connection := range connectionsByParticipant {
		if err := connection.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}

	return closeErr
}

func (e *Engine) CloseParticipant(ctx context.Context, sessionID vo.SessionID, participantID vo.ParticipantID) error {
	_ = ctx
	e.mu.Lock()
	connectionsByParticipant := e.connections[sessionID]
	var connection *pionwebrtc.PeerConnection
	if connectionsByParticipant != nil {
		connection = connectionsByParticipant[participantID]
		delete(connectionsByParticipant, participantID)
		if len(connectionsByParticipant) == 0 {
			delete(e.connections, sessionID)
			delete(e.bridges, sessionID)
		}
	}
	e.mu.Unlock()

	if connection == nil {
		return nil
	}
	return connection.Close()
}

func (e *Engine) registerConnection(sessionID vo.SessionID, participantID vo.ParticipantID, connection *pionwebrtc.PeerConnection) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.connections[sessionID] == nil {
		e.connections[sessionID] = make(map[vo.ParticipantID]*pionwebrtc.PeerConnection)
	}
	e.connections[sessionID][participantID] = connection
}

func (e *Engine) observeConnectionState(
	connection *pionwebrtc.PeerConnection,
	baseChange ConnectionStateChange,
	handler ConnectionStateChangeHandler,
) {
	if handler == nil {
		return
	}

	connection.OnConnectionStateChange(func(state pionwebrtc.PeerConnectionState) {
		change := baseChange
		change.State = mapPeerConnectionState(state)
		handler(change)
	})
}

func (e *Engine) ensureBridge(sessionID vo.SessionID) *audioBridge {
	e.mu.Lock()
	defer e.mu.Unlock()

	bridge := e.bridges[sessionID]
	if bridge == nil {
		bridge = &audioBridge{sessionID: sessionID}
		e.bridges[sessionID] = bridge
	}

	return bridge
}

func (e *Engine) bridgeForSession(sessionID vo.SessionID) *audioBridge {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.bridges[sessionID]
}

func (e *Engine) forwardRemoteTrack(
	baseChange MediaTrackStateChange,
	handler MediaTrackStateChangeHandler,
) func(*pionwebrtc.TrackRemote, *pionwebrtc.RTPReceiver) {
	return func(remoteTrack *pionwebrtc.TrackRemote, _ *pionwebrtc.RTPReceiver) {
		if remoteTrack.Kind() != pionwebrtc.RTPCodecTypeAudio {
			return
		}

		bridge := e.bridgeForSession(baseChange.SessionID)
		if bridge == nil {
			return
		}
		destination := bridge.destinationFor(baseChange.Role)
		if destination == nil {
			return
		}

		notifyMediaTrackState(handler, baseChange, vo.TrackKindAudio, vo.TrackStateActive)
		go func() {
			state := relayRTP(remoteTrack, destination)
			notifyMediaTrackState(handler, baseChange, vo.TrackKindAudio, state)
		}()
	}
}

func relayRTP(remoteTrack *pionwebrtc.TrackRemote, destination *pionwebrtc.TrackLocalStaticRTP) vo.TrackState {
	for {
		packet, _, err := remoteTrack.ReadRTP()
		if err != nil {
			return trackStateForRelayError(err)
		}
		if err := destination.WriteRTP(clonePacket(packet)); err != nil {
			return trackStateForRelayError(err)
		}
	}
}

func notifyMediaTrackState(
	handler MediaTrackStateChangeHandler,
	baseChange MediaTrackStateChange,
	kind vo.TrackKind,
	state vo.TrackState,
) {
	if handler == nil {
		return
	}

	change := baseChange
	change.Kind = kind
	change.State = state
	handler(change)
}

func clonePacket(packet *rtp.Packet) *rtp.Packet {
	if packet == nil {
		return nil
	}

	cloned := *packet
	cloned.Payload = append([]byte(nil), packet.Payload...)
	return &cloned
}

func drainSenderRTCP(sender *pionwebrtc.RTPSender) {
	for {
		if _, _, err := sender.ReadRTCP(); err != nil {
			if err != io.EOF {
				return
			}
			return
		}
	}
}

func (e *Engine) waitForLocalICEGathering(ctx context.Context, done <-chan struct{}) error {
	waitCtx := ctx
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok && e.iceTimeout > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, e.iceTimeout)
	}
	defer cancel()

	if err := waitForICEGatheringComplete(waitCtx, done); err != nil {
		return fmt.Errorf("wait for ICE gathering complete: %w", err)
	}
	return nil
}

func waitForICEGatheringComplete(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func toICEServers(urls []string) []pionwebrtc.ICEServer {
	if len(urls) == 0 {
		return nil
	}

	return []pionwebrtc.ICEServer{{URLs: urls}}
}

func iceGatheringTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	return 5 * time.Second
}

func mapPeerConnectionState(state pionwebrtc.PeerConnectionState) vo.ConnectionState {
	switch state {
	case pionwebrtc.PeerConnectionStateNew:
		return vo.ConnectionStateNew
	case pionwebrtc.PeerConnectionStateConnecting:
		return vo.ConnectionStateConnecting
	case pionwebrtc.PeerConnectionStateConnected:
		return vo.ConnectionStateConnected
	case pionwebrtc.PeerConnectionStateDisconnected:
		return vo.ConnectionStateDisconnected
	case pionwebrtc.PeerConnectionStateFailed:
		return vo.ConnectionStateFailed
	case pionwebrtc.PeerConnectionStateClosed:
		return vo.ConnectionStateClosed
	default:
		return vo.ConnectionStateFailed
	}
}

func trackStateForRelayError(err error) vo.TrackState {
	if errors.Is(err, io.EOF) {
		return vo.TrackStateEnded
	}
	return vo.TrackStateFailed
}
