package domain

import "time"

// Job represents a batch job submission
type Job struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Project       string        `json:"project,omitempty"`
	Queue         string        `json:"queue"`
	Command       string        `json:"command"`
	EntrypointScript string    `json:"entrypointScript,omitempty"`
	Image         string        `json:"image,omitempty"`
	GPUCount      int           `json:"gpuCount"`
	Status        JobStatus     `json:"status"`
	RuntimeID     string        `json:"runtimeId,omitempty"`
	SubmittedAt   time.Time     `json:"submittedAt"`
	StartedAt     time.Time     `json:"startedAt,omitempty"`
	CompletedAt   time.Time     `json:"completedAt,omitempty"`
	AllocatedGPUs []string      `json:"allocatedGpus,omitempty"`
	LauncherSpec  *LauncherSpec `json:"launcherSpec,omitempty"`
	TaskTemplates []TaskTemplate `json:"taskTemplates,omitempty"`
	Tasks         []TaskInstance `json:"tasks,omitempty"`
	SharedEnv     []EnvVar       `json:"sharedEnv,omitempty"`
	Logs          []string       `json:"logs,omitempty"`
	ExitCode      *int           `json:"exitCode,omitempty"`
	ErrorMsg      string         `json:"errorMsg,omitempty"`
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
	Name             string   `json:"name" yaml:"name"`
	Role             string   `json:"role" yaml:"role"`
	Replicas         int      `json:"replicas" yaml:"replicas"`
	MinReplicas      int      `json:"minReplicas,omitempty" yaml:"minReplicas,omitempty"`
	MaxReplicas      int      `json:"maxReplicas,omitempty" yaml:"maxReplicas,omitempty"`
	Image            string   `json:"image,omitempty" yaml:"image,omitempty"`
	Command          string   `json:"command" yaml:"command"`
	EntrypointScript string   `json:"entrypointScript,omitempty" yaml:"entrypointScript,omitempty"`
	WorkingDirectory string   `json:"workingDirectory,omitempty" yaml:"workingDirectory,omitempty"`
	DockerOptions    []string `json:"dockerOptions,omitempty" yaml:"dockerOptions,omitempty"`
	GPUCount         int      `json:"gpuCount,omitempty" yaml:"gpuCount,omitempty"`
	CPUCount         int      `json:"cpuCount,omitempty" yaml:"cpuCount,omitempty"`
	MemoryGB         int      `json:"memoryGb,omitempty" yaml:"memoryGb,omitempty"`
	StartOrdinal     int      `json:"startOrdinal,omitempty" yaml:"startOrdinal,omitempty"`
	Env              []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`
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
	EntrypointScript string `json:"entrypointScript,omitempty"`
	WorkingDirectory string    `json:"workingDirectory,omitempty"`
	DockerOptions    []string  `json:"dockerOptions,omitempty"`
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
	EntrypointScript string     `json:"entrypointScript,omitempty"`
	Image         string         `json:"image,omitempty"`
	GPUCount      int            `json:"gpuCount"`
	LauncherSpec  *LauncherSpec  `json:"launcherSpec,omitempty"`
	TaskTemplates []TaskTemplate `json:"taskTemplates,omitempty"`
	Tasks         []TaskInstance `json:"tasks,omitempty"`
	SharedEnv     []EnvVar       `json:"sharedEnv,omitempty"`
}

type LauncherSpec struct {
	Name         string           `json:"name,omitempty" yaml:"name,omitempty"`
	Project      string           `json:"project,omitempty" yaml:"project,omitempty"`
	Queue        string           `json:"queue,omitempty" yaml:"queue,omitempty"`
	Description  string           `json:"description,omitempty" yaml:"description,omitempty"`
	Dependencies []string         `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	ReplicaPolicy string          `json:"replicaPolicy,omitempty" yaml:"replicaPolicy,omitempty"`
	WorkingDirectory string       `json:"workingDirectory,omitempty" yaml:"workingDirectory,omitempty"`
	EntrypointScript string       `json:"entrypointScript,omitempty" yaml:"entrypointScript,omitempty"`
	Docker       LauncherDocker   `json:"docker,omitempty" yaml:"docker,omitempty"`
	Env          []EnvVar         `json:"env,omitempty" yaml:"env,omitempty"`
	Tasks        []TaskTemplate   `json:"tasks,omitempty" yaml:"tasks,omitempty"`
	APIVersion   string           `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind         string           `json:"kind,omitempty" yaml:"kind,omitempty"`
	Metadata     LauncherMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec         LauncherJobSpec  `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type LauncherMetadata struct {
	Name    string            `json:"name" yaml:"name"`
	Project string            `json:"project,omitempty" yaml:"project,omitempty"`
	Labels  map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type LauncherJobSpec struct {
	Queue            string          `json:"queue" yaml:"queue"`
	ReplicaPolicy    string          `json:"replicaPolicy,omitempty" yaml:"replicaPolicy,omitempty"`
	WorkingDirectory string          `json:"workingDirectory,omitempty" yaml:"workingDirectory,omitempty"`
	EntrypointScript string          `json:"entrypointScript,omitempty" yaml:"entrypointScript,omitempty"`
	Docker           LauncherDocker  `json:"docker,omitempty" yaml:"docker,omitempty"`
	Env              []EnvVar        `json:"env,omitempty" yaml:"env,omitempty"`
	Tasks            []TaskTemplate  `json:"tasks" yaml:"tasks"`
}

type LauncherDocker struct {
	Image   string   `json:"image,omitempty" yaml:"image,omitempty"`
	Options []string `json:"options,omitempty" yaml:"options,omitempty"`
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
