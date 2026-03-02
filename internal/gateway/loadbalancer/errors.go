package loadbalancer

import "errors"

var (
	ErrNoHealthyBackends = errors.New("loadbalancer: no healthy backends available")
)
