package main

import (
	"fmt"

	"github.com/kyh0703/portfoilo-media/internal/core/port"
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
			fx.As(new(usecase.RoomMaintenanceUsecase)),
			fx.As(new(usecase.GetHealthQuery)),
		),
		NewMediaRuntimeEventHandler,
		NewHTTPHandler,
		NewApp,
	),
	fx.Invoke(RegisterMediaRuntimeEvents),
	fx.Invoke(RegisterLifecycle),
)

func RegisterMediaRuntimeEvents(subscriber port.MediaRuntimeEventSubscriber, handler port.MediaRuntimeEventHandler) {
	subscriber.SubscribeRuntimeEvents(handler)
}

func NewMediaRuntimeEventHandler(svc sessionservice.Service) (port.MediaRuntimeEventHandler, error) {
	handler, ok := svc.(port.MediaRuntimeEventHandler)
	if !ok {
		return nil, fmt.Errorf("session service does not implement media runtime event handler")
	}
	return handler, nil
}
