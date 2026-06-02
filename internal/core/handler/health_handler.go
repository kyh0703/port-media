package handler

import (
	"net/http"

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
		{Method: http.MethodGet, Path: "/health", Handler: h.Get},
	}
}

func (h *HealthHandler) Get(w http.ResponseWriter, r *http.Request) error {
	result := h.checker.Check(r.Context())
	if result.Status != health.StatusOK {
		return response.WriteJSON(w, http.StatusServiceUnavailable, response.Success(
			http.StatusServiceUnavailable,
			"Service Unavailable",
			result,
		))
	}

	return response.WriteJSON(w, http.StatusOK, response.OK(result))
}
