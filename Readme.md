# Kuafu - GPU Management Platform

**Kuafu is chasing the sun! Make it great here.**

Kuafu is a world-class Kubernetes-based GPU management platform for large heterogeneous GPU fleets, providing familiar HPC-style interfaces (similar to LSF/PBS/Slurm) combined with modern Kubernetes-native architecture.

## Current Status: Lab Prototype Complete, Production Track Active

Kuafu's target is a production Kubernetes-native GPU management system. The current codebase is a validated **lab prototype** that proves the product flow while the real Kubernetes foundation is being brought up on A00/A01.

The lab prototype includes:

- ✅ Domain models for Node/GPU inventory
- ✅ In-memory repository with A00/A01 testbed data
- ✅ Simulated, in-memory job scheduler with queueing, GPU allocation, start/stop/restart/cancel actions, terminal states, and logs
- ✅ HTTP API server with inventory, jobs, queues, reservations, cluster summary, and React console static serving
- ✅ CLI tool for inventory, jobs, and queues
- ✅ React/Vite GPU lease console with OpenPAI/Slurm/lease-platform-inspired IA: Cluster, Nodes, Jobs, Leases, Workspaces, Monitoring, Cost, Catalog, Projects, Audit, and Admin
- ✅ Node-page lease workflow, job detail/logs/metrics drawer, explicit pending-capability surfaces for runtime, cost, catalog, governance, and admin work
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
The world-class feature model, OpenPAI gap analysis, and 40-step delivery program are tracked in:

- [docs/world-class-capability-model.md](docs/world-class-capability-model.md)
- [docs/openpai-gap-analysis.md](docs/openpai-gap-analysis.md)
- [docs/forty-step-production-program.md](docs/forty-step-production-program.md)
- [docs/configuration.md](docs/configuration.md)
- [docs/kubernetes-client-abstraction.md](docs/kubernetes-client-abstraction.md)
- [docs/scheduler-adapter-contract.md](docs/scheduler-adapter-contract.md)
- [docs/repository-boundary.md](docs/repository-boundary.md)
- [docs/sync-controller.md](docs/sync-controller.md)
- [docs/errors-and-retries.md](docs/errors-and-retries.md)
- [docs/queue-quota-policy.md](docs/queue-quota-policy.md)
- [docs/multi-tenancy.md](docs/multi-tenancy.md)
- [docs/auth-rbac.md](docs/auth-rbac.md)
- [docs/api-contract.md](docs/api-contract.md)
- [docs/volcano-adapter.md](docs/volcano-adapter.md)
- [docs/adr/0001-scheduler-strategy.md](docs/adr/0001-scheduler-strategy.md)

## Quick Start

### Prerequisites

- Docker (for building and testing)
- PowerShell (for validation scripts)
- Go 1.22+ (for local development)

### Web Console

The simplest way to explore Kuafu is through the React web console:

```powershell
# Start the lab-mode server (from repo root)
cd q:\gitroot\kuafu
go run cmd/kuafu-server/main.go --config configs/kuafu.lab.json

# Open in browser
start http://localhost:8080
```

On A00, use the externally reachable port `30000`:

```powershell
go run cmd/kuafu-server/main.go --config configs/kuafu.a00.lab.json

# External URL
# http://<A00-public-ip>:30000
```

Project-scoped lab job submissions use the development `X-Kuafu-User` header. Seeded lab users are `lab-admin`, `lab-user`, and `lab-viewer`.

The console provides:

