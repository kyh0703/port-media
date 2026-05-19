package main

import (
	"strings"

	"github.com/gofiber/contrib/fiberzap/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/handler"
	"github.com/kyh0703/portfoilo-media/internal/core/middleware"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"github.com/kyh0703/portfoilo-media/internal/pkg/observability"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FiberParams struct {
	fx.In

	Config        *configs.Config
	Logger        *zap.Logger
	RequestLogger *middleware.RequestLogger
	Handlers      []handler.Handler `group:"handlers"`
	Session       usecase.SessionUsecase
}

func NewFiber(params FiberParams) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      params.Config.App.Name,
		ErrorHandler: exception.NewErrorHandler(params.Logger),
		ReadTimeout:  params.Config.Server.ReadTimeout,
		WriteTimeout: params.Config.Server.WriteTimeout,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:  strings.Join(params.Config.Server.CORS.AllowedOrigins, ","),
		AllowMethods:  strings.Join(params.Config.Server.CORS.AllowedMethods, ","),
		AllowHeaders:  strings.Join(params.Config.Server.CORS.AllowedHeaders, ","),
		ExposeHeaders: strings.Join(params.Config.Server.CORS.ExposeHeaders, ","),
	}))
	app.Use(fiberzap.New(fiberzap.Config{Logger: params.Logger}))
	app.Use(params.RequestLogger.Handler())

	if params.Config.Observability.MetricsEnabled {
		app.Get("/metrics", func(c *fiber.Ctx) error {
			stats, err := params.Session.GetRuntimeStats(c.Context())
			if err != nil {
				return err
			}

			c.Type("txt", "utf-8")
			return c.SendString(observability.RuntimeStatsPrometheus(stats))
		})
	}

	api := app.Group("/api")
	v1 := api.Group("/v1")
	for _, h := range params.Handlers {
		for _, m := range h.Table() {
			v1.Add(m.Method, m.Path, m.Handler...)
		}
	}

	return app
}
