# Queue And Quota Policy

Status: Step 13 scaffold  
Owner: Kuafu GPU Architect  
Date: 2026-06-25

## Purpose

Kuafu needs queue and quota semantics before it can safely admit jobs to a real scheduler.

## Current Model

The queue domain model now includes:

- Hard GPU quota: `MaxGPUs`, never exceeded by admission.
- Soft GPU quota: `SoftGPUs`, exceeded only when burst is allowed.
- Per-job GPU limit: `MaxGPUsPerJob`.
- Queued and running job limits.
- Burst flag for opportunistic use above soft quota and below hard quota.
- Project ownership placeholder.

The admission scaffold is [internal/policy/queue.go](../internal/policy/queue.go).

## Current Semantics

- Zero-valued limits are treated as unset for backward compatibility with existing lab fixtures.
- Quota denials use typed `quota` errors.
- Invalid job/queue inputs use typed `permanent` or `not_found` errors.
- `AllowBurst` lets a queue exceed soft GPU quota opportunistically, but never hard GPU quota.
- Admission computes queued/running GPU usage from current jobs; it does not use job counts as a proxy for GPU usage.

## Future Scheduler Mapping

- HiveD: map queue/project to virtual cluster and guaranteed cell quota.
- Volcano: map queue fields to Volcano Queue and scheduling policy.
- Kueue: map quota concepts to ClusterQueue/LocalQueue and ResourceFlavor.

## Exit Criteria

Step 13 scaffold is complete when:

- Queue domain can represent hard/soft quota, per-job limits, queue limits, and burst policy.
- Admission policy returns typed errors.
- Tests cover allow, quota deny, permanent deny, and burst behavior.
- Existing lab queue behavior remains compatible.
