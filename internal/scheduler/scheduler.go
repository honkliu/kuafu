package scheduler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/policy"
	"github.com/microsoft/kuafu/internal/repository"
)

// Scheduler manages job scheduling and execution simulation
type Scheduler struct {
	repo     repository.Repository
	mu       sync.Mutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewScheduler creates a new scheduler
func NewScheduler(repo repository.Repository) *Scheduler {
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
	queue, err := s.repo.GetQueue(job.Queue)
	if err != nil || queue.Status == domain.QueueStatusPaused {
		return
	}

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
	}
	if err := s.repo.AllocateGPUs(job.AllocatedGPUs, job.ID); err != nil {
		log.Printf("Failed to allocate GPUs to job %s: %v", job.ID, err)
		return
	}

	// Update job status to running
	job.Status = domain.JobStatusRunning
	job.StartedAt = time.Now()
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] Job started with %d GPUs", time.Now().Format("15:04:05"), job.GPUCount))
	if err := s.repo.AddJob(job); err != nil {
		log.Printf("Failed to update job %s after scheduling: %v", job.ID, err)
		s.releaseAllocatedGPUs(job.AllocatedGPUs)
		return
	}

	// Update queue stats
	if queue != nil {
		queue.JobsQueued--
		queue.JobsRunning++
		if err := s.repo.AddQueue(queue); err != nil {
			log.Printf("Failed to update queue %s after scheduling job %s: %v", queue.Name, job.ID, err)
		}
	}

	// Start job execution simulation in background
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.executeJob(job)
	}()
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
	if err := s.repo.AddJob(job); err != nil {
		log.Printf("Failed to append execution logs for job %s: %v", job.ID, err)
	}

	time.Sleep(duration)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Refresh job state
	currentJob, err := s.repo.GetJob(job.ID)
	if err != nil {
		return
	}

	// Check if job was canceled or stopped while the lab executor was sleeping.
	if currentJob.Status != domain.JobStatusRunning {
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
	if err := s.repo.AddJob(currentJob); err != nil {
		log.Printf("Failed to update completed job %s: %v", currentJob.ID, err)
	}

	// Release GPUs
	s.releaseAllocatedGPUs(currentJob.AllocatedGPUs)

	// Update queue stats
	queue, err := s.repo.GetQueue(currentJob.Queue)
	if err == nil {
		if queue.JobsRunning > 0 {
			queue.JobsRunning--
		}
		if err := s.repo.AddQueue(queue); err != nil {
			log.Printf("Failed to update queue %s after completing job %s: %v", queue.Name, currentJob.ID, err)
		}
	}

	log.Printf("Job %s completed with status %s", currentJob.ID, status)
}

func (s *Scheduler) releaseAllocatedGPUs(gpuIDs []string) {
	if err := s.repo.ReleaseGPUs(nonEmptyGPUIDs(gpuIDs)); err != nil {
		log.Printf("Failed to release GPUs %v: %v", gpuIDs, err)
	}
}

func nonEmptyGPUIDs(gpuIDs []string) []string {
	filtered := make([]string, 0, len(gpuIDs))
	for _, gpuID := range gpuIDs {
		if gpuID != "" {
			filtered = append(filtered, gpuID)
		}
	}
	return filtered
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
	if err := s.repo.ReleaseGPUs(job.AllocatedGPUs); err != nil {
		return err
	}

	// Update job status
	job.Status = domain.JobStatusCanceled
	job.CompletedAt = time.Now()
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] Job canceled by user", time.Now().Format("15:04:05")))
	if err := s.repo.AddJob(job); err != nil {
		return err
	}

	// Update queue stats
	queue, err := s.repo.GetQueue(job.Queue)
	if err == nil {
		if wasQueued && queue.JobsQueued > 0 {
			queue.JobsQueued--
		} else if wasRunning && queue.JobsRunning > 0 {
			queue.JobsRunning--
		}
		if err := s.repo.AddQueue(queue); err != nil {
			return err
		}
	}

	return nil
}

