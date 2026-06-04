package session

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"github.com/kyh0703/portfoilo-media/internal/pkg/openai"
	rtc "github.com/kyh0703/portfoilo-media/internal/pkg/webrtc"
	"go.uber.org/zap"
)

//go:generate go tool counterfeiter -generate
//counterfeiter:generate . Service

type Service interface {
	CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error)
	AcceptOffer(ctx context.Context, req sessiondto.AcceptOfferRequest) (sessiondto.AcceptOfferResponse, error)
	LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error)
	EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error)
	CleanupIdleRooms(ctx context.Context, idleTimeout time.Duration) (int, error)
	ShutdownActiveRooms(ctx context.Context) (int, error)
	GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error)
	GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error)
	GetHealth(ctx context.Context) error
}

type service struct {
	rooms    repository.RoomRepository
	runtime  repository.RoomRuntimeRepository
	states   repository.MediaSessionStateRepository
	media    rtc.PeerConnectionFactory
	provider openai.RealtimeCallManager
	events   repository.ConversationEventPublisher
	log      *zap.Logger
	now      func() time.Time
	locks    sync.Map
	realtime realtimeControlConfig
}

type realtimeControlConfig struct {
	dataChannelLabel          string
	initialEvents             []string
	realtimeEventHistoryLimit int
}

func NewService(
	rooms repository.RoomRepository,
	runtime repository.RoomRuntimeRepository,
	states repository.MediaSessionStateRepository,
	media rtc.PeerConnectionFactory,
	provider openai.RealtimeCallManager,
) Service {
	return newService(rooms, runtime, states, media, provider, noopConversationEventPublisher{}, defaultRealtimeControlConfig(), zap.NewNop())
}

func NewServiceWithConfig(
	rooms repository.RoomRepository,
	runtime repository.RoomRuntimeRepository,
	states repository.MediaSessionStateRepository,
	media rtc.PeerConnectionFactory,
	provider openai.RealtimeCallManager,
	cfg *configs.Config,
) Service {
	return newService(rooms, runtime, states, media, provider, noopConversationEventPublisher{}, realtimeControlConfigFromConfig(cfg), zap.NewNop())
}

func NewServiceWithConfigAndLogger(
	rooms repository.RoomRepository,
	runtime repository.RoomRuntimeRepository,
	states repository.MediaSessionStateRepository,
	media rtc.PeerConnectionFactory,
	provider openai.RealtimeCallManager,
	cfg *configs.Config,
	log *zap.Logger,
) Service {
	return newService(rooms, runtime, states, media, provider, noopConversationEventPublisher{}, realtimeControlConfigFromConfig(cfg), log)
}

func NewServiceWithConfigLoggerAndPublisher(
	rooms repository.RoomRepository,
	runtime repository.RoomRuntimeRepository,
	states repository.MediaSessionStateRepository,
	media rtc.PeerConnectionFactory,
	provider openai.RealtimeCallManager,
	events repository.ConversationEventPublisher,
	cfg *configs.Config,
	log *zap.Logger,
) Service {
	return newService(rooms, runtime, states, media, provider, events, realtimeControlConfigFromConfig(cfg), log)
}

func newService(
	rooms repository.RoomRepository,
	runtime repository.RoomRuntimeRepository,
	states repository.MediaSessionStateRepository,
	media rtc.PeerConnectionFactory,
	provider openai.RealtimeCallManager,
	events repository.ConversationEventPublisher,
	realtime realtimeControlConfig,
	log *zap.Logger,
) Service {
	if log == nil {
		log = zap.NewNop()
	}
	if events == nil {
		events = noopConversationEventPublisher{}
	}
	return &service{
		rooms:    rooms,
		runtime:  runtime,
		states:   states,
		media:    media,
		provider: provider,
		events:   events,
		log:      log,
		now:      time.Now,
		realtime: realtime,
	}
}

func realtimeControlConfigFromConfig(cfg *configs.Config) realtimeControlConfig {
	realtime := defaultRealtimeControlConfig()
	if cfg == nil {
		return realtime
	}
	if label := strings.TrimSpace(cfg.OpenAI.RealtimeDataChannelLabel); label != "" {
		realtime.dataChannelLabel = label
	}
	realtime.initialEvents = compactRealtimeInitialEvents(cfg.OpenAI.RealtimeInitialEvents)
	if cfg.Realtime.RealtimeEventHistoryLimit > 0 {
		realtime.realtimeEventHistoryLimit = cfg.Realtime.RealtimeEventHistoryLimit
	}
	return realtime
}

