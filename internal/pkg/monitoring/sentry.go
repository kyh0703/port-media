package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sentry "github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/kyh0703/portfoilo-media/configs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const flushTimeout = 2 * time.Second

// Sentry wraps the sentry-go client lifecycle. It is always constructed; when no
// DSN is configured it is disabled and every method is a no-op.
type Sentry struct {
	enabled bool
	log     *zap.Logger
}

// NewSentry initializes the global sentry client when a DSN is set and registers
// a flush on shutdown. It returns a non-nil *Sentry even when disabled.
func NewSentry(lc fx.Lifecycle, cfg *configs.Config, log *zap.Logger) (*Sentry, error) {
	dsn := cfg.Sentry.DSN
	if dsn == "" {
		log.Info("sentry_disabled")
		return &Sentry{enabled: false, log: log}, nil
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: cfg.App.Env,
		Release:     cfg.App.Version,
	}); err != nil {
		return nil, fmt.Errorf("init sentry: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			_ = ctx
			sentry.Flush(flushTimeout)
			return nil
		},
	})

	log.Info("sentry_enabled", zap.String("environment", cfg.App.Env))
	return &Sentry{enabled: true, log: log}, nil
}

// Enabled reports whether a DSN was configured.
func (s *Sentry) Enabled() bool {
	return s != nil && s.enabled
}

// Middleware returns a net/http handler that attaches a request-scoped hub to the
// request context. When disabled it is a pass-through.
func (s *Sentry) Middleware(next http.Handler) http.Handler {
	if !s.Enabled() {
		return next
	}
	return sentryhttp.New(sentryhttp.Options{Repanic: true}).Handle(next)
}

// ReportPanic satisfies the middleware.PanicReporter contract. It captures the
// recovered panic on the request-scoped hub when available.
func (s *Sentry) ReportPanic(ctx context.Context, recovered any, stack []byte) {
	if !s.Enabled() {
		return
	}
	_ = stack
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}
	hub.RecoverWithContext(ctx, recovered)
}

// CaptureException reports a non-panic error to Sentry. Safe to call when disabled.
func (s *Sentry) CaptureException(ctx context.Context, err error) {
	if !s.Enabled() || err == nil {
		return
	}
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}
	hub.CaptureException(err)
}
