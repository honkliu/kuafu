# Repository Boundary

Status: Wave 2 foundation  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Kuafu must be able to replace the lab in-memory repository with MongoDB-backed business state without rewriting API handlers or scheduler code.

## Current Boundary

The repository contracts live in [internal/repository/interfaces.go](../internal/repository/interfaces.go):

- `InventoryRepository`: nodes and GPUs.
- `JobRepository`: job intent and history.
- `QueueRepository`: queue state.
- `GPUAllocationRepository`: atomic GPU allocation/release operations used by lab scheduling.
- `Repository`: combined contract used by API and lab scheduler.

`MemoryRepository` remains the lab and test implementation. The API server and lab scheduler now depend on interfaces instead of the concrete memory type.

The first MongoDB implementation lives in [internal/repository/mongodb](../internal/repository/mongodb). It implements the same contract and includes integration tests gated by `KUAFU_MONGODB_TEST_URI`.

## MongoDB Requirements

The future MongoDB implementation must:

- Preserve the same repository contracts.
- Store job intent and history durably.
- Store queues, users/projects, audit events, and usage records in later steps.
- Use atomic writes or transactions for allocation-like state transitions.
- Keep Kubernetes runtime state separate from business/query state.

## Exit Criteria

This boundary is complete when:

- The repository interfaces compile and are used by API and scheduler constructors.
- `MemoryRepository` satisfies the combined contract.
- Existing API and scheduler tests still pass unchanged.
- MongoDB implementation exists as a new package behind the same contract.
