import { CapabilityCard } from '../components/Primitives.jsx';

export default function AdminView({ nodes }) {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Operations</p><h2>Admin</h2><p>These controls are intentionally disabled until audited backend APIs exist.</p></div></div><div className="template-grid"><CapabilityCard title="Drain / cordon node" status="pending" body={`${nodes.length} nodes visible. Requires Kubernetes node operation API and audit.`} /><CapabilityCard title="Disable GPU" status="pending" body="Mark device unhealthy, stop scheduling, preserve root-cause trail." /><CapabilityCard title="Emergency revoke lease" status="pending" body="Force-release capacity with approval, event, and user notification." /><CapabilityCard title="Policy editor" status="pending" body="Queue, budget, preemption, backfill, and idle timeout policy." /></div></section>;
}
