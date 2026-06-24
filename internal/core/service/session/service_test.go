package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	"github.com/kyh0703/portfoilo-media/internal/core/port/portfakes"
	sessionquery "github.com/kyh0703/portfoilo-media/internal/core/query/session"
	"github.com/kyh0703/portfoilo-media/internal/core/query/session/sessionfakes"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func createSessionForTest(t *testing.T, svc Service) {
	t.Helper()
	_, err := svc.CreateSession(context.Background(), sessionio.CreateSessionRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
}

func providerEventPayload(eventType string) string {
	return "provider-event:" + eventType
}

type fakeProviderEventNormalizer struct {
	signal  port.ConversationSignal
	signals map[string]port.ConversationSignal
}

func (f fakeProviderEventNormalizer) Normalize(payload string) port.ConversationSignal {
	signal, ok := f.signals[payload]
	if !ok {
		signal = f.signal
	}
	if signal.Payload == "" {
		signal.Payload = payload
	}
	return signal
}

func TestServiceAcceptsOfferThroughSFU(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	res, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	if res.SDPAnswer != "answer-sdp" {
		t.Fatalf("SDPAnswer = %q, want %q", res.SDPAnswer, "answer-sdp")
	}
	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	if acceptedInput.Role != vo.ParticipantRoleClient {
		t.Fatalf("Role = %q, want %q", acceptedInput.Role, vo.ParticipantRoleClient)
	}
	if acceptedInput.SessionID != vo.SessionID("session-1") {
		t.Fatalf("SessionID = %q, want session-1", acceptedInput.SessionID)
	}
	if !acceptedInput.PublishAudio {
		t.Fatal("PublishAudio = false, want true")
	}
}

func TestServiceRejectsOfferWhenSessionWasNotCreated(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("JoinSession() error = %v, want %v", err, usecase.ErrSessionNotFound)
	}
	if media.AcceptOfferCallCount() != 0 {
		t.Fatalf("media AcceptOffer calls = %d, want 0", media.AcceptOfferCallCount())
	}
}

func TestServiceRejectsJoinWhenRoomIsClosedBeforeAcceptingMediaOffer(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	provider := &fakeRealtimeProvider{}
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	svc := NewService(records, runtime, states, media, provider)

	now := time.Date(2026, 6, 5, 10, 0, 0, 0, time.UTC)
	room := entity.NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	room.Close(now)
	if err := records.Save(context.Background(), entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		t.Fatalf("records Save() error = %v", err)
	}

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if !errors.Is(err, usecase.ErrSessionNotJoinable) {
		t.Fatalf("JoinSession() error = %v, want %v", err, usecase.ErrSessionNotJoinable)
	}
	if media.AcceptOfferCallCount() != 0 {
		t.Fatalf("media AcceptOffer calls = %d, want 0", media.AcceptOfferCallCount())
	}
}

func TestServiceCreatesRoomRuntimeForClientJoin(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	res, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
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
	if room.ParticipantCount() != 2 {
		t.Fatalf("participants len = %d, want 2", room.ParticipantCount())
	}

	participant, ok := room.Participants()[vo.ParticipantID(res.ParticipantID)]
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
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() first error = %v", err)
	}

	_, err = svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() second publisher error = %v", err)
	}
	if media.AcceptOfferCallCount() != 2 {
		t.Fatalf("media AcceptOffer calls = %d, want 2", media.AcceptOfferCallCount())
	}
	if provider.CreateCallCallCount() != 1 {
		t.Fatalf("provider CreateCall calls = %d, want 1", provider.CreateCallCallCount())
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}
	var publishers int
	for _, participant := range room.Participants() {
		if participant.Role == vo.ParticipantRoleClient && participant.PublishAudio {
			publishers++
		}
	}
	if publishers != 2 {
		t.Fatalf("audio publishers = %d, want 2", publishers)
	}
}

func TestServiceAllowsMultipleClientsWhenAdditionalClientIsListener(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "publisher-offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() publisher error = %v", err)
	}

	_, err = svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "listener-offer-sdp",
		AudioMode:      sessionio.AudioModeListener,
	})
	if err != nil {
		t.Fatalf("JoinSession() listener error = %v", err)
	}

	if media.AcceptOfferCallCount() != 2 {
		t.Fatalf("media AcceptOffer calls = %d, want 2", media.AcceptOfferCallCount())
	}
	_, acceptedInput0 := media.AcceptOfferArgsForCall(0)
	_, acceptedInput1 := media.AcceptOfferArgsForCall(1)
	if !acceptedInput0.PublishAudio {
		t.Fatal("publisher PublishAudio = false, want true")
	}
	if acceptedInput1.PublishAudio {
		t.Fatal("listener PublishAudio = true, want false")
	}
	if provider.CreateCallCallCount() != 1 {
		t.Fatalf("provider CreateCall calls = %d, want 1", provider.CreateCallCallCount())
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}
	if room.ParticipantCount() != 3 {
		t.Fatalf("participants len = %d, want 3", room.ParticipantCount())
	}

	var clients, publishers int
	for _, participant := range room.Participants() {
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
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	res, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	leave, err := svc.LeaveParticipant(context.Background(), sessionio.LeaveParticipantRequest{
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
	_, closedSession, closedParticipant := media.CloseParticipantArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	if closedParticipant != vo.ParticipantID(res.ParticipantID) {
		t.Fatalf("closed participant = %q, want %q", closedParticipant, res.ParticipantID)
	}
	if media.CloseSessionCallCount() != 0 {
		t.Fatalf("CloseSession calls = %d, want 0", media.CloseSessionCallCount())
	}
	if provider.HangupCallCallCount() != 0 {
		t.Fatalf("provider hangup calls = %d, want 0", provider.HangupCallCallCount())
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}
	if room.ParticipantCount() != 1 {
		t.Fatalf("participants len = %d, want 1", room.ParticipantCount())
	}
	for _, participant := range room.Participants() {
		if participant.Role != vo.ParticipantRoleOpenAIAgent {
			t.Fatalf("remaining role = %q, want %q", participant.Role, vo.ParticipantRoleOpenAIAgent)
		}
	}

	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("state status = %q, want active", state.Status)
	}
	if state.Participants != 1 {
		t.Fatalf("state participants = %d, want 1", state.Participants)
	}
}

func TestServiceLeavesCriticalParticipantByFailingRoom(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	leave, err := svc.LeaveParticipant(context.Background(), sessionio.LeaveParticipantRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		ParticipantID:  string(createdInput.ParticipantID),
	})
	if err != nil {
		t.Fatalf("LeaveParticipant() error = %v", err)
	}

	if leave.Status != string(vo.RoomStatusFailed) {
		t.Fatalf("leave status = %q, want failed", leave.Status)
	}
	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	if media.CloseSessionCallCount() != 1 {
		t.Fatalf("CloseSession calls = %d, want 1", media.CloseSessionCallCount())
	}
	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}

	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	} else if found {
		t.Fatal("runtime room found, want deleted")
	}

	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("state status = %q, want failed", state.Status)
	}
	if state.Participants != 2 {
		t.Fatalf("state participants = %d, want 2", state.Participants)
	}
}

