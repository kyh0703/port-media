package session

import (
	"context"
	"time"

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
	eventType := s.eventsIn.Type(message.Payload)
	room.RecordRealtimeEvent(eventType, now)

	if err := s.runtime.Save(ctx, room); err != nil {
		return
	}
	if err := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room)); err != nil {
		return
	}
	existingState, _, _ := s.states.FindBySessionID(ctx, message.SessionID)
	if err := s.states.Save(ctx, s.project.ProjectWithRealtimeEvent(
		room,
		room.UserID,
		now,
		eventType,
		existingState.RecentRealtimeEvents,
	)); err != nil {
		s.logRoomErrorEvent("media_session_state_save_failed", room,
			zap.String("operation", "record_realtime_event"),
			zap.Error(err),
		)
	}
	s.publishConversationEvent(ctx, room, eventType, message.Payload, now)
	s.logRoomEvent("media_realtime_event_recorded", room,
		zap.String("participant_id", string(message.ParticipantID)),
		zap.String("participant_role", string(message.Role)),
		zap.String("data_channel_label", message.Label),
		zap.String("realtime_event_type", eventType),
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
	if err := s.states.Save(ctx, s.project.Project(room, room.UserID, now)); err != nil {
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
	if err := s.states.Save(ctx, s.project.Project(room, room.UserID, now)); err != nil {
		s.logRoomErrorEvent("media_session_state_save_failed", room,
			zap.String("operation", "media_track_state_change"),
			zap.Error(err),
		)
	}
}

func (s *service) publishConversationEvent(
	ctx context.Context,
	room entity.Room,
	eventType string,
	payload string,
	occurredAt time.Time,
) {
	event, ok := s.eventsOut.Map(room, eventType, payload, occurredAt)
	if !ok {
		return
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
