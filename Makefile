.PHONY: help build install clean test test-coverage lint fmt pre-commit docker-build docker-run dev setup-hooks man

# Variables
BINARY_NAME=a9s
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X github.com/keanuharrell/a9s/cmd.Version=$(VERSION) -X github.com/keanuharrell/a9s/cmd.BuildTime=$(BUILD_TIME)"
GO_FILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
GOBIN?=$(shell go env GOPATH)/bin

## help: Display this help message
help:
	@echo "Available targets:"
	@echo ""
	@grep -E '^##' Makefile | sed 's/##//'
	@echo ""

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "✓ Build complete: ./$(BINARY_NAME)"

## install: Install the binary to $(GOBIN)
install: build
	@echo "Installing $(BINARY_NAME) to $(GOBIN)..."
	@cp $(BINARY_NAME) $(GOBIN)/
	@echo "✓ Installed to $(GOBIN)/$(BINARY_NAME)"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@go clean
	@echo "✓ Clean complete"

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -timeout 300s ./...
	@echo "✓ Tests complete"

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

## lint: Run linters (go vet + staticcheck)
lint:
	@echo "Running linters..."
	@echo "→ go fmt (check only)..."
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)
	@echo "→ go vet..."
	@go vet ./...
	@echo "→ staticcheck..."
	@if ! command -v staticcheck >/dev/null 2>&1; then \
		echo "Installing staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi
	@$(GOBIN)/staticcheck ./...
	@echo "✓ Lint complete"

## fmt: Format code with gofmt
fmt:
	@echo "Formatting code..."
	@gofmt -s -w $(GO_FILES)
	@echo "✓ Format complete"

## pre-commit: Run all checks before committing
pre-commit: fmt vet lint test
	@echo "✓ All pre-commit checks passed!"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .
	@echo "✓ Docker image built: $(BINARY_NAME):$(VERSION)"

## docker-run: Run Docker container
docker-run:
	@docker run --rm -v ~/.aws:/root/.aws:ro $(BINARY_NAME):latest --help

## dev: Run in development mode
dev: build
	@./$(BINARY_NAME)

## setup-hooks: Install git hooks
setup-hooks:
	@echo "Setting up git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/bash' > .git/hooks/pre-commit
	@echo 'make pre-commit' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✓ Git hooks installed"
	@echo "  - Pre-commit hook will run: fmt, lint, test"

## release-test: Test release build locally
release-test:
	@echo "Testing release build..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser/v2@latest; \
	fi
	@goreleaser build --snapshot --clean --single-target
	@echo "✓ Release test complete"

## release-check: Check release configuration
release-check:
	@echo "Checking GoReleaser configuration..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser/v2@latest; \
	fi
	@goreleaser check
	@echo "✓ Configuration check complete"

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@echo "✓ Dependencies ready"

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✓ Dependencies updated"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

## sec: Run security checks with gosec
sec:
	@echo "Running security checks..."
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -quiet ./...
	@echo "✓ Security checks complete"

## all: Run all checks and build
all: deps fmt lint vet test build
	@echo "✓ All tasks complete!"
