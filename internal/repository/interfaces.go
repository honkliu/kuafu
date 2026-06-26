package repository

import "github.com/microsoft/kuafu/internal/domain"

type InventoryRepository interface {
	AddNode(*domain.Node) error
	GetNode(string) (*domain.Node, error)
	ListNodes() ([]*domain.Node, error)
	AddGPU(*domain.GPU) error
	GetGPU(string) (*domain.GPU, error)
	ListGPUs(string) ([]*domain.GPU, error)
}

type JobRepository interface {
	AddJob(*domain.Job) error
	GetJob(string) (*domain.Job, error)
	ListJobs() ([]*domain.Job, error)
	DeleteJob(string) error
}

type QueueRepository interface {
	AddQueue(*domain.Queue) error
	GetQueue(string) (*domain.Queue, error)
	ListQueues() ([]*domain.Queue, error)
}

type TenantRepository interface {
	AddUser(*domain.User) error
	GetUser(string) (*domain.User, error)
	ListUsers() ([]*domain.User, error)
	AddProject(*domain.Project) error
	GetProject(string) (*domain.Project, error)
	ListProjects() ([]*domain.Project, error)
}

type GPUAllocationRepository interface {
	UpdateGPUAllocation(gpuID, jobID string) error
	AllocateGPUs(gpuIDs []string, jobID string) error
	ReleaseGPUs(gpuIDs []string) error
}

type Repository interface {
	InventoryRepository
	JobRepository
	QueueRepository
	TenantRepository
	GPUAllocationRepository
}
