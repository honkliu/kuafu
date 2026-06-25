package repository

import (
	"testing"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
)

func TestMemoryRepository_Nodes(t *testing.T) {
	repo := NewMemoryRepository()

	// Test AddNode
	node := &domain.Node{
		Name:      "test-node",
		Hostname:  "testhost",
		PrivateIP: "10.0.0.1",
		CPUCount:  96,
		MemoryGB:  1740,
		GPUCount:  8,
		Status:    domain.NodeStatusReady,
		CreatedAt: time.Now(),
	}

	if err := repo.AddNode(node); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	// Test GetNode
	retrieved, err := repo.GetNode("test-node")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}
	if retrieved.Name != "test-node" {
		t.Errorf("Expected name 'test-node', got '%s'", retrieved.Name)
	}

	// Test ListNodes
	nodes, err := repo.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	// Test GetNode not found
	_, err = repo.GetNode("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent node")
	}
}

func TestMemoryRepository_GPUs(t *testing.T) {
	repo := NewMemoryRepository()

	// Test AddGPU
	gpu := &domain.GPU{
		ID:        "gpu-0",
		NodeName:  "test-node",
		Index:     0,
		Model:     "A100",
		MemoryMB:  81920,
		Status:    domain.GPUStatusAvailable,
		CreatedAt: time.Now(),
	}

	if err := repo.AddGPU(gpu); err != nil {
		t.Fatalf("AddGPU failed: %v", err)
	}

	// Test GetGPU
	retrieved, err := repo.GetGPU("gpu-0")
	if err != nil {
		t.Fatalf("GetGPU failed: %v", err)
	}
	if retrieved.ID != "gpu-0" {
		t.Errorf("Expected ID 'gpu-0', got '%s'", retrieved.ID)
	}

	// Test ListGPUs
	gpus, err := repo.ListGPUs("")
	if err != nil {
		t.Fatalf("ListGPUs failed: %v", err)
	}
	if len(gpus) != 1 {
		t.Errorf("Expected 1 GPU, got %d", len(gpus))
	}

	// Test ListGPUs with node filter
	gpus, err = repo.ListGPUs("test-node")
	if err != nil {
		t.Fatalf("ListGPUs with filter failed: %v", err)
	}
	if len(gpus) != 1 {
		t.Errorf("Expected 1 GPU for test-node, got %d", len(gpus))
	}

	gpus, err = repo.ListGPUs("other-node")
	if err != nil {
		t.Fatalf("ListGPUs with filter failed: %v", err)
	}
	if len(gpus) != 0 {
		t.Errorf("Expected 0 GPUs for other-node, got %d", len(gpus))
	}

	// Test GetGPU not found
	_, err = repo.GetGPU("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent GPU")
	}
}

func TestMemoryRepository_EmptyName(t *testing.T) {
	repo := NewMemoryRepository()

	// Test node with empty name
	node := &domain.Node{Name: ""}
	if err := repo.AddNode(node); err == nil {
		t.Error("Expected error for node with empty name")
	}

	// Test GPU with empty ID
	gpu := &domain.GPU{ID: ""}
	if err := repo.AddGPU(gpu); err == nil {
		t.Error("Expected error for GPU with empty ID")
	}
}

func TestMemoryRepository_Jobs(t *testing.T) {
	repo := NewMemoryRepository()

	// Test AddJob
	job := &domain.Job{
		ID:          "job-1",
		Name:        "test-job",
		Queue:       "default",
		Command:     "echo hello",
		GPUCount:    2,
		Status:      domain.JobStatusQueued,
		SubmittedAt: time.Now(),
	}

	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	// Test GetJob
	retrieved, err := repo.GetJob("job-1")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if retrieved.ID != "job-1" {
		t.Errorf("Expected ID 'job-1', got '%s'", retrieved.ID)
	}
	if retrieved.Status != domain.JobStatusQueued {
		t.Errorf("Expected status Queued, got %s", retrieved.Status)
	}

	// Test ListJobs
	jobs, err := repo.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jobs))
	}

	// Test DeleteJob
	if err := repo.DeleteJob("job-1"); err != nil {
		t.Fatalf("DeleteJob failed: %v", err)
	}

	jobs, err = repo.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs after delete failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("Expected 0 jobs after delete, got %d", len(jobs))
	}

	// Test DeleteJob not found
	if err := repo.DeleteJob("nonexistent"); err == nil {
		t.Error("Expected error for deleting nonexistent job")
	}

	// Test job with empty ID
	emptyJob := &domain.Job{ID: ""}
	if err := repo.AddJob(emptyJob); err == nil {
		t.Error("Expected error for job with empty ID")
	}
}

func TestMemoryRepository_Queues(t *testing.T) {
	repo := NewMemoryRepository()

	// Test AddQueue
	queue := &domain.Queue{
		Name:     "test-queue",
		MaxGPUs:  8,
		Priority: 100,
	}

	if err := repo.AddQueue(queue); err != nil {
		t.Fatalf("AddQueue failed: %v", err)
	}

	// Test GetQueue
	retrieved, err := repo.GetQueue("test-queue")
	if err != nil {
		t.Fatalf("GetQueue failed: %v", err)
	}
	if retrieved.Name != "test-queue" {
		t.Errorf("Expected name 'test-queue', got '%s'", retrieved.Name)
	}
	if retrieved.MaxGPUs != 8 {
		t.Errorf("Expected MaxGPUs 8, got %d", retrieved.MaxGPUs)
	}

	// Test ListQueues
	queues, err := repo.ListQueues()
	if err != nil {
		t.Fatalf("ListQueues failed: %v", err)
	}
	if len(queues) != 1 {
		t.Errorf("Expected 1 queue, got %d", len(queues))
	}

	// Test GetQueue not found
	_, err = repo.GetQueue("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent queue")
	}

	// Test queue with empty name
	emptyQueue := &domain.Queue{Name: ""}
	if err := repo.AddQueue(emptyQueue); err == nil {
		t.Error("Expected error for queue with empty name")
	}
}

func TestMemoryRepository_UpdateGPUAllocation(t *testing.T) {
	repo := NewMemoryRepository()

	// Add a GPU
	gpu := &domain.GPU{
		ID:        "gpu-0",
		NodeName:  "test-node",
		Status:    domain.GPUStatusAvailable,
		CreatedAt: time.Now(),
	}
	repo.AddGPU(gpu)

	// Test allocating GPU to a job
	if err := repo.UpdateGPUAllocation("gpu-0", "job-1"); err != nil {
		t.Fatalf("UpdateGPUAllocation failed: %v", err)
	}

	updated, _ := repo.GetGPU("gpu-0")
	if updated.Status != domain.GPUStatusAllocated {
		t.Errorf("Expected status Allocated, got %s", updated.Status)
	}
	if updated.AllocatedTo != "job-1" {
		t.Errorf("Expected AllocatedTo 'job-1', got '%s'", updated.AllocatedTo)
	}

	// Test releasing GPU
	if err := repo.UpdateGPUAllocation("gpu-0", ""); err != nil {
		t.Fatalf("UpdateGPUAllocation release failed: %v", err)
	}

	released, _ := repo.GetGPU("gpu-0")
	if released.Status != domain.GPUStatusAvailable {
		t.Errorf("Expected status Available, got %s", released.Status)
	}
	if released.AllocatedTo != "" {
		t.Errorf("Expected AllocatedTo empty, got '%s'", released.AllocatedTo)
	}

	// Test updating nonexistent GPU
	if err := repo.UpdateGPUAllocation("nonexistent", "job-1"); err == nil {
		t.Error("Expected error for updating nonexistent GPU")
	}
}
