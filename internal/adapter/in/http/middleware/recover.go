package middleware

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/kyh0703/portfoilo-media/internal/pkg/exception"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
	"go.uber.org/zap"
)

type PanicReporter interface {
	ReportPanic(ctx context.Context, recovered any, stack []byte)
}

type PanicReporterFunc func(context.Context, any, []byte)

func (f PanicReporterFunc) ReportPanic(ctx context.Context, recovered any, stack []byte) {
	f(ctx, recovered, stack)
}

type RecoverMiddleware struct {
	log      *zap.Logger
	reporter PanicReporter
}

func NewRecoverMiddleware(log *zap.Logger) *RecoverMiddleware {
	return NewRecoverMiddlewareWithReporter(log, nil)
}

func NewRecoverMiddlewareWithReporter(log *zap.Logger, reporter PanicReporter) *RecoverMiddleware {
	if log == nil {
		log = zap.NewNop()
	}
	return &RecoverMiddleware{
		log:      log,
		reporter: reporter,
	}
}

func (m *RecoverMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &recoverRecorder{ResponseWriter: w}
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			stack := debug.Stack()
			if m.reporter != nil {
				m.reporter.ReportPanic(r.Context(), recovered, stack)
			}

			m.log.Error("http_panic_recovered",
				zap.Any("panic", recovered),
				zap.ByteString("stack", stack),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)

			if rec.wroteHeader {
				return
			}
			_ = response.WriteError(w, http.StatusInternalServerError, "internal server error", exception.CodeInternalError)
		}()

		next.ServeHTTP(rec, r)
	})
}

type recoverRecorder struct {
	http.ResponseWriter
	wroteHeader bool
}

func (r *recoverRecorder) WriteHeader(status int) {
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *recoverRecorder) Write(body []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(body)
}
