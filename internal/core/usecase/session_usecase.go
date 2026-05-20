package usecase

import (
	"context"
	"errors"

	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
)

var ErrSessionNotFound = errors.New("media session not found")

type SessionUsecase interface {
	CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error)
	AcceptOffer(ctx context.Context, req sessiondto.AcceptOfferRequest) (sessiondto.AcceptOfferResponse, error)
	LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error)
	EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error)
	GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error)
	GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error)
	GetHealth(ctx context.Context) error
}
