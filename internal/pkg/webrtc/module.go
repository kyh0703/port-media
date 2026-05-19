package webrtc

import "go.uber.org/fx"

var Module = fx.Module(
	"webrtc",
	fx.Provide(
		fx.Annotate(
			NewEngine,
			fx.As(new(PeerConnectionFactory)),
		),
	),
)
