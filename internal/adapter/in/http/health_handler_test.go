package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/pkg/health"
)

type fakeHealthChecker struct {
	result health.Result
}

func (f fakeHealthChecker) Check(ctx context.Context) health.Result {
	_ = ctx
	return f.result
}

func TestHealthHandlerReturnsCommonOKResponse(t *testing.T) {
	handler := NewHealthHandler(fakeHealthChecker{
		result: health.Result{
			Status: health.StatusOK,
			Checks: map[string]health.DependencyCheck{
				"postgres": {Status: health.StatusOK},
				"redis":    {Status: health.StatusOK},
			},
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}

	var body struct {
		StatusCode int           `json:"statusCode"`
		Message    string        `json:"message"`
		Data       health.Result `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.StatusCode != http.StatusOK {
		t.Fatalf("statusCode = %d, want %d", body.StatusCode, http.StatusOK)
	}
	if body.Message != "OK" {
		t.Fatalf("message = %q, want OK", body.Message)
	}
	if body.Data.Status != health.StatusOK {
		t.Fatalf("data.status = %q, want %q", body.Data.Status, health.StatusOK)
	}
}

func TestHealthHandlerReturnsCommonDegradedResponse(t *testing.T) {
	handler := NewHealthHandler(fakeHealthChecker{
		result: health.Result{
			Status: health.StatusDegraded,
			Checks: map[string]health.DependencyCheck{
				"postgres": {Status: health.StatusOK},
				"redis":    {Status: health.StatusFailed, Error: "connection refused"},
			},
		},
	})

	app := newTestApp()
	for _, route := range handler.Table() {
		app.Add(route.Method, route.Path, route.Handler)
	}

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusServiceUnavailable)
	}

	var body struct {
		StatusCode int           `json:"statusCode"`
		Message    string        `json:"message"`
		Data       health.Result `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("statusCode = %d, want %d", body.StatusCode, http.StatusServiceUnavailable)
	}
	if body.Message != "Service Unavailable" {
		t.Fatalf("message = %q, want Service Unavailable", body.Message)
	}
	if body.Data.Status != health.StatusDegraded {
		t.Fatalf("data.status = %q, want %q", body.Data.Status, health.StatusDegraded)
	}
	if body.Data.Checks["redis"].Status != health.StatusFailed {
		t.Fatalf("redis status = %q, want %q", body.Data.Checks["redis"].Status, health.StatusFailed)
	}
}
