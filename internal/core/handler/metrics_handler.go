package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kyh0703/portfoilo-media/internal/core/usecase"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
)

type MetricsHandler struct {
	session usecase.SessionUsecase
}

func NewMetricsHandler(session usecase.SessionUsecase) *MetricsHandler {
	return &MetricsHandler{session: session}
}

func (h *MetricsHandler) Table() []Mapper {
	return []Mapper{
		{Method: fiber.MethodGet, Path: "/metrics", Handler: []fiber.Handler{h.Get}},
	}
}

func (h *MetricsHandler) Get(c *fiber.Ctx) error {
	stats, err := h.session.GetRuntimeStats(c.Context())
	if err != nil {
		return err
	}

	return c.JSON(response.OK(stats))
}
