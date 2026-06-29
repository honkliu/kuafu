import { CapabilityCard } from '../components/Primitives.jsx';

export default function AdminView({ nodes }) {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Operations</p><h2>Admin</h2><p>Fleet controls with audit, approval, and policy guardrails.</p></div></div><div className="template-grid"><CapabilityCard title="Drain / cordon node" status="Guarded" body={`${nodes.length} nodes visible. Node operations require audit and approval.`} /><CapabilityCard title="Disable GPU" status="Guarded" body="Mark device unhealthy, stop scheduling, preserve root-cause trail." /><CapabilityCard title="Emergency revoke lease" status="Guarded" body="Force-release capacity with approval, event, and user notification." /><CapabilityCard title="Policy editor" status="Policy" body="Queue, budget, preemption, backfill, and idle timeout policy." /></div></section>;
}
