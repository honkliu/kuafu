package scheduler

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/policy"
	"github.com/microsoft/kuafu/internal/repository"
)

// Scheduler manages job scheduling and lab execution.
type Scheduler struct {
	repo     repository.Repository
	mu       sync.Mutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

var runDockerCommand = realDockerCommand

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
	assignGPUsToTasks(job)
	if err := s.repo.AllocateGPUs(job.AllocatedGPUs, job.ID); err != nil {
		log.Printf("Failed to allocate GPUs to job %s: %v", job.ID, err)
		return
	}

	// Update job status to running
	job.Status = domain.JobStatusRunning
	for index := range job.Tasks {
		job.Tasks[index].Status = domain.JobStatusRunning
	}
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

	// Start job execution in background.
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.executeJob(job)
	}()
}

func (s *Scheduler) executeJob(job *domain.Job) {
	result := runJobTasks(job)

	s.mu.Lock()
	defer s.mu.Unlock()

	currentJob, err := s.repo.GetJob(job.ID)
	if err != nil {
		return
	}
	if currentJob.Status != domain.JobStatusRunning {
		return
	}

	currentJob.Logs = append(currentJob.Logs, result.Logs...)
	exitCode := result.ExitCode
	status := domain.JobStatusCompleted
	if exitCode != 0 {
		status = domain.JobStatusFailed
		currentJob.ErrorMsg = result.ErrorMessage
	}

	currentJob.Status = status
	for index := range currentJob.Tasks {
		currentJob.Tasks[index].Status = status
	}
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

type executionResult struct {
	Logs         []string
	ExitCode     int
	ErrorMessage string
}

func runJobTasks(job *domain.Job) executionResult {
	if len(job.Tasks) == 0 {
		return runTask(job.Name, domain.TaskInstance{Name: "job", Role: "job", Rank: 0, Image: job.Image, Command: job.Command})
	}
	result := executionResult{ExitCode: 0}
	for _, task := range job.Tasks {
		taskResult := runTask(job.Name, task)
		result.Logs = append(result.Logs, taskResult.Logs...)
		if taskResult.ExitCode != 0 && result.ExitCode == 0 {
			result.ExitCode = taskResult.ExitCode
			result.ErrorMessage = taskResult.ErrorMessage
		}
	}
	return result
}

func runTask(jobName string, task domain.TaskInstance) executionResult {
	timestamp := time.Now().Format("15:04:05")
	image := task.Image
	if image == "" {
		image = "<image>"
	}
	dockerArgs := dockerRunArgs(jobName, task)
	launch := "docker " + strings.Join(quoteArgsForLog(dockerArgs), " ")
	logs := []string{
		fmt.Sprintf("[%s] Task %s(rank=%d role=%s) image: %s", timestamp, task.Name, task.Rank, task.Role, image),
		fmt.Sprintf("[%s] Task %s docker options: %s", timestamp, task.Name, strings.Join(task.DockerOptions, " ")),
		fmt.Sprintf("[%s] Task %s docker launch: %s", timestamp, task.Name, launch),
		fmt.Sprintf("[%s] Task %s entrypoint script: %s", timestamp, task.Name, emptyDash(task.EntrypointScript)),
		fmt.Sprintf("[%s] Task %s task script: %s", timestamp, task.Name, task.Command),
	}
	stdout, stderr, exitCode, err := runDockerCommand(dockerArgs)
	logs = appendOutputLogs(logs, task.Name, "stdout", stdout)
	logs = appendOutputLogs(logs, task.Name, "stderr", stderr)
	if err != nil {
		logs = append(logs, fmt.Sprintf("[%s] Task %s failed with exit code %d", time.Now().Format("15:04:05"), task.Name, exitCode))
		return executionResult{Logs: logs, ExitCode: exitCode, ErrorMessage: err.Error()}
	}
	logs = append(logs, fmt.Sprintf("[%s] Task %s completed successfully", time.Now().Format("15:04:05"), task.Name))
	return executionResult{Logs: logs, ExitCode: 0}
}

func realDockerCommand(args []string) (string, string, int, error) {
	cmd := exec.Command("docker", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	return stdout.String(), stderr.String(), exitCode, err
}

func appendOutputLogs(logs []string, taskName string, streamName string, output string) []string {
	output = strings.TrimRight(output, "\r\n")
	if output == "" {
		return logs
	}
	for _, line := range strings.Split(output, "\n") {
		logs = append(logs, fmt.Sprintf("[%s] Task %s %s: %s", time.Now().Format("15:04:05"), taskName, streamName, strings.TrimRight(line, "\r")))
	}
	return logs
}

func dockerRunArgs(jobName string, task domain.TaskInstance) []string {
	args := []string{"run", "--rm", "--name", fmt.Sprintf("kuafu-%s-%d", sanitizeDockerName(jobName), task.Rank)}
	args = append(args, expandDockerOptions(task.DockerOptions)...)
	if task.WorkingDirectory != "" {
		args = append(args, "-w", task.WorkingDirectory)
	}
	for _, item := range task.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", item.Name, item.Value))
	}
	image := task.Image
	if image == "" {
		image = "debian:bookworm-slim"
	}
	args = append(args, image, "/bin/bash", "-lc", taskExecutionScript(task))
	return args
}

func taskExecutionScript(task domain.TaskInstance) string {
	parts := []string{"set -e"}
	if strings.TrimSpace(task.EntrypointScript) != "" {
		parts = append(parts, "# kuafu entrypoint", task.EntrypointScript)
	}
	parts = append(parts, "# kuafu task", task.Command)
	return strings.Join(parts, "\n")
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func expandDockerOptions(options []string) []string {
	args := []string{}
	for _, option := range options {
		args = append(args, strings.Fields(option)...)
	}
	return args
}

func quoteArgsForLog(args []string) []string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n'\"") {
			quoted = append(quoted, shellQuote(arg))
		} else {
			quoted = append(quoted, arg)
		}
	}
	return quoted
}

func sanitizeDockerName(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			builder.WriteRune(char)
		} else {
			builder.WriteRune('-')
		}
	}
	result := strings.Trim(builder.String(), "-_.")
	if result == "" {
		return "task"
	}
	return result
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
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

func assignGPUsToTasks(job *domain.Job) {
	if len(job.Tasks) == 0 || len(job.AllocatedGPUs) == 0 {
		return
	}
	nextGPU := 0
	for taskIndex := range job.Tasks {
		count := job.Tasks[taskIndex].GPUCount
		if count <= 0 {
			count = 1
		}
		if nextGPU >= len(job.AllocatedGPUs) {
			return
		}
		end := nextGPU + count
		if end > len(job.AllocatedGPUs) {
			end = len(job.AllocatedGPUs)
		}
		job.Tasks[taskIndex].AllocatedGPUs = append([]string(nil), job.AllocatedGPUs[nextGPU:end]...)
		nextGPU = end
	}
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
	for index := range job.Tasks {
		job.Tasks[index].Status = domain.JobStatusCanceled
	}
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
		for index := range job.Tasks {
			job.Tasks[index].AllocatedGPUs = nil
			job.Tasks[index].Status = domain.JobStatusQueued
		}
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
	for index := range job.Tasks {
		job.Tasks[index].AllocatedGPUs = nil
		job.Tasks[index].Status = domain.JobStatusQueued
	}
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
	for index := range job.Tasks {
		job.Tasks[index].Status = status
		job.Tasks[index].AllocatedGPUs = nil
	}
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
