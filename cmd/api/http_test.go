package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http"
	"github.com/kyh0703/portfoilo-media/internal/adapter/in/http/middleware"
	sessiondto "github.com/kyh0703/portfoilo-media/internal/core/dto/session"
	"go.uber.org/zap"
)

type testHandler struct{}

type panicHandler struct{}

type appFakeSessionUsecases struct {
	stats sessiondto.RuntimeStatsResponse
	err   error
}

func (f appFakeSessionUsecases) CreateSession(ctx context.Context, req sessiondto.CreateSessionRequest) (sessiondto.CreateSessionResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.CreateSessionResponse{}, nil
}

func (f appFakeSessionUsecases) JoinSession(ctx context.Context, req sessiondto.JoinSessionCommand) (sessiondto.JoinSessionResult, error) {
	_ = ctx
	_ = req
	return sessiondto.JoinSessionResult{}, nil
}

func (f appFakeSessionUsecases) LeaveParticipant(ctx context.Context, req sessiondto.LeaveParticipantRequest) (sessiondto.LeaveParticipantResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.LeaveParticipantResponse{}, nil
}

func (f appFakeSessionUsecases) EndSession(ctx context.Context, req sessiondto.EndSessionRequest) (sessiondto.EndSessionResponse, error) {
	_ = ctx
	_ = req
	return sessiondto.EndSessionResponse{}, nil
}

func (f appFakeSessionUsecases) GetSessionStatus(ctx context.Context, req sessiondto.GetSessionStatusRequest) (sessiondto.GetSessionStatusResponse, bool, error) {
	_ = ctx
	_ = req
	return sessiondto.GetSessionStatusResponse{}, false, nil
}

func (f appFakeSessionUsecases) GetRuntimeStats(ctx context.Context) (sessiondto.RuntimeStatsResponse, error) {
	_ = ctx
	if f.err != nil {
		return sessiondto.RuntimeStatsResponse{}, f.err
	}
	return f.stats, nil
}

func (f appFakeSessionUsecases) GetHealth(ctx context.Context) error {
	_ = ctx
	return nil
}

func (testHandler) Table() []handler.Mapper {
	return []handler.Mapper{
		{
			Method: http.MethodPost,
			Path:   "/sessions/{sessionId}/join",
			Handler: func(w http.ResponseWriter, r *http.Request) error {
				_ = r
				w.WriteHeader(http.StatusNoContent)
				return nil
			},
		},
	}
}

func (panicHandler) Table() []handler.Mapper {
	return []handler.Mapper{
		{
			Method: http.MethodGet,
			Path:   "/panic",
			Handler: func(w http.ResponseWriter, r *http.Request) error {
				_ = w
				_ = r
				panic("boom")
			},
		},
	}
}

func TestNewHTTPHandlerHandlesCORSPreflightForJoin(t *testing.T) {
	app := NewHTTPHandler(HTTPParams{
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

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/sessions/session-1/join", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")

	res := serve(app, req)
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

func TestNewHTTPHandlerExposesJoinResponseHeaders(t *testing.T) {
	app := NewHTTPHandler(HTTPParams{
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/session-1/join", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	res := serve(app, req)
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

func TestNewHTTPHandlerRecoversPanics(t *testing.T) {
	app := NewHTTPHandler(HTTPParams{
		Config: &configs.Config{
			App:    configs.AppConfig{Name: "test"},
			Server: configs.ServerConfig{},
		},
		Logger:        zap.NewNop(),
		RequestLogger: middleware.NewRequestLogger(zap.NewNop()),
		Recover:       middleware.NewRecoverMiddleware(zap.NewNop()),
		Handlers:      []handler.Handler{panicHandler{}},
	})

	res := serve(app, httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil))
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusInternalServerError)
	}
	if got := res.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestNewHTTPHandlerDoesNotExposeMetricsEndpoint(t *testing.T) {
	app := NewHTTPHandler(HTTPParams{
		Config: &configs.Config{
			App:    configs.AppConfig{Name: "test"},
			Server: configs.ServerConfig{},
		},
		Logger:        zap.NewNop(),
		RequestLogger: middleware.NewRequestLogger(zap.NewNop()),
		Handlers:      []handler.Handler{testHandler{}},
	})

	res := serve(app, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func serve(app http.Handler, req *http.Request) *http.Response {
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec.Result()
}
