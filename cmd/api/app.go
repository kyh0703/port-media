package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/kyh0703/portfoilo-media/configs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type App struct {
	server *http.Server
	log    *zap.Logger
}

func NewApp(handler http.Handler, cfg *configs.Config, log *zap.Logger) *App {
	return &App{
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
			Handler:      handler,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
		log: log,
	}
}

func RegisterLifecycle(lc fx.Lifecycle, app *App) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			_ = ctx
			go func() {
				if err := app.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					app.log.Error("http server stopped", zap.Error(err))
				}
			}()
			app.log.Info("server_started", zap.String("addr", app.server.Addr))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			app.log.Info("server_stopping")
			return app.server.Shutdown(ctx)
		},
	})
}
