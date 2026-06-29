import { RefreshCw } from 'lucide-react';
import { pageTitle } from '../utils/format.js';

export default function Topbar({ activeView, loading, error, refresh }) {
  return <>
    <header className="topbar">
      <div><p className="eyebrow">Kuafu control plane · A00/A01</p><h1>{pageTitle(activeView)}</h1></div>
      <button className="icon-button" onClick={refresh} disabled={loading}><RefreshCw size={18} />Refresh</button>
    </header>
    {error && <section className="error-banner">{error}</section>}
  </>;
}