func TestServiceLogsMonitoringLifecycleFields(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	core, observed := observer.New(zap.InfoLevel)
	svc := newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, noopProviderEventNormalizer{}, defaultRealtimeControlConfig(), zap.New(core))
	createSessionForTest(t, svc)

	res, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
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
	if joined["audio_mode"] != string(sessionio.AudioModePublisher) {
		t.Fatalf("audio_mode = %v, want publisher", joined["audio_mode"])
	}

	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
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

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		State:         vo.ConnectionStateFailed,
	})
	failedEntries := observed.FilterMessage("media_room_failed").All()
	if len(failedEntries) != 1 {
		t.Fatalf("room failed log entries = %d, want 1", len(failedEntries))
	}
	if failedEntries[0].Level != zapcore.ErrorLevel {
		t.Fatalf("room failed log level = %s, want error", failedEntries[0].Level)
	}
	failed := failedEntries[0].ContextMap()
	if failed["failure_reason"] != "critical_participant_connection_failed" {
		t.Fatalf("failure_reason = %v, want critical_participant_connection_failed", failed["failure_reason"])
	}
	if failed["participant_role"] != string(vo.ParticipantRoleOpenAIAgent) {
		t.Fatalf("participant_role = %v, want openai_agent", failed["participant_role"])
	}
}

func TestServiceLogsRuntimeEventStateSaveFailures(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.SaveReturnsOnCall(1, errors.New("state store unavailable"))
	observed, logs := observer.New(zap.InfoLevel)
	svc := newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, noopProviderEventNormalizer{}, defaultRealtimeControlConfig(), zap.New(observed))
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
		State:         vo.ConnectionStateConnected,
	})

	entries := logs.FilterMessage("media_session_state_save_failed").All()
	if len(entries) != 1 {
		t.Fatalf("state save failure logs = %d, want 1", len(entries))
	}
	if entries[0].Level != zapcore.ErrorLevel {
		t.Fatalf("state save failure log level = %s, want error", entries[0].Level)
	}
	if entries[0].ContextMap()["operation"] != "connection_state_change" {
		t.Fatalf("operation = %v, want connection_state_change", entries[0].ContextMap()["operation"])
	}
}

func TestServicePersistsRoomMetadataForClientJoin(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	record, found, err := records.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata record not found")
	}
	if record.ConversationID != vo.ConversationID("conversation-1") {
		t.Fatalf("ConversationID = %q, want conversation-1", record.ConversationID)
	}
}

