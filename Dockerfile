# -------- Build stage --------
FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata build-base

# Copy go mod/sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary (no CGO)
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w" -o /out/api-gateway ./cmd/api-gateway
RUN go build -ldflags="-s -w" -o /out/superapp ./cmd/superapp

# -------- Runtime stage (distroless) --------
FROM gcr.io/distroless/static:nonroot

WORKDIR /app
USER nonroot:nonroot

# Copy binaries
COPY --from=builder /out/api-gateway /app/api-gateway
COPY --from=builder /out/superapp /app/superapp

# Copy default config & routes (can be overridden by K8s/Compose volumes)
COPY internal/gateway/config/config.yaml /app/internal/gateway/config/config.yaml
COPY internal/gateway/proxy/routes.yaml /app/internal/gateway/proxy/routes.yaml
# Copy example keys (for local/dev only; use K8s Secrets in production)
COPY internal/gateway/jwt/rsa_private.pem /app/internal/gateway/jwt/rsa_private.pem
COPY internal/gateway/jwt/rsa_public.pem  /app/internal/gateway/jwt/rsa_public.pem

EXPOSE 8080

# Default entrypoint uses the thin binary; override to run CLI.
ENTRYPOINT ["/app/api-gateway"]
# To use the CLI in containers, override command e.g.:
# CMD ["/app/superapp", "gateway", "start", "--config", "/app/internal/gateway/config/config.yaml", "--routes", "/app/internal/gateway/proxy/routes.yaml", "--port", "8080"]