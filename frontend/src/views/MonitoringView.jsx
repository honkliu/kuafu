import { Activity, Gauge, Server, ShieldCheck } from 'lucide-react';
import { MetricCard, StatusBadge } from '../components/Primitives.jsx';
import { formatDate } from '../utils/format.js';
import { isGpuBusy } from '../utils/selectors.js';

export default function MonitoringView({ nodes, gpus, gpuTelemetry, jobs, reservations }) {
  const telemetryGpus = gpuTelemetry?.gpus || [];
  const usedGpus = telemetryGpus.filter(isGpuBusy).length;
  const avgUtilization = telemetryGpus.length ? Math.round(telemetryGpus.reduce((sum, gpu) => sum + (gpu.utilization || 0), 0) / telemetryGpus.length) : 0;
  const memoryUsed = telemetryGpus.reduce((sum, gpu) => sum + (gpu.memoryUsedMb || 0), 0);
  const memoryTotal = telemetryGpus.reduce((sum, gpu) => sum + (gpu.memoryTotalMb || 0), 0);
  const processes = telemetryGpus.flatMap((gpu) => (gpu.processes || []).map((process) => ({ ...process, gpu })));
  const taskRows = jobs.flatMap((job) => taskListForJob(job).map((task) => ({ job, task, telemetry: telemetryForTask(task, telemetryGpus) })));

  return <section className="panel stack">
    <div className="section-header"><div><p className="eyebrow">Observability · refreshes every minute</p><h2>Monitoring</h2><p>GPU usage is read from live telemetry, so external workloads such as ComfyUI are visible even when Kuafu did not allocate the GPU.</p></div><StatusBadge status={gpuTelemetry?.source || 'unknown'} /></div>
    {gpuTelemetry?.error && <div className="error-banner">GPU telemetry fallback: {gpuTelemetry.error}</div>}
    <div className="metric-grid"><MetricCard label="Live Busy GPUs" value={`${usedGpus}/${telemetryGpus.length || gpus.length}`} hint={`${avgUtilization}% average SM utilization`} icon={Gauge} /><MetricCard label="GPU Memory Used" value={formatMemory(memoryUsed)} hint={`${formatMemory(memoryTotal)} total`} icon={Server} /><MetricCard label="Running Jobs" value={jobs.filter((job) => job.status === 'Running').length} hint="Kuafu runtime active" icon={Activity} /><MetricCard label="Active Leases" value={reservations.length} hint="dedicated capacity" icon={ShieldCheck} /></div>
    <section className="panel nested-panel"><div className="section-header"><div><p className="eyebrow">Node to GPU</p><h3>Node drilldown</h3><p>Each node shows live GPU pressure and the GPUs behind it.</p></div></div><div className="node-monitor-grid">{nodes.map((node) => <article className="node-monitor-card" key={node.name}><div><strong>{node.name}</strong><StatusBadge status={node.status} /></div><MetricLine label="Busy GPUs" value={`${node.usedTelemetryGpus || 0}/${node.telemetryGpus?.length || node.gpuCount || 0}`} /><MetricLine label="Avg SM" value={`${node.avgUtilization || 0}%`} /><MetricLine label="Memory" value={`${formatMemory(node.memoryUsedMb)} / ${formatMemory(node.memoryTotalMb)}`} /><div className="gpu-chip-row">{(node.telemetryGpus || []).map((gpu) => <span className={isGpuBusy(gpu) ? 'gpu-chip busy' : 'gpu-chip'} key={gpu.uuid || gpu.index}>GPU {gpu.index}</span>)}</div></article>)}</div></section>
    <section className="telemetry-grid">{telemetryGpus.map((gpu) => <GPUCard key={gpu.uuid || gpu.index} gpu={gpu} />)}</section>
    <section className="panel nested-panel"><div className="section-header"><div><p className="eyebrow">Job and task to GPU</p><h3>Task placement and GPU usage</h3><p>Distributed jobs expose master/worker task ranks, GPU allocation, and current telemetry when the GPU is visible.</p></div></div><div className="table-wrap"><table className="data-table compact-table"><thead><tr><th>Job</th><th>Task</th><th>Role</th><th>Rank</th><th>GPU allocation</th><th>Live GPU telemetry</th><th>CPU / Memory request</th></tr></thead><tbody>{taskRows.length === 0 ? <tr><td colSpan="7">No task-level jobs yet.</td></tr> : taskRows.map(({ job, task, telemetry }) => <tr key={task.id}><td><strong>{job.name}</strong><small>{job.status}</small></td><td>{task.name}</td><td><StatusBadge status={task.role} /></td><td>{task.rank}</td><td>{(task.allocatedGpus || []).join(', ') || '-'}</td><td>{telemetry.length ? telemetry.map((gpu) => `GPU ${gpu.index}: ${gpu.utilization || 0}% SM, ${gpu.memoryUsedMb || 0} MB`).join(' | ') : 'pending / external'}</td><td>{task.cpuCount || 0} CPU / {task.memoryGb || 0} GB</td></tr>)}</tbody></table></div></section>
    <section className="panel nested-panel"><div className="section-header"><div><p className="eyebrow">GPU processes</p><h3>Live process owners</h3><p>Processes come from nvidia-smi compute-apps and include non-Kuafu workloads.</p></div><span className="selection-count">generated {formatDate(gpuTelemetry?.generatedAt)}</span></div><div className="table-wrap"><table className="data-table compact-table"><thead><tr><th>GPU</th><th>PID</th><th>Process</th><th>Memory</th></tr></thead><tbody>{processes.length === 0 ? <tr><td colSpan="4">No GPU processes reported.</td></tr> : processes.map((process) => <tr key={`${process.gpuIndex}-${process.pid}-${process.processName}`}><td>GPU {process.gpuIndex}<small>{process.gpu.name}</small></td><td>{process.pid}</td><td>{process.processName}</td><td>{process.usedMemoryMb} MB</td></tr>)}</tbody></table></div></section>
  </section>;
}

