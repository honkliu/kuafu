# Kuafu Testbed: A00 and A01

Status: Kubernetes and GPU device-plugin bootstrap complete  
Date: 2026-06-26

## Access

The first Kuafu testbed has two A100 machines reachable through the same public endpoint with different SSH ports.

| Kuafu name | SSH command | Remote hostname | Private Ethernet IP | First IB IP |
| --- | --- | --- | --- | --- |
| A00 | `ssh -i C:\Users\shuguanl\Downloads\superbench_a100.key -p 50000 hpcuser@52.171.138.19` | `hpcdev000000` | `10.0.0.4` | `172.16.3.146` |
| A01 | `ssh -i C:\Users\shuguanl\Downloads\superbench_a100.key -p 50001 hpcuser@52.171.138.19` | `hpcdev000001` | `10.0.0.5` | `172.16.2.234` |

The SSH key stays outside the repo under `C:\Users\shuguanl\Downloads`. Do not copy the private key into this repository.

## External Access

A00 can expose HTTP externally on port `30000`. Early Kuafu lab or Kubernetes NodePort access should use:

```text
http://<A00-public-ip>:30000
```

This is an external ingress constraint only. Communication between A00 and A01 can use arbitrary ports on the private Ethernet and InfiniBand networks.

## Hardware And OS

| Item | A00 | A01 |
| --- | --- | --- |
| OS | Ubuntu 22.04.5 LTS | Ubuntu 22.04.5 LTS |
| Kernel | Linux 5.15.0-1088-azure x86_64 | Linux 5.15.0-1088-azure x86_64 |
| CPU count | 96 | 96 |
| Memory | 1.7 TiB | 1.7 TiB |
| Root disk | 497 GB total, 309 GB free | 497 GB total, 453 GB free |
| GPUs | 8x NVIDIA A100-SXM4-80GB | 8x NVIDIA A100-SXM4-80GB |
| GPU memory | 81920 MiB each | 81920 MiB each |
| NVIDIA driver | 570.133.20 | 570.133.20 |

Total current testbed capacity: 2 nodes, 16 A100-SXM4-80GB GPUs, 192 CPU cores, about 3.4 TiB memory.

## Runtime Readiness

| Capability | A00 | A01 | Notes |
| --- | --- | --- | --- |
| SSH access | Pass | Pass | Non-interactive key auth works. |
| Docker installed | Pass | Pass | A00 Docker 29.1.2-2, A01 Docker 29.2.1-1. |
| User in docker group | Pass | Pass | `hpcuser` can run Docker without sudo. |
| NVIDIA Container Toolkit | Pass | Pass | A00 1.17.8, A01 1.18.2. |
| Docker GPU smoke test | Pass | Pass | `docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi -L`. |
| Passwordless sudo | Pass | Pass | `sudo -n true` succeeds. |
| Python 3 | Pass | Pass | Python 3.10.12. |
| Git | Pass | Pass | Git 2.34.1. |
| Go | Missing | Missing | Required for Kuafu backend/CLI development. |
| kubectl | Pass | Pass | Kubernetes client v1.36.1. |
| kubeadm | Pass | Pass | kubeadm v1.36.1. |
| kubelet | Pass | Pass | kubelet v1.36.1. |
| Kubernetes node | Ready control-plane | Ready worker | A00 initialized with kubeadm; A01 joined as worker. |
| Kubernetes GPU resource | Pass | Pass | NVIDIA device plugin exposes `nvidia.com/gpu: 8` per node. |

## Network And Topology

Both nodes have one Ethernet interface and eight InfiniBand interfaces.

