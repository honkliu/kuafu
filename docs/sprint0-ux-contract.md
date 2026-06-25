# Kuafu Sprint 0 Frontend/UX Contract

Status: Sprint 0 – Docs Only, No Web Portal  
Date: 2026-06-24  
Author: Kuafu Frontend Developer

## 1. Purpose

This document defines the user-facing surface for Kuafu Sprint 0 **before web portal implementation begins**. Sprint 0 focuses on establishing API contracts, CLI output expectations, and UX principles that will guide future dashboard development.

Per GPU Architect guidance, Sprint 0 excludes React code. This contract ensures that when web UI work begins, frontend implementation aligns with backend capabilities, inventory semantics, and user workflows validated through CLI and API.

## 2. Sprint 0 Scope

### In Scope

- Inventory API response format for nodes, GPUs, SKUs, health, topology, and allocation.
- CLI output format expectations for `kuafu resource` commands.
- API pagination, filtering, and real-time event contracts.
- Error states, loading states, and empty states for future UI components.
- Dashboard card specifications for when web portal begins.
- Acceptance criteria for transitioning from Sprint 0 to web UI implementation.

### Out of Scope

- React components, TypeScript interfaces, or Vite build configuration.
- Web portal navigation, routing, or authentication UI.
- Job submission wizard or reservation workflows (covered in later sprints).
- Full OpenAPI schema generation (deferred to M2 backend milestone).

## 3. Inventory API Contract

### 3.1 Node Inventory

**Endpoint:** `GET /api/resources/nodes`

**Query Parameters:**

- `limit` (default: 50, max: 500)
- `cursor` (opaque pagination token)
- `state` (filter: `ready | cordoned | draining | maintenance | unknown`)
- `rack` (filter by topology)
- `island` (filter by topology)

**Response:**

```json
{
  "items": [
    {
      "name": "hpcdev000000",
      "state": "ready",
      "cpuCount": 96,
      "memoryGiB": 1740,
      "gpuCount": 8,
      "gpuSkus": ["a100-sxm4-80gb"],
      "topology": {
        "rack": "rack-01",
        "island": "testbed-alpha",
        "datacenter": "azure-eastus"
      },
      "health": {
        "status": "healthy",
        "lastCheck": "2026-06-24T12:00:00Z"
      },
      "labels": {
        "kubernetes.io/hostname": "hpcdev000000",
        "node.kubernetes.io/instance-type": "Standard_ND96asr_v4"
      },
      "allocatedGpus": 0,
      "reservedGpus": 0,
      "maintenanceNote": null
    }
  ],
  "nextCursor": "node_cursor_abc123",
  "hasMore": false,
  "total": 2
}
```

**UI Expectations:**

- Node list table with columns: name, state, GPUs (allocated/total), CPU, memory, rack, health.
- Filterable by state, rack, island, and SKU.
- Click node name to drill into GPU detail.
- State badge colors: `ready` (green), `cordoned` (yellow), `draining` (orange), `maintenance` (gray), `unknown` (red).

### 3.2 GPU Inventory

**Endpoint:** `GET /api/resources/gpus`

**Query Parameters:**

- `limit` (default: 50, max: 500)
- `cursor` (opaque pagination token)
- `sku` (filter: `a100 | h100 | mi300 | a100-1g.5gb | ...`)
- `vendor` (filter: `nvidia | amd`)
- `state` (filter: `free | allocated | reserved | maintenance | degraded | failed | unknown`)
- `nodeName` (filter by node)
- `topology.rack` (filter)
- `topology.island` (filter)

**Response:**

```json
{
  "items": [
    {
      "id": "gpu-00000000-1111-2222-3333-444444444444",
      "vendor": "nvidia",
      "sku": "a100-sxm4-80gb",
      "memoryGiB": 80,
      "nodeName": "hpcdev000000",
      "uuid": "GPU-abcd1234-5678-90ef-ghij-klmnopqrstuv",
      "health": "healthy",
      "allocationState": "allocated",
      "currentJobId": "job-xyz789",
      "reservationId": null,
      "topology": {
        "rack": "rack-01",
        "island": "testbed-alpha",
        "nvlinkDomain": "node-local-0",
        "infinibandDomain": "ib-fabric-1"
      },
      "mig": {
        "enabled": false,
        "profile": null,
        "parentGpuId": null
      },
      "lastHealthCheck": "2026-06-24T12:00:00Z"
    }
  ],
  "nextCursor": "gpu_cursor_def456",
  "hasMore": true,
  "total": 16
}
```

