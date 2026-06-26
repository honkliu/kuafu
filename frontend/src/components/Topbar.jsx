import { RefreshCw, ShieldCheck } from 'lucide-react';
import { pageTitle } from '../utils/format.js';

export default function Topbar({ activeView, loading, error, refresh }) {
  return <>
    <header className="topbar">
      <div><p className="eyebrow">Production direction · A00/A01 lab data</p><h1>{pageTitle(activeView)}</h1></div>
      <button className="icon-button" onClick={refresh} disabled={loading}><RefreshCw size={18} />Refresh</button>
    </header>
    <section className="notice"><ShieldCheck size={18} /><span>No prototype posture: UI now models a production GPU lease/control-plane. Pending capabilities are explicit until backend/runtime is real.</span></section>
    {error && <section className="error-banner">{error}</section>}
  </>;
}
