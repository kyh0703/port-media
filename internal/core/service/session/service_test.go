package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"github.com/kyh0703/portfoilo-media/internal/pkg/openai"
	rtc "github.com/kyh0703/portfoilo-media/internal/pkg/webrtc"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type fakeOfferAcceptor struct {
	acceptedInput     rtc.OfferInput
	acceptedInputs    []rtc.OfferInput
	createdInput      rtc.CreateOfferInput
	appliedAnswer     string
	closedSession     vo.SessionID
	closedParticipant vo.ParticipantID
	closeCalls        int
	applyErr          error
	acceptCalls       int
}

func (f *fakeOfferAcceptor) AcceptOffer(ctx context.Context, input rtc.OfferInput) (*rtc.Peer, error) {
	_ = ctx
	f.acceptCalls++
	f.acceptedInput = input
	f.acceptedInputs = append(f.acceptedInputs, input)
	return &rtc.Peer{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
		AnswerSDP:     "answer-sdp",
	}, nil
}

func (f *fakeOfferAcceptor) CreateOffer(ctx context.Context, input rtc.CreateOfferInput) (*rtc.PeerOffer, error) {
	_ = ctx
	f.createdInput = input
	return &rtc.PeerOffer{
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		Role:          input.Role,
		SDPOffer:      "openai-offer-sdp",
	}, nil
}

func (f *fakeOfferAcceptor) ApplyAnswer(ctx context.Context, offer *rtc.PeerOffer, answerSDP string) (*rtc.Peer, error) {
	_ = ctx
	f.appliedAnswer = answerSDP
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return &rtc.Peer{
		SessionID:     offer.SessionID,
		ParticipantID: offer.ParticipantID,
		Role:          offer.Role,
		AnswerSDP:     answerSDP,
	}, nil
}

func (f *fakeOfferAcceptor) CloseSession(ctx context.Context, sessionID vo.SessionID) error {
	_ = ctx
	f.closedSession = sessionID
	f.closeCalls++
	return nil
}

func (f *fakeOfferAcceptor) CloseParticipant(ctx context.Context, sessionID vo.SessionID, participantID vo.ParticipantID) error {
	_ = ctx
	f.closedSession = sessionID
	f.closedParticipant = participantID
	return nil
}

type fakeRealtimeCallCreator struct {
	input       openai.CreateCallInput
	hangupCalls []string
	createErr   error
	createCalls int
}

func (f *fakeRealtimeCallCreator) CreateCall(ctx context.Context, input openai.CreateCallInput) (openai.CreateCallResult, error) {
	_ = ctx
	f.createCalls++
	f.input = input
	if f.createErr != nil {
		return openai.CreateCallResult{}, f.createErr
	}
	return openai.CreateCallResult{
		SDPAnswer:      "openai-answer-sdp",
		ProviderCallID: "rtc_123",
	}, nil
}

func (f *fakeRealtimeCallCreator) HangupCall(ctx context.Context, providerCallID string) error {
	_ = ctx
	f.hangupCalls = append(f.hangupCalls, providerCallID)
	return nil
}

type fakeMediaSessionStateRepository struct {
	states  []entity.MediaSessionState
	deleted []vo.SessionID
}

func (f *fakeMediaSessionStateRepository) Save(ctx context.Context, state entity.MediaSessionState) error {
	_ = ctx
	f.states = append(f.states, state)
	return nil
}

func (f *fakeMediaSessionStateRepository) FindBySessionID(ctx context.Context, sessionID vo.SessionID) (entity.MediaSessionState, bool, error) {
	_ = ctx
	for i := len(f.states) - 1; i >= 0; i-- {
		if f.states[i].SessionID == sessionID {
			return f.states[i], true, nil
		}
	}
	return entity.MediaSessionState{}, false, nil
}

func (f *fakeMediaSessionStateRepository) Delete(ctx context.Context, sessionID vo.SessionID) error {
	_ = ctx
	f.deleted = append(f.deleted, sessionID)
	return nil
}

func TestServiceAcceptsOfferThroughSFU(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	res, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	if res.SDPAnswer != "answer-sdp" {
		t.Fatalf("SDPAnswer = %q, want %q", res.SDPAnswer, "answer-sdp")
	}
	if media.acceptedInput.Role != vo.ParticipantRoleClient {
		t.Fatalf("Role = %q, want %q", media.acceptedInput.Role, vo.ParticipantRoleClient)
	}
	if media.acceptedInput.SessionID != vo.SessionID("session-1") {
		t.Fatalf("SessionID = %q, want session-1", media.acceptedInput.SessionID)
	}
	if !media.acceptedInput.PublishAudio {
		t.Fatal("PublishAudio = false, want true")
	}
}

