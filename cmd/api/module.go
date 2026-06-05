package main

import (
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"app",
	fx.Provide(
		NewSessionServiceOptions,
		fx.Annotate(
			sessionservice.NewServiceWithOptionsLoggerAndPublisher,
			fx.As(new(sessionservice.Service)),
			fx.As(new(usecase.CreateSessionUsecase)),
			fx.As(new(usecase.JoinSessionUsecase)),
			fx.As(new(usecase.LeaveParticipantUsecase)),
			fx.As(new(usecase.EndSessionUsecase)),
			fx.As(new(usecase.GetSessionStatusQuery)),
			fx.As(new(usecase.GetRuntimeStatsQuery)),
			fx.As(new(usecase.GetHealthQuery)),
		),
		NewHTTPHandler,
		NewApp,
	),
	fx.Invoke(RegisterLifecycle),
)