func defaultRealtimeControlConfig() realtimeControlConfig {
	return realtimeControlConfig{
		dataChannelLabel:          "oai-events",
		realtimeEventHistoryLimit: 10,
	}
}

func compactRealtimeInitialEvents(events []string) []string {
	compacted := make([]string, 0, len(events))
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		compacted = append(compacted, event)
	}
	return compacted
}

func (s *service) CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	if sessionID == "" {
		sessionID = vo.NewSessionID()
	}

	conversationID := vo.ConversationID(req.ConversationID)
	roomID := vo.NewRoomID()
	room := entity.NewRoom(roomID, sessionID, conversationID, now)

	if err := s.rooms.Save(ctx, room); err != nil {
		return sessiondto.CreateSessionResponse{}, err
	}

	return sessiondto.CreateSessionResponse{
		SessionID:      string(sessionID),
		ConversationID: string(conversationID),
		RoomID:         string(roomID),
		Status:         string(room.Status),
	}, nil
}

func (s *service) AcceptOffer(ctx context.Context, req sessiondto.AcceptOfferRequest) (sessiondto.AcceptOfferResponse, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	unlock := s.lockSession(sessionID)
	defer unlock()

	room, err := s.findSessionRoom(ctx, sessionID)
	if err != nil {
		return sessiondto.AcceptOfferResponse{}, err
	}
	publishAudio := req.PublishesAudio()
	room.SetUserID(req.UserID, now)

	participant, peer, err := s.acceptClientParticipant(ctx, sessionID, req.SDP, publishAudio, now)
	if err != nil {
		return sessiondto.AcceptOfferResponse{}, err
	}
	room.AddParticipant(participant, now)

	if err := s.ensureOpenAIParticipant(ctx, &room, sessionID, req.UserID, now); err != nil {
		return sessiondto.AcceptOfferResponse{}, err
	}

	if err := s.saveActiveRoomState(ctx, room, req.UserID, now); err != nil {
		return sessiondto.AcceptOfferResponse{}, err
	}

	s.logParticipantEvent("media_participant_joined", room, participant,
		zap.String("audio_mode", participantAudioMode(participant)),
		zap.Int("participants", len(room.Participants)),
	)
	for _, joined := range room.Participants {
		if joined.Role == vo.ParticipantRoleOpenAIAgent {
			s.logParticipantEvent("media_participant_ready", room, joined,
				zap.String("provider_call_id", joined.ProviderCallID),
			)
			break
		}
	}

	return sessiondto.AcceptOfferResponse{
		SDPAnswer:     peer.AnswerSDP,
		RoomID:        string(room.ID),
		ParticipantID: string(participant.ID),
	}, nil
}

func (s *service) findSessionRoom(ctx context.Context, sessionID vo.SessionID) (entity.Room, error) {
	room, found, err := s.runtime.FindBySessionID(ctx, sessionID)
	if err != nil {
		return entity.Room{}, err
	}
	if found {
		return room, nil
	}

	room, found, err = s.rooms.FindBySessionID(ctx, sessionID)
	if err != nil {
		return entity.Room{}, err
	}
	if !found {
		return entity.Room{}, usecase.ErrSessionNotFound
	}
	return room, nil
}

func (s *service) acceptClientParticipant(
	ctx context.Context,
	sessionID vo.SessionID,
	sdp string,
	publishAudio bool,
	now time.Time,
) (entity.Participant, *rtc.Peer, error) {
	participantID := vo.NewParticipantID()
	peer, err := s.media.AcceptOffer(ctx, rtc.OfferInput{
		SessionID:               sessionID,
		ParticipantID:           participantID,
		Role:                    vo.ParticipantRoleClient,
		SDP:                     sdp,
		PublishAudio:            publishAudio,
		OnConnectionStateChange: s.handleConnectionStateChange,
		OnMediaTrackStateChange: s.handleMediaTrackStateChange,
	})
	if err != nil {
		return entity.Participant{}, nil, err
	}

	return newClientParticipant(participantID, publishAudio, now), peer, nil
}

func newClientParticipant(participantID vo.ParticipantID, publishAudio bool, now time.Time) entity.Participant {
	participant := entity.NewParticipant(participantID, vo.ParticipantRoleClient, now)
	participant.SetState(vo.ConnectionStateConnecting, now)
	participant.SetPublishAudio(publishAudio, now)
	participant.AddTrack(entity.NewTrack(vo.NewTrackID(), vo.TrackKindAudio, now), now)
	return participant
}

