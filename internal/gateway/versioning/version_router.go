package versioning

import (
	"net/http"
	"strings"
	"time"
)

type VersionRouter struct {
	defaultVersion string
	routes         map[string]http.Handler
	deprecations   map[string]DeprecationRule
}

func NewVersionRouter(defaultVersion string) *VersionRouter {
	return &VersionRouter{
		defaultVersion: defaultVersion,
		routes:         make(map[string]http.Handler),
		deprecations:   make(map[string]DeprecationRule),
	}
}

func (vr *VersionRouter) AddRoute(version string, h http.Handler) {
	vr.routes[strings.ToLower(version)] = h
}

func (vr *VersionRouter) SetDeprecation(rule DeprecationRule) {
	vr.deprecations[strings.ToLower(rule.Version)] = rule
}

// Resolve order: URL prefix (/api/vX), "Accept-Version" header, query ?version=vX, else default.
func (vr *VersionRouter) resolveVersion(r *http.Request) string {
	// URL path like /api/v1/... or /v1/...
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) > 0 && strings.HasPrefix(strings.ToLower(parts[0]), "v") {
		return strings.ToLower(parts[0])
	}

	// Header
	if h := strings.TrimSpace(r.Header.Get("Accept-Version")); h != "" {
		return strings.ToLower(h)
	}

	// Query
	if q := strings.TrimSpace(r.URL.Query().Get("version")); q != "" {
		return strings.ToLower(q)
	}

	return strings.ToLower(vr.defaultVersion)
}

func (vr *VersionRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v := vr.resolveVersion(r)
	h, ok := vr.routes[v]
	if !ok {
		http.Error(w, "unsupported API version", http.StatusNotFound)
		return
	}

	// Add deprecation headers if applicable (RFC 8594 style)
	if rule, ok := vr.deprecations[v]; ok {
		// If sunset in the future, include Deprecation header + Sunset date
		if !rule.Sunset.IsZero() {
			// Deprecation header: "true" or date
			// We'll include a message via "Link" or a custom header
			w.Header().Set("Deprecation", "true")
			w.Header().Set("Sunset", rule.Sunset.UTC().Format(time.RFC1123))
			if rule.Message != "" {
				w.Header().Set("X-API-Deprecation-Info", rule.Message)
			}
		}
	}

	h.ServeHTTP(w, r)
}
