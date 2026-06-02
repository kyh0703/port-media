package handler

import (
	"github.com/kyh0703/portfoilo-media/internal/pkg/httpx"
	"go.uber.org/fx"
)

//go:generate go tool counterfeiter -generate

type ErrorHandlerFunc = httpx.ErrorHandlerFunc

type Mapper struct {
	Method  string
	Path    string
	Handler httpx.ErrorHandlerFunc
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
