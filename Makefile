# ABOUTME: Build automation for Memory CLI and HMLR MCP server
# ABOUTME: Provides common development tasks (build, test, install, clean, benchmark)

.PHONY: help build build-all test test-unit test-scenario test-load test-bench install clean dev lint fmt check run-mcp run-cli deps

# Binary name
BINARY_NAME=memory
BUILD_DIR=bin

# Go build flags
LDFLAGS=-ldflags "-s -w"
GCFLAGS=-gcflags "all=-trimpath=$(PWD)"

# Default target
help:
	@echo "Memory (HMLR) - Build Targets"
	@echo ""
	@echo "Building:"
	@echo "  make build        - Build memory binary"
	@echo "  make build-all    - Build memory + benchmark binaries"
	@echo "  make install      - Install memory to GOPATH/bin"
	@echo "  make dev          - Development workflow (fmt, test, build, install)"
	@echo ""
	@echo "Running:"
	@echo "  make run-mcp      - Run MCP server"
	@echo "  make run-cli      - Run CLI with arguments (use ARGS='...')"
	@echo ""
	@echo "Testing:"
	@echo "  make test         - Run all tests (unit + scenario + load)"
	@echo "  make test-unit    - Run unit tests only"
	@echo "  make test-scenario - Run scenario tests (requires OPENAI_API_KEY)"
	@echo "  make test-load    - Run load tests (requires OPENAI_API_KEY)"
	@echo "  make test-bench   - Run RAGAS benchmarks"
	@echo "  make test-v       - Run all tests (verbose)"
	@echo "  make test-cover   - Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt          - Format code with gofmt"
	@echo "  make lint         - Run golangci-lint (if available)"
	@echo "  make check        - Run fmt, lint, and test"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps         - Download and tidy dependencies"
	@echo "  make deps-update  - Update all dependencies"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean        - Remove built binaries and test cache"
	@echo "  make clean-all    - Clean + remove XDG test data"
	@echo ""
	@echo "Release:"
	@echo "  make release-snapshot - Create snapshot release (local test)"
	@echo "  make release-test     - Test release configuration"
	@echo ""

# Build the CLI binary
build:
	@echo "🔨 Building $(BINARY_NAME) CLI..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) $(GCFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/memory
	@echo "✓ Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build all binaries (CLI + benchmarks)
build-all: build
	@echo "🔨 Building benchmark binary..."
	go build $(LDFLAGS) $(GCFLAGS) -o $(BUILD_DIR)/hmlr-benchmark ./cmd/benchmark
	@echo "✓ Built $(BUILD_DIR)/hmlr-benchmark"

# Install to GOPATH/bin
install:
	@echo "📦 Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(GCFLAGS) ./cmd/memory
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Development workflow: format, test, build, install
dev: fmt test build install
	@echo "✓ Development build complete"

# Run MCP server
run-mcp:
	@echo "🚀 Starting MCP server..."
	@./$(BUILD_DIR)/$(BINARY_NAME) mcp

# Run CLI with arguments
run-cli:
	@./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Testing targets
test:
	@echo "🧪 Running all tests..."
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "⚠️  OPENAI_API_KEY not set - some tests will be skipped"; \
	fi
	go test -timeout 10m ./internal/...
	@if [ -n "$$OPENAI_API_KEY" ]; then \
		echo "🧪 Running scenario tests..."; \
		go test -timeout 10m -v ./.scratch/ -run TestScenario | grep -E "(RUN|PASS|FAIL)"; \
	fi

test-unit:
	@echo "🧪 Running unit tests..."
	go test ./internal/...

test-scenario:
	@echo "🧪 Running scenario tests (requires OPENAI_API_KEY)..."
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "❌ OPENAI_API_KEY not set"; \
		exit 1; \
	fi
	go test -timeout 10m -v ./.scratch/ -run TestScenario

test-load:
	@echo "🔥 Running load tests (requires OPENAI_API_KEY)..."
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "❌ OPENAI_API_KEY not set"; \
		exit 1; \
	fi
	go test -timeout 10m -v ./.scratch/ -run TestLoadTest

test-bench:
	@echo "📊 Running RAGAS benchmarks (requires OPENAI_API_KEY)..."
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "❌ OPENAI_API_KEY not set"; \
		exit 1; \
	fi
	@if [ ! -f "$(BUILD_DIR)/hmlr-benchmark" ]; then \
		echo "Building benchmark binary..."; \
		$(MAKE) build-all; \
	fi
	./$(BUILD_DIR)/hmlr-benchmark

test-v:
	@echo "🧪 Running tests (verbose)..."
	go test -v -timeout 10m ./...

test-cover:
	@echo "🧪 Running tests with coverage..."
	go test -v -timeout 10m -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Code formatting
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

# Linting (optional - requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "🔍 Running golangci-lint..."; \
		golangci-lint run ./...; \
		echo "✓ Linting complete"; \
	else \
		echo "⚠️  golangci-lint not installed, skipping..."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run all checks
check: fmt lint test
	@echo "✓ All checks passed"

# Dependencies
deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

# Update dependencies
deps-update:
	@echo "📦 Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✓ Dependencies updated"

# Show module graph
deps-graph:
	@echo "📊 Dependency graph:"
	go mod graph

# Cleanup
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)/
	rm -f coverage.out coverage.html
	rm -f benchmark_results.json
	go clean -testcache
	@echo "✓ Clean complete"

# Deep clean (including XDG test data)
clean-all: clean
	@echo "🧹 Removing XDG test data..."
	rm -rf /tmp/hmlr_test_*
	@echo "✓ Deep clean complete"

# Quick CLI usage examples
examples:
	@echo "Memory CLI Examples:"
	@echo ""
	@echo "Add a memory:"
	@echo "  ./bin/memory add 'Met with Alice about project X'"
	@echo "  ./bin/memory add --tags=meeting,project 'Discussed timeline'"
	@echo "  echo 'Note from stdin' | ./bin/memory add"
	@echo ""
	@echo "Search memories:"
	@echo "  ./bin/memory search 'programming'"
	@echo "  ./bin/memory search --limit 10 'API keys'"
	@echo ""
	@echo "List topics:"
	@echo "  ./bin/memory list"
	@echo "  ./bin/memory list --all --format json"
	@echo ""
	@echo "Start MCP server:"
	@echo "  ./bin/memory mcp"
	@echo ""

# Development server with auto-restart (requires entr)
watch:
	@if command -v entr >/dev/null 2>&1; then \
		echo "👀 Watching for changes (Ctrl+C to stop)..."; \
		find . -name '*.go' | entr -r make build run-mcp; \
	else \
		echo "❌ 'entr' not installed"; \
		echo "Install with: brew install entr (macOS) or apt install entr (Linux)"; \
	fi

# Release targets
release-snapshot:
	@echo "📦 Creating snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "❌ goreleaser not installed"; \
		echo "Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

release-test:
	@echo "🧪 Testing release configuration..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
		goreleaser build --snapshot --clean; \
	else \
		echo "❌ goreleaser not installed"; \
		echo "Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi
