import { formatDate } from '../utils/format.js';

export default function AuditView({ jobs, reservations }) {
  const events = [...reservations.map((item) => ({ time: item.createdAt, actor: item.owner || 'lab-user', action: 'lease.create', target: item.name })), ...jobs.map((job) => ({ time: job.submittedAt, actor: job.project || 'lab', action: 'job.submit', target: job.name }))].sort((a, b) => new Date(b.time) - new Date(a.time));
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Governance</p><h2>Audit</h2><p>Every command, lease, admin operation, and runtime mutation must become auditable.</p></div></div><div className="table-wrap"><table className="data-table"><thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Target</th></tr></thead><tbody>{events.map((event, index) => <tr key={index}><td>{formatDate(event.time)}</td><td>{event.actor}</td><td>{event.action}</td><td>{event.target}</td></tr>)}</tbody></table></div></section>;
}
