# Kuafu Sprint 0 - Multi-stage Docker build

# Web build stage
FROM node:22-alpine AS web-builder

WORKDIR /workspace

COPY package.json package-lock.json vite.config.js ./
COPY frontend ./frontend

RUN npm ci && npm run build

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

# Runtime stage. Use glibc so NVIDIA container runtime injected tools such as
# nvidia-smi can execute for live GPU telemetry.
FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates docker.io && rm -rf /var/lib/apt/lists/*

# Copy binaries from builder
COPY --from=builder /app/kuafu-server .
COPY --from=builder /app/kuafu .
COPY --from=web-builder /workspace/web ./web

# Expose API ports used by local lab mode and A00 lab mode.
EXPOSE 8080 30000

# Default command runs the explicit lab prototype.
CMD ["/app/kuafu-server", "--mode", "lab"]
