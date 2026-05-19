package main

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/kyh0703/portfoilo-media/configs"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type App struct {
	fiber *fiber.App
	cfg   *configs.Config
	log   *zap.Logger
}

func NewApp(fiber *fiber.App, cfg *configs.Config, log *zap.Logger) *App {
	return &App{fiber: fiber, cfg: cfg, log: log}
}

func RegisterLifecycle(lc fx.Lifecycle, app *App) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			addr := fmt.Sprintf("%s:%d", app.cfg.Server.Host, app.cfg.Server.Port)
			go func() {
				if err := app.fiber.Listen(addr); err != nil {
					app.log.Error("fiber stopped", zap.Error(err))
				}
			}()
			app.log.Info("server_started", zap.String("addr", addr))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			app.log.Info("server_stopping")
			return app.fiber.ShutdownWithContext(ctx)
		},
	})
}
