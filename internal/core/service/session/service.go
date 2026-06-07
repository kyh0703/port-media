package session

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	sessionquery "github.com/kyh0703/portfoilo-media/internal/core/query/session"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"go.uber.org/zap"
)

//go:generate go tool counterfeiter -generate
//counterfeiter:generate . Service

type Service interface {
	usecase.CreateSessionUsecase
	usecase.JoinSessionUsecase
	usecase.LeaveParticipantUsecase
	usecase.EndSessionUsecase
	usecase.GetSessionStatusQuery
	usecase.GetRuntimeStatsQuery
	usecase.RoomMaintenanceUsecase
	usecase.GetHealthQuery
}

type service struct {
	records   repository.MediaSessionRecordRepository
	runtime   repository.RoomRuntimeRepository
	states    sessionquery.MediaSessionStateRepository
	media     port.MediaGateway
	provider  port.RealtimeProvider
	events    repository.ConversationEventPublisher
	log       *zap.Logger
	now       func() time.Time
	locks     sync.Map
	realtime  realtimeControlConfig
	project   mediaSessionProjector
	stats     roomStatsQuery
	eventsIn  realtimeEventPolicy
	eventsOut conversationEventMapper
}

type realtimeControlConfig struct {
	dataChannelLabel          string
	initialEvents             []string
	realtimeEventHistoryLimit int
}

type ServiceOptions struct {
	RealtimeDataChannelLabel  string
	RealtimeInitialEvents     []string
	RealtimeEventHistoryLimit int
}

func NewService(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	provider port.RealtimeProvider,
) Service {
	return newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, defaultRealtimeControlConfig(), zap.NewNop())
}

func NewServiceWithOptions(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	provider port.RealtimeProvider,
	options ServiceOptions,
) Service {
	return newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, realtimeControlConfigFromOptions(options), zap.NewNop())
}

func NewServiceWithOptionsAndLogger(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	provider port.RealtimeProvider,
	options ServiceOptions,
	log *zap.Logger,
) Service {
	return newService(records, runtime, states, media, provider, noopConversationEventPublisher{}, realtimeControlConfigFromOptions(options), log)
}

func NewServiceWithOptionsLoggerAndPublisher(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	provider port.RealtimeProvider,
	events repository.ConversationEventPublisher,
	options ServiceOptions,
	log *zap.Logger,
) Service {
	return newService(records, runtime, states, media, provider, events, realtimeControlConfigFromOptions(options), log)
}

func newService(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	provider port.RealtimeProvider,
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
	realtimeEvents := realtimeEventPolicy{}
	svc := &service{
		records:   records,
		runtime:   runtime,
		states:    states,
		media:     media,
		provider:  provider,
		events:    events,
		log:       log,
		now:       time.Now,
		realtime:  realtime,
		project:   mediaSessionProjector{realtimeEventHistoryLimit: realtime.realtimeEventHistoryLimit},
		stats:     roomStatsQuery{},
		eventsIn:  realtimeEvents,
		eventsOut: conversationEventMapper{realtime: realtimeEvents},
	}
	return svc
}

func realtimeControlConfigFromOptions(options ServiceOptions) realtimeControlConfig {
	realtime := defaultRealtimeControlConfig()
	if label := strings.TrimSpace(options.RealtimeDataChannelLabel); label != "" {
		realtime.dataChannelLabel = label
	}
	realtime.initialEvents = compactRealtimeInitialEvents(options.RealtimeInitialEvents)
	if options.RealtimeEventHistoryLimit > 0 {
		realtime.realtimeEventHistoryLimit = options.RealtimeEventHistoryLimit
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

	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		return sessiondto.CreateSessionResponse{}, err
	}

	return sessiondto.CreateSessionResponse{
		SessionID:      string(sessionID),
		ConversationID: string(conversationID),
		RoomID:         string(roomID),
		Status:         string(room.Status),
	}, nil
}

