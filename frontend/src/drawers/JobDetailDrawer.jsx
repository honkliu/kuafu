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
      <section className="drawer-section job-output-detail"><h3>Output</h3><TaskOutput logs={detail.logs} /></section>
      <TaskSystemUsage tasks={detail.taskMetrics || []} />
      <section className="drawer-section detail-grid">
        <div><h3>Summary</h3><KeyValue label="Job ID" value={detail.job.id} /><KeyValue label="Status" value={<StatusBadge status={detail.job.status} />} /><KeyValue label="Queue" value={detail.job.queue} /><KeyValue label="Submitted" value={formatDate(detail.job.submittedAt)} /><KeyValue label="Started" value={formatDate(detail.job.startedAt)} /></div>
        <div><h3>Exact Docker launch</h3><CodeBlock value={renderJobDockerCommand(detail.job)} /></div>
      </section>
      <TaskPlan job={detail.job} />
      <section className="drawer-section"><h3>GPU usage</h3><div className="usage-grid">{detail.metrics.length === 0 ? <p className="muted-text">No GPU metrics for this job.</p> : detail.metrics.map((gpu) => <GpuUsageCard key={gpu.gpuId} gpu={gpu} />)}</div></section>
      <TaskHistory tasks={detail.taskMetrics || []} />
      <section className="drawer-section"><h3>Launcher spec</h3><CodeBlock value={launcherSpecToYaml(detail.job.launcherSpec || launcherSpecFromJob(detail.job))} /></section>
      <section className="drawer-section"><h3>Submitted spec</h3><KeyValue label="Image" value={detail.job.image || '-'} /><KeyValue label="Entrypoint" value={detail.job.entrypointScript || '-'} /><KeyValue label="Task script" value={detail.job.command} /><KeyValue label="Allocated GPUs" value={(detail.job.allocatedGpus || []).join(', ') || '-'} /><KeyValue label="Allocated nodes" value={nodesForGPUIds(detail.job.allocatedGpus || [], allGpus).join(', ') || '-'} /></section>
    </div>}
  </aside>;
}

function TaskOutput({ logs }) {
  const parsed = parseJobOutput(logs || []);
  return <div className="stack">{parsed.tasks.length ? <div className="task-output-grid">{parsed.tasks.map((task) => <article className="task-output-card" key={task.name}><div><strong>{task.name}</strong><StatusBadge status={task.status || 'Output'} /></div>{task.stdout.length > 0 && <div><span>stdout</span><CodeBlock value={task.stdout.join('\n')} /></div>}{task.stderr.length > 0 && <div><span>stderr</span><CodeBlock value={task.stderr.join('\n')} /></div>}</article>)}</div> : <CodeBlock value="No stdout or stderr yet." />}<details className="json-details"><summary>Diagnostics</summary><CodeBlock value={logs.length ? logs.join('\n') : 'No diagnostic logs yet.'} /></details></div>;
}

function parseJobOutput(logs) {
  const byTask = new Map();
  const taskFor = (name) => {
    if (!byTask.has(name)) byTask.set(name, { name, stdout: [], stderr: [], status: '' });
    return byTask.get(name);
  };
  for (const line of logs || []) {
    const stdout = line.match(/Task ([^ ]+) stdout: ?(.*)$/);
    if (stdout) {
      taskFor(stdout[1]).stdout.push(stdout[2]);
      continue;
    }
    const stderr = line.match(/Task ([^ ]+) stderr: ?(.*)$/);
    if (stderr) {
      taskFor(stderr[1]).stderr.push(stderr[2]);
      continue;
    }
    const complete = line.match(/Task ([^ ]+) completed successfully/);
    if (complete) taskFor(complete[1]).status = 'Completed';
    const failed = line.match(/Task ([^ ]+) failed/);
    if (failed) taskFor(failed[1]).status = 'Failed';
  }
  return { tasks: [...byTask.values()] };
}

function TaskSystemUsage({ tasks }) {
  return <section className="drawer-section task-system-usage-section">
    <div className="section-header compact-header"><div><h3>Task system usage</h3><p>CPU, memory, GPU SM, and GPU memory by task.</p></div></div>
    {tasks.length === 0 ? <p className="muted-text">No task-level metrics reported yet.</p> : <div className="task-usage-grid">{tasks.map((task) => <TaskUsageCard key={task.id || `${task.name}-${task.rank}`} task={task} />)}</div>}
  </section>;
}

