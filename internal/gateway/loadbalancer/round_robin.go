package loadbalancer

import "sync/atomic"

// RoundRobin selects backends in a circular order (only among healthy ones).
type RoundRobin struct {
	reg *Registry
	idx uint32
}

func NewRoundRobin(reg *Registry) *RoundRobin {
	return &RoundRobin{reg: reg}
}

func (rr *RoundRobin) Next() (*Backend, error) {
	healthy := rr.reg.Healthy()
	n := len(healthy)
	if n == 0 {
		return nil, ErrNoHealthyBackends
	}
	i := atomic.AddUint32(&rr.idx, 1)
	return healthy[int(i-1)%n], nil
}
