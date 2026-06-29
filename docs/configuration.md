# Kuafu Configuration

Status: Step 2 runtime boundary  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Precedence

Kuafu loads configuration in this order:

- Built-in defaults.
- Optional JSON config file passed with `--config`.
- Environment variables.
- Explicit command-line flags.

Flags only override config when they are explicitly present on the command line.

## Lab Mode

Lab mode is the only runnable mode today. It uses the in-memory repository and Docker-backed lab scheduler. When the server runs inside a container, mount the host Docker socket so submitted tasks can execute real containers and capture stdout/stderr logs.

```powershell
go run ./cmd/kuafu-server --config configs/kuafu.lab.json
```

Equivalent flags:

```powershell
go run ./cmd/kuafu-server --mode lab --addr :8080 --seed=true
```

Containerized lab server:

```powershell
docker run --rm -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock kuafu:mvp
```

## A00 External Lab Access

A00 currently exposes port `30000` externally. Use the A00-specific lab config when the lab API/dashboard should be reachable from outside the machine:

```powershell
go run ./cmd/kuafu-server --config configs/kuafu.a00.lab.json
```

External URL:

```text
http://<A00-public-ip>:30000
```

Inter-node communication inside the A00/A01 testbed can use arbitrary ports over the private Ethernet or InfiniBand networks; the `30000` constraint is only for external ingress.

## Production Mode

Production mode validates configuration and optional dependency connectivity, then exits because Kubernetes runtime, MongoDB repository, and scheduler adapters are not wired yet. This is intentional. Production must not silently fall back to lab behavior.

```powershell
go run ./cmd/kuafu-server --config configs/kuafu.production.example.json
```

Required production fields:

- `production.kubernetes.kubeConfigPath` or `production.kubernetes.apiServerURL`
- `production.mongoDB.uri`
- `production.scheduler.backend`: `frameworkcontroller`, `hived`, or `volcano`
- `production.auth.provider`

Optional dependency validation:

```powershell
go run ./cmd/kuafu-server --config configs/kuafu.production.example.json --check-connectivity
```

## Environment Variables

| Environment Variable | Config Field |
| --- | --- |
| `KUAFU_ADDR` | `server.addr` |
| `KUAFU_MODE` | `runtime.mode` |
| `KUAFU_LAB_SEED_DATA` | `lab.seedData` |
| `KUAFU_KUBECONFIG` | `production.kubernetes.kubeConfigPath` |
| `KUAFU_KUBERNETES_API_SERVER` | `production.kubernetes.apiServerURL` |
| `KUAFU_MONGODB_URI` | `production.mongoDB.uri` |
| `KUAFU_SCHEDULER_BACKEND` | `production.scheduler.backend` |
| `KUAFU_AUTH_PROVIDER` | `production.auth.provider` |
| `KUAFU_VALIDATE_CONNECTIVITY` | `production.validation.checkConnectivity` |

## Runtime Boundary

The current production startup contract is:

- Missing production dependencies produce structured validation errors.
- Valid production config exits with an explicit message that runtime wiring is incomplete.
- Lab mode must be selected explicitly and remains the only mode that starts the HTTP API.

This boundary keeps Kuafu honest while Steps 3-10 add Kubernetes, GPU Operator, scheduler adapters, and MongoDB persistence.

Step 3 consumes this boundary by adding real Kubernetes bootstrap validation.
