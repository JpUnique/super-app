package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/JpUnique/super-app/internal/gateway/auth"
	"github.com/JpUnique/super-app/internal/gateway/config"
	"github.com/JpUnique/super-app/internal/gateway/health"
	"github.com/JpUnique/super-app/internal/gateway/jwt"
	"github.com/JpUnique/super-app/internal/gateway/loadbalancer"
	"github.com/JpUnique/super-app/internal/gateway/logs"
	middlewares "github.com/JpUnique/super-app/internal/gateway/middleware"
	"github.com/JpUnique/super-app/internal/gateway/proxy"
	"github.com/JpUnique/super-app/internal/gateway/ratelimit"
	"github.com/JpUnique/super-app/internal/gateway/router"
)

func main() {
	// 1) Config
	cfg := config.Load()

	// 2) Logging
	logs.MustInit()
	logs.L().Gateway.Info("gateway_start",
		slog.String("port", cfg.Port),
	)

	// 3) Registry + health checker
	reg := loadbalancer.NewRegistry()
	reg.Register(&loadbalancer.Backend{Name: "orders-1", URL: "http://localhost:8081"})
	reg.Register(&loadbalancer.Backend{Name: "orders-2", URL: "http://localhost:8082"})

	// 3b) Load proxy routing rules from YAML file
	sr, strategies, err := proxy.LoadRoutesFromYAML(cfg.RoutesFile)
	if err != nil {
		log.Fatal("could not load routes:", err)
	}

	// 3c) Balancers per service (RR/LC); you can mix & match
	rr := loadbalancer.NewRoundRobin(reg)
	lc := loadbalancer.NewLeastConn(reg)

	// 3d) Create proxy and assign strategies
	px := proxy.New(reg, sr, nil) // default transport; customize later if needed
	for service, strategy := range strategies {
		if strategy == "least_conn" {
			px.SetBalancer(service, lc)
		} else {
			px.SetBalancer(service, rr) // default to round_robin
		}
	}

	hc := loadbalancer.NewHealthChecker(reg, loadbalancer.HealthCheckerConfig{
		Interval: 5 * time.Second,
		Timeout:  2 * time.Second,
		Path:     "/health",
	})
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	go hc.Start(bgCtx)
	logs.L().LB.Info("health_checker_started",
		slog.Duration("interval", 5*time.Second),
	)

	// 4) Auth/JWT/RateLimit
	authService := auth.NewAuthService(auth.NewAPIKeyValidator(cfg.APIKeyHeader, []string{"SUPERSECRET"}))

	validator, err := jwt.NewTokenValidator(cfg.JWTPublicKey, cfg.JWTHeader)
	if err != nil {
		log.Fatal("JWT validator error:", err)
	}

	store := ratelimit.NewRedisStore(cfg.RedisURL)
	limiter := ratelimit.NewRateLimiter(ratelimit.Config{
		Requests:   20,
		WindowSec:  60,
		RedisURL:   cfg.RedisURL,
		HeaderName: cfg.JWTHeader,
	}, store)

	// 5) Version router
	vr := router.LoadVersionRouter()

	// 6) Health endpoints
	rchk, err := health.NewRedisChecker("redis", cfg.RedisURL)
	if err != nil {
		log.Fatal("redis checker init:", err)
	}
	privKeyChk := health.NewFileKeyChecker("jwt_private_key", cfg.JWTPrivateKey, "rsa_private")
	pubKeyChk := health.NewFileKeyChecker("jwt_public_key", cfg.JWTPublicKey, "rsa_public")
	regChk := health.NewRegistryChecker("registry", reg, false)
	hh := health.NewHandler([]health.Checker{rchk, privKeyChk, pubKeyChk, regChk}, health.Options{
		PerCheckTimeout: 1 * time.Second,
		ShowDetails:     true,
	})

	// 7) Top-level mux
	root := http.NewServeMux()
	root.HandleFunc("/livez", hh.Liveness)
	root.HandleFunc("/readyz", hh.Readiness)
	root.HandleFunc("/health", hh.Readiness)

	for _, prefix := range sr.Prefixes() {
		root.Handle(prefix, px)
	}

	root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"service":"superapp-gateway","status":"running","try":["/v1/ping","/v2/ping","/livez","/readyz"]}`))
			return
		}
		vr.ServeHTTP(w, r)
	})

	finalHandler := router.NewRouter(
		root, // ensure NewRouter accepts http.Handler
		middlewares.RequestID(),
		middlewares.Logging(),
		middlewares.Recovery(),
		ratelimit.Middleware(limiter, cfg.JWTHeader),
		middlewares.AuthMiddleware(authService, cfg.APIKeyHeader),
		middlewares.JWTMiddleware(validator, cfg.JWTHeader),
	)

	// 8) HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  75 * time.Second,
	}

	// Signal handling
	stopCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logs.L().Gateway.Info("http_listen", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.L().Gateway.Error("server_listen_error", slog.String("error", err.Error()))
			stop()
		}
	}()

	// Block until signal
	<-stopCtx.Done()
	logs.L().Gateway.Info("shutdown_signal_received")

	// Flip readiness to draining
	hh.SetDraining(true)

	// Grace period to let LB stop sending traffic (optional)
	time.Sleep(500 * time.Millisecond)

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop background tasks
	bgCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logs.L().Gateway.Error("server_shutdown_error", slog.String("error", err.Error()))
	} else {
		logs.L().Gateway.Info("server_shutdown_complete")
	}
}
