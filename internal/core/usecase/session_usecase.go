package usecase

import (
	"context"
	"errors"
	"time"

	sessionio "github.com/kyh0703/portfoilo-media/internal/core/usecase/sessionio"
)

//go:generate go tool counterfeiter -generate

var (
	ErrSessionNotFound    = errors.New("media session not found")
	ErrSessionNotJoinable = errors.New("media session is not joinable")
)

//counterfeiter:generate . CreateSessionUsecase
type CreateSessionUsecase interface {
	CreateSession(ctx context.Context, req sessionio.CreateSessionRequest) (sessionio.CreateSessionResponse, error)
}

//counterfeiter:generate . JoinSessionUsecase
type JoinSessionUsecase interface {
	JoinSession(ctx context.Context, cmd sessionio.JoinSessionCommand) (sessionio.JoinSessionResult, error)
}

//counterfeiter:generate . LeaveParticipantUsecase
type LeaveParticipantUsecase interface {
	LeaveParticipant(ctx context.Context, req sessionio.LeaveParticipantRequest) (sessionio.LeaveParticipantResponse, error)
}

//counterfeiter:generate . EndSessionUsecase
type EndSessionUsecase interface {
	EndSession(ctx context.Context, req sessionio.EndSessionRequest) (sessionio.EndSessionResponse, error)
}

//counterfeiter:generate . GetSessionStatusQuery
type GetSessionStatusQuery interface {
	GetSessionStatus(ctx context.Context, req sessionio.GetSessionStatusRequest) (sessionio.GetSessionStatusResult, bool, error)
}

//counterfeiter:generate . GetRuntimeStatsQuery
type GetRuntimeStatsQuery interface {
	GetRuntimeStats(ctx context.Context) (sessionio.RuntimeStatsResponse, error)
}

//counterfeiter:generate . RoomMaintenanceUsecase
type RoomMaintenanceUsecase interface {
	CleanupIdleRooms(ctx context.Context, idleTimeout time.Duration) (int, error)
	ShutdownActiveRooms(ctx context.Context) (int, error)
}

//counterfeiter:generate . GetHealthQuery
type GetHealthQuery interface {
	GetHealth(ctx context.Context) error
}
