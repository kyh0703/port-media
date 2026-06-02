package exception

import (
	"errors"
	"net/http"

	"github.com/kyh0703/portfoilo-media/internal/pkg/response"
	"go.uber.org/zap"
)

func NewHTTPErrorHandler(log *zap.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		_ = r

		var appErr AppError
		if errors.As(err, &appErr) {
			_ = response.WriteError(w, appErr.Status, appErr.Message, appErr.Code)
			return
		}

		log.Error("unhandled request error", zap.Error(err))
		_ = response.WriteError(w, http.StatusInternalServerError, "internal server error", CodeInternalError)
	}
}
