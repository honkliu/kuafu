import { useState } from 'react';
import { X } from 'lucide-react';
import { CodeBlock } from '../components/Primitives.jsx';

export default function LeaseDrawer({ nodes, selectedNodes, setSelectedNodes, activeReservations, leaseResult, submitLease, submitLeaseCommand, close }) {
  const [reservationId, setReservationId] = useState('');
  const [leaseForm, setLeaseForm] = useState({ name: 'dedicated-debug', owner: 'lab-user', durationHours: 4 });
  const [commandForm, setCommandForm] = useState({ image: 'nvcr.io/nvidia/pytorch:24.05-py3', dockerRunOptions: '--gpus all --ipc=host --ulimit memlock=-1', entryPoint: '/bin/bash', workingDirectory: '/workspace', environment: 'NCCL_DEBUG=INFO', startCommand: "-lc 'nvidia-smi && torchrun --nproc_per_node=8 train.py'" });
  const [error, setError] = useState('');
  const reservedNodeNames = new Set(activeReservations.flatMap((reservation) => reservation.nodeNames || []));
  const targetReservationId = reservationId || leaseResult?.reservation?.id || '';

  async function onLease(event) {
    event.preventDefault();
    setError('');
    try {
      const lease = await submitLease({ ...leaseForm, durationHours: Number(leaseForm.durationHours) || 0, nodeNames: selectedNodes });
      setReservationId(lease.id);
    } catch (err) {
      setError(err.message);
    }
  }

  async function onCommand(event) {
    event.preventDefault();
    setError('');
    try {
      await submitLeaseCommand(targetReservationId, { ...commandForm, environment: commandForm.environment.split('\n').map((line) => line.trim()).filter(Boolean) });
    } catch (err) {
      setError(err.message);
    }
  }

  return <aside className="drawer wide-drawer"><div className="drawer-header"><div><p className="eyebrow">Lease workspace</p><h2>Reserve and run command</h2></div><button className="icon-only" onClick={close}><X size={20} /></button></div>{error && <div className="error-banner">{error}</div>}<section className="drawer-section"><h3>Selected nodes</h3><div className="node-chip-grid">{nodes.map((node) => { const disabled = reservedNodeNames.has(node.name) && !selectedNodes.includes(node.name); return <button key={node.name} className={selectedNodes.includes(node.name) ? 'node-chip active' : 'node-chip'} disabled={disabled} onClick={() => setSelectedNodes(selectedNodes.includes(node.name) ? selectedNodes.filter((name) => name !== node.name) : [...selectedNodes, node.name])}><strong>{node.name}</strong><span>{node.freeGpus}/{node.gpuCount} GPUs free</span></button>; })}</div></section><form className="drawer-section form-grid" onSubmit={onLease}><h3>Create lease</h3><label>Name<input value={leaseForm.name} onChange={(event) => setLeaseForm({ ...leaseForm, name: event.target.value })} required /></label><label>Owner<input value={leaseForm.owner} onChange={(event) => setLeaseForm({ ...leaseForm, owner: event.target.value })} /></label><label>Duration hours<input type="number" min="1" value={leaseForm.durationHours} onChange={(event) => setLeaseForm({ ...leaseForm, durationHours: event.target.value })} /></label><button className="primary-button" disabled={selectedNodes.length === 0}>Reserve {selectedNodes.length} node(s)</button></form><form className="drawer-section form-grid" onSubmit={onCommand}><h3>Command / workspace spec</h3><label>Lease<select value={targetReservationId} onChange={(event) => setReservationId(event.target.value)} required><option value="">Select lease</option>{activeReservations.map((reservation) => <option key={reservation.id} value={reservation.id}>{reservation.name} ({(reservation.nodeNames || []).join(', ')})</option>)}{leaseResult?.reservation && <option value={leaseResult.reservation.id}>{leaseResult.reservation.name} (new)</option>}</select></label><label>Image<input value={commandForm.image} onChange={(event) => setCommandForm({ ...commandForm, image: event.target.value })} required /></label><label>Docker run options<input value={commandForm.dockerRunOptions} onChange={(event) => setCommandForm({ ...commandForm, dockerRunOptions: event.target.value })} /></label><label>Entrypoint<input value={commandForm.entryPoint} onChange={(event) => setCommandForm({ ...commandForm, entryPoint: event.target.value })} /></label><label>Working directory<input value={commandForm.workingDirectory} onChange={(event) => setCommandForm({ ...commandForm, workingDirectory: event.target.value })} /></label><label>Environment<textarea rows="3" value={commandForm.environment} onChange={(event) => setCommandForm({ ...commandForm, environment: event.target.value })} /></label><label className="full-span">Start command<textarea rows="4" value={commandForm.startCommand} onChange={(event) => setCommandForm({ ...commandForm, startCommand: event.target.value })} required /></label><button className="primary-button" disabled={!targetReservationId}>Record command</button></form>{leaseResult?.command && <section className="drawer-section"><h3>Rendered per-node commands</h3><CodeBlock value={(leaseResult.command.renderedCommands || []).map((item) => `${item.nodeName}: ${item.command}`).join('\n')} /></section>}</aside>;
}
