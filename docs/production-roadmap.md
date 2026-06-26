# Kuafu Production Roadmap

Status: active production plan  
Date: 2026-06-24  
Owner: Kuafu PM  
Reviewers: Kuafu GPU Architect, Kuafu Researcher, Kuafu Backend Developer, Kuafu Frontend Developer

## PM Correction

The current in-memory E2E application is a lab prototype. It is useful for validating the API, CLI, and dashboard experience, but it is not the target system.

Kuafu is a world-class Kubernetes-based GPU management platform. Production work must now prioritize the real control plane: Kubernetes, GPU device plugins, scheduler integration, persistent state, observability, and multi-tenant operations.

## Production North Star

Kuafu will manage large heterogeneous GPU fleets through a Kubernetes-native control plane while offering a familiar HPC-style user experience.

Target capabilities:

- Manage up to 4000 GPUs across NVIDIA A100/H100 and AMD MI300-class SKUs.
- Submit batch and distributed GPU jobs through CLI, API, and UI.
- Support queues, quotas, priorities, reservations, and usage accounting.
- Preserve topology-aware placement for NVLink and InfiniBand training workloads.
- Provide auditable, recoverable, observable operations for administrators.
- Integrate with proven Kubernetes schedulers and controllers instead of rebuilding a scheduler from scratch.

## Current Lab Prototype

The lab prototype should stay available as `lab mode` for local development and UI/API iteration.

The server binary must not silently run simulated scheduling in production mode. Production mode should fail fast until Kubernetes, GPU device plugin, scheduler adapter, and persistent repository wiring exist.

Keep:

- Domain model concepts for nodes, GPUs, queues, and jobs.
- CLI command structure as a UX baseline.
- Web dashboard as a quick operator/user experience surface.
- In-memory repository and simulated scheduler as test doubles.

Replace for production:

- In-memory state with MongoDB-backed business records.
- Simulated scheduler with Kubernetes scheduler/launcher adapters.
- Seeded A00/A01 inventory with Kubernetes device-plugin discovery.
- Static-only dashboard data with live Kubernetes/MongoDB-backed API responses.

## Architecture Decisions

| Decision | Production Direction | Rationale |
| --- | --- | --- |
| Runtime control plane | Kubernetes API and etcd | Runtime state, watches, RBAC, reconciliation, node/pod lifecycle. |
| Business/query store | MongoDB | Jobs history, users, projects, quotas, billing, audit, dashboard queries. |
| GPU device management | NVIDIA GPU Operator first | Production standard for NVIDIA GPUs, DCGM metrics, device plugin, MIG path. |
| AMD support | AMD GPU/ROCm device plugin later | MI300 requires separate validation and hardware. |
| Scheduler spike | OpenPAI FrameworkController + HiveD first | Best topology-aware virtual-cluster fit and local PAI reference. |
| Scheduler fallback | Volcano, then Kueue where useful | Active ecosystem and batch/gang scheduling fallback. |
| Local prototype | Lab mode only | Useful for fast dev, not production behavior. |

## Milestones

### M1: A00/A01 Kubernetes Foundation

Status: complete with NVIDIA device plugin fallback. GPU Operator remains a follow-up.

Goal: turn A00/A01 into the first real Kuafu Kubernetes testbed.

Acceptance criteria:

- A00 is a Kubernetes control-plane node.
- A01 joins as a worker node.
- `kubectl get nodes` shows both nodes Ready.
- NVIDIA GPU Operator or NVIDIA device plugin exposes 8 GPUs per node.
- A GPU pod requesting `nvidia.com/gpu: 1` runs and sees `nvidia-smi`.
- Basic node/GPU metrics are documented as a gap until DCGM/Prometheus is installed.
- Bootstrap steps are documented and repeatable.

### M2: Scheduler Adapter Spike

Goal: prove real multi-GPU Kubernetes job orchestration.

Acceptance criteria:

- Deploy FrameworkController from the OpenPAI ecosystem or standalone upstream.
- Deploy HiveD if feasible, with a simple virtual-cluster config over A00/A01.
- Submit a test distributed/gang job requesting multiple GPUs.
- Confirm all-or-nothing scheduling behavior.
- Capture logs/status through Kubernetes.
- If HiveD is blocked, run a Volcano job spike and document fallback tradeoffs.
- Architect records the scheduler decision.

### M3: Kuafu API To Kubernetes Bridge

Goal: replace simulated execution with Kubernetes-backed job submission.

Acceptance criteria:

- Kuafu API accepts a job request and creates a Kubernetes runtime object through an adapter.
- Kuafu job status is updated from Kubernetes status, not simulation.
- CLI `jobs submit/list/describe/logs/cancel` works against the Kubernetes-backed path.
- Lab mode still works for local development but is clearly labeled.

### M4: Persistent Business State

Goal: add durable job/user/project/queue state.

Acceptance criteria:

- MongoDB-backed repository for jobs, queues, users/projects, and audit events.
- Sync-controller pattern defined and implemented for job intent/status.
- Restarting Kuafu API does not lose job history.
- Kubernetes runtime state and MongoDB business state have clear conflict resolution rules.

### M5: Production UX And Operations

Goal: make Kuafu usable by admins and users beyond the lab.

Acceptance criteria:

- Dashboard reads real cluster and job data.
- Auth/RBAC design is implemented for at least admin/user roles.
- Queue/quota management is backed by Kubernetes scheduler configuration.
- Observability dashboards exist for API, scheduler, nodes, GPUs, and jobs.
- Runbooks exist for node drain, GPU health, failed jobs, and scheduler outages.

## Role Assignments

| Role | Immediate Ownership |
| --- | --- |
| PM | Keep milestones, acceptance criteria, risks, and team handoffs current. Do not call lab mode production. |
| Researcher | Validate kubeadm/GPU Operator/FrameworkController/HiveD commands and risks against A00/A01 and PAI. |
| GPU Architect | Own scheduler choice, Kubernetes architecture, data ownership, and production review gates. |
| Backend Developer | Build Kubernetes adapter, MongoDB repository, sync controller, API contracts, tests. |
| Frontend Developer | Keep dashboard aligned with real API contracts; mark lab/simulated state clearly until replaced. |

## PM Risk Register

| Risk | Severity | Mitigation |
| --- | --- | --- |
| Mistaking lab prototype for production | High | README and docs now label current code as lab mode; production milestones are explicit. |
| Kubernetes install disrupts A00/A01 | High | Require explicit approval before running bootstrap commands; document rollback/rebuild assumptions. |
| HiveD maintenance burden | Medium | Spike HiveD first, keep Volcano fallback, document decision. |
| 4000-GPU claims unvalidated | High | Treat A00/A01 as functional validation only; add scale simulation/load tests later. |
| MI300 path unvalidated | Medium | Keep AMD as separate milestone requiring hardware or ROCm testbed. |

## Immediate PM Actions

1. Get explicit approval before modifying A00/A01 with kubeadm or GPU Operator installation.
2. Assign Researcher and Architect to refine [a00-a01-kubernetes-bootstrap.md](a00-a01-kubernetes-bootstrap.md).
3. Assign Backend to create Kubernetes adapter interfaces and lab-mode separation.
4. Assign Frontend to label simulated data in UI and prepare real-cluster data states.
5. Run architect review after every milestone before declaring completion.

The execution backlog with owners and acceptance criteria is tracked in [production-backlog.md](production-backlog.md).

## Definition Of Done For Production Phase 1

Kuafu can only be called Kubernetes-based when a real job path goes through Kubernetes on A00/A01:

```text
kuafu CLI/API -> Kuafu backend -> Kubernetes runtime object -> Scheduler -> GPU pod -> status/logs back to Kuafu
```

Until then, the existing application is a validated lab prototype, not the production platform.