func (s *service) ensureOpenAIParticipant(
	ctx context.Context,
	room *entity.Room,
	sessionID vo.SessionID,
	userID string,
	now time.Time,
) error {
	if room.HasParticipantRole(vo.ParticipantRoleOpenAIAgent) {
		return nil
	}

	openAIParticipant, err := s.connectOpenAIParticipant(ctx, sessionID, now)
	if err != nil {
		_ = s.failRoom(ctx, *room, userID, now, "openai_setup_failed")
		return err
	}
	room.AddParticipant(openAIParticipant, now)
	return nil
}

func (s *service) saveActiveRoomState(ctx context.Context, room entity.Room, userID string, now time.Time) error {
	if err := s.runtime.Save(ctx, room); err != nil {
		return err
	}
	if err := s.rooms.Save(ctx, room); err != nil {
		return err
	}
	if err := s.states.Save(ctx, s.mediaSessionState(room, userID, now)); err != nil {
		return err
	}
	return nil
}

func (s *service) lockSession(sessionID vo.SessionID) func() {
	lock, _ := s.locks.LoadOrStore(sessionID, &sync.Mutex{})
	sessionLock := lock.(*sync.Mutex)
	sessionLock.Lock()
	return func() {
		sessionLock.Unlock()
	}
}

func (s *service) connectOpenAIParticipant(ctx context.Context, sessionID vo.SessionID, now time.Time) (entity.Participant, error) {
	participantID := vo.NewParticipantID()
	offer, err := s.media.CreateOffer(ctx, rtc.CreateOfferInput{
		SessionID:               sessionID,
		ParticipantID:           participantID,
		Role:                    vo.ParticipantRoleOpenAIAgent,
		DataChannelLabel:        s.realtime.dataChannelLabel,
		InitialDataMessages:     s.realtime.initialEvents,
		OnConnectionStateChange: s.handleConnectionStateChange,
		OnMediaTrackStateChange: s.handleMediaTrackStateChange,
		OnDataChannelMessage:    s.handleOpenAIDataChannelMessage,
	})
	if err != nil {
		return entity.Participant{}, err
	}

	call, err := s.provider.CreateCall(ctx, openai.CreateCallInput{
		SDPOffer: offer.SDPOffer,
	})
	if err != nil {
		return entity.Participant{}, err
	}

	if _, err := s.media.ApplyAnswer(ctx, offer, call.SDPAnswer); err != nil {
		_ = s.provider.HangupCall(ctx, call.ProviderCallID)
		return entity.Participant{}, err
	}

	participant := entity.NewParticipant(participantID, vo.ParticipantRoleOpenAIAgent, now)
	participant.SetState(vo.ConnectionStateConnecting, now)
	participant.SetProviderCallID(call.ProviderCallID, now)
	participant.AddTrack(entity.NewTrack(vo.NewTrackID(), vo.TrackKindAudio, now), now)
	return participant, nil
}

func (s *service) handleOpenAIDataChannelMessage(message rtc.DataChannelMessage) {
	ctx := context.Background()
	now := s.now().UTC()
	unlock := s.lockSession(message.SessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, message.SessionID)
	if err != nil || !found {
		return
	}
	eventType := realtimeEventType(message.Payload)
	room.RecordRealtimeEvent(eventType, now)

	if err := s.runtime.Save(ctx, room); err != nil {
		return
	}
	if err := s.rooms.Save(ctx, room); err != nil {
		return
	}
	existingState, _, _ := s.states.FindBySessionID(ctx, message.SessionID)
	_ = s.states.Save(ctx, s.mediaSessionStateWithRealtimeEvent(
		room,
		room.UserID,
		now,
		eventType,
		existingState.RecentRealtimeEvents,
	))
	s.publishConversationEvent(ctx, room, eventType, message.Payload, now)
	s.logRoomEvent("media_realtime_event_recorded", room,
		zap.String("participant_id", string(message.ParticipantID)),
		zap.String("participant_role", string(message.Role)),
		zap.String("data_channel_label", message.Label),
		zap.String("realtime_event_type", eventType),
	)
}

