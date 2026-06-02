package main

import "go.uber.org/fx"

var Module = fx.Module(
	"app",
	fx.Provide(
		NewHTTPHandler,
		NewApp,
	),
	fx.Invoke(RegisterLifecycle),
)
