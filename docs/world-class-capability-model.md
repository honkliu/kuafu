# World-Class GPU Management System Capability Model

Status: active product definition  
Owner: Kuafu PM  
Date: 2026-06-25

## Definition

A world-class GPU management system is not only a scheduler. It is a full product and control plane for heterogeneous accelerated compute: users can discover capacity, submit and debug work, reserve resources, understand cost, collaborate safely, and trust the platform to operate at large scale.

Kuafu's target is to exceed OpenPAI-style Kubernetes AI platforms by combining:

- HPC-grade CLI ergonomics from Slurm, PBS, and LSF.
- Kubernetes-native runtime control and reconciliation.
- Topology-aware GPU scheduling for NVLink, NVSwitch, and InfiniBand.
- Run:ai-class user experience for GPU sharing, reservations, and quota visibility.
- Open, extensible architecture suitable for on-prem and hybrid GPU fleets.

## Personas

| Persona | Primary Jobs To Be Done |
| --- | --- |
| ML engineer | Submit jobs, monitor progress, debug failures, compare runs, manage artifacts. |
| Research scientist | Run interactive notebooks, reserve GPUs, launch distributed experiments. |
| Platform admin | Manage nodes, GPUs, queues, quotas, health, upgrades, incidents, policies. |
| Team/project owner | Allocate quota, review usage, approve reservations, control spend. |
| Finance/budget owner | Track chargeback/showback, forecast cost, enforce budgets. |
| Security/compliance owner | Audit access, isolate tenants, manage secrets, enforce policy. |
| SRE/on-call | Monitor SLOs, diagnose incidents, repair nodes/GPUs, run playbooks. |

## Capability Domains

### 1. CLI

World-class CLI capabilities:

- `kuafu job submit/list/show/logs/cancel/hold/release/requeue/clone/array`.
- `kuafu queue list/show/fair-share/backfill`.
- `kuafu node list/show/drain/cordon/topology`.
- `kuafu gpu list/show/health/topology/mig`.
- `kuafu reserve create/list/show/extend/cancel/login`.
- `kuafu project list/show/quota/usage/cost`.
- `kuafu admin health/alerts/audit/runbook`.
- JSON, YAML, table, and custom column output.
- Profiles, `.kuafurc`, environment variables, and shell completion.
- Batch operations, job arrays, dependencies, and live watch mode.

OpenPAI gap: `paictl` is mostly admin-focused and does not provide rich HPC-style user workflows.

### 2. Web UI

World-class UI surfaces:

- Cluster overview with live capacity, health, utilization, and cost.
- Job submission wizard with templates, validation, estimates, and dry run.
- Job details with lifecycle timeline, logs, metrics, artifacts, retry advice, and clone.
- Queue dashboard with wait estimates, fair-share, backfill, and quota pressure.
- GPU inventory heatmaps by node, SKU, health, topology, allocation, MIG slice.
- Reservation calendar for interactive GPU access.
- Project/user portal for quota, budget, usage, and approvals.
- Admin operations for drain, cordon, maintenance, repair, and policy changes.
- Accessibility, responsive layout, and role-specific dashboards.

OpenPAI gap: good portal foundation, but limited reservations, billing, user cost visibility, topology UX, and modern self-service operations.

### 3. API And SDK

World-class API/SDK capabilities:

- Versioned OpenAPI 3.1 REST API.
- Streaming logs and metrics through WebSocket or SSE.
- Webhooks for job lifecycle, quota, reservations, and alerts.
- Cursor pagination, filtering, sorting, and bulk operations.
- Python, Go, JavaScript/TypeScript SDKs.
- API keys, OAuth/OIDC, service accounts, mTLS option.
- Stable deprecation policy and compatibility tests.

OpenPAI gap: strong REST reference, but Kuafu needs richer streaming, bulk, SDK, and webhook surfaces.

### 4. Scheduling

World-class scheduler capabilities:

- Gang scheduling for distributed jobs.
- Topology-aware placement across NVLink/NVSwitch/InfiniBand domains.
- Heterogeneous SKU scheduling across A100/H100/MI300 and future accelerators.
- Queue priority, preemption, backfill, fair-share, and wait-time prediction.
- Reservation-aware scheduling and idle-reservation backfill.
- MIG, MPS, time-slicing, and fractional GPU support where hardware permits.
- Elastic jobs, job arrays, dependencies, and workflow DAGs.
- Policy plugin framework for custom scoring and admission.

OpenPAI strength: HiveD virtual clusters reserve topology, which is a key capability to preserve or surpass.  
OpenPAI gap: no full reservation/billing/fractional GPU product layer.

### 5. Queues, Quotas, And Fairness

World-class requirements:

- Hierarchical queues: organization, team, project, user.
- Guaranteed quota plus burst quota.
- Hard and soft limits for GPU count, GPU-hours, runtime, storage, and budget.
- Quota borrowing and temporary grants.
- Queue ACLs and policy templates.
- Fair-share accounting with decay and priority interactions.
- Queue analytics and wait/backfill prediction.

OpenPAI gap: virtual clusters are strong but not enough for modern quota, burst, and billing workflows.

### 6. Reservations And Interactive Access

World-class requirements:

- Calendar-based reservations for GPU, node, rack, or topology slice.
- Reservation cost estimation and approval flows.
- SSH, JupyterLab, VS Code remote, and web terminal access modes.
- Expiration, extension, cancellation, and idle backfill.
- Recurring reservations and shared team reservations.
- Reservation audit, quota impact, and budget tracking.

OpenPAI gap: no first-class reservation product.

### 7. Billing, Chargeback, And Cost Optimization

World-class requirements:

- GPU-hour, CPU-hour, memory, storage, and network usage metering.
- SKU-based pricing, peak/off-peak rates, and spot/preemptible discounts.
- Project budgets, credits, warnings, and auto-suspend policy.
- Chargeback and showback reports by user/project/org.
- Cost advisor: recommend off-peak, smaller SKU, spot queue, or right-size request.
- Immutable billing ledger and export APIs.

OpenPAI gap: no mature billing or chargeback layer.

### 8. Multi-Tenancy, Security, And Compliance

World-class requirements:

- OIDC/SAML/LDAP/AD authentication.
- Fine-grained RBAC and optional ABAC.
- Project, namespace, network, storage, and secret isolation.
- Pod security, image policy, admission controls, and signed images.
- Secret management through Kubernetes Secrets or Vault.
- Immutable audit logs and compliance exports.
- Data residency, retention, and privacy controls.

OpenPAI gap: has basic auth/OIDC patterns, but Kuafu needs stronger enterprise-grade policy, audit, and isolation.

### 9. Observability And Incident Response

World-class requirements:

- Prometheus metrics for API, scheduler, controller, node, GPU, queue, job.
- DCGM/NVIDIA metrics and AMD ROCm metrics.
- Logs via Fluent Bit/Fluentd to Loki or Elasticsearch.
- Distributed tracing through OpenTelemetry.
- Alerts with runbook links for GPU failures, queue stalls, API latency, scheduler failures.
- SLO dashboards, error budgets, and capacity forecasting.
- Auto-drain and repair workflows for unhealthy GPUs/nodes.

OpenPAI strength: solid Prometheus/Grafana/Fluentd reference.  
OpenPAI gap: limited user-facing observability, cost visibility, and automated repair.

### 10. Data, Model, And Environment Management

World-class requirements:

- Dataset catalog, metadata, tags, lineage, and ACLs.
- Dataset versioning and staging/cache to local NVMe or fast storage.
- Model registry integration with versions, metrics, owners, and promotion workflow.
- Environment registry: container images, Conda/pip snapshots, SBOM, reproducibility.
- Job templates and marketplace for reusable workflows.
- Storage-aware scheduling hints.

OpenPAI strength: marketplace and storage integration.  
OpenPAI gap: not a full modern data/model/environment governance product.

### 11. Reliability, HA, DR, And Scale

World-class requirements:

- HA API servers and controller leader election.
- HA MongoDB and Kubernetes etcd backup/restore.
- Zero-downtime upgrades and rollback.
- Load tests for 4000 GPU equivalent inventory and high job submission throughput.
- Chaos testing for node, GPU, API, scheduler, and database failure.
- DR runbooks with RTO/RPO targets.

OpenPAI gap: deployment runbooks exist, but Kuafu must prove scale, HA, and DR with explicit tests.

### 12. Extensibility And Ecosystem

World-class requirements:

- Scheduler policy plugins.
- Admission plugins.
- Cost plugins.
- Runtime templates for PyTorch, TensorFlow, Ray, MPI, Jupyter, TensorBoard.
- GitOps deployment and configuration.
- SDKs and plugin development guide.
- Compatibility matrix across Kubernetes, GPU driver, CUDA, ROCm, and scheduler versions.

OpenPAI gap: modular architecture exists, but active extensibility and plugin governance are limited.

## Competitive Target

Kuafu should beat OpenPAI by adding:

1. Rich HPC-style CLI.
2. Reservation system.
3. Billing and chargeback.
4. NVIDIA and AMD first-class support.
5. Explicit lab/production runtime separation.
6. Stronger RBAC/audit/compliance.
7. Modern UX and user cost visibility.
8. 4000-GPU validation plan.
9. Plugin architecture.
10. Active scheduler fallback strategy: HiveD first, Volcano/Kueue fallback.

Kuafu should compete with Run:ai by matching GPU sharing/quota UX while staying open and on-prem friendly. It should compete with Slurm/LSF/PBS by offering familiar CLI patterns while using Kubernetes-native runtime primitives.
