package loadbalancer

// LeastConn picks the healthy backend with the fewest active connections.
type LeastConn struct {
	reg *Registry
}

func NewLeastConn(reg *Registry) *LeastConn {
	return &LeastConn{reg: reg}
}

func (lc *LeastConn) Next() (*Backend, error) {
	healthy := lc.reg.Healthy()
	if len(healthy) == 0 {
		return nil, ErrNoHealthyBackends
	}
	var best *Backend
	for _, b := range healthy {
		if best == nil || b.Active() < best.Active() {
			best = b
		}
	}
	return best, nil
}
