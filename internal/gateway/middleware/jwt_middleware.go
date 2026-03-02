package middlewares

import (
	"context"
	"net/http"
	"strings"

	jwtauth "github.com/JpUnique/super-app/internal/gateway/jwt"
)

type jwtCtxKey string

const claimsKey jwtCtxKey = "jwt_claims"

// JWTMiddleware validates Bearer tokens using the provided TokenValidator.
// On success, it injects claims into the request context.
func JWTMiddleware(validator jwtauth.TokenValidator, headerKey string) func(http.Handler) http.Handler {
	if headerKey == "" {
		headerKey = "Authorization"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimSpace(r.Header.Get(headerKey))
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Support "Bearer <token>" and raw token
			token := raw
			if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
				token = strings.TrimSpace(raw[len("bearer "):])
				r.Header.Set(headerKey, token) // normalize for validator
			}

			claims, err := validator.ValidateToken(r)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(r *http.Request) *jwtauth.Claims {
	if v := r.Context().Value(claimsKey); v != nil {
		if c, ok := v.(*jwtauth.Claims); ok {
			return c
		}
	}
	return nil
}
