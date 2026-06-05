package webrtc

import "go.uber.org/fx"

var Module = fx.Module(
	"webrtc",
	fx.Provide(
		NewEngine,
		NewGateway,
	),
)
