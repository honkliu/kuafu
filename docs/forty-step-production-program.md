# Kuafu 40-Step Production Program

Status: active delivery program  
Owner: Kuafu PM  
Date: 2026-06-25

## Purpose

The user asked for five more rounds of at least eight iterations each: at least 40 concrete steps to enrich functionality, improve architecture, and raise code quality. This document is the execution plan.

The current lab prototype remains useful, but production Kuafu starts here.

## Wave 1: Kubernetes Foundation And Runtime Boundary

- **Step 1: Code quality gates**: add Makefile, lint config, CI workflow, formatting/test commands.
- **Step 2: Runtime config separation**: lab mode explicit; production mode validates Kubernetes/MongoDB/scheduler config and fails fast if missing.
- **Step 3: A00/A01 kubeadm bootstrap**: install Kubernetes control plane and worker after explicit approval.
- **Step 4: NVIDIA GPU Operator**: expose A100 GPUs through Kubernetes and validate GPU pod.
- **Step 5: Kubernetes client abstraction**: add client-go/fake-client boundary for nodes/GPUs.
- **Step 6: GPU inventory from Kubernetes**: replace seeded inventory in production mode with device-plugin data.
- **Step 7: Job CRD scaffold**: define `KuafuJob` or adapter-facing runtime object.
- **Step 8: FrameworkController/HiveD spike**: validate OpenPAI-style gang/topology scheduling on A00/A01.

## Wave 2: Persistence And Sync

- **Step 9: MongoDB repository foundation**: durable jobs, queues, projects, users, audit stubs.
- **Step 10: DB-to-Kubernetes sync controller**: create runtime objects from job intent.
- **Step 11: Kubernetes-to-DB status watcher**: write runtime status/log pointers back to business state.
- **Step 12: Structured errors and retries**: typed errors, retry/backoff, clear API messages.
- **Step 13: Queue and quota model**: hierarchy, max GPUs, max jobs, priority, policy validation.
- **Step 14: User/project multi-tenancy**: project ownership and membership model.
- **Step 15: Auth/RBAC baseline**: OIDC/JWT middleware and admin/user roles.
- **Step 16: OpenAPI contract**: versioned API spec, generated docs, compatibility rules.

## Wave 3: Scheduler, Reservations, Billing, Observability

- **Step 17: Volcano fallback spike**: compare with HiveD/FrameworkController.
- **Step 18: Scheduler decision ADR**: choose production path or dual adapter support.
- **Step 19: Reservation domain and API**: time-window GPU reservations with status and conflicts.
- **Step 20: Interactive access prototype**: SSH/Jupyter/VS Code remote path for reserved GPU sessions.
- **Step 21: Usage accounting**: GPU-hour, queue, user, project, SKU usage records.
- **Step 22: Billing/showback model**: SKU pricing, budgets, reports, exports.
- **Step 23: Prometheus metrics**: API, queue, job, GPU, scheduler, controller metrics.
- **Step 24: Logging and tracing**: structured logs, trace IDs, OpenTelemetry plan.

## Wave 4: Scale, HA, Multi-Vendor, Repair

- **Step 25: API HA and leader election**: multi-replica API and singleton controllers.
- **Step 26: Backup and DR**: MongoDB backups, etcd backup/runbook, RTO/RPO targets.
- **Step 27: 4000-GPU scale simulation**: virtual nodes, fake device inventory, API and scheduler load tests.
- **Step 28: MIG support**: inventory model, scheduler request model, pricing for MIG slices.
- **Step 29: AMD MI300 support plan**: ROCm/device plugin validation path and SKU model.
- **Step 30: Topology discovery**: NVLink/IB topology model and placement constraints.
- **Step 31: GPU health and repair**: DCGM alerts, auto-drain, maintenance state, repair runbooks.
- **Step 32: Spot/preemptible GPU tier**: cheaper preemptible queue with checkpoint hooks.

## Wave 5: Product Excellence And Ecosystem

- **Step 33: Advanced CLI**: watch, arrays, dependencies, templates, shell completion, config profiles.
- **Step 34: React dashboard rewrite**: production UI with real-time data, accessibility, typed API client.
- **Step 35: Plugin architecture**: scheduler/admission/cost/runtime plugins with SDK.
- **Step 36: Workflow DAG jobs**: preprocess/train/eval pipelines with dependencies and retries.
- **Step 37: Autoscaling and smart scheduling**: queue-aware scaling and placement optimization.
- **Step 38: Security hardening**: network policies, secrets, mTLS, image policy, scans.
- **Step 39: Docs and onboarding excellence**: user/admin/developer docs, tutorials, runbooks, searchable site.
- **Step 40: Benchmark and launch readiness**: compare Kuafu vs OpenPAI/Run:ai/Slurm on features, UX, throughput, utilization.

## Execution Rules

- Every step must have executable validation or documented blocker evidence.
- Architect approves all runtime, scheduler, persistence, and security gates.
- PM must not call lab mode production.
- Researcher updates market/OpenPAI assumptions when evidence changes.
- Backend and Frontend implement only against architect-approved contracts.

## Immediate Step

Step 1 is active now: code quality gates. It does not require mutating A00/A01 and can be validated with the A00 Go container path.

## Step 1 Exit Criteria

Step 1 is complete only when all of the following are true:

- `gofmt` check passes for `cmd`, `internal`, and `pkg`.
- `go test ./...` passes.
- Both binaries build: `cmd/kuafu-server` and `cmd/kuafu`.
- `golangci-lint run` passes with `.golangci.yml`.
- GitHub Actions runs formatting, coverage test, build, and lint on pull request.
- README links to the world-class capability model, OpenPAI gap analysis, and this 40-step program.
- Production mode is still fail-fast until Kubernetes, MongoDB, and scheduler adapters are wired.

