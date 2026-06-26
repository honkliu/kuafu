import { useEffect, useMemo, useState } from 'react';
import Sidebar from './components/Sidebar.jsx';
import Topbar from './components/Topbar.jsx';
import ClusterView from './views/ClusterView.jsx';
import NodesView from './views/NodesView.jsx';
import JobsView from './views/JobsView.jsx';
import QueuesView from './views/QueuesView.jsx';
import LeasesView from './views/LeasesView.jsx';
import WorkspacesView from './views/WorkspacesView.jsx';
import MonitoringView from './views/MonitoringView.jsx';
import CostView from './views/CostView.jsx';
import CatalogView from './views/CatalogView.jsx';
import ProjectsView from './views/ProjectsView.jsx';
import AuditView from './views/AuditView.jsx';
import AdminView from './views/AdminView.jsx';
import LeaseDrawer from './drawers/LeaseDrawer.jsx';
import JobDetailDrawer from './drawers/JobDetailDrawer.jsx';
import { createLease, createLeaseCommand, createQueue, deleteQueue, loadClusterState, loadJobDetail, releaseLease as releaseLeaseApi, runJobAction, submitJob, updateQueue } from './api/client.js';
import { enrichNodes, isExpired, normalizeQueues } from './utils/selectors.js';

export default function App() {
  const [activeView, setActiveView] = useState('cluster');
  const [data, setData] = useState({ nodes: [], gpus: [], jobs: [], queues: [], reservations: [] });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedNodes, setSelectedNodes] = useState([]);
  const [leaseDrawerOpen, setLeaseDrawerOpen] = useState(false);
  const [leaseResult, setLeaseResult] = useState(null);
  const [jobDetailId, setJobDetailId] = useState('');
  const [jobDetail, setJobDetail] = useState(null);
  const [filters, setFilters] = useState({ keyword: '', status: 'all', queue: 'all' });

  async function loadAll() {
    setLoading(true);
    setError('');
    try {
      const state = await loadClusterState();
      setData({ ...state, queues: normalizeQueues(state.queues) });
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadAll();
    const timer = window.setInterval(loadAll, 10000);
    return () => window.clearInterval(timer);
  }, []);

  const activeReservations = useMemo(
    () => data.reservations.filter((reservation) => reservation.status === 'Active' && !isExpired(reservation)),
    [data.reservations],
  );
  const nodes = useMemo(() => enrichNodes(data.nodes, data.gpus, activeReservations, data.jobs), [data.nodes, data.gpus, activeReservations, data.jobs]);
  const activeJobs = data.jobs.filter((job) => ['Queued', 'Running'].includes(job.status));

  function toggleNode(nodeName) {
    setSelectedNodes((current) => current.includes(nodeName) ? current.filter((name) => name !== nodeName) : [...current, nodeName]);
  }

  function openLeaseDrawer(nodeNames = selectedNodes) {
    setSelectedNodes(nodeNames);
    setLeaseResult(null);
    setLeaseDrawerOpen(true);
    setActiveView('nodes');
  }

  async function submitLease(payload) {
    const reservation = await createLease(payload);
    setLeaseResult({ reservation, command: null });
    await loadAll();
    return reservation;
  }

  async function submitLeaseCommand(reservationId, payload) {
    const command = await createLeaseCommand(reservationId, payload);
    setLeaseResult((current) => ({ ...(current || {}), command }));
    await loadAll();
    return command;
  }

  async function releaseLease(reservationId) {
    await releaseLeaseApi(reservationId);
    await loadAll();
  }

  async function openJobDetail(jobId) {
    setJobDetailId(jobId);
    setJobDetail(null);
    setJobDetail(await loadJobDetail(jobId));
  }

  async function handleSubmitJob(payload) {
    const job = await submitJob(payload);
    await loadAll();
    return job;
  }

  async function handleJobAction(jobId, action) {
    const job = await runJobAction(jobId, action);
    await loadAll();
    if (jobDetailId === jobId) {
      setJobDetail(await loadJobDetail(jobId));
    }
    return job;
  }

  async function handleSaveQueue(queue, originalName) {
    const saved = originalName ? await updateQueue(originalName, queue) : await createQueue(queue);
    await loadAll();
    return saved;
  }

  async function handleDeleteQueue(name) {
    await deleteQueue(name);
    await loadAll();
  }

  return <div className="app-shell">
    <Sidebar activeView={activeView} setActiveView={setActiveView} />
    <main className="main-panel">
      <Topbar activeView={activeView} loading={loading} error={error} refresh={loadAll} />
      {activeView === 'cluster' && <ClusterView data={data} nodes={nodes} activeJobs={activeJobs} activeReservations={activeReservations} openLeaseDrawer={openLeaseDrawer} />}
      {activeView === 'nodes' && <NodesView nodes={nodes} selectedNodes={selectedNodes} toggleNode={toggleNode} openLeaseDrawer={openLeaseDrawer} releaseLease={releaseLease} />}
      {activeView === 'jobs' && <JobsView jobs={data.jobs} queues={data.queues} filters={filters} setFilters={setFilters} openJobDetail={openJobDetail} submitJob={handleSubmitJob} runJobAction={handleJobAction} />}
      {activeView === 'queues' && <QueuesView queues={data.queues} jobs={data.jobs} saveQueue={handleSaveQueue} deleteQueue={handleDeleteQueue} />}
      {activeView === 'leases' && <LeasesView reservations={data.reservations} openLeaseDrawer={openLeaseDrawer} releaseLease={releaseLease} />}
      {activeView === 'workspaces' && <WorkspacesView openLeaseDrawer={openLeaseDrawer} />}
      {activeView === 'monitoring' && <MonitoringView nodes={nodes} gpus={data.gpus} jobs={data.jobs} reservations={activeReservations} />}
      {activeView === 'cost' && <CostView jobs={data.jobs} reservations={activeReservations} />}
      {activeView === 'catalog' && <CatalogView />}
      {activeView === 'projects' && <ProjectsView queues={data.queues} />}
      {activeView === 'audit' && <AuditView jobs={data.jobs} reservations={data.reservations} />}
      {activeView === 'admin' && <AdminView nodes={nodes} />}
    </main>
    {leaseDrawerOpen && <LeaseDrawer nodes={nodes} selectedNodes={selectedNodes} setSelectedNodes={setSelectedNodes} activeReservations={activeReservations} leaseResult={leaseResult} submitLease={submitLease} submitLeaseCommand={submitLeaseCommand} close={() => setLeaseDrawerOpen(false)} />}
    {jobDetailId && <JobDetailDrawer detail={jobDetail} close={() => setJobDetailId('')} allGpus={data.gpus} />}
  </div>;
}
