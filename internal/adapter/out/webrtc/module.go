package webrtc

import (
	"context"

	"github.com/kyh0703/portfoilo-media/internal/core/port"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"webrtc",
	fx.Provide(
		NewEngine,
		NewGatewayForLifecycle,
		NewMediaGateway,
		NewMediaRuntimeEventSubscriber,
	),
)

type GatewayParams struct {
	fx.In

	Engine    *Engine
	Lifecycle fx.Lifecycle
}

func NewGatewayForLifecycle(params GatewayParams) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())
	params.Lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
	return NewGatewayWithEventContext(params.Engine, ctx)
}

func NewMediaGateway(gateway *Gateway) port.MediaGateway {
	return gateway
}

func NewMediaRuntimeEventSubscriber(gateway *Gateway) port.MediaRuntimeEventSubscriber {
	return gateway
}
