.PHONY: test test-unit test-integration test-all coverage clean lint release-patch release-minor release-major build install

# Run all unit tests
test:
	go test ./...

# Alias for test
test-unit: test

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests for specific packages
test-internal:
	go test ./internal/...

test-pkg:
	go test ./pkg/...

test-examples:
	go test ./examples/...

# Run integration tests (requires PostgreSQL)
test-integration:
	go test -v -tags=integration ./...

# Run all tests
test-all: test-unit test-integration

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with coverage for specific packages
coverage-internal:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html

coverage-pkg:
	go test -coverprofile=coverage.out ./pkg/...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Build example applications
build:
	go build -o bin/counter ./examples/counter

# Install dependencies
install:
	go mod download
	go mod tidy

# Clean build artifacts and coverage reports
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Release commands (Go-specific tags only for monorepo)
release-patch:
	@echo "Creating Go patch release..."
	@latest_tag=$$(git tag -l "go/v*.*.*" | sort -V | tail -1); \
	current_version=$${latest_tag#go/}; \
	current_version=$${current_version:-v0.0.0}; \
	new_version=$$(echo $$current_version | awk -F. '{$$3 = $$3 + 1; print $$1"."$$2"."$$3}'); \
	go_tag="go/$$new_version"; \
	echo "Bumping from $$current_version to $$new_version (tag: $$go_tag)"; \
	git tag $$go_tag; \
	git push origin $$go_tag; \
	echo "Released $$go_tag"

release-minor:
	@echo "Creating Go minor release..."
	@latest_tag=$$(git tag -l "go/v*.*.*" | sort -V | tail -1); \
	current_version=$${latest_tag#go/}; \
	current_version=$${current_version:-v0.0.0}; \
	new_version=$$(echo $$current_version | awk -F. '{$$2 = $$2 + 1; $$3 = 0; print $$1"."$$2"."$$3}'); \
	go_tag="go/$$new_version"; \
	echo "Bumping from $$current_version to $$new_version (tag: $$go_tag)"; \
	git tag $$go_tag; \
	git push origin $$go_tag; \
	echo "Released $$go_tag"

release-major:
	@echo "Creating Go major release..."
	@latest_tag=$$(git tag -l "go/v*.*.*" | sort -V | tail -1); \
	current_version=$${latest_tag#go/}; \
	current_version=$${current_version:-v0.0.0}; \
	new_version=$$(echo $$current_version | awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0; print $$1"."$$2"."$$3}' | sed 's/^v/v/'); \
	go_tag="go/$$new_version"; \
	echo "Bumping from $$current_version to $$new_version (tag: $$go_tag)"; \
	git tag $$go_tag; \
	git push origin $$go_tag; \
	echo "Released $$go_tag"

# Help command
help:
	@echo "Available commands:"
	@echo "  make test              - Run all unit tests"
	@echo "  make test-verbose      - Run tests with verbose output"
	@echo "  make test-internal     - Run tests for internal packages only"
	@echo "  make test-pkg          - Run tests for pkg packages only"
	@echo "  make test-examples     - Run tests for examples only"
	@echo "  make test-integration  - Run integration tests (requires PostgreSQL)"
	@echo "  make test-all          - Run all tests (unit + integration)"
	@echo "  make coverage          - Generate test coverage report"
	@echo "  make lint              - Run golangci-lint"
	@echo "  make build             - Build example applications"
	@echo "  make install           - Install/update dependencies"
	@echo "  make clean             - Clean build artifacts"
	@echo "  make release-patch     - Create Go patch release (go/v0.0.X)"
	@echo "  make release-minor     - Create Go minor release (go/v0.X.0)"
	@echo "  make release-major     - Create Go major release (go/vX.0.0)"
	@echo "  make help              - Show this help message"

.DEFAULT_GOAL := test
