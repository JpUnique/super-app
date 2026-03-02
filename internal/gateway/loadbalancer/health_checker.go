package loadbalancer

import (
	"context"
	"net"
	"net/http"
	"time"
)

// HealthChecker probes backends and updates health flags.
type HealthChecker struct {
	reg      *Registry
	client   *http.Client
	interval time.Duration
	path     string
}

type HealthCheckerConfig struct {
	Interval time.Duration // e.g., 5 * time.Second
	Timeout  time.Duration // e.g., 2 * time.Second
	Path     string        // e.g., "/health"
}

func NewHealthChecker(reg *Registry, cfg HealthCheckerConfig) *HealthChecker {
	if cfg.Path == "" {
		cfg.Path = "/health"
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}
	return &HealthChecker{
		reg:  reg,
		path: cfg.Path,
		client: &http.Client{Timeout: cfg.Timeout, Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: cfg.Timeout}).DialContext,
		}},
		interval: cfg.Interval,
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// one immediate pass
	hc.checkAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.checkAll(ctx)
		}
	}
}

func (hc *HealthChecker) checkAll(ctx context.Context) {
	for _, b := range hc.reg.All() {
		u := b.URL + hc.path
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := hc.client.Do(req)
		if err != nil {
			b.SetHealthy(false)
			continue
		}
		_ = resp.Body.Close()
		b.SetHealthy(resp.StatusCode >= 200 && resp.StatusCode < 300)
	}
}
