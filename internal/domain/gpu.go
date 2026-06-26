package domain

import "time"

// GPU represents a physical GPU device
type GPU struct {
	ID          string    `json:"id"`
	NodeName    string    `json:"nodeName"`
	Index       int       `json:"index"`
	Model       string    `json:"model"`
	MemoryMB    int       `json:"memoryMb"`
	UUID        string    `json:"uuid,omitempty"`
	Driver      string    `json:"driver,omitempty"`
	Status      GPUStatus `json:"status"`
	AllocatedTo string    `json:"allocatedTo,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type GPUUsageSnapshot struct {
	GeneratedAt time.Time  `json:"generatedAt"`
	Source      string     `json:"source"`
	GPUs        []GPUUsage `json:"gpus"`
	Error       string     `json:"error,omitempty"`
}

type GPUUsage struct {
	Index         int          `json:"index"`
	UUID          string       `json:"uuid,omitempty"`
	NodeName      string       `json:"nodeName,omitempty"`
	Name          string       `json:"name"`
	MemoryTotalMB int          `json:"memoryTotalMb"`
	MemoryUsedMB  int          `json:"memoryUsedMb"`
	Utilization   int          `json:"utilization"`
	PowerWatts    int          `json:"powerWatts"`
	TemperatureC  int          `json:"temperatureC"`
	Processes     []GPUProcess `json:"processes,omitempty"`
}

type GPUProcess struct {
	GPUIndex     int    `json:"gpuIndex"`
	GPUUUID      string `json:"gpuUuid,omitempty"`
	PID          int    `json:"pid"`
	ProcessName  string `json:"processName"`
	UsedMemoryMB int    `json:"usedMemoryMb"`
}

// GPUStatus represents the operational state of a GPU
type GPUStatus string

const (
	GPUStatusAvailable   GPUStatus = "Available"
	GPUStatusAllocated   GPUStatus = "Allocated"
	GPUStatusUnavailable GPUStatus = "Unavailable"
	GPUStatusUnknown     GPUStatus = "Unknown"
)
