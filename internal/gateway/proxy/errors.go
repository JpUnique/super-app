package proxy

import "errors"

var (
	ErrNoRoute = errors.New("proxy: no matching route")
)
