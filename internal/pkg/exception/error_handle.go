package exception

import (
	"errors"
	"net/http"
	"strings"

	"github.com/kyh0703/portfoilo-media/internal/pkg/constant"
	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
	"go.uber.org/zap"
)

func NewHTTPErrorHandler(log *zap.Logger) func(http.ResponseWriter, *http.Request, error) {
	if log == nil {
		log = zap.NewNop()
	}

	return func(w http.ResponseWriter, r *http.Request, err error) {
		var appErr AppError
		if errors.As(err, &appErr) {
			_ = response.WriteError(w, appErr.Status, appErr.Message, appErr.Code)
			return
		}

		log.Error("unhandled request error",
			zap.Error(err),
			zap.String("request_id", strings.TrimSpace(r.Header.Get(constant.HeaderRequestID))),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		_ = response.WriteError(w, http.StatusInternalServerError, "internal server error", CodeInternalError)
	}
}
