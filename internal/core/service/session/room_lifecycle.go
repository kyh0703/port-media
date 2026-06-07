package session

import (
	"context"
	"errors"
	"time"

	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"github.com/kyh0703/portfoilo-media/internal/core/domain/vo"
	"go.uber.org/zap"
)

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

func (s *service) failRoom(ctx context.Context, room entity.Room, userID string, now time.Time, reason string, fields ...zap.Field) error {
	room.Fail(now)
	cleanupErr := errors.Join(
		s.hangupOpenAIParticipants(ctx, room),
		s.media.CloseSession(ctx, room.SessionID),
		s.runtime.Delete(ctx, room.ID),
	)
	if cleanupErr != nil {
		s.logRoomEvent("media_room_cleanup_failed", room,
			zap.String("operation", "fail_room"),
			zap.String("failure_reason", reason),
			zap.Error(cleanupErr),
		)
	}
	recordErr := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room))
	stateErr := s.states.Save(ctx, s.project.Project(room, userID, now))
	if recordErr != nil || stateErr != nil {
		s.logRoomEvent("media_room_failure_state_save_failed", room,
			zap.String("failure_reason", reason),
			zap.Error(errors.Join(recordErr, stateErr)),
		)
	}
	logFields := append([]zap.Field{zap.String("failure_reason", reason)}, fields...)
	s.logRoomEvent("media_room_failed", room, logFields...)
	return errors.Join(recordErr, stateErr, cleanupErr)
}

func (s *service) closeRoom(ctx context.Context, room entity.Room, userID string, now time.Time, reason string) error {
	cleanupErr := errors.Join(
		s.hangupOpenAIParticipants(ctx, room),
		s.media.CloseSession(ctx, room.SessionID),
	)
	if cleanupErr != nil {
		s.logRoomEvent("media_room_cleanup_failed", room,
			zap.String("operation", "close_room"),
			zap.String("close_reason", reason),
			zap.Error(cleanupErr),
		)
	}

	room.Close(now)
	deleteErr := s.runtime.Delete(ctx, room.ID)
	recordErr := s.records.Save(ctx, entity.NewMediaSessionRecordFromRoom(room))
	stateErr := s.states.Save(ctx, s.project.Project(room, userID, now))
	persistErr := errors.Join(deleteErr, recordErr, stateErr)
	if persistErr != nil {
		s.logRoomEvent("media_room_close_state_save_failed", room,
			zap.String("close_reason", reason),
			zap.Error(persistErr),
		)
	}
	s.logRoomEvent("media_room_closed", room,
		zap.String("close_reason", reason),
		zap.Int("participants", room.ParticipantCount()),
	)
	return errors.Join(persistErr, cleanupErr)
}

func (s *service) hangupOpenAIParticipants(ctx context.Context, room entity.Room) error {
	for _, participant := range room.Participants() {
		if participant.Role == vo.ParticipantRoleOpenAIAgent && participant.ProviderCallID != "" {
			if err := s.provider.HangupCall(ctx, participant.ProviderCallID); err != nil {
				return err
			}
		}
	}
	return nil
}
