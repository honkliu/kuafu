# Kuafu Sprint 0 - Multi-stage Docker build

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod ./

# Copy source
COPY . .

# Build binaries
RUN go build -o /app/kuafu-server ./cmd/kuafu-server
RUN go build -o /app/kuafu ./cmd/kuafu

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/kuafu-server .
COPY --from=builder /app/kuafu .
COPY --from=builder /workspace/web ./web

# Expose API port
EXPOSE 8080

# Default command runs the explicit lab prototype.
CMD ["/app/kuafu-server", "--mode", "lab"]
