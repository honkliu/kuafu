# Sync Controller

Status: Step 10/11 scaffold  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Kuafu separates business intent from runtime state:

- Repository stores job intent, history, and queryable business state.
- Scheduler adapters create or read Kubernetes runtime objects.
- Sync controller reconciles between those two worlds.

## Current Scaffold

The implementation is in [internal/controller/sync_controller.go](../internal/controller/sync_controller.go).

Current behavior:

- Queued jobs without `RuntimeID` are submitted through `scheduler.RuntimeAdapter`.
- Successful submission stores `RuntimeID` and marks the job running.
- Non-terminal jobs with `RuntimeID` read runtime status and write status changes back to the repository.
- Terminal jobs are skipped.

## Boundaries

This controller is not wired into production startup yet. It is a tested scaffold for the future MongoDB + Kubernetes runtime path.

The lab scheduler remains separate and should not be used in production mode.

## Exit Criteria

The Step 10/11 scaffold is complete when:

- Job intent submission to runtime adapter is implemented and tested.
- Runtime status synchronization back to repository is implemented and tested.
- The controller depends only on repository and scheduler adapter interfaces.
- Production startup still fails before runtime wiring is complete.
