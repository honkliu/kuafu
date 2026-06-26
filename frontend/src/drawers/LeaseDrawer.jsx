import { useMemo, useState } from 'react';
import { X } from 'lucide-react';
import { CodeBlock } from '../components/Primitives.jsx';

export default function LeaseDrawer({ nodes, selectedNodes, setSelectedNodes, activeReservations, leaseResult, submitLease, submitLeaseCommand, close }) {
  const [reservationId, setReservationId] = useState('');
  const [leaseForm, setLeaseForm] = useState({ name: 'dedicated-debug', owner: 'lab-user', durationHours: 4 });
  const [commandForm, setCommandForm] = useState({ image: 'nvcr.io/nvidia/pytorch:24.05-py3', dockerRunOptions: '--gpus all --ipc=host --ulimit memlock=-1', entryPoint: '/bin/bash', workingDirectory: '/workspace', environment: 'NCCL_DEBUG=INFO', startCommand: "-lc 'nvidia-smi && torchrun --nproc_per_node=8 train.py'" });
  const [commandPreview, setCommandPreview] = useState(null);
  const [error, setError] = useState('');
  const reservedNodeNames = new Set(activeReservations.flatMap((reservation) => reservation.nodeNames || []));
  const targetReservationId = reservationId || leaseResult?.reservation?.id || '';
  const targetReservation = useMemo(() => findTargetReservation(activeReservations, leaseResult?.reservation, targetReservationId), [activeReservations, leaseResult, targetReservationId]);

  async function onLease(event) {
    event.preventDefault();
    setError('');
    try {
      const lease = await submitLease({ ...leaseForm, durationHours: Number(leaseForm.durationHours) || 0, nodeNames: selectedNodes });
      setReservationId(lease.id);
      setCommandPreview(null);
    } catch (err) {
      setError(err.message);
    }
  }

  function onPreviewCommand(event) {
    event.preventDefault();
    setError('');
    if (!targetReservation) {
      setError('select a lease before previewing commands');
      return;
    }
    const payload = commandPayload(commandForm);
    setCommandPreview({ payload, renderedCommands: renderReservationCommands(targetReservation.nodeNames || [], payload) });
  }

  async function onRecordReviewedCommand() {
    if (!commandPreview) return;
    setError('');
    try {
      await submitLeaseCommand(targetReservationId, { ...commandPreview.payload, renderedCommands: commandPreview.renderedCommands });
      setCommandPreview(null);
    } catch (err) {
      setError(err.message);
    }
  }

  function updateCommand(field, value) {
    setCommandPreview(null);
    setCommandForm((current) => ({ ...current, [field]: value }));
  }

  return <aside className="drawer wide-drawer">
    <div className="drawer-header"><div><p className="eyebrow">Lease workspace</p><h2>Reserve and run command</h2></div><button className="icon-only" onClick={close}><X size={20} /></button></div>
    {error && <div className="error-banner">{error}</div>}
    <section className="drawer-section"><h3>Selected nodes</h3><div className="node-chip-grid">{nodes.map((node) => { const disabled = reservedNodeNames.has(node.name) && !selectedNodes.includes(node.name); return <button key={node.name} className={selectedNodes.includes(node.name) ? 'node-chip active' : 'node-chip'} disabled={disabled} onClick={() => setSelectedNodes(selectedNodes.includes(node.name) ? selectedNodes.filter((name) => name !== node.name) : [...selectedNodes, node.name])}><strong>{node.name}</strong><span>{node.freeGpus}/{node.gpuCount} GPUs free</span></button>; })}</div></section>
    <form className="drawer-section form-grid" onSubmit={onLease}><h3>Create lease</h3><label>Name<input value={leaseForm.name} onChange={(event) => setLeaseForm({ ...leaseForm, name: event.target.value })} required /></label><label>Owner<input value={leaseForm.owner} onChange={(event) => setLeaseForm({ ...leaseForm, owner: event.target.value })} /></label><label>Duration hours<input type="number" min="1" value={leaseForm.durationHours} onChange={(event) => setLeaseForm({ ...leaseForm, durationHours: event.target.value })} /></label><button className="primary-button" disabled={selectedNodes.length === 0}>Reserve {selectedNodes.length} node(s)</button></form>
    <form className="drawer-section form-grid" onSubmit={onPreviewCommand}><h3>Command / workspace spec</h3><label>Lease<select value={targetReservationId} onChange={(event) => { setReservationId(event.target.value); setCommandPreview(null); }} required><option value="">Select lease</option>{activeReservations.map((reservation) => <option key={reservation.id} value={reservation.id}>{reservation.name} ({(reservation.nodeNames || []).join(', ')})</option>)}{leaseResult?.reservation && <option value={leaseResult.reservation.id}>{leaseResult.reservation.name} (new)</option>}</select></label><label>Image<input value={commandForm.image} onChange={(event) => updateCommand('image', event.target.value)} required /></label><label>Docker run options<input value={commandForm.dockerRunOptions} onChange={(event) => updateCommand('dockerRunOptions', event.target.value)} /></label><label>Entrypoint<input value={commandForm.entryPoint} onChange={(event) => updateCommand('entryPoint', event.target.value)} /></label><label>Working directory<input value={commandForm.workingDirectory} onChange={(event) => updateCommand('workingDirectory', event.target.value)} /></label><label>Environment<textarea rows="3" value={commandForm.environment} onChange={(event) => updateCommand('environment', event.target.value)} /></label><label className="full-span">Start command<textarea rows="4" value={commandForm.startCommand} onChange={(event) => updateCommand('startCommand', event.target.value)} required /></label><button className="primary-button" disabled={!targetReservationId}>Preview node commands</button></form>
    {commandPreview && <section className="drawer-section stack"><div className="section-header compact-header"><div><p className="eyebrow">Review before record</p><h3>Editable final node commands</h3><p className="muted-text">These exact commands are recorded for the reservation.</p></div><button className="primary-button" onClick={onRecordReviewedCommand}>Record reviewed command</button></div><div className="node-command-grid">{commandPreview.renderedCommands.map((item, index) => <label key={item.nodeName}>{item.nodeName}<textarea rows="5" value={item.command} onChange={(event) => setCommandPreview((current) => ({ ...current, renderedCommands: current.renderedCommands.map((command, commandIndex) => commandIndex === index ? { ...command, command: event.target.value } : command) }))} /></label>)}</div><CodeBlock value={commandPreview.renderedCommands.map((item) => `${item.nodeName}: ${item.command}`).join('\n')} /></section>}
    {leaseResult?.command && <section className="drawer-section"><h3>Recorded per-node commands</h3><CodeBlock value={(leaseResult.command.renderedCommands || []).map((item) => `${item.nodeName}: ${item.command}`).join('\n')} /></section>}
  </aside>;
}

