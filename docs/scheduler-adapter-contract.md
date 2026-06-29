# Scheduler Adapter Contract

Status: Step 6/7 scaffold  
Owner: Kuafu GPU Architect  
Date: 2026-06-25

## Purpose

Kuafu must not hard-code one scheduler too early. The production path needs an adapter boundary so FrameworkController, HiveD, Volcano, and later Kueue-oriented admission can be evaluated without rewriting API handlers.

## Contract

The initial contract is in [internal/scheduler/adapter.go](../internal/scheduler/adapter.go):

- `Submit(ctx, RuntimeJobSpec)` creates a runtime object and returns a runtime ID. It must be idempotent by Kuafu `JobID`: re-submitting the same job returns the existing runtime object and must not create duplicates.
- `Cancel(ctx, jobID)` cancels the runtime job.
- `Status(ctx, jobID)` reads runtime job status.

The domain projection is [internal/domain/job.go](../internal/domain/job.go): `Job.RuntimeSpec()` maps Kuafu job intent to the scheduler-adapter-facing shape.

## Current State

The repo includes `UnsupportedRuntimeAdapter` so production startup can fail honestly until a real adapter is wired. The existing lab scheduler remains separate from production adapters and executes task commands through Docker in lab mode.

## Adapter Requirements

FrameworkController adapter:

- Convert `RuntimeJobSpec` to FrameworkController `Framework` CR.
- Preserve job ID and queue labels.
- Request `nvidia.com/gpu` resources.
- Surface pod logs and terminal state back to Kuafu.

HiveD integration:

- Map Kuafu queue/project to HiveD virtual cluster.
- Preserve topology constraints when added to `RuntimeJobSpec`.

Volcano fallback:

- Convert `RuntimeJobSpec` to Volcano job/queue objects.
- Preserve gang scheduling and queue semantics.

## Exit Criteria

This scaffold is complete when:

- `RuntimeJobSpec` exists in the domain model.
- A scheduler adapter interface exists.
- Unsupported production adapter fails clearly.
- Tests cover projection and unsupported behavior.

Real adapter implementation remains blocked by Step 3/4 cluster availability.
