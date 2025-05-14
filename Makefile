# Variables
APP_NAME := badge-service
BINARY_NAME := $(APP_NAME)
DOCKER_IMAGE := finki/$(APP_NAME)
DOCKER_TAG := latest

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

# Build flags
LDFLAGS := -ldflags "-s -w"

.PHONY: all build clean run test docker-build docker-push

# Default target
all: build

# Build the Go binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(GOBIN)/$(BINARY_NAME) ./cmd/server

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(GOBIN)
	@go clean

# Run the application locally
run:
	@echo "Running $(BINARY_NAME)..."
	@go run ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Build Docker image
build-image:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Push Docker image to registry
push-image:
	@echo "Pushing Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

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
	@echo "  build        - Build the Go binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  build-image  - Build Docker image"
	@echo "  push-image   - Push Docker image to registry"
	@echo "  deps         - Install dependencies"
	@echo "  docs         - Generate documentation"
	@echo "  db-init      - Initialize database directory"
	@echo "  help         - Show this help message"