func TestServiceConnectsOpenAIParticipantForClientJoin(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	if createdInput.Role != vo.ParticipantRoleOpenAIAgent {
		t.Fatalf("created role = %q, want %q", createdInput.Role, vo.ParticipantRoleOpenAIAgent)
	}
	if createdInput.DataChannelLabel != "oai-events" {
		t.Fatalf("data channel label = %q, want oai-events", createdInput.DataChannelLabel)
	}
	_, createCallInput := provider.CreateCallArgsForCall(0)
	if createCallInput.SDPOffer != "openai-offer-sdp" {
		t.Fatalf("provider SDPOffer = %q, want openai-offer-sdp", createCallInput.SDPOffer)
	}
	_, _, appliedAnswer := media.ApplyAnswerArgsForCall(0)
	if appliedAnswer != "openai-answer-sdp" {
		t.Fatalf("applied answer = %q, want openai-answer-sdp", appliedAnswer)
	}

	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("room not found")
	}

	var foundAgent bool
	for _, participant := range room.Participants() {
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

func TestServiceUsesDefaultOpenAIRealtimeOpeningDisclaimer(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDReturns(sessionquery.MediaSessionState{}, false, nil)
	svc := NewServiceWithOptions(records, runtime, states, media, provider, ServiceOptions{})
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	if len(createdInput.InitialDataMessages) != 1 {
		t.Fatalf("InitialDataMessages len = %d, want 1", len(createdInput.InitialDataMessages))
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(createdInput.InitialDataMessages[0]), &event); err != nil {
		t.Fatalf("decode default initial event: %v", err)
	}
	if event["type"] != "response.create" {
		t.Fatalf("event.type = %v, want response.create", event["type"])
	}
	response, ok := event["response"].(map[string]any)
	if !ok {
		t.Fatalf("event.response = %#v, want object", event["response"])
	}
	instructions, ok := response["instructions"].(string)
	if !ok {
		t.Fatalf("response.instructions = %#v, want string", response["instructions"])
	}
	for _, want := range []string{"AI 정서 지원 상담원", "의료 전문가", "119", "전문기관", "지금 어떤 마음"} {
		if !strings.Contains(instructions, want) {
			t.Fatalf("instructions missing %q: %q", want, instructions)
		}
	}
}

func TestServiceIgnoresBlankConfiguredRealtimeInitialEvents(t *testing.T) {
	realtime := realtimeControlConfigFromOptions(ServiceOptions{
		RealtimeInitialEvents: []string{" ", "\t"},
	})

	if len(realtime.initialEvents) != 1 {
		t.Fatalf("initialEvents len = %d, want default event", len(realtime.initialEvents))
	}
}

func TestServiceUsesConfiguredOpenAIRealtimeDataChannel(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewServiceWithOptions(records, runtime, states, media, provider, ServiceOptions{
		RealtimeDataChannelLabel: "custom-events",
		RealtimeInitialEvents: []string{
			`{"type":"session.update"}`,
			" ",
			`{"type":"response.create"}`,
		},
	})
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	if createdInput.DataChannelLabel != "custom-events" {
		t.Fatalf("DataChannelLabel = %q, want custom-events", createdInput.DataChannelLabel)
	}
	if len(createdInput.InitialDataMessages) != 2 {
		t.Fatalf("InitialDataMessages len = %d, want 2", len(createdInput.InitialDataMessages))
	}
	if createdInput.InitialDataMessages[0] != `{"type":"session.update"}` {
		t.Fatalf("InitialDataMessages[0] = %q", createdInput.InitialDataMessages[0])
	}
}

func TestServiceStoresRealtimeDataChannelEventInLiveState(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, fakeProviderEventNormalizer{
		signal: port.ConversationSignal{
			Type:              port.ConversationEventTypeAssistantRespCompleted,
			ProviderEventType: port.ProviderEventType("provider.response.completed"),
		},
	}, defaultRealtimeControlConfig(), zap.NewNop())
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       `{"type":"response.done"}`,
	})

	if states.SaveCallCount() != 2 {
		t.Fatalf("state saves = %d, want 2", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(1)
	if state.LastRealtimeEventType != "assistant.response.completed" {
		t.Fatalf("LastRealtimeEventType = %q, want assistant.response.completed", state.LastRealtimeEventType)
	}
	if state.LastRealtimeEventAt.IsZero() {
		t.Fatal("LastRealtimeEventAt is zero")
	}
	if len(state.RecentRealtimeEvents) != 1 {
		t.Fatalf("RecentRealtimeEvents len = %d, want 1", len(state.RecentRealtimeEvents))
	}
	if state.RecentRealtimeEvents[0].Type != "assistant.response.completed" {
		t.Fatalf("RecentRealtimeEvents[0].Type = %q, want assistant.response.completed", state.RecentRealtimeEvents[0].Type)
	}

	status, found, err := svc.GetSessionStatus(context.Background(), sessionio.GetSessionStatusRequest{
		SessionID: "session-1",
	})
	if err != nil {
		t.Fatalf("GetSessionStatus() error = %v", err)
	}
	if !found {
		t.Fatal("found = false, want true")
	}
	if status.LastRealtimeEventType != "assistant.response.completed" {
		t.Fatalf("status LastRealtimeEventType = %q, want assistant.response.completed", status.LastRealtimeEventType)
	}
	if status.LastRealtimeEventAt.IsZero() {
		t.Fatal("status LastRealtimeEventAt is zero")
	}
	if len(status.RecentRealtimeEvents) != 1 {
		t.Fatalf("status RecentRealtimeEvents len = %d, want 1", len(status.RecentRealtimeEvents))
	}
	if status.RecentRealtimeEvents[0].Type != "assistant.response.completed" {
		t.Fatalf("status RecentRealtimeEvents[0].Type = %q, want assistant.response.completed", status.RecentRealtimeEvents[0].Type)
	}
}

func TestServicePublishesAllowlistedConversationEvent(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	publisher := &portfakes.FakeConversationEventPublisher{}
	svc := NewServiceWithOptionsLoggerAndPublisher(
		records,
		runtime,
		states,
		media,
		provider,
		publisher,
		fakeProviderEventNormalizer{
			signal: port.ConversationSignal{
				Type:              port.ConversationEventTypeSpeechInputTranscriptCompleted,
				ProviderEventType: port.ProviderEventType("provider.input_transcript.completed"),
				ProviderEventID:   "evt_1",
				ProviderItemID:    "item-1",
				ProviderRespID:    "resp-1",
				Payload:           `{"text":"hello"}`,
				Publishable:       true,
			},
		},
		ServiceOptions{},
		zap.NewNop(),
	)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       `provider raw payload`,
	})

	if publisher.PublishCallCount() != 1 {
		t.Fatalf("published events = %d, want 1", publisher.PublishCallCount())
	}
	_, event := publisher.PublishArgsForCall(0)
	if event.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", event.SchemaVersion)
	}
	if event.EventID != "evt_1" {
		t.Fatalf("EventID = %q, want evt_1", event.EventID)
	}
	if event.ConversationID != vo.ConversationID("conversation-1") {
		t.Fatalf("ConversationID = %q, want conversation-1", event.ConversationID)
	}
	if event.SessionID != vo.SessionID("session-1") {
		t.Fatalf("SessionID = %q, want session-1", event.SessionID)
	}
	if event.RoomID == "" {
		t.Fatal("RoomID is empty")
	}
	if event.ProviderCallID != "rtc_123" {
		t.Fatalf("ProviderCallID = %q, want rtc_123", event.ProviderCallID)
	}
	if event.Type != "speech.input_transcript.completed" {
		t.Fatalf("Type = %q, want speech.input_transcript.completed", event.Type)
	}
	if event.ProviderEventType != "provider.input_transcript.completed" {
		t.Fatalf("ProviderEventType = %q", event.ProviderEventType)
	}
	if event.ProviderItemID != "item-1" {
		t.Fatalf("ProviderItemID = %q, want item-1", event.ProviderItemID)
	}
	if event.PreviousProviderItemID != "" {
		t.Fatalf("PreviousProviderItemID = %q, want empty", event.PreviousProviderItemID)
	}
	if event.ProviderRespID != "resp-1" {
		t.Fatalf("ProviderRespID = %q, want resp-1", event.ProviderRespID)
	}
	if event.OccurredAt.IsZero() {
		t.Fatal("OccurredAt is zero")
	}

	if event.Payload != `{"text":"hello"}` {
		t.Fatalf("Payload = %q, want normalized payload", event.Payload)
	}
}

func TestServicePublishesConversationEventsWithProviderItemLinksAsReceived(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	publisher := &portfakes.FakeConversationEventPublisher{}
	secondPayload := `provider raw second`
	firstPayload := `provider raw first`
	svc := NewServiceWithOptionsLoggerAndPublisher(
		records,
		runtime,
		states,
		media,
		provider,
		publisher,
		fakeProviderEventNormalizer{
			signals: map[string]port.ConversationSignal{
				secondPayload: {
					Type:                   port.ConversationEventTypeSpeechInputTranscriptCompleted,
					ProviderEventType:      port.ProviderEventType("provider.input_transcript.completed"),
					ProviderEventID:        "evt_2",
					ProviderItemID:         "item-2",
					PreviousProviderItemID: "item-1",
					Payload:                secondPayload,
					Publishable:            true,
				},
				firstPayload: {
					Type:              port.ConversationEventTypeSpeechInputTranscriptCompleted,
					ProviderEventType: port.ProviderEventType("provider.input_transcript.completed"),
					ProviderEventID:   "evt_1",
					ProviderItemID:    "item-1",
					Payload:           firstPayload,
					Publishable:       true,
				},
			},
		},
		ServiceOptions{},
		zap.NewNop(),
	)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       secondPayload,
	})
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       firstPayload,
	})

	if publisher.PublishCallCount() != 2 {
		t.Fatalf("published events = %d, want 2", publisher.PublishCallCount())
	}
	_, first := publisher.PublishArgsForCall(0)
	_, second := publisher.PublishArgsForCall(1)
	if first.ProviderItemID != "item-2" || first.PreviousProviderItemID != "item-1" {
		t.Fatalf("first provider item link = %q <- %q, want item-2 <- item-1", first.ProviderItemID, first.PreviousProviderItemID)
	}
	if second.ProviderItemID != "item-1" || second.PreviousProviderItemID != "" {
		t.Fatalf("second provider item link = %q <- %q, want item-1 <- empty", second.ProviderItemID, second.PreviousProviderItemID)
	}
}

