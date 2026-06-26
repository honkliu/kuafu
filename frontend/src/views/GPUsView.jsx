import { StatusBadge } from '../components/Primitives.jsx';

export default function GPUsView({ gpus }) {
  return <section className="panel"><div className="section-header"><div><p className="eyebrow">Accelerators</p><h2>GPU Inventory</h2></div></div><div className="gpu-grid">{gpus.map((gpu) => <article key={gpu.id} className="gpu-card"><div><strong>{gpu.id}</strong><StatusBadge status={gpu.status} /></div><span>{gpu.nodeName} · {gpu.model}</span><small>{gpu.memoryMb} MB · Driver {gpu.driver || '-'}</small>{gpu.allocatedTo && <code>{gpu.allocatedTo}</code>}</article>)}</div></section>;
}
