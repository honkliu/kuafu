package domain

import "time"

// Job represents a batch job submission
type Job struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Queue         string    `json:"queue"`
	Command       string    `json:"command"`
	Image         string    `json:"image,omitempty"`
	GPUCount      int       `json:"gpuCount"`
	Status        JobStatus `json:"status"`
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
	Queue    string `json:"queue"`
	Command  string `json:"command"`
	Image    string `json:"image,omitempty"`
	GPUCount int    `json:"gpuCount"`
}
