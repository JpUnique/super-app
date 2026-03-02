package loadbalancer

import (
	"sync"
	"sync/atomic"
)

// Backend represents a single upstream service instance.
type Backend struct {
	Name        string // logical name
	URL         string // http://host:port
	healthy     atomic.Bool
	activeConns atomic.Int64
}

func (b *Backend) IsHealthy() bool   { return b.healthy.Load() }
func (b *Backend) SetHealthy(h bool) { b.healthy.Store(h) }
func (b *Backend) Active() int64     { return b.activeConns.Load() }
func (b *Backend) IncActive()        { b.activeConns.Add(1) }
func (b *Backend) DecActive()        { b.activeConns.Add(-1) }

// Registry keeps the set of known backends (thread-safe).
type Registry struct {
	mu       sync.RWMutex
	backends []*Backend
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) Register(b *Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends = append(r.backends, b)
}

func (r *Registry) Deregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.backends[:0]
	for _, b := range r.backends {
		if b.Name != name {
			out = append(out, b)
		}
	}
	r.backends = out
}

func (r *Registry) All() []*Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make([]*Backend, len(r.backends))
	copy(cp, r.backends)
	return cp
}

func (r *Registry) Healthy() []*Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var healthy []*Backend
	for _, b := range r.backends {
		if b.IsHealthy() {
			healthy = append(healthy, b)
		}
	}
	return healthy
}
