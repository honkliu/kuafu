export function normalizeQueues(queues) {
  return queues
    .map((queue) => ({ ...queue, status: queue.status || 'Active' }))
    .sort((left, right) => (left.name === 'default' ? -1 : right.name === 'default' ? 1 : left.name.localeCompare(right.name)));
}

export function isExpired(reservation) {
  return reservation.expiresAt && new Date(reservation.expiresAt) < new Date();
}

export function enrichNodes(nodes, gpus, reservations, jobs) {
  return nodes
    .map((node) => {
      const nodeGpus = gpus.filter((gpu) => gpu.nodeName === node.name);
      const allocatedGpus = nodeGpus.filter((gpu) => gpu.status === 'Allocated').length;
      const reservation = reservations.find((item) => (item.nodeNames || []).includes(node.name));
      return {
        ...node,
        gpus: nodeGpus,
        gpuCount: node.gpuCount || nodeGpus.length,
        allocatedGpus,
        freeGpus: Math.max(0, (node.gpuCount || nodeGpus.length) - allocatedGpus),
        reservation,
        jobs: jobs.filter((job) => (job.allocatedGpus || []).some((gpuId) => nodeGpus.some((gpu) => gpu.id === gpuId))),
      };
    })
    .sort((left, right) => left.name.localeCompare(right.name));
}

export function nodesForGPUIds(gpuIds, allGpus) {
  return Array.from(new Set(gpuIds.map((gpuId) => allGpus.find((gpu) => gpu.id === gpuId)?.nodeName).filter(Boolean))).sort();
}