func (s *service) LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	participantID := vo.ParticipantID(req.ParticipantID)
	unlock := s.lockSession(sessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, sessionID)
	if err != nil {
		return sessiondto.LeaveParticipantResponse{}, err
	}
	if !found {
		return sessiondto.LeaveParticipantResponse{}, nil
	}

	participant, found := room.Participant(participantID)
	if !found {
		return sessiondto.LeaveParticipantResponse{
			SessionID:     string(sessionID),
			RoomID:        string(room.ID),
			ParticipantID: string(participantID),
			Status:        string(room.Status),
		}, nil
	}
	if isCriticalParticipant(participant.Role) {
		_ = s.failRoom(ctx, room, req.UserID, now, "critical_participant_left",
			zap.String("participant_id", string(participant.ID)),
			zap.String("participant_role", string(participant.Role)),
		)
		return sessiondto.LeaveParticipantResponse{
			SessionID:     string(sessionID),
			RoomID:        string(room.ID),
			ParticipantID: string(participantID),
			Status:        string(vo.RoomStatusFailed),
		}, nil
	}

	room.RemoveParticipant(participantID, now)
	if err := s.media.CloseParticipant(ctx, sessionID, participantID); err != nil {
		return sessiondto.LeaveParticipantResponse{}, err
	}
	if err := s.runtime.Save(ctx, room); err != nil {
		return sessiondto.LeaveParticipantResponse{}, err
	}
	if err := s.rooms.Save(ctx, room); err != nil {
		return sessiondto.LeaveParticipantResponse{}, err
	}
	if err := s.states.Save(ctx, s.mediaSessionState(room, req.UserID, now)); err != nil {
		return sessiondto.LeaveParticipantResponse{}, err
	}

	s.logParticipantEvent("media_participant_left", room, participant,
		zap.String("audio_mode", participantAudioMode(participant)),
		zap.Int("participants", len(room.Participants)),
	)

	return sessiondto.LeaveParticipantResponse{
		SessionID:     string(sessionID),
		RoomID:        string(room.ID),
		ParticipantID: string(participantID),
		Status:        string(room.Status),
	}, nil
}

func (s *service) EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	unlock := s.lockSession(sessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, sessionID)
	if err != nil {
		return sessiondto.EndSessionResponse{}, err
	}
	if !found {
		room, found, err = s.rooms.FindBySessionID(ctx, sessionID)
		if err != nil {
			return sessiondto.EndSessionResponse{}, err
		}
		if !found {
			return sessiondto.EndSessionResponse{}, nil
		}
	}

	if err := s.closeRoom(ctx, room, req.UserID, now, "explicit_end"); err != nil {
		return sessiondto.EndSessionResponse{}, err
	}

	return sessiondto.EndSessionResponse{
		SessionID: string(sessionID),
		RoomID:    string(room.ID),
		Status:    string(vo.RoomStatusClosed),
	}, nil
}

func (s *service) CleanupIdleRooms(ctx context.Context, idleTimeout time.Duration) (int, error) {
	if idleTimeout <= 0 {
		return 0, nil
	}

	now := s.now().UTC()
	rooms, err := s.runtime.List(ctx)
	if err != nil {
		return 0, err
	}

	var cleaned int
	for _, room := range rooms {
		if room.Status == vo.RoomStatusClosed || room.Status == vo.RoomStatusFailed {
			continue
		}
		if now.Sub(room.UpdatedAt) < idleTimeout {
			continue
		}
		if err := s.closeRoom(ctx, room, room.UserID, now, "idle_timeout"); err != nil {
			return cleaned, err
		}
		cleaned++
	}

	return cleaned, nil
}

func (s *service) ShutdownActiveRooms(ctx context.Context) (int, error) {
	now := s.now().UTC()
	rooms, err := s.runtime.List(ctx)
	if err != nil {
		return 0, err
	}

	var cleaned int
	for _, room := range rooms {
		if room.Status == vo.RoomStatusClosed || room.Status == vo.RoomStatusFailed {
			continue
		}
		if err := s.closeRoom(ctx, room, room.UserID, now, "shutdown"); err != nil {
			return cleaned, err
		}
		cleaned++
	}

	return cleaned, nil
}

func (s *service) GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error) {
	state, found, err := s.states.FindBySessionID(ctx, vo.SessionID(req.SessionID))
	if err != nil || !found {
		return sessiondto.GetSessionStatusResponse{}, found, err
	}

	return sessiondto.GetSessionStatusResponse{
		SessionID:             string(state.SessionID),
		ConversationID:        string(state.ConversationID),
		UserID:                state.UserID,
		RoomID:                string(state.RoomID),
		Status:                string(state.Status),
		ConnectionState:       string(state.ConnectionState),
		MediaState:            string(state.MediaState),
		OpenAIProviderCallID:  state.OpenAIProviderCallID,
		Participants:          state.Participants,
		ParticipantStates:     participantStateResponses(state.ParticipantStates),
		LastRealtimeEventType: state.LastRealtimeEventType,
		LastRealtimeEventAt:   formatOptionalTime(state.LastRealtimeEventAt),
		RecentRealtimeEvents:  realtimeEventResponses(state.RecentRealtimeEvents),
		StartedAt:             state.StartedAt.Format(time.RFC3339Nano),
		LastActiveAt:          state.UpdatedAt.Format(time.RFC3339Nano),
	}, true, nil
}

