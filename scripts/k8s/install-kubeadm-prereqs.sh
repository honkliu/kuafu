#!/usr/bin/env bash
set -euo pipefail

KUBERNETES_MINOR="${KUBERNETES_MINOR:-v1.31}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/safety.sh"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run as root: sudo KUBERNETES_MINOR=${KUBERNETES_MINOR} bash $0" >&2
  exit 1
fi

ensure_safe_to_mutate_container_host

echo "Installing kubeadm prerequisites for ${KUBERNETES_MINOR} on $(hostname)"

swapoff -a
sed -i.bak '/\sswap\s/s/^/#/' /etc/fstab

cat >/etc/modules-load.d/kuafu-k8s.conf <<'EOF'
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter

cat >/etc/sysctl.d/99-kuafu-k8s.conf <<'EOF'
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
EOF
sysctl --system

apt-get update
apt-get install -y apt-transport-https ca-certificates curl gpg

install -d -m 0755 /etc/apt/keyrings
curl -fsSL "https://pkgs.k8s.io/core:/stable:/${KUBERNETES_MINOR}/deb/Release.key" \
  | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
chmod 0644 /etc/apt/keyrings/kubernetes-apt-keyring.gpg

cat >/etc/apt/sources.list.d/kubernetes.list <<EOF
deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/${KUBERNETES_MINOR}/deb/ /
EOF
chmod 0644 /etc/apt/sources.list.d/kubernetes.list

apt-get update
apt-get install -y kubelet kubeadm kubectl containernetworking-plugins
apt-mark hold kubelet kubeadm kubectl

mkdir -p /etc/containerd
containerd config default >/etc/containerd/config.toml
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
if command -v nvidia-ctk >/dev/null 2>&1; then
  nvidia-ctk runtime configure --runtime=containerd --set-as-default
fi
systemctl restart containerd
systemctl enable --now kubelet

echo "Installed versions:"
kubeadm version
kubectl version --client
kubelet --version