func TestServiceCreatesRoomRuntimeForClientJoin(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	res, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}
	if room.Status != vo.RoomStatusActive {
		t.Fatalf("room status = %q, want %q", room.Status, vo.RoomStatusActive)
	}
	if len(room.Participants) != 2 {
		t.Fatalf("participants len = %d, want 2", len(room.Participants))
	}

	participant, ok := room.Participants[vo.ParticipantID(res.ParticipantID)]
	if !ok {
		t.Fatalf("participant %q not found in room", res.ParticipantID)
	}
	if participant.Role != vo.ParticipantRoleClient {
		t.Fatalf("participant role = %q, want %q", participant.Role, vo.ParticipantRoleClient)
	}
	if participant.State != vo.ConnectionStateConnecting {
		t.Fatalf("participant state = %q, want %q", participant.State, vo.ConnectionStateConnecting)
	}
	if len(participant.Tracks) != 1 {
		t.Fatalf("track len = %d, want 1", len(participant.Tracks))
	}
	for _, track := range participant.Tracks {
		if track.Kind != vo.TrackKindAudio {
			t.Fatalf("track kind = %q, want %q", track.Kind, vo.TrackKindAudio)
		}
	}
}

func TestServiceAllowsMultipleAudioPublishers(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() first error = %v", err)
	}

	_, err = svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() second publisher error = %v", err)
	}
	if media.acceptCalls != 2 {
		t.Fatalf("media AcceptOffer calls = %d, want 2", media.acceptCalls)
	}
	if provider.createCalls != 1 {
		t.Fatalf("provider CreateCall calls = %d, want 1", provider.createCalls)
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}
	var publishers int
	for _, participant := range room.Participants {
		if participant.Role == vo.ParticipantRoleClient && participant.PublishAudio {
			publishers++
		}
	}
	if publishers != 2 {
		t.Fatalf("audio publishers = %d, want 2", publishers)
	}
}

func TestServiceAllowsMultipleClientsWhenAdditionalClientIsListener(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "publisher-offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() publisher error = %v", err)
	}

	_, err = svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "listener-offer-sdp",
		AudioMode:      sessiondto.AudioModeListener,
	})
	if err != nil {
		t.Fatalf("AcceptOffer() listener error = %v", err)
	}

	if media.acceptCalls != 2 {
		t.Fatalf("media AcceptOffer calls = %d, want 2", media.acceptCalls)
	}
	if len(media.acceptedInputs) != 2 {
		t.Fatalf("accepted inputs len = %d, want 2", len(media.acceptedInputs))
	}
	if !media.acceptedInputs[0].PublishAudio {
		t.Fatal("publisher PublishAudio = false, want true")
	}
	if media.acceptedInputs[1].PublishAudio {
		t.Fatal("listener PublishAudio = true, want false")
	}
	if provider.createCalls != 1 {
		t.Fatalf("provider CreateCall calls = %d, want 1", provider.createCalls)
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}
	if len(room.Participants) != 3 {
		t.Fatalf("participants len = %d, want 3", len(room.Participants))
	}

	var clients, publishers int
	for _, participant := range room.Participants {
		if participant.Role != vo.ParticipantRoleClient {
			continue
		}
		clients++
		if participant.PublishAudio {
			publishers++
		}
	}
	if clients != 2 {
		t.Fatalf("client participants = %d, want 2", clients)
	}
	if publishers != 1 {
		t.Fatalf("audio publishers = %d, want 1", publishers)
	}
}

func TestServiceLeavesClientParticipantWithoutClosingRoom(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	res, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	leave, err := svc.LeaveParticipant(context.Background(), sessiondto.LeaveParticipantRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		ParticipantID:  res.ParticipantID,
	})
	if err != nil {
		t.Fatalf("LeaveParticipant() error = %v", err)
	}

	if leave.Status != string(vo.RoomStatusActive) {
		t.Fatalf("leave status = %q, want active", leave.Status)
	}
	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if media.closedParticipant != vo.ParticipantID(res.ParticipantID) {
		t.Fatalf("closed participant = %q, want %q", media.closedParticipant, res.ParticipantID)
	}
	if media.closeCalls != 0 {
		t.Fatalf("CloseSession calls = %d, want 0", media.closeCalls)
	}
	if len(provider.hangupCalls) != 0 {
		t.Fatalf("provider hangup calls = %d, want 0", len(provider.hangupCalls))
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}
	if len(room.Participants) != 1 {
		t.Fatalf("participants len = %d, want 1", len(room.Participants))
	}
	for _, participant := range room.Participants {
		if participant.Role != vo.ParticipantRoleOpenAIAgent {
			t.Fatalf("remaining role = %q, want %q", participant.Role, vo.ParticipantRoleOpenAIAgent)
		}
	}

	state := states.states[len(states.states)-1]
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("state status = %q, want active", state.Status)
	}
	if state.Participants != 1 {
		t.Fatalf("state participants = %d, want 1", state.Participants)
	}
}

func TestServiceLeavesCriticalParticipantByFailingRoom(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	leave, err := svc.LeaveParticipant(context.Background(), sessiondto.LeaveParticipantRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		ParticipantID:  string(media.createdInput.ParticipantID),
	})
	if err != nil {
		t.Fatalf("LeaveParticipant() error = %v", err)
	}

	if leave.Status != string(vo.RoomStatusFailed) {
		t.Fatalf("leave status = %q, want failed", leave.Status)
	}
	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if media.closeCalls != 1 {
		t.Fatalf("CloseSession calls = %d, want 1", media.closeCalls)
	}
	if len(provider.hangupCalls) != 1 || provider.hangupCalls[0] != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.hangupCalls)
	}

	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	} else if found {
		t.Fatal("runtime room found, want deleted")
	}

	state := states.states[len(states.states)-1]
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("state status = %q, want failed", state.Status)
	}
	if state.Participants != 2 {
		t.Fatalf("state participants = %d, want 2", state.Participants)
	}
}

