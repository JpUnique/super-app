package ratelimit

import (
	"net/http"
	"strings"
)

func Middleware(limiter Limiter, headerKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			clientID := strings.TrimSpace(r.Header.Get(headerKey))
			if clientID == "" {
				clientID = r.RemoteAddr // fallback
			}

			err := limiter.Allow(r.Context(), clientID)
			if err == ErrRateLimitExceeded {
				http.Error(w, "Too many requests", http.StatusTooManyRequests)
				return
			}

			if err != nil {
				http.Error(w, "Rate limiter internal error", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
