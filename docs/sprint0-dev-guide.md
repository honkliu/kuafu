# Kuafu Sprint 0 Development Guide

**Status**: Sprint 0 - Resource Inventory Foundation  
**Date**: 2026-06-24

## Overview

Sprint 0 delivers the foundational resource inventory system for Kuafu:

- Domain models for Node and GPU inventory
- In-memory repository with A00/A01 testbed data
- HTTP API server with health and resource endpoints
- CLI tool with nodes and gpus commands
- Unit tests for repository and API handlers

## Constraints

- **No external dependencies**: MongoDB, Kubernetes, auth, schedulers excluded
- **Standard library preferred**: Minimal external dependencies
- **Docker validation**: Local Go may be missing, use Docker for builds

## Project Structure

```text
kuafu/
├── cmd/
│   ├── kuafu-server/    # HTTP API server
│   └── kuafu/           # CLI tool
├── internal/
│   ├── domain/          # Node and GPU domain models
│   ├── repository/      # In-memory storage
│   └── api/             # HTTP server and handlers
├── pkg/
│   └── fixtures/        # Testbed seed data (A00/A01)
├── docs/
│   └── sprint0-dev-guide.md
├── go.mod
└── Dockerfile
```

## Building

### With Local Go

```powershell
# Build server
go build -o bin/kuafu-server.exe ./cmd/kuafu-server

# Build CLI
go build -o bin/kuafu.exe ./cmd/kuafu
```

### With Docker

```powershell
# Build image
docker build -t kuafu:sprint0 .

# Build binaries using Docker
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go build -o /workspace/bin/kuafu-server ./cmd/kuafu-server
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go build -o /workspace/bin/kuafu ./cmd/kuafu
```

## Testing

### Run Unit Tests

```powershell
# Local Go
go test ./...

# Docker
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test ./...
```

### Test Coverage

```powershell
go test -cover ./...
```

## Running the Server

### Start Server (seeded with A00/A01 data)

```powershell
# Local
./bin/kuafu-server.exe --mode lab

# Docker
docker run --rm -p 8080:8080 kuafu:sprint0 /app/kuafu-server --mode lab
```

Server starts on `http://localhost:8080` by default.

### Server Options

```powershell
./bin/kuafu-server.exe --mode lab -addr :9000      # Custom port
./bin/kuafu-server.exe --mode lab -seed=false      # Don't seed testbed data
```

## API Endpoints

| Endpoint | Method | Description |
| --- | --- | --- |
| `/health` | GET | Health check |
| `/ready` | GET | Readiness check |
| `/api/v1/nodes` | GET | List all nodes |
| `/api/v1/nodes/{name}` | GET | Get node details |
| `/api/v1/gpus` | GET | List all GPUs |
| `/api/v1/gpus?node={name}` | GET | List GPUs for a node |
| `/api/v1/gpus/{id}` | GET | Get GPU details |

### Example API Calls

```powershell
# Health check
curl http://localhost:8080/health

# List nodes
curl http://localhost:8080/api/v1/nodes

# Get node A00
curl http://localhost:8080/api/v1/nodes/A00

# List all GPUs
curl http://localhost:8080/api/v1/gpus

# List GPUs on A00
curl http://localhost:8080/api/v1/gpus?node=A00

# Get specific GPU
curl http://localhost:8080/api/v1/gpus/A00-GPU-0
```

## CLI Usage

### Online Mode (requires running server)

```powershell
# Set API URL (optional, defaults to http://localhost:8080)
$env:KUAFU_API_URL = "http://localhost:8080"

# List nodes
./bin/kuafu.exe nodes list

# Describe specific node
./bin/kuafu.exe nodes describe A00

# List GPUs
./bin/kuafu.exe gpus list

# Use custom API URL
./bin/kuafu.exe --api-url http://localhost:9000 nodes list
```

### Offline Fixture Mode (no server required)

```powershell
# List nodes from fixture data
./bin/kuafu.exe --fixture nodes list

# Describe node from fixture
./bin/kuafu.exe --fixture nodes describe A00

# List GPUs from fixture
./bin/kuafu.exe --fixture gpus list
```

## Validation Workflow

```powershell
# 1. Run tests
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -v ./...

# 2. Build binaries
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go build -o /workspace/bin/kuafu-server ./cmd/kuafu-server
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go build -o /workspace/bin/kuafu ./cmd/kuafu

# 3. Start server
docker run --rm -d -p 8080:8080 --name kuafu-server kuafu:sprint0 /app/kuafu-server --mode lab

# 4. Test API
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/nodes

# 5. Test CLI (from host)
./bin/kuafu.exe nodes list
./bin/kuafu.exe nodes describe A00
./bin/kuafu.exe gpus list

# 6. Test offline mode
./bin/kuafu.exe --fixture nodes list

# 7. Stop server
docker stop kuafu-server
```

## Testbed Data

The repository is seeded with two nodes from the testbed configuration:

### Node A00

- Hostname: hpcdev000000
- Private IP: 10.0.0.4
- 96 CPUs, 1740 GB memory
- 8x NVIDIA A100-SXM4-80GB GPUs

### Node A01

- Hostname: hpcdev000001
- Private IP: 10.0.0.5
- 96 CPUs, 1740 GB memory
- 8x NVIDIA A100-SXM4-80GB GPUs

Total: 2 nodes, 16 GPUs, 192 CPU cores

## Code Quality

All code is:

- `gofmt` formatted
- Using standard library where possible
- Minimal external dependencies
- Idiomatic Go patterns
- Thread-safe (repository uses sync.RWMutex)
- Unit tested

## Next Steps

Sprint 1 will add:

- Job domain models and lifecycle
- Queue management
- Basic scheduler integration
- Extended CLI commands for job submission

## Troubleshooting

### Server won't start

- Check if port 8080 is already in use
- Use `-addr :9000` to try a different port

### CLI "connection refused"

- Ensure server is running
- Check `KUAFU_API_URL` environment variable
- Use `--fixture` mode to test offline

### Tests fail

- Ensure Go 1.22+ is installed (or use Docker)
- Run `go mod tidy` to verify dependencies