func TestServiceLogsMonitoringLifecycleFields(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	core, observed := observer.New(zap.InfoLevel)
	svc := newService(rooms, runtime, states, media, provider, defaultRealtimeControlConfig(), zap.New(core))

	res, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	joinedEntries := observed.FilterMessage("media_participant_joined").All()
	if len(joinedEntries) != 1 {
		t.Fatalf("participant joined log entries = %d, want 1", len(joinedEntries))
	}
	joined := joinedEntries[0].ContextMap()
	if joined["session_id"] != "session-1" {
		t.Fatalf("session_id = %v, want session-1", joined["session_id"])
	}
	if joined["conversation_id"] != "conversation-1" {
		t.Fatalf("conversation_id = %v, want conversation-1", joined["conversation_id"])
	}
	if joined["participant_id"] != res.ParticipantID {
		t.Fatalf("participant_id = %v, want %s", joined["participant_id"], res.ParticipantID)
	}
	if joined["participant_role"] != string(vo.ParticipantRoleClient) {
		t.Fatalf("participant_role = %v, want client", joined["participant_role"])
	}
	if joined["audio_mode"] != string(sessiondto.AudioModePublisher) {
		t.Fatalf("audio_mode = %v, want publisher", joined["audio_mode"])
	}

	media.acceptedInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.acceptedInput.SessionID,
		ParticipantID: media.acceptedInput.ParticipantID,
		Role:          media.acceptedInput.Role,
		State:         vo.ConnectionStateConnected,
	})
	connectionEntries := observed.FilterMessage("media_participant_connection_state_changed").All()
	if len(connectionEntries) != 1 {
		t.Fatalf("connection state log entries = %d, want 1", len(connectionEntries))
	}
	connection := connectionEntries[0].ContextMap()
	if connection["connection_state"] != string(vo.ConnectionStateConnected) {
		t.Fatalf("connection_state = %v, want connected", connection["connection_state"])
	}

	media.createdInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.createdInput.SessionID,
		ParticipantID: media.createdInput.ParticipantID,
		Role:          media.createdInput.Role,
		State:         vo.ConnectionStateFailed,
	})
	failedEntries := observed.FilterMessage("media_room_failed").All()
	if len(failedEntries) != 1 {
		t.Fatalf("room failed log entries = %d, want 1", len(failedEntries))
	}
	failed := failedEntries[0].ContextMap()
	if failed["failure_reason"] != "critical_participant_connection_failed" {
		t.Fatalf("failure_reason = %v, want critical_participant_connection_failed", failed["failure_reason"])
	}
	if failed["participant_role"] != string(vo.ParticipantRoleOpenAIAgent) {
		t.Fatalf("participant_role = %v, want openai_agent", failed["participant_role"])
	}
}

func TestServicePersistsRoomMetadataForClientJoin(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	room, found, err := rooms.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata room not found")
	}
	if room.ConversationID != vo.ConversationID("conversation-1") {
		t.Fatalf("ConversationID = %q, want conversation-1", room.ConversationID)
	}
}

func TestServiceConnectsOpenAIParticipantForClientJoin(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	if media.createdInput.Role != vo.ParticipantRoleOpenAIAgent {
		t.Fatalf("created role = %q, want %q", media.createdInput.Role, vo.ParticipantRoleOpenAIAgent)
	}
	if media.createdInput.DataChannelLabel != "oai-events" {
		t.Fatalf("data channel label = %q, want oai-events", media.createdInput.DataChannelLabel)
	}
	if provider.input.SDPOffer != "openai-offer-sdp" {
		t.Fatalf("provider SDPOffer = %q, want openai-offer-sdp", provider.input.SDPOffer)
	}
	if media.appliedAnswer != "openai-answer-sdp" {
		t.Fatalf("applied answer = %q, want openai-answer-sdp", media.appliedAnswer)
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}

	var foundAgent bool
	for _, participant := range room.Participants {
		if participant.Role == vo.ParticipantRoleOpenAIAgent {
			foundAgent = true
			if participant.State != vo.ConnectionStateConnecting {
				t.Fatalf("openai participant state = %q, want %q", participant.State, vo.ConnectionStateConnecting)
			}
			if participant.ProviderCallID != "rtc_123" {
				t.Fatalf("provider call id = %q, want rtc_123", participant.ProviderCallID)
			}
			if len(participant.Tracks) != 1 {
				t.Fatalf("openai participant track len = %d, want 1", len(participant.Tracks))
			}
		}
	}
	if !foundAgent {
		t.Fatal("openai participant not found")
	}
}

