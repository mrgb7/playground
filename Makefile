# Playground - Makefile for development tasks

# Variables
BINARY_NAME=playground
BUILD_DIR=bin
MAIN_PACKAGE=.
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")

# Default target
.PHONY: all
all: clean test build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Multi-platform binaries built in $(BUILD_DIR)/"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	@go test -v -race ./...

# Benchmark tests
.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
	fi

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	@go mod download

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(MAIN_PACKAGE)

# Run the application
.PHONY: run
run:
	@go run $(MAIN_PACKAGE) $(ARGS)

# Development setup
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@go mod tidy
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "Development environment ready!"

# Check for security vulnerabilities
.PHONY: security
security:
	@echo "Checking for security vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Install it with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# Pre-commit checks
.PHONY: pre-commit
pre-commit: fmt lint test
	@echo "Pre-commit checks completed successfully!"

# CI pipeline
.PHONY: ci
ci: deps fmt lint test-race test-coverage
	@echo "CI pipeline completed successfully!"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-race     - Run tests with race detection"
	@echo "  benchmark     - Run benchmark tests"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  tidy          - Tidy dependencies"
	@echo "  deps          - Download dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install the binary"
	@echo "  run           - Run the application (use ARGS=... for arguments)"
	@echo "  dev-setup     - Set up development environment"
	@echo "  security      - Check for security vulnerabilities"
	@echo "  docs          - Generate and serve documentation"
	@echo "  pre-commit    - Run pre-commit checks"
	@echo "  ci            - Run CI pipeline"
	@echo "  help          - Show this help message" 