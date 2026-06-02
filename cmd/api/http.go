package main

import (
	"net/http"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/core/handler"
	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"github.com/kyh0703/portfoilo-media/internal/pkg/httpx"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const apiV1Prefix = "/api/v1"

type HTTPParams struct {
	fx.In

	Config        *configs.Config
	Logger        *zap.Logger
	RequestLogger *httpx.RequestLogger
	Handlers      []handler.Handler `group:"handlers"`
}

func NewHTTPHandler(params HTTPParams) http.Handler {
	mux := http.NewServeMux()
	handleError := exception.NewHTTPErrorHandler(params.Logger)
	for _, h := range params.Handlers {
		for _, route := range h.Table() {
			mux.Handle(routePattern(apiV1Prefix, route), httpx.WithErrorHandler(route.Handler, handleError))
		}
	}

	var app http.Handler = mux
	if params.RequestLogger != nil {
		app = params.RequestLogger.Handler(app)
	}
	app = httpx.CORS(params.Config.Server.CORS)(app)

	return app
}

func routePattern(prefix string, route handler.Mapper) string {
	return route.Method + " " + prefix + route.Path
}