**UI Expectations:**

- GPU list table with columns: ID (short), SKU, node, state, health, job/reservation, topology.
- Filterable by SKU, vendor, state, health, node, rack, island.
- Click GPU ID to see full UUID, detailed topology, health history.
- State badge colors: `free` (green), `allocated` (blue), `reserved` (purple), `maintenance` (gray), `degraded` (yellow), `failed` (red), `unknown` (gray).
- MIG slices display parent GPU link when `mig.enabled=true`.

### 3.3 GPU Topology View

**Endpoint:** `GET /api/resources/topology`

**Query Parameters:**

- `view` (enum: `cluster | rack | island | nvlink | infiniband`)

**Response:**

```json
{
  "view": "nvlink",
  "nodes": [
    {
      "nodeName": "hpcdev000000",
      "gpus": [
        {
          "id": "gpu-00000000-1111-2222-3333-444444444444",
          "index": 0,
          "sku": "a100-sxm4-80gb",
          "nvlinkPeers": [1, 2, 3, 4, 5, 6, 7],
          "state": "allocated"
        }
      ],
      "nvlinkTopology": "NV12"
    }
  ]
}
```

**UI Expectations:**

- Topology explorer with tree/graph views: cluster → rack → island → node → GPU.
- NVLink visualization showing GPU-to-GPU connectivity within nodes.
- InfiniBand visualization showing node-to-node fabric groups.
- Color-code GPUs by allocation state.
- Highlight GPU groups suitable for gang scheduling (same NVLink domain, same IB domain).

### 3.4 SKU Summary

**Endpoint:** `GET /api/resources/skus`

**Response:**

```json
{
  "skus": [
    {
      "name": "a100-sxm4-80gb",
      "vendor": "nvidia",
      "memoryGiB": 80,
      "count": {
        "total": 16,
        "free": 8,
        "allocated": 6,
        "reserved": 0,
        "maintenance": 2,
        "degraded": 0,
        "failed": 0,
        "unknown": 0
      },
      "nodes": ["hpcdev000000", "hpcdev000001"],
      "capabilities": ["cuda", "nvlink", "mig"]
    }
  ]
}
```

**UI Expectations:**

- Dashboard card: SKU inventory with donut chart (free vs allocated vs reserved vs maintenance).
- SKU detail modal: nodes, capabilities, MIG profiles, pricing (future).

## 4. CLI Output Expectations

### 4.1 Node List

**Command:** `kuafu resource node list`

**Output Format:**

```text
NAME             STATE    GPUS       CPU    MEMORY      RACK     HEALTH
hpcdev000000     ready    0/8        96     1740 GiB    rack-01  healthy
hpcdev000001     ready    0/8        96     1740 GiB    rack-01  healthy
```

**Output for filtered/sorted:**

- `kuafu resource node list --state=cordoned`
- `kuafu resource node list --rack=rack-02`

**Detail View:** `kuafu resource node info hpcdev000000`

```text
Node:           hpcdev000000
State:          ready
Health:         healthy (last check: 2026-06-24 12:00:00 UTC)
CPU:            96 cores
Memory:         1740 GiB
GPUs:           8 total (0 allocated, 0 reserved, 8 free)
GPU SKUs:       a100-sxm4-80gb
Topology:
  Rack:         rack-01
  Island:       testbed-alpha
  Datacenter:   azure-eastus
Labels:
  kubernetes.io/hostname: hpcdev000000
  node.kubernetes.io/instance-type: Standard_ND96asr_v4
```

### 4.2 GPU List

**Command:** `kuafu resource gpu list`

**Output Format:**

```text
ID (short)    SKU              NODE             STATE       HEALTH    JOB/RESERVATION
gpu-00...44   a100-sxm4-80gb   hpcdev000000     allocated   healthy   job-xyz789
gpu-11...55   a100-sxm4-80gb   hpcdev000000     free        healthy   -
gpu-22...66   a100-sxm4-80gb   hpcdev000001     reserved    healthy   res-abc123
```

