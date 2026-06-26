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
	TaskTemplates []TaskTemplate `json:"taskTemplates,omitempty"`
	Tasks         []TaskInstance `json:"tasks,omitempty"`
	SharedEnv     []EnvVar       `json:"sharedEnv,omitempty"`
	Logs          []string  `json:"logs,omitempty"`
	ExitCode      *int      `json:"exitCode,omitempty"`
	ErrorMsg      string    `json:"errorMsg,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ReservedEnvVar struct {
	Name        string `json:"name"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description"`
}

type TaskTemplate struct {
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	Replicas     int      `json:"replicas"`
	Image        string   `json:"image,omitempty"`
	Command      string   `json:"command"`
	GPUCount     int      `json:"gpuCount,omitempty"`
	CPUCount     int      `json:"cpuCount,omitempty"`
	MemoryGB     int      `json:"memoryGb,omitempty"`
	StartOrdinal int      `json:"startOrdinal,omitempty"`
	Env          []EnvVar `json:"env,omitempty"`
}

type TaskInstance struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Role          string    `json:"role"`
	Rank          int       `json:"rank"`
	Ordinal       int       `json:"ordinal"`
	Address       string    `json:"address"`
	Image         string    `json:"image,omitempty"`
	Command       string    `json:"command"`
	GPUCount      int       `json:"gpuCount,omitempty"`
	CPUCount      int       `json:"cpuCount,omitempty"`
	MemoryGB      int       `json:"memoryGb,omitempty"`
	AllocatedGPUs []string  `json:"allocatedGpus,omitempty"`
	Status        JobStatus `json:"status"`
	Env           []EnvVar  `json:"env,omitempty"`
	CPUUsage      int       `json:"cpuUsage,omitempty"`
	MemoryUsedMB  int       `json:"memoryUsedMb,omitempty"`
}

// JobStatus represents the state of a job
type JobStatus string

const (
	JobStatusQueued    JobStatus = "Queued"
	JobStatusRunning   JobStatus = "Running"
	JobStatusCompleted JobStatus = "Completed"
	JobStatusFailed    JobStatus = "Failed"
	JobStatusCanceled  JobStatus = "Canceled"
	JobStatusStopped   JobStatus = "Stopped"
)

// JobSubmitRequest represents a job submission request
type JobSubmitRequest struct {
	Name          string         `json:"name"`
	Project       string         `json:"project,omitempty"`
	Queue         string         `json:"queue"`
	Command       string         `json:"command"`
	Image         string         `json:"image,omitempty"`
	GPUCount      int            `json:"gpuCount"`
	TaskTemplates []TaskTemplate `json:"taskTemplates,omitempty"`
	Tasks         []TaskInstance `json:"tasks,omitempty"`
	SharedEnv     []EnvVar       `json:"sharedEnv,omitempty"`
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
