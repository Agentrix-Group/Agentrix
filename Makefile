# Variables and Environment
-include .env
export

BINARY_NAME=agentrix
BUILD_DIR=bin
MAIN_PATH=main.go
DB_SCRIPT=./script/setup_postgres.sh
TEST_PARALLEL=4
GO_PACKAGES=./...
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_NAME ?= agentrix

.DEFAULT_GOAL := help
.PHONY: help setup setup-all agentrix-setup db-setup db-migrate db-status db-reset db-seed db-bootstrap \
        build build-all build-engine build-engines lint test test-race test-integration coverage run clean \
        web-install web-build web-dev

# Show available targets
help:
	@echo "Agentrix Platform - Available Targets:"
	@echo ""
	@echo "  Setup & Initialization:"
	@echo "    make setup          - Setup backend dependencies and PostgreSQL database"
	@echo "    make setup-all      - Full setup (Go deps + PostgreSQL + Frontend npm)"
	@echo "    make agentrix-setup - Install Go dependencies (tidy + download)"
	@echo ""
	@echo "  Database (PostgreSQL & Goose):"
	@echo "    make db-migrate     - Apply forward-only database migrations"
	@echo "    make db-status      - Check connection, schema version, and pending migrations"
	@echo "    make db-setup       - Initialize database schema and seeds"
	@echo "    make db-seed        - Reapply seed data (00_seeds_postgresql.sql)"
	@echo "    make db-reset       - Recreate database from scratch (drop, schema, seeds)"
	@echo ""
	@echo "  Quality & Testing:"
	@echo "    make lint           - Format (gofmt) and analyze (go vet) code"
	@echo "    make test           - Run all tests in parallel"
	@echo "    make test-race      - Run all tests with race detector enabled"
	@echo "    make test-integration - Run integration tests with PostgreSQL"
	@echo "    make coverage       - Generate coverage profile and display percentage"
	@echo ""
	@echo "  Compilation & Execution:"
	@echo "    make build          - Lint, test, and compile Go server (bin/agentrix)"
	@echo "    make build-all      - Compile both backend binary and frontend web bundle"
	@echo "    make build-engine   - Build Starfighter Rust engine (bin/starfighter-engine)"
	@echo "    make build-engines  - Build Starfighter Rust engine (bin/starfighter-engine)"
	@echo "    make run            - Run Go server directly"
	@echo "    make clean          - Remove binaries, test artifacts, coverage, and dist"
	@echo ""
	@echo "  Frontend Web (React + Vite):"
	@echo "    make web-install    - Install npm dependencies in web/"
	@echo "    make web-dev        - Start frontend dev server with API proxy"
	@echo "    make web-build      - Build production frontend bundle in web/dist/"
	@echo ""

# Full setup (Go backend + PostgreSQL)
setup: agentrix-setup db-setup
	@echo "Backend and database setup complete!"

# Fullstack setup (Go backend + PostgreSQL + Frontend)
setup-all: setup web-install
	@echo "All backend, database, and frontend dependencies setup complete!"

# Install Go dependencies
agentrix-setup:
	@echo "Installing Go dependencies..."
	go mod tidy
	go mod download
	@echo "Go dependencies installed!"

# Setup PostgreSQL database (schema + seeds)
db-setup:
	@echo "Setting up PostgreSQL database..."
	@$(DB_SCRIPT) || (echo "Database setup failed!" && exit 1)

# Apply forward-only database migrations
db-migrate:
	@echo "Applying forward-only database migrations..."
	@go run ./cmd/migrate up

# Check PostgreSQL connection status, schema version, and pending migrations
db-status:
	@echo "Checking PostgreSQL connection on $(DB_HOST):$(DB_PORT)..."
	@pg_isready -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) || (echo "PostgreSQL is not reachable!" && exit 1)
	@go run ./cmd/migrate status

# Reapply seed data only
db-seed:
	@echo "Applying seed data to $(DB_NAME)..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f ./script/data/00_seeds_postgresql.sql
	@echo "Seeds applied successfully!"

# Idempotently bootstrap Starfighter demo environment (migrations + seeds + bots + demo match & replay)
db-bootstrap:
	@echo "Bootstrapping Starfighter demo environment..."
	@go run ./cmd/bootstrap

# Reset database completely (drop database, re-run DDL and seeds)
db-reset:
	@echo "Resetting database $(DB_NAME)..."
	@dropdb --if-exists -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)
	@$(DB_SCRIPT)
	@echo "Database reset complete!"

# Build the backend application (lint + test + compile)
build: lint test
	@echo "Building application..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete!"

# Fullstack build (backend binary + frontend production bundle)
build-all: build web-build
	@echo "Fullstack build complete (backend + frontend)!"

# Starfighter Rust engine build
ENGINE_SRC ?= ../agentrix_engine
build-engine:
	@echo "Building Starfighter Rust engine..."
	@./script/build_engine.sh $(ENGINE_SRC) $(BUILD_DIR)/starfighter-engine

build-engines: build-engine

# Lint the code
lint:
	@echo "Linting code..."
	@echo "Running gofmt with simplify..."
	gofmt -s -w .
	@echo "Running go vet..."
	go vet $(GO_PACKAGES)
	@echo "Lint complete!"

# Run tests
test:
	@echo "Running tests..."
	go test -short -v -parallel $(TEST_PARALLEL) $(GO_PACKAGES)
	@echo "Test summary complete!"

# Run tests with race detection
test-race:
	@echo "Running tests with race detector..."
	go test -race -v $(GO_PACKAGES)
	@echo "Race detection test complete!"

# Run integration tests against PostgreSQL
test-integration:
	@echo "Running integration tests against PostgreSQL..."
	@GOCACHE=/tmp/agentrix-go-cache go test -v ./test/integration/...

# Run tests with coverage
coverage:
	@mkdir -p $(BUILD_DIR)
	@go test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	@cat $(BUILD_DIR)/coverage.out | grep -v "mock" | grep -v "_test.go" > $(BUILD_DIR)/coverage.filtered.out
	@go tool cover -func=$(BUILD_DIR)/coverage.filtered.out | grep "total:" | awk '{print $$3}'

# Run the backend server directly
run:
	@echo "Starting server..."
	go run $(MAIN_PATH)

# Clean build artifacts, temporary test files, and caches
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/
	rm -rf web/dist/
	rm -rf artifacts/replays/* artifacts/submissions/*
	rm -f coverage.out coverage.filtered.out
	go clean -cache
	@echo "Clean complete!"

# Frontend Web targets
web-install:
	@echo "Installing frontend dependencies..."
	cd web && npm install

web-build:
	@echo "Building frontend application..."
	cd web && npm run build

web-dev:
	@echo "Starting frontend dev server..."
	cd web && npm run dev
