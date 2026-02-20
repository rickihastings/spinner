.PHONY: build test lint format format-check install-hooks clean release snapshot bake-dev tui-preview

# Version information
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X github.com/rickihastings/spinner/internal/version.Version=$(VERSION) \
	-X github.com/rickihastings/spinner/internal/version.Commit=$(COMMIT) \
	-X github.com/rickihastings/spinner/internal/version.Date=$(DATE)

# Build the spinner binary
build:
	go clean -cache
	go build -ldflags "$(LDFLAGS)" -o dist/spinner

# Run unit tests
test:
	go test ./internal/... -v

# Run docker tests
test-docker:
	go test ./test/integration/ -v -run Docker

# Run gcp tests
test-gcp:
	go test ./test/integration/ -v -run GCP -timeout 60m

# Run all tests
test-all: test test-docker test-gcp

# Run linter
lint:
	go vet ./...
	golangci-lint run

# Run autofix for linting
lint-fix:
	golangci-lint run --fix

# Format code
format:
	go fmt ./...

# Check formatting (used by pre-commit hook)
format-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make format' to fix formatting" && gofmt -l . && exit 1)

# Install git pre-commit hooks
install-hooks:
	@mkdir -p .git/hooks
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed successfully"

# Clean build artifacts
clean:
	rm -rf dist/

# Create a release (requires goreleaser and a git tag)
release:
	goreleaser release --clean

# Create a snapshot release (for testing, no tag required)
snapshot:
	goreleaser release --snapshot --clean

# Preview TUI formatting by replaying a raw log file
# Usage: make tui-preview LOG=~/.spinner/<name>/logs/raw.log
tui-preview:
	go run ./tools/tuipreview $(LOG)

# Bake GCP dev image with Go, Docker, Tailscale, and golangci-lint
bake-dev: build
	./dist/spinner setup --name spinner-dev --bake-script ./scripts/dev-bake.sh
