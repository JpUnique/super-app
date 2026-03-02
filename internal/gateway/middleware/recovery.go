package middlewares

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/JpUnique/super-app/internal/gateway/logs"
)

func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					l := logs.WithRequestID(logs.L().Gateway, GetRequestID(r))
					l.Error("panic recovered",
						slog.Any("error", rec),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
						slog.String("ua", r.UserAgent()),
						slog.String("stack", string(debug.Stack())),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
