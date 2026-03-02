package ratelimit

import (
	"context"
	"fmt"
	"time"
)

type Limiter interface {
	Allow(ctx context.Context, identifier string) error
}

type rateLimiter struct {
	cfg   Config
	store Store
}

func NewRateLimiter(cfg Config, store Store) Limiter {
	return &rateLimiter{
		cfg:   cfg,
		store: store,
	}
}

func (l *rateLimiter) Allow(ctx context.Context, identifier string) error {
	if identifier == "" {
		identifier = "anonymous"
	}

	window := time.Duration(l.cfg.WindowSec) * time.Second

	count, err := l.store.AddRequest(ctx, fmt.Sprintf("rl:%s", identifier), window)
	if err != nil {
		return ErrRedisFailure
	}

	if count > l.cfg.Requests {
		return ErrRateLimitExceeded
	}

	return nil
}
