# Variables
APP_NAME := badge-service
BINARY_NAME := $(APP_NAME)
DOCKER_IMAGE := registry.gitlab.software.geant.org/software-licensing/softwarecerthub
DOCKER_TAG := latest
PORT ?= 9000

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

# Version variables
VERSION := $(shell git describe --tags --always 2>/dev/null | sed 's/^v//' || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags "-s -w \
	-X github.com/finki/badges/internal/version.Version=$(VERSION) \
	-X github.com/finki/badges/internal/version.Commit=$(COMMIT) \
	-X github.com/finki/badges/internal/version.BuildDate=$(BUILD_DATE)"

.PHONY: all build clean run test build-image push-image docker-run docker-stop docker-restart docker-logs version bump-patch bump-minor bump-major

# Default target
all: build

# Build the Go binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME) ./cmd/server

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(GOBIN)
	@go clean

# Run the application locally
run:
	@echo "Running $(BINARY_NAME) on port $(PORT)..."
	@PORT=$(PORT) go run $(LDFLAGS) ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Print current version
version:
	@echo $(VERSION)

# Bump patch version (v0.1.0 -> v0.1.1)
bump-patch:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$latest | sed 's/^v//' | cut -d. -f1); \
	minor=$$(echo $$latest | sed 's/^v//' | cut -d. -f2); \
	patch=$$(echo $$latest | sed 's/^v//' | cut -d. -f3); \
	new_patch=$$((patch + 1)); \
	new_tag="v$$major.$$minor.$$new_patch"; \
	echo "Bumping $$latest -> $$new_tag"; \
	git tag -a $$new_tag -m "Release $$new_tag"

# Bump minor version (v0.1.0 -> v0.2.0)
bump-minor:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$latest | sed 's/^v//' | cut -d. -f1); \
	minor=$$(echo $$latest | sed 's/^v//' | cut -d. -f2); \
	new_minor=$$((minor + 1)); \
	new_tag="v$$major.$$new_minor.0"; \
	echo "Bumping $$latest -> $$new_tag"; \
	git tag -a $$new_tag -m "Release $$new_tag"

# Bump major version (v0.1.0 -> v1.0.0)
bump-major:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$latest | sed 's/^v//' | cut -d. -f1); \
	new_major=$$((major + 1)); \
	new_tag="v$$new_major.0.0"; \
	echo "Bumping $$latest -> $$new_tag"; \
	git tag -a $$new_tag -m "Release $$new_tag"

# Build Docker image
build-image:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Push Docker image to registry
push-image:
	@echo "Pushing Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Run Docker image
docker-run:
	@echo "Running Docker image $(DOCKER_IMAGE):$(DOCKER_TAG) on port $(PORT)..."
	@docker run -e PORT=$(PORT) -p $(PORT):$(PORT) --name $(APP_NAME) -d $(DOCKER_IMAGE):$(DOCKER_TAG)

# Stop and remove Docker container
docker-stop:
	@echo "Stopping Docker container $(APP_NAME)..."
	@docker stop $(APP_NAME) || true
	@docker rm $(APP_NAME) || true

# Restart Docker container (stop and run)
docker-restart: docker-stop docker-run

# View Docker container logs
docker-logs:
	@echo "Viewing logs for Docker container $(APP_NAME)..."
	@docker logs -f $(APP_NAME)

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download

# Generate documentation
docs:
	@echo "Generating documentation..."
	@go doc -all > docs/api.txt

# Create database directory if it doesn't exist
db-init:
	@echo "Initializing database directory..."
	@mkdir -p db

# Help target
help:
	@echo "Available targets:"
	@echo "  build          - Build the Go binary"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Run the application locally"
	@echo "  test           - Run tests"
	@echo "  version        - Print current version"
	@echo "  bump-patch     - Bump patch version and create git tag"
	@echo "  bump-minor     - Bump minor version and create git tag"
	@echo "  bump-major     - Bump major version and create git tag"
	@echo "  build-image    - Build Docker image"
	@echo "  push-image     - Push Docker image to registry"
	@echo "  docker-run     - Run the built Docker image"
	@echo "  docker-stop    - Stop and remove the Docker container"
	@echo "  docker-restart - Restart the Docker container"
	@echo "  docker-logs    - View logs for the Docker container"
	@echo "  deps           - Install dependencies"
	@echo "  docs           - Generate documentation"
	@echo "  db-init        - Initialize database directory"
	@echo "  help           - Show this help message"
