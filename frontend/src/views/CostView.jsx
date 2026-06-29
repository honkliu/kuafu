import { DollarSign, HardDrive, ShieldCheck, Users } from 'lucide-react';
import { MetricCard } from '../components/Primitives.jsx';
import { skuPrices } from '../data/productPlan.js';

export default function CostView({ jobs, reservations }) {
  const gpuHours = jobs.reduce((sum, job) => sum + (job.gpuCount || 0) * (job.status === 'Running' ? 1 : 0.1), 0) + reservations.length * 8;
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Accounting</p><h2>Cost and Usage</h2><p>Cost, duration, image, GPU SKU, and project chargeback stay visible before and after launch.</p></div></div><div className="metric-grid"><MetricCard label="Estimated GPU Hours" value={gpuHours.toFixed(1)} hint="lab estimate" icon={HardDrive} /><MetricCard label="Estimated Cost" value={`$${(gpuHours * 12.8).toFixed(2)}`} hint="A100 lab price book" icon={DollarSign} /><MetricCard label="Billable Leases" value={reservations.length} hint="active/durable records" icon={ShieldCheck} /><MetricCard label="Charge Project" value="lab" hint="project scoped" icon={Users} /></div><div className="table-wrap"><table className="data-table"><thead><tr><th>SKU</th><th>Price / GPU-hour</th><th>Region</th><th>Availability</th></tr></thead><tbody>{skuPrices.map((row) => <tr key={row.sku}><td><strong>{row.sku}</strong></td><td>${row.price}</td><td>{row.region}</td><td>{row.availability}</td></tr>)}</tbody></table></div></section>;
}
