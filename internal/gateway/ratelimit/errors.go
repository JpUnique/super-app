package ratelimit

import "errors"

var (
	ErrRateLimitExceeded = errors.New("ratelimit: request limit exceeded")
	ErrRedisFailure      = errors.New("ratelimit: redis operation failed")
)
