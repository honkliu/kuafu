# A00/A01 Kubernetes Bootstrap Plan

Status: Kubernetes bootstrap complete; GPU device plugin validated  
Date: 2026-06-25

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

## External Access Constraint

A00 currently has external ingress available on port `30000`. Use this as the public HTTP entry point for early Kuafu API/dashboard exposure:

```text
http://<A00-public-ip>:30000
```

Kubernetes should expose the Kuafu API through a NodePort pinned to `30000` on A00 for the early testbed. An example service manifest is available at [deploy/kubernetes/kuafu-api-nodeport.example.yaml](../deploy/kubernetes/kuafu-api-nodeport.example.yaml).

Internal A00/A01 communication is not constrained to this port. Pods, services, scheduler components, MongoDB, and controller traffic can use normal Kubernetes, private Ethernet, or InfiniBand ports inside the testbed.

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

## Preflight Result: 2026-06-25

Read-only checks passed on both hosts.

| Check | A00 | A01 |
| --- | --- | --- |
| SSH and hostname | `hpcdev000000` | `hpcdev000001` |
| OS | Ubuntu 22.04.5 LTS | Ubuntu 22.04.5 LTS |
| Kernel | Linux 5.15.0-1088-azure | Linux 5.15.0-1088-azure |
| CPU | 96 CPUs, 2 sockets, 48 cores/socket, 4 NUMA nodes | 96 CPUs, 2 sockets, 48 cores/socket, 4 NUMA nodes |
| Root disk | 497 GB total, 306 GB free | 497 GB total, 453 GB free |
| Passwordless sudo | Pass | Pass |
| Docker | Client 29.1.2-2, server 28.5.2-1, cgroup v2/systemd | Client/server 29.2.1-1, cgroup v2/systemd |
| containerd | 1.7.29-1, active | 2.2.1-1, active |
| NVIDIA GPUs | 8x A100-SXM4-80GB | 8x A100-SXM4-80GB |
| NVIDIA driver | 570.133.20 | 570.133.20 |
| NVIDIA Container Toolkit | 1.17.8 | 1.18.2 |
| Kubernetes tools | `kubectl`, `kubeadm`, `kubelet` missing | `kubectl`, `kubeadm`, `kubelet` missing |
| Swap | No active swap listed | No active swap listed |
| Private Ethernet ping | A00 -> A01 `10.0.0.5`: pass, 0% loss | A01 -> A00 `10.0.0.4`: pass, 0% loss |
| IB ping | A00 -> A01 `172.16.2.234`: pass, 0% loss | A01 -> A00 `172.16.3.146`: pass, 0% loss |
| Kubernetes package source | `https://pkgs.k8s.io/` reachable | `https://pkgs.k8s.io/` reachable |

Notable install consideration: A00 and A01 have different Docker/containerd versions. The kubeadm scripts must use containerd through the CRI socket and configure `SystemdCgroup = true` on both nodes.

Initial blocker for mutating bootstrap: running Docker containers existed and Docker live-restore was disabled.

| Node | Running containers | Live restore |
| --- | --- | --- |
| A00 | `sb-workspace`, `comfyui` | `false` |
| A01 | `sb-workspace`, `qwen`, `open-webui` | `false` |

To honor the requirement that existing containers are not impacted, mutating scripts now refuse to run while these containers are active unless `KUAFU_ALLOW_CONTAINER_IMPACT=true` is set during an approved maintenance window.

Safety guard validation: `install-kubeadm-prereqs.sh` was invoked on A00 without the override and refused before package installation, listing the running containers. No Kubernetes packages were installed and no runtime service was restarted.

## Bootstrap Result: 2026-06-26

After explicit approval, the existing Docker containers were stopped, Kubernetes was bootstrapped, GPU scheduling was validated, and the stopped containers were restarted.

| Check | Result |
| --- | --- |
| Kubernetes version | `v1.36.1` on A00 and A01. The Microsoft Ubuntu package feed supplied v1.36.1 despite the script's v1.31 source being configured. |
| Control plane | A00 `hpcdev000000` initialized successfully with kubeadm. |
| Worker | A01 `hpcdev000001` joined successfully with kubeadm. |
| CNI | Flannel installed; `containernetworking-plugins` was required because `/opt/cni/bin/loopback` was initially missing. |
| NVIDIA runtime | `nvidia-ctk runtime configure --runtime=containerd --set-as-default` required on both nodes. |
| GPU plugin | NVIDIA device plugin `v0.17.1` installed as the GPU Operator fallback. |
| Node readiness | A00 and A01 are `Ready`. |
| GPU resource advertisement | Each node advertises `nvidia.com/gpu: 8`. |
| Kubernetes GPU smoke | `kubectl run gpu-smoke ... nvidia-smi -L` succeeded and printed an A100 GPU. |
| Docker containers restored | A00 `sb-workspace`, `comfyui`; A01 `sb-workspace`, `qwen`, `open-webui`. |
| Docker GPU smoke after restore | Passed on both A00 and A01. |
| A01 open-webui | Restarted and reported `healthy`; still exposes host port `30000`. |

Important follow-up: because A01 `open-webui` uses host port `30000`, do not create a cluster-wide Kuafu NodePort `30000` service until ingress/port ownership is decided. The existing [NodePort example](../deploy/kubernetes/kuafu-api-nodeport.example.yaml) should be treated as A00-only intent, not applied blindly while A01 owns that port.

## Bootstrap Scripts

The executable scripts are stored in [scripts/k8s](../scripts/k8s):

| Script | Mutates host? | Purpose |
| --- | --- | --- |
| `preflight.sh` | No | Repeat read-only host/runtime/GPU checks. |
| `install-kubeadm-prereqs.sh` | Yes | Install kubeadm/kubelet/kubectl, configure sysctl/modules, configure containerd. |
| `init-control-plane-a00.sh` | Yes | Initialize A00 as control plane, install Flannel, make A00 schedulable, emit join command. |
| `join-worker-a01.sh` | Yes | Join A01 using the A00-generated `kubeadm join` command. |
| `validate-cluster.sh` | No | Validate nodes, pods, and GPU resource visibility after bootstrap. |

Script syntax was validated with `bash -n scripts/k8s/*.sh` on A00.

## Bootstrap Outline

The exact command script should be reviewed by Architect before running. The high-level steps are:

1. Install Kubernetes package prerequisites on both nodes, including CNI plugins.
2. Configure kernel modules and sysctl for Kubernetes networking.
3. Configure container runtime for Kubernetes CRI compatibility and NVIDIA as default containerd runtime.
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
- Kuafu API/dashboard can later be exposed through A00 NodePort `30000` without changing internal service ports.

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
