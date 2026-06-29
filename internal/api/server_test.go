package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/kuafu/internal/auth"
	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
	"github.com/microsoft/kuafu/internal/scheduler"
	"github.com/microsoft/kuafu/pkg/fixtures"
)

func setupTestServer(t *testing.T) *Server {
	repo := repository.NewMemoryRepository()
	if err := fixtures.SeedTestbedA00A01(repo.AddNode, repo.AddGPU, repo.AddQueue, repo.AddJob); err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}
	if err := fixtures.SeedLabTenants(repo.AddProject, repo.AddUser); err != nil {
		t.Fatalf("Failed to seed lab tenants: %v", err)
	}

	sched := scheduler.NewScheduler(repo)
	server := NewServer(repo, sched, ":8080")
	server.SetAuthenticator(auth.HeaderAuthenticator{Users: repo})
	return server
}

func TestHealth(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

func TestReady(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response["ready"] {
		t.Error("Expected ready to be true")
	}
}

func TestListNodes(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	w := httptest.NewRecorder()

	server.handleNodes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Nodes []*domain.Node `json:"nodes"`
		Count int            `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 2 {
		t.Errorf("Expected 2 nodes, got %d", response.Count)
	}
}

func TestGetNode(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/A00", nil)
	w := httptest.NewRecorder()

	server.handleNode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var node domain.Node
	if err := json.NewDecoder(w.Body).Decode(&node); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if node.Name != "A00" {
		t.Errorf("Expected node name 'A00', got '%s'", node.Name)
	}
	if node.GPUCount != 8 {
		t.Errorf("Expected 8 GPUs, got %d", node.GPUCount)
	}
}

func TestGetNodeNotFound(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/nonexistent", nil)
	w := httptest.NewRecorder()

	server.handleNode(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestListGPUs(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus", nil)
	w := httptest.NewRecorder()

	server.handleGPUs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		GPUs  []*domain.GPU `json:"gpus"`
		Count int           `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 16 {
		t.Errorf("Expected 16 GPUs, got %d", response.Count)
	}
}

func TestGPUTelemetryFallback(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpu-telemetry", nil)
	w := httptest.NewRecorder()

	server.handleGPUTelemetry(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var response domain.GPUUsageSnapshot
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(response.GPUs) == 0 {
		t.Fatalf("expected telemetry fallback GPUs, got %#v", response)
	}
	if response.Source == "" {
		t.Fatalf("expected telemetry source, got %#v", response)
	}
}

func TestGPUTelemetryMergesLiveAndRepositoryInventory(t *testing.T) {
	server := setupTestServer(t)
	snapshot := &domain.GPUUsageSnapshot{
		GeneratedAt: time.Now(),
		Source:      "nvidia-smi",
		GPUs:        []domain.GPUUsage{{Index: 0, NodeName: "A00", Name: "NVIDIA A100-SXM4-80GB", MemoryTotalMB: 81920, MemoryUsedMB: 1024, Utilization: 77}},
	}
	merged := mergeRepositoryTelemetry(server.repo, snapshot)
	if merged.Source != "nvidia-smi+repository" {
		t.Fatalf("expected merged source, got %s", merged.Source)
	}
	if len(merged.GPUs) != 16 {
		t.Fatalf("expected all A00/A01 GPUs after merge, got %d", len(merged.GPUs))
	}
	foundA01 := false
	for _, gpu := range merged.GPUs {
		foundA01 = foundA01 || gpu.NodeName == "A01"
	}
	if !foundA01 {
		t.Fatalf("expected A01 repository GPUs in merged telemetry: %#v", merged.GPUs)
	}
}

func TestReservedEnvCatalog(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/reserved-env", nil)
	w := httptest.NewRecorder()

	server.handleReservedEnv(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var response struct {
		ReservedEnv []domain.ReservedEnvVar `json:"reservedEnv"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(response.ReservedEnv) == 0 {
		t.Fatalf("expected reserved env catalog")
	}
	foundRank := false
	foundTask0 := false
	for _, item := range response.ReservedEnv {
		foundRank = foundRank || item.Name == "RANK"
		foundTask0 = foundTask0 || item.Name == "TASK0_ADDRESS"
	}
	if !foundRank || !foundTask0 {
		t.Fatalf("reserved env catalog missing RANK or TASK0_ADDRESS: %#v", response.ReservedEnv)
	}
}

func TestGetGPU(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/gpus/A00-GPU-0", nil)
	w := httptest.NewRecorder()

	server.handleGPU(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var gpu domain.GPU
	if err := json.NewDecoder(w.Body).Decode(&gpu); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if gpu.ID != "A00-GPU-0" {
		t.Errorf("Expected GPU ID 'A00-GPU-0', got '%s'", gpu.ID)
	}
	if gpu.NodeName != "A00" {
		t.Errorf("Expected node 'A00', got '%s'", gpu.NodeName)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nodes", nil)
	w := httptest.NewRecorder()

	server.handleNodes(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestSubmitJob(t *testing.T) {
	server := setupTestServer(t)

	reqBody := domain.JobSubmitRequest{
		Name:     "test-job",
		Queue:    "default",
		Command:  "echo hello",
		GPUCount: 1,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var job domain.Job
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if job.Name != "test-job" {
		t.Errorf("Expected job name 'test-job', got '%s'", job.Name)
	}
	if job.Status != domain.JobStatusQueued {
		t.Errorf("Expected status Queued, got %s", job.Status)
	}
	if len(job.Tasks) != 2 || job.Tasks[0].Role != "master" || job.Tasks[1].Role != "worker" {
		t.Fatalf("expected command job to normalize to master/worker task plan, got %#v", job.Tasks)
	}
	if job.GPUCount != 1 || job.Tasks[0].GPUCount != 1 || job.Tasks[1].GPUCount != 0 {
		t.Fatalf("expected GPU demand to stay 1 with zero-GPU worker control task, got job=%d tasks=%#v", job.GPUCount, job.Tasks)
	}
}

func TestSubmitDistributedJobRendersTaskReservedEnv(t *testing.T) {
	server := setupTestServer(t)
	reqBody := domain.JobSubmitRequest{
		Name:     "glm52-dist",
		Queue:    "default",
		Command:  "sglang serve",
		GPUCount: 2,
		SharedEnv: []domain.EnvVar{{Name: "MODEL_PATH", Value: "zai-org/GLM-5.2-FP8"}},
		TaskTemplates: []domain.TaskTemplate{
			{Name: "master", Role: "master", Replicas: 1, Command: "sglang serve --model-path $MODEL_PATH --node-rank $RANK --dist-init-addr $TASK0_ADDRESS:20000", GPUCount: 1},
			{Name: "worker", Role: "worker", Replicas: 1, Command: "sglang serve --model-path $MODEL_PATH --node-rank $RANK --dist-init-addr $TASK0_ADDRESS:20000", GPUCount: 1},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	var job domain.Job
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(job.Tasks) != 2 {
		t.Fatalf("expected 2 rendered tasks, got %#v", job.Tasks)
	}
	if job.Tasks[0].Rank != 0 || job.Tasks[1].Rank != 1 {
		t.Fatalf("expected ranks 0 and 1, got %#v", job.Tasks)
	}
	if !strings.Contains(job.Tasks[0].Command, "--node-rank 0") || !strings.Contains(job.Tasks[1].Command, "--node-rank 1") {
		t.Fatalf("commands did not render rank variables: %#v", job.Tasks)
	}
	if !strings.Contains(job.Tasks[0].Command, "glm52-dist-task-0:20000") || !strings.Contains(job.Tasks[1].Command, "glm52-dist-task-0:20000") {
		t.Fatalf("commands did not render shared master address: %#v", job.Tasks)
	}
}

func TestSubmitLauncherSpecRendersAndPersistsTaskPlan(t *testing.T) {
	server := setupTestServer(t)
	reqBody := domain.JobSubmitRequest{LauncherSpec: &domain.LauncherSpec{
		Name:             "mpi-aout",
		Project:          "lab",
		Queue:            "training",
		ReplicaPolicy:    "fixed",
		WorkingDirectory: "/workspace/mpi",
		EntrypointScript: "export MODEL_PATH=$MODEL_PATH\nexport MASTER=$TASK0_ADDRESS",
		Dependencies:     []string{"dataset://imagenet-v1", "module://openmpi"},
		Docker:           domain.LauncherDocker{Image: "mpi/openmpi:latest", Options: []string{"--network=host", "--ipc=host"}},
		Env:              []domain.EnvVar{{Name: "OMPI_MCA_btl", Value: "tcp,self"}, {Name: "MODEL_PATH", Value: "/models/glm"}},
		Tasks: []domain.TaskTemplate{
			{Name: "master", Role: "master", Replicas: 1, Command: "./a.out -p 0 --master $TASK0_ADDRESS --world-size $WORLD_SIZE", GPUCount: 4, CPUCount: 32, MemoryGB: 256},
			{Name: "slave", Role: "slave", Replicas: 1, Command: "./a.out -p $TASK_RANK --master $TASK0_ADDRESS --world-size $WORLD_SIZE", GPUCount: 4, CPUCount: 32, MemoryGB: 256},
		},
	}}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set("x-kuafu-user", "lab-admin")
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	var job domain.Job
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if job.Name != "mpi-aout" || job.Queue != "training" || job.Image != "mpi/openmpi:latest" {
		t.Fatalf("launcher spec did not populate job fields: %#v", job)
	}
	if job.LauncherSpec == nil || job.LauncherSpec.WorkingDirectory != "/workspace/mpi" || len(job.LauncherSpec.Dependencies) != 2 {
		t.Fatalf("launcher spec was not persisted: %#v", job.LauncherSpec)
	}
	if len(job.Tasks) != 2 || job.GPUCount != 8 {
		t.Fatalf("expected two 4-GPU tasks and 8 total GPUs, got job=%d tasks=%#v", job.GPUCount, job.Tasks)
	}
	if job.Tasks[0].WorkingDirectory != "/workspace/mpi" || len(job.Tasks[0].DockerOptions) != 2 {
		t.Fatalf("expected task launcher fields, got %#v", job.Tasks[0])
	}
	if !strings.Contains(job.Tasks[0].Command, "./a.out -p 0") || !strings.Contains(job.Tasks[1].Command, "./a.out -p 1") || !strings.Contains(job.Tasks[1].Command, "mpi-aout-task-0") {
		t.Fatalf("expected rendered worker launcher command, got %#v", job.Tasks[1].Command)
	}
	if !strings.Contains(job.Tasks[0].EntrypointScript, "export MODEL_PATH=$MODEL_PATH") || !strings.Contains(job.Tasks[0].EntrypointScript, "export MASTER=mpi-aout-task-0") {
		t.Fatalf("expected rendered entrypoint setup before task, got %#v", job.Tasks[0].EntrypointScript)
	}
	if strings.Contains(job.Tasks[0].Command, "MODEL_PATH") || strings.Contains(job.Tasks[0].Command, "export MASTER") {
		t.Fatalf("entrypoint leaked into task script: %#v", job.Tasks[0].Command)
	}
}

func TestSubmitSingleNodeSGLangLauncherCommand(t *testing.T) {
	server := setupTestServer(t)
	command := "python -m sglang.launch_server --model-path /mnt/nvme_raid0/Qwen3-Next-80B-A3B-Instruct --tp-size 8 --mem-fraction-static 0.8 --context-length 262144 --reasoning-parser qwen3 --tool-call-parser qwen3_coder --host 0.0.0.0 --api-key san+3dy9 --port 8000 > /mnt/nvme_raid0/gitroot/qwen80b.log 2>&1 &"
	reqBody := domain.JobSubmitRequest{LauncherSpec: &domain.LauncherSpec{
		Name:             "qwen80b-sglang",
		Queue:            "default",
		ReplicaPolicy:    "fixed",
		WorkingDirectory: "/mnt/nvme_raid0/gitroot",
		Docker:           domain.LauncherDocker{Image: "lmsysorg/sglang:latest", Options: []string{"--gpus all", "--network=host", "--ipc=host", "--shm-size=32g", "-v /mnt/nvme_raid0:/mnt/nvme_raid0"}},
		Tasks: []domain.TaskTemplate{
			{Name: "server", Role: "server", Replicas: 1, Command: command, GPUCount: 8, CPUCount: 32, MemoryGB: 256},
		},
	}}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	var job domain.Job
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(job.Tasks) != 1 || job.Tasks[0].Command != command {
		t.Fatalf("expected exact sglang command to be preserved, got %#v", job.Tasks)
	}
	if job.GPUCount != 8 || job.Tasks[0].GPUCount != 8 {
		t.Fatalf("expected single 8-GPU server task, got job=%d task=%d", job.GPUCount, job.Tasks[0].GPUCount)
	}
}

func TestSubmitDistributedJobRejectsReservedEnvOverride(t *testing.T) {
	server := setupTestServer(t)
	reqBody := domain.JobSubmitRequest{
		Name:      "bad-env",
		Queue:     "default",
		Command:   "echo no",
		GPUCount:  1,
		SharedEnv: []domain.EnvVar{{Name: "RANK", Value: "7"}},
		TaskTemplates: []domain.TaskTemplate{
			{Name: "worker", Role: "worker", Replicas: 1, Command: "echo $RANK", GPUCount: 1},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "reserved environment variable") {
		t.Fatalf("expected reserved env error, got %s", w.Body.String())
	}
}

func TestSubmitDistributedJobAcceptsReviewedTaskCommands(t *testing.T) {
	server := setupTestServer(t)
	reqBody := domain.JobSubmitRequest{
		Name:     "reviewed-dist",
		Queue:    "default",
		Command:  "sglang serve",
		GPUCount: 2,
		TaskTemplates: []domain.TaskTemplate{
			{Name: "master", Role: "master", Replicas: 1, Command: "rank $RANK", GPUCount: 1},
			{Name: "worker", Role: "worker", Replicas: 1, Command: "rank $RANK", GPUCount: 1},
		},
		Tasks: []domain.TaskInstance{
			{Rank: 0, Command: "reviewed master --node-rank 0"},
			{Rank: 1, Command: "reviewed worker --node-rank 1"},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	var job domain.Job
	if err := json.NewDecoder(w.Body).Decode(&job); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if job.Tasks[0].Command != "reviewed master --node-rank 0" || job.Tasks[1].Command != "reviewed worker --node-rank 1" {
		t.Fatalf("reviewed commands were not preserved: %#v", job.Tasks)
	}
}

func TestSubmitJobRejectsOverQuotaBeforePersisting(t *testing.T) {
	repo := repository.NewMemoryRepository()
	queue := &domain.Queue{Name: "limited", MaxGPUs: 4}
	if err := repo.AddQueue(queue); err != nil {
		t.Fatalf("AddQueue failed: %v", err)
	}
	runningJob := &domain.Job{ID: "running", Name: "running", Queue: "limited", Command: "sleep", GPUCount: 4, Status: domain.JobStatusRunning}
	if err := repo.AddJob(runningJob); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	server := NewServer(repo, scheduler.NewScheduler(repo), ":8080")

	reqBody := domain.JobSubmitRequest{Name: "too-big", Queue: "limited", Command: "echo no", GPUCount: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	jobs, err := repo.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected rejected job not to be persisted, jobs=%#v", jobs)
	}
}

func TestSubmitJobRequiresProjectSubmitRole(t *testing.T) {
	repo := repository.NewMemoryRepository()
	queue := &domain.Queue{Name: "training", Project: "vision", MaxGPUs: 8}
	if err := repo.AddQueue(queue); err != nil {
		t.Fatalf("AddQueue failed: %v", err)
	}
	user := &domain.User{Alias: "viewer", Memberships: []domain.ProjectMembership{{Project: "vision", Role: domain.ProjectRoleViewer}}}
	if err := repo.AddUser(user); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	server := NewServer(repo, scheduler.NewScheduler(repo), ":8080")
	server.SetAuthenticator(auth.HeaderAuthenticator{Users: repo})

	reqBody := domain.JobSubmitRequest{Name: "blocked", Project: "vision", Queue: "training", Command: "echo no", GPUCount: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set(auth.UserHeader, "viewer")
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestSubmitJobAllowsProjectSubmitter(t *testing.T) {
	repo := repository.NewMemoryRepository()
	queue := &domain.Queue{Name: "training", Project: "vision", MaxGPUs: 8}
	if err := repo.AddQueue(queue); err != nil {
		t.Fatalf("AddQueue failed: %v", err)
	}
	user := &domain.User{Alias: "submitter", Memberships: []domain.ProjectMembership{{Project: "vision", Role: domain.ProjectRoleSubmitter}}}
	if err := repo.AddUser(user); err != nil {
		t.Fatalf("AddUser failed: %v", err)
	}
	server := NewServer(repo, scheduler.NewScheduler(repo), ":8080")
	server.SetAuthenticator(auth.HeaderAuthenticator{Users: repo})

	reqBody := domain.JobSubmitRequest{Name: "allowed", Project: "vision", Queue: "training", Command: "echo yes", GPUCount: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set(auth.UserHeader, "submitter")
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestSubmitJobAllowsSeededLabUser(t *testing.T) {
	server := setupTestServer(t)

	reqBody := domain.JobSubmitRequest{Name: "lab-project-job", Project: "lab", Queue: "default", Command: "echo lab", GPUCount: 1}
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))
	req.Header.Set(auth.UserHeader, "lab-user")
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
}

func TestListJobs(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()

	server.handleJobs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Jobs  []*domain.Job `json:"jobs"`
		Count int           `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func TestListQueues(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/queues", nil)
	w := httptest.NewRecorder()

	server.handleQueues(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Queues []*domain.Queue `json:"queues"`
		Count  int             `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count < 1 {
		t.Errorf("Expected at least 1 queue, got %d", response.Count)
	}
}

func TestGetQueue(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/queues/default", nil)
	w := httptest.NewRecorder()

	server.handleQueue(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var queue domain.Queue
	if err := json.NewDecoder(w.Body).Decode(&queue); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if queue.Name != "default" {
		t.Errorf("Expected queue name 'default', got '%s'", queue.Name)
	}
}

func TestJobStartStopRestartActions(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/job-1719244800000000000/stop", nil)
	w := httptest.NewRecorder()
	server.handleJobDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected stop status 200, got %d: %s", w.Code, w.Body.String())
	}
	var stopped domain.Job
	if err := json.NewDecoder(w.Body).Decode(&stopped); err != nil {
		t.Fatalf("Decode stopped job failed: %v", err)
	}
	if stopped.Status != domain.JobStatusStopped {
		t.Fatalf("expected stopped job, got %#v", stopped)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+stopped.ID+"/start", nil)
	w = httptest.NewRecorder()
	server.handleJobDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected start status 200, got %d: %s", w.Code, w.Body.String())
	}
	var started domain.Job
	if err := json.NewDecoder(w.Body).Decode(&started); err != nil {
		t.Fatalf("Decode started job failed: %v", err)
	}
	if started.Status != domain.JobStatusQueued {
		t.Fatalf("expected queued job after start, got %#v", started)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/jobs/job-1719248400000000000/restart", nil)
	w = httptest.NewRecorder()
	server.handleJobDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected restart status 200, got %d: %s", w.Code, w.Body.String())
	}
	var restarted domain.Job
	if err := json.NewDecoder(w.Body).Decode(&restarted); err != nil {
		t.Fatalf("Decode restarted job failed: %v", err)
	}
	if restarted.Status != domain.JobStatusQueued || len(restarted.AllocatedGPUs) != 0 {
		t.Fatalf("expected restarted queued job without GPUs, got %#v", restarted)
	}
}

func TestCreateUpdateDeleteQueue(t *testing.T) {
	server := setupTestServer(t)

	queue := domain.Queue{Name: "research", Status: domain.QueueStatusActive, Project: "lab", MaxGPUs: 12, SoftGPUs: 8, MaxGPUsPerJob: 4, MaxQueuedJobs: 20, MaxRunningJobs: 4, Priority: 250, AllowBurst: true}
	body, err := json.Marshal(queue)
	if err != nil {
		t.Fatalf("Marshal queue failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queues", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleQueues(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", w.Code, w.Body.String())
	}

	queue.Status = domain.QueueStatusPaused
	queue.MaxGPUs = 10
	body, err = json.Marshal(queue)
	if err != nil {
		t.Fatalf("Marshal update failed: %v", err)
	}
	req = httptest.NewRequest(http.MethodPut, "/api/v1/queues/research", bytes.NewReader(body))
	w = httptest.NewRecorder()
	server.handleQueue(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", w.Code, w.Body.String())
	}
	var updated domain.Queue
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("Decode queue failed: %v", err)
	}
	if updated.Status != domain.QueueStatusPaused || updated.MaxGPUs != 10 {
		t.Fatalf("unexpected updated queue: %#v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/queues/research", nil)
	w = httptest.NewRecorder()
	server.handleQueue(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteQueueRejectsActiveJobs(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/queues/training", nil)
	w := httptest.NewRecorder()
	server.handleQueue(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected conflict deleting queue with active jobs, got %d", w.Code)
	}
}

func TestCreateReservationAndCommand(t *testing.T) {
	server := setupTestServer(t)

	createReq := domain.ReservationCreateRequest{
		Name:          "debug-session",
		Owner:         "lab-user",
		NodeNames:     []string{"A00", "A01"},
		DurationHours: 4,
	}
	body, err := json.Marshal(createReq)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleReservations(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	var reservation domain.NodeReservation
	if err := json.NewDecoder(w.Body).Decode(&reservation); err != nil {
		t.Fatalf("Decode reservation failed: %v", err)
	}
	if len(reservation.NodeNames) != 2 || reservation.Status != domain.ReservationStatusActive {
		t.Fatalf("unexpected reservation: %#v", reservation)
	}

	commandReq := domain.ReservationCommandRequest{
		Image:            "nvcr.io/nvidia/pytorch:24.05-py3",
		DockerRunOptions: "--gpus all --ipc=host --ulimit memlock=-1",
		EntryPoint:       "/bin/bash",
		WorkingDirectory: "/workspace",
		Environment:      []string{"NCCL_DEBUG=INFO"},
		StartCommand:     "-lc 'nvidia-smi && torchrun train.py'",
	}
	body, err = json.Marshal(commandReq)
	if err != nil {
		t.Fatalf("Marshal command failed: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/reservations/"+reservation.ID+"/commands", bytes.NewReader(body))
	w = httptest.NewRecorder()
	server.handleReservationDetail(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected command status 201, got %d: %s", w.Code, w.Body.String())
	}
	var command domain.ReservationCommand
	if err := json.NewDecoder(w.Body).Decode(&command); err != nil {
		t.Fatalf("Decode command failed: %v", err)
	}
	if len(command.RenderedCommands) != 2 {
		t.Fatalf("expected rendered command per reserved node, got %#v", command.RenderedCommands)
	}
	if !strings.Contains(command.RenderedCommands[0].Command, "--gpus all") {
		t.Fatalf("expected docker options in rendered command, got %s", command.RenderedCommands[0].Command)
	}
}

func TestCreateReservationCommandAcceptsReviewedCommandOverrides(t *testing.T) {
	server := setupTestServer(t)

	createReq := domain.ReservationCreateRequest{Name: "reviewed", NodeNames: []string{"A00", "A01"}}
	body, err := json.Marshal(createReq)
	if err != nil {
		t.Fatalf("Marshal reservation failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleReservations(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected reservation create 201, got %d: %s", w.Code, w.Body.String())
	}
	var reservation domain.NodeReservation
	if err := json.NewDecoder(w.Body).Decode(&reservation); err != nil {
		t.Fatalf("Decode reservation failed: %v", err)
	}

	commandReq := domain.ReservationCommandRequest{
		Image:        "ubuntu:22.04",
		StartCommand: "bash -lc 'echo default'",
		RenderedCommands: []domain.ReservationNodeCommand{
			{NodeName: "A00", Command: "docker run reviewed-a00"},
			{NodeName: "A01", Command: "docker run reviewed-a01"},
		},
	}
	body, err = json.Marshal(commandReq)
	if err != nil {
		t.Fatalf("Marshal command failed: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/reservations/"+reservation.ID+"/commands", bytes.NewReader(body))
	w = httptest.NewRecorder()
	server.handleReservationDetail(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected command create 201, got %d: %s", w.Code, w.Body.String())
	}
	var command domain.ReservationCommand
	if err := json.NewDecoder(w.Body).Decode(&command); err != nil {
		t.Fatalf("Decode command failed: %v", err)
	}
	if command.RenderedCommands[0].Command != "docker run reviewed-a00" || command.RenderedCommands[1].Command != "docker run reviewed-a01" {
		t.Fatalf("expected reviewed command overrides, got %#v", command.RenderedCommands)
	}
}

func TestCreateReservationRejectsNodeConflict(t *testing.T) {
	server := setupTestServer(t)

	first := domain.ReservationCreateRequest{Name: "first", NodeNames: []string{"A00"}}
	body, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleReservations(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected first create to succeed, got %d", w.Code)
	}

	second := domain.ReservationCreateRequest{Name: "second", NodeNames: []string{"A00"}}
	body, err = json.Marshal(second)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/reservations", bytes.NewReader(body))
	w = httptest.NewRecorder()
	server.handleReservations(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected conflict, got %d", w.Code)
	}
}

func TestJobMetricsForRunningJob(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/job-1719248400000000000/metrics", nil)
	w := httptest.NewRecorder()

	server.handleJobDetail(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var response struct {
		JobID string                   `json:"jobId"`
		GPUs  []map[string]interface{} `json:"gpus"`
		Tasks []map[string]interface{} `json:"tasks"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Decode metrics failed: %v", err)
	}
	if response.JobID == "" || len(response.GPUs) == 0 {
		t.Fatalf("expected job metrics with GPUs, got %#v", response)
	}
	if len(response.Tasks) == 0 {
		t.Fatalf("expected task-level metrics, got %#v", response)
	}
	if _, ok := response.Tasks[0]["cpuUsage"]; !ok {
		t.Fatalf("expected task cpuUsage metric, got %#v", response.Tasks[0])
	}
	if _, ok := response.Tasks[0]["memoryUsedMb"]; !ok {
		t.Fatalf("expected task memoryUsedMb metric, got %#v", response.Tasks[0])
	}
	if _, ok := response.Tasks[0]["gpus"]; !ok {
		t.Fatalf("expected task GPU metrics, got %#v", response.Tasks[0])
	}
}

func TestClusterSummary(t *testing.T) {
	server := setupTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/cluster/summary", nil)
	w := httptest.NewRecorder()

	server.handleClusterSummary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var summary map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	cluster, ok := summary["cluster"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected cluster field in summary")
	}

	if totalNodes := cluster["totalNodes"].(float64); totalNodes != 2 {
		t.Errorf("Expected 2 total nodes, got %v", totalNodes)
	}
}
