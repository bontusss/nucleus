# Multi-stage build for Hotel ERP
FROM golang:1.24.3-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w -extldflags '-static'" \
    -o bin/app .

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S hotel-erp && \
    adduser -u 1001 -S hotel-erp -G hotel-erp

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/app .

# Create necessary directories
RUN mkdir -p tmp logs && \
    chown -R hotel-erp:hotel-erp /app

# Switch to non-root user
USER hotel-erp

# Expose port
EXPOSE 9000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9000/health || exit 1

# Use exec form to ensure proper signal handling
ENTRYPOINT ["./app"]
