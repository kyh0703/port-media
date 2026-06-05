package auth

import (
	"github.com/kyh0703/portfoilo-media/internal/core/port"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"auth",
	fx.Provide(
		fx.Annotate(
			NewRedisMediaTokenStore,
			fx.As(new(port.MediaTokenStore)),
		),
		NewMediaTokenVerifier,
	),
)
