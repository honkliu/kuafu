package fixtures

import (
	"fmt"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
)

// SeedTestbedA00A01 populates repository with A00 and A01 testbed data
func SeedTestbedA00A01(addNode func(*domain.Node) error, addGPU func(*domain.GPU) error, addQueue func(*domain.Queue) error, addJob func(*domain.Job) error) error {
	now := time.Now()
	runningJobID := "job-1719248400000000000"

	// Node A00
	a00 := &domain.Node{
		Name:          "A00",
		Hostname:      "hpcdev000000",
		PrivateIP:     "10.0.0.4",
		InfiniBandIPs: []string{"172.16.3.146"},
		OS:            "Ubuntu 22.04.5 LTS",
		Kernel:        "Linux 5.15.0-1088-azure x86_64",
		CPUCount:      96,
		MemoryGB:      1740, // 1.7 TiB
		DiskGB:        497,
		GPUCount:      8,
		Status:        domain.NodeStatusReady,
		Labels: map[string]string{
			"testbed":   "initial",
			"gpu-model": "A100-SXM4-80GB",
		},
		LastHeartbeat: now,
		CreatedAt:     now,
	}

	if err := addNode(a00); err != nil {
		return err
	}

	// Node A01
	a01 := &domain.Node{
		Name:          "A01",
		Hostname:      "hpcdev000001",
		PrivateIP:     "10.0.0.5",
		InfiniBandIPs: []string{"172.16.2.234"},
		OS:            "Ubuntu 22.04.5 LTS",
		Kernel:        "Linux 5.15.0-1088-azure x86_64",
		CPUCount:      96,
		MemoryGB:      1740,
		DiskGB:        497,
		GPUCount:      8,
		Status:        domain.NodeStatusReady,
		Labels: map[string]string{
			"testbed":   "initial",
			"gpu-model": "A100-SXM4-80GB",
		},
		LastHeartbeat: now,
		CreatedAt:     now,
	}

	if err := addNode(a01); err != nil {
		return err
	}

	// Create GPUs for A00
	for i := 0; i < 8; i++ {
		status := domain.GPUStatusAvailable
		allocatedTo := ""
		if i == 0 {
			status = domain.GPUStatusAllocated
			allocatedTo = runningJobID
		}

		gpu := &domain.GPU{
			ID:          fmt.Sprintf("A00-GPU-%d", i),
			NodeName:    "A00",
			Index:       i,
			Model:       "NVIDIA A100-SXM4-80GB",
			MemoryMB:    81920,
			Driver:      "570.133.20",
			Status:      status,
			AllocatedTo: allocatedTo,
			CreatedAt:   now,
		}
		if err := addGPU(gpu); err != nil {
			return err
		}
	}

	// Create GPUs for A01
	for i := 0; i < 8; i++ {
		gpu := &domain.GPU{
			ID:        fmt.Sprintf("A01-GPU-%d", i),
			NodeName:  "A01",
			Index:     i,
			Model:     "NVIDIA A100-SXM4-80GB",
			MemoryMB:  81920,
			Driver:    "570.133.20",
			Status:    domain.GPUStatusAvailable,
			CreatedAt: now,
		}
		if err := addGPU(gpu); err != nil {
			return err
		}
	}

	// Create sample queues
	defaultQueue := &domain.Queue{
		Name:        "default",
		MaxGPUs:     8,
		Priority:    100,
		JobsQueued:  0,
		JobsRunning: 0,
	}
	if err := addQueue(defaultQueue); err != nil {
		return err
	}

	trainingQueue := &domain.Queue{
		Name:        "training",
		MaxGPUs:     16,
		Priority:    200,
		JobsQueued:  1,
		JobsRunning: 0,
	}
	if err := addQueue(trainingQueue); err != nil {
		return err
	}

	inferenceQueue := &domain.Queue{
		Name:        "inference",
		MaxGPUs:     4,
		Priority:    150,
		JobsQueued:  0,
		JobsRunning: 1,
	}
	if err := addQueue(inferenceQueue); err != nil {
		return err
	}

	// Create sample jobs
	job1 := &domain.Job{
		ID:          "job-1719244800000000000",
		Name:        "bert-training",
		Queue:       "training",
		Command:     "python train.py --model bert-base --epochs 10",
		Image:       "pytorch/pytorch:2.0.0-cuda11.7-cudnn8-runtime",
		GPUCount:    4,
		Status:      domain.JobStatusQueued,
		SubmittedAt: now.Add(-1 * time.Hour),
		Logs:        []string{"[12:00:00] Job submitted to queue training"},
	}
	if err := addJob(job1); err != nil {
		return err
	}

	job2 := &domain.Job{
		ID:            runningJobID,
		Name:          "inference-service",
		Queue:         "inference",
		Command:       "python serve.py --model resnet50 --port 8080",
		Image:         "tensorflow/tensorflow:2.13.0-gpu",
		GPUCount:      1,
		Status:        domain.JobStatusRunning,
		SubmittedAt:   now.Add(-30 * time.Minute),
		StartedAt:     now.Add(-25 * time.Minute),
		AllocatedGPUs: []string{"A00-GPU-0"},
		Logs: []string{
			"[12:30:00] Job submitted to queue inference",
			"[12:35:00] Job started on node A00",
			"[12:35:02] Loading model resnet50...",
			"[12:35:05] Model loaded successfully",
			"[12:35:05] Serving on port 8080",
		},
	}
	if err := addJob(job2); err != nil {
		return err
	}

	return nil
}

// SeedLabTenants populates repository with default lab project and users.
func SeedLabTenants(addProject func(*domain.Project) error, addUser func(*domain.User) error) error {
	now := time.Now()
	project := &domain.Project{
		Name:         "lab",
		DisplayName:  "Kuafu Lab",
		DefaultQueue: "default",
		CreatedAt:    now,
	}
	if err := addProject(project); err != nil {
		return err
	}

	users := []*domain.User{
		{
			Alias:       "lab-admin",
			DisplayName: "Lab Admin",
			Memberships: []domain.ProjectMembership{{Project: "lab", Role: domain.ProjectRoleAdmin}},
			CreatedAt:   now,
		},
		{
			Alias:       "lab-user",
			DisplayName: "Lab User",
			Memberships: []domain.ProjectMembership{{Project: "lab", Role: domain.ProjectRoleSubmitter}},
			CreatedAt:   now,
		},
		{
			Alias:       "lab-viewer",
			DisplayName: "Lab Viewer",
			Memberships: []domain.ProjectMembership{{Project: "lab", Role: domain.ProjectRoleViewer}},
			CreatedAt:   now,
		},
	}
	for _, user := range users {
		if err := addUser(user); err != nil {
			return err
		}
	}
	return nil
}
