package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// Options control how health endpoints behave.
type Options struct {
	PerCheckTimeout time.Duration
	ShowDetails     bool
}

type Handler struct {
	checks   []Checker
	options  Options
	draining atomic.Bool
}

func NewHandler(checks []Checker, opts Options) *Handler {
	if opts.PerCheckTimeout <= 0 {
		opts.PerCheckTimeout = 1 * time.Second
	}
	return &Handler{checks: checks, options: opts}
}

func (h *Handler) SetDraining(v bool) {
	h.draining.Store(v)
}

// Liveness: process is up. Keep minimal: always 200 to avoid restart loops.
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "alive",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	})
}

// Readiness: dependencies and not draining
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.draining.Load() {
		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "draining",
			"time":   time.Now().UTC().Format(time.RFC3339Nano),
		})
		return
	}

	ctx := r.Context()
	ok, results := RunAll(ctx, h.options.PerCheckTimeout, h.checks...)

	payload := map[string]any{
		"status":  ternary(ok, "ready", "not_ready"),
		"time":    time.Now().UTC().Format(time.RFC3339Nano),
		"summary": summarize(results),
	}
	if h.options.ShowDetails {
		payload["components"] = results
	}

	code := http.StatusOK
	if !ok {
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, code, payload)
}

func summarize(results []CheckResult) map[string]any {
	total, healthy := 0, 0
	for _, r := range results {
		total++
		if r.Healthy {
			healthy++
		}
	}
	return map[string]any{"total": total, "healthy": healthy}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}