func TestServiceUsesConfiguredOpenAIRealtimeDataChannel(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewServiceWithConfig(rooms, runtime, states, media, provider, &configs.Config{
		OpenAI: configs.OpenAIConfig{
			RealtimeDataChannelLabel: "custom-events",
			RealtimeInitialEvents: []string{
				`{"type":"session.update"}`,
				" ",
				`{"type":"response.create"}`,
			},
		},
	})

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	if media.createdInput.DataChannelLabel != "custom-events" {
		t.Fatalf("DataChannelLabel = %q, want custom-events", media.createdInput.DataChannelLabel)
	}
	if len(media.createdInput.InitialDataMessages) != 2 {
		t.Fatalf("InitialDataMessages len = %d, want 2", len(media.createdInput.InitialDataMessages))
	}
	if media.createdInput.InitialDataMessages[0] != `{"type":"session.update"}` {
		t.Fatalf("InitialDataMessages[0] = %q", media.createdInput.InitialDataMessages[0])
	}
}

func TestServiceStoresRealtimeDataChannelEventInLiveState(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	media.createdInput.OnDataChannelMessage(rtc.DataChannelMessage{
		SessionID:     media.createdInput.SessionID,
		ParticipantID: media.createdInput.ParticipantID,
		Role:          media.createdInput.Role,
		Label:         media.createdInput.DataChannelLabel,
		Payload:       `{"type":"response.done"}`,
	})

	if len(states.states) != 2 {
		t.Fatalf("state saves = %d, want 2", len(states.states))
	}
	state := states.states[1]
	if state.LastRealtimeEventType != "response.done" {
		t.Fatalf("LastRealtimeEventType = %q, want response.done", state.LastRealtimeEventType)
	}
	if state.LastRealtimeEventAt.IsZero() {
		t.Fatal("LastRealtimeEventAt is zero")
	}
	if len(state.RecentRealtimeEvents) != 1 {
		t.Fatalf("RecentRealtimeEvents len = %d, want 1", len(state.RecentRealtimeEvents))
	}
	if state.RecentRealtimeEvents[0].Type != "response.done" {
		t.Fatalf("RecentRealtimeEvents[0].Type = %q, want response.done", state.RecentRealtimeEvents[0].Type)
	}

	status, found, err := svc.GetSessionStatus(context.Background(), sessiondto.GetSessionStatusRequest{
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("GetSessionStatus() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if status.LastRealtimeEventType != "response.done" {
		t.Fatalf("status LastRealtimeEventType = %q, want response.done", status.LastRealtimeEventType)
	}
	if status.LastRealtimeEventAt == "" {
		t.Fatal("status LastRealtimeEventAt is empty")
	}
	if len(status.RecentRealtimeEvents) != 1 {
		t.Fatalf("status RecentRealtimeEvents len = %d, want 1", len(status.RecentRealtimeEvents))
	}
	if status.RecentRealtimeEvents[0].Type != "response.done" {
		t.Fatalf("status RecentRealtimeEvents[0].Type = %q, want response.done", status.RecentRealtimeEvents[0].Type)
	}
}

func TestServiceLimitsRecentRealtimeEvents(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewServiceWithConfig(rooms, runtime, states, media, provider, &configs.Config{
		Realtime: configs.RealtimeConfig{RealtimeEventHistoryLimit: 2},
	})

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	for _, eventType := range []string{"session.created", "response.created", "response.done"} {
		media.createdInput.OnDataChannelMessage(rtc.DataChannelMessage{
			SessionID:     media.createdInput.SessionID,
			ParticipantID: media.createdInput.ParticipantID,
			Role:          media.createdInput.Role,
			Label:         media.createdInput.DataChannelLabel,
			Payload:       `{"type":"` + eventType + `"}`,
		})
	}

	state := states.states[len(states.states)-1]
	if len(state.RecentRealtimeEvents) != 2 {
		t.Fatalf("RecentRealtimeEvents len = %d, want 2", len(state.RecentRealtimeEvents))
	}
	if state.RecentRealtimeEvents[0].Type != "response.created" {
		t.Fatalf("RecentRealtimeEvents[0].Type = %q, want response.created", state.RecentRealtimeEvents[0].Type)
	}
	if state.RecentRealtimeEvents[1].Type != "response.done" {
		t.Fatalf("RecentRealtimeEvents[1].Type = %q, want response.done", state.RecentRealtimeEvents[1].Type)
	}
}

func TestRealtimeEventTypeFallsBackToUnknown(t *testing.T) {
	if got := realtimeEventType(`{"event":"missing"}`); got != "unknown" {
		t.Fatalf("realtimeEventType() = %q, want unknown", got)
	}
	if got := realtimeEventType(`not-json`); got != "unknown" {
		t.Fatalf("realtimeEventType() invalid = %q, want unknown", got)
	}
}

func TestServiceStoresActiveMediaSessionStateForClientJoin(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	if len(states.states) != 1 {
		t.Fatalf("state saves = %d, want 1", len(states.states))
	}
	state := states.states[0]
	if state.SessionID != vo.SessionID("session-1") {
		t.Fatalf("SessionID = %q, want session-1", state.SessionID)
	}
	if state.ConversationID != vo.ConversationID("conversation-1") {
		t.Fatalf("ConversationID = %q, want conversation-1", state.ConversationID)
	}
	if state.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", state.UserID)
	}
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusActive)
	}
	if state.ConnectionState != vo.ConnectionStateConnecting {
		t.Fatalf("ConnectionState = %q, want %q", state.ConnectionState, vo.ConnectionStateConnecting)
	}
	if state.MediaState != vo.TrackStatePending {
		t.Fatalf("MediaState = %q, want %q", state.MediaState, vo.TrackStatePending)
	}
	if state.OpenAIProviderCallID != "rtc_123" {
		t.Fatalf("OpenAIProviderCallID = %q, want rtc_123", state.OpenAIProviderCallID)
	}
	if state.Participants != 2 {
		t.Fatalf("Participants = %d, want 2", state.Participants)
	}
	if len(state.ParticipantStates) != 2 {
		t.Fatalf("ParticipantStates len = %d, want 2", len(state.ParticipantStates))
	}
	var foundPublisher bool
	for _, participantState := range state.ParticipantStates {
		if participantState.Role == vo.ParticipantRoleClient && participantState.AudioMode == "publisher" {
			foundPublisher = true
		}
	}
	if !foundPublisher {
		t.Fatalf("participant states = %#v, want client publisher", state.ParticipantStates)
	}
	if state.RoomID == "" {
		t.Fatal("RoomID is empty")
	}
	if state.StartedAt.IsZero() || state.UpdatedAt.IsZero() {
		t.Fatalf("timestamps must be set: started=%v updated=%v", state.StartedAt, state.UpdatedAt)
	}
}

