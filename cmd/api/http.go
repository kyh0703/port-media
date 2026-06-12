package main

import (
	"net/http"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http/middleware"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const apiV1Prefix = "/api/v1"

type HTTPParams struct {
	fx.In

	Config        *configs.Config
	Logger        *zap.Logger
	RequestLogger *middleware.RequestLogger
	Recover       *middleware.RecoverMiddleware
	Handlers      []handler.Handler `group:"handlers"`
}

func NewHTTPHandler(params HTTPParams) http.Handler {
	api := http.NewServeMux()
	handleError := exception.NewHTTPErrorHandler(params.Logger)
	for _, h := range params.Handlers {
		for _, route := range h.Table() {
			api.Handle(routePattern(route), middleware.WithErrorHandler(route.Handler, handleError))
		}
	}

	root := http.NewServeMux()
	root.Handle(apiV1Prefix+"/", http.StripPrefix(apiV1Prefix, api))

	var app http.Handler = root
	if params.Recover != nil {
		app = params.Recover.Handler(app)
	}
	app = middleware.CORS(params.Config.Server.CORS)(app)
	if params.RequestLogger != nil {
		app = params.RequestLogger.Handler(app)
	}

	return app
}

func routePattern(route handler.Mapper) string {
	return route.Method + " " + route.Path
}
