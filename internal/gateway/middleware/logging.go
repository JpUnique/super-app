package middlewares

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/JpUnique/super-app/internal/gateway/logs"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.status = code
	lrw.ResponseWriter.WriteHeader(code)
}
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.status == 0 {
		lrw.status = http.StatusOK
	}
	n, err := lrw.ResponseWriter.Write(b)
	lrw.size += n
	return n, err
}

func Logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w}
			reqID := GetRequestID(r)

			remote := clientIP(r)
			l := logs.WithRequestID(logs.L().Gateway, reqID)

			next.ServeHTTP(lrw, r)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Int("status", lrw.status),
				slog.Int("bytes", lrw.size),
				slog.String("ip", remote),
				slog.String("ua", r.UserAgent()),
				slog.String("proto", r.Proto),
				slog.Duration("latency", time.Since(start)),
			}

			level := slog.LevelInfo
			switch {
			case lrw.status >= 500:
				level = slog.LevelError
			case lrw.status >= 400:
				level = slog.LevelWarn
			}
			l.LogAttrs(r.Context(), level, "http_request", attrs...)
		})
	}
}

func clientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return xr
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