function findTargetReservation(activeReservations, latestReservation, reservationId) {
  if (!reservationId) return null;
  if (latestReservation?.id === reservationId) return latestReservation;
  return activeReservations.find((reservation) => reservation.id === reservationId) || null;
}

function commandPayload(form) {
  return {
    image: form.image,
    dockerRunOptions: form.dockerRunOptions,
    entryPoint: form.entryPoint,
    workingDirectory: form.workingDirectory,
    environment: form.environment.split('\n').map((line) => line.trim()).filter(Boolean),
    startCommand: form.startCommand,
  };
}

function renderReservationCommands(nodeNames, command) {
  return nodeNames.map((nodeName) => ({ nodeName, command: renderNodeCommand(nodeName, command) }));
}

function renderNodeCommand(nodeName, command) {
  const parts = ['docker run --rm', '--name', shellQuote(`kuafu-${nodeName}-preview`)];
  if (command.dockerRunOptions) parts.push(command.dockerRunOptions);
  if (command.workingDirectory) parts.push('-w', shellQuote(command.workingDirectory));
  for (const env of command.environment || []) parts.push('-e', shellQuote(env));
  if (command.entryPoint) parts.push('--entrypoint', shellQuote(command.entryPoint));
  parts.push(shellQuote(command.image), command.startCommand);
  return parts.join(' ');
}

function shellQuote(value) {
  const text = String(value || '');
  return `'${text.replaceAll("'", "'\\''")}'`;
}
