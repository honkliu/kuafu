#!/usr/bin/env bash
set -euo pipefail

ensure_safe_to_mutate_container_host() {
  local running
  running="$(docker ps -q 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "${running}" != "0" && "${KUAFU_ALLOW_CONTAINER_IMPACT:-false}" != "true" ]]; then
    echo "Refusing to mutate Kubernetes/container runtime state on $(hostname): ${running} Docker containers are running." >&2
    echo "Running containers:" >&2
    docker ps --format '  {{.ID}} {{.Image}} {{.Names}} {{.Status}}' >&2
    echo "Set KUAFU_ALLOW_CONTAINER_IMPACT=true only during an approved maintenance window." >&2
    exit 2
  fi
}