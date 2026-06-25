package scheduler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
)

// Scheduler manages job scheduling and execution simulation
type Scheduler struct {
	repo     *repository.MemoryRepository
	mu       sync.Mutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewScheduler creates a new scheduler
func NewScheduler(repo *repository.MemoryRepository) *Scheduler {
	return &Scheduler{
		repo:     repo,
		stopChan: make(chan struct{}),
	}
}

// Start begins the scheduler loop
func (s *Scheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.schedulerLoop(ctx)
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

func (s *Scheduler) schedulerLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.scheduleJobs()
			s.checkRunningJobs()
		}
	}
}

func (s *Scheduler) scheduleJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := s.repo.ListJobs()
	if err != nil {
		return
	}

	// Sort queued jobs by submission time (FIFO)
	var queuedJobs []*domain.Job
	for _, job := range jobs {
		if job.Status == domain.JobStatusQueued {
			queuedJobs = append(queuedJobs, job)
		}
	}

	sort.Slice(queuedJobs, func(i, j int) bool {
		return queuedJobs[i].SubmittedAt.Before(queuedJobs[j].SubmittedAt)
	})

	// Try to schedule each queued job
	for _, job := range queuedJobs {
		s.tryScheduleJob(job)
	}
}

func (s *Scheduler) tryScheduleJob(job *domain.Job) {
	// Find available GPUs
	gpus, err := s.repo.ListGPUs("")
	if err != nil {
		return
	}

	var availableGPUs []*domain.GPU
	for _, gpu := range gpus {
		if gpu.Status == domain.GPUStatusAvailable {
			availableGPUs = append(availableGPUs, gpu)
		}
	}

	// Check if we have enough GPUs
	if len(availableGPUs) < job.GPUCount {
		return
	}

	// Allocate GPUs to job
	allocatedGPUs := availableGPUs[:job.GPUCount]
	job.AllocatedGPUs = make([]string, len(allocatedGPUs))
	for i, gpu := range allocatedGPUs {
		job.AllocatedGPUs[i] = gpu.ID
		s.repo.UpdateGPUAllocation(gpu.ID, job.ID)
	}

	// Update job status to running
	job.Status = domain.JobStatusRunning
	job.StartedAt = time.Now()
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] Job started with %d GPUs", time.Now().Format("15:04:05"), job.GPUCount))
	s.repo.AddJob(job)

	// Update queue stats
	queue, err := s.repo.GetQueue(job.Queue)
	if err == nil {
		queue.JobsQueued--
		queue.JobsRunning++
		s.repo.AddQueue(queue)
	}

	// Start job execution simulation in background
	go s.executeJob(job)
}

func (s *Scheduler) executeJob(job *domain.Job) {
	// Simulate job execution (2-5 seconds for demo)
	duration := 2 * time.Second
	if job.GPUCount > 2 {
		duration = 3 * time.Second
	}

	// Add execution logs
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] Executing command: %s", time.Now().Format("15:04:05"), job.Command))
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] Using image: %s", time.Now().Format("15:04:05"), job.Image))
	s.repo.AddJob(job)

	time.Sleep(duration)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Refresh job state
	currentJob, err := s.repo.GetJob(job.ID)
	if err != nil {
		return
	}

	// Check if job was canceled
	if currentJob.Status == domain.JobStatusCanceled {
		return
	}

	// Complete the job successfully (90% success rate)
	exitCode := 0
	status := domain.JobStatusCompleted

	// Simulate occasional failures
	if time.Now().Unix()%10 == 0 {
		exitCode = 1
		status = domain.JobStatusFailed
		currentJob.ErrorMsg = "simulated failure"
		currentJob.Logs = append(currentJob.Logs, fmt.Sprintf("[%s] ERROR: Command failed", time.Now().Format("15:04:05")))
	} else {
		currentJob.Logs = append(currentJob.Logs, fmt.Sprintf("[%s] Command completed successfully", time.Now().Format("15:04:05")))
	}

	currentJob.Status = status
	currentJob.CompletedAt = time.Now()
	currentJob.ExitCode = &exitCode
	s.repo.AddJob(currentJob)

	// Release GPUs
	for _, gpuID := range currentJob.AllocatedGPUs {
		s.repo.UpdateGPUAllocation(gpuID, "")
	}

	// Update queue stats
	queue, err := s.repo.GetQueue(currentJob.Queue)
	if err == nil {
		if queue.JobsRunning > 0 {
			queue.JobsRunning--
		}
		s.repo.AddQueue(queue)
	}

	log.Printf("Job %s completed with status %s", currentJob.ID, status)
}

func (s *Scheduler) checkRunningJobs() {
	// This method can be extended to check for stuck jobs, timeouts, etc.
	// For now, execution is handled in executeJob goroutines
}

// CancelJob cancels a running or queued job
func (s *Scheduler) CancelJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.repo.GetJob(jobID)
	if err != nil {
		return err
	}

	if job.Status != domain.JobStatusQueued && job.Status != domain.JobStatusRunning {
		return fmt.Errorf("job %s is in state %s and cannot be canceled", jobID, job.Status)
	}

	wasQueued := job.Status == domain.JobStatusQueued
	wasRunning := job.Status == domain.JobStatusRunning

	// Release allocated GPUs if any
	for _, gpuID := range job.AllocatedGPUs {
		s.repo.UpdateGPUAllocation(gpuID, "")
	}

	// Update job status
	job.Status = domain.JobStatusCanceled
	job.CompletedAt = time.Now()
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] Job canceled by user", time.Now().Format("15:04:05")))
	s.repo.AddJob(job)

	// Update queue stats
	queue, err := s.repo.GetQueue(job.Queue)
	if err == nil {
		if wasQueued && queue.JobsQueued > 0 {
			queue.JobsQueued--
		} else if wasRunning && queue.JobsRunning > 0 {
			queue.JobsRunning--
		}
		s.repo.AddQueue(queue)
	}

	return nil
}
