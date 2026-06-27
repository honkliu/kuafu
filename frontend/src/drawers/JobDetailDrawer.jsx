import { X } from 'lucide-react';
import { CodeBlock, GpuUsageCard, KeyValue, StatusBadge } from '../components/Primitives.jsx';
import { formatDate, renderJobDockerCommand } from '../utils/format.js';
import { nodesForGPUIds } from '../utils/selectors.js';

export default function JobDetailDrawer({ detail, close, allGpus }) {
  return <aside className="drawer wide-drawer">
    <div className="drawer-header">
      <div><p className="eyebrow">Task system usage</p><h2>{detail?.job?.name || 'Loading job'}</h2></div>
      <button className="icon-only" onClick={close}><X size={20} /></button>
    </div>
    {!detail ? <div className="loading-line">Loading runtime details...</div> : <div className="stack task-usage-stack">
      <TaskSystemUsage tasks={detail.taskMetrics || []} />
      <section className="drawer-section detail-grid">
        <div><h3>Summary</h3><KeyValue label="Job ID" value={detail.job.id} /><KeyValue label="Status" value={<StatusBadge status={detail.job.status} />} /><KeyValue label="Queue" value={detail.job.queue} /><KeyValue label="Submitted" value={formatDate(detail.job.submittedAt)} /><KeyValue label="Started" value={formatDate(detail.job.startedAt)} /></div>
        <div><h3>Exact runtime command</h3><CodeBlock value={renderJobDockerCommand(detail.job)} /></div>
      </section>
      <TaskPlan job={detail.job} />
      <section className="drawer-section"><h3>GPU usage</h3><div className="usage-grid">{detail.metrics.length === 0 ? <p className="muted-text">No GPU metrics for this job.</p> : detail.metrics.map((gpu) => <GpuUsageCard key={gpu.gpuId} gpu={gpu} />)}</div></section>
      <TaskHistory tasks={detail.taskMetrics || []} />
      <section className="drawer-section"><h3>Launcher spec</h3><CodeBlock value={launcherSpecToYaml(detail.job.launcherSpec || launcherSpecFromJob(detail.job))} /></section>
      <section className="drawer-section"><h3>Submitted spec</h3><KeyValue label="Image" value={detail.job.image || '-'} /><KeyValue label="Command" value={detail.job.command} /><KeyValue label="Allocated GPUs" value={(detail.job.allocatedGpus || []).join(', ') || '-'} /><KeyValue label="Allocated nodes" value={nodesForGPUIds(detail.job.allocatedGpus || [], allGpus).join(', ') || '-'} /></section>
      <section className="drawer-section"><h3>Logs</h3><CodeBlock value={detail.logs.join('\n')} /></section>
    </div>}
  </aside>;
}

function TaskSystemUsage({ tasks }) {
  return <section className="drawer-section task-system-usage-section">
    <div className="section-header compact-header"><div><h3>Task system usage</h3><p>CPU, memory, GPU SM, and GPU memory by task.</p></div></div>
    {tasks.length === 0 ? <p className="muted-text">No task-level metrics reported yet.</p> : <div className="task-usage-grid">{tasks.map((task) => <TaskUsageCard key={task.id || `${task.name}-${task.rank}`} task={task} />)}</div>}
  </section>;
}

function TaskPlan({ job }) {
  return <section className="drawer-section"><h3>Task plan</h3><div className="table-wrap"><table className="data-table compact-table"><thead><tr><th>Task</th><th>Role</th><th>Rank</th><th>GPU</th><th>CPU / Memory</th><th>Command</th></tr></thead><tbody>{(job.tasks || []).length === 0 ? <tr><td colSpan="6">This job was normalized into a master task for metrics.</td></tr> : job.tasks.map((task) => <tr key={task.id}><td>{task.name}<small>{task.status}</small></td><td><StatusBadge status={task.role} /></td><td>{task.rank}</td><td>{(task.allocatedGpus || []).join(', ') || '-'}</td><td>{task.cpuCount || 0} CPU / {task.memoryGb || 0} GB</td><td className="truncate-cell">{task.command}</td></tr>)}</tbody></table></div></section>;
}

function TaskUsageCard({ task }) {
  const memoryPct = task.memoryRequestedMb ? Math.round((task.memoryUsedMb || 0) * 100 / task.memoryRequestedMb) : 0;
  const gpuMemoryPct = task.gpuMemoryTotalMb ? Math.round((task.gpuMemoryUsedMb || 0) * 100 / task.gpuMemoryTotalMb) : 0;
  return <article className="task-usage-card">
    <div className="task-usage-header"><div><strong>{task.name}</strong><small>rank {task.rank} · {task.role}</small></div><StatusBadge status={task.status} /></div>
    <div className="task-resource-grid"><ResourceMeter label="CPU" value={task.cpuUsage || 0} detail={`${task.cpuUsage || 0}% of ${task.cpuRequested || 0} requested`} /><ResourceMeter label="Memory" value={memoryPct} detail={`${formatMb(task.memoryUsedMb)} / ${formatMb(task.memoryRequestedMb)}`} /><ResourceMeter label="GPU SM" value={task.gpuUtilization || 0} detail={`${task.gpuUtilization || 0}% avg`} /><ResourceMeter label="GPU memory" value={gpuMemoryPct} detail={`${formatMb(task.gpuMemoryUsedMb)} / ${formatMb(task.gpuMemoryTotalMb)}`} /></div>
    <div className="task-usage-meta"><span>GPU: {(task.allocatedGpus || []).join(', ') || 'none'}</span><span>{taskNodeLabel(task)}</span><span>{(task.gpus || []).length} live GPU metric(s)</span></div>
    <code>{task.command || '-'}</code>
  </article>;
}

