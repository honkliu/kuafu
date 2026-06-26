import { Search } from 'lucide-react';
import { useState } from 'react';
import { StatusBadge } from '../components/Primitives.jsx';
import { formatDate } from '../utils/format.js';

const emptyJobForm = { name: '', queue: 'default', project: '', image: 'nvidia/cuda:12.0-runtime', command: 'nvidia-smi', gpuCount: 1 };

export default function JobsView({ jobs, queues, filters, setFilters, openJobDetail, submitJob, runJobAction }) {
  const [jobForm, setJobForm] = useState(emptyJobForm);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');
  const filtered = jobs.filter((job) => (`${job.id} ${job.name} ${job.command}`.toLowerCase().includes(filters.keyword.toLowerCase())) && (filters.status === 'all' || job.status === filters.status) && (filters.queue === 'all' || job.queue === filters.queue));
  const statuses = ['all', ...Array.from(new Set(jobs.map((job) => job.status)))];

  async function onSubmit(event) {
    event.preventDefault();
    setError('');
    setMessage('');
    try {
      const job = await submitJob({ ...jobForm, gpuCount: Number(jobForm.gpuCount) || 1, project: jobForm.project || undefined });
      setMessage(`Submitted ${job.name} to ${job.queue}`);
      setJobForm({ ...emptyJobForm, queue: jobForm.queue });
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
    <div className="section-header"><div><p className="eyebrow">Workloads</p><h2>Jobs</h2><p>Submit, stop, start, restart, cancel, and inspect jobs from one operational surface.</p></div></div>
    {message && <div className="success-banner">{message}</div>}
    {error && <div className="error-banner">{error}</div>}
    <form className="inline-form-grid" onSubmit={onSubmit}>
      <label>Name<input value={jobForm.name} onChange={(event) => setJobForm({ ...jobForm, name: event.target.value })} required /></label>
      <label>Queue<select value={jobForm.queue} onChange={(event) => setJobForm({ ...jobForm, queue: event.target.value })}>{queues.map((queue) => <option key={queue.name} value={queue.name}>{queue.name}</option>)}</select></label>
      <label>Project<input value={jobForm.project} onChange={(event) => setJobForm({ ...jobForm, project: event.target.value })} placeholder="optional" /></label>
      <label>GPUs<input type="number" min="1" value={jobForm.gpuCount} onChange={(event) => setJobForm({ ...jobForm, gpuCount: event.target.value })} /></label>
      <label className="double-span">Image<input value={jobForm.image} onChange={(event) => setJobForm({ ...jobForm, image: event.target.value })} /></label>
      <label className="double-span">Command<input value={jobForm.command} onChange={(event) => setJobForm({ ...jobForm, command: event.target.value })} required /></label>
      <button className="primary-button">Submit job</button>
    </form>
    <div className="filter-bar"><label className="search-box"><Search size={16} /><input placeholder="Search jobs, IDs, commands" value={filters.keyword} onChange={(event) => setFilters({ ...filters, keyword: event.target.value })} /></label><select value={filters.status} onChange={(event) => setFilters({ ...filters, status: event.target.value })}>{statuses.map((status) => <option key={status}>{status}</option>)}</select><select value={filters.queue} onChange={(event) => setFilters({ ...filters, queue: event.target.value })}><option value="all">all queues</option>{queues.map((queue) => <option key={queue.name} value={queue.name}>{queue.name}</option>)}</select></div>
    <div className="table-wrap"><table className="data-table"><thead><tr><th>Job</th><th>Status</th><th>Queue</th><th>Image</th><th>GPU</th><th>Submitted</th><th>Actions</th></tr></thead><tbody>{filtered.map((job) => <tr key={job.id}><td><strong>{job.name}</strong><small>{job.id}</small></td><td><StatusBadge status={job.status} /></td><td>{job.queue}</td><td className="truncate-cell">{job.image || '-'}</td><td>{job.gpuCount}</td><td>{formatDate(job.submittedAt)}</td><td><div className="row-actions"><button className="ghost-button" onClick={() => openJobDetail(job.id)}>Details</button>{job.status === 'Running' && <button className="ghost-button" onClick={() => onAction(job, 'stop')}>Stop</button>}{job.status === 'Queued' && <button className="ghost-button danger" onClick={() => onAction(job, 'cancel')}>Cancel</button>}{['Stopped', 'Failed', 'Canceled', 'Completed'].includes(job.status) && <button className="ghost-button" onClick={() => onAction(job, 'start')}>Start</button>}{job.status !== 'Queued' && <button className="ghost-button" onClick={() => onAction(job, 'restart')}>Restart</button>}</div></td></tr>)}</tbody></table></div>
  </section>;
}

function actionLabel(action) {
  return { start: 'Started', stop: 'Stopped', restart: 'Restarted', cancel: 'Canceled' }[action] || action;
}