function GPUCard({ gpu }) {
  const memoryPct = gpu.memoryTotalMb ? Math.round((gpu.memoryUsedMb / gpu.memoryTotalMb) * 100) : 0;
  return <article className="telemetry-card"><div><strong>GPU {gpu.index}</strong><StatusBadge status={isGpuBusy(gpu) ? 'Busy' : 'Idle'} /></div><span>{gpu.name}</span><MetricLine label="SM" value={`${gpu.utilization || 0}%`} /><Meter value={gpu.utilization || 0} /><MetricLine label="Memory" value={`${gpu.memoryUsedMb || 0} / ${gpu.memoryTotalMb || 0} MB`} /><Meter value={memoryPct} /><MetricLine label="Power" value={`${gpu.powerWatts || 0} W`} /><MetricLine label="Temp" value={`${gpu.temperatureC || 0} C`} />{(gpu.processes || []).length > 0 && <small>{gpu.processes.length} process(es)</small>}</article>;
}

function MetricLine({ label, value }) {
  return <div className="metric-line"><span>{label}</span><strong>{value}</strong></div>;
}

function Meter({ value }) {
  return <div className="meter"><span style={{ width: `${Math.max(0, Math.min(100, value || 0))}%` }} /></div>;
}

function formatMemory(valueMb) {
  if (!valueMb) return '0 GB';
  return `${(valueMb / 1024).toFixed(1)} GB`;
}

function telemetryForTask(task, telemetryGpus) {
  const allocated = new Set(task.allocatedGpus || []);
  if (!allocated.size) return [];
  return telemetryGpus.filter((gpu) => allocated.has(gpu.id) || allocated.has(`${gpu.nodeName || 'A00'}-GPU-${gpu.index}`));
}

function taskListForJob(job) {
  if ((job.tasks || []).length) return job.tasks;
  return [{
    id: `${job.id}-master-0`,
    name: 'master-0',
    role: 'master',
    rank: 0,
    allocatedGpus: job.allocatedGpus || [],
    cpuCount: 4,
    memoryGb: 16,
  }];
}
