export function MetricCard({ label, value, hint, icon: Icon }) {
  return <article className="metric-card"><Icon size={20} /><span>{label}</span><strong>{value}</strong><small>{hint}</small></article>;
}

export function StatusBadge({ status }) {
  return <span className={`status-badge status-${String(status || 'unknown').toLowerCase().replaceAll(' ', '-')}`}>{status || 'Unknown'}</span>;
}

export function PressureBar({ value }) {
  return <div className="pressure"><span style={{ width: `${Math.max(0, Math.min(100, value || 0))}%` }} /><strong>{value || 0}%</strong></div>;
}

export function GpuUsageCard({ gpu }) {
  return <article className="usage-card"><strong>{gpu.gpuId}</strong><span>{gpu.nodeName} · {gpu.model}</span><div>Utilization <b>{gpu.utilization}%</b></div><div>Memory <b>{gpu.memoryUsedMb} / {gpu.memoryTotalMb} MB</b></div><div>Power <b>{gpu.powerWatts} W</b> · Temp <b>{gpu.temperatureC} C</b></div></article>;
}

export function KeyValue({ label, value }) {
  return <div className="kv"><span>{label}</span><strong>{value}</strong></div>;
}

export function CodeBlock({ value }) {
  return <pre className="code-block">{value || '-'}</pre>;
}

export function CapabilityCard({ title, status, body }) {
  return <article className="capability-card"><div><strong>{title}</strong><StatusBadge status={status} /></div><p>{body}</p></article>;
}
