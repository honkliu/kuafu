# Kuafu Kubernetes Bootstrap Scripts

These scripts support Step 3. Only `preflight.sh` is read-only. The other scripts install packages or change Kubernetes state and require explicit approval before execution on A00/A01.

Mutating scripts refuse to run when Docker containers are active unless `KUAFU_ALLOW_CONTAINER_IMPACT=true` is set. Use that override only during an approved maintenance window.

Suggested order:

1. Copy `scripts/k8s/preflight.sh` to A00 and A01; run it as `hpcuser`.
2. Copy `install-kubeadm-prereqs.sh` to both nodes; run with `sudo`.
3. Run `init-control-plane-a00.sh` with `sudo` on A00.
4. Copy `/tmp/kuafu-kubeadm-join.sh` output from A00 and pass it to `join-worker-a01.sh` with `sudo` on A01.
5. Run `validate-cluster.sh` from A00 as `hpcuser`.

Do not commit kubeconfig files, join tokens, private keys, or command output containing secrets.

Keep mutating commands separated from read-only preflight output.

The safety guard should be tested before any approved maintenance-window execution.

The prerequisite script installs CNI plugins and configures NVIDIA as the default containerd runtime when `nvidia-ctk` is present. Both were required for the A00/A01 GPU bootstrap.