## Step 2 Entry Criteria

Step 2 may start only after Step 1 gates are green. Step 2 does not make the lab prototype production-ready by itself; it only creates validated configuration and runtime separation for the later Kubernetes and MongoDB work.

Wave 1 Kubernetes work may start only after explicit user approval to mutate A00/A01, because kubeadm, kubelet, CNI, and GPU Operator installation change the testbed machines.

## Step 2 Exit Criteria

Step 2 is complete only when all of the following are true:

- A config package loads defaults, JSON config file, environment variables, and explicit flags in a documented precedence order.
- Lab mode can still start the in-memory API and simulated scheduler from a config file.
- Production mode validates required Kubernetes, MongoDB, scheduler, and auth settings.
- Production mode optionally checks dependency connectivity.
- Valid production config still exits before starting runtime services until Kubernetes, scheduler adapter, and MongoDB repository work is complete.
- Unit tests cover validation, precedence, and invalid environment values.

## Step 3/4 Status

Step 3 is complete: A00 is a kubeadm control plane, A01 joined as worker, and both nodes are `Ready`.

Step 4 is complete with the NVIDIA device plugin fallback: each node advertises `nvidia.com/gpu: 8`, and a Kubernetes GPU smoke pod ran `nvidia-smi -L` successfully. NVIDIA GPU Operator and DCGM/Prometheus remain follow-up work.

The existing Docker containers were stopped before bootstrap and restarted after validation. A01 `open-webui` is healthy and still owns host port `30000`, so the Kuafu NodePort `30000` example must not be applied blindly.

## Step 5 Exit Criteria

Step 5 is complete when all of the following are true:

- Kuafu has a Kubernetes inventory abstraction for nodes and GPUs.
- The abstraction has a fake client for unit tests without a live cluster.
- Inventory sync can write discovered nodes and GPUs through the repository boundary.
- Production client-go adapter requirements are documented.
- Real adapter implementation waits for Step 3/4 cluster and GPU device-plugin validation.

## Step 6/7 Scaffold Exit Criteria

The non-mutating Step 6/7 scaffold is complete when all of the following are true:

- Kuafu jobs can be projected into a scheduler-adapter-facing runtime spec.
- A runtime adapter interface exists for submit, cancel, and status.
- Unsupported production adapters fail clearly instead of falling back to lab scheduling.
- FrameworkController/HiveD/Volcano adapter requirements are documented.
- Real CRD/controller or scheduler adapter implementation waits for Step 3/4 cluster availability.

## Step 9 Foundation Exit Criteria

Step 9 foundation is complete when all of the following are true:

- The API and scheduler depend on repository interfaces, not only `MemoryRepository`.
- A MongoDB-backed repository package implements the same contracts for nodes, GPUs, jobs, queues, and GPU allocation state.
- MongoDB integration tests are available and safely skipped unless `KUAFU_MONGODB_TEST_URI` is set.
- Lab mode remains on `MemoryRepository` until MongoDB deployment is available.

## Step 10/11 Scaffold Exit Criteria

The non-mutating sync scaffold is complete when all of the following are true:

- Queued job intent can be submitted through a runtime adapter and marked with a runtime ID.
- Runtime status can be read through the adapter and written back to the repository.
- Terminal jobs are skipped.
- Tests cover submission, status update, and terminal skip behavior.
- Production startup remains fail-fast until real Kubernetes and MongoDB runtime wiring is complete.

## Step 12 Scaffold Exit Criteria

Step 12 scaffold is complete when all of the following are true:

- Typed error kinds exist for transient, permanent, not-found, and quota conditions.
- A context-aware retry helper exists.
- Sync controller retries transient runtime adapter failures.
- Sync controller does not retry permanent runtime adapter failures.
- Tests cover retry and non-retry behavior.

## Step 13 Scaffold Exit Criteria

Step 13 scaffold is complete when all of the following are true:

- Queue domain can represent hard/soft GPU quota, per-job GPU limit, queue job limits, burst policy, and project ownership.
- Admission policy validates job submissions against queue policy.
- Quota denials use typed quota errors.
- Existing lab queues remain backward compatible with zero-valued optional limits.

## Step 14 Scaffold Exit Criteria

Step 14 scaffold is complete when all of the following are true:

- User and project domain models exist.
- Repository contracts and memory/MongoDB repositories support users and projects.
- Jobs and queues can carry project ownership.
- Queue admission rejects project mismatches.
- User submit policy allows project admins/submitters and rejects viewers/non-members.

## Step 15 Scaffold Exit Criteria

Step 15 scaffold is complete when all of the following are true:

- API has an authenticator interface.
- Project-scoped job submission derives user identity from the auth layer.
- Viewers and non-submitters are rejected at the API boundary.
- Submitters and admins can submit project-scoped jobs.
- Docs state the current header authenticator is a development scaffold, not production OIDC.

## Step 16 Scaffold Exit Criteria

Step 16 scaffold is complete when all of the following are true:

- OpenAPI 3.1 contract exists for current `/api/v1` endpoints.
- Contract documents current schemas for nodes, GPUs, jobs, queues, and cluster summary.
- Contract documents dev-only `X-Kuafu-User` behavior and states production must use OIDC/JWT.
- Contract includes local lab and A00 `30000` server examples.

## Step 17/18 Scaffold Exit Criteria

The non-mutating scheduler fallback scaffold is complete when all of the following are true:

- Volcano runtime manifest translation exists for `RuntimeJobSpec`.
- Translation preserves queue, labels, command, image, and GPU request.
- Translation is unit-tested without live Kubernetes.
- Scheduler strategy ADR records FrameworkController/HiveD primary spike and Volcano fallback.
