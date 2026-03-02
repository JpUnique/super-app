package proxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/JpUnique/super-app/internal/gateway/loadbalancer"
	"github.com/JpUnique/super-app/internal/gateway/logs"
)

type Balancer interface {
	Next() (*loadbalancer.Backend, error)
}

type Proxy struct {
	reg     *loadbalancer.Registry
	pickers map[string]Balancer // serviceName -> strategy (RR/LC)
	router  *StaticServiceRouter
	client  *http.Client
}

func New(reg *loadbalancer.Registry, router *StaticServiceRouter, transport http.RoundTripper) *Proxy {
	if transport == nil {
		transport = &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	return &Proxy{
		reg:     reg,
		router:  router,
		client:  &http.Client{Transport: transport},
		pickers: make(map[string]Balancer),
	}
}

// Register a balancing strategy for a service.
func (p *Proxy) SetBalancer(service string, b Balancer) {
	p.pickers[service] = b
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Resolve service by path
	service, subpath, ok := p.router.Resolve(r)
	if !ok {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	bal, ok := p.pickers[service]
	if !ok {
		http.Error(w, "no balancer for service", http.StatusBadGateway)
		return
	}

	backend, err := bal.Next()
	if err != nil {
		http.Error(w, "no healthy backends", http.StatusBadGateway)
		return
	}

	// Build upstream URL
	base, _ := url.Parse(backend.URL)
	// Preserve query
	target := *base
	if subpath != "" {
		target.Path = singleJoiningSlash(base.Path, subpath)
	} else {
		target.Path = singleJoiningSlash(base.Path, r.URL.Path)
	}
	target.RawQuery = r.URL.RawQuery

	// Clone request
	req, _ := http.NewRequestWithContext(r.Context(), r.Method, target.String(), r.Body)
	req.Header = r.Header.Clone()

	// Mark active conn
	backend.IncActive()
	defer backend.DecActive()

	// Forward
	resp, err := p.client.Do(req)
	if err != nil {
		logs.L().LB.Error("proxy_upstream_error", "error", err.Error(), "backend", backend.Name)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
