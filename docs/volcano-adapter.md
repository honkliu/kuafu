# Volcano Adapter Scaffold

Status: Step 17 scaffold  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Volcano is the Kubernetes-native fallback scheduler if FrameworkController/HiveD integration is blocked.

## Current Scaffold

The non-mutating manifest builder is in [internal/scheduler/volcano](../internal/scheduler/volcano).

It converts `RuntimeJobSpec` into a Volcano-like `batch.volcano.sh/v1alpha1` job manifest shape:

- Preserves Kuafu labels.
- Maps Kuafu queue to Volcano queue.
- Requests `nvidia.com/gpu` limit.
- Uses one worker task with `restartPolicy: Never` for the first scaffold.

## Boundaries

This does not create Kubernetes objects. A real client-go adapter is blocked until A00/A01 Kubernetes is available.

## Exit Criteria

Step 17 scaffold is complete when:

- Volcano manifest translation exists.
- Translation is unit-tested.
- No live cluster dependency is introduced.
- Scheduler strategy ADR documents the fallback decision.
