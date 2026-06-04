package handler

import (
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http/middleware"
	"go.uber.org/fx"
)

//go:generate go tool counterfeiter -generate

type ErrorHandlerFunc = middleware.ErrorHandlerFunc

type Mapper struct {
	Method  string
	Path    string
	Handler middleware.ErrorHandlerFunc
}

//counterfeiter:generate . Handler
type Handler interface {
	Table() []Mapper
}

func AsHandler(constructor any) any {
	return fx.Annotate(
		constructor,
		fx.As(new(Handler)),
		fx.ResultTags(`group:"handlers"`),
	)
}