function TaskPlan({ job }) {
  return <section className="drawer-section"><h3>Task plan</h3><div className="table-wrap"><table className="data-table compact-table"><thead><tr><th>Task</th><th>Role</th><th>Rank</th><th>GPU</th><th>CPU / Memory</th><th>Docker options</th><th>Script</th></tr></thead><tbody>{(job.tasks || []).length === 0 ? <tr><td colSpan="7">This job was normalized into a master task for metrics.</td></tr> : job.tasks.map((task) => <tr key={task.id}><td>{task.name}<small>{task.status}</small></td><td><StatusBadge status={task.role} /></td><td>{task.rank}</td><td>{(task.allocatedGpus || []).join(', ') || '-'}</td><td>{task.cpuCount || 0} CPU / {task.memoryGb || 0} GB</td><td className="truncate-cell">{(task.dockerOptions || []).join(' ') || '-'}</td><td className="truncate-cell script-cell">{task.command}</td></tr>)}</tbody></table></div><div className="docker-preview-grid">{(job.tasks || []).map((task) => <CodeBlock key={`${task.id}-docker`} value={renderDockerLaunchCommand(task, job)} />)}</div></section>;
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
  return firstGpu?.split('-GPU-')[0] || 'placement unavailable';
}

function renderDockerLaunchCommand(task, job) {
  const parts = ['docker run --rm', '--name', shellQuote(`kuafu-${task.name || job.id}`)];
  const options = task.dockerOptions || job.launcherSpec?.docker?.options || [];
  if (options.length) parts.push(options.join(' '));
  if (task.workingDirectory) parts.push('-w', shellQuote(task.workingDirectory));
  for (const item of task.env || []) {
    parts.push('-e', shellQuote(`${item.name}=${item.value}`));
  }
  parts.push(shellQuote(task.image || job.image || 'debian:bookworm-slim'), '/bin/bash', '-lc', shellQuote(['set -e', task.entrypointScript || job.entrypointScript, task.command || job.command || ''].filter((part) => String(part || '').trim()).join('\n')));
  return parts.join(' ');
}

function shellQuote(value) {
  const text = String(value || '');
  return `'${text.replaceAll("'", "'\\''")}'`;
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
    name: job.name,
    project: job.project || undefined,
    queue: job.queue,
    replicaPolicy: 'fixed',
    workingDirectory: job.tasks?.[0]?.workingDirectory || '/workspace',
    entrypointScript: job.entrypointScript || job.launcherSpec?.entrypointScript || job.tasks?.[0]?.entrypointScript || '',
    dependencies: [],
    docker: { image: job.image, options: job.tasks?.[0]?.dockerOptions || [] },
    env: job.sharedEnv || [],
    tasks: (job.taskTemplates || []).length ? job.taskTemplates : [{ name: 'master', role: 'master', replicas: 1, image: job.image, command: job.command, gpuCount: job.gpuCount, cpuCount: 4, memoryGb: 16 }],
  };
}

function launcherSpecToYaml(spec) {
  spec = normalizeLauncherSpec(spec);
  const env = spec.env || [];
  const dependencies = spec.dependencies || [];
  const tasks = spec.tasks || [];
  return [`name: ${spec.name || 'unnamed-job'}`, spec.project ? `project: ${spec.project}` : '', `queue: ${spec.queue || 'default'}`, `replicaPolicy: ${spec.replicaPolicy || 'fixed'}`, `workingDirectory: ${spec.workingDirectory || '/workspace'}`, spec.entrypointScript ? `entrypointScript: ${spec.entrypointScript}` : '', 'dependencies:', ...(dependencies.length ? dependencies.map((item) => `  - ${item}`) : ['  []']), 'docker:', `  image: ${spec.docker?.image || ''}`, `  options: [${(spec.docker?.options || []).join(', ')}]`, 'env:', ...(env.length ? env.map((item) => `  - ${item.name}: ${item.value || ''}`) : ['  []']), 'tasks:', ...tasks.flatMap((task) => [`  - name: ${task.name}`, `    role: ${task.role}`, `    replicas: ${task.replicas || 1}`, task.image ? `    image: ${task.image}` : '', `    command: ${task.command || ''}`, `    workingDirectory: ${task.workingDirectory || spec.workingDirectory || '/workspace'}`, `    dockerOptions: [${(task.dockerOptions || spec.docker?.options || []).join(', ')}]`, `    gpuCount: ${task.gpuCount || 0}`, `    cpuCount: ${task.cpuCount || 0}`, `    memoryGb: ${task.memoryGb || 0}`])].filter((line) => line !== '').join('\n');
}

function normalizeLauncherSpec(spec) {
  if (spec.spec || spec.metadata) {
    return {
      name: spec.name || spec.metadata?.name || 'unnamed-job',
      project: spec.project || spec.metadata?.project || undefined,
      queue: spec.queue || spec.spec?.queue || 'default',
      replicaPolicy: spec.replicaPolicy || spec.spec?.replicaPolicy || 'fixed',
      workingDirectory: spec.workingDirectory || spec.spec?.workingDirectory || '/workspace',
      entrypointScript: spec.entrypointScript || spec.spec?.entrypointScript || '',
      dependencies: spec.dependencies || spec.spec?.dependencies || [],
      docker: spec.docker?.image ? spec.docker : (spec.spec?.docker || {}),
      env: spec.env?.length ? spec.env : (spec.spec?.env || []),
      tasks: spec.tasks?.length ? spec.tasks : (spec.spec?.tasks || []),
    };
  }
  return spec;
}
