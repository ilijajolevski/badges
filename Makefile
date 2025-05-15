# Variables
APP_NAME := badge-service
BINARY_NAME := $(APP_NAME)
DOCKER_IMAGE := ilijajolevski/$(APP_NAME)
DOCKER_TAG := latest

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

# Build flags
LDFLAGS := -ldflags "-s -w"

.PHONY: all build clean run test build-image push-image docker-run docker-stop docker-restart docker-logs

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

# Run Docker image
docker-run:
	@echo "Running Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker run -p 8080:8080 --name $(APP_NAME) -d $(DOCKER_IMAGE):$(DOCKER_TAG)

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
	@echo "  build        - Build the Go binary"
	@echo "  clean        - Clean build artifacts"
	@echo "  run          - Run the application locally"
	@echo "  test         - Run tests"
	@echo "  build-image  - Build Docker image"
	@echo "  push-image   - Push Docker image to registry"
	@echo "  docker-run   - Run the built Docker image"
	@echo "  docker-stop  - Stop and remove the Docker container"
	@echo "  docker-restart - Restart the Docker container"
	@echo "  docker-logs  - View logs for the Docker container"
	@echo "  deps         - Install dependencies"
	@echo "  docs         - Generate documentation"
	@echo "  db-init      - Initialize database directory"
	@echo "  help         - Show this help message"