func (s *service) JoinSession(ctx context.Context, req sessiondto.JoinSessionCommand) (sessiondto.JoinSessionResult, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	unlock := s.lockSession(sessionID)
	defer unlock()

	room, err := s.findSessionRoom(ctx, sessionID)
	if err != nil {
		return sessiondto.JoinSessionResult{}, err
	}
	if !room.CanJoinParticipants() {
		return sessiondto.JoinSessionResult{}, usecase.ErrSessionNotJoinable
	}
	publishAudio := req.PublishesAudio()
	room.SetUserID(req.UserID, now)

	participant, peer, err := s.acceptClientParticipant(ctx, sessionID, req.SDP, publishAudio, now)
	if err != nil {
		return sessiondto.JoinSessionResult{}, err
	}
	if err := room.JoinClient(participant, now); err != nil {
		_ = s.media.CloseParticipant(ctx, sessionID, participant.ID)
		return sessiondto.JoinSessionResult{}, toJoinSessionError(err)
	}

	if err := s.ensureOpenAIParticipant(ctx, &room, sessionID, req.UserID, now); err != nil {
		return sessiondto.JoinSessionResult{}, err
	}

	if err := s.saveActiveRoomState(ctx, room, req.UserID, now); err != nil {
		_ = s.failRoom(ctx, room, req.UserID, now, "join_state_persist_failed")
		return sessiondto.JoinSessionResult{}, err
	}

	s.logParticipantEvent("media_participant_joined", room, participant,
		zap.String("audio_mode", participantAudioMode(participant)),
		zap.Int("participants", room.ParticipantCount()),
	)
	for _, joined := range room.Participants() {
		if joined.Role == vo.ParticipantRoleOpenAIAgent {
			s.logParticipantEvent("media_participant_ready", room, joined,
				zap.String("provider_call_id", joined.ProviderCallID),
			)
			break
		}
	}

	return sessiondto.JoinSessionResult{
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

	record, found, err := s.records.FindBySessionID(ctx, sessionID)
	if err != nil {
		return entity.Room{}, err
	}
	if !found {
		return entity.Room{}, usecase.ErrSessionNotFound
	}
	return record.RuntimeRoom(), nil
}

func (s *service) acceptClientParticipant(
	ctx context.Context,
	sessionID vo.SessionID,
	sdp string,
	publishAudio bool,
	now time.Time,
) (entity.Participant, *port.Peer, error) {
	participantID := vo.NewParticipantID()
	peer, err := s.media.AcceptOffer(ctx, port.OfferInput{
		SessionID:     sessionID,
		ParticipantID: participantID,
		Role:          vo.ParticipantRoleClient,
		SDP:           sdp,
		PublishAudio:  publishAudio,
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
	if room.HasOpenAIAgent() {
		return nil
	}

	openAIParticipant, err := s.connectOpenAIParticipant(ctx, sessionID, now)
	if err != nil {
		_ = s.failRoom(ctx, *room, userID, now, "openai_setup_failed")
		return err
	}
	if err := room.AttachOpenAIAgent(openAIParticipant, now); err != nil {
		return toJoinSessionError(err)
	}
	return nil
}

func toJoinSessionError(err error) error {
	if errors.Is(err, entity.ErrRoomNotJoinable) {
		return usecase.ErrSessionNotJoinable
	}
	return err
}

func (s *service) saveActiveRoomState(ctx context.Context, room entity.Room, userID string, now time.Time) error {
	if err := s.runtime.Save(ctx, room); err != nil {
		return err
	}
	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		return err
	}
	if err := s.states.Save(ctx, s.project.Project(room, userID, now)); err != nil {
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
	offer, err := s.media.CreateOffer(ctx, port.CreateOfferInput{
		SessionID:           sessionID,
		ParticipantID:       participantID,
		Role:                vo.ParticipantRoleOpenAIAgent,
		DataChannelLabel:    s.realtime.dataChannelLabel,
		InitialDataMessages: s.realtime.initialEvents,
	})
	if err != nil {
		return entity.Participant{}, err
	}

	call, err := s.provider.CreateCall(ctx, port.CreateCallInput{
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
	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		return sessiondto.LeaveParticipantResponse{}, err
	}
	if err := s.states.Save(ctx, s.project.Project(room, req.UserID, now)); err != nil {
		return sessiondto.LeaveParticipantResponse{}, err
	}

	s.logParticipantEvent("media_participant_left", room, participant,
		zap.String("audio_mode", participantAudioMode(participant)),
		zap.Int("participants", room.ParticipantCount()),
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
		record, recordFound, recordErr := s.records.FindBySessionID(ctx, sessionID)
		err = recordErr
		if err != nil {
			return sessiondto.EndSessionResponse{}, err
		}
		if !recordFound {
			return sessiondto.EndSessionResponse{}, nil
		}
		room = record.RuntimeRoom()
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

type noopConversationEventPublisher struct{}

func (noopConversationEventPublisher) Publish(ctx context.Context, event entity.ConversationEvent) error {
	_ = ctx
	_ = event
	return nil
}
