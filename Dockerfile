# Build stage
FROM golang:1.21-alpine AS builder

# Install git (needed for go mod)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Set GOPROXY to use Chinese mirror
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.org

# Copy go mod file first to leverage Docker cache
COPY go.mod ./

# Download dependencies and generate go.sum
RUN go mod download -x && \
    go mod tidy && \
    go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o git-watcher ./cmd/server

# Final stage
FROM alpine:3.17

# Use Chinese mirror for Alpine
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

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