func TestServiceStoresFailedMediaSessionStateWhenOpenAICallFails(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{createErr: errors.New("openai unavailable")}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err == nil {
		t.Fatal("AcceptOffer() error = nil, want error")
	}

	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if len(provider.hangupCalls) != 0 {
		t.Fatalf("hangup calls = %#v, want none because provider call was not created", provider.hangupCalls)
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
	if len(states.states) != 1 {
		t.Fatalf("state saves = %d, want 1", len(states.states))
	}
	state := states.states[0]
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusFailed)
	}
	if state.ConnectionState != vo.ConnectionStateFailed {
		t.Fatalf("ConnectionState = %q, want %q", state.ConnectionState, vo.ConnectionStateFailed)
	}
	if state.MediaState != vo.TrackStateFailed {
		t.Fatalf("MediaState = %q, want %q", state.MediaState, vo.TrackStateFailed)
	}

	room, found, err := rooms.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata room not found")
	}
	if room.Status != vo.RoomStatusFailed {
		t.Fatalf("metadata status = %q, want %q", room.Status, vo.RoomStatusFailed)
	}
}

func TestServiceHangsUpOpenAICallWhenApplyAnswerFails(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{applyErr: errors.New("invalid openai answer")}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err == nil {
		t.Fatal("AcceptOffer() error = nil, want error")
	}

	if len(provider.hangupCalls) != 1 || provider.hangupCalls[0] != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.hangupCalls)
	}
	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if len(states.states) != 1 {
		t.Fatalf("state saves = %d, want 1", len(states.states))
	}
	if states.states[0].Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", states.states[0].Status, vo.RoomStatusFailed)
	}
}

func TestServiceStoresMediaSessionStateWhenTrackStateChanges(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	media.acceptedInput.OnMediaTrackStateChange(rtc.MediaTrackStateChange{
		SessionID:     media.acceptedInput.SessionID,
		ParticipantID: media.acceptedInput.ParticipantID,
		Role:          media.acceptedInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateActive,
	})

	if len(states.states) != 2 {
		t.Fatalf("state saves = %d, want 2", len(states.states))
	}
	state := states.states[1]
	if state.MediaState != vo.TrackStateActive {
		t.Fatalf("MediaState = %q, want %q", state.MediaState, vo.TrackStateActive)
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("runtime room not found")
	}
	participant := room.Participants[media.acceptedInput.ParticipantID]
	for _, track := range participant.Tracks {
		if track.Kind == vo.TrackKindAudio && track.State != vo.TrackStateActive {
			t.Fatalf("audio track state = %q, want %q", track.State, vo.TrackStateActive)
		}
	}
}

func TestServiceKeepsRoomActiveWhenClientConnectionFails(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	media.acceptedInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.acceptedInput.SessionID,
		ParticipantID: media.acceptedInput.ParticipantID,
		Role:          media.acceptedInput.Role,
		State:         vo.ConnectionStateFailed,
	})

	if media.closedSession != "" {
		t.Fatalf("closed session = %q, want empty", media.closedSession)
	}
	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("runtime room not found")
	}
	state := states.states[len(states.states)-1]
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusActive)
	}
	if state.ConnectionState == vo.ConnectionStateFailed {
		t.Fatalf("ConnectionState = %q, want non-failed", state.ConnectionState)
	}
	participant := room.Participants[media.acceptedInput.ParticipantID]
	if participant.State != vo.ConnectionStateFailed {
		t.Fatalf("client participant state = %q, want failed", participant.State)
	}
}

