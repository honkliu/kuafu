package domain

import "time"

// Node represents a physical compute node in the cluster
type Node struct {
	Name          string            `json:"name"`
	Hostname      string            `json:"hostname"`
	PrivateIP     string            `json:"privateIp"`
	InfiniBandIPs []string          `json:"infinibandIps,omitempty"`
	OS            string            `json:"os"`
	Kernel        string            `json:"kernel"`
	CPUCount      int               `json:"cpuCount"`
	MemoryGB      int               `json:"memoryGb"`
	DiskGB        int               `json:"diskGb"`
	GPUCount      int               `json:"gpuCount"`
	Status        NodeStatus        `json:"status"`
	Labels        map[string]string `json:"labels,omitempty"`
	LastHeartbeat time.Time         `json:"lastHeartbeat"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// NodeStatus represents the operational state of a node
type NodeStatus string

const (
	NodeStatusReady       NodeStatus = "Ready"
	NodeStatusNotReady    NodeStatus = "NotReady"
	NodeStatusUnknown     NodeStatus = "Unknown"
	NodeStatusMaintenance NodeStatus = "Maintenance"
)
