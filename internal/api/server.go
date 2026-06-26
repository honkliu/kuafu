package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/microsoft/kuafu/internal/auth"
	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/policy"
	"github.com/microsoft/kuafu/internal/repository"
	"github.com/microsoft/kuafu/internal/scheduler"
)

// Server provides the HTTP API server
type Server struct {
	repo      repository.Repository
	scheduler *scheduler.Scheduler
	auth      auth.Authenticator
	server    *http.Server
}

// NewServer creates a new API server
func NewServer(repo repository.Repository, sched *scheduler.Scheduler, addr string) *Server {
	s := &Server{
		repo:      repo,
		scheduler: sched,
		auth:      auth.AnonymousAuthenticator{},
	}

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/api/v1/nodes", s.handleNodes)
	mux.HandleFunc("/api/v1/nodes/", s.handleNode)
	mux.HandleFunc("/api/v1/gpus", s.handleGPUs)
	mux.HandleFunc("/api/v1/gpus/", s.handleGPU)
	mux.HandleFunc("/api/v1/jobs", s.handleJobs)
	mux.HandleFunc("/api/v1/jobs/", s.handleJobDetail)
	mux.HandleFunc("/api/v1/queues", s.handleQueues)
	mux.HandleFunc("/api/v1/queues/", s.handleQueue)
	mux.HandleFunc("/api/v1/cluster/summary", s.handleClusterSummary)

	// Serve static files from web/ directory
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) SetAuthenticator(authenticator auth.Authenticator) {
	s.auth = authenticator
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ready": true})
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodes, err := s.repo.ListNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

func (s *Server) handleNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract node name from path: /api/v1/nodes/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/nodes/")
	if name == "" {
		http.Error(w, "node name required", http.StatusBadRequest)
		return
	}

	node, err := s.repo.GetNode(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, node)
}

func (s *Server) handleGPUs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodeName := r.URL.Query().Get("node")
	gpus, err := s.repo.ListGPUs(nodeName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"gpus":  gpus,
		"count": len(gpus),
	})
}

func (s *Server) handleGPU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract GPU ID from path: /api/v1/gpus/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/gpus/")
	if id == "" {
		http.Error(w, "gpu id required", http.StatusBadRequest)
		return
	}

	gpu, err := s.repo.GetGPU(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, gpu)
}

func writeJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

// Helper to write JSON error response
func writeError(w http.ResponseWriter, message string, code int) {
	writeJSON(w, code, map[string]string{"error": message})
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listJobs(w, r)
	case http.MethodPost:
		s.submitJob(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.repo.ListJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

func (s *Server) submitJob(w http.ResponseWriter, r *http.Request) {
	var req domain.JobSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" || req.Queue == "" || req.Command == "" {
		writeError(w, "name, queue, and command are required", http.StatusBadRequest)
		return
	}

	if req.GPUCount <= 0 {
		req.GPUCount = 1
	}
	if req.Project != "" {
		user, err := s.auth.Authenticate(r)
		if err != nil {
			writeError(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := policy.CanSubmit(user, req.Project); err != nil {
			writeError(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	// Check if queue exists
	queue, err := s.repo.GetQueue(req.Queue)
	if err != nil {
		writeError(w, fmt.Sprintf("queue %s not found", req.Queue), http.StatusNotFound)
		return
	}

	// Generate job ID
	jobID := fmt.Sprintf("job-%d", time.Now().UnixNano())

	// Create job
	job := &domain.Job{
		ID:          jobID,
		Name:        req.Name,
		Project:     req.Project,
		Queue:       req.Queue,
		Command:     req.Command,
		Image:       req.Image,
		GPUCount:    req.GPUCount,
		Status:      domain.JobStatusQueued,
		SubmittedAt: time.Now(),
		Logs:        []string{fmt.Sprintf("[%s] Job submitted to queue %s", time.Now().Format("15:04:05"), req.Queue)},
	}

	jobs, err := s.repo.ListJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := policy.AdmitJob(queue, job, policy.UsageForQueue(queue.Name, jobs)); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.repo.AddJob(job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update queue stats
	queue.JobsQueued++
	if err := s.repo.AddQueue(queue); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from path: /api/v1/jobs/{id} or /api/v1/jobs/{id}/logs
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
	parts := strings.Split(path, "/")
	jobID := parts[0]

	if jobID == "" {
		writeError(w, "job id required", http.StatusBadRequest)
		return
	}

	// Check if this is a logs request
	if len(parts) > 1 && parts[1] == "logs" {
		s.getJobLogs(w, r, jobID)
		return
	}

	// Handle job detail operations
	switch r.Method {
	case http.MethodGet:
		s.getJob(w, r, jobID)
	case http.MethodDelete:
		s.cancelJob(w, r, jobID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request, jobID string) {
	job, err := s.repo.GetJob(jobID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request, jobID string) {
	if err := s.scheduler.CancelJob(jobID); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	job, err := s.repo.GetJob(jobID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) getJobLogs(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	job, err := s.repo.GetJob(jobID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobId": job.ID,
		"logs":  job.Logs,
	})
}

func (s *Server) handleQueues(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	queues, err := s.repo.ListQueues()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"queues": queues,
		"count":  len(queues),
	})
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract queue name from path: /api/v1/queues/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/queues/")
	if name == "" {
		writeError(w, "queue name required", http.StatusBadRequest)
		return
	}

	queue, err := s.repo.GetQueue(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, queue)
}

func (s *Server) handleClusterSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nodes, err := s.repo.ListNodes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	gpus, err := s.repo.ListGPUs("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jobs, err := s.repo.ListJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	queues, err := s.repo.ListQueues()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate stats
	totalGPUs := len(gpus)
	allocatedGPUs := 0
	availableGPUs := 0
	for _, gpu := range gpus {
		switch gpu.Status {
		case domain.GPUStatusAllocated:
			allocatedGPUs++
		case domain.GPUStatusAvailable:
			availableGPUs++
		}
	}

	jobsQueued := 0
	jobsRunning := 0
	jobsCompleted := 0
	jobsFailed := 0
	for _, job := range jobs {
		switch job.Status {
		case domain.JobStatusQueued:
			jobsQueued++
		case domain.JobStatusRunning:
			jobsRunning++
		case domain.JobStatusCompleted:
			jobsCompleted++
		case domain.JobStatusFailed:
			jobsFailed++
		}
	}

	summary := map[string]interface{}{
		"cluster": map[string]interface{}{
			"totalNodes": len(nodes),
			"gpus": map[string]interface{}{
				"total":     totalGPUs,
				"allocated": allocatedGPUs,
				"available": availableGPUs,
			},
		},
		"jobs": map[string]interface{}{
			"queued":    jobsQueued,
			"running":   jobsRunning,
			"completed": jobsCompleted,
			"failed":    jobsFailed,
		},
		"queues": len(queues),
	}

	writeJSON(w, http.StatusOK, summary)
}