func (s *service) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	rooms, err := s.runtime.List(ctx)
	if err != nil {
		return sessiondto.RuntimeStatsResponse{}, err
	}

	stats := sessiondto.RuntimeStatsResponse{
		Rooms:           len(rooms),
		Sessions:        len(rooms),
		ByStatus:        make(map[string]int),
		ByConnection:    make(map[string]int),
		ByMedia:         make(map[string]int),
		ByRole:          make(map[string]int),
		ByAudioMode:     make(map[string]int),
		ByRealtimeEvent: make(map[string]int),
		RoomsDetail:     make([]sessiondto.RuntimeRoomStatDetail, 0, len(rooms)),
	}

	for _, room := range rooms {
		connectionState := roomConnectionState(room)
		mediaState := roomMediaState(room)
		trackCount := countTracks(room)

		stats.Participants += len(room.Participants)
		stats.Tracks += trackCount
		stats.ByStatus[string(room.Status)]++
		stats.ByConnection[string(connectionState)]++
		stats.ByMedia[string(mediaState)]++
		if room.LastRealtimeEventType != "" {
			stats.ByRealtimeEvent[room.LastRealtimeEventType]++
		}
		publishers, listeners := countClientAudioModes(room)
		for _, participant := range room.Participants {
			stats.ByRole[string(participant.Role)]++
			if participant.Role == vo.ParticipantRoleClient {
				stats.ByAudioMode[participantAudioMode(participant)]++
			}
		}
		stats.RoomsDetail = append(stats.RoomsDetail, sessiondto.RuntimeRoomStatDetail{
			RoomID:                string(room.ID),
			SessionID:             string(room.SessionID),
			ConversationID:        string(room.ConversationID),
			Status:                string(room.Status),
			ConnectionState:       string(connectionState),
			MediaState:            string(mediaState),
			Participants:          len(room.Participants),
			Publishers:            publishers,
			Listeners:             listeners,
			LastRealtimeEventType: room.LastRealtimeEventType,
			LastRealtimeEventAt:   formatOptionalTime(room.LastRealtimeEventAt),
			Tracks:                trackCount,
		})
	}

	return stats, nil
}

func (s *service) GetHealth(ctx context.Context) error {
	_ = ctx
	return nil
}

func (s *service) handleConnectionStateChange(change rtc.ConnectionStateChange) {
	ctx := context.Background()
	now := s.now().UTC()
	unlock := s.lockSession(change.SessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, change.SessionID)
	if err != nil || !found {
		return
	}
	if !room.UpdateParticipantState(change.ParticipantID, change.State, now) {
		return
	}

	if err := s.runtime.Save(ctx, room); err != nil {
		return
	}
	if err := s.rooms.Save(ctx, room); err != nil {
		return
	}
	s.logRoomEvent("media_participant_connection_state_changed", room,
		zap.String("participant_id", string(change.ParticipantID)),
		zap.String("participant_role", string(change.Role)),
		zap.String("connection_state", string(change.State)),
	)
	if change.State == vo.ConnectionStateFailed && isCriticalParticipant(change.Role) {
		_ = s.failRoom(ctx, room, room.UserID, now, "critical_participant_connection_failed",
			zap.String("participant_id", string(change.ParticipantID)),
			zap.String("participant_role", string(change.Role)),
			zap.String("connection_state", string(change.State)),
		)
		return
	}
	_ = s.states.Save(ctx, s.mediaSessionState(room, room.UserID, now))
}

func (s *service) handleMediaTrackStateChange(change rtc.MediaTrackStateChange) {
	ctx := context.Background()
	now := s.now().UTC()
	unlock := s.lockSession(change.SessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, change.SessionID)
	if err != nil || !found {
		return
	}
	if !room.UpdateParticipantTrackState(change.ParticipantID, change.Kind, change.State, now) {
		return
	}

	if err := s.runtime.Save(ctx, room); err != nil {
		return
	}
	if err := s.rooms.Save(ctx, room); err != nil {
		return
	}
	s.logRoomEvent("media_track_state_changed", room,
		zap.String("participant_id", string(change.ParticipantID)),
		zap.String("participant_role", string(change.Role)),
		zap.String("track_type", string(change.Kind)),
		zap.String("media_state", string(change.State)),
	)
	if change.State == vo.TrackStateFailed && isCriticalParticipant(change.Role) {
		_ = s.failRoom(ctx, room, room.UserID, now, "critical_participant_track_failed",
			zap.String("participant_id", string(change.ParticipantID)),
			zap.String("participant_role", string(change.Role)),
			zap.String("track_type", string(change.Kind)),
			zap.String("media_state", string(change.State)),
		)
		return
	}
	_ = s.states.Save(ctx, s.mediaSessionState(room, room.UserID, now))
}

