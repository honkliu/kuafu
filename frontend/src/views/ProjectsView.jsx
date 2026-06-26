import { CapabilityCard } from '../components/Primitives.jsx';

export default function ProjectsView({ queues }) {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Tenancy</p><h2>Projects</h2><p>Project owners need quotas, queues, members, budgets, and approvals.</p></div></div><div className="template-grid"><CapabilityCard title="lab project" status="seeded" body={`${queues.length} queues · lab-admin/lab-user/lab-viewer seeded. OIDC/RBAC pending.`} /><CapabilityCard title="Queue ACLs" status="pending" body="Submitter/viewer/admin roles per queue and per project." /><CapabilityCard title="Budget approvals" status="pending" body="Approval for expensive leases and priority queues." /></div></section>;
}
