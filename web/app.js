// Kuafu Dashboard Application

const app = {
    // API base URL (defaults to same origin)
    apiBase: window.location.origin,
    
    // Current active tab
    currentTab: 'overview',
    
    // Cache for data
    cache: {
        nodes: [],
        gpus: [],
        jobs: [],
        queues: [],
        summary: {}
    },

    // Initialize the application
    init() {
        this.setupEventListeners();
        this.loadData();
        // Auto-refresh every 10 seconds
        setInterval(() => this.refresh(), 10000);
    },

    // Setup event listeners
    setupEventListeners() {
        // Tab navigation
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                const tab = e.target.dataset.tab;
                this.switchTab(tab);
            });
        });

        // Job submission form
        const form = document.getElementById('submit-job-form');
        form.addEventListener('submit', (e) => {
            e.preventDefault();
            this.submitJob(new FormData(form));
        });
    },

    // Switch between tabs
    switchTab(tab) {
        // Update nav links
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.toggle('active', link.dataset.tab === tab);
        });

        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.toggle('active', content.id === `${tab}-tab`);
        });

        this.currentTab = tab;
    },

    // Load all data
    async loadData() {
        try {
            await Promise.all([
                this.loadClusterSummary(),
                this.loadNodes(),
                this.loadGPUs(),
                this.loadJobs(),
                this.loadQueues()
            ]);
            this.render();
        } catch (error) {
            console.error('Error loading data:', error);
            this.showError('Failed to load cluster data');
        }
    },

    // Refresh current view
    async refresh() {
        await this.loadData();
    },

    // API call wrapper
    async apiCall(endpoint) {
        const response = await fetch(`${this.apiBase}${endpoint}`);
        if (!response.ok) {
            throw new Error(`API error: ${response.status}`);
        }
        return response.json();
    },

    // Load cluster summary
    async loadClusterSummary() {
        try {
            this.cache.summary = await this.apiCall('/api/v1/cluster/summary');
        } catch (error) {
            // Fallback: calculate summary from other endpoints
            this.cache.summary = {
                nodeCount: this.cache.nodes.length,
                gpuCount: this.cache.gpus.length,
                activeJobs: this.cache.jobs.filter(j => j.status === 'Running').length,
                queueCount: this.cache.queues.length
            };
        }
    },

    // Load nodes
    async loadNodes() {
        const data = await this.apiCall('/api/v1/nodes');
        this.cache.nodes = data.nodes || [];
    },

    // Load GPUs
    async loadGPUs() {
        const data = await this.apiCall('/api/v1/gpus');
        this.cache.gpus = data.gpus || [];
    },

    // Load jobs
    async loadJobs() {
        try {
            const data = await this.apiCall('/api/v1/jobs');
            this.cache.jobs = data.jobs || [];
        } catch (error) {
            // Jobs endpoint might not be implemented yet
            this.cache.jobs = [];
        }
    },

    // Load queues
    async loadQueues() {
        try {
            const data = await this.apiCall('/api/v1/queues');
            this.cache.queues = data.queues || [];
        } catch (error) {
            // Queues endpoint might not be implemented yet
            this.cache.queues = [];
        }
    },

    // Submit a job
    async submitJob(formData) {
        const resultDiv = document.getElementById('submit-result');
        resultDiv.className = 'result-message';
        resultDiv.style.display = 'none';

        try {
            const jobData = {
                name: formData.get('name'),
                queue: formData.get('queue'),
                image: formData.get('image'),
                command: formData.get('command'),
                gpuCount: parseInt(formData.get('gpuCount')) || 1
            };

            const response = await fetch(`${this.apiBase}/api/v1/jobs`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(jobData)
            });

            if (response.ok) {
                const result = await response.json();
                resultDiv.className = 'result-message success';
                resultDiv.textContent = `Job submitted successfully! ID: ${result.id || 'pending'}`;
                resultDiv.style.display = 'block';
                document.getElementById('submit-job-form').reset();
                // Refresh jobs list
                await this.loadJobs();
                this.renderJobs();
            } else {
                throw new Error(`Failed to submit job: ${response.statusText}`);
            }
        } catch (error) {
            resultDiv.className = 'result-message error';
            resultDiv.textContent = `Error: ${error.message}`;
            resultDiv.style.display = 'block';
        }
    },

    // Render all views
    render() {
        this.renderOverview();
        this.renderNodes();
        this.renderGPUs();
        this.renderJobs();
        this.renderQueues();
        this.populateQueueDropdown();
    },

    // Render overview cards
    renderOverview() {
        // Nodes metric
        const nodesMetric = document.getElementById('nodes-metric');
        nodesMetric.querySelector('.value').textContent = this.cache.nodes.length;
        
        const nodesStatus = document.getElementById('nodes-status');
        const nodesByStatus = this.groupByStatus(this.cache.nodes);
        nodesStatus.innerHTML = Object.entries(nodesByStatus)
            .map(([status, count]) => 
                `<span class="status-badge ${status.toLowerCase()}">${status}: ${count}</span>`
            ).join('');

        // GPUs metric
        const gpusMetric = document.getElementById('gpus-metric');
        gpusMetric.querySelector('.value').textContent = this.cache.gpus.length;
        
        const gpusStatus = document.getElementById('gpus-status');
        const gpusByStatus = this.groupByStatus(this.cache.gpus);
        gpusStatus.innerHTML = Object.entries(gpusByStatus)
            .map(([status, count]) => 
                `<span class="status-badge ${status.toLowerCase()}">${status}: ${count}</span>`
            ).join('');

        // Jobs metric
        const jobsMetric = document.getElementById('jobs-metric');
        const activeJobs = this.cache.jobs.filter(j => 
            j.status === 'Running' || j.status === 'Pending'
        );
        jobsMetric.querySelector('.value').textContent = activeJobs.length;
        
        const jobsStatus = document.getElementById('jobs-status');
        const jobsByStatus = this.groupByStatus(this.cache.jobs);
        jobsStatus.innerHTML = Object.entries(jobsByStatus)
            .map(([status, count]) => 
                `<span class="status-badge ${status.toLowerCase()}">${status}: ${count}</span>`
            ).join('');

        // Queues metric
        const queuesMetric = document.getElementById('queues-metric');
        queuesMetric.querySelector('.value').textContent = this.cache.queues.length;
        
        const queuesStatus = document.getElementById('queues-status');
        const queuesByStatus = this.groupByStatus(this.cache.queues);
        queuesStatus.innerHTML = Object.entries(queuesByStatus)
            .map(([status, count]) => 
                `<span class="status-badge ${status.toLowerCase()}">${status}: ${count}</span>`
            ).join('');
    },

    // Group items by status
    groupByStatus(items) {
        const groups = {};
        items.forEach(item => {
            const status = item.status || 'Unknown';
            groups[status] = (groups[status] || 0) + 1;
        });
        return groups;
    },

    // Render nodes table
    renderNodes() {
        const tbody = document.querySelector('#nodes-table tbody');
        
        if (this.cache.nodes.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No nodes found</td></tr>';
            return;
        }

        tbody.innerHTML = this.cache.nodes.map(node => `
            <tr>
                <td><strong>${this.escape(node.name)}</strong></td>
                <td><span class="status-badge ${node.status.toLowerCase()}">${this.escape(node.status)}</span></td>
                <td>${this.escape(node.hostname || '-')}</td>
                <td>${node.cpuCount || 0}</td>
                <td>${node.memoryGb || 0}</td>
                <td>${node.gpuCount || 0}</td>
                <td>${this.formatDate(node.lastHeartbeat)}</td>
            </tr>
        `).join('');
    },

    // Render GPUs table
    renderGPUs() {
        const tbody = document.querySelector('#gpus-table tbody');
        
        if (this.cache.gpus.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No GPUs found</td></tr>';
            return;
        }

        tbody.innerHTML = this.cache.gpus.map(gpu => `
            <tr>
                <td><code>${this.escape(gpu.id)}</code></td>
                <td>${this.escape(gpu.nodeName)}</td>
                <td>${gpu.index}</td>
                <td>${this.escape(gpu.model)}</td>
                <td>${gpu.memoryMb || 0}</td>
                <td><span class="status-badge ${gpu.status.toLowerCase()}">${this.escape(gpu.status)}</span></td>
                <td>${this.escape(gpu.allocatedTo || '-')}</td>
            </tr>
        `).join('');
    },

    // Render jobs table
    renderJobs() {
        const tbody = document.querySelector('#jobs-table tbody');
        
        if (this.cache.jobs.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No jobs found</td></tr>';
            return;
        }

        tbody.innerHTML = this.cache.jobs.map(job => `
            <tr>
                <td><code>${this.escape(job.id)}</code></td>
                <td><strong>${this.escape(job.name)}</strong></td>
                <td>${this.escape(job.queue || '-')}</td>
                <td><span class="status-badge ${job.status.toLowerCase()}">${this.escape(job.status)}</span></td>
                <td>${job.gpuCount || 0}</td>
                <td>${this.formatDate(job.submittedAt)}</td>
                <td>
                    ${job.status === 'Running' || job.status === 'Pending' ? 
                        `<button class="btn-danger" onclick="app.cancelJob('${job.id}')">Cancel</button>` : 
                        '-'}
                </td>
            </tr>
        `).join('');
    },

    // Render queues table
    renderQueues() {
        const tbody = document.querySelector('#queues-table tbody');
        
        if (this.cache.queues.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="empty-state">No queues found</td></tr>';
            return;
        }

        tbody.innerHTML = this.cache.queues.map(queue => `
            <tr>
                <td><strong>${this.escape(queue.name)}</strong></td>
                <td><span class="status-badge ${queue.status.toLowerCase()}">${this.escape(queue.status)}</span></td>
                <td>${queue.priority || 0}</td>
                <td>${queue.maxGpus || 'unlimited'}</td>
                <td>${queue.pendingJobs || 0}</td>
                <td>${queue.runningJobs || 0}</td>
            </tr>
        `).join('');
    },

    // Populate queue dropdown in job form
    populateQueueDropdown() {
        const select = document.getElementById('job-queue');
        const currentValue = select.value;
        
        select.innerHTML = '<option value="">Select Queue</option>' +
            this.cache.queues
                .filter(q => q.status === 'Active')
                .map(q => `<option value="${this.escape(q.name)}">${this.escape(q.name)}</option>`)
                .join('');
        
        if (currentValue) {
            select.value = currentValue;
        }
    },

    // Cancel a job
    async cancelJob(jobId) {
        if (!confirm(`Are you sure you want to cancel job ${jobId}?`)) {
            return;
        }

        try {
            const response = await fetch(`${this.apiBase}/api/v1/jobs/${jobId}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                await this.loadJobs();
                this.renderJobs();
            } else {
                throw new Error(`Failed to cancel job: ${response.statusText}`);
            }
        } catch (error) {
            alert(`Error canceling job: ${error.message}`);
        }
    },

    // Utility: Format date
    formatDate(dateStr) {
        if (!dateStr) return '-';
        const date = new Date(dateStr);
        return date.toLocaleString();
    },

    // Utility: Escape HTML
    escape(str) {
        if (str == null) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    },

    // Show error message
    showError(message) {
        console.error(message);
        // Could show a toast notification here
    }
};

// Initialize on DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => app.init());
} else {
    app.init();
}