func TestServiceBuildsConversationEventFromInjectedProviderNormalizer(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	publisher := &portfakes.FakeConversationEventPublisher{}
	eventsIn := fakeProviderEventNormalizer{
		signal: port.ConversationSignal{
			Type:              port.ConversationEventTypeSpeechInputTranscriptCompleted,
			ProviderEventType: port.ProviderEventType("stt.segment.completed"),
			ProviderEventID:   "stt-event-1",
			ProviderItemID:    "stt-item-1",
			Payload:           `{"text":"hello from stt"}`,
			Publishable:       true,
		},
	}
	svc := NewServiceWithOptionsLoggerAndPublisher(
		records,
		runtime,
		states,
		media,
		provider,
		publisher,
		eventsIn,
		ServiceOptions{},
		zap.NewNop(),
	)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       `stt-provider-raw-payload`,
	})

	if publisher.PublishCallCount() != 1 {
		t.Fatalf("published events = %d, want 1", publisher.PublishCallCount())
	}
	_, event := publisher.PublishArgsForCall(0)
	if event.ProviderEventType != port.ProviderEventType("stt.segment.completed") {
		t.Fatalf("ProviderEventType = %q, want stt.segment.completed", event.ProviderEventType)
	}
	if event.Type != port.ConversationEventTypeSpeechInputTranscriptCompleted {
		t.Fatalf("Type = %q, want %s", event.Type, port.ConversationEventTypeSpeechInputTranscriptCompleted)
	}
	if event.ProviderItemID != "stt-item-1" {
		t.Fatalf("ProviderItemID = %q, want stt-item-1", event.ProviderItemID)
	}
}

func TestServiceIgnoresNonAllowlistedConversationEvent(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	publisher := &portfakes.FakeConversationEventPublisher{}
	svc := NewServiceWithOptionsLoggerAndPublisher(
		records,
		runtime,
		states,
		media,
		provider,
		publisher,
		fakeProviderEventNormalizer{
			signal: port.ConversationSignal{
				Type:              port.ConversationEventTypeSessionProviderCreated,
				ProviderEventType: port.ProviderEventType("provider.session.created"),
				Publishable:       false,
			},
		},
		ServiceOptions{},
		zap.NewNop(),
	)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       providerEventPayload("session.created"),
	})

	if publisher.PublishCallCount() != 0 {
		t.Fatalf("published events = %d, want 0", publisher.PublishCallCount())
	}
	if states.SaveCallCount() != 2 {
		t.Fatalf("state saves = %d, want 2", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(1)
	if state.LastRealtimeEventType != string(port.ConversationEventTypeSessionProviderCreated) {
		t.Fatalf("LastRealtimeEventType = %q, want %s", state.LastRealtimeEventType, port.ConversationEventTypeSessionProviderCreated)
	}
}

func TestServiceFallbackConversationEventIDIsStable(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	publisher := &portfakes.FakeConversationEventPublisher{}
	payload := `provider output text completed`
	svc := NewServiceWithOptionsLoggerAndPublisher(
		records,
		runtime,
		states,
		media,
		provider,
		publisher,
		fakeProviderEventNormalizer{
			signal: port.ConversationSignal{
				Type:              port.ConversationEventTypeAssistantOutputTextCompleted,
				ProviderEventType: port.ProviderEventType("provider.output_text.completed"),
				Payload:           payload,
				Publishable:       true,
			},
		},
		ServiceOptions{},
		zap.NewNop(),
	)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	message := port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       payload,
	}
	svc.(*service).HandleDataChannelMessage(context.Background(), message)
	svc.(*service).HandleDataChannelMessage(context.Background(), message)

	if publisher.PublishCallCount() != 2 {
		t.Fatalf("published events = %d, want 2", publisher.PublishCallCount())
	}
	_, firstEvent := publisher.PublishArgsForCall(0)
	_, secondEvent := publisher.PublishArgsForCall(1)
	if firstEvent.EventID == "" {
		t.Fatal("fallback EventID is empty")
	}
	if firstEvent.EventID != secondEvent.EventID {
		t.Fatalf("fallback EventID changed: %q != %q", firstEvent.EventID, secondEvent.EventID)
	}
}

func TestServiceLogsPublishErrorAndKeepsLiveStateUpdate(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	publisher := &portfakes.FakeConversationEventPublisher{}
	publisher.PublishReturns(errors.New("publish failed"))
	observed, logs := observer.New(zap.WarnLevel)
	svc := NewServiceWithOptionsLoggerAndPublisher(
		records,
		runtime,
		states,
		media,
		provider,
		publisher,
		fakeProviderEventNormalizer{
			signal: port.ConversationSignal{
				Type:              port.ConversationEventTypeAssistantOutputTextCompleted,
				ProviderEventType: port.ProviderEventType("provider.output_text.completed"),
				Payload:           `{"text":"hello"}`,
				Publishable:       true,
			},
		},
		ServiceOptions{},
		zap.New(observed),
	)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
		Payload:       `provider output text completed`,
	})

	if publisher.PublishCallCount() != 1 {
		t.Fatalf("published events = %d, want 1 attempt", publisher.PublishCallCount())
	}
	if states.SaveCallCount() != 2 {
		t.Fatalf("state saves = %d, want 2", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(1)
	if state.LastRealtimeEventType != string(port.ConversationEventTypeAssistantOutputTextCompleted) {
		t.Fatalf("LastRealtimeEventType = %q, want %s", state.LastRealtimeEventType, port.ConversationEventTypeAssistantOutputTextCompleted)
	}
	if logs.FilterMessage("media_conversation_event_publish_failed").Len() != 1 {
		t.Fatalf("publish failure logs = %d, want 1", logs.FilterMessage("media_conversation_event_publish_failed").Len())
	}
}

func TestServiceLimitsRecentRealtimeEvents(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, fakeProviderEventNormalizer{
		signals: map[string]port.ConversationSignal{
			providerEventPayload("session.created"): {
				Type:              port.ConversationEventTypeSessionProviderCreated,
				ProviderEventType: port.ProviderEventType("provider.session.created"),
			},
			providerEventPayload("response.created"): {
				Type:              port.ConversationEventTypeAssistantRespCreated,
				ProviderEventType: port.ProviderEventType("provider.response.created"),
			},
			providerEventPayload("response.done"): {
				Type:              port.ConversationEventTypeAssistantRespCompleted,
				ProviderEventType: port.ProviderEventType("provider.response.completed"),
			},
		},
	}, realtimeControlConfigFromOptions(ServiceOptions{RealtimeEventHistoryLimit: 2}), zap.NewNop())
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	for _, eventType := range []string{
		"session.created",
		"response.created",
		"response.done",
	} {
		svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
			SessionID:     createdInput.SessionID,
			ParticipantID: createdInput.ParticipantID,
			Role:          createdInput.Role,
			Label:         createdInput.DataChannelLabel,
			Payload:       providerEventPayload(eventType),
		})
	}

	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if len(state.RecentRealtimeEvents) != 2 {
		t.Fatalf("RecentRealtimeEvents len = %d, want 2", len(state.RecentRealtimeEvents))
	}
	if state.RecentRealtimeEvents[0].Type != string(port.ConversationEventTypeAssistantRespCreated) {
		t.Fatalf("RecentRealtimeEvents[0].Type = %q, want %s", state.RecentRealtimeEvents[0].Type, port.ConversationEventTypeAssistantRespCreated)
	}
	if state.RecentRealtimeEvents[1].Type != string(port.ConversationEventTypeAssistantRespCompleted) {
		t.Fatalf("RecentRealtimeEvents[1].Type = %q, want %s", state.RecentRealtimeEvents[1].Type, port.ConversationEventTypeAssistantRespCompleted)
	}
}

