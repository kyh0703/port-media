package httpx

import "net/http"

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

type ErrorHandler func(http.ResponseWriter, *http.Request, error)

func WithErrorHandler(h ErrorHandlerFunc, handleError ErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			handleError(w, r, err)
		}
	}
}
