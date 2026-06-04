package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestRecoverMiddlewareRecoversPanic(t *testing.T) {
	var reported any
	var stack []byte
	recoverMiddleware := NewRecoverMiddlewareWithReporter(
		zap.NewNop(),
		PanicReporterFunc(func(ctx context.Context, recovered any, recoveredStack []byte) {
			_ = ctx
			reported = recovered
			stack = recoveredStack
		}),
	)
	handler := recoverMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = w
		_ = r
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if reported != "boom" {
		t.Fatalf("reported panic = %v, want boom", reported)
	}
	if len(stack) == 0 {
		t.Fatal("reported stack is empty")
	}
}

func TestRecoverMiddlewareDoesNotWriteAfterHeader(t *testing.T) {
	recoverMiddleware := NewRecoverMiddleware(zap.NewNop())
	handler := recoverMiddleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.WriteHeader(http.StatusAccepted)
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Body.String(); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
}
