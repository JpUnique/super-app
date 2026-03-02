package versioning

import (
	"fmt"
	"net/http"
)

// Example mapping for v2 routes (simple ServeMux). Replace with your real handlers.
func V2Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v2/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"version":"v2","message":"pong"}`)
	})

	// TODO: register v2 endpoints (breaking changes, new resources, etc.)

	return mux
}
