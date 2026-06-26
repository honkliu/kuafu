import { Package } from 'lucide-react';
import { StatusBadge } from '../components/Primitives.jsx';
import { catalogTemplates } from '../data/productPlan.js';

export default function CatalogView() {
  return <section className="panel stack"><div className="section-header"><div><p className="eyebrow">Images and templates</p><h2>Catalog</h2><p>Professional platforms reduce command errors with approved templates, image metadata, and compatibility checks.</p></div></div><div className="template-grid">{catalogTemplates.map((template) => <article className="template-card" key={template.name}><div><Package size={18} /><StatusBadge status={template.status} /></div><strong>{template.name}</strong><span>{template.image}</span><code>{template.command}</code><small>{template.gpu}</small></article>)}</div></section>;
}
