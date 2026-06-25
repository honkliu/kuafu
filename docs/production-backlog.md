# Kuafu Production Backlog

Status: active  
Owner: Kuafu PM  
Date: 2026-06-24

## P0: Production Foundation

### P0-1: Approve A00/A01 Kubernetes Bootstrap

Owner: PM  
Depends on: user approval to modify remote machines  
Outcome: explicit approval or rejection recorded before any kubeadm/GPU Operator commands run.

Acceptance criteria:

- Approval covers package installs, service changes, kubeadm init/join, and GPU Operator install.
- Any existing workloads on A00/A01 are identified or confirmed irrelevant.
- Rollback/rebuild expectations are documented.

### P0-2: Bootstrap Kubernetes On A00/A01

Owner: Backend Developer with GPU Architect review  
Depends on: P0-1  
Outcome: two-node Kubernetes testbed.

Acceptance criteria:

- A00 is Kubernetes control plane.
- A01 joins as worker.
- `kubectl get nodes -o wide` shows both nodes Ready.
- Bootstrap commands and outputs are captured in [a00-a01-kubernetes-bootstrap.md](a00-a01-kubernetes-bootstrap.md).

### P0-3: Expose NVIDIA GPUs To Kubernetes

Owner: GPU Architect  
Depends on: P0-2  
Outcome: Kubernetes sees A00/A01 GPUs.

Acceptance criteria:

- NVIDIA GPU Operator or fallback device plugin is installed.
- Each node advertises 8 GPUs through `nvidia.com/gpu`.
- A CUDA pod requesting one GPU runs and prints `nvidia-smi -L`.
- GPU Operator vs standalone device plugin decision is recorded.

### P0-4: Split Lab Mode From Production Mode

Owner: Backend Developer  
Depends on: current lab prototype  
Outcome: lab-mode code cannot be mistaken for production runtime.

Acceptance criteria:

- CLI/server expose explicit `--lab-mode` or equivalent config.
- Dashboard clearly shows lab mode when simulated scheduler is active.
- Production mode fails fast if Kubernetes/MongoDB config is missing.
- Tests cover lab mode and production-mode config validation separately.

## P1: Scheduler And Runtime

### P1-1: FrameworkController Spike

Owner: Backend Developer  
Depends on: P0-3  
Outcome: real Kubernetes job orchestration spike.

Acceptance criteria:

- FrameworkController is deployed on A00/A01 cluster.
- A Framework CRD creates pods and reaches terminal status.
- Logs are retrievable from Kubernetes.
- Integration complexity and fork/dependency recommendation are documented.

### P1-2: HiveD Spike

Owner: GPU Architect  
Depends on: P1-1  
Outcome: topology-aware virtual cluster scheduling decision.

Acceptance criteria:

- HiveD is deployed or documented as blocked with evidence.
- At least one virtual cluster maps to A00/A01 GPU capacity.
- A multi-GPU job demonstrates gang-style allocation.
- Topology and queue semantics are evaluated against Kuafu needs.

### P1-3: Volcano Fallback Spike

Owner: Researcher and Backend Developer  
Depends on: P0-3  
Outcome: active-upstream fallback comparison.

Acceptance criteria:

- Volcano is deployed on A00/A01 or blocked with evidence.
- A VolcanoJob requests GPUs and runs.
- Queue/gang/preemption behavior is compared with HiveD.
- Architect records scheduler decision.

## P2: Kuafu Production Services

### P2-1: Kubernetes Scheduler Adapter Interface

Owner: Backend Developer  
Depends on: P1 scheduler decision  
Outcome: Kuafu stops calling simulated scheduler in production mode.

Acceptance criteria:

- Adapter interface supports submit, cancel, describe, logs, resource snapshot, and queue status.
- Lab adapter remains for tests only.
- Kubernetes adapter creates the chosen runtime object.
- API and CLI use the adapter boundary.

### P2-2: MongoDB Repository

Owner: Backend Developer  
Depends on: adapter interface  
Outcome: durable job and queue business state.

Acceptance criteria:

- MongoDB stores job intent, queue config, user/project stubs, and audit events.
- Restarting Kuafu does not lose job history.
- Repository interfaces have memory and Mongo implementations.

### P2-3: Sync Controller

Owner: Backend Developer with GPU Architect review  
Depends on: P2-1, P2-2  
Outcome: MongoDB intent reconciles with Kubernetes runtime state.

Acceptance criteria:

- New job intent creates Kubernetes runtime object.
- Kubernetes status updates MongoDB status.
- Completed jobs keep durable history.
- Reconciliation handles restart and duplicate events safely.

## P3: Product Surface

### P3-1: Production Dashboard Data

Owner: Frontend Developer  
Depends on: P2 services  
Outcome: dashboard reflects real Kubernetes/MongoDB-backed state.

Acceptance criteria:

- Lab-mode banner disappears only in production mode.
- Dashboard shows real nodes, GPUs, queues, jobs, logs, and errors.
- Empty/loading/error states are production-ready.

### P3-2: Auth/RBAC Design And Stub

Owner: Backend Developer and PM  
Depends on: P2 services  
Outcome: first multi-tenant security boundary.

Acceptance criteria:

- Admin/user roles are defined.
- API enforces at least basic role checks.
- Future OIDC integration path is documented.

## Review Gates

- GPU Architect must approve P0-2, P0-3, P1 scheduler decision, and P2 adapter/sync design.
- PM must not mark any milestone complete without executable validation evidence.
- Researcher must update risks when OpenPAI/HiveD or Volcano assumptions change.
