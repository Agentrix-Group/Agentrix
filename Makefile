# Variables
BINARY_NAME=agentrix
BUILD_DIR=bin
MAIN_PATH=main.go
DB_SCRIPT=./script/setup_postgres.sh
TEST_PARALLEL=4
GO_PACKAGES=./...

.DEFAULT_GOAL := help
.PHONY: setup agentrix-setup db-setup build lint test run clean help coverage web-install web-build web-dev

# Show available targets
help:
	@echo "Available targets:"
	@echo "  setup          - Full project setup (deps + db)"
	@echo "  build          - Build application (lint + test + compile)"
	@echo "  test           - Run tests"
	@echo "  coverage       - Show total coverage percentage"
	@echo "  lint           - Format and check code"
	@echo "  run            - Start server"
	@echo "  clean          - Remove build artifacts"

# Full setup (agentrix-setup + db)
setup: agentrix-setup db-setup
	@echo "Setup complete!"

# Install dependencies
agentrix-setup:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed!"

# Setup database
db-setup:
	@echo "Setting up database..."
	@$(DB_SCRIPT) || (echo "Database setup failed!" && exit 1)

# Build the application
build: lint test
	@echo "Building application..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete!"

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

# Run tests with coverage
coverage:
	@mkdir -p $(BUILD_DIR)
	@go test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	@cat $(BUILD_DIR)/coverage.out | grep -v "mock" | grep -v "_test.go" > $(BUILD_DIR)/coverage.filtered.out
	@go tool cover -func=$(BUILD_DIR)/coverage.filtered.out | grep "total:" | awk '{print $$3}'

# Run the server
run:
	@echo "Starting server..."
	go run $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/
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

