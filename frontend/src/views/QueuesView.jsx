import { useState } from 'react';
import { StatusBadge } from '../components/Primitives.jsx';

const emptyQueueForm = { name: '', status: 'Active', project: '', maxGpus: 8, softGpus: '', maxGpusPerJob: '', maxQueuedJobs: '', maxRunningJobs: '', priority: 100, allowBurst: false };

export default function QueuesView({ queues, jobs, saveQueue, deleteQueue }) {
  const [form, setForm] = useState(emptyQueueForm);
  const [editingName, setEditingName] = useState('');
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  async function onSubmit(event) {
    event.preventDefault();
    setMessage('');
    setError('');
    try {
      const queue = await saveQueue(toQueuePayload(form), editingName);
      setMessage(`${editingName ? 'Updated' : 'Created'} queue ${queue.name}`);
      setForm(emptyQueueForm);
      setEditingName('');
    } catch (err) {
      setError(err.message);
    }
  }

  async function onDelete(queue) {
    setMessage('');
    setError('');
    try {
      await deleteQueue(queue.name);
      setMessage(`Deleted queue ${queue.name}`);
      if (editingName === queue.name) {
        setEditingName('');
        setForm(emptyQueueForm);
      }
    } catch (err) {
      setError(err.message);
    }
  }

  async function togglePaused(queue) {
    setMessage('');
    setError('');
    try {
      const nextStatus = (queue.status || 'Active') === 'Paused' ? 'Active' : 'Paused';
      const updated = await saveQueue({ ...queue, status: nextStatus }, queue.name);
      setMessage(`${updated.name} is now ${updated.status}`);
    } catch (err) {
      setError(err.message);
    }
  }

  function edit(queue) {
    setEditingName(queue.name);
    setForm({
      name: queue.name,
      status: queue.status || 'Active',
      project: queue.project || '',
      maxGpus: queue.maxGpus || 1,
      softGpus: queue.softGpus || '',
      maxGpusPerJob: queue.maxGpusPerJob || '',
      maxQueuedJobs: queue.maxQueuedJobs || '',
      maxRunningJobs: queue.maxRunningJobs || '',
      priority: queue.priority || 100,
      allowBurst: Boolean(queue.allowBurst),
    });
  }

  return <section className="panel stack">
    <div className="section-header"><div><p className="eyebrow">Scheduling policy</p><h2>Queues</h2><p>Create queues, tune quotas, pause scheduling, and remove inactive queues.</p></div></div>
    {message && <div className="success-banner">{message}</div>}
    {error && <div className="error-banner">{error}</div>}
    <form className="inline-form-grid queue-form" onSubmit={onSubmit}>
      <label>Name<input value={form.name} disabled={Boolean(editingName)} onChange={(event) => setForm({ ...form, name: event.target.value })} required /></label>
      <label>Status<select value={form.status} onChange={(event) => setForm({ ...form, status: event.target.value })}><option>Active</option><option>Paused</option></select></label>
      <label>Project<input value={form.project} onChange={(event) => setForm({ ...form, project: event.target.value })} placeholder="optional" /></label>
      <label>Priority<input type="number" value={form.priority} onChange={(event) => setForm({ ...form, priority: event.target.value })} /></label>
      <label>Hard GPUs<input type="number" min="1" value={form.maxGpus} onChange={(event) => setForm({ ...form, maxGpus: event.target.value })} required /></label>
      <label>Soft GPUs<input type="number" min="0" value={form.softGpus} onChange={(event) => setForm({ ...form, softGpus: event.target.value })} /></label>
      <label>Max/job<input type="number" min="0" value={form.maxGpusPerJob} onChange={(event) => setForm({ ...form, maxGpusPerJob: event.target.value })} /></label>
      <label>Queued jobs<input type="number" min="0" value={form.maxQueuedJobs} onChange={(event) => setForm({ ...form, maxQueuedJobs: event.target.value })} /></label>
      <label>Running jobs<input type="number" min="0" value={form.maxRunningJobs} onChange={(event) => setForm({ ...form, maxRunningJobs: event.target.value })} /></label>
      <label className="checkbox-label"><input type="checkbox" checked={form.allowBurst} onChange={(event) => setForm({ ...form, allowBurst: event.target.checked })} />Allow soft quota burst</label>
      <div className="form-actions"><button className="primary-button">{editingName ? 'Update queue' : 'Create queue'}</button>{editingName && <button type="button" className="ghost-button" onClick={() => { setEditingName(''); setForm(emptyQueueForm); }}>Cancel edit</button>}</div>
    </form>
    <div className="table-wrap"><table className="data-table"><thead><tr><th>Name</th><th>Status</th><th>Project</th><th>Priority</th><th>GPU limits</th><th>Jobs</th><th>Actions</th></tr></thead><tbody>{queues.map((queue) => <QueueRow key={queue.name} queue={queue} jobs={jobs} edit={edit} togglePaused={togglePaused} onDelete={onDelete} />)}</tbody></table></div>
  </section>;
}

function QueueRow({ queue, jobs, edit, togglePaused, onDelete }) {
  const activeJobs = jobs.filter((job) => job.queue === queue.name && ['Queued', 'Running'].includes(job.status));
  return <tr><td><strong>{queue.name}</strong></td><td><StatusBadge status={queue.status || 'Active'} /></td><td>{queue.project || '-'}</td><td>{queue.priority}</td><td>{queue.maxGpus} hard<small>{queue.softGpus || '-'} soft · {queue.maxGpusPerJob || '-'} per job</small></td><td>{queue.jobsQueued} queued / {queue.jobsRunning} running<small>{activeJobs.length} active records</small></td><td><div className="row-actions"><button className="ghost-button" onClick={() => edit(queue)}>Edit</button><button className="ghost-button" onClick={() => togglePaused(queue)}>{(queue.status || 'Active') === 'Paused' ? 'Resume' : 'Pause'}</button><button className="ghost-button danger" disabled={activeJobs.length > 0} title={activeJobs.length > 0 ? 'Stop or finish active jobs first' : ''} onClick={() => onDelete(queue)}>Delete</button></div></td></tr>;
}

function toQueuePayload(form) {
  return {
    name: form.name.trim(),
    status: form.status,
    project: form.project.trim(),
    maxGpus: Number(form.maxGpus) || 0,
    softGpus: Number(form.softGpus) || 0,
    maxGpusPerJob: Number(form.maxGpusPerJob) || 0,
    maxQueuedJobs: Number(form.maxQueuedJobs) || 0,
    maxRunningJobs: Number(form.maxRunningJobs) || 0,
    priority: Number(form.priority) || 0,
    allowBurst: Boolean(form.allowBurst),
  };
}
