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
	mux := http.NewServeMux()
	handleError := exception.NewHTTPErrorHandler(params.Logger)
	for _, h := range params.Handlers {
		for _, route := range h.Table() {
			mux.Handle(routePattern(apiV1Prefix, route), middleware.WithErrorHandler(route.Handler, handleError))
		}
	}

	var app http.Handler = mux
	if params.Recover != nil {
		app = params.Recover.Handler(app)
	}
	if params.RequestLogger != nil {
		app = params.RequestLogger.Handler(app)
	}
	app = middleware.CORS(params.Config.Server.CORS)(app)

	return app
}

func routePattern(prefix string, route handler.Mapper) string {
	return route.Method + " " + prefix + route.Path
}
