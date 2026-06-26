#!/usr/bin/env bash
set -euo pipefail

APISERVER_ADVERTISE_ADDRESS="${APISERVER_ADVERTISE_ADDRESS:-10.0.0.4}"
POD_NETWORK_CIDR="${POD_NETWORK_CIDR:-10.244.0.0/16}"
CRI_SOCKET="${CRI_SOCKET:-unix:///run/containerd/containerd.sock}"
KUBE_USER="${KUBE_USER:-hpcuser}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/safety.sh"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run as root: sudo bash $0" >&2
  exit 1
fi

ensure_safe_to_mutate_container_host

if [[ -f /etc/kubernetes/admin.conf ]]; then
  echo "Kubernetes control plane already appears initialized: /etc/kubernetes/admin.conf exists"
  exit 0
fi

kubeadm init \
  --apiserver-advertise-address="${APISERVER_ADVERTISE_ADDRESS}" \
  --pod-network-cidr="${POD_NETWORK_CIDR}" \
  --cri-socket="${CRI_SOCKET}"

install -d -m 0755 "/home/${KUBE_USER}/.kube"
cp -f /etc/kubernetes/admin.conf "/home/${KUBE_USER}/.kube/config"
chown "${KUBE_USER}:${KUBE_USER}" "/home/${KUBE_USER}/.kube/config"

sudo -u "${KUBE_USER}" kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml

# This is a two-node GPU testbed; A00 must also be schedulable for GPU workloads.
sudo -u "${KUBE_USER}" kubectl taint nodes --all node-role.kubernetes.io/control-plane- || true

kubeadm token create --print-join-command >/tmp/kuafu-kubeadm-join.sh
chmod 0600 /tmp/kuafu-kubeadm-join.sh

echo "Join command written to /tmp/kuafu-kubeadm-join.sh"
sudo -u "${KUBE_USER}" kubectl get nodes -o wide