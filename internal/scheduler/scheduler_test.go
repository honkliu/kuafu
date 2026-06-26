package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
)

func TestScheduler_JobLifecycle(t *testing.T) {
	repo := repository.NewMemoryRepository()

	// Add GPUs
	for i := 0; i < 4; i++ {
		gpu := &domain.GPU{
			ID:        fmt.Sprintf("gpu-%d", i),
			NodeName:  "test-node",
			Index:     i,
			Status:    domain.GPUStatusAvailable,
			CreatedAt: time.Now(),
		}
		if err := repo.AddGPU(gpu); err != nil {
			t.Fatalf("AddGPU failed: %v", err)
		}
	}

	// Add queue
	queue := &domain.Queue{
		Name:     "test-queue",
		MaxGPUs:  8,
		Priority: 100,
	}
	if err := repo.AddQueue(queue); err != nil {
		t.Fatalf("AddQueue failed: %v", err)
	}

	// Create and start scheduler
	sched := NewScheduler(repo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched.Start(ctx)

	// Submit a job
	job := &domain.Job{
		ID:          "test-job-1",
		Name:        "test",
		Queue:       "test-queue",
		Command:     "echo test",
		GPUCount:    2,
		Status:      domain.JobStatusQueued,
		SubmittedAt: time.Now(),
	}
	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	// Wait for job to start
	time.Sleep(1 * time.Second)

	updatedJob, _ := repo.GetJob("test-job-1")
	if updatedJob.Status != domain.JobStatusRunning {
		t.Errorf("Expected job to be Running, got %s", updatedJob.Status)
	}

	if len(updatedJob.AllocatedGPUs) != 2 {
		t.Errorf("Expected 2 allocated GPUs, got %d", len(updatedJob.AllocatedGPUs))
	}

	// Wait for job to complete
	time.Sleep(3 * time.Second)

	completedJob, _ := repo.GetJob("test-job-1")
	if completedJob.Status != domain.JobStatusCompleted && completedJob.Status != domain.JobStatusFailed {
		t.Errorf("Expected job to be Completed or Failed, got %s", completedJob.Status)
	}

	// Verify GPUs are released
	gpus, _ := repo.ListGPUs("")
	allocatedCount := 0
	for _, gpu := range gpus {
		if gpu.Status == domain.GPUStatusAllocated {
			allocatedCount++
		}
	}
	if allocatedCount != 0 {
		t.Errorf("Expected 0 allocated GPUs after completion, got %d", allocatedCount)
	}

	sched.Stop()
}

func TestScheduler_CancelJob(t *testing.T) {
	repo := repository.NewMemoryRepository()

	// Add GPUs
	for i := 0; i < 2; i++ {
		gpu := &domain.GPU{
			ID:        fmt.Sprintf("gpu-%d", i),
			NodeName:  "test-node",
			Status:    domain.GPUStatusAvailable,
			CreatedAt: time.Now(),
		}
		if err := repo.AddGPU(gpu); err != nil {
			t.Fatalf("AddGPU failed: %v", err)
		}
	}

	// Add queue
	queue := &domain.Queue{
		Name:     "test-queue",
		MaxGPUs:  4,
		Priority: 100,
	}
	if err := repo.AddQueue(queue); err != nil {
		t.Fatalf("AddQueue failed: %v", err)
	}

	// Create scheduler
	sched := NewScheduler(repo)

	// Submit a job
	job := &domain.Job{
		ID:          "test-job-cancel",
		Name:        "test-cancel",
		Queue:       "test-queue",
		Command:     "sleep 100",
		GPUCount:    1,
		Status:      domain.JobStatusQueued,
		SubmittedAt: time.Now(),
	}
	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	// Cancel the job
	if err := sched.CancelJob("test-job-cancel"); err != nil {
		t.Fatalf("CancelJob failed: %v", err)
	}

	canceledJob, _ := repo.GetJob("test-job-cancel")
	if canceledJob.Status != domain.JobStatusCanceled {
		t.Errorf("Expected job to be Canceled, got %s", canceledJob.Status)
	}

	// Try to cancel already canceled job
	if err := sched.CancelJob("test-job-cancel"); err == nil {
		t.Error("Expected error when canceling already canceled job")
	}
}

func TestScheduler_FIFO(t *testing.T) {
	repo := repository.NewMemoryRepository()

	// Add limited GPUs
	for i := 0; i < 2; i++ {
		gpu := &domain.GPU{
			ID:        fmt.Sprintf("gpu-%d", i),
			NodeName:  "test-node",
			Status:    domain.GPUStatusAvailable,
			CreatedAt: time.Now(),
		}
		if err := repo.AddGPU(gpu); err != nil {
			t.Fatalf("AddGPU failed: %v", err)
		}
	}

	// Add queue
	queue := &domain.Queue{
		Name:     "test-queue",
		MaxGPUs:  4,
		Priority: 100,
	}
	if err := repo.AddQueue(queue); err != nil {
		t.Fatalf("AddQueue failed: %v", err)
	}

	// Create and start scheduler
	sched := NewScheduler(repo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched.Start(ctx)

	// Submit two jobs that need 2 GPUs each (only one can run at a time)
	job1 := &domain.Job{
		ID:          "job1",
		Name:        "first",
		Queue:       "test-queue",
		Command:     "echo first",
		GPUCount:    2,
		Status:      domain.JobStatusQueued,
		SubmittedAt: time.Now(),
	}
	if err := repo.AddJob(job1); err != nil {
		t.Fatalf("AddJob job1 failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	job2 := &domain.Job{
		ID:          "job2",
		Name:        "second",
		Queue:       "test-queue",
		Command:     "echo second",
		GPUCount:    2,
		Status:      domain.JobStatusQueued,
		SubmittedAt: time.Now(),
	}
	if err := repo.AddJob(job2); err != nil {
		t.Fatalf("AddJob job2 failed: %v", err)
	}

	// Wait for first job to start
	time.Sleep(1 * time.Second)

	j1, _ := repo.GetJob("job1")
	j2, _ := repo.GetJob("job2")

	// First job should be running, second should still be queued
	if j1.Status != domain.JobStatusRunning {
		t.Errorf("Expected job1 to be Running, got %s", j1.Status)
	}
	if j2.Status != domain.JobStatusQueued {
		t.Errorf("Expected job2 to be Queued, got %s", j2.Status)
	}

	sched.Stop()
}