func TestServiceKeepsRoomActiveWhenClientMediaTrackFails(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	media.acceptedInput.OnMediaTrackStateChange(rtc.MediaTrackStateChange{
		SessionID:     media.acceptedInput.SessionID,
		ParticipantID: media.acceptedInput.ParticipantID,
		Role:          media.acceptedInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateFailed,
	})

	if media.closedSession != "" {
		t.Fatalf("closed session = %q, want empty", media.closedSession)
	}
	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("runtime room not found")
	}
	state := states.states[len(states.states)-1]
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusActive)
	}
	if state.MediaState == vo.TrackStateFailed {
		t.Fatalf("MediaState = %q, want non-failed", state.MediaState)
	}
	participant := room.Participants[media.acceptedInput.ParticipantID]
	for _, track := range participant.Tracks {
		if track.Kind == vo.TrackKindAudio && track.State != vo.TrackStateFailed {
			t.Fatalf("client audio track state = %q, want failed", track.State)
		}
	}
}

func TestServiceFailsSessionWhenOpenAIConnectionFails(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	media.createdInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.createdInput.SessionID,
		ParticipantID: media.createdInput.ParticipantID,
		Role:          media.createdInput.Role,
		State:         vo.ConnectionStateFailed,
	})

	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if len(provider.hangupCalls) != 1 || provider.hangupCalls[0] != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.hangupCalls)
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
	state := states.states[len(states.states)-1]
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusFailed)
	}
	if state.ConnectionState != vo.ConnectionStateFailed {
		t.Fatalf("ConnectionState = %q, want %q", state.ConnectionState, vo.ConnectionStateFailed)
	}
}

func TestServiceFailsSessionWhenOpenAIMediaTrackFails(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	media.createdInput.OnMediaTrackStateChange(rtc.MediaTrackStateChange{
		SessionID:     media.createdInput.SessionID,
		ParticipantID: media.createdInput.ParticipantID,
		Role:          media.createdInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateFailed,
	})

	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if len(provider.hangupCalls) != 1 || provider.hangupCalls[0] != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.hangupCalls)
	}
	state := states.states[len(states.states)-1]
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusFailed)
	}
	if state.MediaState != vo.TrackStateFailed {
		t.Fatalf("MediaState = %q, want %q", state.MediaState, vo.TrackStateFailed)
	}
}

func TestServiceStoresMediaSessionStateWhenConnectionStateChanges(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	media.acceptedInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.acceptedInput.SessionID,
		ParticipantID: media.acceptedInput.ParticipantID,
		Role:          media.acceptedInput.Role,
		State:         vo.ConnectionStateConnected,
	})
	media.createdInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.createdInput.SessionID,
		ParticipantID: media.createdInput.ParticipantID,
		Role:          media.createdInput.Role,
		State:         vo.ConnectionStateConnected,
	})

	if len(states.states) != 3 {
		t.Fatalf("state saves = %d, want 3", len(states.states))
	}
	state := states.states[2]
	if state.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", state.UserID)
	}
	if state.ConnectionState != vo.ConnectionStateConnected {
		t.Fatalf("ConnectionState = %q, want %q", state.ConnectionState, vo.ConnectionStateConnected)
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("runtime room not found")
	}
	participant := room.Participants[media.acceptedInput.ParticipantID]
	if participant.State != vo.ConnectionStateConnected {
		t.Fatalf("participant state = %q, want %q", participant.State, vo.ConnectionStateConnected)
	}
}

func TestServiceEndsSessionCleanup(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	res, err := svc.EndSession(context.Background(), sessiondto.EndSessionRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
	})
	if err != nil {
		t.Fatalf("EndSession() error = %v", err)
	}

	if res.Status != string(vo.RoomStatusClosed) {
		t.Fatalf("status = %q, want %q", res.Status, vo.RoomStatusClosed)
	}
	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if len(provider.hangupCalls) != 1 || provider.hangupCalls[0] != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.hangupCalls)
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}

	room, found, err := rooms.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("metadata FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata room not found")
	}
	if room.Status != vo.RoomStatusClosed {
		t.Fatalf("metadata status = %q, want %q", room.Status, vo.RoomStatusClosed)
	}
}

