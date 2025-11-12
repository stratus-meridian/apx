# APX Platform Makefile
# Quick commands for development, build, test, and deploy

.PHONY: help init build test deploy clean

# Detect if private code exists (open-core model)
PRIVATE_DIR = .private
HAS_PRIVATE := $(shell [ -d $(PRIVATE_DIR) ] && echo true)

# Include private targets if available
-include .private/Makefile.private

# Default target
help:
	@echo "APX Platform - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make check-private - Check if private code is available"
	@echo "  make init          - Initialize local development environment"
	@echo "  make up            - Start all services via docker-compose"
	@echo "  make down          - Stop all services"
	@echo "  make logs          - Tail logs from all services"
	@echo "  make shell-edge    - Shell into edge container"
	@echo "  make shell-router  - Shell into router container"
	@echo ""
	@echo "Build:"
	@echo "  make build         - Build all components"
	@echo "  make build-edge    - Build edge gateway image"
	@echo "  make build-router  - Build router service"
	@echo "  make build-wasm    - Build WASM filters"
	@echo ""
	@echo "Test:"
	@echo "  make test          - Run all tests"
	@echo "  make test-router   - Run router tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make lint          - Run linters"
	@echo ""
	@echo "Deploy:"
	@echo "  make deploy-dev    - Deploy to dev environment"
	@echo "  make deploy-staging - Deploy to staging"
	@echo "  make deploy-prod   - Deploy to production"
	@echo ""
	@echo "Policy Management:"
	@echo "  make compile-policies - Compile YAML configs to artifacts"
	@echo "  make apply-policies   - Apply policies to environment"
	@echo ""
	@echo "Observability:"
	@echo "  make metrics       - Open Prometheus UI"
	@echo "  make dashboards    - Open Grafana UI"
	@echo ""

# Check if private code is available
check-private:
ifeq ($(HAS_PRIVATE),true)
	@echo "✅ Private code detected (.private/)"
	@echo "   Running with enterprise features (agents, advanced analytics)"
else
	@echo "ℹ️  Running in open-source mode (no .private/)"
	@echo "   To add enterprise features:"
	@echo "   git clone git@github.com:apx-platform/apx-private.git .private"
endif

# Initialize local environment
init: check-private
	@echo ""
	@echo "Initializing APX local environment..."
	@cp -n .env.example .env || true
	@echo "✓ Created .env file (edit with your settings)"
	@mkdir -p observability/otel observability/prometheus observability/grafana
	@echo "✓ Created observability directories"
	@echo ""
	@echo "Run 'make up' to start all services"

# Docker Compose commands
up: check-private
	@echo ""
	docker-compose up -d
ifeq ($(HAS_PRIVATE),true)
	@echo "✅ Services starting with enterprise features"
else
	@echo "✅ Services starting in open-source mode"
endif
	@echo "Check status with 'make status'"

down:
	docker-compose down

logs:
	docker-compose logs -f

status:
	docker-compose ps

# Build commands
build: build-edge build-router

build-edge:
	@echo "Building edge gateway..."
	cd edge && docker build -t apx-edge:latest .

build-router:
	@echo "Building router service..."
	cd router && go build -o bin/router cmd/router/main.go

build-wasm:
	@echo "Building WASM filters..."
	cd edge/wasm-filters && make build

# Test commands
test: test-router test-integration

test-router:
	@echo "Running router tests..."
	cd router && go test -v ./...

test-integration:
	@echo "Running integration tests..."
	@echo "TODO: Implement integration tests"

lint:
	@echo "Running linters..."
	cd router && golangci-lint run ./...

# Policy management
compile-policies:
	@echo "Compiling policies..."
	./tools/cli/apx compile configs/samples/*.yaml

apply-policies:
	@echo "Applying policies to dev environment..."
	./tools/cli/apx apply --env dev configs/samples/*.yaml

# Deploy commands
deploy-dev:
	@echo "Deploying to dev environment..."
	./tools/cli/apx deploy edge --env dev
	./tools/cli/apx deploy router --env dev

deploy-staging:
	@echo "Deploying to staging environment..."
	./tools/cli/apx deploy edge --env staging
	./tools/cli/apx deploy router --env staging

deploy-prod:
	@echo "⚠️  Deploying to PRODUCTION..."
	@read -p "Are you sure? (yes/no): " confirm && [ "$$confirm" = "yes" ]
	./tools/cli/apx deploy edge --env production
	./tools/cli/apx deploy router --env production

# Observability
metrics:
	@echo "Opening Prometheus at http://localhost:9090"
	open http://localhost:9090 || xdg-open http://localhost:9090

dashboards:
	@echo "Opening Grafana at http://localhost:3000"
	@echo "Default credentials: admin / admin"
	open http://localhost:3000 || xdg-open http://localhost:3000

# Shell access
shell-edge:
	docker-compose exec edge sh

shell-router:
	docker-compose exec router sh

# Clean up
clean:
	@echo "Cleaning up..."
	docker-compose down -v
	rm -rf router/bin
	rm -rf edge/wasm-filters/build
	@echo "✓ Cleaned"