func (s *service) failRoom(ctx context.Context, room entity.Room, userID string, now time.Time, reason string, fields ...zap.Field) error {
	room.Fail(now)
	_ = s.hangupOpenAIParticipants(ctx, room)
	_ = s.media.CloseSession(ctx, room.SessionID)
	_ = s.runtime.Delete(ctx, room.ID)
	if err := s.rooms.Save(ctx, room); err != nil {
		return err
	}
	if err := s.states.Save(ctx, s.mediaSessionState(room, userID, now)); err != nil {
		return err
	}
	logFields := append([]zap.Field{zap.String("failure_reason", reason)}, fields...)
	s.logRoomEvent("media_room_failed", room, logFields...)
	return nil
}

func (s *service) closeRoom(ctx context.Context, room entity.Room, userID string, now time.Time, reason string) error {
	if err := s.hangupOpenAIParticipants(ctx, room); err != nil {
		return err
	}
	if err := s.media.CloseSession(ctx, room.SessionID); err != nil {
		return err
	}

	room.Close(now)
	if err := s.runtime.Delete(ctx, room.ID); err != nil {
		return err
	}
	if err := s.rooms.Save(ctx, room); err != nil {
		return err
	}
	if err := s.states.Save(ctx, s.mediaSessionState(room, userID, now)); err != nil {
		return err
	}
	s.logRoomEvent("media_room_closed", room,
		zap.String("close_reason", reason),
		zap.Int("participants", len(room.Participants)),
	)
	return nil
}

