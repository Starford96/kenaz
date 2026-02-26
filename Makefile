# Build variables
VERSION?=latest
BUILD_DIR=build
BINARY_NAME=app
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"
BUILD_FLAGS=-trimpath -installsuffix cgo -tags sqlite_fts5
LINT_BIN_PATH?=$(shell go env GOPATH)/bin

.PHONY: build build-linux run test clean docker-build docker-push docker-up docker-down docker-logs fmt lint install-lint deps help openapi openapi-check client-gen frontend-build frontend-prod dev-backend dev-frontend prod release

# Build for Linux (Docker).
build-linux:
	GOOS=linux make build

# Build the application.
build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build \
		$(BUILD_FLAGS) \
		$(LDFLAGS) \
		-o $(BUILD_DIR)/$(BINARY_NAME) \
		./cmd/app

# Run the application locally.
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) --config config/config.yaml

# Run tests.
test:
	CGO_ENABLED=1 go test -v -tags sqlite_fts5 ./...

# Clean build artifacts.
clean:
	rm -rf bin/
	rm -rf $(BUILD_DIR)/$(BINARY_NAME)

# Docker
IMAGE ?= kenaz:$(VERSION)
REGISTRY ?= 192.168.48.58:5005
PLATFORM ?= linux/amd64

docker-build:
	docker buildx build --platform $(PLATFORM) -t $(IMAGE) -f $(BUILD_DIR)/Dockerfile .

# Build and push to local registry.
docker-push:
	docker buildx build --platform $(PLATFORM) -t $(REGISTRY)/kenaz:$(VERSION) -f $(BUILD_DIR)/Dockerfile --push .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Install dependencies.
deps:
	go mod download
	go mod tidy

# Format code.
fmt:
	@if ! command -v gofumpt > /dev/null; then \
		echo "gofumpt is not installed. Installing..."; \
		go install mvdan.cc/gofumpt@latest; \
	fi
	gofumpt -l -w .

# Install golangci-lint.
install-lint:
	@echo "Installing golangci-lint to $(LINT_BIN_PATH)..."
	@rm -f $(LINT_BIN_PATH)/golangci-lint
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LINT_BIN_PATH) latest
	@$(LINT_BIN_PATH)/golangci-lint version

# Run linter.
lint:
	@if [ -f "$(LINT_BIN_PATH)/golangci-lint" ]; then \
		$(LINT_BIN_PATH)/golangci-lint run ./... --new-from-rev=origin/main; \
	elif command -v golangci-lint > /dev/null; then \
		golangci-lint run ./... --new-from-rev=origin/main; \
	else \
		echo "golangci-lint is not installed. Run 'make install-lint' first."; \
		exit 1; \
	fi

# Generate TypeScript types from OpenAPI spec.
# Generate OpenAPI 3.1 spec from backend swag annotations.
openapi:
	$(shell go env GOPATH)/bin/swag init -g cmd/app/main.go -o api/schema/kenaz/swag --outputTypes json --quiet
	node scripts/swagger2openapi.mjs api/schema/kenaz/swag/swagger.json api/schema/kenaz/openapi.yaml

# Drift check: regenerate and fail if spec changed.
openapi-check: openapi
	@git diff --exit-code api/schema/kenaz/openapi.yaml || (echo "ERROR: openapi.yaml is out of date — run 'make openapi' and commit" && exit 1)

# Generate typed frontend client from OpenAPI spec.
client-gen: openapi
	cd frontend && npm run generate

# Build the frontend (quick, uses existing node_modules).
frontend-build:
	cd frontend && npm run build

# Clean production frontend build.
frontend-prod:
	cd frontend && npm ci && npm run build

# Development: backend only (no frontend serving, use with dev-frontend).
dev-backend: build
	FRONTEND_ENABLED=false ./$(BUILD_DIR)/$(BINARY_NAME) --config config/config.yaml

# Development: frontend only (Vite dev server with HMR, proxies API to :8080).
dev-frontend:
	cd frontend && npm run dev

# Production: build frontend, build backend, serve everything from backend.
prod: frontend-build build
	./$(BUILD_DIR)/$(BINARY_NAME) --config config/config.yaml

# Full release: build frontend + backend image and push to local registry.
release:
	@echo "==> Running tests…"
	@$(MAKE) test
	@echo "==> Building and pushing kenaz:$(VERSION) to $(REGISTRY)…"
	@$(MAKE) docker-push
	@echo "==> Release complete: $(REGISTRY)/kenaz:$(VERSION)"

# Help.
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux (Docker)"
	@echo "  run           - Run locally"
	@echo "  test          - Run tests"
	@echo "  lint          - Run golangci-lint"
	@echo "  install-lint  - Install golangci-lint"
	@echo "  fmt           - Format code with gofumpt"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-push   - Build and push to local registry ($(REGISTRY))"
	@echo "  docker-up     - Start containers"
	@echo "  docker-down   - Stop containers"
	@echo "  docker-logs   - View logs"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  openapi       - Generate OpenAPI 3.1 spec from backend annotations"
	@echo "  openapi-check - Drift check: fail if spec is out of date"
	@echo "  client-gen    - Generate typed frontend client from OpenAPI spec"
	@echo "  frontend-build - Build frontend (quick)"
	@echo "  frontend-prod  - Clean production frontend build (npm ci + build)"
	@echo "  dev-backend   - Run backend only (frontend disabled)"
	@echo "  dev-frontend  - Run Vite dev server with HMR"
	@echo "  prod          - Build everything and run in production mode"
	@echo "  release       - Run tests, build Docker image, push to registry"
	@echo "  help          - Show this help"
