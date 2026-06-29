import { CapabilityCard } from '../components/Primitives.jsx';
import { workspaceTemplates } from '../data/productPlan.js';

export default function WorkspacesView({ openLeaseDrawer }) {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Interactive sessions</p><h2>Workspaces</h2><p>Lease-backed Jupyter, VS Code, SSH, and web terminals with TTL, audit, secrets, and mounted storage.</p></div><button className="primary-button" onClick={() => openLeaseDrawer([])}>Reserve capacity</button></div><div className="template-grid">{workspaceTemplates.map((name) => <CapabilityCard key={name} title={name} status="Workspace" body="Launch on reserved GPUs with URL/SSH access and policy controls." />)}</div></section>;
}
