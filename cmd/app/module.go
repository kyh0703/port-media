package main

import "go.uber.org/fx"

var Module = fx.Module(
	"app",
	fx.Provide(
		NewFiber,
		NewApp,
	),
	fx.Invoke(RegisterLifecycle),
)
