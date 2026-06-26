package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
