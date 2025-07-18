# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Debug: List files to confirm config.yaml is present
RUN ls -la /app/configs/

RUN go build -o homeinsight ./cmd/api

# Stage 2: Run stage with minimal base image
FROM alpine:latest

# Install ca-certificates for TLS support
RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /app/homeinsight .
COPY --from=builder /app/configs/config.yaml ./configs/config.yaml

EXPOSE 8000

CMD ["./homeinsight"]