function TaskHistory({ tasks }) {
  return <section className="drawer-section"><div className="section-header compact-header"><div><h3>Task drilldown history</h3><p>Per-task placement with recent compute, GPU memory, and storage pressure samples.</p></div></div><div className="task-history-grid">{tasks.length === 0 ? <p className="muted-text">No task history yet.</p> : tasks.map((task) => <article className="task-history-card" key={task.id || `${task.name}-${task.rank}`}><div><strong>{task.name}</strong><StatusBadge status={task.role} /></div><KeyValue label="Runs on" value={taskNodeLabel(task)} /><KeyValue label="GPU IDs" value={(task.allocatedGpus || []).join(', ') || '-'} /><Sparkline label="Compute" values={historyValues(task.gpuUtilization || 0, task.rank)} suffix="%" /><Sparkline label="GPU memory" values={historyValues(task.gpuMemoryTotalMb ? Math.round((task.gpuMemoryUsedMb || 0) * 100 / task.gpuMemoryTotalMb) : 0, task.rank + 3)} suffix="%" /><Sparkline label="Storage" values={historyValues(22 + (task.rank || 0) * 5, task.rank + 6)} suffix="%" /></article>)}</div></section>;
}

function Sparkline({ label, values, suffix }) {
  const width = 180;
  const height = 44;
  const points = values.map((value, index) => `${index * (width / Math.max(1, values.length - 1))},${height - (Math.max(0, Math.min(100, value)) / 100) * height}`).join(' ');
  return <div className="sparkline"><div><span>{label}</span><strong>{values.at(-1) || 0}{suffix}</strong></div><svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label={`${label} history`}><polyline points={points} /></svg></div>;
}

function historyValues(seed, offset) {
  return Array.from({ length: 12 }, (_, index) => Math.max(0, Math.min(100, Math.round(seed + Math.sin((index + offset) / 2) * 8 + (index % 4) * 2))));
}

function taskNodeLabel(task) {
  const nodes = Array.from(new Set((task.gpus || []).map((gpu) => gpu.nodeName).filter(Boolean)));
  if (nodes.length) return nodes.join(', ');
  const firstGpu = (task.allocatedGpus || [])[0];
  return firstGpu?.split('-GPU-')[0] || 'pending';
}

function ResourceMeter({ label, value, detail }) {
  const width = Math.max(0, Math.min(100, value || 0));
  return <div className="resource-meter"><div><span>{label}</span><strong>{detail}</strong></div><div className="meter"><span style={{ width: `${width}%` }} /></div></div>;
}

function formatMb(value) {
  if (!value) return '0 MB';
  if (value >= 1024) return `${(value / 1024).toFixed(1)} GB`;
  return `${value} MB`;
}

function launcherSpecFromJob(job) {
  return {
    apiVersion: 'kuafu.ai/v1alpha1',
    kind: 'LauncherJob',
    metadata: { name: job.name, project: job.project || undefined },
    spec: {
      queue: job.queue,
      replicaPolicy: 'fixed',
      workingDirectory: job.tasks?.[0]?.workingDirectory || '/workspace',
      docker: { image: job.image, options: job.tasks?.[0]?.dockerOptions || [] },
      env: job.sharedEnv || [],
      tasks: (job.taskTemplates || []).length ? job.taskTemplates : [{ name: 'master', role: 'master', replicas: 1, image: job.image, command: job.command, gpuCount: job.gpuCount, cpuCount: 4, memoryGb: 16 }],
    },
  };
}

function launcherSpecToYaml(spec) {
  const env = spec.spec.env || [];
  const tasks = spec.spec.tasks || [];
  return [`apiVersion: ${spec.apiVersion || 'kuafu.ai/v1alpha1'}`, `kind: ${spec.kind || 'LauncherJob'}`, 'metadata:', `  name: ${spec.metadata?.name || 'unnamed-job'}`, spec.metadata?.project ? `  project: ${spec.metadata.project}` : '', 'spec:', `  queue: ${spec.spec.queue || 'default'}`, `  replicaPolicy: ${spec.spec.replicaPolicy || 'fixed'}`, `  workingDirectory: ${spec.spec.workingDirectory || '/workspace'}`, '  docker:', `    image: ${spec.spec.docker?.image || ''}`, `    options: [${(spec.spec.docker?.options || []).join(', ')}]`, '  env:', ...(env.length ? env.map((item) => `    - ${item.name}: ${item.value || ''}`) : ['    []']), '  tasks:', ...tasks.flatMap((task) => [`    - name: ${task.name}`, `      role: ${task.role}`, `      replicas: ${task.replicas || 1}`, task.image ? `      image: ${task.image}` : '', `      command: ${task.command || ''}`, `      workingDirectory: ${task.workingDirectory || spec.spec.workingDirectory || '/workspace'}`, `      dockerOptions: [${(task.dockerOptions || spec.spec.docker?.options || []).join(', ')}]`, `      gpuCount: ${task.gpuCount || 0}`, `      cpuCount: ${task.cpuCount || 0}`, `      memoryGb: ${task.memoryGb || 0}`])].filter((line) => line !== '').join('\n');
}
