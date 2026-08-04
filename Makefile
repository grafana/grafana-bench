# Makefile for building Grafana Bench Docker images and running CI

# Variables
BENCH_REVISION ?= $(shell git rev-parse --short HEAD)
DOCKER_BUILD_ARGS = --build-arg BENCH_REVISION=$(BENCH_REVISION)

# Image tags
SLIM_DEV_TAG = grafana-bench:dev-$(BENCH_REVISION)
SLIM_PROD_TAG = grafana-bench:$(BENCH_REVISION)
PLAYWRIGHT_DEV_TAG = grafana-bench-playwright:dev-$(BENCH_REVISION)
PLAYWRIGHT_PROD_TAG = grafana-bench-playwright:$(BENCH_REVISION)

.PHONY: build-all build-slim-dev build-slim-prod build-playwright-dev build-playwright-prod test docs libsonnet install-deps check-licenses clean help analyze-smoke analyze-smoke-up analyze-smoke-down

# Build all images
build-all: build-slim-dev build-slim-prod build-playwright-dev build-playwright-prod
	@echo "✅ All images built successfully"

# Build slim development image (with fixuid)
build-slim-dev:
	@echo "🔨 Building slim development image..."
	docker build $(DOCKER_BUILD_ARGS) -f Dockerfile.dev -t $(SLIM_DEV_TAG) .
	@echo "✅ Built: $(SLIM_DEV_TAG)"

# Build slim production image
build-slim-prod:
	@echo "🔨 Building slim production image..."
	docker build $(DOCKER_BUILD_ARGS) -f Dockerfile -t $(SLIM_PROD_TAG) .
	@echo "✅ Built: $(SLIM_PROD_TAG)"

# Build playwright development image (with fixuid)
build-playwright-dev:
	@echo "🔨 Building playwright development image..."
	docker build $(DOCKER_BUILD_ARGS) -f Dockerfile-playwright.dev -t $(PLAYWRIGHT_DEV_TAG) .
	@echo "✅ Built: $(PLAYWRIGHT_DEV_TAG)"

# Build playwright production image
build-playwright-prod:
	@echo "🔨 Building playwright production image..."
	docker build $(DOCKER_BUILD_ARGS) -f Dockerfile-playwright -t $(PLAYWRIGHT_PROD_TAG) .
	@echo "✅ Built: $(PLAYWRIGHT_PROD_TAG)"

# Run tests including integration tests
test:
	@echo "🧪 Running tests..."
	go test ./...

# End-to-end smoke test for `bench analyze` against a real Loki.
# See test/analyze/README.md for what this does and how to debug failures.
analyze-smoke: analyze-smoke-up
	@echo "🌱 Seeding fixtures..."
	@go run ./test/analyze/seed -loki http://localhost:3100
	@echo "🔍 Running grafana-bench analyze..."
	@OUT=$$(go run . analyze \
		--analyze-loki-url http://localhost:3100 \
		--analyze-service grafana-pro \
		--analyze-run-stage ci \
		--analyze-window 24h); \
	echo "$$OUT"; \
	echo "$$OUT" | grep -q '"msg":"defectConfirmed"' || { echo "❌ no defectConfirmed event emitted"; $(MAKE) analyze-smoke-down; exit 1; }; \
	echo "$$OUT" | grep -q '"confidence":"confirmed"' || { echo "❌ event was not confidence=confirmed"; $(MAKE) analyze-smoke-down; exit 1; }; \
	echo "$$OUT" | grep -q '"grafanaVersion":"13.0.0-23542128402"' || { echo "❌ event missing expected bad version"; $(MAKE) analyze-smoke-down; exit 1; }
	@$(MAKE) analyze-smoke-down
	@echo "✅ analyze smoke test passed"

analyze-smoke-up:
	@echo "🐳 Starting Loki..."
	@docker compose -f test/analyze/docker-compose.yml up -d --wait

analyze-smoke-down:
	@echo "🧹 Stopping Loki..."
	@docker compose -f test/analyze/docker-compose.yml down -v >/dev/null 2>&1 || true

# Generate documentation
docs: install-deps
	@echo "📚 Generating documentation..."
	git fetch --tags
	go run ./generators/doc -o docs

# Generate libsonnet library (local versions only)
libsonnet: install-deps
	@echo "🔧 Generating libsonnet for $(or $(TARGET_VERSION),experimental) (local versions only)..."
	@go run ./generators/libsonnet generate --target-version "$(or $(TARGET_VERSION),experimental)" -o libsonnet >/dev/null 2>&1
	@VERSION="$(or $(TARGET_VERSION),experimental)"; LABEL=$$([ "$$VERSION" = "experimental" ] && git rev-parse HEAD || echo "$$VERSION"); go run ./generators/libsonnet versions --target-version "$$VERSION" --versions-list local --latest-version-sha "$$LABEL" -o libsonnet >/dev/null 2>&1
	@cd libsonnet && jsonnet "$(or $(TARGET_VERSION),experimental)/main_test.jsonnet" && echo "✅ All tests passed"