func TestServiceStoresActiveMediaSessionStateForClientJoin(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	if states.SaveCallCount() != 1 {
		t.Fatalf("state saves = %d, want 1", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(0)
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
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{}, errors.New("openai unavailable"))
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err == nil {
		t.Fatal("JoinSession() error = nil, want error")
	}

	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	if provider.HangupCallCallCount() != 0 {
		t.Fatalf("hangup calls = %#v, want none because provider call was not created", provider.Invocations())
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
	if states.SaveCallCount() != 1 {
		t.Fatalf("state saves = %d, want 1", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(0)
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusFailed)
	}
	if state.ConnectionState != vo.ConnectionStateFailed {
		t.Fatalf("ConnectionState = %q, want %q", state.ConnectionState, vo.ConnectionStateFailed)
	}
	if state.MediaState != vo.TrackStateFailed {
		t.Fatalf("MediaState = %q, want %q", state.MediaState, vo.TrackStateFailed)
	}

	record, found, err := records.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata record not found")
	}
	if record.Status != vo.RoomStatusFailed {
		t.Fatalf("record status = %q, want %q", record.Status, vo.RoomStatusFailed)
	}
}

func TestServiceCleansUpJoinResourcesWhenActiveStateSaveFails(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.SaveReturns(errors.New("state store unavailable"))
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err == nil {
		t.Fatal("JoinSession() error = nil, want state save error")
	}

	_, closedSession := media.CloseSessionArgsForCall(0)
	if media.CloseSessionCallCount() != 1 || closedSession != vo.SessionID("session-1") {
		t.Fatalf("CloseSession calls = %#v, want session-1", media.closeSessionArgs)
	}
	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
	record, found, err := records.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata record not found")
	}
	if record.Status != vo.RoomStatusFailed {
		t.Fatalf("record status = %q, want %q", record.Status, vo.RoomStatusFailed)
	}
	if states.SaveCallCount() != 2 {
		t.Fatalf("state saves = %d, want active attempt and failed cleanup attempt", states.SaveCallCount())
	}
	_, failedState := states.SaveArgsForCall(1)
	if failedState.Status != vo.RoomStatusFailed {
		t.Fatalf("cleanup state status = %q, want %q", failedState.Status, vo.RoomStatusFailed)
	}
}

func TestServiceHangsUpOpenAICallWhenApplyAnswerFails(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(nil, errors.New("invalid openai answer"))
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err == nil {
		t.Fatal("JoinSession() error = nil, want error")
	}

	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}
	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	if states.SaveCallCount() != 1 {
		t.Fatalf("state saves = %d, want 1", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(0)
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusFailed)
	}
}

func TestServiceStoresMediaSessionStateWhenTrackStateChanges(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	svc.(*service).HandleMediaTrackStateChange(context.Background(), port.MediaTrackStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateActive,
	})

	if states.SaveCallCount() != 2 {
		t.Fatalf("state saves = %d, want 2", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(1)
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
	participant := room.Participants()[acceptedInput.ParticipantID]
	for _, track := range participant.Tracks {
		if track.Kind == vo.TrackKindAudio && track.State != vo.TrackStateActive {
			t.Fatalf("audio track state = %q, want %q", track.State, vo.TrackStateActive)
		}
	}
}

func TestServiceKeepsRoomActiveWhenClientConnectionFails(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
		State:         vo.ConnectionStateFailed,
	})

	if media.CloseSessionCallCount() != 0 {
		t.Fatalf("CloseSession calls = %d, want 0", media.CloseSessionCallCount())
	}
	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("runtime room not found")
	}
	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusActive)
	}
	if state.ConnectionState == vo.ConnectionStateFailed {
		t.Fatalf("ConnectionState = %q, want non-failed", state.ConnectionState)
	}
	participant := room.Participants()[acceptedInput.ParticipantID]
	if participant.State != vo.ConnectionStateFailed {
		t.Fatalf("client participant state = %q, want failed", participant.State)
	}
}

func TestServiceKeepsRoomActiveWhenClientMediaTrackFails(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	svc.(*service).HandleMediaTrackStateChange(context.Background(), port.MediaTrackStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateFailed,
	})

	if media.CloseSessionCallCount() != 0 {
		t.Fatalf("CloseSession calls = %d, want 0", media.CloseSessionCallCount())
	}
	room, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("runtime room not found")
	}
	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if state.Status != vo.RoomStatusActive {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusActive)
	}
	if state.MediaState == vo.TrackStateFailed {
		t.Fatalf("MediaState = %q, want non-failed", state.MediaState)
	}
	participant := room.Participants()[acceptedInput.ParticipantID]
	for _, track := range participant.Tracks {
		if track.Kind == vo.TrackKindAudio && track.State != vo.TrackStateFailed {
			t.Fatalf("client audio track state = %q, want failed", track.State)
		}
	}
}

