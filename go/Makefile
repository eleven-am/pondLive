.PHONY: test test-unit test-integration test-all coverage clean release-patch release-minor release-major

# Run unit tests only
test-unit:
	go test -v ./parser ./generator ./diff ./introspect

# Run integration tests (requires PostgreSQL)
test-integration:
	go test -v -tags=integration ./...

# Run all tests
test: test-unit

test-all: test-unit test-integration

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./parser ./generator ./diff ./introspect
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run specific package tests
test-parser:
	go test -v ./parser

test-generator:
	go test -v ./generator

test-diff:
	go test -v ./diff

test-introspect:
	go test -v ./introspect

# Run linter
lint:
	golangci-lint run ./...

# Release commands
release-patch:
	@echo "Creating patch release..."
	@current_version=$$(git tag -l "v*.*.*" | sort -V | tail -1 || echo "v0.0.0"); \
	new_version=$$(echo $$current_version | awk -F. '{$$3 = $$3 + 1; print $$1"."$$2"."$$3}'); \
	echo "Bumping from $$current_version to $$new_version"; \
	git tag $$new_version; \
	git push origin $$new_version; \
	git tag go$$new_version; \
	git push origin go$$new_version; \
	echo "Released $$new_version"

release-minor:
	@echo "Creating minor release..."
	@current_version=$$(git tag -l "v*.*.*" | sort -V | tail -1 || echo "v0.0.0"); \
	new_version=$$(echo $$current_version | awk -F. '{$$2 = $$2 + 1; $$3 = 0; print $$1"."$$2"."$$3}'); \
	echo "Bumping from $$current_version to $$new_version"; \
	git tag $$new_version; \
	git push origin $$new_version; \
	git tag go$$new_version; \
	git push origin go$$new_version; \
	echo "Released $$new_version"

release-major:
	@echo "Creating major release..."
	@current_version=$$(git tag -l "v*.*.*" | sort -V | tail -1 || echo "v0.0.0"); \
	new_version=$$(echo $$current_version | awk -F. '{$$1 = $$1 + 1; $$2 = 0; $$3 = 0; print $$1"."$$2"."$$3}' | sed 's/^v/v/'); \
	echo "Bumping from $$current_version to $$new_version"; \
	git tag $$new_version; \
	git push origin $$new_version; \
	git tag go$$new_version; \
	git push origin go$$new_version; \
	echo "Released $$new_version"

.DEFAULT_GOAL := test