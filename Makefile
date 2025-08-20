# Makefile for building Grafana Bench Docker images and running CI

# Variables
BENCH_REVISION ?= $(shell git rev-parse --short HEAD)
DOCKER_BUILD_ARGS = --build-arg BENCH_REVISION=$(BENCH_REVISION)

# Image tags
SLIM_DEV_TAG = grafana-bench:dev-$(BENCH_REVISION)
SLIM_PROD_TAG = grafana-bench:$(BENCH_REVISION)
PLAYWRIGHT_DEV_TAG = grafana-bench-playwright:dev-$(BENCH_REVISION)
PLAYWRIGHT_PROD_TAG = grafana-bench-playwright:$(BENCH_REVISION)

.PHONY: build-all build-slim-dev build-slim-prod build-playwright-dev build-playwright-prod clean help

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
	@echo "🚀 Grafana Bench Docker Build Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all                  - Build all images (default)"
	@echo "  build-all           - Build all images"
	@echo "  build-slim-dev      - Build slim development image (with fixuid)"
	@echo "  build-slim-prod     - Build slim production image"
	@echo "  build-playwright-dev - Build playwright development image (with fixuid)"
	@echo "  build-playwright-prod - Build playwright production image"
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
