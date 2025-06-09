# Use official Go image
FROM golang:1.24 as builder
# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64
# Set working directory
WORKDIR /app
# Copy go.mod and go.sum files first
COPY go.mod ./
# If you have go.sum, uncomment the next line
# COPY go.sum ./
# Download dependencies
RUN go mod download
# Copy the rest of the application code
COPY . .
# Build the application
RUN go build -o homeinsight ./cmd/api
# Use a minimal image to run the compiled binary
FROM alpine:latest
# Set working directory
WORKDIR /root/
# Copy binary from builder
COPY --from=builder /app/homeinsight .
# Copy config file (optional, if your app requires it)
COPY configs/config.yaml ./configs/config.yaml
# Set environment variables if needed
# ENV CONFIG_PATH=./configs/config.yaml
# Expose port 8000
EXPOSE 8000
# Run the binary
CMD ["./homeinsight"]