func TestServiceCleansUpIdleRuntimeRooms(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)
	svcImpl := svc.(*service)

	oldNow := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	currentNow := oldNow.Add(3 * time.Minute)
	svcImpl.now = func() time.Time { return currentNow }

	room := entity.NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), oldNow)
	room.SetUserID("user-1", oldNow)
	client := entity.NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleClient, oldNow)
	client.SetState(vo.ConnectionStateConnected, oldNow)
	client.AddTrack(entity.NewTrack(vo.TrackID("track-1"), vo.TrackKindAudio, oldNow), oldNow)
	room.AddParticipant(client, oldNow)
	agent := entity.NewParticipant(vo.ParticipantID("agent-1"), vo.ParticipantRoleOpenAIAgent, oldNow)
	agent.SetState(vo.ConnectionStateConnected, oldNow)
	agent.SetProviderCallID("rtc_123", oldNow)
	agent.AddTrack(entity.NewTrack(vo.TrackID("track-2"), vo.TrackKindAudio, oldNow), oldNow)
	room.AddParticipant(agent, oldNow)
	room.UpdatedAt = oldNow

	if err := runtime.Save(context.Background(), room); err != nil {
		t.Fatalf("runtime Save() error = %v", err)
	}
	if err := rooms.Save(context.Background(), room); err != nil {
		t.Fatalf("rooms Save() error = %v", err)
	}

	cleaned, err := svc.CleanupIdleRooms(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("CleanupIdleRooms() error = %v", err)
	}

	if cleaned != 1 {
		t.Fatalf("cleaned = %d, want 1", cleaned)
	}
	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if len(provider.hangupCalls) != 1 || provider.hangupCalls[0] != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.hangupCalls)
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}

	room, found, err := rooms.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("metadata FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata room not found")
	}
	if room.Status != vo.RoomStatusClosed {
		t.Fatalf("metadata status = %q, want %q", room.Status, vo.RoomStatusClosed)
	}
	if len(states.states) != 1 || states.states[0].Status != vo.RoomStatusClosed {
		t.Fatalf("states = %#v, want one closed state", states.states)
	}
}

func TestServiceKeepsRecentlyUpdatedRuntimeRooms(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)
	svcImpl := svc.(*service)

	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	svcImpl.now = func() time.Time { return now }
	room := entity.NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now.Add(-30*time.Second))
	room.SetUserID("user-1", now.Add(-30*time.Second))
	if err := runtime.Save(context.Background(), room); err != nil {
		t.Fatalf("runtime Save() error = %v", err)
	}

	cleaned, err := svc.CleanupIdleRooms(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("CleanupIdleRooms() error = %v", err)
	}

	if cleaned != 0 {
		t.Fatalf("cleaned = %d, want 0", cleaned)
	}
	if media.closeCalls != 0 {
		t.Fatalf("media close calls = %d, want 0", media.closeCalls)
	}
}

func TestServiceShutdownClosesActiveRuntimeRooms(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)
	svcImpl := svc.(*service)

	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	svcImpl.now = func() time.Time { return now }

	room := entity.NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	room.SetUserID("user-1", now)
	client := entity.NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleClient, now)
	client.SetState(vo.ConnectionStateConnected, now)
	client.AddTrack(entity.NewTrack(vo.TrackID("track-1"), vo.TrackKindAudio, now), now)
	room.AddParticipant(client, now)
	agent := entity.NewParticipant(vo.ParticipantID("agent-1"), vo.ParticipantRoleOpenAIAgent, now)
	agent.SetState(vo.ConnectionStateConnected, now)
	agent.SetProviderCallID("rtc_123", now)
	agent.AddTrack(entity.NewTrack(vo.TrackID("track-2"), vo.TrackKindAudio, now), now)
	room.AddParticipant(agent, now)

	if err := runtime.Save(context.Background(), room); err != nil {
		t.Fatalf("runtime Save() error = %v", err)
	}

	cleaned, err := svc.ShutdownActiveRooms(context.Background())
	if err != nil {
		t.Fatalf("ShutdownActiveRooms() error = %v", err)
	}

	if cleaned != 1 {
		t.Fatalf("cleaned = %d, want 1", cleaned)
	}
	if media.closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", media.closedSession)
	}
	if len(provider.hangupCalls) != 1 || provider.hangupCalls[0] != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.hangupCalls)
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
	if len(states.states) != 1 || states.states[0].Status != vo.RoomStatusClosed {
		t.Fatalf("states = %#v, want one closed state", states.states)
	}
}

func TestServiceStoresClosedMediaSessionStateWhenSessionEnds(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	_, err = svc.EndSession(context.Background(), sessiondto.EndSessionRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
	})
	if err != nil {
		t.Fatalf("EndSession() error = %v", err)
	}

	if len(states.states) != 2 {
		t.Fatalf("state saves = %d, want 2", len(states.states))
	}
	state := states.states[1]
	if state.SessionID != vo.SessionID("session-1") {
		t.Fatalf("SessionID = %q, want session-1", state.SessionID)
	}
	if state.Status != vo.RoomStatusClosed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusClosed)
	}
	if state.ConnectionState != vo.ConnectionStateClosed {
		t.Fatalf("ConnectionState = %q, want %q", state.ConnectionState, vo.ConnectionStateClosed)
	}
	if state.UserID != "user-1" {
		t.Fatalf("UserID = %q, want user-1", state.UserID)
	}
	if state.OpenAIProviderCallID != "rtc_123" {
		t.Fatalf("OpenAIProviderCallID = %q, want rtc_123", state.OpenAIProviderCallID)
	}
}

