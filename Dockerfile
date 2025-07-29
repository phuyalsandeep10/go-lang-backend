# Stage 1: builder
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o homeinsight ./cmd/api

# Stage 2: final image
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the Go binary and config file from builder stage
COPY --from=builder /app/homeinsight ./homeinsight
COPY --from=builder /app/configs/config.yaml ./configs/config.yaml

# (Optional) Copy .env file if you need it inside the container
# You can uncomment this line if your Go app reads .env directly
# COPY --from=builder /app/.env .env
COPY --from=builder /app/.env .env


RUN chmod +x ./homeinsight

# Expose the port your Go app listens on
EXPOSE 8000

# Set environment variables using --env or --env-file during docker run
# Example:
# docker run -p 8000:8000 -e MONGO_URI=... your-image-name
# or
# docker run --env-file .env -p 8000:8000 your-image-name

CMD ["./homeinsight"]
