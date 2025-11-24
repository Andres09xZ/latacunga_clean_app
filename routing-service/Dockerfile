# ============================================================
# Multi-stage Dockerfile for Routing Service
# ============================================================

# Stage 1: Build
FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git curl

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o routing-service ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates for HTTPS and curl for healthcheck
RUN apk --no-cache add ca-certificates curl

# Copy binary from builder
COPY --from=builder /app/routing-service .

# Run the application
CMD ["./routing-service"]
