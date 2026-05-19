package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kyh0703/portfoilo-media/internal/pkg/health"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
)

type HealthHandler struct {
	checker health.Checker
}

func NewHealthHandler(checker health.Checker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

func (h *HealthHandler) Table() []Mapper {
	return []Mapper{
		{Method: fiber.MethodGet, Path: "/health", Handler: []fiber.Handler{h.Get}},
	}
}

func (h *HealthHandler) Get(c *fiber.Ctx) error {
	result := h.checker.Check(c.Context())
	if result.Status != health.StatusOK {
		return c.Status(fiber.StatusServiceUnavailable).JSON(response.Success(
			fiber.StatusServiceUnavailable,
			"Service Unavailable",
			result,
		))
	}

	return c.JSON(response.OK(result))
}