func TestServiceReturnsRuntimeStats(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}
	media.acceptedInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.acceptedInput.SessionID,
		ParticipantID: media.acceptedInput.ParticipantID,
		Role:          media.acceptedInput.Role,
		State:         vo.ConnectionStateConnected,
	})
	media.createdInput.OnConnectionStateChange(rtc.ConnectionStateChange{
		SessionID:     media.createdInput.SessionID,
		ParticipantID: media.createdInput.ParticipantID,
		Role:          media.createdInput.Role,
		State:         vo.ConnectionStateConnected,
	})
	media.acceptedInput.OnMediaTrackStateChange(rtc.MediaTrackStateChange{
		SessionID:     media.acceptedInput.SessionID,
		ParticipantID: media.acceptedInput.ParticipantID,
		Role:          media.acceptedInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateActive,
	})
	media.createdInput.OnDataChannelMessage(rtc.DataChannelMessage{
		SessionID:     media.createdInput.SessionID,
		ParticipantID: media.createdInput.ParticipantID,
		Role:          media.createdInput.Role,
		Label:         media.createdInput.DataChannelLabel,
		Payload:       `{"type":"response.done"}`,
	})

	stats, err := svc.GetRuntimeStats(context.Background())
	if err != nil {
		t.Fatalf("GetRuntimeStats() error = %v", err)
	}

	if stats.Rooms != 1 {
		t.Fatalf("Rooms = %d, want 1", stats.Rooms)
	}
	if stats.Sessions != 1 {
		t.Fatalf("Sessions = %d, want 1", stats.Sessions)
	}
	if stats.Participants != 2 {
		t.Fatalf("Participants = %d, want 2", stats.Participants)
	}
	if stats.Tracks != 2 {
		t.Fatalf("Tracks = %d, want 2", stats.Tracks)
	}
	if stats.ByStatus[string(vo.RoomStatusActive)] != 1 {
		t.Fatalf("active status count = %d, want 1", stats.ByStatus[string(vo.RoomStatusActive)])
	}
	if stats.ByConnection[string(vo.ConnectionStateConnected)] != 1 {
		t.Fatalf("connected count = %d, want 1", stats.ByConnection[string(vo.ConnectionStateConnected)])
	}
	if stats.ByMedia[string(vo.TrackStateActive)] != 1 {
		t.Fatalf("active media count = %d, want 1", stats.ByMedia[string(vo.TrackStateActive)])
	}
	if stats.ByRole[string(vo.ParticipantRoleClient)] != 1 {
		t.Fatalf("client role count = %d, want 1", stats.ByRole[string(vo.ParticipantRoleClient)])
	}
	if stats.ByAudioMode[string(sessiondto.AudioModePublisher)] != 1 {
		t.Fatalf("publisher count = %d, want 1", stats.ByAudioMode[string(sessiondto.AudioModePublisher)])
	}
	if stats.ByRealtimeEvent["response.done"] != 1 {
		t.Fatalf("response.done count = %d, want 1", stats.ByRealtimeEvent["response.done"])
	}
	if len(stats.RoomsDetail) != 1 {
		t.Fatalf("RoomsDetail len = %d, want 1", len(stats.RoomsDetail))
	}
	if stats.RoomsDetail[0].Publishers != 1 {
		t.Fatalf("room publishers = %d, want 1", stats.RoomsDetail[0].Publishers)
	}
	if stats.RoomsDetail[0].Listeners != 0 {
		t.Fatalf("room listeners = %d, want 0", stats.RoomsDetail[0].Listeners)
	}
	if stats.RoomsDetail[0].LastRealtimeEventType != "response.done" {
		t.Fatalf("room last realtime event = %q, want response.done", stats.RoomsDetail[0].LastRealtimeEventType)
	}
	if stats.RoomsDetail[0].LastRealtimeEventAt == "" {
		t.Fatal("room last realtime event at is empty")
	}
}

func TestServiceGetsSessionStatusFromRedisState(t *testing.T) {
	rooms := newMemoryRoomRepositoryForTest()
	runtime := newMemoryRoomRepositoryForTest()
	media := &fakeOfferAcceptor{}
	provider := &fakeRealtimeCallCreator{}
	states := &fakeMediaSessionStateRepository{}
	svc := NewService(rooms, runtime, states, media, provider)

	_, err := svc.AcceptOffer(context.Background(), sessiondto.AcceptOfferRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("AcceptOffer() error = %v", err)
	}

	status, found, err := svc.GetSessionStatus(context.Background(), sessiondto.GetSessionStatusRequest{
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("GetSessionStatus() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if status.SessionID != "session-1" {
		t.Fatalf("SessionID = %q, want session-1", status.SessionID)
	}
	if status.Status != string(vo.RoomStatusActive) {
		t.Fatalf("Status = %q, want %q", status.Status, vo.RoomStatusActive)
	}
	if status.LastActiveAt == "" {
		t.Fatal("LastActiveAt is empty")
	}
	if len(status.ParticipantStates) != 2 {
		t.Fatalf("ParticipantStates len = %d, want 2", len(status.ParticipantStates))
	}
	var foundPublisher bool
	for _, participant := range status.ParticipantStates {
		if participant.Role == string(vo.ParticipantRoleClient) && participant.AudioMode == string(sessiondto.AudioModePublisher) {
			foundPublisher = true
		}
	}
	if !foundPublisher {
		t.Fatalf("participant states = %#v, want client publisher", status.ParticipantStates)
	}
}
