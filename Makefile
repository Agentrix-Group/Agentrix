# Agentrix developer, database and CI entry points.
# Quality gates check and fail; they never rewrite files (use `make fmt` explicitly to format).
SHELL := /bin/bash
.DEFAULT_GOAL := help

-include .env
export

BINARY_NAME ?= agentrix
BUILD_DIR ?= bin
MAIN_PATH ?= main.go
COMPOSE ?= docker compose
GO ?= go
GOFLAGS_RO := -mod=readonly

DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_NAME ?= agentrix
DB_SCRIPT ?= ./script/setup_postgres.sh
PG_TEST_PORT ?= 55432

.PHONY: help setup setup-all agentrix-setup db-setup db-migrate db-status db-reset db-seed db-bootstrap \
        build build-all build-engine build-engines engine engine-test bot-runtimes bot-runtimes-lock neural-templates run clean coverage \
        fmt fmt-check vet test test-race test-integration openapi-check check \
        web-install web-dev web-lint web-test web-build web-e2e \
        demo-up demo-down demo-reset demo-smoke

help:
	@echo "Agentrix Platform - Available Targets:"
	@echo ""
	@echo "  Quality Gates & CI (Strict):"
	@echo "    make check            - Run fmt-check, vet, and unit + race tests"
	@echo "    make fmt-check        - Check gofmt compliance without modifying files"
	@echo "    make fmt              - Format Go code with gofmt"
	@echo "    make vet              - Run go vet on all packages"
	@echo "    make test             - Run unit tests in parallel"
	@echo "    make test-race        - Run unit tests with race detector enabled"
	@echo "    make test-integration - Run integration tests against PostgreSQL & engine"
	@echo "    make openapi-check    - Verify OpenAPI route policies and contracts"
	@echo "    make coverage         - Generate coverage profile and display percentage"
	@echo ""
	@echo "  Engine (Rust Starfighter):"
	@echo "    make engine           - Build bin/starfighter-engine from commit pinned in engine.lock"
	@echo "    make engine-test      - Run cargo fmt, clippy -D warnings and tests on agentrix_engine"
	@echo "    make bot-runtimes     - Build bot runtimes (python-ml-cpu) into bin/runtimes"
	@echo "    make neural-templates - Rebuild the ONNX/NPZ bot templates in web/public"
	@echo ""
	@echo "  Local Development & Build:"
	@echo "    make build            - Compile Go backend binary (bin/agentrix)"
	@echo "    make build-all        - Compile Go backend and build frontend web bundle"
	@echo "    make run              - Run Go backend server locally"
	@echo "    make clean            - Clean build artifacts, dist, coverage and caches"
	@echo ""
	@echo "  Database (PostgreSQL & Goose):"
	@echo "    make db-migrate       - Apply forward-only database migrations"
	@echo "    make db-status        - Check connection, schema version and pending migrations"
	@echo "    make db-bootstrap     - Bootstrap Starfighter demo data (migrations + bots + demo match)"
	@echo "    make db-seed          - Reapply seed data (00_seeds_postgresql.sql)"
	@echo "    make db-reset         - Recreate database from scratch (drop, schema, seeds)"
	@echo ""
	@echo "  Frontend Web (React + Vite):"
	@echo "    make web-install      - Install frontend dependencies"
	@echo "    make web-dev          - Start frontend dev server with API proxy"
	@echo "    make web-test         - Run Vitest suite and i18n parity check"
	@echo "    make web-build        - Build production bundle in web/dist/ and check size"
	@echo "    make web-lint         - Run ESLint on web frontend"
	@echo "    make web-e2e          - Run Playwright end-to-end tests"
	@echo ""
	@echo "  Demo & Docker Compose:"
	@echo "    make demo-up          - Start full stack with Docker Compose + bootstrap"
	@echo "    make demo-down        - Stop Docker Compose stack"
	@echo "    make demo-reset       - Remove demo containers/volumes and restart"
	@echo "    make demo-smoke       - Run HTTP smoke verification against stack"

# --- Quality Gates & CI ------------------------------------------------------

fmt:
	gofmt -w $$(git ls-files '*.go')

fmt-check:
	@out=$$(gofmt -l $$(git ls-files '*.go' 2>/dev/null || find . -name '*.go' -not -path './web/*')); \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet $(GOFLAGS_RO) ./...

test:
	$(GO) test $(GOFLAGS_RO) -count=1 $$($(GO) list ./... | grep -v /test/integration)

test-race:
	$(GO) test $(GOFLAGS_RO) -race -count=1 $$($(GO) list ./... | grep -v /test/integration)

