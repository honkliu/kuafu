import { navigation } from '../data/productPlan.js';

export default function Sidebar({ activeView, setActiveView }) {
  return <aside className="sidebar">
    <div className="brand">
      <div className="brand-mark">K</div>
      <div><strong>Kuafu</strong><span>GPU Lease Platform</span></div>
    </div>
    <nav className="side-nav">
      {navigation.map(({ id, label, icon: Icon }) => (
        <button key={id} className={activeView === id ? 'active' : ''} onClick={() => setActiveView(id)}>
          <Icon size={18} />{label}
        </button>
      ))}
    </nav>
  </aside>;
}
