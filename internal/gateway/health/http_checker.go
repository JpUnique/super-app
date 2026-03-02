package health

import (
	"context"
	"fmt"
	"os"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/JpUnique/super-app/internal/gateway/loadbalancer"
)

// CheckResult describes the outcome of a single health check.
type CheckResult struct {
	Name       string `json:"name"`
	Healthy    bool   `json:"healthy"`
	Details    string `json:"details,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

// Checker is implemented by all health checkers.
type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// -------------------- Redis Checker --------------------

type RedisChecker struct {
	name   string
	client *redis.Client
}

func NewRedisChecker(name, redisURL string) (*RedisChecker, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &RedisChecker{
		name:   name,
		client: redis.NewClient(opt),
	}, nil
}

func (r *RedisChecker) Name() string { return r.name }

func (r *RedisChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	res := CheckResult{Name: r.name}

	if r.client == nil {
		res.Healthy = false
		res.Details = "redis client not initialized"
		return finalize(res, start)
	}

	if err := r.client.Ping(ctx).Err(); err != nil {
		res.Healthy = false
		res.Details = "redis ping failed: " + err.Error()
		return finalize(res, start)
	}

	res.Healthy = true
	res.Details = "redis ok"
	return finalize(res, start)
}

// -------------------- File/Key Checker --------------------

// FileKeyChecker validates that a PEM key file exists and is parseable.
// kind: "rsa_private" or "rsa_public"
type FileKeyChecker struct {
	name string
	path string
	kind string
}

func NewFileKeyChecker(name, path, kind string) *FileKeyChecker {
	return &FileKeyChecker{name: name, path: path, kind: kind}
}

func (k *FileKeyChecker) Name() string { return k.name }

func (k *FileKeyChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	res := CheckResult{Name: k.name}

	b, err := os.ReadFile(k.path)
	if err != nil {
		res.Healthy = false
		res.Details = fmt.Sprintf("read %s: %v", k.path, err)
		return finalize(res, start)
	}

	switch k.kind {
	case "rsa_private":
		if _, err := jwtlib.ParseRSAPrivateKeyFromPEM(b); err != nil {
			res.Healthy = false
			res.Details = "parse private key failed: " + err.Error()
			return finalize(res, start)
		}
	case "rsa_public":
		if _, err := jwtlib.ParseRSAPublicKeyFromPEM(b); err != nil {
			res.Healthy = false
			res.Details = "parse public key failed: " + err.Error()
			return finalize(res, start)
		}
	default:
		res.Healthy = false
		res.Details = "unknown key kind: " + k.kind
		return finalize(res, start)
	}

	res.Healthy = true
	res.Details = fmt.Sprintf("%s ok", k.kind)
	return finalize(res, start)
}

// -------------------- Registry Checker --------------------

type RegistryChecker struct {
	name string
	reg  *loadbalancer.Registry
	// requireHealthy ensures we only report healthy=true if there is at least one healthy backend.
	requireHealthy bool
}

func NewRegistryChecker(name string, reg *loadbalancer.Registry, requireHealthy bool) *RegistryChecker {
	return &RegistryChecker{name: name, reg: reg, requireHealthy: requireHealthy}
}

func (rc *RegistryChecker) Name() string { return rc.name }

func (rc *RegistryChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	res := CheckResult{Name: rc.name}

	all := rc.reg.All()
	healthy := rc.reg.Healthy()

	if len(all) == 0 {
		res.Healthy = !rc.requireHealthy // if not required, absence of backends is not fatal
		res.Details = "no backends registered"
		return finalize(res, start)
	}

	if len(healthy) == 0 && rc.requireHealthy {
		res.Healthy = false
		res.Details = fmt.Sprintf("registered=%d healthy=%d", len(all), len(healthy))
		return finalize(res, start)
	}

	res.Healthy = true
	res.Details = fmt.Sprintf("registered=%d healthy=%d", len(all), len(healthy))
	return finalize(res, start)
}

// -------------------- Utilities --------------------

func finalize(r CheckResult, start time.Time) CheckResult {
	r.DurationMS = time.Since(start).Milliseconds()
	return r
}

// RunAll runs all checks with a per-check timeout and returns overall status and results.
func RunAll(ctx context.Context, perCheckTimeout time.Duration, checks ...Checker) (bool, []CheckResult) {
	results := make([]CheckResult, len(checks))
	overall := true

	for i, c := range checks {
		// enforce per-check timeout
		ctxt, cancel := context.WithTimeout(ctx, perCheckTimeout)
		res := c.Check(ctxt)
		cancel()

		results[i] = res
		if !res.Healthy {
			overall = false
		}
	}
	return overall, results
}
