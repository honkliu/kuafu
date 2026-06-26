import { Layers, Play, Server, Cpu } from '../data/productPlan.js';
import { iterations } from '../data/productPlan.js';
import { MetricCard } from '../components/Primitives.jsx';

export default function ClusterView({ data, nodes, activeJobs, activeReservations, openLeaseDrawer }) {
  const allocated = data.gpus.filter((gpu) => gpu.status === 'Allocated').length;
  const free = data.gpus.filter((gpu) => gpu.status === 'Available').length;
  return <div className="stack">
    <div className="metric-grid">
      <MetricCard label="Nodes" value={nodes.length} hint={`${activeReservations.length} active leases`} icon={Server} />
      <MetricCard label="GPUs" value={data.gpus.length} hint={`${free} free · ${allocated} allocated`} icon={Cpu} />
      <MetricCard label="Active Jobs" value={activeJobs.length} hint={`${data.jobs.length} total workloads`} icon={Play} />
      <MetricCard label="Queues" value={data.queues.length} hint="quota entry points" icon={Layers} />
    </div>
    <section className="panel hero-panel">
      <div><p className="eyebrow">Roadmap driven by OpenPAI / Slurm / lease platforms</p><h2>Eight production iterations</h2><p className="muted-text">Each card is a concrete product iteration. Current lab-backed features are interactive; runtime-dependent work is marked pending rather than faked.</p></div>
      <button className="primary-button" onClick={() => openLeaseDrawer([])}><Server size={18} />Create Lease</button>
    </section>
    <div className="iteration-grid">{iterations.map(([title, description], index) => <article key={title} className="iteration-card"><span>{index + 1}</span><strong>{title}</strong><p>{description}</p></article>)}</div>
  </div>;
}