func TestServiceFailsSessionWhenOpenAIConnectionFails(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		State:         vo.ConnectionStateFailed,
	})

	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusFailed)
	}
	if state.ConnectionState != vo.ConnectionStateFailed {
		t.Fatalf("ConnectionState = %q, want %q", state.ConnectionState, vo.ConnectionStateFailed)
	}
}

func TestServiceFailsSessionWhenOpenAIMediaTrackFails(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleMediaTrackStateChange(context.Background(), port.MediaTrackStateChange{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateFailed,
	})

	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}
	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if state.Status != vo.RoomStatusFailed {
		t.Fatalf("Status = %q, want %q", state.Status, vo.RoomStatusFailed)
	}
	if state.MediaState != vo.TrackStateFailed {
		t.Fatalf("MediaState = %q, want %q", state.MediaState, vo.TrackStateFailed)
	}
}

func TestServiceStoresMediaSessionStateWhenConnectionStateChanges(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
		State:         vo.ConnectionStateConnected,
	})
	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		State:         vo.ConnectionStateConnected,
	})

	if states.SaveCallCount() != 3 {
		t.Fatalf("state saves = %d, want 3", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(2)
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
	participant := room.Participants()[acceptedInput.ParticipantID]
	if participant.State != vo.ConnectionStateConnected {
		t.Fatalf("participant state = %q, want %q", participant.State, vo.ConnectionStateConnected)
	}
}

func TestServiceEndsSessionCleanup(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	res, err := svc.EndSession(context.Background(), sessionio.EndSessionRequest{
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
	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}

	record, found, err := records.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("record FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata record not found")
	}
	if record.Status != vo.RoomStatusClosed {
		t.Fatalf("record status = %q, want %q", record.Status, vo.RoomStatusClosed)
	}
}

func TestServiceStoresClosedStateWhenCleanupFailsDuringEndSession(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	provider.HangupCallReturns(errors.New("provider hangup unavailable"))
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, err = svc.EndSession(context.Background(), sessionio.EndSessionRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
	})
	if err == nil {
		t.Fatal("EndSession() error = nil, want cleanup error")
	}

	record, found, err := records.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata record not found")
	}
	if record.Status != vo.RoomStatusClosed {
		t.Fatalf("record status = %q, want %q", record.Status, vo.RoomStatusClosed)
	}
	_, state := states.SaveArgsForCall(states.SaveCallCount() - 1)
	if state.Status != vo.RoomStatusClosed {
		t.Fatalf("state status = %q, want %q", state.Status, vo.RoomStatusClosed)
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
}

func TestServiceCleansUpIdleRuntimeRooms(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	svcImpl := svc.(*service)

	oldNow := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	currentNow := oldNow.Add(3 * time.Minute)
	svcImpl.now = func() time.Time { return currentNow }

	room := entity.NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), oldNow)
	room.SetUserID("user-1", oldNow)
	client := entity.NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleClient, oldNow)
	client.SetState(vo.ConnectionStateConnected, oldNow)
	client.AddTrack(entity.NewTrack(vo.TrackID("track-1"), vo.TrackKindAudio, oldNow), oldNow)
	if err := room.JoinClient(client, oldNow); err != nil {
		t.Fatalf("JoinClient() error = %v", err)
	}
	agent := entity.NewParticipant(vo.ParticipantID("agent-1"), vo.ParticipantRoleOpenAIAgent, oldNow)
	agent.SetState(vo.ConnectionStateConnected, oldNow)
	agent.SetProviderCallID("rtc_123", oldNow)
	agent.AddTrack(entity.NewTrack(vo.TrackID("track-2"), vo.TrackKindAudio, oldNow), oldNow)
	if err := room.AttachOpenAIAgent(agent, oldNow); err != nil {
		t.Fatalf("AttachOpenAIAgent() error = %v", err)
	}
	room.UpdatedAt = oldNow

	if err := runtime.Save(context.Background(), room); err != nil {
		t.Fatalf("runtime Save() error = %v", err)
	}
	if err := records.Save(context.Background(), entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		t.Fatalf("records Save() error = %v", err)
	}

	cleaned, err := svc.CleanupIdleRooms(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("CleanupIdleRooms() error = %v", err)
	}

	if cleaned != 1 {
		t.Fatalf("cleaned = %d, want 1", cleaned)
	}
	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}

	record, found, err := records.FindBySessionID(context.Background(), vo.SessionID("session-1"))
	if err != nil {
		t.Fatalf("record FindBySessionID() error = %v", err)
	}
	if !found {
		t.Fatal("metadata record not found")
	}
	if record.Status != vo.RoomStatusClosed {
		t.Fatalf("record status = %q, want %q", record.Status, vo.RoomStatusClosed)
	}
	_, state := states.SaveArgsForCall(0)
	if states.SaveCallCount() != 1 || state.Status != vo.RoomStatusClosed {
		t.Fatalf("states = %#v, want one closed state", states.Invocations())
	}
}

func TestServiceKeepsRecentlyUpdatedRuntimeRooms(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
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
	if media.CloseSessionCallCount() != 0 {
		t.Fatalf("media close calls = %d, want 0", media.CloseSessionCallCount())
	}
}

func TestServiceShutdownClosesActiveRuntimeRooms(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	svcImpl := svc.(*service)

	now := time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC)
	svcImpl.now = func() time.Time { return now }

	room := entity.NewRoom(vo.RoomID("room-1"), vo.SessionID("session-1"), vo.ConversationID("conversation-1"), now)
	room.SetUserID("user-1", now)
	client := entity.NewParticipant(vo.ParticipantID("client-1"), vo.ParticipantRoleClient, now)
	client.SetState(vo.ConnectionStateConnected, now)
	client.AddTrack(entity.NewTrack(vo.TrackID("track-1"), vo.TrackKindAudio, now), now)
	if err := room.JoinClient(client, now); err != nil {
		t.Fatalf("JoinClient() error = %v", err)
	}
	agent := entity.NewParticipant(vo.ParticipantID("agent-1"), vo.ParticipantRoleOpenAIAgent, now)
	agent.SetState(vo.ConnectionStateConnected, now)
	agent.SetProviderCallID("rtc_123", now)
	agent.AddTrack(entity.NewTrack(vo.TrackID("track-2"), vo.TrackKindAudio, now), now)
	if err := room.AttachOpenAIAgent(agent, now); err != nil {
		t.Fatalf("AttachOpenAIAgent() error = %v", err)
	}

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
	_, closedSession := media.CloseSessionArgsForCall(0)
	if closedSession != vo.SessionID("session-1") {
		t.Fatalf("closed session = %q, want session-1", closedSession)
	}
	_, hangupProviderCallID := provider.HangupCallArgsForCall(0)
	if provider.HangupCallCallCount() != 1 || hangupProviderCallID != "rtc_123" {
		t.Fatalf("hangup calls = %#v, want [rtc_123]", provider.Invocations())
	}
	if _, found, err := runtime.FindBySessionID(context.Background(), vo.SessionID("session-1")); err != nil || found {
		t.Fatalf("runtime found=%v err=%v, want not found", found, err)
	}
	_, state := states.SaveArgsForCall(0)
	if states.SaveCallCount() != 1 || state.Status != vo.RoomStatusClosed {
		t.Fatalf("states = %#v, want one closed state", states.Invocations())
	}
}

