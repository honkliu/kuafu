# Kuafu - GPU Management Platform

**Kuafu is chasing the sun! Make it great here.**

Kuafu is a world-class Kubernetes-based GPU management platform for large heterogeneous GPU fleets, providing familiar HPC-style interfaces (similar to LSF/PBS/Slurm) combined with modern Kubernetes-native architecture.

## Current Status: Lab Prototype Complete, Production Track Active

Kuafu's target is a production Kubernetes-native GPU management system. The current codebase is a validated **lab prototype** that proves the product flow while the real Kubernetes foundation is being brought up on A00/A01.

The lab prototype includes:

- ✅ Domain models for Node/GPU inventory
- ✅ In-memory repository with A00/A01 testbed data
- ✅ Simulated, in-memory job scheduler with queueing, GPU allocation, terminal states, cancellation, and logs
- ✅ HTTP API server with inventory, jobs, queues, cluster summary, and dashboard static serving
- ✅ CLI tool for inventory, jobs, and queues
- ✅ Vanilla HTML/CSS/JS dashboard with job submission
- ✅ Unit tests, Docker image validation, and E2E smoke validation on A00
- ✅ Full documentation

The production track starts now:

- Kubernetes bootstrap on A00/A01 with kubeadm
- NVIDIA GPU Operator/device plugin validation
- Real GPU pod scheduling smoke tests
- FrameworkController + HiveD spike, with Volcano/Kueue as fallback options
- MongoDB-backed business state and Kubernetes-backed runtime state

See [docs/production-roadmap.md](docs/production-roadmap.md) and [docs/a00-a01-kubernetes-bootstrap.md](docs/a00-a01-kubernetes-bootstrap.md).
The concrete engineering queue is tracked in [docs/production-backlog.md](docs/production-backlog.md).

## Quick Start

### Prerequisites

- Docker (for building and testing)
- PowerShell (for validation scripts)
- Go 1.22+ (for local development)

### Web Dashboard

The simplest way to explore Kuafu is through the web dashboard:

```powershell
# Start the server (from repo root)
cd q:\gitroot\kuafu
go run cmd/kuafu-server/main.go --addr :8080 --seed

# Open in browser
start http://localhost:8080
```

The dashboard provides:

- **Cluster overview** with real-time metrics (nodes, GPUs, jobs, queues)
- **Node inventory table** with status and resource details
- **GPU inventory table** with allocation status
- **Job queue** with running and pending jobs
- **Queue management** showing priority and capacity
- **Job submission form** for quick testing

All data auto-refreshes every 10 seconds.

### Validation

```powershell
# Start Docker Desktop, then run:
cd q:\gitroot\kuafu
.\validate-mvp.ps1
```

This will build the image, run tests, start the server, validate inventory/job/queue endpoints, exercise the CLI, and test the simulated job lifecycle.

### Manual Testing

```powershell
# Build and run server
docker build -t kuafu:mvp .
docker run --rm -p 8080:8080 kuafu:mvp

# In another terminal, test API
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/nodes

# Or use CLI offline mode
docker run --rm kuafu:mvp /app/kuafu --fixture nodes list
```

## Project Structure

```text
kuafu/
├── cmd/
│   ├── kuafu-server/         # HTTP API server
│   └── kuafu/                # CLI tool
├── internal/
│   ├── domain/               # Node/GPU domain models
│   ├── repository/           # In-memory storage
│   ├── scheduler/            # Simulated job scheduler
│   └── api/                  # HTTP handlers
├── pkg/
│   └── fixtures/             # Testbed seed data
├── web/                      # Static dashboard
├── docs/
│   ├── kuafu-design.md       # Product design document
│   ├── testbed-a00-a01.md    # Testbed configuration
│   └── sprint0-dev-guide.md  # Development guide
├── Dockerfile                # Multi-stage build
├── go.mod                    # Go module definition
└── validate-mvp.ps1          # Automated validation script
```

## API Endpoints

| Endpoint                      | Description              |
|-------------------------------|--------------------------|
| `GET /health`                 | Health check             |
| `GET /ready`                  | Readiness check          |
| `GET /api/v1/cluster/summary` | Cluster statistics       |
| `GET /api/v1/nodes`           | List all nodes           |
| `GET /api/v1/nodes/{name}`    | Get node details         |
| `GET /api/v1/gpus`            | List all GPUs            |
| `GET /api/v1/gpus/{id}`       | Get GPU details          |
| `GET /api/v1/jobs`            | List all jobs            |
| `POST /api/v1/jobs`           | Submit a new job         |
| `GET /api/v1/jobs/{id}`       | Get job details          |
| `DELETE /api/v1/jobs/{id}`    | Cancel a job             |
| `GET /api/v1/queues`          | List all queues          |
| `GET /api/v1/queues/{name}`   | Get queue details        |

## CLI Commands

```bash
# List nodes
kuafu nodes list

# Describe specific node
kuafu nodes describe A00

# List GPUs
kuafu gpus list

# Offline mode (no server required)
kuafu --fixture nodes list
```

## Documentation

- **[Design Document](docs/kuafu-design.md)** - Product vision, architecture, and OpenPAI analysis
- **[Sprint 0 Guide](docs/sprint0-dev-guide.md)** - Build, test, and run instructions
- **[Testbed Config](docs/testbed-a00-a01.md)** - A00/A01 hardware specifications

## Development

See [docs/sprint0-dev-guide.md](docs/sprint0-dev-guide.md) for detailed build and test instructions.

### Running Tests

```bash
# Using Docker
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -v ./...
```

## Vision

The platform will ultimately manage up to 4000 GPUs across different SKUs (NVIDIA A100/H100, AMD MI300), supporting:

- Job submission to queues with LSF/PBS/Slurm-style CLI
- GPU reservations for interactive use
- Usage accounting and chargeback
- Multi-tenant authentication and authorization
- Web UI and REST API
- Kubernetes-native scheduling

The lab prototype is not the product destination. Production Kuafu will replace simulated scheduling with Kubernetes-native job orchestration, GPU device plugins, persistent state, scheduler adapters, quota enforcement, and operational observability.
