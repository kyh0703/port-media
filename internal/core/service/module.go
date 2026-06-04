package service

import (
	sessionservice "github.com/kyh0703/portfoilo-media/internal/core/service/session"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"service",
	fx.Provide(
		fx.Annotate(
			sessionservice.NewServiceWithConfigLoggerAndPublisher,
			fx.As(new(sessionservice.Service)),
			fx.As(new(usecase.SessionUsecase)),
		),
	),
	fx.Invoke(sessionservice.RegisterIdleCleanup),
)