**Filtered:**

- `kuafu resource gpu list --sku=a100 --state=free`
- `kuafu resource gpu list --node=hpcdev000000`

**Detail View:** `kuafu resource gpu info gpu-00000000-1111-2222-3333-444444444444`

```text
GPU ID:         gpu-00000000-1111-2222-3333-444444444444
UUID:           GPU-abcd1234-5678-90ef-ghij-klmnopqrstuv
Vendor:         nvidia
SKU:            a100-sxm4-80gb
Memory:         80 GiB
Node:           hpcdev000000
State:          allocated
Health:         healthy (last check: 2026-06-24 12:00:00 UTC)
Job:            job-xyz789
Reservation:    -
Topology:
  Rack:         rack-01
  Island:       testbed-alpha
  NVLink:       node-local-0
  InfiniBand:   ib-fabric-1
MIG:
  Enabled:      false
```

### 4.3 Topology View

**Command:** `kuafu resource topology`

**Output Format:**

```text
Cluster: Kuafu Testbed
├── Island: testbed-alpha
│   ├── Rack: rack-01
│   │   ├── Node: hpcdev000000 (8 GPUs, 0 allocated)
│   │   │   └── NVLink Domain: node-local-0 (8x a100-sxm4-80gb, NV12 topology)
│   │   └── Node: hpcdev000001 (8 GPUs, 0 allocated)
│   │       └── NVLink Domain: node-local-1 (8x a100-sxm4-80gb, NV12 topology)
│   └── InfiniBand Fabric: ib-fabric-1 (2 nodes connected)
```

### 4.4 SKU Summary

**Command:** `kuafu resource sku list`

**Output Format:**

```text
SKU                VENDOR   MEMORY   TOTAL   FREE   ALLOC   RESERVED   MAINT   FAILED
a100-sxm4-80gb     nvidia   80 GiB   16      8      6       0          2       0
```

## 5. Error, Loading, and Empty States

### 5.1 API Error Responses

**Standard Error Format:**

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "GPU with ID 'gpu-invalid' not found",
    "details": {
      "requestId": "req-123456",
      "timestamp": "2026-06-24T12:34:56Z"
    }
  }
}
```

**Common Error Codes:**

| Code | HTTP Status | Meaning | Frontend Action |
| ------ | ------------- | --------- | ----------------- |
| `RESOURCE_NOT_FOUND` | 404 | GPU/node/job not found | Show "Resource not found" message, offer navigation back |
| `VALIDATION_ERROR` | 400 | Invalid query parameters | Show validation message inline, highlight invalid field |
| `UNAUTHORIZED` | 401 | Auth token missing/expired | Redirect to login |
| `FORBIDDEN` | 403 | User lacks permission | Show "You don't have access" message |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests | Show "Please wait and try again" with retry countdown |
| `INTERNAL_ERROR` | 500 | Backend failure | Show generic error, offer retry button, log details |
| `SERVICE_UNAVAILABLE` | 503 | Backend temporarily down | Show "Service unavailable" with estimated recovery time |

### 5.2 Loading States

**CLI:**

- Show spinner with message: `Fetching resources...`
- Timeout after 30 seconds with error: `Request timed out. Please check your connection and try again.`

**Future Web UI:**

- Skeleton loaders for tables (show column headers + loading rows).
- Loading spinner for cards and detail panels.
- Progress indicators for long-running operations (job submit, reservation create).
- Disable action buttons during async operations to prevent double-submission.

### 5.3 Empty States

**CLI:**

```text
kuafu resource gpu list --sku=h100

No GPUs found matching filters.
Try:
  kuafu resource sku list           (see available SKUs)
  kuafu resource gpu list           (list all GPUs)
