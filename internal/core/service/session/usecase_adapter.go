package session

import "github.com/kyh0703/portfoilo-media/internal/core/usecase"

var _ usecase.CreateSessionUsecase = (*service)(nil)
var _ usecase.JoinSessionUsecase = (*service)(nil)
var _ usecase.LeaveParticipantUsecase = (*service)(nil)
var _ usecase.EndSessionUsecase = (*service)(nil)
var _ usecase.GetSessionStatusQuery = (*service)(nil)
var _ usecase.GetRuntimeStatsQuery = (*service)(nil)
var _ usecase.GetHealthQuery = (*service)(nil)
