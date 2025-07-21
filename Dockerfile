# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

# Set environment variables for Go build
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set working directory
WORKDIR /app

# Copy and download Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Debug: List files to confirm config.yaml is present
RUN ls -la /app/configs/

# Build the Go binary
RUN go build -o homeinsight ./cmd/api

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for TLS support
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /root/

# Copy the Go binary and configuration file from the builder stage
COPY --from=builder /app/homeinsight ./homeinsight
COPY --from=builder /app/configs/config.yaml ./configs/config.yaml

# Ensure the binary is executable
RUN chmod +x /root/homeinsight

# Expose port for the Go application
EXPOSE 8000

# Run the Go application
CMD ["./homeinsight"]