```

**Future Web UI:**

**Empty GPU List:**

- Message: "No GPUs found"
- Subtext: "Try adjusting filters or contact your administrator to add GPUs to the cluster."
- Action: "Clear Filters" button (if filters applied), "View All SKUs" link.

**Empty Job List:**

- Message: "No jobs yet"
- Subtext: "Submit your first job to get started."
- Action: "Submit Job" button.

**Empty Reservation List:**

- Message: "No active reservations"
- Subtext: "Reserve GPUs for interactive work or guaranteed future batch jobs."
- Action: "Create Reservation" button.

**No Free GPUs:**

- Message: "All GPUs are currently allocated or reserved"
- Subtext: "Queue your job or check back later."
- Action: "View Queue Status" link, "Submit to Queue" button.

### 5.4 Degraded/Maintenance States

**GPU in Degraded Health:**

- CLI: Show warning icon + `(degraded)` in health column.
- Web UI: Yellow warning badge, tooltip with last error message and health check timestamp.

**Node in Maintenance:**

- CLI: Show `maintenance` state, optional maintenance note.
- Web UI: Gray badge, show maintenance note in tooltip or detail panel.
- Jobs cannot be scheduled to maintenance nodes; existing jobs are gracefully drained.

**Unknown State (Node/GPU Lost):**

- CLI: Show `unknown` state with last-seen timestamp.
- Web UI: Red "unknown" badge, alert message: "This resource has not reported status recently. Contact your administrator."

## 6. Future Dashboard Cards (Web Portal)

### 6.1 Cluster Overview Card

**Metrics:**

- Total GPUs by state (donut chart: free, allocated, reserved, maintenance, degraded, failed).
- Total nodes by state (bar chart: ready, cordoned, draining, maintenance).
- GPU utilization percentage (cluster-wide).
- Active jobs count.
- Active reservations count.

**Actions:**

- "View All Resources" link.
- "Submit Job" button.
- "Create Reservation" button.

### 6.2 GPU Inventory Card

**Content:**

- SKU breakdown (table):
  - SKU name, vendor, memory, total count, free count, allocated count, reserved count.
- Click SKU row to filter GPU list by that SKU.

**Actions:**

- "View Topology" link (opens topology explorer).

### 6.3 Recent Jobs Card

**Content:**

- Last 10 jobs (table): job name, user, queue, state, submitted time, GPUs allocated.
- Real-time state updates via event stream.

**Actions:**

- "View All Jobs" link.
- Click job row to open job detail.

### 6.4 Queue Status Card

**Content:**

- Queue list (table): queue name, pending jobs, running jobs, GPU quota, GPU usage.
- Quota bar (visual): used GPUs / total quota.

**Actions:**

- "Manage Queues" link (admin only).

### 6.5 Health Summary Card

**Content:**

- Recent health alerts (last 5): node/GPU name, issue, timestamp.
- Health trend chart (last 24h): degraded/failed count over time.

**Actions:**

- "View Health Details" link.
- "Acknowledge Alert" button (admin only).

### 6.6 Cost and Usage Card (Future)

**Content:**

- Current billing period GPU-hours by SKU.
- Project credit balance (if applicable).
- Estimated cost for active reservations.

**Actions:**

- "View Detailed Report" link.

## 7. Real-Time Event Stream Contract

**Protocol:** Server-Sent Events (SSE) on `GET /api/events`

**Client Behavior:**

- Subscribe on dashboard mount.
- Reconnect with `Last-Event-ID` header on disconnect.
- Refetch full state if server responds with `REFETCH_REQUIRED` event.

**Event Types:**

| Type | Payload | Frontend Action |
| ------ | --------- | ----------------- |
| `gpu.health_change` | `{ gpuId, oldHealth, newHealth, timestamp }` | Update GPU list/card health badge |
| `gpu.allocation_change` | `{ gpuId, oldState, newState, jobId?, reservationId? }` | Update GPU list/card allocation state |
| `node.state_change` | `{ nodeName, oldState, newState, timestamp }` | Update node list/card state badge |
| `job.state_change` | `{ jobId, oldState, newState, timestamp }` | Update job list/card, show notification |
| `reservation.state_change` | `{ reservationId, oldState, newState, timestamp }` | Update reservation list/card |
| `queue.utilization_change` | `{ queueName, pendingCount, runningCount, gpuUsage }` | Update queue card metrics |

**Event Format:**

```text
id: event-123
event: gpu.health_change
data: {"gpuId":"gpu-00000000-1111-2222-3333-444444444444","oldHealth":"healthy","newHealth":"degraded","timestamp":"2026-06-24T12:34:56Z"}

