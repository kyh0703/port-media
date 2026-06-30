package session

import "github.com/kyh0703/portfoilo-media/internal/core/domain/vo"

func isCriticalParticipant(role vo.ParticipantRole) bool {
	return role == vo.ParticipantRoleAgent
}
