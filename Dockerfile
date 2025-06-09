# Stage 1: Build the Go binary
FROM golang:1.21-alpine AS builder

# Set environment variables for static builds
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set working directory inside container
WORKDIR /app

# Copy go.mod and go.sum for dependency resolution
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code
COPY . .

# Build the Go app
RUN go build -o homeinsight ./cmd/api

# Stage 2: Run stage with minimal base image
FROM alpine:latest

# Set working directory
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/homeinsight .

# Copy config file if your app expects it at runtime
COPY --from=builder /app/configs/config.yaml ./configs/config.yaml

# Expose the port your app listens on
EXPOSE 8000

# Run the compiled Go binary
CMD ["./homeinsight"]
