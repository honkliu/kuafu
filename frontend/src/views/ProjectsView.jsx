import { CapabilityCard } from '../components/Primitives.jsx';

export default function ProjectsView({ queues }) {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Tenancy</p><h2>Projects</h2><p>Project owners manage quotas, queues, members, budgets, and approvals.</p></div></div><div className="template-grid"><CapabilityCard title="lab project" status="Configured" body={`${queues.length} queues · lab-admin/lab-user/lab-viewer identities.`} /><CapabilityCard title="Queue ACLs" status="Access" body="Submitter/viewer/admin roles per queue and per project." /><CapabilityCard title="Budget approvals" status="Approval" body="Approval for expensive leases and priority queues." /></div></section>;
}