func TestServiceStoresClosedMediaSessionStateWhenSessionEnds(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	_, err = svc.EndSession(context.Background(), sessionio.EndSessionRequest{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
	})
	if err != nil {
		t.Fatalf("EndSession() error = %v", err)
	}

	if states.SaveCallCount() != 2 {
		t.Fatalf("state saves = %d, want 2", states.SaveCallCount())
	}
	_, state := states.SaveArgsForCall(1)
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
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, fakeProviderEventNormalizer{
		signal: port.ConversationSignal{
			Type:              port.ConversationEventTypeAssistantRespCompleted,
			ProviderEventType: port.ProviderEventType("provider.response.completed"),
		},
	}, defaultRealtimeControlConfig(), zap.NewNop())
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}
	_, acceptedInput := media.AcceptOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
		State:         vo.ConnectionStateConnected,
	})
	_, createdInput := media.CreateOfferArgsForCall(0)
	svc.(*service).HandleConnectionStateChange(context.Background(), port.ConnectionStateChange{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		State:         vo.ConnectionStateConnected,
	})
	svc.(*service).HandleMediaTrackStateChange(context.Background(), port.MediaTrackStateChange{
		SessionID:     acceptedInput.SessionID,
		ParticipantID: acceptedInput.ParticipantID,
		Role:          acceptedInput.Role,
		Kind:          vo.TrackKindAudio,
		State:         vo.TrackStateActive,
	})
	svc.(*service).HandleDataChannelMessage(context.Background(), port.DataChannelMessage{
		SessionID:     createdInput.SessionID,
		ParticipantID: createdInput.ParticipantID,
		Role:          createdInput.Role,
		Label:         createdInput.DataChannelLabel,
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
	if stats.ByAudioMode[string(sessionio.AudioModePublisher)] != 1 {
		t.Fatalf("publisher count = %d, want 1", stats.ByAudioMode[string(sessionio.AudioModePublisher)])
	}
	if stats.ByRealtimeEvent["assistant.response.completed"] != 1 {
		t.Fatalf("assistant.response.completed count = %d, want 1", stats.ByRealtimeEvent["assistant.response.completed"])
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
	if stats.RoomsDetail[0].LastRealtimeEventType != "assistant.response.completed" {
		t.Fatalf("room last realtime event = %q, want assistant.response.completed", stats.RoomsDetail[0].LastRealtimeEventType)
	}
	if stats.RoomsDetail[0].LastRealtimeEventAt == "" {
		t.Fatal("room last realtime event at is empty")
	}
}

func TestServiceGetsSessionStatusFromRedisState(t *testing.T) {
	records := newMemoryMediaSessionRecordRepositoryForTest()
	runtime := newMemoryRoomRuntimeRepositoryForTest()
	media := &fakeMediaGateway{}
	media.AcceptOfferReturns(&port.Peer{AnswerSDP: "answer-sdp"}, nil)
	media.CreateOfferReturns(&port.PeerOffer{SDPOffer: "openai-offer-sdp"}, nil)
	media.ApplyAnswerReturns(&port.Peer{}, nil)
	provider := &fakeRealtimeProvider{}
	provider.CreateCallReturns(port.CreateCallResult{SDPAnswer: "openai-answer-sdp", ProviderCallID: "rtc_123"}, nil)
	states := &sessionfakes.FakeMediaSessionStateRepository{}
	states.FindBySessionIDCalls(func(ctx context.Context, sessionID vo.SessionID) (sessionquery.MediaSessionState, bool, error) {
		_ = ctx
		for i := states.SaveCallCount() - 1; i >= 0; i-- {
			_, state := states.SaveArgsForCall(i)
			if state.SessionID == sessionID {
				return state, true, nil
			}
		}
		return sessionquery.MediaSessionState{}, false, nil
	})
	svc := NewService(records, runtime, states, media, provider)
	createSessionForTest(t, svc)

	_, err := svc.JoinSession(context.Background(), sessionio.JoinSessionCommand{
		SessionID:      "session-1",
		ConversationID: "conversation-1",
		UserID:         "user-1",
		SDP:            "offer-sdp",
	})
	if err != nil {
		t.Fatalf("JoinSession() error = %v", err)
	}

	status, found, err := svc.GetSessionStatus(context.Background(), sessionio.GetSessionStatusRequest{
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
	if status.LastActiveAt.IsZero() {
		t.Fatal("LastActiveAt is zero")
	}
	if len(status.ParticipantStates) != 2 {
		t.Fatalf("ParticipantStates len = %d, want 2", len(status.ParticipantStates))
	}
	var foundPublisher bool
	for _, participant := range status.ParticipantStates {
		if participant.Role == string(vo.ParticipantRoleClient) && participant.AudioMode == string(sessionio.AudioModePublisher) {
			foundPublisher = true
		}
	}
	if !foundPublisher {
		t.Fatalf("participant states = %#v, want client publisher", status.ParticipantStates)
	}
}

type fakeMediaGateway struct {
	acceptOfferReturns struct {
		peer *port.Peer
		err  error
	}
	acceptOfferArgs []struct {
		ctx   context.Context
		input port.OfferInput
	}
	createOfferReturns struct {
		offer *port.PeerOffer
		err   error
	}
	createOfferArgs []struct {
		ctx   context.Context
		input port.CreateOfferInput
	}
	applyAnswerReturns struct {
		peer *port.Peer
		err  error
	}
	applyAnswerArgs []struct {
		ctx       context.Context
		offer     *port.PeerOffer
		answerSDP string
	}
	closeSessionArgs []struct {
		ctx       context.Context
		sessionID vo.SessionID
	}
	closeSessionReturns struct {
		err error
	}
	closeParticipantArgs []struct {
		ctx           context.Context
		sessionID     vo.SessionID
		participantID vo.ParticipantID
	}
}

func (f *fakeMediaGateway) AcceptOffer(ctx context.Context, input port.OfferInput) (*port.Peer, error) {
	f.acceptOfferArgs = append(f.acceptOfferArgs, struct {
		ctx   context.Context
		input port.OfferInput
	}{ctx: ctx, input: input})
	return f.acceptOfferReturns.peer, f.acceptOfferReturns.err
}

func (f *fakeMediaGateway) AcceptOfferReturns(peer *port.Peer, err error) {
	f.acceptOfferReturns.peer = peer
	f.acceptOfferReturns.err = err
}

func (f *fakeMediaGateway) AcceptOfferCallCount() int {
	return len(f.acceptOfferArgs)
}

func (f *fakeMediaGateway) AcceptOfferArgsForCall(i int) (context.Context, port.OfferInput) {
	arg := f.acceptOfferArgs[i]
	return arg.ctx, arg.input
}

func (f *fakeMediaGateway) CreateOffer(ctx context.Context, input port.CreateOfferInput) (*port.PeerOffer, error) {
	f.createOfferArgs = append(f.createOfferArgs, struct {
		ctx   context.Context
		input port.CreateOfferInput
	}{ctx: ctx, input: input})
	return f.createOfferReturns.offer, f.createOfferReturns.err
}

func (f *fakeMediaGateway) CreateOfferReturns(offer *port.PeerOffer, err error) {
	f.createOfferReturns.offer = offer
	f.createOfferReturns.err = err
}

func (f *fakeMediaGateway) CreateOfferArgsForCall(i int) (context.Context, port.CreateOfferInput) {
	arg := f.createOfferArgs[i]
	return arg.ctx, arg.input
}

func (f *fakeMediaGateway) ApplyAnswer(ctx context.Context, offer *port.PeerOffer, answerSDP string) (*port.Peer, error) {
	f.applyAnswerArgs = append(f.applyAnswerArgs, struct {
		ctx       context.Context
		offer     *port.PeerOffer
		answerSDP string
	}{ctx: ctx, offer: offer, answerSDP: answerSDP})
	return f.applyAnswerReturns.peer, f.applyAnswerReturns.err
}

func (f *fakeMediaGateway) ApplyAnswerReturns(peer *port.Peer, err error) {
	f.applyAnswerReturns.peer = peer
	f.applyAnswerReturns.err = err
}

func (f *fakeMediaGateway) ApplyAnswerArgsForCall(i int) (context.Context, *port.PeerOffer, string) {
	arg := f.applyAnswerArgs[i]
	return arg.ctx, arg.offer, arg.answerSDP
}

func (f *fakeMediaGateway) CloseSession(ctx context.Context, sessionID vo.SessionID) error {
	f.closeSessionArgs = append(f.closeSessionArgs, struct {
		ctx       context.Context
		sessionID vo.SessionID
	}{ctx: ctx, sessionID: sessionID})
	return f.closeSessionReturns.err
}

func (f *fakeMediaGateway) CloseSessionReturns(err error) {
	f.closeSessionReturns.err = err
}

func (f *fakeMediaGateway) CloseSessionCallCount() int {
	return len(f.closeSessionArgs)
}

func (f *fakeMediaGateway) CloseSessionArgsForCall(i int) (context.Context, vo.SessionID) {
	arg := f.closeSessionArgs[i]
	return arg.ctx, arg.sessionID
}

func (f *fakeMediaGateway) CloseParticipant(ctx context.Context, sessionID vo.SessionID, participantID vo.ParticipantID) error {
	f.closeParticipantArgs = append(f.closeParticipantArgs, struct {
		ctx           context.Context
		sessionID     vo.SessionID
		participantID vo.ParticipantID
	}{ctx: ctx, sessionID: sessionID, participantID: participantID})
	return nil
}

func (f *fakeMediaGateway) CloseParticipantArgsForCall(i int) (context.Context, vo.SessionID, vo.ParticipantID) {
	arg := f.closeParticipantArgs[i]
	return arg.ctx, arg.sessionID, arg.participantID
}

type fakeRealtimeProvider struct {
	createCallReturns struct {
		result port.CreateCallResult
		err    error
	}
	createCallArgs []struct {
		ctx   context.Context
		input port.CreateCallInput
	}
	hangupCallArgs []struct {
		ctx            context.Context
		providerCallID string
	}
	hangupCallReturns struct {
		err error
	}
}

func (f *fakeRealtimeProvider) CreateCall(ctx context.Context, input port.CreateCallInput) (port.CreateCallResult, error) {
	f.createCallArgs = append(f.createCallArgs, struct {
		ctx   context.Context
		input port.CreateCallInput
	}{ctx: ctx, input: input})
	return f.createCallReturns.result, f.createCallReturns.err
}

func (f *fakeRealtimeProvider) CreateCallReturns(result port.CreateCallResult, err error) {
	f.createCallReturns.result = result
	f.createCallReturns.err = err
}

func (f *fakeRealtimeProvider) CreateCallCallCount() int {
	return len(f.createCallArgs)
}

func (f *fakeRealtimeProvider) CreateCallArgsForCall(i int) (context.Context, port.CreateCallInput) {
	arg := f.createCallArgs[i]
	return arg.ctx, arg.input
}

func (f *fakeRealtimeProvider) HangupCall(ctx context.Context, providerCallID string) error {
	f.hangupCallArgs = append(f.hangupCallArgs, struct {
		ctx            context.Context
		providerCallID string
	}{ctx: ctx, providerCallID: providerCallID})
	return f.hangupCallReturns.err
}

func (f *fakeRealtimeProvider) HangupCallReturns(err error) {
	f.hangupCallReturns.err = err
}

func (f *fakeRealtimeProvider) HangupCallCallCount() int {
	return len(f.hangupCallArgs)
}

func (f *fakeRealtimeProvider) HangupCallArgsForCall(i int) (context.Context, string) {
	arg := f.hangupCallArgs[i]
	return arg.ctx, arg.providerCallID
}

func (f *fakeRealtimeProvider) Invocations() map[string][][]interface{} {
	invocations := make(map[string][][]interface{})
	for _, arg := range f.createCallArgs {
		invocations["CreateCall"] = append(invocations["CreateCall"], []interface{}{arg.ctx, arg.input})
	}
	for _, arg := range f.hangupCallArgs {
		invocations["HangupCall"] = append(invocations["HangupCall"], []interface{}{arg.ctx, arg.providerCallID})
	}
	return invocations
}
