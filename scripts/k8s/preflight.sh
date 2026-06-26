#!/usr/bin/env bash
set -euo pipefail

echo "NODE=$(hostname)"
whoami
uname -srmo
head -5 /etc/os-release
nproc
lscpu | grep -E '^(CPU\(s\)|Socket|Core|Thread|NUMA)' || true
free -h | head -2
df -h /

echo "== sudo =="
sudo -n true && echo sudo-ok

echo "== runtime =="
docker --version
docker info --format 'ServerVersion={{.ServerVersion}} CgroupDriver={{.CgroupDriver}} CgroupVersion={{.CgroupVersion}}'
containerd --version || true
systemctl is-active docker || true
systemctl is-active containerd || true

echo "== nvidia =="
nvidia-smi -L
nvidia-smi --query-gpu=index,name,memory.total,driver_version --format=csv,noheader
nvidia-ctk --version || true

echo "== kubernetes tools =="
kubectl version --client || true
kubeadm version || true
kubelet --version || true

echo "== swap =="
swapon --show || true

echo "== network =="
hostname -I
ip -br addr | head -20