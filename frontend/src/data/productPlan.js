import {
  Activity,
  ClipboardList,
  Cpu,
  DollarSign,
  FileText,
  Gauge,
  HardDrive,
  Layers,
  Package,
  Play,
  Server,
  Settings,
  ShieldCheck,
  SquareTerminal,
  Users,
} from 'lucide-react';

export const navigation = [
  { id: 'cluster', label: 'Cluster', icon: Activity },
  { id: 'nodes', label: 'Nodes', icon: Server },
  { id: 'jobs', label: 'Jobs', icon: ClipboardList },
  { id: 'queues', label: 'Queues', icon: Layers },
  { id: 'leases', label: 'Leases', icon: ShieldCheck },
  { id: 'workspaces', label: 'Workspaces', icon: SquareTerminal },
  { id: 'monitoring', label: 'Monitoring', icon: Gauge },
  { id: 'cost', label: 'Cost', icon: DollarSign },
  { id: 'catalog', label: 'Catalog', icon: Package },
  { id: 'projects', label: 'Projects', icon: Users },
  { id: 'audit', label: 'Audit', icon: FileText },
  { id: 'admin', label: 'Admin', icon: Settings },
];

export const iterations = [
  ['Platform IA', 'Cluster, nodes, jobs, leases, workspaces, monitoring, cost, catalog, projects, audit, and admin surfaces.'],
  ['Production Job Spec', 'Task roles, env, mounts, ports, secrets, retry policy, priority, SKU/flavor, topology, and YAML/API preview.'],
  ['Scheduler Runtime', 'Adapter-backed dry run, quota admission, gang scheduling, topology requests, and real Kubernetes status sync.'],
  ['Lease Enforcement', 'Time-window leases, conflict checks, expiry, extension, idle backfill policy, and scheduler enforcement.'],
  ['Workspace Execution', 'Jupyter, VS Code, SSH/web terminal, custom containers, access URLs, storage mounts, TTL, and audit.'],
  ['Monitoring', 'DCGM/ROCm metrics, Prometheus, Grafana deep links, logs, events, queue pressure, and failure diagnosis.'],
  ['Cost Accounting', 'GPU-hour estimates, SKU prices, balance/budget, chargeback, immutable usage ledger, and CSV export.'],
  ['Governance', 'OIDC/RBAC, template catalog, approvals, audit, node drain/cordon, GPU quarantine, and break-glass controls.'],
];

export const catalogTemplates = [
  { id: 'pytorch-ddp', name: 'PyTorch distributed training', image: 'nvcr.io/nvidia/pytorch:24.05-py3', command: 'torchrun --nproc_per_node=8 train.py', gpu: 'A100/H100', gpuCount: 8, status: 'Approved', description: 'Baseline multi-GPU training template.' },
  { id: 'jupyter-gpu', name: 'Jupyter GPU workspace', image: 'quay.io/jupyter/pytorch-notebook:cuda12', command: 'start-notebook.py', gpu: '1-8 GPUs', gpuCount: 1, status: 'Workspace', description: 'Interactive notebook workspace template.' },
  { id: 'ray-head', name: 'Ray cluster head', image: 'rayproject/ray:latest-gpu', command: 'ray start --head', gpu: 'multi-node', gpuCount: 1, status: 'Cluster', description: 'Cluster head template for Ray workloads.' },
  { id: 'nccl-benchmark', name: 'MPI / NCCL benchmark', image: 'nvcr.io/nvidia/nccl:latest', command: 'all_reduce_perf', gpu: 'topology-aware', gpuCount: 8, status: 'Approved', description: 'Fabric and collectives validation template.' },
  {
    id: 'glm52-h200-fp8-low-latency',
    name: 'GLM-5.2 FP8 low-latency serving',
    image: 'lmsysorg/sglang:latest',
    command: 'sglang serve --model-path zai-org/GLM-5.2-FP8 --tp 8 --speculative-algorithm EAGLE --speculative-num-steps 5 --speculative-eagle-topk 1 --speculative-num-draft-tokens 6 --mem-fraction-static 0.8 --cuda-graph-max-bs 32 --host 0.0.0.0 --port 30000 --reasoning-parser glm45 --tool-call-parser glm47',
    gpu: '8x H200 · FP8 · single node',
    gpuCount: 8,
    status: 'Verified recipe',
    description: 'SGLang GLM-5.2 recipe for chat/agent latency: TP8, EAGLE MTP 5-1-6, reasoning and tool-call parsers enabled.',
    source: 'SGLang GLM-5.2 cookbook',
  },
  {
    id: 'glm52-h200-fp8-balanced',
    name: 'GLM-5.2 FP8 balanced serving',
    image: 'lmsysorg/sglang:latest',
    command: 'sglang serve --model-path zai-org/GLM-5.2-FP8 --tp 8 --dp 8 --enable-dp-attention --moe-a2a-backend deepep --speculative-algorithm EAGLE --speculative-num-steps 1 --speculative-eagle-topk 1 --speculative-num-draft-tokens 2 --mem-fraction-static 0.85 --cuda-graph-max-bs 128 --chunked-prefill-size 32768 --max-running-requests 80 --host 0.0.0.0 --port 30000 --reasoning-parser glm45 --tool-call-parser glm47',
    gpu: '8x H200 · FP8 · multi-user',
    gpuCount: 8,
    status: 'Verified recipe',
    description: 'SGLang balanced point: DP-attention, DeepEP, larger chunked prefill, and concurrency cap for long-context serving.',
    source: 'SGLang GLM-5.2 cookbook',
  },
  {
    id: 'glm52-b300-nvfp4-low-latency',
    name: 'GLM-5.2 NVFP4 low-latency serving',
    image: 'lmsysorg/sglang:dev-glm52-nvfp4',
    command: 'sglang serve --trust-remote-code --model-path nvidia/GLM-5.2-NVFP4 --tp 4 --quantization modelopt_fp4 --speculative-algorithm EAGLE --speculative-num-steps 5 --speculative-eagle-topk 1 --speculative-num-draft-tokens 6 --chunked-prefill-size 131072 --mem-fraction-static 0.70 --host 0.0.0.0 --port 30000 --reasoning-parser glm45 --tool-call-parser glm47',
    gpu: '4x B300/GB300 · NVFP4',
    gpuCount: 4,
    status: 'Reference recipe',
    description: 'Blackwell NVFP4 reference recipe using SGLang dev image and modelopt_fp4 quantization.',
    source: 'SGLang GLM-5.2 cookbook',
  },
];

export const skuPrices = [
  { sku: 'A100-SXM4-80GB', price: 12.8, region: 'A00/A01 lab', availability: '16 GPUs' },
  { sku: 'H100-80GB', price: 28.0, region: 'future pool', availability: 'planned' },
  { sku: 'MI300X', price: 18.0, region: 'future pool', availability: 'planned' },
];

export const workspaceTemplates = ['JupyterLab GPU', 'VS Code Server', 'SSH Shell', 'Web Terminal', 'Ray Workspace', 'Custom Container'];

export { HardDrive, Play, Server, Cpu, Layers, Activity, ShieldCheck, Gauge, DollarSign, Package };