func (s *service) hangupOpenAIParticipants(ctx context.Context, room entity.Room) error {
	for _, participant := range room.Participants {
		if participant.Role == vo.ParticipantRoleOpenAIAgent && participant.ProviderCallID != "" {
			if err := s.provider.HangupCall(ctx, participant.ProviderCallID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) logParticipantEvent(event string, room entity.Room, participant entity.Participant, fields ...zap.Field) {
	logFields := append(roomLogFields(room), participantLogFields(participant)...)
	logFields = append(logFields, fields...)
	s.log.Info(event, logFields...)
}

func (s *service) logRoomEvent(event string, room entity.Room, fields ...zap.Field) {
	logFields := append(roomLogFields(room), fields...)
	s.log.Info(event, logFields...)
}

func roomLogFields(room entity.Room) []zap.Field {
	return []zap.Field{
		zap.String("session_id", string(room.SessionID)),
		zap.String("conversation_id", string(room.ConversationID)),
		zap.String("room_id", string(room.ID)),
		zap.String("room_status", string(room.Status)),
	}
}

func participantLogFields(participant entity.Participant) []zap.Field {
	return []zap.Field{
		zap.String("participant_id", string(participant.ID)),
		zap.String("participant_role", string(participant.Role)),
		zap.String("connection_state", string(participant.State)),
	}
}

func (s *service) mediaSessionState(room entity.Room, userID string, now time.Time) entity.MediaSessionState {
	return entity.MediaSessionState{
		SessionID:            room.SessionID,
		ConversationID:       room.ConversationID,
		UserID:               coalesceUserID(userID, room.UserID),
		RoomID:               room.ID,
		Status:               room.Status,
		ConnectionState:      roomConnectionState(room),
		MediaState:           roomMediaState(room),
		OpenAIProviderCallID: openAIProviderCallID(room),
		Participants:         len(room.Participants),
		ParticipantStates:    mediaSessionParticipantStates(room),
		StartedAt:            room.CreatedAt,
		UpdatedAt:            now,
	}
}

func (s *service) mediaSessionStateWithRealtimeEvent(
	room entity.Room,
	userID string,
	now time.Time,
	eventType string,
	recentEvents []entity.RealtimeEvent,
) entity.MediaSessionState {
	state := s.mediaSessionState(room, userID, now)
	state.LastRealtimeEventType = eventType
	state.LastRealtimeEventAt = now
	state.RecentRealtimeEvents = appendRealtimeEvent(recentEvents, entity.RealtimeEvent{
		Type: eventType,
		At:   now,
	}, s.realtime.realtimeEventHistoryLimit)
	return state
}

func realtimeEventType(payload string) string {
	eventType, _ := realtimeEventEnvelopeFields(payload)
	if eventType == "" {
		return "unknown"
	}
	return eventType
}

func realtimeEventEnvelopeFields(payload string) (string, string) {
	var event struct {
		Type    string `json:"type"`
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return "", ""
	}
	if strings.TrimSpace(event.Type) == "" {
		return "", strings.TrimSpace(event.EventID)
	}
	return strings.TrimSpace(event.Type), strings.TrimSpace(event.EventID)
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func appendRealtimeEvent(
	events []entity.RealtimeEvent,
	event entity.RealtimeEvent,
	limit int,
) []entity.RealtimeEvent {
	if limit <= 0 {
		return nil
	}
	next := append(append([]entity.RealtimeEvent(nil), events...), event)
	if len(next) <= limit {
		return next
	}
	return next[len(next)-limit:]
}

func realtimeEventResponses(events []entity.RealtimeEvent) []sessiondto.RealtimeEventResponse {
	responses := make([]sessiondto.RealtimeEventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, sessiondto.RealtimeEventResponse{
			Type: event.Type,
			At:   formatOptionalTime(event.At),
		})
	}
	return responses
}

func openAIProviderCallID(room entity.Room) string {
	for _, participant := range room.Participants {
		if participant.Role == vo.ParticipantRoleOpenAIAgent {
			return participant.ProviderCallID
		}
	}
	return ""
}

func (s *service) publishConversationEvent(
	ctx context.Context,
	room entity.Room,
	eventType string,
	payload string,
	occurredAt time.Time,
) {
	if !isPublishableConversationEvent(eventType) {
		return
	}

	eventID := realtimeEventID(payload)
	sanitizedPayload := sanitizeRealtimePayload(payload)
	providerCallID := openAIProviderCallID(room)
	if eventID == "" {
		eventID = fallbackConversationEventID(room.SessionID, providerCallID, eventType, sanitizedPayload)
	}

	event := entity.ConversationEvent{
		SchemaVersion:     1,
		EventID:           eventID,
		ConversationID:    room.ConversationID,
		SessionID:         room.SessionID,
		RoomID:            room.ID,
		ProviderCallID:    providerCallID,
		ProviderEventType: eventType,
		OccurredAt:        occurredAt,
		Payload:           sanitizedPayload,
	}
	if err := s.events.Publish(ctx, event); err != nil {
		s.log.Warn("media_conversation_event_publish_failed",
			zap.Error(err),
			zap.String("session_id", string(event.SessionID)),
			zap.String("conversation_id", string(event.ConversationID)),
			zap.String("room_id", string(event.RoomID)),
			zap.String("provider_call_id", event.ProviderCallID),
			zap.String("provider_event_type", event.ProviderEventType),
		)
	}
}

func realtimeEventID(payload string) string {
	_, eventID := realtimeEventEnvelopeFields(payload)
	return eventID
}

func fallbackConversationEventID(
	sessionID vo.SessionID,
	providerCallID string,
	eventType string,
	payload string,
) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		string(sessionID),
		providerCallID,
		eventType,
		payload,
	}, "\x00")))
	return fmt.Sprintf("media-%x", sum[:])
}

func sanitizeRealtimePayload(payload string) string {
	var decoded any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return payload
	}
	sanitized := sanitizeRealtimeValue(decoded)
	body, err := json.Marshal(sanitized)
	if err != nil {
		return payload
	}
	return string(body)
}

func sanitizeRealtimeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSecretPayloadField(key) {
				continue
			}
			out[key] = sanitizeRealtimeValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeRealtimeValue(item))
		}
		return out
	default:
		return typed
	}
}

func isSecretPayloadField(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "api_key", "authorization", "token", "secret", "client_secret":
		return true
	default:
		return false
	}
}

func isPublishableConversationEvent(eventType string) bool {
	_, ok := conversationEventAllowlist[eventType]
	return ok
}

var conversationEventAllowlist = map[string]struct{}{
	"conversation.item.created":                             {},
	"conversation.item.input_audio_transcription.completed": {},
	"conversation.item.input_audio_transcription.failed":    {},
	"response.output_audio_transcript.done":                 {},
	"response.output_text.done":                             {},
	"response.function_call_arguments.done":                 {},
	"response.mcp_call_arguments.done":                      {},
	"response.mcp_call.completed":                           {},
	"response.mcp_call.failed":                              {},
	"mcp_list_tools.completed":                              {},
	"mcp_list_tools.failed":                                 {},
	"response.output_item.done":                             {},
	"error":                                                 {},
}

type noopConversationEventPublisher struct{}

