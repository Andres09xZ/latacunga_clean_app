# ============================================================
# Multi-stage Dockerfile for Operations Service
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
RUN CGO_ENABLED=0 GOOS=linux go build -o operations-service ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /root/

# Install ca-certificates for HTTPS and curl for healthcheck
RUN apk --no-cache add ca-certificates curl

# Copy binary from builder
COPY --from=builder /app/operations-service .

# Expose port
EXPOSE 8085

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8085/health || exit 1

# Run the application
CMD ["./operations-service"]
