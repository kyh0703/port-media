package httpx

import (
	"net/http"

	"go.uber.org/zap"
)

type RequestLogger struct {
	log *zap.Logger
}

func NewRequestLogger(log *zap.Logger) *RequestLogger {
	return &RequestLogger{log: log}
}

func (m *RequestLogger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		m.log.Info("http_request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rec.status),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
