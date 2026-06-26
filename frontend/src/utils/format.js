export function formatDate(value) {
  if (!value || String(value).startsWith('0001-')) return '-';
  return new Date(value).toLocaleString();
}

export function pageTitle(tab) {
  return {
    cluster: 'Cluster',
    nodes: 'Nodes',
    jobs: 'Jobs',
    queues: 'Queues',
    leases: 'Leases',
    workspaces: 'Workspaces',
    monitoring: 'Monitoring',
    cost: 'Cost',
    catalog: 'Catalog',
    projects: 'Projects',
    audit: 'Audit',
    admin: 'Admin',
  }[tab] || 'Kuafu';
}

export function renderJobDockerCommand(job) {
  const gpuArg = (job.allocatedGpus || []).length > 0 ? `--gpus 'device=${job.allocatedGpus.join(',')}'` : `--gpus ${job.gpuCount || 1}`;
  return `docker run --rm --name kuafu-${job.id} ${gpuArg} ${job.image || '<image>'} ${job.command}`;
}
