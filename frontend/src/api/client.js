const apiBase = window.location.origin;

export async function api(path, options) {
  const response = await fetch(`${apiBase}${path}`, options);
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `${response.status} ${response.statusText}`);
  }
  return response.json();
}

export async function loadClusterState() {
  const [nodes, gpus, jobs, queues, reservations] = await Promise.all([
    api('/api/v1/nodes'),
    api('/api/v1/gpus'),
    api('/api/v1/jobs'),
    api('/api/v1/queues'),
    api('/api/v1/reservations'),
  ]);

  return {
    nodes: nodes.nodes || [],
    gpus: gpus.gpus || [],
    jobs: jobs.jobs || [],
    queues: queues.queues || [],
    reservations: reservations.reservations || [],
  };
}

export function createLease(payload) {
  return api('/api/v1/reservations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export function releaseLease(reservationId) {
  return api(`/api/v1/reservations/${reservationId}`, { method: 'DELETE' });
}

export function createLeaseCommand(reservationId, payload) {
  return api(`/api/v1/reservations/${reservationId}/commands`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export async function loadJobDetail(jobId) {
  const [job, logs, metrics] = await Promise.all([
    api(`/api/v1/jobs/${jobId}`),
    api(`/api/v1/jobs/${jobId}/logs`),
    api(`/api/v1/jobs/${jobId}/metrics`),
  ]);
  return { job, logs: logs.logs || [], metrics: metrics.gpus || [], generatedAt: metrics.generatedAt };
}

export function submitJob(payload) {
  return api('/api/v1/jobs', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export function runJobAction(jobId, action) {
  if (action === 'cancel') {
    return api(`/api/v1/jobs/${jobId}`, { method: 'DELETE' });
  }
  return api(`/api/v1/jobs/${jobId}/${action}`, { method: 'POST' });
}

export function createQueue(payload) {
  return api('/api/v1/queues', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export function updateQueue(name, payload) {
  return api(`/api/v1/queues/${encodeURIComponent(name)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
}

export function deleteQueue(name) {
  return api(`/api/v1/queues/${encodeURIComponent(name)}`, { method: 'DELETE' });
}
