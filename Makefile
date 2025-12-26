.PHONY: all fmt lint test build install-hooks check

# Default target
all: fmt lint test build

# Format code
fmt:
	@echo "Formatting..."
	@gofmt -w .

# Run linter
lint:
	@echo "Linting..."
	@golangci-lint run

# Run tests
test:
	@echo "Testing..."
	@go test ./...

# Build binary
build:
	@echo "Building..."
	@go build -o pipe .

# Install git hooks
install-hooks:
	@echo "Installing git hooks..."
	@cp scripts/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed!"

# Run all checks (used by pre-commit hook)
check: fmt lint test
	@echo "All checks passed!"
