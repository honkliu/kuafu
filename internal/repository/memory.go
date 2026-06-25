package repository

import (
	"fmt"
	"sync"

	"github.com/microsoft/kuafu/internal/domain"
)

// MemoryRepository provides in-memory storage for nodes, GPUs, jobs, and queues
type MemoryRepository struct {
	nodes  map[string]*domain.Node
	gpus   map[string]*domain.GPU
	jobs   map[string]*domain.Job
	queues map[string]*domain.Queue
	mu     sync.RWMutex
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		nodes:  make(map[string]*domain.Node),
		gpus:   make(map[string]*domain.GPU),
		jobs:   make(map[string]*domain.Job),
		queues: make(map[string]*domain.Queue),
	}
}

// AddNode adds or updates a node
func (r *MemoryRepository) AddNode(node *domain.Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if node.Name == "" {
		return fmt.Errorf("node name cannot be empty")
	}

	r.nodes[node.Name] = node
	return nil
}

// GetNode retrieves a node by name
func (r *MemoryRepository) GetNode(name string) (*domain.Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	node, exists := r.nodes[name]
	if !exists {
		return nil, fmt.Errorf("node %s not found", name)
	}

	return node, nil
}

// ListNodes returns all nodes
func (r *MemoryRepository) ListNodes() ([]*domain.Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]*domain.Node, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// AddGPU adds or updates a GPU
func (r *MemoryRepository) AddGPU(gpu *domain.GPU) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if gpu.ID == "" {
		return fmt.Errorf("gpu id cannot be empty")
	}

	r.gpus[gpu.ID] = gpu
	return nil
}

// GetGPU retrieves a GPU by ID
func (r *MemoryRepository) GetGPU(id string) (*domain.GPU, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gpu, exists := r.gpus[id]
	if !exists {
		return nil, fmt.Errorf("gpu %s not found", id)
	}

	return gpu, nil
}

// ListGPUs returns all GPUs, optionally filtered by node
func (r *MemoryRepository) ListGPUs(nodeName string) ([]*domain.GPU, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	gpus := make([]*domain.GPU, 0)
	for _, gpu := range r.gpus {
		if nodeName == "" || gpu.NodeName == nodeName {
			gpus = append(gpus, gpu)
		}
	}

	return gpus, nil
}

// AddJob adds or updates a job
func (r *MemoryRepository) AddJob(job *domain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if job.ID == "" {
		return fmt.Errorf("job id cannot be empty")
	}

	r.jobs[job.ID] = job
	return nil
}

// GetJob retrieves a job by ID
func (r *MemoryRepository) GetJob(id string) (*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, exists := r.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job %s not found", id)
	}

	return job, nil
}

// ListJobs returns all jobs
func (r *MemoryRepository) ListJobs() ([]*domain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jobs := make([]*domain.Job, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// DeleteJob removes a job by ID
func (r *MemoryRepository) DeleteJob(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[id]; !exists {
		return fmt.Errorf("job %s not found", id)
	}

	delete(r.jobs, id)
	return nil
}

// AddQueue adds or updates a queue
func (r *MemoryRepository) AddQueue(queue *domain.Queue) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if queue.Name == "" {
		return fmt.Errorf("queue name cannot be empty")
	}

	r.queues[queue.Name] = queue
	return nil
}

// GetQueue retrieves a queue by name
func (r *MemoryRepository) GetQueue(name string) (*domain.Queue, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	queue, exists := r.queues[name]
	if !exists {
		return nil, fmt.Errorf("queue %s not found", name)
	}

	return queue, nil
}

// ListQueues returns all queues
func (r *MemoryRepository) ListQueues() ([]*domain.Queue, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	queues := make([]*domain.Queue, 0, len(r.queues))
	for _, queue := range r.queues {
		queues = append(queues, queue)
	}

	return queues, nil
}

// UpdateGPUAllocation updates GPU allocation status
func (r *MemoryRepository) UpdateGPUAllocation(gpuID, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	gpu, exists := r.gpus[gpuID]
	if !exists {
		return fmt.Errorf("gpu %s not found", gpuID)
	}

	if jobID != "" {
		gpu.Status = domain.GPUStatusAllocated
		gpu.AllocatedTo = jobID
	} else {
		gpu.Status = domain.GPUStatusAvailable
		gpu.AllocatedTo = ""
	}

	return nil
}
