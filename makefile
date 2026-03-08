# ---------------- Super-App API Gateway Makefile ----------------
# Loads .env if present, exports to shell
include $(if $(wildcard .env), .env)
export



# Default goal
.DEFAULT_GOAL := start

# Project settings
APP_NAME        ?= api-gateway
CMD_GATEWAY     ?= ./cmd/api-gateway
CMD_CLI         ?= ./cmd/superapp
BIN_DIR         ?= ./bin
BIN_PATH        ?= $(BIN_DIR)/$(APP_NAME)

# Runtime/config (can be overridden in .env or inline)
PORT            ?= 8080
REDIS_URL       ?= redis://localhost:6379
ROUTES_FILE     ?= ./internal/gateway/proxy/routes.yaml
CONFIG_FILE     ?= ./internal/gateway/config/config.yaml
JWT_PRIVATE_KEY ?= ./internal/gateway/jwt/rsa_private.pem
JWT_PUBLIC_KEY  ?= ./internal/gateway/jwt/rsa_public.pem
LOG_FORMAT      ?= json          # json|console
LOG_LEVEL       ?= info          # debug|info|warn|error

# Go
GO              ?= go
GOFLAGS         ?=
LDFLAGS         ?= -s -w
TAGS            ?=

# Tools
GOIMPORTS_VER   ?= v0.29.0

# Phony targets
.PHONY: fmt build clean start run run-cli test tidy deps gen-keys start-redis stop-redis logs-redis print-env

# --- Formatting & Hygiene ----------------------------------------------------

fmt:
    $(GO) fmt ./...
    $(GO) run golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VER) -w .

tidy:
    $(GO) mod tidy


# --- Build / Test / Clean ----------------------------------------------------

build:
    @mkdir -p $(BIN_DIR)
    $(GO) build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $(BIN_PATH) $(CMD_GATEWAY)

test:
    $(GO) test ./... -v -short

clean:
    $(GO) clean
    $(GO) clean -cache
    rm -rf $(BIN_DIR)

# --- Run (binary & cli) ------------------------------------------------------

# Start using the built binary
start: build
    @echo "→ starting $(APP_NAME) on :$(PORT)"
    LOG_FORMAT=$(LOG_FORMAT) LOG_LEVEL=$(LOG_LEVEL) \
    GATEWAY_CONFIG_DIR=$$(dirname $(CONFIG_FILE)) \
    GATEWAY_PORT=$(PORT) \
    GATEWAY_REDIS_URL=$(REDIS_URL) \
    GATEWAY_JWT_PRIVATE_KEY=$(JWT_PRIVATE_KEY) \
    GATEWAY_JWT_PUBLIC_KEY=$(JWT_PUBLIC_KEY) \
    GATEWAY_API_KEY_HEADER=X-API-KEY \
    GATEWAY_JWT_HEADER=Authorization \
    $(BIN_PATH)

# Run directly via 'go run' (main entry)
run:
    LOG_FORMAT=$(LOG_FORMAT) LOG_LEVEL=$(LOG_LEVEL) \
    GATEWAY_CONFIG_DIR=$$(dirname $(CONFIG_FILE)) \
    GATEWAY_PORT=$(PORT) \
    GATEWAY_REDIS_URL=$(REDIS_URL) \
    GATEWAY_JWT_PRIVATE_KEY=$(JWT_PRIVATE_KEY) \
    GATEWAY_JWT_PUBLIC_KEY=$(JWT_PUBLIC_KEY) \
    GATEWAY_API_KEY_HEADER=X-API-KEY \
    GATEWAY_JWT_HEADER=Authorization \
    $(GO) run $(GOFLAGS) $(CMD_GATEWAY)

# Run via Cobra CLI: `superapp gateway start`
run-cli:
    LOG_FORMAT=$(LOG_FORMAT) LOG_LEVEL=$(LOG_LEVEL) \
    $(GO) run $(GOFLAGS) $(CMD_CLI) gateway start \
      --config $(CONFIG_FILE) \
      --routes $(ROUTES_FILE) \
      --port $(PORT)

# --- Utilities ---------------------------------------------------------------

gen-keys:
    @mkdir -p $$(dirname "$(JWT_PRIVATE_KEY)")
    @mkdir -p $$(dirname "$(JWT_PUBLIC_KEY)")
    @echo "→ generating RSA private key: $(JWT_PRIVATE_KEY)"
    openssl genrsa -out "$(JWT_PRIVATE_KEY)" 2048
    @echo "→ generating RSA public key:  $(JWT_PUBLIC_KEY)"
    openssl rsa -in "$(JWT_PRIVATE_KEY)" -pubout -out "$(JWT_PUBLIC_KEY)"
    @echo "✓ keys generated"

start-redis:
    docker run --name superapp-redis -p 6379:6379 -d redis:7

stop-redis:
    - docker stop superapp-redis >/dev/null 2>&1 || true
    - docker rm superapp-redis >/dev/null 2>&1 || true
    @echo "✓ redis container removed"

logs-redis:
    docker logs -f superapp-redis

print-env:
    @echo "APP_NAME=$(APP_NAME)"
    @echo "CMD_GATEWAY=$(CMD_GATEWAY)"
    @echo "CMD_CLI=$(CMD_CLI)"
    @echo "BIN_PATH=$(BIN_PATH)"
    @echo "PORT=$(PORT)"
    @echo "REDIS_URL=$(REDIS_URL)"
    @echo "CONFIG_FILE=$(CONFIG_FILE)"
    @echo "ROUTES_FILE=$(ROUTES_FILE)"
    @echo "JWT_PRIVATE_KEY=$(JWT_PRIVATE_KEY)"
    @echo "JWT_PUBLIC_KEY=$(JWT_PUBLIC_KEY)"
    @echo "LOG_FORMAT=$(LOG_FORMAT)"
    @echo "LOG_LEVEL=$(LOG_LEVEL)"

up:
    