package ratelimit

type Config struct {
	Requests   int    // allowed number of requests
	WindowSec  int    // sliding window time (seconds)
	RedisURL   string // redis connection URL
	HeaderName string // client identifier, e.g., "X-API-KEY" or "Authorization"
}
