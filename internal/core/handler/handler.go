package handler

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type Mapper struct {
	Method  string
	Path    string
	Handler []fiber.Handler
}

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