check: fmt-check vet test-race

test-integration:
	@set -o pipefail; AGENTRIX_REQUIRE_DB=1 AGENTRIX_REQUIRE_SANDBOX=1 AGENTRIX_REQUIRE_ENGINE=1 \
	  $(GO) test $(GOFLAGS_RO) -race -count=1 -v -timeout 20m ./test/integration/ ./src/executor/ | tee integration.log
	@if grep -q -- '--- SKIP' integration.log; then echo "unexpected skipped tests"; grep -- '--- SKIP' integration.log; exit 1; fi

openapi-check:
	$(GO) test $(GOFLAGS_RO) -count=1 -run 'TestOpenAPI|TestRoutePolicy' ./src/server/

coverage:
	@mkdir -p $(BUILD_DIR)
	@$(GO) test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	@cat $(BUILD_DIR)/coverage.out | grep -v "mock" | grep -v "_test.go" > $(BUILD_DIR)/coverage.filtered.out
	@$(GO) tool cover -func=$(BUILD_DIR)/coverage.filtered.out | grep "total:" | awk '{print $$3}'

# --- Engine (Rust Starfighter) -----------------------------------------------

engine:
	./script/build_engine.sh

# Runtimes de bots (ADR-0014): Python 3.12 independiente + python-ml-cpu.
bot-runtimes:
	./script/build_bot_runtimes.sh

# Plantillas de bots con red neuronal para la web (ADR-0014, N5).
neural-templates:
	python3 ./script/build_neural_templates.py

# Regenera el lock con hashes de python-ml-cpu (requiere red).
bot-runtimes-lock:
	uv pip compile runtimes/python-ml-cpu/requirements.in --generate-hashes \
	  --python-version 3.12 --python-platform x86_64-manylinux_2_28 --no-header \
	  -o runtimes/python-ml-cpu/requirements.lock

build-engine: engine
build-engines: engine

engine-test:
	cd ../agentrix_engine && cargo fmt --check && cargo clippy --locked --all-targets -- -D warnings && cargo test --locked

# --- Compilation & Execution -------------------------------------------------

build: fmt-check vet
	@echo "Building backend binary..."
	mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS_RO) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: build web-build
	@echo "Fullstack build complete (backend + frontend)!"

run:
	@echo "Starting server..."
	$(GO) run $(MAIN_PATH)

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/
	rm -rf web/dist/ /tmp/agentrix-web-build
	rm -rf artifacts/replays/* artifacts/submissions/*
	rm -f coverage.out coverage.filtered.out integration.log
	$(GO) clean -cache
	@echo "Clean complete!"

# --- Database & Setup --------------------------------------------------------

setup: agentrix-setup db-setup

setup-all: setup web-install

agentrix-setup:
	@echo "Tidying and downloading Go dependencies..."
	$(GO) mod tidy
	$(GO) mod download

db-setup:
	@echo "Setting up PostgreSQL database..."
	@$(DB_SCRIPT) || (echo "Database setup failed!" && exit 1)

db-migrate:
	@echo "Applying forward-only database migrations..."
	@$(GO) run ./cmd/migrate up

db-status:
	@echo "Checking PostgreSQL connection on $(DB_HOST):$(DB_PORT)..."
	@pg_isready -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) || (echo "PostgreSQL is not reachable!" && exit 1)
	@$(GO) run ./cmd/migrate status

db-seed:
	@echo "Applying seed data to $(DB_NAME)..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f ./script/data/00_seeds_postgresql.sql

db-bootstrap:
	@echo "Bootstrapping Starfighter demo environment..."
	@$(GO) run ./cmd/bootstrap

db-reset:
	@echo "Resetting database $(DB_NAME)..."
	@dropdb --if-exists -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)
	@$(DB_SCRIPT)

# --- Frontend Web ------------------------------------------------------------

web-install:
	cd web && npm ci

web-dev:
	cd web && npm run dev

web-lint:
	cd web && npm run lint

web-test:
	cd web && npm test && npm run test:parity

web-build:
	cd web && npm run build

web-e2e:
	cd web && npx playwright test

# --- Demo & Docker Compose ---------------------------------------------------

demo-up:
	./script/checkout_engine.sh
	$(COMPOSE) --profile demo up --build -d
	$(COMPOSE) --profile demo wait bootstrap 2>/dev/null || $(COMPOSE) logs -f bootstrap

demo-smoke:
	$(COMPOSE) --profile smoke run --rm smoke

demo-down:
	$(COMPOSE) down

demo-reset:
	$(COMPOSE) --profile demo --profile smoke down --volumes --remove-orphans
	$(MAKE) demo-up
