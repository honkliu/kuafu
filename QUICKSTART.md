<!-- markdownlint-disable MD022 MD031 MD032 MD040 -->

# Kuafu Sprint 0 - Quick Reference

## Validation (First Time Setup)

```powershell
# 1. Start Docker Desktop
# 2. Run validation
cd q:\gitroot\kuafu
.\validate-sprint0.ps1
```

Expected output: ✓ All tests passed, binaries built, API validated

## Running the Server

### Docker (Recommended)
```powershell
# Build
docker build -t kuafu:sprint0 .

# Run (with testbed data)
docker run --rm -p 8080:8080 kuafu:sprint0

# Run on different port
docker run --rm -p 9000:8080 -e PORT=8080 kuafu:sprint0 /app/kuafu-server -addr :8080
```

### Binaries (After Docker Build)
```powershell
# Build binaries
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go build -o /workspace/bin/kuafu-server ./cmd/kuafu-server

# Run
./bin/kuafu-server.exe
./bin/kuafu-server.exe -addr :9000  # Different port
```

## API Quick Tests

```powershell
# Health check
curl http://localhost:8080/health

# List nodes
curl http://localhost:8080/api/v1/nodes | ConvertFrom-Json | ConvertTo-Json -Depth 10

# Get A00
curl http://localhost:8080/api/v1/nodes/A00 | ConvertFrom-Json | ConvertTo-Json -Depth 10

# List GPUs
curl http://localhost:8080/api/v1/gpus | ConvertFrom-Json | ConvertTo-Json -Depth 10

# Get specific GPU
curl http://localhost:8080/api/v1/gpus/A00-GPU-0 | ConvertFrom-Json | ConvertTo-Json -Depth 10
```

## CLI Commands

### Online Mode (Server Running)
```powershell
$env:KUAFU_API_URL = "http://localhost:8080"

./bin/kuafu.exe nodes list
./bin/kuafu.exe nodes describe A00
./bin/kuafu.exe nodes describe A01
./bin/kuafu.exe gpus list
```

### Offline Mode (No Server)
```powershell
./bin/kuafu.exe --fixture nodes list
./bin/kuafu.exe --fixture nodes describe A00
./bin/kuafu.exe --fixture gpus list
```

### Custom API URL
```powershell
./bin/kuafu.exe --api-url http://localhost:9000 nodes list
```

## Expected Output Examples

### nodes list
```
NAME  HOSTNAME       STATUS  CPUs  MEMORY(GB)  GPUs  PRIVATE IP
A00   hpcdev000000   Ready   96    1740        8     10.0.0.4
A01   hpcdev000001   Ready   96    1740        8     10.0.0.5
```

### nodes describe A00
```
Name:          A00
Hostname:      hpcdev000000
Status:        Ready
Private IP:    10.0.0.4
InfiniBand:    172.16.3.146
OS:            Ubuntu 22.04.5 LTS
Kernel:        Linux 5.15.0-1088-azure x86_64
CPUs:          96
Memory:        1740 GB
Disk:          497 GB
GPUs:          8
Labels:
  gpu-model: A100-SXM4-80GB
  testbed: initial
Created:       2026-06-24 ...
```

### gpus list
```
ID          NODE  INDEX  MODEL                     MEMORY(MB)  STATUS      ALLOCATED TO
A00-GPU-0   A00   0      NVIDIA A100-SXM4-80GB     81920       Available   -
A00-GPU-1   A00   1      NVIDIA A100-SXM4-80GB     81920       Available   -
...
A01-GPU-7   A01   7      NVIDIA A100-SXM4-80GB     81920       Available   -
```

## Running Tests

```powershell
# All tests
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -v ./...

# Repository tests only
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -v ./internal/repository

# API tests only
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -v ./internal/api

# With coverage
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -cover ./...
```

## Troubleshooting

### "connection refused" error
- Server not running → Start with `docker run -p 8080:8080 kuafu:sprint0`
- Wrong port → Check `$env:KUAFU_API_URL` or use `--api-url`
- Use `--fixture` mode to test CLI offline

### Docker build fails
- Docker Desktop not running → Start Docker Desktop
- Check Docker version: `docker --version`

### Port 8080 already in use
- Use different port: `docker run -p 9000:8080 ...`
- Update CLI: `--api-url http://localhost:9000`

## What's Seeded in Repository

- **2 Nodes**: A00, A01
- **16 GPUs**: 8 per node (A00-GPU-0 through A00-GPU-7, A01-GPU-0 through A01-GPU-7)
- **All GPUs**: NVIDIA A100-SXM4-80GB, 81920 MB memory, Available status
- **All Nodes**: Ubuntu 22.04.5, 96 CPUs, 1740 GB RAM, Ready status

## Files to Read

- `docs/sprint0-dev-guide.md` - Full build/test guide
- `SPRINT0-IMPLEMENTATION.md` - Implementation summary
- `Readme.md` - Project overview and quick start
- `docs/kuafu-design.md` - Product vision and architecture

## Docker Commands Reference

```powershell
# Build image
docker build -t kuafu:sprint0 .

# Run server (foreground)
docker run --rm -p 8080:8080 kuafu:sprint0

# Run server (background)
docker run -d -p 8080:8080 --name kuafu-server kuafu:sprint0

# Stop background server
docker stop kuafu-server

# View server logs
docker logs kuafu-server

# Run CLI inside container
docker run --rm kuafu:sprint0 /app/kuafu --fixture nodes list

# Shell into container (debug)
docker run --rm -it kuafu:sprint0 sh
```

## Next Sprint Preview

Sprint 1 will add:
- Job domain models (Job, JobSpec, JobStatus)
- Job repository and lifecycle operations
- Queue domain model
- CLI: `kuafu jobs submit`, `kuafu jobs list`, `kuafu jobs describe`
- API: `/api/v1/jobs`, `/api/v1/queues`
