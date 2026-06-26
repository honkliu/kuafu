# OpenPAI Gap Analysis For Kuafu

Status: active research note  
Owner: Kuafu Researcher  
Date: 2026-06-25

## OpenPAI Summary

OpenPAI is a mature Kubernetes-based AI platform. Its strongest reusable ideas are FrameworkController, HiveD, the OpenPAI job protocol, the database-controller pattern, and a complete portal/API/monitoring stack.

The local reference repo at `q:\gitroot\pai` is stable/read-only after v1.8.1. Kuafu should learn from it and selectively reuse components or patterns, but should not simply clone OpenPAI as the final product.

## Strengths To Preserve

| OpenPAI Strength | Why It Matters For Kuafu |
| --- | --- |
| Kubernetes-native post-v1 architecture | Aligns with Kuafu's production target. |
| FrameworkController | Proven generic workload orchestration and gang-style execution. |
| HiveD scheduler | Topology-aware virtual clusters are rare and valuable for distributed GPU jobs. |
| OpenPAI protocol | Portable job specification model. |
| Database-controller pattern | Useful MongoDB/PostgreSQL-to-Kubernetes reconciliation model. |
| Marketplace | Good idea for reusable job/data templates. |
| Prometheus/Grafana/Fluentd stack | Good observability starting point. |
| Auth and user/group management | Baseline enterprise concepts. |
| Admin and user manuals | Good documentation discipline to emulate. |

## Gaps Kuafu Must Close

| Gap | OpenPAI Limitation | Kuafu Direction |
| --- | --- | --- |
| HPC-style CLI | `paictl` is admin-focused. | Rich `kuafu job/queue/node/gpu/reserve/project/admin` CLI. |
| Reservations | No first-class GPU reservation product. | Calendar/time-window GPU reservation and interactive access. |
| Billing/chargeback | No mature cost model. | GPU-hour ledger, SKU pricing, budgets, showback/chargeback. |
| Quota model | Mostly VC quota. | Hierarchical quota, burst quota, borrowing, fair-share. |
| Fractional GPU | Not a product-level feature. | MIG/MPS/time-slicing awareness where safe. |
| User observability | Strong admin monitoring, weaker user cost/job analytics. | User/project dashboards and cost visibility. |
| Multi-vendor GPU | NVIDIA-centric. | NVIDIA first, AMD MI300/ROCm path built into roadmap. |
| Active development | Repo is read-only/stable. | Kuafu owns active evolution and forks where needed. |
| HA/DR validation | Not explicit enough for Kuafu bar. | RTO/RPO, backups, chaos tests, scale tests. |
| Plugin ecosystem | Modular but not a modern plugin product. | Scheduler/admission/cost/runtime plugins. |

## Component Reuse Decision

| Component | Decision | Notes |
| --- | --- | --- |
| FrameworkController | Spike first | Strong candidate for Kuafu runtime adapter. |
| HiveD | Spike first | Strong topology-aware scheduler candidate; maintenance risk. |
| Volcano | Keep fallback | Active CNCF project if HiveD is too costly. |
| Kueue | Use selectively | Strong quota/admission ideas, not enough alone. |
| REST server | Reference only | Kuafu backend is Go and should not copy Node service wholesale. |
| Web portal | Reference only | Useful UX patterns, but Kuafu needs modern purpose-built UI. |
| Database controller | Pattern reuse | Reimplement in Go with MongoDB/Kubernetes ownership rules. |
| Monitoring configs | Reuse where practical | Prometheus/Grafana/Fluentd configs are useful starting points. |
| Device plugin | Prefer GPU Operator | NVIDIA GPU Operator is production target. |

## OpenPAI-Plus Requirements

Kuafu should be positioned as OpenPAI plus:

- Rich CLI.
- Reservations.
- Billing.
- Modern dashboard.
- GPU fractionalization.
- Multi-vendor GPU support.
- Stronger observability and repair automation.
- Production-grade HA/DR and scale tests.
- Plugin architecture.
- Clear lab/production boundary.

## Risks

| Risk | Severity | Mitigation |
| --- | --- | --- |
| HiveD is stale or hard to maintain | High | Spike early; keep Volcano fallback. |
| FrameworkController does not fit Kuafu job model | Medium | Adapter boundary; evaluate with real A00/A01 Kubernetes cluster. |
| OpenPAI dependencies are old | Medium | Fork only needed components; update dependencies intentionally. |
| OpenPAI feature assumptions do not cover MI300 | Medium | Add AMD device plugin milestone and separate validation path. |
| Copying OpenPAI too closely limits differentiation | High | Use OpenPAI as reference, not product ceiling. |
