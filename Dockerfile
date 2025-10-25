# Multi-stage Dockerfile for Stream Application
# Build stage uses full Go environment, final stage uses minimal scratch image

# ========================================================================
# Build Stage
# ========================================================================
FROM golang:1.23-alpine AS builder

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

# Install build dependencies (including gcc and musl-dev for CGO/sqlite3)
RUN apk add --no-cache git make ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with CGO enabled for sqlite3
# Note: GOARCH is omitted to use native architecture (prevents cross-compilation issues with CGO)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -v \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o /app/stream \
    .

# ========================================================================
# Final Stage
# ========================================================================
FROM alpine:latest

# Install runtime dependencies (including sqlite libs for CGO)
RUN apk --no-cache add ca-certificates tzdata sqlite-libs

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/stream /app/stream

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port (adjust based on your application)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/stream", "health"]

# Run the application
ENTRYPOINT ["/app/stream"]
CMD ["serve"]
