package proxy

import (
	"net/http"
	"strings"
)

type Route struct {
	Prefix      string // e.g., "/v1/payments/"
	ServiceName string // e.g., "payments"
	StripPrefix bool   // whether to strip Prefix when forwarding
}

type StaticServiceRouter struct {
	routes []Route
}

func NewStaticServiceRouter(routes []Route) *StaticServiceRouter {
	return &StaticServiceRouter{routes: routes}
}

// Resolve finds the first matching route by longest prefix.
func (sr *StaticServiceRouter) Resolve(r *http.Request) (serviceName, upstreamPath string, ok bool) {
	path := r.URL.Path
	var matched *Route
	for i := range sr.routes {
		rt := &sr.routes[i]
		if strings.HasPrefix(path, rt.Prefix) {
			if matched == nil || len(rt.Prefix) > len(matched.Prefix) {
				matched = rt
			}
		}
	}
	if matched == nil {
		return "", "", false
	}
	if matched.StripPrefix {
		return matched.ServiceName, strings.TrimPrefix(path, matched.Prefix), true
	}
	return matched.ServiceName, path, true
}