| Check | Result |
| --- | --- |
| A00 to A01 Ethernet ping | Pass, `10.0.0.4 -> 10.0.0.5`, 0 percent packet loss. |
| A01 to A00 Ethernet ping | Pass, `10.0.0.5 -> 10.0.0.4`, 0 percent packet loss. |
| A00 to A01 IB ping | Pass, `172.16.3.146 -> 172.16.2.234`, 0 percent packet loss. |
| A01 to A00 IB ping | Pass, `172.16.2.234 -> 172.16.3.146`, 0 percent packet loss. |
| InfiniBand HCAs | 8 active Mellanox `mlx5_ib*` devices per node. |
| IB rate | 200 Gb/s per active HCA. |
| Intra-node GPU topology | All 8 GPUs are connected by `NV12` links on both nodes. |
| GPU to NIC locality | GPUs are paired with nearby NIC groups by NUMA locality. |

This is enough for early two-node distributed GPU experiments and scheduler adapter spikes.

## Kubernetes Bootstrap Result

| Item | Result |
| --- | --- |
| Control plane | A00 `hpcdev000000` |
| Worker | A01 `hpcdev000001` |
| Kubernetes version | v1.36.1 from Microsoft Ubuntu package feed |
| CNI | Flannel, plus `containernetworking-plugins` for `/opt/cni/bin/loopback` |
| GPU management | NVIDIA device plugin v0.17.1 fallback |
| GPU resource | `nvidia.com/gpu: 8` allocatable on each node |
| GPU smoke | Kubernetes pod requesting `nvidia.com/gpu: 1` ran `nvidia-smi -L` successfully |
| Existing Docker containers | Stopped before bootstrap and restarted after validation |
| A01 open-webui | Restored and healthy on host port `30000` |

## Commands Already Run

Read-only probes:

```powershell
ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15 -i 'C:\Users\shuguanl\Downloads\superbench_a100.key' -p 50000 hpcuser@52.171.138.19 "hostname; whoami; uname -srmo; head -5 /etc/os-release; nproc; free -h; df -h /; nvidia-smi -L; nvidia-smi --query-gpu=index,name,memory.total,driver_version --format=csv,noheader; docker --version; containerd --version; kubectl version --client; kubelet --version"
ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=15 -i 'C:\Users\shuguanl\Downloads\superbench_a100.key' -p 50001 hpcuser@52.171.138.19 "hostname; whoami; uname -srmo; head -5 /etc/os-release; nproc; free -h; df -h /; nvidia-smi -L; nvidia-smi --query-gpu=index,name,memory.total,driver_version --format=csv,noheader; docker --version; containerd --version; kubectl version --client; kubelet --version"
```

Docker GPU smoke test:

```powershell
ssh -i 'C:\Users\shuguanl\Downloads\superbench_a100.key' -p 50000 hpcuser@52.171.138.19 "docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi -L"
ssh -i 'C:\Users\shuguanl\Downloads\superbench_a100.key' -p 50001 hpcuser@52.171.138.19 "docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi -L"
```

Topology and cross-node reachability checks:

```powershell
ssh -i 'C:\Users\shuguanl\Downloads\superbench_a100.key' -p 50000 hpcuser@52.171.138.19 "hostname -I; ip -br addr | head -20; nvidia-smi topo -m; ibstat; ibv_devinfo | head -80"
ssh -i 'C:\Users\shuguanl\Downloads\superbench_a100.key' -p 50001 hpcuser@52.171.138.19 "hostname -I; ip -br addr | head -20; nvidia-smi topo -m; ibstat; ibv_devinfo | head -80"
```

## Immediate Test Plan

1. Run FrameworkController/HiveD scheduler adapter spike.
2. Keep Volcano fallback scaffold ready for live Kubernetes validation.
3. Run a multi-node container communication test over IB before distributed training launcher integration.
4. Use this testbed for resource inventory discovery, scheduler adapter spike, and CLI/API smoke tests.

## Constraints

- Go is not installed yet, so Kuafu Go backend/CLI binaries cannot be built on these machines without setup.
- This testbed is NVIDIA A100-only. It does not validate MI300/ROCm behavior.
- NVIDIA GPU Operator is not installed yet; the current validated GPU path uses the standalone NVIDIA device plugin fallback.
- A01 `open-webui` owns host port `30000`; do not blindly apply a cluster-wide Kuafu NodePort `30000` service until port ownership is resolved.
