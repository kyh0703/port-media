package middleware

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kyh0703/portfoilo-media/internal/pkg/constant"
	"go.uber.org/zap"
)

type RequestLogger struct {
	log *zap.Logger
}

func NewRequestLogger(log *zap.Logger) *RequestLogger {
	if log == nil {
		log = zap.NewNop()
	}
	return &RequestLogger{log: log}
}

func (m *RequestLogger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := ensureRequestID(w, r)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_ip", remoteIP(r)),
			zap.String("user_agent", r.UserAgent()),
			zap.Int("status", rec.status),
			zap.Int("bytes", rec.bytes),
			zap.Duration("duration", time.Since(started)),
		}
		if r.URL.RawQuery != "" {
			fields = append(fields, zap.String("query", r.URL.RawQuery))
		}
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			fields = append(fields, zap.String("forwarded_for", forwardedFor))
		}

		switch {
		case rec.status >= http.StatusInternalServerError:
			m.log.Error("http_request", fields...)
		case rec.status >= http.StatusBadRequest:
			m.log.Warn("http_request", fields...)
		default:
			m.log.Info("http_request", fields...)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	written, err := r.ResponseWriter.Write(body)
	r.bytes += written
	return written, err
}

func ensureRequestID(w http.ResponseWriter, r *http.Request) string {
	requestID := strings.TrimSpace(r.Header.Get(constant.HeaderRequestID))
	if requestID == "" {
		requestID = uuid.NewString()
		r.Header.Set(constant.HeaderRequestID, requestID)
	}
	w.Header().Set(constant.HeaderRequestID, requestID)
	return requestID
}

func requestIDFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(constant.HeaderRequestID))
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
