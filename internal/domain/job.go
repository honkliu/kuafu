package domain

import "time"

// Job represents a batch job submission
type Job struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Project       string    `json:"project,omitempty"`
	Queue         string    `json:"queue"`
	Command       string    `json:"command"`
	Image         string    `json:"image,omitempty"`
	GPUCount      int       `json:"gpuCount"`
	Status        JobStatus `json:"status"`
	RuntimeID     string    `json:"runtimeId,omitempty"`
	SubmittedAt   time.Time `json:"submittedAt"`
	StartedAt     time.Time `json:"startedAt,omitempty"`
	CompletedAt   time.Time `json:"completedAt,omitempty"`
	AllocatedGPUs []string  `json:"allocatedGpus,omitempty"`
	Logs          []string  `json:"logs,omitempty"`
	ExitCode      *int      `json:"exitCode,omitempty"`
	ErrorMsg      string    `json:"errorMsg,omitempty"`
}

// JobStatus represents the state of a job
type JobStatus string

const (
	JobStatusQueued    JobStatus = "Queued"
	JobStatusRunning   JobStatus = "Running"
	JobStatusCompleted JobStatus = "Completed"
	JobStatusFailed    JobStatus = "Failed"
	JobStatusCanceled  JobStatus = "Canceled"
)

// JobSubmitRequest represents a job submission request
type JobSubmitRequest struct {
	Name     string `json:"name"`
	Project  string `json:"project,omitempty"`
	Queue    string `json:"queue"`
	Command  string `json:"command"`
	Image    string `json:"image,omitempty"`
	GPUCount int    `json:"gpuCount"`
}

// RuntimeJobSpec is the scheduler-adapter-facing representation of a Kuafu job.
type RuntimeJobSpec struct {
	JobID    string            `json:"jobId"`
	Name     string            `json:"name"`
	Project  string            `json:"project,omitempty"`
	Queue    string            `json:"queue"`
	Image    string            `json:"image,omitempty"`
	Command  []string          `json:"command"`
	GPUCount int               `json:"gpuCount"`
	Labels   map[string]string `json:"labels,omitempty"`
}

func (j *Job) RuntimeSpec() RuntimeJobSpec {
	return RuntimeJobSpec{
		JobID:    j.ID,
		Name:     j.Name,
		Project:  j.Project,
		Queue:    j.Queue,
		Image:    j.Image,
		Command:  []string{j.Command},
		GPUCount: j.GPUCount,
		Labels: map[string]string{
			"kuafu.dev/job-id":  j.ID,
			"kuafu.dev/project": j.Project,
			"kuafu.dev/queue":   j.Queue,
		},
	}
}
