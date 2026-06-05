package session

import (
	"github.com/kyh0703/portfoilo-media/internal/core/domain/entity"
	"go.uber.org/zap"
)

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