```

**Replay Window:** 1 hour. If client reconnects with `Last-Event-ID` older than 1 hour, server responds with `REFETCH_REQUIRED` event.

## 8. Pagination and Filtering Best Practices

### 8.1 Cursor-Based Pagination

**Why:** Stable for concurrent updates, efficient for large datasets, avoids page-number drift.

**Request:**

```text
GET /api/resources/gpus?limit=50&cursor=<opaque_token>
```

**Response:**

```json
{
  "items": [...],
  "nextCursor": "gpu_cursor_xyz",
  "hasMore": true,
  "total": 1024
}
```

**Frontend Rules:**

- Store `nextCursor` to fetch next page.
- Disable "Next" button when `hasMore=false`.
- Show "Showing 1-50 of 1024" using `total`.
- Do not construct cursor values; treat as opaque.

### 8.2 Server-Side Filtering

**All list endpoints support filters as query parameters:**

- Use `&` for AND logic.
- Use comma-separated values for OR within a field (e.g., `state=free,allocated`).
- Backend validates allowed filter fields and values.

**Example:**

```text
GET /api/resources/gpus?sku=a100,h100&state=free&rack=rack-01
```

Means: `(sku=a100 OR sku=h100) AND state=free AND rack=rack-01`

**Frontend:**

- Build filter UI with dropdowns/checkboxes for valid enum values.
- Show active filters as removable chips.
- Persist filter state in URL query params for shareable links.

### 8.3 Sorting

**Default Sort:**

- Nodes: by name ascending.
- GPUs: by creation timestamp descending, then ID ascending.
- Jobs: by submission timestamp descending.

**Custom Sort (if supported):**

```text
GET /api/resources/gpus?sort=memoryGiB:desc,nodeName:asc
```

**Frontend:**

- Show sort direction indicator (up/down arrow) on column headers.
- Click column header to toggle sort.

## 9. Acceptance Criteria for Web UI Begin

Sprint 0 transitions to web portal implementation when:

1. **Backend inventory API is live** and returns test data from A00/A01 nodes (2 nodes, 16 GPUs).
2. **CLI commands work** and output matches formats in section 4.
3. **OpenAPI schema is published** for `/api/resources/nodes`, `/api/resources/gpus`, `/api/resources/skus`, `/api/resources/topology`.
4. **Error response format is standardized** per section 5.1.
5. **Event stream endpoint returns mock events** (even if real Kubernetes integration is incomplete).
6. **PM confirms dashboard card priorities** from section 6.
7. **GPU Architect approves API contracts** and pagination/filtering rules.
8. **Backend Developer confirms pagination cursors are stable** and replay window is implemented.

**Handoff Artifacts:**

- This UX contract document (current file).
- OpenAPI spec file for inventory endpoints.
- Sample API responses (JSON) for nodes, GPUs, SKUs, topology.
- Sample event stream data (SSE format).
- CLI demo recording or transcript showing all `kuafu resource` commands.

## 10. Out-of-Scope Deferred to Later Sprints

- Job submission API and UI (M2/M4).
- Reservation API and UI (M5).
- Queue and quota management UI (M2/M4).
- Admin node drain/cordon/maintenance workflows (M6).
- MIG profile configuration UI (post-M6).
- Usage and billing reports UI (M5).
- Authentication and RBAC UI (M2).

## 11. Summary

Sprint 0 establishes the **inventory foundation** for Kuafu without building React components. The contracts defined here ensure:

- CLI users can inspect cluster resources immediately after backend deployment.
- API consumers (future web UI, automation scripts) have stable response formats.
- Frontend Developer can plan component architecture with known data shapes.
- Error, loading, and empty states are handled consistently.
- Real-time updates are scoped and testable.

**Next Steps:**

1. Backend Developer: implement inventory API per section 3.
2. Backend Developer: implement CLI output per section 4.
3. Frontend Developer: validate API contracts with mock responses.
4. PM: confirm dashboard card priorities for M4.
5. GPU Architect: review and approve before web portal work begins.
