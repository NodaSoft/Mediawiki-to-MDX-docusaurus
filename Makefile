.PHONY: lint fmt vet golangci-lint install-linter build clean test help

# Default target
help:
	@echo "Available targets:"
	@echo "  make build          - Build the project"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make test           - Run tests"
	@echo "  make lint           - Run all linters (fmt, vet, golangci-lint)"
	@echo "  make fmt            - Run go fmt to check formatting"
	@echo "  make vet            - Run go vet to check for suspicious code"
	@echo "  make golangci-lint  - Run golangci-lint (requires installation)"
	@echo "  make install-linter - Install golangci-lint"

# Run all linters
lint: fmt vet golangci-lint

# Check code formatting
fmt:
	@echo "Running go fmt..."
	@gofmt -l -s .
	@if [ -n "$$(gofmt -l -s .)" ]; then \
		echo "Code is not formatted. Run 'gofmt -w -s .' to fix."; \
		exit 1; \
	fi
	@echo "✓ Code is properly formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ go vet passed"

# Run golangci-lint
golangci-lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "✓ golangci-lint passed"; \
	else \
		echo "golangci-lint is not installed. Run 'make install-linter' to install it."; \
		exit 1; \
	fi

# Install golangci-lint
install-linter:
	@echo "Installing golangci-lint..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	@echo "✓ golangci-lint installed successfully"

# Build the project
build:
	@echo "Building project..."
	@go build -o bin/wikiToMdx ./cmd
	@echo "✓ Build complete: bin/wikiToMdx"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "✓ Tests complete"
