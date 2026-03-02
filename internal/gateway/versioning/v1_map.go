package versioning

import (
	"fmt"
	"net/http"
)

// Example mapping for v1 routes (simple ServeMux). Replace with your real handlers.
func V1Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"version":"v1","message":"pong"}`)
	})

	// TODO: register v1 endpoints for payments, rides, food, etc.

	return mux
}
