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

// GPUStatus represents the operational state of a GPU
type GPUStatus string

const (
	GPUStatusAvailable   GPUStatus = "Available"
	GPUStatusAllocated   GPUStatus = "Allocated"
	GPUStatusUnavailable GPUStatus = "Unavailable"
	GPUStatusUnknown     GPUStatus = "Unknown"
)
