# Kuafu Testbed: A00 and A01

Status: initial smoke test complete  
Date: 2026-06-24

## Access

The first Kuafu testbed has two A100 machines reachable through the same public endpoint with different SSH ports.

| Kuafu name | SSH command | Remote hostname | Private Ethernet IP | First IB IP |
| --- | --- | --- | --- | --- |
| A00 | `ssh -i C:\Users\shuguanl\Downloads\superbench_a100.key -p 50000 hpcuser@52.171.138.19` | `hpcdev000000` | `10.0.0.4` | `172.16.3.146` |
| A01 | `ssh -i C:\Users\shuguanl\Downloads\superbench_a100.key -p 50001 hpcuser@52.171.138.19` | `hpcdev000001` | `10.0.0.5` | `172.16.2.234` |

The SSH key stays outside the repo under `C:\Users\shuguanl\Downloads`. Do not copy the private key into this repository.

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
| kubectl | Missing | Missing | Required before Kubernetes tests. |
| kubelet | Missing | Missing | Required before these machines can be Kubernetes nodes. |

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

This is enough for early two-node distributed GPU experiments after Kubernetes or a lightweight launcher is installed.

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

1. Install missing developer and Kubernetes tools: Go, `kubectl`, `kubelet`, `kubeadm` or the selected lightweight Kubernetes distribution.
2. Decide whether the first two-node cluster should use kubeadm, kind-on-Docker, k3s, or an OpenPAI-compatible Kubernetes setup.
3. Install or validate NVIDIA Kubernetes device plugin / GPU Operator on the cluster.
4. Run a two-node Kubernetes GPU scheduling smoke test with one pod per node.
5. Run a multi-node container communication test over IB before implementing distributed job launcher integration.
6. Use this testbed for the first Kuafu proof of concept: resource inventory discovery, scheduler adapter spike, and CLI/API smoke tests.

## Constraints

- Kubernetes is not installed yet on either machine.
- Go is not installed yet, so Kuafu Go backend/CLI binaries cannot be built on these machines without setup.
- This testbed is NVIDIA A100-only. It does not validate MI300/ROCm behavior.
- The current report is based on read-only inspection plus a Docker GPU smoke test. No system packages or Kubernetes components were installed.
