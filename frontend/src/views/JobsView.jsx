import { Search } from 'lucide-react';
import { useMemo, useState } from 'react';
import { CodeBlock, StatusBadge } from '../components/Primitives.jsx';
import { catalogTemplates } from '../data/productPlan.js';
import { formatDate } from '../utils/format.js';

const defaultImage = 'nvidia/cuda:12.0-runtime';
const defaultCommand = 'nvidia-smi';
const emptyJobForm = { name: '', queue: 'default', project: '', image: defaultImage, command: defaultCommand, useSameCommand: true, sharedEnv: '', taskTemplates: defaultDistributedTemplates(defaultCommand, defaultImage) };

export default function JobsView({ jobs, queues, reservedEnv = [], filters, setFilters, openJobDetail, submitJob, runJobAction }) {
  const [jobForm, setJobForm] = useState(emptyJobForm);
  const [preview, setPreview] = useState(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const taskTemplates = effectiveTaskTemplates(jobForm);
  const taskGpuTotal = totalTemplateGPUs(taskTemplates);
  const taskCount = totalTemplateCount(taskTemplates);
  const filtered = jobs.filter((job) => (`${job.id} ${job.name} ${job.command}`.toLowerCase().includes(filters.keyword.toLowerCase())) && (filters.status === 'all' || job.status === filters.status) && (filters.queue === 'all' || job.queue === filters.queue));
  const statuses = ['all', ...Array.from(new Set(jobs.map((job) => job.status)))];
  const previewSpec = useMemo(() => buildPreviewSpec(jobForm, preview), [jobForm, preview]);

  function onPreview(event) {
    event.preventDefault();
    setError('');
    setMessage('');
    setPreview({
      spec: toJobPayload(jobForm),
      command: renderSubmitCommand(jobForm),
      runtimeCommand: renderRuntimeCommand(jobForm),
      tasks: renderPreviewTasks(jobForm),
    });
  }

  async function onConfirmSubmit() {
    if (!preview) return;
    setError('');
    setMessage('');
    try {
      const job = await submitJob({ ...preview.spec, command: preview.spec.command, taskTemplates: preview.spec.taskTemplates, tasks: preview.tasks, sharedEnv: parseEnvLines(jobForm.sharedEnv) });
      setMessage(`Submitted ${job.name} to ${job.queue}`);
      setJobForm({ ...emptyJobForm, queue: job.queue });
      setPreview(null);
    } catch (err) {
      setError(err.message);
    }
  }

  async function onAction(job, action) {
    setError('');
    setMessage('');
    try {
      const updated = await runJobAction(job.id, action);
      setMessage(`${actionLabel(action)} ${updated.name}: ${updated.status}`);
    } catch (err) {
      setError(err.message);
    }
  }

  return <section className="panel stack">
    <div className="section-header"><div><p className="eyebrow">Workloads</p><h2>Jobs</h2><p>Every job submission is previewed as editable commands before execution.</p></div></div>
    {message && <div className="success-banner">{message}</div>}
    {error && <div className="error-banner">{error}</div>}
    <section className="workflow-shell">
      <div className="task-support-panel"><div><p className="eyebrow">Distributed task plan</p><h3>{taskCount} tasks · {taskGpuTotal} GPUs</h3><p>Every Kuafu job is submitted as a reviewed master/worker task plan.</p></div><div className="support-chips"><span>Master/worker</span><span>Reserved env</span><span>Reviewed commands</span><span>Task GPU map</span></div></div>
      <div className="job-workbench">
        <form className="inline-form-grid job-compose-grid" onSubmit={onPreview}>
          <label>Template<select value="" onChange={(event) => applyTemplate(event.target.value, setJobForm, setPreview)}><option value="">Choose template</option>{catalogTemplates.map((template) => <option key={template.id || template.name} value={template.id || template.name}>{template.name}</option>)}</select></label>
          <label>Name<input value={jobForm.name} onChange={(event) => updateForm(setJobForm, setPreview, 'name', event.target.value)} required /></label>
          <label>Queue<select value={jobForm.queue} onChange={(event) => updateForm(setJobForm, setPreview, 'queue', event.target.value)}>{queues.map((queue) => <option key={queue.name} value={queue.name}>{queue.name}</option>)}</select></label>
          <label>Project<input value={jobForm.project} onChange={(event) => updateForm(setJobForm, setPreview, 'project', event.target.value)} placeholder="optional" /></label>
          <label>Total GPUs<input value={taskGpuTotal} readOnly /></label>
          <label className="double-span">Image<input value={jobForm.image} onChange={(event) => updateForm(setJobForm, setPreview, 'image', event.target.value)} /></label>
          <label className="double-span">Common command<input value={jobForm.command} onChange={(event) => updateForm(setJobForm, setPreview, 'command', event.target.value)} required /></label>
          <label className="checkbox-label double-span"><input type="checkbox" checked={jobForm.useSameCommand} onChange={(event) => setSameCommand(setJobForm, setPreview, event.target.checked)} />Master and worker use the same command</label>
          <details className="advanced-fields double-span"><summary>Advanced environment</summary><label>Shared env<textarea rows="3" value={jobForm.sharedEnv} onChange={(event) => updateForm(setJobForm, setPreview, 'sharedEnv', event.target.value)} placeholder="MODEL_PATH=zai-org/GLM-5.2-FP8" /></label></details>
          <button className="primary-button form-submit-button">Preview task plan</button>
        </form>
        <PlanSummary form={jobForm} templates={taskTemplates} taskCount={taskCount} taskGpuTotal={taskGpuTotal} />
      </div>
    </section>
    <section className="command-preview-panel">
      <div className="section-header compact-header"><div><p className="eyebrow">Task templates</p><h3>Master / worker roles</h3><p>{jobForm.useSameCommand ? 'All roles inherit the common command.' : 'Customize per-role launch commands. Use $RANK and $TASK0_ADDRESS directly.'}</p></div><div className="toolbar-actions"><button className="ghost-button" onClick={() => addTaskTemplate(setJobForm, setPreview, 'worker')}>Add worker</button><button className="ghost-button" onClick={() => addTaskTemplate(setJobForm, setPreview, 'task')}>Add custom task</button><button className="ghost-button" onClick={() => setJobForm((current) => ({ ...current, taskTemplates: defaultDistributedTemplates(current.command, current.image) }))}>Reset</button></div></div>
      <ReservedEnvTable reservedEnv={reservedEnv} />
      <div className="task-template-grid">{jobForm.taskTemplates.map((template, index) => <TaskTemplateEditor key={index} template={template} commonCommand={jobForm.command} useSameCommand={jobForm.useSameCommand} canRemove={jobForm.taskTemplates.length > 1} update={(next) => updateTaskTemplate(setJobForm, setPreview, index, next)} remove={() => removeTaskTemplate(setJobForm, setPreview, index)} />)}</div>
    </section>
    {preview && <section className="command-preview-panel">
      <div className="section-header compact-header"><div><p className="eyebrow">Review before submit</p><h3>Reviewed task plan</h3><p>These concrete task commands are what Kuafu stores and schedules. Edit them before submitting.</p></div><button className="primary-button" onClick={onConfirmSubmit}>Submit reviewed task plan</button></div>
      <div className="review-command-line"><span>{preview.command}</span></div>
      <div className="node-command-grid">{preview.tasks.map((task, index) => <label key={task.id}>{task.name} · rank {task.rank}<textarea rows="4" value={task.command} onChange={(event) => setPreview((current) => ({ ...current, tasks: current.tasks.map((item, taskIndex) => taskIndex === index ? { ...item, command: event.target.value } : item) }))} /></label>)}</div>
      <details className="json-details"><summary>View submitted JSON</summary><CodeBlock value={JSON.stringify(previewSpec, null, 2)} /></details>
    </section>}
    <div className="filter-bar"><label className="search-box"><Search size={16} /><input placeholder="Search jobs, IDs, commands" value={filters.keyword} onChange={(event) => setFilters({ ...filters, keyword: event.target.value })} /></label><select value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>{statuses.map((status) => <option key={status}>{status}</option>)}</select><select value={filters.queue} onChange={(event) => setFilters({ ...filters, queue: event.target.value })}><option value="all">all queues</option>{queues.map((queue) => <option key={queue.name} value={queue.name}>{queue.name}</option>)}</select></div>
    <div className="table-wrap"><table className="data-table"><thead><tr><th>Job</th><th>Status</th><th>Queue</th><th>Image</th><th>GPU</th><th>Submitted</th><th>Actions</th></tr></thead><tbody>{filtered.map((job) => <tr key={job.id}><td><strong>{job.name}</strong><small>{job.id}</small></td><td><StatusBadge status={job.status} /></td><td>{job.queue}</td><td className="truncate-cell">{job.image || '-'}</td><td>{job.gpuCount}</td><td>{formatDate(job.submittedAt)}</td><td><div className="row-actions"><button className="ghost-button" onClick={() => openJobDetail(job.id)}>Details</button>{job.status === 'Running' && <button className="ghost-button" onClick={() => onAction(job, 'stop')}>Stop</button>}{job.status === 'Queued' && <button className="ghost-button danger" onClick={() => onAction(job, 'cancel')}>Cancel</button>}{['Stopped', 'Failed', 'Canceled', 'Completed'].includes(job.status) && <button className="ghost-button" onClick={() => onAction(job, 'start')}>Start</button>}{job.status !== 'Queued' && <button className="ghost-button" onClick={() => onAction(job, 'restart')}>Restart</button>}</div></td></tr>)}</tbody></table></div>
  </section>;
}

function PlanSummary({ form, templates, taskCount, taskGpuTotal }) {
  return <aside className="plan-summary-card"><div><p className="eyebrow">Plan summary</p><h3>{form.name || 'unnamed-job'}</h3></div><dl><div><dt>Queue</dt><dd>{form.queue}</dd></div><div><dt>Tasks</dt><dd>{taskCount}</dd></div><div><dt>GPUs</dt><dd>{taskGpuTotal}</dd></div><div><dt>Command mode</dt><dd>{form.useSameCommand ? 'shared' : 'per-role'}</dd></div></dl><div className="plan-role-list">{templates.map((template) => <span key={`${template.name}-${template.role}`}>{template.role}: {template.replicas} x {template.gpuCount} GPU</span>)}</div></aside>;
}

function applyTemplate(templateId, setJobForm, setPreview) {
  if (!templateId) return;
  const template = catalogTemplates.find((item) => (item.id || item.name) === templateId);
  if (!template) return;
  setPreview(null);
  setJobForm((current) => ({
    ...current,
    name: template.id || sanitizeName(template.name),
    image: template.image,
    command: template.command,
    taskTemplates: template.id?.startsWith('glm52') ? sglangDistributedTemplates(template.command, template.image) : defaultDistributedTemplates(template.command, template.image),
  }));
}

function TaskTemplateEditor({ template, commonCommand, useSameCommand, canRemove, update, remove }) {
  return <article className="task-template-card"><div className="task-template-header"><span><strong>{template.name}</strong><StatusBadge status={template.role} /></span>{canRemove && <button className="ghost-button danger" onClick={remove}>Remove</button>}</div><label>Name<input value={template.name} onChange={(event) => update({ ...template, name: sanitizeName(event.target.value) })} /></label><label>Role<input value={template.role} onChange={(event) => update({ ...template, role: sanitizeName(event.target.value) })} /></label><label>Replicas<input type="number" min="1" value={template.replicas} onChange={(event) => update({ ...template, replicas: Number(event.target.value) || 1 })} /></label><label>GPUs/task<input type="number" min="0" value={template.gpuCount} onChange={(event) => update({ ...template, gpuCount: Number(event.target.value) || 0 })} /></label><label>CPU/task<input type="number" min="0" value={template.cpuCount || 0} onChange={(event) => update({ ...template, cpuCount: Number(event.target.value) || 0 })} /></label><label>Memory GB<input type="number" min="0" value={template.memoryGb || 0} onChange={(event) => update({ ...template, memoryGb: Number(event.target.value) || 0 })} /></label>{useSameCommand ? <div className="full-span inherited-command"><span>Command inherited from common command</span><code>{commonCommand}</code></div> : <label className="full-span">Command<textarea rows="5" value={template.command} onChange={(event) => update({ ...template, command: event.target.value })} /></label>}</article>;
}

function addTaskTemplate(setJobForm, setPreview, role) {
  setPreview(null);
  setJobForm((current) => {
    const index = current.taskTemplates.length;
    const command = current.taskTemplates[0]?.command || defaultDistributedTemplates()[0].command;
    return { ...current, taskTemplates: [...current.taskTemplates, { name: `${role}-${index}`, role, replicas: 1, image: current.image, command, gpuCount: 1, cpuCount: 8, memoryGb: 64 }] };
  });
}

function removeTaskTemplate(setJobForm, setPreview, index) {
  setPreview(null);
  setJobForm((current) => ({ ...current, taskTemplates: current.taskTemplates.filter((_, i) => i !== index) }));
}

function ReservedEnvTable({ reservedEnv }) {
  return <div className="reserved-env-strip"><strong>Kuafu system reserved env</strong><div>{reservedEnv.map((item) => <span key={item.name} title={item.description}>{`$${item.name}${item.value ? `=${item.value}` : ''}`}</span>)}</div></div>;
}

function setSameCommand(setJobForm, setPreview, checked) {
  setPreview(null);
  setJobForm((current) => ({ ...current, useSameCommand: checked }));
}

function updateTaskTemplate(setJobForm, setPreview, index, next) {
  setPreview(null);
  setJobForm((current) => ({ ...current, taskTemplates: current.taskTemplates.map((template, i) => i === index ? next : template) }));
}

function updateForm(setJobForm, setPreview, field, value) {
  setPreview(null);
  setJobForm((current) => ({ ...current, [field]: value }));
}

function toJobPayload(form) {
  const templates = effectiveTaskTemplates(form);
  return {
    name: form.name,
    project: form.project || undefined,
    queue: form.queue,
    command: form.command,
    image: form.image,
    gpuCount: totalTemplateGPUs(templates),
    sharedEnv: parseEnvLines(form.sharedEnv),
    taskTemplates: templates,
  };
}

function renderSubmitCommand(form) {
  return `kuafu jobs submit --name ${shellQuote(form.name)} --queue ${shellQuote(form.queue)} --gpus ${totalTemplateGPUs(effectiveTaskTemplates(form))} --task-spec <reviewed JSON below>`;
}

function renderRuntimeCommand(form) {
  return 'Distributed job: review and edit the concrete master/worker task commands below. Kuafu submits them as a task plan, not as one docker run.';
}

function buildPreviewSpec(form, preview) {
  const spec = toJobPayload(form);
  if (!preview) return spec;
  return { ...spec, previewCommand: preview.command, runtimeCommand: preview.runtimeCommand, renderedTasks: preview.tasks };
}

function effectiveTaskTemplates(form) {
  if (!form.useSameCommand) return form.taskTemplates;
  return form.taskTemplates.map((template) => ({ ...template, command: form.command, image: template.image || form.image }));
}

function totalTemplateGPUs(templates = []) {
  return templates.reduce((sum, template) => sum + ((Number(template.replicas) || 1) * (Number(template.gpuCount) || 0)), 0) || 1;
}

function totalTemplateCount(templates = []) {
  return templates.reduce((sum, template) => sum + (Number(template.replicas) || 1), 0) || 1;
}

function defaultDistributedTemplates(command = defaultCommand, image = defaultImage) {
  return [
    { name: 'master', role: 'master', replicas: 1, image, command, gpuCount: 1, cpuCount: 4, memoryGb: 16 },
    { name: 'worker', role: 'worker', replicas: 1, image, command, gpuCount: 0, cpuCount: 2, memoryGb: 8 },
  ];
}

function sglangDistributedTemplates(command, image) {
  const common = command.includes('--nnodes') ? command : `${command} --nnodes $WORLD_SIZE --node-rank $RANK --dist-init-addr $TASK0_ADDRESS:$MASTER_PORT`;
  return [
    { name: 'master', role: 'master', replicas: 1, image, command: common, gpuCount: 8, cpuCount: 32, memoryGb: 256 },
    { name: 'worker', role: 'worker', replicas: 1, image, command: common, gpuCount: 8, cpuCount: 32, memoryGb: 256 },
  ];
}

function parseEnvLines(text) {
  return String(text || '').split('\n').map((line) => line.trim()).filter(Boolean).map((line) => {
    const split = line.indexOf('=');
    if (split < 0) return { name: line, value: '' };
    return { name: line.slice(0, split).trim(), value: line.slice(split + 1).trim() };
  }).filter((item) => item.name);
}

function renderPreviewTasks(form) {
  const sharedEnv = parseEnvLines(form.sharedEnv);
  const templates = effectiveTaskTemplates(form) || [];
  const worldSize = templates.reduce((sum, template) => sum + (Number(template.replicas) || 1), 0) || 1;
  const addresses = Array.from({ length: worldSize }, (_, index) => `${form.name || 'job'}-task-${index}`);
  let rank = 0;
  return templates.flatMap((template) => Array.from({ length: Number(template.replicas) || 1 }, (_, ordinal) => {
    const taskRank = rank++;
    const env = mergeEnv(sharedEnv, [{ name: 'RANK', value: String(taskRank) }, { name: 'TASK_RANK', value: String(taskRank) }, { name: 'TASK_INDEX', value: String(taskRank) }, { name: 'TASK_ORDINAL', value: String(ordinal) }, { name: 'TASK_ROLE', value: template.role }, { name: 'WORLD_SIZE', value: String(worldSize) }, { name: 'TASK0_ADDRESS', value: addresses[0] }, { name: 'MASTER_ADDRESS', value: addresses[0] }, { name: 'MASTER_PORT', value: '20000' }, { name: 'KUAFU_TASK_ADDRESS', value: addresses[taskRank] }]);
    return { id: `${template.name}-${taskRank}`, name: `${template.name}-${ordinal}`, role: template.role, rank: taskRank, ordinal, image: template.image || form.image, gpuCount: template.gpuCount, cpuCount: template.cpuCount, memoryGb: template.memoryGb, command: renderEnv(template.command, env), env };
  }));
}

function mergeEnv(...groups) {
  const merged = [];
  const indexByName = new Map();
  for (const group of groups) {
    for (const item of group) {
      if (indexByName.has(item.name)) {
        merged[indexByName.get(item.name)] = item;
      } else {
        indexByName.set(item.name, merged.length);
        merged.push(item);
      }
    }
  }
  return merged;
}

function renderEnv(command, env) {
  for (const item of env) {
    command = command.replaceAll(`$${item.name}`, item.value).replaceAll(`\${${item.name}}`, item.value);
  }
  return command;
}

function sanitizeName(value) {
  return String(value).toLowerCase().replace(/[^a-z0-9_.-]+/g, '-').replace(/^-+|-+$/g, '') || 'job';
}

function shellQuote(value) {
  const text = String(value || '');
  return `'${text.replaceAll("'", "'\\''")}'`;
}

function actionLabel(action) {
  return { start: 'Started', stop: 'Stopped', restart: 'Restarted', cancel: 'Canceled' }[action] || action;
}
