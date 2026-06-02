package httpx

import (
	"net/http"
	"strings"

	"github.com/kyh0703/portfoilo-media/configs"
)

func CORS(cfg configs.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeCORSHeaders(w, r, cfg)
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeCORSHeaders(w http.ResponseWriter, r *http.Request, cfg configs.CORSConfig) {
	if origin := allowedOrigin(r.Header.Get("Origin"), cfg.AllowedOrigins); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	if len(cfg.AllowedMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ","))
	}
	if len(cfg.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ","))
	}
	if len(cfg.ExposeHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ","))
	}
}

func allowedOrigin(origin string, allowed []string) string {
	if origin == "" {
		return ""
	}
	for _, candidate := range allowed {
		if candidate == "*" || candidate == origin {
			return candidate
		}
	}
	return ""
}
