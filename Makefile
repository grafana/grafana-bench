# Makefile for building Grafana Bench Docker images and running CI

# Variables
BENCH_REVISION ?= $(shell git rev-parse --short HEAD)
DOCKER_BUILD_ARGS = --build-arg BENCH_REVISION=$(BENCH_REVISION)

# Image tags
SLIM_DEV_TAG = grafana-bench:dev-$(BENCH_REVISION)
SLIM_PROD_TAG = grafana-bench:$(BENCH_REVISION)
PLAYWRIGHT_DEV_TAG = grafana-bench-playwright:dev-$(BENCH_REVISION)
PLAYWRIGHT_PROD_TAG = grafana-bench-playwright:$(BENCH_REVISION)

.PHONY: build-all build-slim-dev build-slim-prod build-playwright-dev build-playwright-prod test docs libsonnet install-deps test-argo-local clean help

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

# Generate documentation
docs:
	@echo "📚 Generating documentation..."
	go run ./generators/doc -o docs

# Generate libsonnet library
libsonnet: install-deps
	@echo "🔧 Generating libsonnet library..."
	@go run ./generators/libsonnet -o libsonnet
	@echo "🧪 Testing generated libsonnet functions..."
	@cd libsonnet && jsonnet bench_functions_test.jsonnet | grep -q '"allTestsPassed": true' && echo "✅ All libsonnet function tests passed" || (echo "❌ Libsonnet function tests failed" && exit 1)

# Generate libsonnet library with specific version (for releases)
libsonnet-release: install-deps
	@echo "🔧 Generating libsonnet library for version $(VERSION)..."
	@go run ./generators/libsonnet -version $(VERSION) -o libsonnet
	@echo "🧪 Testing generated libsonnet functions..."
	@cd libsonnet && jsonnet bench_functions_test.jsonnet | grep -q '"allTestsPassed": true' && echo "✅ All libsonnet function tests passed" || (echo "❌ Libsonnet function tests failed" && exit 1)

# Test Argo workflow validation locally
test-argo-local: install-deps
	@echo "🧪 Testing Argo workflow validation locally..."
	./scripts/test-argo-local.sh

# Install development dependencies
install-deps:
	@echo "🔍 Debugging Go environment:"
	@echo "GOPATH: $$(go env GOPATH)"
	@echo "GOBIN: $$(go env GOBIN)"  
	@echo "PATH: $$PATH"
	@echo "🔍 Checking for jsonnet..."
	@if ! command -v jsonnet >/dev/null 2>&1; then \
		echo "📦 Installing jsonnet..."; \
		if command -v go >/dev/null 2>&1; then \
			go install github.com/google/go-jsonnet/cmd/jsonnet@latest; \
			export PATH="$$(go env GOPATH)/bin:$$PATH"; \
			if command -v jsonnet >/dev/null 2>&1; then \
				echo "✅ jsonnet installed at $$(which jsonnet)"; \
			else \
				echo "⚠️ jsonnet installed but not found in PATH. Check: $$(go env GOPATH)/bin/jsonnet"; \
			fi; \
		else \
			echo "❌ Go not found. Please install Go first or install jsonnet manually."; \
			exit 1; \
		fi; \
	else \
		echo "✅ jsonnet already installed at $$(which jsonnet)"; \
	fi
	@echo "🔍 Checking for jsonnetfmt..."
	@if ! command -v jsonnetfmt >/dev/null 2>&1; then \
		echo "📦 Installing jsonnetfmt..."; \
		if command -v go >/dev/null 2>&1; then \
			go install github.com/google/go-jsonnet/cmd/jsonnetfmt@latest; \
			export PATH="$$(go env GOPATH)/bin:$$PATH"; \
			if command -v jsonnetfmt >/dev/null 2>&1; then \
				echo "✅ jsonnetfmt installed at $$(which jsonnetfmt)"; \
			else \
				echo "⚠️ jsonnetfmt installed but not found in PATH. Check: $$(go env GOPATH)/bin/jsonnetfmt"; \
			fi; \
		else \
			echo "❌ Go not found. Please install Go first or install jsonnetfmt manually."; \
			exit 1; \
		fi; \
	else \
		echo "✅ jsonnetfmt already installed at $$(which jsonnetfmt)"; \
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
	@echo "  libsonnet           - Generate libsonnet library"
	@echo "  install-deps        - Install development dependencies (jsonnetfmt, etc.)"
	@echo "  test-argo-local     - Test Argo workflow validation locally"
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