func (s *Scheduler) StartJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.repo.GetJob(jobID)
	if err != nil {
		return err
	}

	switch job.Status {
	case domain.JobStatusQueued:
		return nil
	case domain.JobStatusStopped, domain.JobStatusCanceled, domain.JobStatusFailed, domain.JobStatusCompleted:
		queue, err := s.repo.GetQueue(job.Queue)
		if err != nil {
			return err
		}
		if queue.Status == domain.QueueStatusPaused {
			return fmt.Errorf("queue %s is paused", queue.Name)
		}
		jobs, err := s.repo.ListJobs()
		if err != nil {
			return err
		}
		if err := policy.AdmitJob(queue, job, usageForQueueExcluding(queue.Name, jobs, job.ID)); err != nil {
			return err
		}
		if err := s.repo.ReleaseGPUs(nonEmptyGPUIDs(job.AllocatedGPUs)); err != nil {
			return err
		}
		job.Status = domain.JobStatusQueued
		job.RuntimeID = ""
		job.AllocatedGPUs = nil
		job.StartedAt = time.Time{}
		job.CompletedAt = time.Time{}
		job.ExitCode = nil
		job.ErrorMsg = ""
		job.Logs = append(job.Logs, fmt.Sprintf("[%s] Job started by user and returned to queue %s", time.Now().Format("15:04:05"), job.Queue))
		if err := s.repo.AddJob(job); err != nil {
			return err
		}
		queue.JobsQueued++
		return s.repo.AddQueue(queue)
	default:
		return fmt.Errorf("job %s is in state %s and cannot be started", jobID, job.Status)
	}
}

func (s *Scheduler) StopJob(jobID string) error {
	return s.stopJob(jobID, domain.JobStatusStopped, "Job stopped by user")
}

func (s *Scheduler) RestartJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.repo.GetJob(jobID)
	if err != nil {
		return err
	}
	queue, err := s.repo.GetQueue(job.Queue)
	if err != nil {
		return err
	}
	if queue.Status == domain.QueueStatusPaused {
		return fmt.Errorf("queue %s is paused", queue.Name)
	}
	jobs, err := s.repo.ListJobs()
	if err != nil {
		return err
	}
	if err := policy.AdmitJob(queue, job, usageForQueueExcluding(queue.Name, jobs, job.ID)); err != nil {
		return err
	}
	wasQueued := job.Status == domain.JobStatusQueued
	wasRunning := job.Status == domain.JobStatusRunning
	if err := s.repo.ReleaseGPUs(nonEmptyGPUIDs(job.AllocatedGPUs)); err != nil {
		return err
	}
	job.Status = domain.JobStatusQueued
	job.RuntimeID = ""
	job.AllocatedGPUs = nil
	job.StartedAt = time.Time{}
	job.CompletedAt = time.Time{}
	job.ExitCode = nil
	job.ErrorMsg = ""
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] Job restarted by user", time.Now().Format("15:04:05")))
	if err := s.repo.AddJob(job); err != nil {
		return err
	}
	if wasRunning && queue.JobsRunning > 0 {
		queue.JobsRunning--
	}
	if !wasQueued {
		queue.JobsQueued++
	}
	return s.repo.AddQueue(queue)
}

func usageForQueueExcluding(queueName string, jobs []*domain.Job, excludedJobID string) policy.QueueUsage {
	filtered := make([]*domain.Job, 0, len(jobs))
	for _, job := range jobs {
		if job.ID != excludedJobID {
			filtered = append(filtered, job)
		}
	}
	return policy.UsageForQueue(queueName, filtered)
}

func (s *Scheduler) stopJob(jobID string, status domain.JobStatus, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, err := s.repo.GetJob(jobID)
	if err != nil {
		return err
	}
	if job.Status != domain.JobStatusQueued && job.Status != domain.JobStatusRunning {
		return fmt.Errorf("job %s is in state %s and cannot be stopped", jobID, job.Status)
	}
	wasQueued := job.Status == domain.JobStatusQueued
	wasRunning := job.Status == domain.JobStatusRunning

	if err := s.repo.ReleaseGPUs(nonEmptyGPUIDs(job.AllocatedGPUs)); err != nil {
		return err
	}
	job.Status = status
	job.CompletedAt = time.Now()
	job.AllocatedGPUs = nil
	job.Logs = append(job.Logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), message))
	if err := s.repo.AddJob(job); err != nil {
		return err
	}

	queue, err := s.repo.GetQueue(job.Queue)
	if err == nil {
		if wasQueued && queue.JobsQueued > 0 {
			queue.JobsQueued--
		} else if wasRunning && queue.JobsRunning > 0 {
			queue.JobsRunning--
		}
		if err := s.repo.AddQueue(queue); err != nil {
			return err
		}
	}
	return nil
}
