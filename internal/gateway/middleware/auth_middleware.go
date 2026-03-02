package middlewares

import (
	"net/http"
	"strings"

	"github.com/JpUnique/super-app/internal/gateway/auth"
)

// AuthMiddleware validates API keys (for service-to-service or admin calls).
// It uses the provided AuthService; if no API key header is present, it passes through.
func AuthMiddleware(svc auth.AuthService, headerName string) func(http.Handler) http.Handler {
	if headerName == "" {
		headerName = "X-API-KEY"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get(headerName))
			if key == "" {
				// No API key — let other middlewares (e.g., JWT) handle user auth.
				next.ServeHTTP(w, r)
				return
			}
			if err := svc.AuthenticateAPIKey(r); err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
