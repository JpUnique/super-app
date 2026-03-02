package router

import (
	"net/http"
)

// NewRouter wraps the given root handler with the provided middlewares,
// in the same order as they are passed.
func NewRouter(root http.Handler, middlewaresList ...func(http.Handler) http.Handler) http.Handler {
	var handler http.Handler = root

	// Apply middlewares in order
	for _, m := range middlewaresList {
		handler = m(handler)
	}

	return handler
}
