import { Search } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { CodeBlock, StatusBadge } from '../components/Primitives.jsx';
import { catalogTemplates } from '../data/productPlan.js';
import { formatDate } from '../utils/format.js';

const defaultImage = 'debian:bookworm-slim';
const defaultEntrypointScript = 'echo "prepare rank=$RANK world=$WORLD_SIZE task=$KUAFU_TASK_ADDRESS"';
const defaultTaskScript = 'echo "run task rank=$TASK_RANK"';
const emptyJobForm = { name: '', queue: 'default', project: '', image: defaultImage, entrypointScript: defaultEntrypointScript, command: defaultTaskScript, useSameCommand: true, replicaPolicy: 'fixed', workingDirectory: '/workspace', dockerOptions: '--network=host\n--ipc=host', sharedEnv: '', taskTemplates: defaultDistributedTemplates(defaultTaskScript, defaultImage) };

export default function JobsView({ jobs, queues, reservedEnv = [], filters, setFilters, openJobDetail, loadJobDetail, submitJob, runJobAction }) {
  const [jobForm, setJobForm] = useState(emptyJobForm);
  const [preview, setPreview] = useState(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const [shareSpec, setShareSpec] = useState(null);
  const [selectedOutputJobId, setSelectedOutputJobId] = useState('');
  const [outputDetail, setOutputDetail] = useState(null);
  const [outputLoading, setOutputLoading] = useState(false);
  const outputRef = useRef(null);
  const taskTemplates = effectiveTaskTemplates(jobForm);
  const taskGpuTotal = totalTemplateGPUs(taskTemplates);
  const taskCount = totalTemplateCount(taskTemplates);
  const filtered = jobs.filter((job) => (`${job.id} ${job.name} ${job.command}`.toLowerCase().includes(filters.keyword.toLowerCase())) && (filters.status === 'all' || job.status === filters.status) && (filters.queue === 'all' || job.queue === filters.queue));
  const statuses = ['all', ...Array.from(new Set(jobs.map((job) => job.status)))];
  const selectedOutputJob = outputDetail?.job || jobs.find((job) => job.id === selectedOutputJobId) || jobs[0] || null;
  const previewSpec = useMemo(() => buildPreviewSpec(jobForm, preview), [jobForm, preview]);
  const launcherYaml = useMemo(() => launcherSpecToYaml(toLauncherSpec(jobForm)), [jobForm]);
  const [launcherYamlDraft, setLauncherYamlDraft] = useState(launcherYaml);

  useEffect(() => {
    setLauncherYamlDraft(launcherYaml);
  }, [launcherYaml]);

  useEffect(() => {
    if (!selectedOutputJobId) return undefined;
    let canceled = false;
    let timer = null;
    async function refreshOutput() {
      setOutputLoading(true);
      try {
        const detail = await loadJobDetail(selectedOutputJobId);
        if (canceled) return;
        setOutputDetail(detail);
        if (['Queued', 'Running'].includes(detail.job.status)) {
          timer = window.setTimeout(refreshOutput, 2000);
        }
      } catch (err) {
        if (!canceled) setError(err.message);
      } finally {
        if (!canceled) setOutputLoading(false);
      }
    }
    refreshOutput();
    return () => {
      canceled = true;
      if (timer) window.clearTimeout(timer);
    };
  }, [selectedOutputJobId, loadJobDetail]);

  function showOutput(jobId) {
    setOutputDetail(null);
    setSelectedOutputJobId(jobId);
    window.requestAnimationFrame(() => outputRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' }));
  }

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
      setOutputDetail({ job, logs: job.logs || [], metrics: [], taskMetrics: [] });
      showOutput(job.id);
      setJobForm({ ...emptyJobForm, queue: job.queue });
      setPreview(null);
    } catch (err) {
      setError(err.message);
    }
  }

  function importLauncherYaml() {
    setError('');
    try {
      setJobForm(formFromLauncherSpec(parseLauncherYaml(launcherYamlDraft)));
      setPreview(null);
      setMessage('Launcher YAML imported into the submit form.');
    } catch (err) {
      setError(err.message);
    }
  }

  function cloneJob(job) {
    setJobForm(formFromLauncherSpec(job.launcherSpec || launcherSpecFromJob(job)));
    setPreview(null);
    setMessage(`Cloned ${job.name} into the launcher form.`);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  async function resubmitJob(job) {
    setError('');
    setMessage('');
    try {
      const next = await submitJob({ launcherSpec: renameLauncherSpec(job.launcherSpec || launcherSpecFromJob(job), `${job.name}-resubmit`) });
      setMessage(`Resubmitted ${job.name} as ${next.name}`);
    } catch (err) {
      setError(err.message);
    }
  }

  function shareJob(job) {
    const yaml = launcherSpecToYaml(job.launcherSpec || launcherSpecFromJob(job));
    setShareSpec({ jobName: job.name, yaml, copied: false });
    setMessage(`Share spec opened for ${job.name}.`);
  }

  async function copyShareSpec() {
    if (!shareSpec) return;
    try {
      await navigator.clipboard?.writeText(shareSpec.yaml);
      setShareSpec({ ...shareSpec, copied: true });
    } catch (err) {
      setError(`Copy failed: ${err.message}`);
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
    <div className="section-header"><div><p className="eyebrow">Workloads</p><h2>Jobs</h2><p>Every job submission is previewed as editable scripts before execution.</p></div></div>
    {message && <div className="success-banner">{message}</div>}
    {error && <div className="error-banner">{error}</div>}
    <section className="job-launcher-shell">
      <form className="job-intent-form" onSubmit={onPreview}>
        <div className="launcher-step-header"><span>1</span><div><p className="eyebrow">Configure once</p><h3>Entrypoint script and runtime</h3></div></div>
        <div className="launcher-form-grid">
          <label>Template<select value="" onChange={(event) => applyTemplate(event.target.value, setJobForm, setPreview)}><option value="">Choose template</option>{catalogTemplates.map((template) => <option key={template.id || template.name} value={template.id || template.name}>{template.name}</option>)}</select></label>
          <label>Name<input value={jobForm.name} onChange={(event) => updateForm(setJobForm, setPreview, 'name', event.target.value)} required /></label>
          <label>Queue<select value={jobForm.queue} onChange={(event) => updateForm(setJobForm, setPreview, 'queue', event.target.value)}>{queues.map((queue) => <option key={queue.name} value={queue.name}>{queue.name}</option>)}</select></label>
          <label>Project<input value={jobForm.project} onChange={(event) => updateForm(setJobForm, setPreview, 'project', event.target.value)} placeholder="optional" /></label>
          <label className="double-span">Image<input value={jobForm.image} onChange={(event) => updateForm(setJobForm, setPreview, 'image', event.target.value)} /></label>
          <label>Working directory<input value={jobForm.workingDirectory} onChange={(event) => updateForm(setJobForm, setPreview, 'workingDirectory', event.target.value)} /></label>
          <label>Replica policy<select value={jobForm.replicaPolicy} onChange={(event) => updateForm(setJobForm, setPreview, 'replicaPolicy', event.target.value)}><option value="fixed">fixed</option><option value="elastic">elastic</option></select></label>
          <label className="script-editor-label full-span">Entrypoint script (runs before each task)<textarea rows="6" value={jobForm.entrypointScript} onChange={(event) => updateForm(setJobForm, setPreview, 'entrypointScript', event.target.value)} /></label>
          <label className="script-editor-label full-span">Task script (multi-line)<textarea rows="10" value={jobForm.command} onChange={(event) => updateForm(setJobForm, setPreview, 'command', event.target.value)} required /></label>
          <label className="double-span">Docker run options<textarea rows="4" value={jobForm.dockerOptions} onChange={(event) => updateForm(setJobForm, setPreview, 'dockerOptions', event.target.value)} placeholder="--network=host&#10;--ipc=host" /></label>
          <label className="checkbox-label"><input type="checkbox" checked={jobForm.useSameCommand} onChange={(event) => setSameCommand(setJobForm, setPreview, event.target.checked)} />Use this script for every role</label>
          <details className="advanced-fields"><summary>Environment</summary><label>Shared env<textarea rows="4" value={jobForm.sharedEnv} onChange={(event) => updateForm(setJobForm, setPreview, 'sharedEnv', event.target.value)} placeholder="MODEL_PATH=zai-org/GLM-5.2-FP8" /></label><p className="muted-text">Environment variables are injected before the entrypoint and task scripts run.</p></details>
        </div>
        <button className="primary-button form-submit-button">Preview task plan</button>
      </form>
      <aside className="launcher-review-rail">
        <PlanSummary form={jobForm} templates={taskTemplates} taskCount={taskCount} taskGpuTotal={taskGpuTotal} />
        <ReservedEnvTable reservedEnv={reservedEnv} />
      </aside>
    </section>
    <section className="task-layout-panel command-preview-panel">
      <div className="section-header compact-header"><div><p className="eyebrow">Task layout</p><h3>Roles and resources</h3><p>Scripts, image, Docker options, and env inherit from the job unless a role override is opened.</p></div><div className="toolbar-actions"><button className="ghost-button" onClick={() => addTaskTemplate(setJobForm, setPreview, 'worker')}>Add worker</button><button className="ghost-button" onClick={() => addTaskTemplate(setJobForm, setPreview, 'task')}>Add custom task</button><button className="ghost-button" onClick={() => setJobForm((current) => ({ ...current, taskTemplates: defaultDistributedTemplates(current.command, current.image), useSameCommand: true }))}>Reset</button></div></div>
      <div className="task-template-grid">{jobForm.taskTemplates.map((template, index) => <TaskTemplateEditor key={index} template={template} commonCommand={jobForm.command} commonImage={jobForm.image} commonDockerOptions={jobForm.dockerOptions} commonWorkingDirectory={jobForm.workingDirectory} useSameCommand={jobForm.useSameCommand} canRemove={jobForm.taskTemplates.length > 1} update={(next) => updateTaskTemplate(setJobForm, setPreview, index, next)} remove={() => removeTaskTemplate(setJobForm, setPreview, index)} />)}</div>
    </section>
    <details className="command-preview-panel launcher-spec-panel"><summary>Launcher YAML</summary><div className="section-header compact-header"><div><p className="eyebrow">Launcher protocol</p><h3>Portable job spec</h3></div><button className="ghost-button" onClick={importLauncherYaml}>Import YAML into form</button></div><textarea className="launcher-yaml-editor" rows="16" value={launcherYamlDraft} onChange={(event) => setLauncherYamlDraft(event.target.value)} /></details>
    {preview && <section className="command-preview-panel review-submit-panel">
      <div className="section-header compact-header"><div><p className="eyebrow">Review before submit</p><h3>Concrete task plan</h3><p>These are the exact task scripts and Docker launch lines Kuafu will store and execute.</p></div><button className="primary-button" onClick={onConfirmSubmit}>Submit reviewed plan</button></div>
      <div className="review-command-line"><span>{preview.command}</span></div>
      <div className="node-command-grid review-task-grid">{preview.tasks.map((task, index) => <label key={task.id}>{task.name} · rank {task.rank}<textarea rows="7" value={task.command} onChange={(event) => setPreview((current) => ({ ...current, tasks: current.tasks.map((item, taskIndex) => taskIndex === index ? { ...item, command: event.target.value } : item) }))} /></label>)}</div>
      <details className="json-details" open><summary>View Docker launch lines</summary><div className="docker-preview-grid">{preview.tasks.map((task) => <CodeBlock key={`${task.id}-docker`} value={renderDockerLaunchCommand(task)} />)}</div></details>
      <details className="json-details" open><summary>View submitted launcher YAML</summary><CodeBlock value={launcherSpecToYaml(previewSpec.launcherSpec || toLauncherSpec(jobForm))} /></details>
      <details className="json-details"><summary>View submitted JSON</summary><CodeBlock value={JSON.stringify(previewSpec, null, 2)} /></details>
    </section>}
    <JobOutputPanel refNode={outputRef} job={selectedOutputJob} detail={outputDetail} loading={outputLoading} jobs={jobs} selectedJobId={selectedOutputJob?.id || ''} showOutput={showOutput} openJobDetail={openJobDetail} />
    <div className="filter-bar"><label className="search-box"><Search size={16} /><input placeholder="Search jobs, IDs, scripts" value={filters.keyword} onChange={(event) => setFilters({ ...filters, keyword: event.target.value })} /></label><select value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>{statuses.map((status) => <option key={status}>{status}</option>)}</select><select value={filters.queue} onChange={(event) => setFilters({ ...filters, queue: event.target.value })}><option value="all">all queues</option>{queues.map((queue) => <option key={queue.name} value={queue.name}>{queue.name}</option>)}</select></div>
    <div className="table-wrap"><table className="data-table"><thead><tr><th>Job</th><th>Status</th><th>Queue</th><th>Image</th><th>GPU</th><th>Submitted</th><th>Actions</th></tr></thead><tbody>{filtered.map((job) => <tr key={job.id} className={selectedOutputJob?.id === job.id ? 'selected-row' : ''}><td><strong>{job.name}</strong><small>{job.id}</small></td><td><StatusBadge status={job.status} /></td><td>{job.queue}</td><td className="truncate-cell">{job.image || '-'}</td><td>{job.gpuCount}</td><td>{formatDate(job.submittedAt)}</td><td><div className="row-actions"><button className="primary-button output-action-button" onClick={() => showOutput(job.id)}>Output</button><button className="ghost-button" onClick={() => openJobDetail(job.id)}>Details</button><button className="ghost-button" onClick={() => cloneJob(job)}>Clone</button><button className="ghost-button" onClick={() => resubmitJob(job)}>Resubmit</button><button className="ghost-button" onClick={() => shareJob(job)}>Share YAML</button>{job.status === 'Running' && <button className="ghost-button" onClick={() => onAction(job, 'stop')}>Stop</button>}{job.status === 'Queued' && <button className="ghost-button danger" onClick={() => onAction(job, 'cancel')}>Cancel</button>}{['Stopped', 'Failed', 'Canceled', 'Completed'].includes(job.status) && <button className="ghost-button" onClick={() => onAction(job, 'start')}>Start</button>}{job.status !== 'Queued' && <button className="ghost-button" onClick={() => onAction(job, 'restart')}>Restart</button>}</div></td></tr>)}</tbody></table></div>
    {shareSpec && <ShareSpecDialog shareSpec={shareSpec} copyShareSpec={copyShareSpec} close={() => setShareSpec(null)} />}
  </section>;
}

function JobOutputPanel({ refNode, job, detail, loading, jobs, selectedJobId, showOutput, openJobDetail }) {
  const logs = detail?.logs || job?.logs || [];
  const parsed = parseJobOutput(logs);
  return <section className="job-output-panel" ref={refNode}>
    <div className="section-header compact-header"><div><p className="eyebrow">Output</p><h3>{job?.name || 'No job selected'}</h3><p>{job ? `${job.status}${loading ? ' · refreshing' : ''} · ${logs.length} log lines` : 'Submit or select a job to inspect stdout and stderr.'}</p></div><div className="toolbar-actions"><select value={selectedJobId} onChange={(event) => showOutput(event.target.value)}><option value="">Latest job</option>{jobs.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select>{job && <button className="ghost-button" onClick={() => showOutput(job.id)}>Refresh output</button>}{job && <button className="ghost-button" onClick={() => openJobDetail(job.id)}>Open details</button>}</div></div>
    {parsed.tasks.length ? <div className="task-output-grid">{parsed.tasks.map((task) => <article className="task-output-card" key={task.name}><div><strong>{task.name}</strong><StatusBadge status={task.status || 'Output'} /></div>{task.stdout.length > 0 && <div><span>stdout</span><CodeBlock value={task.stdout.join('\n')} /></div>}{task.stderr.length > 0 && <div><span>stderr</span><CodeBlock value={task.stderr.join('\n')} /></div>}</article>)}</div> : <CodeBlock value="No stdout or stderr yet." />}
    <details className="json-details"><summary>Diagnostics</summary><CodeBlock value={logs.length ? logs.join('\n') : 'No diagnostic logs yet.'} /></details>
  </section>;
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

function ShareSpecDialog({ shareSpec, copyShareSpec, close }) {
  return <div className="modal-backdrop" role="presentation">
    <section className="share-spec-dialog" role="dialog" aria-modal="true" aria-labelledby="share-spec-title">
      <div className="section-header compact-header"><div><p className="eyebrow">Share launcher spec</p><h3 id="share-spec-title">{shareSpec.jobName}</h3><p>Copy this YAML into CLI submission, another Kuafu UI, or a review thread.</p></div><button className="icon-only" onClick={close} aria-label="Close share spec">×</button></div>
      <textarea className="share-spec-textarea" readOnly value={shareSpec.yaml} rows="20" />
      <div className="modal-actions"><button className="ghost-button" onClick={close}>Close</button><button className="primary-button" onClick={copyShareSpec}>{shareSpec.copied ? 'Copied' : 'Copy YAML'}</button></div>
    </section>
  </div>;
}

function PlanSummary({ form, templates, taskCount, taskGpuTotal }) {
  return <aside className="plan-summary-card"><div><p className="eyebrow">Plan summary</p><h3>{form.name || 'unnamed-job'}</h3></div><dl><div><dt>Queue</dt><dd>{form.queue}</dd></div><div><dt>Tasks</dt><dd>{taskCount}</dd></div><div><dt>GPUs</dt><dd>{taskGpuTotal}</dd></div><div><dt>Script mode</dt><dd>{form.useSameCommand ? 'same script' : 'per-role script'}</dd></div></dl><div className="plan-role-list">{templates.map((template) => <span key={`${template.name}-${template.role}`}>{template.role}: {template.replicas} x {template.gpuCount} GPU</span>)}</div></aside>;
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
    entrypointScript: current.entrypointScript || '',
    command: template.command,
    useSameCommand: true,
    taskTemplates: template.id?.startsWith('glm52') ? sglangDistributedTemplates(template.command, template.image) : defaultDistributedTemplates(template.command, template.image),
  }));
}

function TaskTemplateEditor({ template, commonCommand, commonImage, commonDockerOptions, commonWorkingDirectory, useSameCommand, canRemove, update, remove }) {
  const hasRuntimeOverride = Boolean((template.image && template.image !== commonImage) || template.workingDirectory || (template.dockerOptions || []).length);
  return <article className="task-template-card compact-task-template"><div className="task-template-header"><span><strong>{template.name}</strong><StatusBadge status={template.role} /></span>{canRemove && <button className="ghost-button danger" onClick={remove}>Remove</button>}</div><label>Name<input value={template.name} onChange={(event) => update({ ...template, name: sanitizeName(event.target.value) })} /></label><label>Role<input value={template.role} onChange={(event) => update({ ...template, role: sanitizeName(event.target.value) })} /></label><label>Replicas<input type="number" min="1" value={template.replicas} onChange={(event) => update({ ...template, replicas: Number(event.target.value) || 1 })} /></label><label>GPUs/task<input type="number" min="0" value={template.gpuCount} onChange={(event) => update({ ...template, gpuCount: Number(event.target.value) || 0 })} /></label><label>CPU/task<input type="number" min="0" value={template.cpuCount || 0} onChange={(event) => update({ ...template, cpuCount: Number(event.target.value) || 0 })} /></label><label>Memory GB<input type="number" min="0" value={template.memoryGb || 0} onChange={(event) => update({ ...template, memoryGb: Number(event.target.value) || 0 })} /></label><div className="full-span inherited-command"><span>{useSameCommand ? 'Script inherited' : 'Role script'}</span>{useSameCommand ? <code>{commonCommand}</code> : <textarea rows="6" value={template.command || commonCommand} onChange={(event) => update({ ...template, command: event.target.value })} />}</div><details className="role-override-details full-span" open={hasRuntimeOverride}><summary>Runtime overrides</summary><label>Image override<input value={template.image && template.image !== commonImage ? template.image : ''} onChange={(event) => update({ ...template, image: event.target.value || commonImage })} placeholder={commonImage} /></label><label>Working directory override<input value={template.workingDirectory || ''} onChange={(event) => update({ ...template, workingDirectory: event.target.value })} placeholder={commonWorkingDirectory} /></label><label>Docker options override<textarea rows="3" value={(template.dockerOptions || []).join('\n')} onChange={(event) => update({ ...template, dockerOptions: splitLines(event.target.value) })} placeholder={commonDockerOptions} /></label></details></article>;
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
  setJobForm((current) => ({ ...current, useSameCommand: checked, taskTemplates: checked ? current.taskTemplates.map((template) => ({ ...template, command: current.command })) : current.taskTemplates }));
}

function updateTaskTemplate(setJobForm, setPreview, index, next) {
  setPreview(null);
  setJobForm((current) => ({ ...current, taskTemplates: current.taskTemplates.map((template, i) => i === index ? next : template) }));
}

function updateForm(setJobForm, setPreview, field, value) {
  setPreview(null);
  setJobForm((current) => ({ ...current, [field]: value, taskTemplates: field === 'command' && current.useSameCommand ? current.taskTemplates.map((template) => ({ ...template, command: value })) : current.taskTemplates }));
}

function toJobPayload(form) {
  const templates = effectiveTaskTemplates(form);
  const launcherSpec = toLauncherSpec(form);
  return {
    name: form.name,
    project: form.project || undefined,
    queue: form.queue,
    command: form.command,
    entrypointScript: form.entrypointScript,
    image: form.image,
    gpuCount: totalTemplateGPUs(templates),
    launcherSpec,
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

function toLauncherSpec(form) {
  return {
    name: form.name || 'unnamed-job',
    project: form.project || undefined,
    queue: form.queue,
    replicaPolicy: form.replicaPolicy || 'fixed',
    workingDirectory: form.workingDirectory || '/workspace',
    entrypointScript: form.entrypointScript || '',
    dependencies: [],
    docker: { image: form.image, options: splitLines(form.dockerOptions) },
    env: parseEnvLines(form.sharedEnv),
    tasks: effectiveTaskTemplates(form).map((template) => ({ ...template, workingDirectory: template.workingDirectory || form.workingDirectory || '/workspace', dockerOptions: template.dockerOptions || splitLines(form.dockerOptions) })),
  };
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

function renameLauncherSpec(spec, name) {
  return { ...spec, name, metadata: spec.metadata ? { ...spec.metadata, name } : undefined };
}

function formFromLauncherSpec(spec) {
  const normalized = normalizeLauncherSpec(spec);
  const tasks = normalized.tasks?.length ? normalized.tasks : defaultDistributedTemplates(defaultTaskScript, defaultImage);
  const firstTask = tasks[0] || {};
  return {
    name: normalized.name || '',
    queue: normalized.queue || 'default',
    project: normalized.project || '',
    image: normalized.docker?.image || firstTask.image || defaultImage,
    command: firstTask.command || defaultTaskScript,
    entrypointScript: normalized.entrypointScript || firstTask.entrypointScript || '',
    useSameCommand: tasks.every((task) => task.command === firstTask.command),
    replicaPolicy: normalized.replicaPolicy || 'fixed',
    workingDirectory: normalized.workingDirectory || firstTask.workingDirectory || '/workspace',
    dockerOptions: (normalized.docker?.options || firstTask.dockerOptions || []).join('\n'),
    sharedEnv: (normalized.env || []).map((item) => `${item.name}=${item.value || ''}`).join('\n'),
    taskTemplates: tasks,
  };
}

function launcherSpecToYaml(spec) {
  spec = normalizeLauncherSpec(spec);
  const env = spec.env || [];
  const dependencies = spec.dependencies || [];
  const tasks = spec.tasks || [];
  return [
    `name: ${spec.name || 'unnamed-job'}`,
    spec.project ? `project: ${spec.project}` : '',
    `queue: ${spec.queue || 'default'}`,
    `replicaPolicy: ${spec.replicaPolicy || 'fixed'}`,
    `workingDirectory: ${spec.workingDirectory || '/workspace'}`,
    spec.entrypointScript ? `entrypointScript: ${yamlScalar(spec.entrypointScript, 2)}` : '',
    'dependencies:',
    ...(dependencies.length ? dependencies.map((item) => `  - ${item}`) : ['  []']),
    'docker:',
    `  image: ${yamlScalar(spec.docker?.image || defaultImage, 2)}`,
    `  options: ${yamlList(spec.docker?.options || [])}`,
    'env:',
    ...(env.length ? env.map((item) => `  - ${item.name}: ${yamlScalar(item.value || '', 4)}`) : ['  []']),
    'tasks:',
    ...tasks.flatMap((task) => [
      `  - name: ${task.name}`,
      `    role: ${task.role}`,
      `    replicas: ${task.replicas || 1}`,
      task.minReplicas ? `    minReplicas: ${task.minReplicas}` : '',
      task.maxReplicas ? `    maxReplicas: ${task.maxReplicas}` : '',
      task.image ? `    image: ${yamlScalar(task.image, 4)}` : '',
      task.entrypointScript ? `    entrypointScript: ${yamlScalar(task.entrypointScript, 6)}` : '',
      `    command: ${yamlScalar(task.command || defaultTaskScript, 6)}`,
      `    workingDirectory: ${yamlScalar(task.workingDirectory || spec.workingDirectory || '/workspace', 4)}`,
      `    dockerOptions: ${yamlList(task.dockerOptions || spec.docker?.options || [])}`,
      `    gpuCount: ${task.gpuCount || 0}`,
      `    cpuCount: ${task.cpuCount || 0}`,
      `    memoryGb: ${task.memoryGb || 0}`,
    ]),
  ].filter((line) => line !== '').join('\n');
}

function yamlScalar(value, blockIndent) {
  const text = String(value ?? '');
  if (text.includes('\n')) {
    const indent = ' '.repeat(blockIndent);
    return `|-\n${text.split('\n').map((line) => `${indent}${line}`).join('\n')}`;
  }
  if (text === '' || /[:#\[\]{}&,*!|>'"%@`]|^\s|\s$/.test(text)) return JSON.stringify(text);
  return text;
}

function yamlList(values) {
  const items = values || [];
  if (!items.length) return '[]';
  return `[${items.map((item) => yamlScalar(item, 0)).join(', ')}]`;
}

function parseLauncherYaml(text) {
  const spec = { replicaPolicy: 'fixed', docker: {}, env: [], dependencies: [], tasks: [] };
  let section = '';
  let nested = '';
  let currentTask = null;
  const lines = String(text || '').split('\n');
  for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
    const rawLine = lines[lineIndex];
    const line = rawLine.replace(/\s+$/, '');
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const indent = line.length - line.trimStart().length;
    let [key, value] = splitYamlPair(trimmed.replace(/^-\s+/, ''));
    if (isBlockScalar(value)) {
      const block = readYamlBlock(lines, lineIndex, indent);
      value = block.value;
      lineIndex = block.nextIndex;
    }
    if (indent === 0) {
      currentTask = null;
      if (['name', 'project', 'queue', 'description', 'replicaPolicy', 'workingDirectory', 'entrypointScript'].includes(key)) spec[key] = value;
      if (key === 'apiVersion') spec.apiVersion = value;
      if (key === 'kind') spec.kind = value;
      if (['metadata', 'spec', 'docker', 'env', 'dependencies', 'tasks'].includes(key)) { section = key; nested = ''; }
      continue;
    }
    if (section === 'dependencies') {
      if (trimmed.startsWith('- ')) spec.dependencies.push(key);
      continue;
    }
    if (section === 'metadata') {
      if (key === 'name') spec.name = value;
      if (key === 'project') spec.project = value;
      continue;
    }
    if (section === 'docker') {
      if (key === 'image') spec.docker.image = value;
      if (key === 'options') spec.docker.options = parseYamlList(value);
      continue;
    }
    if (section === 'env') {
      if (trimmed.startsWith('- ')) spec.env.push({ name: key, value });
      continue;
    }
    if (section === 'tasks') {
      parseTaskLine(spec.tasks, { currentTaskRef: (next) => { currentTask = next; }, currentTask }, key, value, trimmed);
      if (trimmed.startsWith('- ')) currentTask = spec.tasks.at(-1);
      continue;
    }
    if (section !== 'spec') continue;
    if (indent === 2 && ['docker', 'env', 'tasks'].includes(key)) { nested = key; continue; }
    if (!nested) {
      if (key === 'queue') spec.queue = value;
      if (key === 'replicaPolicy') spec.replicaPolicy = value;
      if (key === 'workingDirectory') spec.workingDirectory = value;
      if (key === 'entrypointScript') spec.entrypointScript = value;
      continue;
    }
    if (nested === 'docker') {
      if (key === 'image') spec.docker.image = value;
      if (key === 'options') spec.docker.options = parseYamlList(value);
      continue;
    }
    if (nested === 'env') {
      if (trimmed.startsWith('- ')) spec.env.push({ name: key, value });
      continue;
    }
    if (nested === 'tasks') {
      parseTaskLine(spec.tasks, { currentTaskRef: (next) => { currentTask = next; }, currentTask }, key, value, trimmed);
      if (trimmed.startsWith('- ')) currentTask = spec.tasks.at(-1);
    }
  }
  if (!spec.name) throw new Error('launcher YAML must include name');
  if (!spec.queue) spec.queue = 'default';
  if (!spec.tasks.length) throw new Error('launcher YAML must include tasks');
  return spec;
}

function normalizeLauncherSpec(spec) {
  if (!spec) return toLauncherSpec(emptyJobForm);
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

function parseTaskLine(tasks, state, key, value, trimmed) {
  let currentTask = state.currentTask;
  if (trimmed.startsWith('- ')) {
    currentTask = {};
    tasks.push(currentTask);
    state.currentTaskRef(currentTask);
  }
  if (!currentTask) return;
  if (['replicas', 'minReplicas', 'maxReplicas', 'gpuCount', 'cpuCount', 'memoryGb'].includes(key)) currentTask[key] = Number(value) || 0;
  else if (key === 'dockerOptions') currentTask.dockerOptions = parseYamlList(value);
  else currentTask[key] = value;
}

function splitYamlPair(line) {
  const index = line.indexOf(':');
  if (index < 0) return [line.trim(), ''];
  return [line.slice(0, index).trim(), unquote(line.slice(index + 1).trim())];
}

function parseYamlList(value) {
  const cleaned = String(value || '').trim().replace(/^\[/, '').replace(/\]$/, '');
  if (!cleaned) return [];
  return cleaned.split(',').map((item) => unquote(item.trim())).filter(Boolean);
}

function isBlockScalar(value) {
  return value === '|' || value === '|-' || value === '|+';
}

function readYamlBlock(lines, startIndex, parentIndent) {
  const blockIndent = parentIndent + 2;
  const values = [];
  let index = startIndex + 1;
  for (; index < lines.length; index++) {
    const rawLine = lines[index].replace(/\s+$/, '');
    const trimmed = rawLine.trim();
    const indent = rawLine.length - rawLine.trimStart().length;
    if (trimmed && indent <= parentIndent) break;
    values.push(rawLine.length >= blockIndent ? rawLine.slice(blockIndent) : '');
  }
  return { value: values.join('\n').replace(/\n$/, ''), nextIndex: index - 1 };
}

function unquote(value) {
  if (value.startsWith('"') && value.endsWith('"')) {
    try {
      return JSON.parse(value);
    } catch {
      return value.slice(1, -1);
    }
  }
  if (value.startsWith("'") && value.endsWith("'")) return value.slice(1, -1);
  return value;
}

function splitLines(value) {
  return String(value || '').split('\n').map((line) => line.trim()).filter(Boolean);
}

function effectiveTaskTemplates(form) {
  if (!form.useSameCommand) return form.taskTemplates.map((template) => ({ ...template, command: template.command || form.command, image: template.image || form.image }));
  return form.taskTemplates.map((template) => ({ ...template, command: form.command, image: template.image || form.image }));
}

function totalTemplateGPUs(templates = []) {
  return templates.reduce((sum, template) => sum + ((Number(template.replicas) || 1) * (Number(template.gpuCount) || 0)), 0) || 1;
}

function totalTemplateCount(templates = []) {
  return templates.reduce((sum, template) => sum + (Number(template.replicas) || 1), 0) || 1;
}

function defaultDistributedTemplates(command = defaultTaskScript, image = defaultImage) {
  return [
    { name: 'master', role: 'master', replicas: 1, image, command, gpuCount: 1, cpuCount: 4, memoryGb: 16 },
    { name: 'worker', role: 'worker', replicas: 1, image, command, gpuCount: 1, cpuCount: 4, memoryGb: 16 },
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
    return { id: `${template.name}-${taskRank}`, name: `${template.name}-${ordinal}`, role: template.role, rank: taskRank, ordinal, image: template.image || form.image, workingDirectory: template.workingDirectory || form.workingDirectory || '/workspace', dockerOptions: template.dockerOptions || splitLines(form.dockerOptions), gpuCount: template.gpuCount, cpuCount: template.cpuCount, memoryGb: template.memoryGb, entrypointScript: renderEnv(template.entrypointScript || form.entrypointScript || '', env), command: renderEnv(template.command, env), env };
  }));
}

function renderDockerLaunchCommand(task) {
  const parts = ['docker run --rm', '--name', shellQuote(`kuafu-${task.name}`)];
  const options = task.dockerOptions || [];
  if (options.length) parts.push(options.join(' '));
  if (task.workingDirectory) parts.push('-w', shellQuote(task.workingDirectory));
  for (const item of task.env || []) {
    parts.push('-e', shellQuote(`${item.name}=${item.value}`));
  }
  parts.push(shellQuote(task.image || defaultImage), '/bin/bash', '-lc', shellQuote(renderTaskShellScript(task)));
  return parts.join(' ');
}

function renderTaskShellScript(task) {
  return ['set -e', task.entrypointScript, task.command].filter((part) => String(part || '').trim()).join('\n');
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
    if (!isReservedEnvName(item.name)) continue;
    command = command.replaceAll(`$${item.name}`, item.value).replaceAll(`\${${item.name}}`, item.value);
  }
  return command;
}

function isReservedEnvName(name) {
  return ['RANK', 'TASK_RANK', 'TASK_INDEX', 'TASK_ORDINAL', 'TASK_ROLE', 'WORLD_SIZE', 'TASK0_ADDRESS', 'MASTER_ADDRESS', 'MASTER_PORT', 'KUAFU_TASK_ADDRESS'].includes(String(name || '').toUpperCase());
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
