package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyh0703/portfoilo-media/internal/pkg/constant"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestLoggerLogsHTTPContext(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	handler := NewRequestLogger(zap.New(core)).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/health?verbose=true", nil)
	req.RemoteAddr = "203.0.113.10:4321"
	req.Header.Set(constant.HeaderRequestID, "request-1")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get(constant.HeaderRequestID); got != "request-1" {
		t.Fatalf("%s = %q, want request-1", constant.HeaderRequestID, got)
	}
	entries := observed.FilterMessage("http_request").All()
	if len(entries) != 1 {
		t.Fatalf("http_request logs = %d, want 1", len(entries))
	}
	if entries[0].Level != zapcore.InfoLevel {
		t.Fatalf("log level = %s, want info", entries[0].Level)
	}
	fields := entries[0].ContextMap()
	wantFields := map[string]any{
		"request_id":    "request-1",
		"method":        http.MethodGet,
		"path":          "/health",
		"query":         "verbose=true",
		"remote_ip":     "203.0.113.10",
		"user_agent":    "test-agent",
		"forwarded_for": "198.51.100.20",
		"status":        int64(http.StatusOK),
		"bytes":         int64(2),
	}
	for key, want := range wantFields {
		if got := fields[key]; got != want {
			t.Fatalf("%s = %v, want %v", key, got, want)
		}
	}
	if _, ok := fields["duration"]; !ok {
		t.Fatal("duration field is missing")
	}
}

func TestRequestLoggerGeneratesRequestID(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	handler := NewRequestLogger(zap.New(core)).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = w
		_ = r
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	requestID := rec.Header().Get(constant.HeaderRequestID)
	if requestID == "" {
		t.Fatalf("%s is empty", constant.HeaderRequestID)
	}
	fields := observed.FilterMessage("http_request").All()[0].ContextMap()
	if fields["request_id"] != requestID {
		t.Fatalf("request_id log field = %v, want %s", fields["request_id"], requestID)
	}
}

func TestRequestLoggerUsesErrorLevelForServerErrors(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	handler := NewRequestLogger(zap.New(core)).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.WriteHeader(http.StatusInternalServerError)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/broken", nil))

	entries := observed.FilterMessage("http_request").All()
	if len(entries) != 1 {
		t.Fatalf("http_request logs = %d, want 1", len(entries))
	}
	if entries[0].Level != zapcore.ErrorLevel {
		t.Fatalf("log level = %s, want error", entries[0].Level)
	}
}
