#!/usr/bin/env bash
set -euo pipefail

JOIN_COMMAND="${1:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/safety.sh"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run as root: sudo bash $0 '<kubeadm join ...>'" >&2
  exit 1
fi

ensure_safe_to_mutate_container_host

if [[ -f /etc/kubernetes/kubelet.conf ]]; then
  echo "Worker already appears joined: /etc/kubernetes/kubelet.conf exists"
  exit 0
fi

if [[ -z "${JOIN_COMMAND}" ]]; then
  echo "Usage: sudo bash $0 '<kubeadm join ...>'" >&2
  exit 1
fi

${JOIN_COMMAND}