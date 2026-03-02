package logs

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Subsystems (filenames are <name>.log in this folder)
const (
	SubsystemGateway    = "gateway"
	SubsystemAuth       = "auth"
	SubsystemJWT        = "jwt"
	SubsystemRateLimit  = "ratelimit"
	SubsystemLB         = "lb"
	SubsystemVersioning = "versioning"
)

type LoggerSet struct {
	Gateway    *slog.Logger
	Auth       *slog.Logger
	JWT        *slog.Logger
	RateLimit  *slog.Logger
	LB         *slog.Logger
	Versioning *slog.Logger
}

var (
	once    sync.Once
	global  *LoggerSet
	initErr error

	baseDir = "./internal/gateway/logs" // adjust if needed
)

// Init initializes subsystem loggers. Safe to call multiple times.
func Init() (*LoggerSet, error) {
	once.Do(func() {
		if err := os.MkdirAll(baseDir, 0o755); err != nil {
			initErr = err
			return
		}

		format := strings.ToLower(os.Getenv("LOG_FORMAT")) // "json" or "console"
		if format == "" {
			format = "json"
		}
		level := parseLevel(os.Getenv("LOG_LEVEL")) // default info

		makeLogger := func(subsystem, filename string) (*slog.Logger, error) {
			w, err := NewFileMultiWriter(filepath.Join(baseDir, filename))
			if err != nil {
				return nil, err
			}

			var h slog.Handler
			if format == "console" {
				// Human-friendly console handler (time is compacted)
				h = slog.NewTextHandler(w, &slog.HandlerOptions{
					Level: level,
					ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
						// compact time
						if a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
							t := a.Value.Time()
							a.Value = slog.StringValue(t.UTC().Format(time.RFC3339Nano))
						}
						return a
					},
				})
			} else {
				// JSON handler for ingestion in ELK/Datadog/Grafana
				h = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
			}

			return slog.New(h).With(
				slog.String("subsystem", subsystem),
			), nil
		}

		gw, err := makeLogger(SubsystemGateway, "gateway.log")
		if err != nil {
			initErr = err
			return
		}
		au, err := makeLogger(SubsystemAuth, "auth.log")
		if err != nil {
			initErr = err
			return
		}
		jt, err := makeLogger(SubsystemJWT, "jwt.log")
		if err != nil {
			initErr = err
			return
		}
		rl, err := makeLogger(SubsystemRateLimit, "ratelimit.log")
		if err != nil {
			initErr = err
			return
		}
		lb, err := makeLogger(SubsystemLB, "lb.log")
		if err != nil {
			initErr = err
			return
		}
		vr, err := makeLogger(SubsystemVersioning, "versioning.log")
		if err != nil {
			initErr = err
			return
		}

		global = &LoggerSet{
			Gateway:    gw,
			Auth:       au,
			JWT:        jt,
			RateLimit:  rl,
			LB:         lb,
			Versioning: vr,
		}
	})

	return global, initErr
}

// MustInit panics if Init fails.
func MustInit() *LoggerSet {
	l, err := Init()
	if err != nil {
		panic(err)
	}
	return l
}

func L() *LoggerSet { return global }

// WithRequestID adds req_id attribute to a logger.
func WithRequestID(l *slog.Logger, reqID string) *slog.Logger {
	if reqID == "" {
		return l
	}
	return l.With(slog.String("req_id", reqID))
}

// -- helpers --

func parseLevel(s string) slog.Leveler {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
