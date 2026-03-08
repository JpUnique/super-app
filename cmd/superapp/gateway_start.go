package main

import (
    "context"
    "net/http"
    "os/signal"
    "syscall"
    "time"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"

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

var (
    flagPort     string
    flagRoutes   string
    flagRedisURL string
)

var gatewayStartCmd = &cobra.Command{
    Use:   "gateway start",
    Short: "Start the API Gateway",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1) Load file config
        cfg := config.Load()

        // 2) CLI overrides
        if flagPort != "" {
            cfg.Port = flagPort
        }
        if flagRoutes != "" {
            cfg.RoutesFile = flagRoutes
        }
        if flagRedisURL != "" {
            cfg.RedisURL = flagRedisURL
        }

        // 3) Logging
        logs.MustInit()
        logs.L().Gateway.Info("gateway_start_cli", "port", cfg.Port, "routes", cfg.RoutesFile)

        // 4) Registry (seed healthy to avoid cold-start 502s)
        reg := loadbalancer.NewRegistry()
        b1 := &loadbalancer.Backend{Name: "orders-1", URL: "http://localhost:8081"}
        b1.SetHealthy(true)
        reg.Register(b1)
        b2 := &loadbalancer.Backend{Name: "orders-2", URL: "http://localhost:8082"}
        b2.SetHealthy(true)
        reg.Register(b2)

        // 5) Health checker
        hc := loadbalancer.NewHealthChecker(reg, loadbalancer.HealthCheckerConfig{
            Interval: 5 * time.Second,
            Timeout:  2 * time.Second,
            Path:     "/health",
        })
        bgCtx, cancel := context.WithCancel(context.Background())
        defer cancel()
        go hc.Start(bgCtx)

        // 6) Auth / JWT / Rate Limit
        authService := auth.NewAuthService(auth.NewAPIKeyValidator(cfg.APIKeyHeader, []string{"SUPERSECRET"}))

        validator, err := jwt.NewTokenValidator(cfg.JWTPublicKey, cfg.JWTHeader)
        if err != nil {
            return err
        }

        store := ratelimit.NewRedisStore(cfg.RedisURL)
        limiter := ratelimit.NewRateLimiter(ratelimit.Config{
            Requests:   20,
            WindowSec:  60,
            RedisURL:   cfg.RedisURL,
            HeaderName: cfg.JWTHeader,
        }, store)

        // 7) Version router
        vr := router.LoadVersionRouter()

        // 8) YAML routing + proxy
        sr, strategies, err := proxy.LoadRoutesFromYAML(cfg.RoutesFile)
        if err != nil {
            return err
        }
        px := proxy.New(reg, sr, nil)
        for _, svc := range sr.Services() {
            switch strategies[svc] {
            case "least_conn":
                px.SetBalancer(svc, loadbalancer.NewLeastConn(reg))
            default:
                px.SetBalancer(svc, loadbalancer.NewRoundRobin(reg))
            }
        }

        // 9) Health handlers
        rchk, err := health.NewRedisChecker("redis", cfg.RedisURL)
        if err != nil {
            return err
        }
        privKeyChk := health.NewFileKeyChecker("jwt_private_key", cfg.JWTPrivateKey, "rsa_private")
        pubKeyChk := health.NewFileKeyChecker("jwt_public_key", cfg.JWTPublicKey, "rsa_public")
        regChk := health.NewRegistryChecker("registry", reg, false)
        hh := health.NewHandler([]health.Checker{rchk, privKeyChk, pubKeyChk, regChk}, health.Options{
            PerCheckTimeout: 1 * time.Second,
            ShowDetails:     true,
        })

        // 10) Root mux
        root := http.NewServeMux()
        root.HandleFunc("/livez", hh.Liveness)
        root.HandleFunc("/readyz", hh.Readiness)
        root.HandleFunc("/health", hh.Readiness)

        // Mount YAML prefixes
        for _, p := range sr.Prefixes() {
            root.Handle(p, px)
        }

        // Friendly index + fallback to version router
        root.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
            if r.URL.Path == "/" {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                _, _ = w.Write([]byte(`{"service":"superapp-gateway","status":"running","try":["/v1/ping","/v2/ping","/livez","/readyz"]}`))
                return
            }
            vr.ServeHTTP(w, r)
        })

        // 11) Middleware pipeline
        finalHandler := router.NewRouter(
            root, // ensure router.NewRouter accepts http.Handler
            middlewares.RequestID(),
            middlewares.Logging(),
            middlewares.Recovery(),
            ratelimit.Middleware(limiter, cfg.JWTHeader),
            middlewares.AuthMiddleware(authService, cfg.APIKeyHeader),
            middlewares.JWTMiddleware(validator, cfg.JWTHeader),
        )

        // 12) HTTP server + graceful shutdown
        srv := &http.Server{
            Addr:         ":" + cfg.Port,
            Handler:      finalHandler,
            ReadTimeout:  15 * time.Second,
            WriteTimeout: 20 * time.Second,
            IdleTimeout:  75 * time.Second,
        }

        stopCtx, stop := signalNotifyContext()
        defer stop()

        go func() {
            logs.L().Gateway.Info("http_listen", "addr", srv.Addr)
            if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                logs.L().Gateway.Error("server_listen_error", "error", err.Error())
                stop()
            }
        }()

        <-stopCtx.Done()
        logs.L().Gateway.Info("shutdown_signal_received")

        // readiness -> draining
        hh.SetDraining(true)
        time.Sleep(500 * time.Millisecond)

        shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel2()
        cancel() // stop checker

        if err := srv.Shutdown(shutdownCtx); err != nil {
            logs.L().Gateway.Error("server_shutdown_error", "error", err.Error())
        } else {
            logs.L().Gateway.Info("server_shutdown_complete")
        }
        return nil
    },
}

func init() {
    rootCmd.AddCommand(gatewayStartCmd)

    gatewayStartCmd.Flags().StringVar(&flagPort, "port", "", "override HTTP port (e.g., 8080)")
    _ = viper.BindPFlag("port", gatewayStartCmd.Flags().Lookup("port"))

    gatewayStartCmd.Flags().StringVar(&flagRoutes, "routes", "", "path to routes YAML")
    _ = viper.BindPFlag("routes_file", gatewayStartCmd.Flags().Lookup("routes"))

    gatewayStartCmd.Flags().StringVar(&flagRedisURL, "redis", "", "Redis URL for rate limiting")
    _ = viper.BindPFlag("redis_url", gatewayStartCmd.Flags().Lookup("redis"))
}

func signalNotifyContext() (context.Context, context.CancelFunc) {
    return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}