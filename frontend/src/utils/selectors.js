export function normalizeQueues(queues) {
  return queues
    .map((queue) => ({ ...queue, status: queue.status || 'Active' }))
    .sort((left, right) => (left.name === 'default' ? -1 : right.name === 'default' ? 1 : left.name.localeCompare(right.name)));
}

export function isExpired(reservation) {
  return reservation.expiresAt && new Date(reservation.expiresAt) < new Date();
}

export function enrichNodes(nodes, gpus, reservations, jobs, telemetry = []) {
  return nodes
    .map((node) => {
      const nodeGpus = gpus.filter((gpu) => gpu.nodeName === node.name);
      const telemetryGpus = telemetry.filter((gpu) => !gpu.nodeName || gpu.nodeName === node.name);
      const allocatedGpus = nodeGpus.filter((gpu) => gpu.status === 'Allocated').length;
      const usedTelemetryGpus = telemetryGpus.filter(isGpuBusy).length;
      const avgUtilization = telemetryGpus.length ? Math.round(telemetryGpus.reduce((sum, gpu) => sum + (gpu.utilization || 0), 0) / telemetryGpus.length) : 0;
      const memoryUsedMb = telemetryGpus.reduce((sum, gpu) => sum + (gpu.memoryUsedMb || 0), 0);
      const memoryTotalMb = telemetryGpus.reduce((sum, gpu) => sum + (gpu.memoryTotalMb || 0), 0);
      const reservation = reservations.find((item) => (item.nodeNames || []).includes(node.name));
      return {
        ...node,
        gpus: nodeGpus,
        telemetryGpus,
        gpuCount: node.gpuCount || nodeGpus.length,
        allocatedGpus,
        usedTelemetryGpus,
        avgUtilization,
        memoryUsedMb,
        memoryTotalMb,
        freeGpus: Math.max(0, (node.gpuCount || nodeGpus.length) - allocatedGpus),
        reservation,
        jobs: jobs.filter((job) => (job.allocatedGpus || []).some((gpuId) => nodeGpus.some((gpu) => gpu.id === gpuId))),
      };
    })
    .sort((left, right) => left.name.localeCompare(right.name));
}

export function isGpuBusy(gpu) {
  return (gpu.processes || []).length > 0 || (gpu.utilization || 0) > 0 || (gpu.memoryUsedMb || 0) > 256;
}

export function nodesForGPUIds(gpuIds, allGpus) {
  return Array.from(new Set(gpuIds.map((gpuId) => allGpus.find((gpu) => gpu.id === gpuId)?.nodeName).filter(Boolean))).sort();
}
