# Build stage
FROM golang:1.20-alpine AS builder

# Install git (needed for go mod)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o git-watcher ./cmd/server

# Final stage
FROM alpine:3.17

# Install git and ca-certificates (required for git and HTTPS)
RUN apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Create a non-root user
RUN adduser -D -h /app appuser

# Create necessary directories with correct permissions
RUN mkdir -p /app/configs /app/repos && \
    chown -R appuser:appuser /app

# Set working directory
WORKDIR /app

# Copy the compiled binary
COPY --from=builder /app/git-watcher .

# Copy configuration file
COPY --from=builder /app/configs/config.json ./configs/

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the HTTP port (must match server.port in config.json)
EXPOSE 8080

# Run the application
ENTRYPOINT ["./git-watcher"]
CMD ["-config=./configs/config.json"] 