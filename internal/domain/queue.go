package domain

// Queue represents a job queue with resource limits
type Queue struct {
	Name           string `json:"name"`
	Project        string `json:"project,omitempty"`
	MaxGPUs        int    `json:"maxGpus"`
	SoftGPUs       int    `json:"softGpus,omitempty"`
	MaxGPUsPerJob  int    `json:"maxGpusPerJob,omitempty"`
	MaxQueuedJobs  int    `json:"maxQueuedJobs,omitempty"`
	MaxRunningJobs int    `json:"maxRunningJobs,omitempty"`
	Priority       int    `json:"priority"`
	AllowBurst     bool   `json:"allowBurst,omitempty"`
	JobsQueued     int    `json:"jobsQueued"`
	JobsRunning    int    `json:"jobsRunning"`
}
