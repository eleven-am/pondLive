.PHONY: test test-go test-unit test-integration test-all coverage clean lint release-patch release-minor release-major build install client-install client-test client-typecheck client-build client-check contract-fixtures protocol-fixtures client-wire

# Run all unit tests
test: test-go client-test client-typecheck

test-go:
	go test ./...

client-install: client/node_modules

client/node_modules: client/package-lock.json
	cd client && npm ci
	@touch client/node_modules

client-test: client/node_modules
	cd client && npx vitest run

client-typecheck: client/node_modules
	cd client && npx tsc --noEmit

client-build: client/node_modules
	cd client && node esbuild.config.js

client-check: client/node_modules
	@rm -rf client/.bundle-check && mkdir -p client/.bundle-check && cp -R internal/server/static/. client/.bundle-check; \
	trap 'rm -rf internal/server/static; mkdir -p internal/server/static; cp -R client/.bundle-check/. internal/server/static; rm -rf client/.bundle-check' EXIT; \
	(cd client && node esbuild.config.js >/dev/null) || { echo "client build failed"; exit 1; }; \
	if diff -rq client/.bundle-check internal/server/static; then \
		echo "client bundle is up to date"; \
	else \
		echo "committed client bundle is stale: run 'make client-build' and commit the result"; exit 1; \
	fi

contract-fixtures:
	CONTRACT_FIXTURES_DIR=$(CURDIR)/client/test/fixtures/contract go test ./internal/contract/ -run TestGenerateFixtures -count=1

client-wire:
	WIRE_OUT=$(CURDIR)/client/src go test ./internal/protocol/ -run TestClientWireFileIsGeneratedFromTheGoTables -count=1

protocol-fixtures:
	PROTOCOL_FIXTURES_DIR=$(CURDIR)/client/test/fixtures/protocol go test ./internal/server/ -run TestGenerateProtocolTranscripts -count=1

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
build: client-build
	go build -o bin/counter ./examples/counter

# Install dependencies
install:
	go mod download
	go mod tidy

# Clean build artifacts and coverage reports
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Release commands
release-patch:
	@echo "Creating patch release..."
	@latest_tag=$$(git tag -l "v*.*.*" | sort -V | tail -1); \
	current_version=$${latest_tag:-v0.0.0}; \
	new_version=$$(echo $$current_version | awk -F. '{$$3 = $$3 + 1; print $$1"."$$2"."$$3}'); \
	echo "Bumping from $$current_version to $$new_version"; \
	git tag $$new_version; \
	git push origin $$new_version; \
	echo "Released $$new_version"

release-minor:
	@echo "Creating minor release..."
	@latest_tag=$$(git tag -l "v*.*.*" | sort -V | tail -1); \
	current_version=$${latest_tag:-v0.0.0}; \
	new_version=$$(echo $$current_version | awk -F. '{$$2 = $$2 + 1; $$3 = 0; print $$1"."$$2"."$$3}'); \
	echo "Bumping from $$current_version to $$new_version"; \
	git tag $$new_version; \
	git push origin $$new_version; \
	echo "Released $$new_version"

release-major:
	@echo "Creating major release..."
	@latest_tag=$$(git tag -l "v*.*.*" | sort -V | tail -1); \
	current_version=$${latest_tag:-v0.0.0}; \
	new_version=$$(echo $$current_version | awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0; print $$1"."$$2"."$$3}' | sed 's/^v/v/'); \
	echo "Bumping from $$current_version to $$new_version"; \
	git tag $$new_version; \
	git push origin $$new_version; \
	echo "Released $$new_version"

# Help command
help:
	@echo "Available commands:"
	@echo "  make test              - Run Go tests, client tests and client typecheck"
	@echo "  make test-go           - Run Go unit tests only"
	@echo "  make client-test       - Run client unit and contract tests"
	@echo "  make client-typecheck  - Typecheck the client"
	@echo "  make client-build      - Build the client bundle into internal/server/static"
	@echo "  make client-check      - Fail if the committed bundle differs from a fresh build"
	@echo "  make contract-fixtures - Regenerate Go to client contract fixtures"
	@echo "  make protocol-fixtures - Regenerate Go to client protocol transcripts"
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
	@echo "  make release-patch     - Create patch release (v0.0.X)"
	@echo "  make release-minor     - Create minor release (v0.X.0)"
	@echo "  make release-major     - Create major release (vX.0.0)"
	@echo "  make help              - Show this help message"

.DEFAULT_GOAL := test