# Generate libsonnet library with remote version fetching
libsonnet-fetch: install-deps
	@echo "🔧 Generating libsonnet for $(or $(TARGET_VERSION),experimental) (fetching remote versions)..."
	@go run ./generators/libsonnet generate --target-version "$(or $(TARGET_VERSION),experimental)" -o libsonnet >/dev/null 2>&1
	@VERSION="$(or $(TARGET_VERSION),experimental)"; LABEL=$$([ "$$VERSION" = "experimental" ] && git rev-parse HEAD || echo "$$VERSION"); go run ./generators/libsonnet versions --target-version "$$VERSION" --versions-list fetch --latest-version-sha "$$LABEL" -o libsonnet >/dev/null 2>&1
	@cd libsonnet && jsonnet "$(or $(TARGET_VERSION),experimental)/main_test.jsonnet" && echo "✅ All tests passed"


# Check dependency licenses for AGPL-3.0 compatibility
check-licenses:
	@if ! command -v go-licenses >/dev/null 2>&1; then \
		echo "📦 Installing go-licenses..."; \
		go install github.com/google/go-licenses@latest; \
	fi
	go-licenses check ./... \
		--allowed_licenses=MIT,ISC,BSD-3-Clause,BSD-2-Clause,Apache-2.0,AGPL-3.0,MPL-2.0 \
		--ignore buf.build/gen/go/prometheus/prometheus/protocolbuffers/go \
		--ignore github.com/grafana/grafana-bench

# Install development dependencies
install-deps:
	@if ! command -v jsonnet >/dev/null 2>&1; then \
		echo "📦 Installing jsonnet..."; \
		if command -v go >/dev/null 2>&1; then \
			go install github.com/google/go-jsonnet/cmd/jsonnet@latest; \
			export PATH="$$(go env GOPATH)/bin:$$PATH"; \
			if command -v jsonnet >/dev/null 2>&1; then \
				echo "✅ jsonnet installed"; \
			else \
				echo "⚠️ jsonnet installed but not found in PATH. Check: $$(go env GOPATH)/bin/jsonnet"; \
			fi; \
		else \
			echo "❌ Go not found. Please install Go first or install jsonnet manually."; \
			exit 1; \
		fi; \
	fi
	@if ! command -v jsonnetfmt >/dev/null 2>&1; then \
		echo "📦 Installing jsonnetfmt..."; \
		if command -v go >/dev/null 2>&1; then \
			go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest; \
			export PATH="$$(go env GOPATH)/bin:$$PATH"; \
			if command -v jsonnetfmt >/dev/null 2>&1; then \
				echo "✅ jsonnetfmt installed"; \
			else \
				echo "⚠️ jsonnetfmt installed but not found in PATH. Check: $$(go env GOPATH)/bin/jsonnetfmt"; \
			fi; \
		else \
			echo "❌ Go not found. Please install Go first or install jsonnetfmt manually."; \
			exit 1; \
		fi; \
	fi
	@if ! command -v jsonnet-lint >/dev/null 2>&1; then \
		echo "📦 Installing jsonnet-lint..."; \
		if command -v go >/dev/null 2>&1; then \
			go install github.com/google/go-jsonnet/cmd/jsonnet-lint@latest; \
			export PATH="$$(go env GOPATH)/bin:$$PATH"; \
			if command -v jsonnet-lint >/dev/null 2>&1; then \
				echo "✅ jsonnet-lint installed"; \
			else \
				echo "⚠️ jsonnet-lint installed but not found in PATH. Check: $$(go env GOPATH)/bin/jsonnet-lint"; \
			fi; \
		else \
			echo "❌ Go not found. Please install Go first or install jsonnet-lint manually."; \
			exit 1; \
		fi; \
	fi

# Clean up all built images
clean:
	@echo "🧹 Cleaning up images..."
	-docker rmi $(SLIM_DEV_TAG) $(SLIM_PROD_TAG) $(PLAYWRIGHT_DEV_TAG) $(PLAYWRIGHT_PROD_TAG) 2>/dev/null || true
	@echo "✅ Cleanup complete"

# Show image sizes
sizes: build-all
	@echo "📊 Image sizes:"
	@docker images --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}" | grep -E "(grafana-bench|REPOSITORY)"

# List all built images
list:
	@echo "📋 Built images:"
	@docker images | grep grafana-bench

# Show help
help:
	@echo "🚀 Grafana Bench Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all                  - Build all images (default)"
	@echo "  build-all           - Build all images"
	@echo "  build-slim-dev      - Build slim development image (with fixuid)"
	@echo "  build-slim-prod     - Build slim production image"
	@echo "  build-playwright-dev - Build playwright development image (with fixuid)"
	@echo "  build-playwright-prod - Build playwright production image"
	@echo "  test                - Run all tests including integration tests"
	@echo "  docs                - Generate documentation"
	@echo "  libsonnet           - Generate libsonnet library (local versions only)"
	@echo "  libsonnet-fetch     - Generate libsonnet library with remote version fetching"
	@echo "  install-deps        - Install development dependencies (jsonnetfmt, etc.)"
	@echo "  sizes               - Show image sizes"
	@echo "  list                - List all built images"
	@echo "  clean               - Remove all built images"
	@echo "  help                - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  BENCH_REVISION      - Git revision for build (default: short commit hash)"
	@echo ""
	@echo "Examples:"
	@echo "  make build-slim-dev"
	@echo "  make BENCH_REVISION=v1.2.3 build-all"
