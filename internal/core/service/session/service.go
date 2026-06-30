package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/repository"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	sessionquery "github.com/kyh0703/portfoilo-media/internal/core/query/session"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
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
	records  repository.MediaSessionRecordRepository
	runtime  repository.RoomRuntimeRepository
	states   sessionquery.MediaSessionStateRepository
	media    port.MediaGateway
	log      *zap.Logger
	now      func() time.Time
	locks    sync.Map
	realtime realtimeControlConfig
	project  mediaSessionProjector
	stats    roomStatsQuery
}

type realtimeControlConfig struct {
	runtimeEventHistoryLimit int
}

type ServiceOptions struct {
	RuntimeEventHistoryLimit int
}

func NewService(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
) Service {
	return newService(records, runtime, states, media, defaultRealtimeControlConfig(), zap.NewNop())
}

func NewServiceWithOptions(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	options ServiceOptions,
) Service {
	return newService(records, runtime, states, media, realtimeControlConfigFromOptions(options), zap.NewNop())
}

func NewServiceWithOptionsAndLogger(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	options ServiceOptions,
	log *zap.Logger,
) Service {
	return newService(records, runtime, states, media, realtimeControlConfigFromOptions(options), log)
}

func newService(
	records repository.MediaSessionRecordRepository,
	runtime repository.RoomRuntimeRepository,
	states sessionquery.MediaSessionStateRepository,
	media port.MediaGateway,
	realtime realtimeControlConfig,
	log *zap.Logger,
) Service {
	if log == nil {
		log = zap.NewNop()
	}
	svc := &service{
		records:  records,
		runtime:  runtime,
		states:   states,
		media:    media,
		log:      log,
		now:      time.Now,
		realtime: realtime,
		project:  mediaSessionProjector{runtimeEventHistoryLimit: realtime.runtimeEventHistoryLimit},
		stats:    roomStatsQuery{},
	}
	return svc
}

func realtimeControlConfigFromOptions(options ServiceOptions) realtimeControlConfig {
	realtime := defaultRealtimeControlConfig()
	if options.RuntimeEventHistoryLimit > 0 {
		realtime.runtimeEventHistoryLimit = options.RuntimeEventHistoryLimit
	}
	return realtime
}

func defaultRealtimeControlConfig() realtimeControlConfig {
	return realtimeControlConfig{
		runtimeEventHistoryLimit: 10,
	}
}

func (s *service) CreateSession(ctx context.Context, req sessionio.CreateSessionRequest) (sessionio.CreateSessionResponse, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	if sessionID == "" {
		sessionID = vo.NewSessionID()
	}

	conversationID := vo.ConversationID(req.ConversationID)
	roomID := vo.NewRoomID()
	room := entity.NewRoom(roomID, sessionID, conversationID, now)

	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		return sessionio.CreateSessionResponse{}, err
	}

	return sessionio.CreateSessionResponse{
		SessionID:      string(sessionID),
		ConversationID: string(conversationID),
		RoomID:         string(roomID),
		Status:         string(room.Status),
	}, nil
}

