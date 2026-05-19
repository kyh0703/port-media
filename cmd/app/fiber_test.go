package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/kyh0703/portfoilo-media/configs"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"github.com/kyh0703/portfoilo-media/internal/core/handler"
	"github.com/kyh0703/portfoilo-media/internal/core/middleware"
	"go.uber.org/zap"
)

type testHandler struct{}

type appFakeSessionUsecase struct {
	stats sessiondto.RuntimeStatsResponse
	err   error
}

func (f appFakeSessionUsecase) CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.CreateSessionResponse{}, nil
}

func (f appFakeSessionUsecase) AcceptOffer(ctx context.Context, req sessiondto.AcceptOfferRequest) (sessiondto.AcceptOfferResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.AcceptOfferResponse{}, nil
}

func (f appFakeSessionUsecase) LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.LeaveParticipantResponse{}, nil
}

func (f appFakeSessionUsecase) EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.EndSessionResponse{}, nil
}

func (f appFakeSessionUsecase) GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error) {
	_ = ctx
	_ = req
	return sessiondto.GetSessionStatusResponse{}, false, nil
}

func (f appFakeSessionUsecase) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	_ = ctx
	if f.err != nil {
		return sessiondto.RuntimeStatsResponse{}, f.err
	}
	return f.stats, nil
}

func (f appFakeSessionUsecase) GetHealth(ctx context.Context) error {
	_ = ctx
	return nil
}

func (testHandler) Table() []handler.Mapper {
	return []handler.Mapper{
		{
			Method: fiber.MethodPost,
			Path:   "/sessions/:sessionId/offer",
			Handler: []fiber.Handler{func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNoContent)
			}},
		},
	}
}

func TestNewFiberHandlesCORSPreflightForOffer(t *testing.T) {
	app := NewFiber(FiberParams{
		Config: &configs.Config{
			App: configs.AppConfig{Name: "test"},
			Server: configs.ServerConfig{
				CORS: configs.CORSConfig{
					AllowedOrigins: []string{"http://localhost:3000"},
					AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
					AllowedHeaders: []string{"Authorization", "Content-Type"},
					ExposeHeaders:  []string{"X-Room-Id", "X-Participant-Id"},
				},
			},
		},
		Logger:        zap.NewNop(),
		RequestLogger: middleware.NewRequestLogger(zap.NewNop()),
		Handlers:      []handler.Handler{testHandler{}},
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/sessions/session-1/offer", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNoContent)
	}
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want http://localhost:3000", got)
	}
	if got := res.Header.Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("Access-Control-Allow-Methods is empty")
	}
	if got := res.Header.Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("Access-Control-Allow-Headers is empty")
	}
}

func TestNewFiberExposesOfferResponseHeaders(t *testing.T) {
	app := NewFiber(FiberParams{
		Config: &configs.Config{
			App: configs.AppConfig{Name: "test"},
			Server: configs.ServerConfig{
				CORS: configs.CORSConfig{
					AllowedOrigins: []string{"http://localhost:3000"},
					AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
					AllowedHeaders: []string{"Authorization", "Content-Type"},
					ExposeHeaders:  []string{"X-Room-Id", "X-Participant-Id"},
				},
			},
		},
		Logger:        zap.NewNop(),
		RequestLogger: middleware.NewRequestLogger(zap.NewNop()),
		Handlers:      []handler.Handler{testHandler{}},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/session-1/offer", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNoContent)
	}
	exposed := res.Header.Values("Access-Control-Expose-Headers")
	if len(exposed) == 0 {
		t.Fatal("Access-Control-Expose-Headers is empty")
	}
	joined := strings.Join(exposed, ",")
	if !strings.Contains(joined, "X-Room-Id") {
		t.Fatalf("Access-Control-Expose-Headers = %q, want X-Room-Id", joined)
	}
	if !strings.Contains(joined, "X-Participant-Id") {
		t.Fatalf("Access-Control-Expose-Headers = %q, want X-Participant-Id", joined)
	}
}

func TestNewFiberExposesPrometheusMetricsWhenEnabled(t *testing.T) {
	app := NewFiber(FiberParams{
		Config: &configs.Config{
			App: configs.AppConfig{Name: "test"},
			Server: configs.ServerConfig{
				CORS: configs.CORSConfig{
					AllowedOrigins: []string{"*"},
					AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
					AllowedHeaders: []string{"Authorization", "Content-Type"},
				},
			},
			Observability: configs.ObservabilityConfig{MetricsEnabled: true},
		},
		Logger:        zap.NewNop(),
		RequestLogger: middleware.NewRequestLogger(zap.NewNop()),
		Handlers:      []handler.Handler{testHandler{}},
		Session: appFakeSessionUsecase{
			stats: sessiondto.RuntimeStatsResponse{
				Rooms:        1,
				Sessions:     1,
				Participants: 2,
				Tracks:       3,
				ByStatus:     map[string]int{"active": 1},
			},
		},
	})

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if got := res.Header.Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), "dubu_media_participants 2") {
		t.Fatalf("metrics body missing participant gauge:\n%s", string(body))
	}
}

func TestNewFiberDoesNotExposePrometheusMetricsWhenDisabled(t *testing.T) {
	app := NewFiber(FiberParams{
		Config: &configs.Config{
			App:           configs.AppConfig{Name: "test"},
			Server:        configs.ServerConfig{},
			Observability: configs.ObservabilityConfig{MetricsEnabled: false},
		},
		Logger:        zap.NewNop(),
		RequestLogger: middleware.NewRequestLogger(zap.NewNop()),
		Handlers:      []handler.Handler{testHandler{}},
		Session:       appFakeSessionUsecase{},
	})

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestNewFiberReturnsCommonErrorWhenPrometheusMetricsFail(t *testing.T) {
	app := NewFiber(FiberParams{
		Config: &configs.Config{
			App:           configs.AppConfig{Name: "test"},
			Server:        configs.ServerConfig{},
			Observability: configs.ObservabilityConfig{MetricsEnabled: true},
		},
		Logger:        zap.NewNop(),
		RequestLogger: middleware.NewRequestLogger(zap.NewNop()),
		Handlers:      []handler.Handler{testHandler{}},
		Session:       appFakeSessionUsecase{err: errors.New("stats unavailable")},
	})

	res, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusInternalServerError)
	}

	var body struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if body.StatusCode != http.StatusInternalServerError {
		t.Fatalf("statusCode = %d, want %d", body.StatusCode, http.StatusInternalServerError)
	}
}
