import { PressureBar, StatusBadge } from '../components/Primitives.jsx';
import { formatDate } from '../utils/format.js';

export default function NodesView({ nodes, selectedNodes, toggleNode, openLeaseDrawer, releaseLease }) {
  return <section className="panel stack">
    <div className="section-header"><div><p className="eyebrow">Hardware and lease entry point</p><h2>Nodes</h2><p>Select nodes and reserve from here, matching the requested primary workflow.</p></div><div className="toolbar-actions"><span>{selectedNodes.length} selected</span><button className="primary-button" disabled={selectedNodes.length === 0} onClick={() => openLeaseDrawer(selectedNodes)}>Reserve</button></div></div>
    <div className="table-wrap"><table className="data-table"><thead><tr><th></th><th>Node</th><th>Health</th><th>GPU Capacity</th><th>Lease</th><th>Owner / Expires</th><th>Pressure</th><th>Actions</th></tr></thead><tbody>{nodes.map((node) => <NodeRow key={node.name} node={node} selected={selectedNodes.includes(node.name)} toggleNode={toggleNode} openLeaseDrawer={openLeaseDrawer} releaseLease={releaseLease} />)}</tbody></table></div>
  </section>;
}

function NodeRow({ node, selected, toggleNode, openLeaseDrawer, releaseLease }) {
  const leased = Boolean(node.reservation);
  return <tr className={selected ? 'selected-row' : ''}>
    <td><input type="checkbox" checked={selected} disabled={leased} onChange={() => toggleNode(node.name)} /></td>
    <td><strong>{node.name}</strong><small>{node.hostname}</small></td>
    <td><StatusBadge status={node.status} /></td>
    <td>{node.freeGpus} free / {node.allocatedGpus} allocated / {node.gpuCount} total</td>
    <td>{leased ? <StatusBadge status="Leased" /> : <StatusBadge status="Available" />}</td>
    <td>{leased ? <>{node.reservation.owner || '-'}<small>{formatDate(node.reservation.expiresAt)}</small></> : '-'}</td>
    <td><PressureBar value={node.gpuCount ? Math.round((node.allocatedGpus / node.gpuCount) * 100) : 0} /></td>
    <td>{leased ? <button className="ghost-button danger" onClick={() => releaseLease(node.reservation.id)}>Release</button> : <button className="ghost-button" onClick={() => openLeaseDrawer([node.name])}>Reserve</button>}</td>
  </tr>;
}
