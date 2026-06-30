package session

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	"go.uber.org/zap"
)

func (s *service) HandleDataChannelMessage(ctx context.Context, message port.DataChannelMessage) {
	now := s.now().UTC()
	unlock := s.lockSession(message.SessionID)
	defer unlock()

	room, found, err := s.runtime.FindBySessionID(ctx, message.SessionID)
	if err != nil || !found {
		return
	}
	room.RecordRuntimeEvent(runtimeEventTypeDataChannelMessage, now)

	if err := s.runtime.Save(ctx, room); err != nil {
		return
	}
	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		return
	}
	existingState, _, _ := s.states.FindBySessionID(ctx, message.SessionID)
	if err := s.states.Save(ctx, s.project.ProjectWithRuntimeEvent(
		room,
		room.UserID,
		now,
		runtimeEventTypeDataChannelMessage,
		existingState.RecentRuntimeEvents,
	)); err != nil {
		s.logRoomErrorEvent("media_session_state_save_failed", room,
			zap.String("operation", "record_runtime_event"),
			zap.Error(err),
		)
	}
	s.logRoomEvent("media_runtime_event_recorded", room,
		zap.String("participant_id", string(message.ParticipantID)),
		zap.String("participant_role", string(message.Role)),
		zap.String("data_channel_label", message.Label),
		zap.String("runtime_event_type", runtimeEventTypeDataChannelMessage),
	)
}

func (s *service) HandleConnectionStateChange(ctx context.Context, change port.ConnectionStateChange) {
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
	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
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
	if err := s.states.Save(ctx, s.projectedState(ctx, room, room.UserID, now)); err != nil {
		s.logRoomErrorEvent("media_session_state_save_failed", room,
			zap.String("operation", "connection_state_change"),
			zap.Error(err),
		)
	}
}

func (s *service) HandleMediaTrackStateChange(ctx context.Context, change port.MediaTrackStateChange) {
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
	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
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
	if err := s.states.Save(ctx, s.projectedState(ctx, room, room.UserID, now)); err != nil {
		s.logRoomErrorEvent("media_session_state_save_failed", room,
			zap.String("operation", "media_track_state_change"),
			zap.Error(err),
		)
	}
}
