package domain

// Queue represents a job queue with resource limits
type Queue struct {
	Name        string `json:"name"`
	MaxGPUs     int    `json:"maxGpus"`
	Priority    int    `json:"priority"`
	JobsQueued  int    `json:"jobsQueued"`
	JobsRunning int    `json:"jobsRunning"`
}
