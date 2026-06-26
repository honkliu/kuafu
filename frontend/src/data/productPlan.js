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
  { name: 'PyTorch distributed training', image: 'nvcr.io/nvidia/pytorch:24.05-py3', command: 'torchrun --nproc_per_node=8 train.py', gpu: 'A100/H100', status: 'Approved' },
  { name: 'Jupyter GPU workspace', image: 'quay.io/jupyter/pytorch-notebook:cuda12', command: 'start-notebook.py', gpu: '1-8 GPUs', status: 'Executor pending' },
  { name: 'Ray cluster head', image: 'rayproject/ray:latest-gpu', command: 'ray start --head', gpu: 'multi-node', status: 'Runtime pending' },
  { name: 'MPI / NCCL benchmark', image: 'nvcr.io/nvidia/nccl:latest', command: 'all_reduce_perf', gpu: 'topology-aware', status: 'Approved' },
];

export const skuPrices = [
  { sku: 'A100-SXM4-80GB', price: 12.8, region: 'A00/A01 lab', availability: '16 GPUs' },
  { sku: 'H100-80GB', price: 28.0, region: 'future pool', availability: 'planned' },
  { sku: 'MI300X', price: 18.0, region: 'future pool', availability: 'planned' },
];

export const workspaceTemplates = ['JupyterLab GPU', 'VS Code Server', 'SSH Shell', 'Web Terminal', 'Ray Workspace', 'Custom Container'];

export { HardDrive, Play, Server, Cpu, Layers, Activity, ShieldCheck, Gauge, DollarSign, Package };
