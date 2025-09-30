# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o badge-service ./cmd/server

# Stage 2: Create the final image
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates librsvg

# Create a non-root user
RUN adduser -D -g '' appuser

# Create necessary directories
RUN mkdir -p /app/db /app/templates /app/static
RUN chown -R appuser:appuser /app

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/badge-service .

# Copy templates and static files
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Copy only the initial badges JSON (keep DB file out)
COPY --from=builder --chown=appuser:appuser /app/db/initial_badges.json ./db/

# Switch to non-root user
USER appuser

# Expose the port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV LOG_LEVEL=production
ENV DB_PATH=/app/db/badges.db

# Run the application
CMD ["./badge-service"]
