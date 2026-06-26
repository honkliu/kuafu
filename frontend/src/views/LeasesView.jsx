import { StatusBadge } from '../components/Primitives.jsx';
import { formatDate } from '../utils/format.js';

export default function LeasesView({ reservations, openLeaseDrawer, releaseLease }) {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Dedicated capacity</p><h2>Leases</h2><p>Whole-node leases today; GPU/MIG leases become a production runtime/controller iteration.</p></div><button className="primary-button" onClick={() => openLeaseDrawer([])}>Create lease</button></div><div className="lease-grid">{reservations.length === 0 ? <p className="muted-text">No leases yet.</p> : reservations.map((lease) => <article className="lease-card" key={lease.id}><div><strong>{lease.name}</strong><StatusBadge status={lease.status} /></div><span>{(lease.nodeNames || []).join(', ')}</span><small>{lease.owner || '-'} · expires {formatDate(lease.expiresAt)} · {lease.commands?.length || 0} command records</small>{lease.status === 'Active' && <button className="ghost-button danger" onClick={() => releaseLease(lease.id)}>Release</button>}</article>)}</div></section>;
}
