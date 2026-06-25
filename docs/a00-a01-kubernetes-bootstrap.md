# A00/A01 Kubernetes Bootstrap Plan

Status: requires explicit approval before execution  
Date: 2026-06-24

## Purpose

This document defines the first real Kuafu production testbed milestone: bootstrap Kubernetes on A00/A01 and expose their A100 GPUs through Kubernetes.

Do not run these steps without explicit approval. They install system packages, configure Kubernetes services, and may alter Docker/container runtime behavior on the remote machines.

## Testbed

| Name | Role | Hostname | SSH port | Private IP | First IB IP | GPUs |
| --- | --- | --- | --- | --- | --- | --- |
| A00 | Proposed control plane | `hpcdev000000` | 50000 | `10.0.0.4` | `172.16.3.146` | 8x A100-SXM4-80GB |
| A01 | Proposed worker | `hpcdev000001` | 50001 | `10.0.0.5` | `172.16.2.234` | 8x A100-SXM4-80GB |

SSH key path:

```powershell
C:\Users\shuguanl\Downloads\superbench_a100.key
```

## Recommended Stack

| Layer | Choice | Reason |
| --- | --- | --- |
| Kubernetes distribution | kubeadm | Standard production path and closest to future large-cluster operations. |
| Control plane | A00 first | Small testbed; A00 has working Docker/NVIDIA runtime and private network access to A01. |
| Worker | A01 | Validates cross-node scheduling. |
| GPU management | NVIDIA GPU Operator, or standalone device plugin as fallback | Operator is production target; device plugin is faster fallback. |
| Scheduler spike | FrameworkController + HiveD first | Best OpenPAI-aligned topology-aware path. |
| Fallback scheduler | Volcano | Active CNCF option if HiveD is blocked. |

## Preflight Checks

Run read-only checks first:

```powershell
ssh -i C:\Users\shuguanl\Downloads\superbench_a100.key -p 50000 hpcuser@52.171.138.19 "hostname; docker --version; nvidia-smi -L; sudo -n true && echo sudo-ok"
ssh -i C:\Users\shuguanl\Downloads\superbench_a100.key -p 50001 hpcuser@52.171.138.19 "hostname; docker --version; nvidia-smi -L; sudo -n true && echo sudo-ok"
```

Expected:

- Both hosts reachable.
- Docker works.
- `nvidia-smi -L` shows 8 A100 GPUs per node.
- Passwordless sudo works.
- A00 can ping A01 private IP and IB IP; A01 can ping A00 private IP and IB IP.

## Bootstrap Outline

The exact command script should be reviewed by Architect before running. The high-level steps are:

1. Install Kubernetes package prerequisites on both nodes.
2. Configure kernel modules and sysctl for Kubernetes networking.
3. Configure container runtime for Kubernetes CRI compatibility.
4. Install `kubelet`, `kubeadm`, and `kubectl` on both nodes.
5. Initialize control plane on A00 using kubeadm.
6. Install a CNI plugin.
7. Join A01 as a worker.
8. Install NVIDIA GPU Operator or fallback NVIDIA device plugin.
9. Validate GPU scheduling with a CUDA pod.
10. Document all commands and outputs.

## Acceptance Criteria

Kubernetes bootstrap is complete when all checks pass:

```bash
kubectl get nodes -o wide
kubectl describe node hpcdev000000 | grep -i nvidia.com/gpu
kubectl describe node hpcdev000001 | grep -i nvidia.com/gpu
kubectl run gpu-smoke --rm -it --restart=Never --image=nvidia/cuda:12.4.1-base-ubuntu22.04 --limits='nvidia.com/gpu=1' -- nvidia-smi -L
```

Expected:

- A00 and A01 are `Ready`.
- Each node advertises 8 GPUs through Kubernetes.
- GPU smoke pod runs and prints at least one GPU.

## Post-Bootstrap Scheduler Spike

After GPU scheduling works:

1. Deploy FrameworkController.
2. Deploy HiveD or prepare Volcano fallback.
3. Submit a multi-GPU test job.
4. Verify gang scheduling behavior.
5. Wire Kuafu API to create Kubernetes runtime objects through a scheduler adapter.

## Safety Notes

- These steps alter system packages and services on A00/A01.
- Existing Docker workloads may be affected by container runtime changes.
- Do not store SSH keys or kubeconfig secrets in the repo.
- Keep lab-mode Kuafu available locally while production cluster work proceeds.

## PM Gate

Before execution, PM must confirm:

- A00/A01 are allowed to be modified.
- kubeadm is acceptable for the testbed.
- Any running workloads on A00/A01 can be interrupted if Kubernetes install requires service restarts.
- The team accepts this as the next production milestone.