func (s *service) JoinSession(ctx context.Context, req sessionio.JoinSessionCommand) (sessionio.JoinSessionResult, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	unlock := s.lockSession(sessionID)
	defer unlock()

	room, err := s.findSessionRoom(ctx, sessionID)
	if err != nil {
		return sessionio.JoinSessionResult{}, err
	}
	if !room.CanJoinParticipants() {
		return sessionio.JoinSessionResult{}, usecase.ErrSessionNotJoinable
	}
	publishAudio := req.PublishesAudio()
	room.SetUserID(req.UserID, now)

	participant, peer, err := s.acceptParticipant(ctx, sessionID, req, publishAudio, now)
	if err != nil {
		return sessionio.JoinSessionResult{}, err
	}
	if err := room.JoinParticipant(participant, now); err != nil {
		_ = s.media.CloseParticipant(ctx, sessionID, participant.ID)
		return sessionio.JoinSessionResult{}, toJoinSessionError(err)
	}

	if err := s.saveActiveRoomState(ctx, room, req.UserID, now); err != nil {
		_ = s.failRoom(ctx, room, req.UserID, now, "join_state_persist_failed")
		return sessionio.JoinSessionResult{}, err
	}

	s.logParticipantEvent("media_participant_joined", room, participant,
		zap.String("audio_mode", participantAudioMode(participant)),
		zap.Int("participants", room.ParticipantCount()),
	)
	return sessionio.JoinSessionResult{
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

func (s *service) acceptParticipant(
	ctx context.Context,
	sessionID vo.SessionID,
	req sessionio.JoinSessionCommand,
	publishAudio bool,
	now time.Time,
) (entity.Participant, *port.Peer, error) {
	participantID := vo.ParticipantID(req.ParticipantID)
	if participantID == "" {
		participantID = vo.NewParticipantID()
	}
	role := participantRole(req.ParticipantRole)
	peer, err := s.media.AcceptOffer(ctx, port.OfferInput{
		SessionID:     sessionID,
		ParticipantID: participantID,
		Role:          role,
		SDP:           req.SDP,
		PublishAudio:  publishAudio,
	})
	if err != nil {
		return entity.Participant{}, nil, err
	}

	return newParticipant(participantID, role, publishAudio, now), peer, nil
}

func participantRole(role string) vo.ParticipantRole {
	switch vo.ParticipantRole(role) {
	case vo.ParticipantRoleAgent,
		vo.ParticipantRoleRecorder,
		vo.ParticipantRoleSIP,
		vo.ParticipantRoleMonitor,
		vo.ParticipantRoleService:
		return vo.ParticipantRole(role)
	default:
		return vo.ParticipantRoleUser
	}
}

func newParticipant(participantID vo.ParticipantID, role vo.ParticipantRole, publishAudio bool, now time.Time) entity.Participant {
	participant := entity.NewParticipant(participantID, role, now)
	participant.SetState(vo.ConnectionStateConnecting, now)
	participant.SetPublishAudio(publishAudio, now)
	if publishAudio {
		participant.AddTrack(entity.NewTrack(vo.NewTrackID(), vo.TrackKindAudio, now), now)
	}
	return participant
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
	if err := s.states.Save(ctx, s.projectedState(ctx, room, userID, now)); err != nil {
		return err
	}
	return nil
}

func (s *service) projectedState(ctx context.Context, room entity.Room, userID string, now time.Time) sessionquery.MediaSessionState {
	existingState, found, err := s.states.FindBySessionID(ctx, room.SessionID)
	if err != nil || !found {
		return s.project.Project(room, userID, now)
	}
	return s.project.ProjectWithRuntimeEventHistory(room, userID, now, existingState.RecentRuntimeEvents)
}

func (s *service) lockSession(sessionID vo.SessionID) func() {
	lock, _ := s.locks.LoadOrStore(sessionID, &sync.Mutex{})
	sessionLock := lock.(*sync.Mutex)
	sessionLock.Lock()
	return func() {
		sessionLock.Unlock()
	}
}

func (s *service) LeaveParticipant(ctx context.Context, req sessionio.LeaveParticipantRequest) (sessionio.LeaveParticipantResponse, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	participantID := vo.ParticipantID(req.ParticipantID)
	unlock := s.lockSession(sessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, sessionID)
	if err != nil {
		return sessionio.LeaveParticipantResponse{}, err
	}
	if !found {
		return sessionio.LeaveParticipantResponse{}, nil
	}

	participant, found := room.Participant(participantID)
	if !found {
		return sessionio.LeaveParticipantResponse{
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
		return sessionio.LeaveParticipantResponse{
			SessionID:     string(sessionID),
			RoomID:        string(room.ID),
			ParticipantID: string(participantID),
			Status:        string(vo.RoomStatusFailed),
		}, nil
	}

	room.RemoveParticipant(participantID, now)
	if err := s.media.CloseParticipant(ctx, sessionID, participantID); err != nil {
		return sessionio.LeaveParticipantResponse{}, err
	}
	if err := s.runtime.Save(ctx, room); err != nil {
		return sessionio.LeaveParticipantResponse{}, err
	}
	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		return sessionio.LeaveParticipantResponse{}, err
	}
	if err := s.states.Save(ctx, s.projectedState(ctx, room, req.UserID, now)); err != nil {
		return sessionio.LeaveParticipantResponse{}, err
	}

	s.logParticipantEvent("media_participant_left", room, participant,
		zap.String("audio_mode", participantAudioMode(participant)),
		zap.Int("participants", room.ParticipantCount()),
	)

	return sessionio.LeaveParticipantResponse{
		SessionID:     string(sessionID),
		RoomID:        string(room.ID),
		ParticipantID: string(participantID),
		Status:        string(room.Status),
	}, nil
}

func (s *service) EndSession(ctx context.Context, req sessionio.EndSessionRequest) (sessionio.EndSessionResponse, error) {
	now := s.now().UTC()
	sessionID := vo.SessionID(req.SessionID)
	unlock := s.lockSession(sessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, sessionID)
	if err != nil {
		return sessionio.EndSessionResponse{}, err
	}
	if !found {
		record, recordFound, recordErr := s.records.FindBySessionID(ctx, sessionID)
		err = recordErr
		if err != nil {
			return sessionio.EndSessionResponse{}, err
		}
		if !recordFound {
			return sessionio.EndSessionResponse{}, nil
		}
		room = record.RuntimeRoom()
	}

	if err := s.closeRoom(ctx, room, req.UserID, now, "explicit_end"); err != nil {
		return sessionio.EndSessionResponse{}, err
	}

	return sessionio.EndSessionResponse{
		SessionID: string(sessionID),
		RoomID:    string(room.ID),
		Status:    string(vo.RoomStatusClosed),
	}, nil
}
