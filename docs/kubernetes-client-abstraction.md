# Kubernetes Client Abstraction

Status: Step 5 initial abstraction  
Owner: Kuafu Backend Developer  
Date: 2026-06-25

## Purpose

Kuafu production code must depend on a narrow Kubernetes-facing contract instead of hard-coding lab fixtures or binding every package directly to `client-go`.

The first contract is inventory discovery:

- `ListNodes(ctx)` returns Kubernetes-discovered compute nodes projected into Kuafu domain nodes.
- `ListGPUs(ctx)` returns GPU resources projected into Kuafu domain GPUs.
- `SyncInventory(ctx, client, repo)` stores discovered inventory in the repository boundary.

## Current Implementation

Files:

- [internal/k8s/inventory.go](../internal/k8s/inventory.go)
- [internal/k8s/fake.go](../internal/k8s/fake.go)
- [internal/k8s/inventory_test.go](../internal/k8s/inventory_test.go)

The current implementation includes a fake inventory client for tests. It is not a Kubernetes runtime adapter.

## Production Adapter Requirements

The real adapter should be added after Step 3/4 create a working A00/A01 Kubernetes cluster with GPU resources exposed by NVIDIA GPU Operator or the device plugin.

The real adapter must:

- Use Kubernetes API discovery, not shell commands.
- Map node readiness to `domain.NodeStatus`.
- Map allocatable `nvidia.com/gpu` and device-plugin metadata to `domain.GPU` records.
- Preserve labels useful for topology, SKU, vendor, and scheduling.
- Avoid storing kubeconfig or secrets in the repo.
- Use fake client tests for projection logic and real-cluster smoke tests for A00/A01.

## Step 5 Exit Criteria

Step 5 is complete when:

- The inventory abstraction compiles and is tested.
- The fake client supports unit tests without a cluster.
- The production adapter boundary is documented.
- A later real client-go adapter can be added without changing API handlers.

The actual live Kubernetes adapter remains blocked until Step 3/4 are complete.
