import { CapabilityCard } from '../components/Primitives.jsx';
import { workspaceTemplates } from '../data/productPlan.js';

export default function WorkspacesView({ openLeaseDrawer }) {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Interactive sessions</p><h2>Workspaces</h2><p>Lease-backed Jupyter, VS Code, SSH, and web terminals. Requires secure executor and network policy.</p></div><button className="primary-button" onClick={() => openLeaseDrawer([])}>Reserve capacity first</button></div><div className="template-grid">{workspaceTemplates.map((name) => <CapabilityCard key={name} title={name} status="executor pending" body="Will launch on reserved GPUs with TTL, audit, URL/SSH access, secrets, and mounted storage." />)}</div></section>;
}
