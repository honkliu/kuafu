#!/usr/bin/env bash
set -euo pipefail

kubectl get nodes -o wide
kubectl get pods -A

for node in hpcdev000000 hpcdev000001; do
  echo "== ${node} GPU allocatable =="
  kubectl describe node "${node}" | grep -i 'nvidia.com/gpu' || true
done