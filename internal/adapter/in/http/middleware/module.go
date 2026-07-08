package middleware

import (
	"github.com/kyh0703/portfoilo-media/internal/pkg/monitoring"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var Module = fx.Module(
	"http-middleware",
	fx.Provide(NewRequestLogger),
	fx.Provide(newRecoverMiddleware),
)

func newRecoverMiddleware(log *zap.Logger, sentry *monitoring.Sentry) *RecoverMiddleware {
	return NewRecoverMiddlewareWithReporter(log, sentry)
}