- **Cluster overview** with real-time metrics (nodes, GPUs, jobs, queues)
- **Node inventory table** with status, free/allocated GPU counts, reservation state, and reserve actions
- **Node reservation drawer** launched from the Nodes page, with multi-node selection, duration/owner fields, and reserved-node command rendering
- **Lease platform views** for Workspaces, Monitoring, Cost, Catalog, Projects, Audit, and Admin, with runtime-dependent items marked as pending instead of faked
- **GPU inventory table** with allocation status
- **Job management** with submission, filters, start/stop/restart/cancel actions, and running/completed job details
- **Job detail drawer** showing submitted spec, exact runtime command, logs, allocated GPUs/nodes, and GPU usage snapshot
- **Queue management** with create/edit/delete, Active/Paused scheduling state, hard/soft quotas, per-job limits, queue depth limits, priority, and burst policy
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
├── frontend/                 # React/Vite console source
├── web/                      # Built static console served by the Go API server
├── docs/
│   ├── kuafu-design.md       # Product design document
│   ├── testbed-a00-a01.md    # Testbed configuration
│   └── sprint0-dev-guide.md  # Development guide
├── Dockerfile                # Multi-stage build
├── go.mod                    # Go module definition
└── validate-mvp.ps1          # Automated validation script
```

## API Endpoints

| Endpoint                                         | Description                         |
|--------------------------------------------------|-------------------------------------|
| `GET /health`                                    | Health check                        |
| `GET /ready`                                     | Readiness check                     |
| `GET /api/v1/cluster/summary`                    | Cluster statistics                  |
| `GET /api/v1/nodes`                              | List all nodes                      |
| `GET /api/v1/nodes/{name}`                       | Get node details                    |
| `GET /api/v1/gpus`                               | List all GPUs                       |
| `GET /api/v1/gpus/{id}`                          | Get GPU details                     |
| `GET /api/v1/jobs`                               | List all jobs                       |
| `POST /api/v1/jobs`                              | Submit a new job                    |
| `GET /api/v1/jobs/{id}`                          | Get job details                     |
| `POST /api/v1/jobs/{id}/start`                   | Start/requeue a stopped job         |
| `POST /api/v1/jobs/{id}/stop`                    | Stop a queued/running job           |
| `POST /api/v1/jobs/{id}/restart`                 | Restart a job                       |
| `DELETE /api/v1/jobs/{id}`                       | Cancel a job                        |
| `GET /api/v1/jobs/{id}/logs`                     | Get job logs                        |
| `GET /api/v1/jobs/{id}/metrics`                  | Get job GPU metrics snapshot        |
| `GET /api/v1/queues`                             | List all queues                     |
| `POST /api/v1/queues`                            | Create a queue                      |
| `GET /api/v1/queues/{name}`                      | Get queue details                   |
| `PUT /api/v1/queues/{name}`                      | Update queue policy/state           |
| `DELETE /api/v1/queues/{name}`                   | Delete an inactive queue            |
| `GET /api/v1/reservations`                       | List node reservations              |
| `POST /api/v1/reservations`                      | Reserve dedicated nodes             |
| `GET /api/v1/reservations/{id}`                  | Get reservation details             |
| `DELETE /api/v1/reservations/{id}`               | Release reservation                 |
| `POST /api/v1/reservations/{id}/commands`        | Record/render reserved-node command |

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

# Manage jobs
kuafu jobs start <id>
kuafu jobs stop <id>
kuafu jobs restart <id>
kuafu jobs cancel <id>

# Manage queues
kuafu queues create --name research --max-gpus 16 --soft-gpus 12 --priority 200 --allow-burst
kuafu queues update research --max-gpus 24 --max-gpus-per-job 8
kuafu queues pause research
kuafu queues resume research
kuafu queues delete research
```

## Documentation

- **[Design Document](docs/kuafu-design.md)** - Product vision, architecture, and OpenPAI analysis
- **[Sprint 0 Guide](docs/sprint0-dev-guide.md)** - Build, test, and run instructions
- **[Testbed Config](docs/testbed-a00-a01.md)** - A00/A01 hardware specifications

## Development

See [docs/sprint0-dev-guide.md](docs/sprint0-dev-guide.md) for detailed build and test instructions.

### Frontend Development

```powershell
# Build React console into web/ for the Go server
npm ci
npm run build

# Run Vite dev server for frontend-only iteration
npm run dev
```

The Dockerfile also runs `npm ci && npm run build`, so the production image always contains a fresh React build.

### Runtime Modes

- `production` is the default binary mode and intentionally fails until Kubernetes, GPU device plugins, scheduler adapter, and persistent repository are configured.
- `lab` runs the current in-memory repository and simulated scheduler for local development and UI/API iteration.

```powershell
# Fails fast until production Kubernetes integration exists
go run cmd/kuafu-server/main.go

# Validates production config, then exits because runtime wiring is not complete yet
go run cmd/kuafu-server/main.go --config configs/kuafu.production.example.json

# Starts the lab prototype explicitly
go run cmd/kuafu-server/main.go --config configs/kuafu.lab.json
```

### Running Tests

```bash
# Using Docker
docker run --rm -v ${PWD}:/workspace kuafu:sprint0 go test -v ./...

# With local Go
make validate
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
