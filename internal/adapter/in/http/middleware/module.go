package middleware

import "go.uber.org/fx"

var Module = fx.Module(
	"http-middleware",
	fx.Provide(NewRequestLogger),
	fx.Provide(NewRecoverMiddleware),
)