func (noopConversationEventPublisher) Publish(ctx context.Context, event entity.ConversationEvent) error {
	_ = ctx
	_ = event
	return nil
}

func coalesceUserID(userID string, fallback string) string {
	if userID != "" {
		return userID
	}
	return fallback
}

func isCriticalParticipant(role vo.ParticipantRole) bool {
	return role == vo.ParticipantRoleOpenAIAgent
}

func roomConnectionState(room entity.Room) vo.ConnectionState {
	if room.Status == vo.RoomStatusFailed {
		return vo.ConnectionStateFailed
	}
	if room.Status == vo.RoomStatusClosed {
		return vo.ConnectionStateClosed
	}
	if len(room.Participants) == 0 {
		return vo.ConnectionStateNew
	}

	hasConnected := false
	hasConnecting := false
	hasDisconnected := false
	for _, participant := range room.Participants {
		switch participant.State {
		case vo.ConnectionStateFailed:
			if isCriticalParticipant(participant.Role) {
				return vo.ConnectionStateFailed
			}
			hasDisconnected = true
		case vo.ConnectionStateDisconnected:
			hasDisconnected = true
		case vo.ConnectionStateConnected:
			hasConnected = true
		case vo.ConnectionStateConnecting, vo.ConnectionStateNew:
			hasConnecting = true
		default:
			hasConnecting = true
		}
	}
	if hasConnected {
		return vo.ConnectionStateConnected
	}
	if hasConnecting {
		return vo.ConnectionStateConnecting
	}
	if hasDisconnected {
		return vo.ConnectionStateDisconnected
	}
	return vo.ConnectionStateNew
}

func roomMediaState(room entity.Room) vo.TrackState {
	if room.Status == vo.RoomStatusFailed {
		return vo.TrackStateFailed
	}
	if room.Status == vo.RoomStatusClosed {
		return vo.TrackStateEnded
	}

	hasTrack := false
	hasPending := false
	hasFailed := false
	hasEnded := false
	for _, participant := range room.Participants {
		for _, track := range participant.Tracks {
			if track.Kind != vo.TrackKindAudio {
				continue
			}
			hasTrack = true
			switch track.State {
			case vo.TrackStateFailed:
				if isCriticalParticipant(participant.Role) {
					return vo.TrackStateFailed
				}
				hasFailed = true
			case vo.TrackStateActive:
				return vo.TrackStateActive
			case vo.TrackStatePending:
				hasPending = true
			case vo.TrackStateEnded:
				hasEnded = true
			default:
				hasPending = true
			}
		}
	}
	if !hasTrack || hasPending {
		return vo.TrackStatePending
	}
	if hasFailed {
		return vo.TrackStateFailed
	}
	if hasEnded {
		return vo.TrackStateEnded
	}
	return vo.TrackStatePending
}

func countTracks(room entity.Room) int {
	var count int
	for _, participant := range room.Participants {
		count += len(participant.Tracks)
	}
	return count
}

func mediaSessionParticipantStates(room entity.Room) []entity.MediaSessionParticipantState {
	states := make([]entity.MediaSessionParticipantState, 0, len(room.Participants))
	for _, participant := range room.Participants {
		states = append(states, entity.MediaSessionParticipantState{
			ID:              participant.ID,
			Role:            participant.Role,
			AudioMode:       participantAudioMode(participant),
			ConnectionState: participant.State,
			Tracks:          len(participant.Tracks),
		})
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].ID < states[j].ID
	})
	return states
}

func participantStateResponses(states []entity.MediaSessionParticipantState) []sessiondto.ParticipantStateResponse {
	responses := make([]sessiondto.ParticipantStateResponse, 0, len(states))
	for _, state := range states {
		responses = append(responses, sessiondto.ParticipantStateResponse{
			ID:              string(state.ID),
			Role:            string(state.Role),
			AudioMode:       state.AudioMode,
			ConnectionState: string(state.ConnectionState),
			Tracks:          state.Tracks,
		})
	}
	return responses
}

func countClientAudioModes(room entity.Room) (publishers int, listeners int) {
	for _, participant := range room.Participants {
		if participant.Role != vo.ParticipantRoleClient {
			continue
		}
		if participant.PublishAudio {
			publishers++
			continue
		}
		listeners++
	}
	return publishers, listeners
}

func participantAudioMode(participant entity.Participant) string {
	if participant.Role != vo.ParticipantRoleClient {
		return ""
	}
	if participant.PublishAudio {
		return string(sessiondto.AudioModePublisher)
	}
	return string(sessiondto.AudioModeListener)
}
