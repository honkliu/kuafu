package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
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
	mux.HandleFunc("/api/v1/reservations", s.handleReservations)
	mux.HandleFunc("/api/v1/reservations/", s.handleReservationDetail)
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
	if len(parts) > 1 && parts[1] == "metrics" {
		s.getJobMetrics(w, r, jobID)
		return
	}
	if len(parts) > 1 {
		s.handleJobAction(w, r, jobID, parts[1])
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

func (s *Server) handleJobAction(w http.ResponseWriter, r *http.Request, jobID string, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var err error
	switch action {
	case "start":
		err = s.scheduler.StartJob(jobID)
	case "stop":
		err = s.scheduler.StopJob(jobID)
	case "restart":
		err = s.scheduler.RestartJob(jobID)
	default:
		writeError(w, fmt.Sprintf("unsupported job action %s", action), http.StatusNotFound)
		return
	}
	if err != nil {
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

func (s *Server) getJobMetrics(w http.ResponseWriter, r *http.Request, jobID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	job, err := s.repo.GetJob(jobID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	gpus, err := s.repo.ListGPUs("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	gpusByID := make(map[string]*domain.GPU, len(gpus))
	for _, gpu := range gpus {
		gpusByID[gpu.ID] = gpu
	}

	usage := make([]map[string]interface{}, 0, len(job.AllocatedGPUs))
	for _, gpuID := range job.AllocatedGPUs {
		gpu, exists := gpusByID[gpuID]
		if !exists {
			continue
		}
		utilization := 0
		memoryUsedMB := 0
		powerWatts := 0
		temperatureC := 0
		if job.Status == domain.JobStatusRunning {
			utilization = 65 + ((gpu.Index + len(job.ID)) % 25)
			memoryUsedMB = gpu.MemoryMB * (45 + gpu.Index%20) / 100
			powerWatts = 250 + gpu.Index*7
			temperatureC = 58 + gpu.Index%9
		}
		usage = append(usage, map[string]interface{}{
			"gpuId":         gpu.ID,
			"nodeName":      gpu.NodeName,
			"index":         gpu.Index,
			"model":         gpu.Model,
			"memoryTotalMb": gpu.MemoryMB,
			"memoryUsedMb":  memoryUsedMB,
			"utilization":   utilization,
			"powerWatts":    powerWatts,
			"temperatureC":  temperatureC,
			"health":        "OK",
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobId":       job.ID,
		"status":      job.Status,
		"generatedAt": time.Now(),
		"gpus":        usage,
	})
}

func (s *Server) handleReservations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listReservations(w, r)
	case http.MethodPost:
		s.createReservation(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := s.repo.ListReservations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sort.Slice(reservations, func(i, j int) bool {
		return reservations[i].CreatedAt.After(reservations[j].CreatedAt)
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"reservations": reservations,
		"count":        len(reservations),
	})
}

func (s *Server) createReservation(w http.ResponseWriter, r *http.Request) {
	var req domain.ReservationCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || len(req.NodeNames) == 0 {
		writeError(w, "name and at least one node are required", http.StatusBadRequest)
		return
	}
	nodeNames := uniqueStrings(req.NodeNames)
	for _, nodeName := range nodeNames {
		if _, err := s.repo.GetNode(nodeName); err != nil {
			writeError(w, fmt.Sprintf("node %s not found", nodeName), http.StatusNotFound)
			return
		}
	}
	if conflict := s.activeReservationConflict("", nodeNames); conflict != "" {
		writeError(w, conflict, http.StatusConflict)
		return
	}

	now := time.Now()
	reservation := &domain.NodeReservation{
		ID:        fmt.Sprintf("res-%d", now.UnixNano()),
		Name:      req.Name,
		Owner:     req.Owner,
		NodeNames: nodeNames,
		Status:    domain.ReservationStatusActive,
		CreatedAt: now,
	}
	if req.DurationHours > 0 {
		reservation.ExpiresAt = now.Add(time.Duration(req.DurationHours) * time.Hour)
	}
	if err := s.repo.AddReservation(reservation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, reservation)
}

func (s *Server) handleReservationDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/reservations/")
	parts := strings.Split(path, "/")
	reservationID := parts[0]
	if reservationID == "" {
		writeError(w, "reservation id required", http.StatusBadRequest)
		return
	}
	if len(parts) > 1 && parts[1] == "commands" {
		s.createReservationCommand(w, r, reservationID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getReservation(w, r, reservationID)
	case http.MethodDelete:
		s.releaseReservation(w, r, reservationID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getReservation(w http.ResponseWriter, r *http.Request, reservationID string) {
	reservation, err := s.repo.GetReservation(reservationID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, reservation)
}

func (s *Server) releaseReservation(w http.ResponseWriter, r *http.Request, reservationID string) {
	reservation, err := s.repo.GetReservation(reservationID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	reservation.Status = domain.ReservationStatusReleased
	if err := s.repo.AddReservation(reservation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, reservation)
}

func (s *Server) createReservationCommand(w http.ResponseWriter, r *http.Request, reservationID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	reservation, err := s.repo.GetReservation(reservationID)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	if reservation.Status != domain.ReservationStatusActive {
		writeError(w, "reservation is not active", http.StatusBadRequest)
		return
	}

	var req domain.ReservationCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Image == "" || req.StartCommand == "" {
		writeError(w, "image and startCommand are required", http.StatusBadRequest)
		return
	}

	now := time.Now()
	command := domain.ReservationCommand{
		ID:               fmt.Sprintf("cmd-%d", now.UnixNano()),
		Image:            req.Image,
		DockerRunOptions: req.DockerRunOptions,
		EntryPoint:       req.EntryPoint,
		StartCommand:     req.StartCommand,
		WorkingDirectory: req.WorkingDirectory,
		Environment:      nonEmptyStrings(req.Environment),
		Status:           domain.ReservationCommandStatusRecorded,
		CreatedAt:        now,
		Logs: []string{
			fmt.Sprintf("[%s] LAB MODE: command recorded and rendered; remote execution is not wired yet", now.Format("15:04:05")),
		},
	}
	command.RenderedCommands = renderReservationNodeCommands(reservation.NodeNames, command)
	reservation.Commands = append(reservation.Commands, command)
	if err := s.repo.AddReservation(reservation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, command)
}

func (s *Server) activeReservationConflict(currentID string, nodeNames []string) string {
	reservations, err := s.repo.ListReservations()
	if err != nil {
		return "failed to check existing reservations"
	}
	wanted := stringSet(nodeNames)
	now := time.Now()
	for _, reservation := range reservations {
		if reservation.ID == currentID || reservation.Status != domain.ReservationStatusActive {
			continue
		}
		if !reservation.ExpiresAt.IsZero() && reservation.ExpiresAt.Before(now) {
			continue
		}
		for _, nodeName := range reservation.NodeNames {
			if wanted[nodeName] {
				return fmt.Sprintf("node %s is already reserved by %s", nodeName, reservation.Name)
			}
		}
	}
	return ""
}

func renderReservationNodeCommands(nodeNames []string, command domain.ReservationCommand) []domain.ReservationNodeCommand {
	rendered := make([]domain.ReservationNodeCommand, 0, len(nodeNames))
	for _, nodeName := range nodeNames {
		parts := []string{"docker run --rm", "--name", shellQuote(fmt.Sprintf("kuafu-%s-%s", nodeName, command.ID))}
		if command.DockerRunOptions != "" {
			parts = append(parts, command.DockerRunOptions)
		}
		if command.WorkingDirectory != "" {
			parts = append(parts, "-w", shellQuote(command.WorkingDirectory))
		}
		for _, env := range command.Environment {
			parts = append(parts, "-e", shellQuote(env))
		}
		if command.EntryPoint != "" {
			parts = append(parts, "--entrypoint", shellQuote(command.EntryPoint))
		}
		parts = append(parts, shellQuote(command.Image), command.StartCommand)
		rendered = append(rendered, domain.ReservationNodeCommand{
			NodeName: nodeName,
			Command:  strings.Join(parts, " "),
		})
	}
	return rendered
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func nonEmptyStrings(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (s *Server) handleQueues(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listQueues(w, r)
	case http.MethodPost:
		s.createQueue(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listQueues(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) createQueue(w http.ResponseWriter, r *http.Request) {
	var queue domain.Queue
	if err := json.NewDecoder(r.Body).Decode(&queue); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := normalizeQueueForWrite(&queue, false); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := s.repo.GetQueue(queue.Name); err == nil {
		writeError(w, fmt.Sprintf("queue %s already exists", queue.Name), http.StatusConflict)
		return
	}
	if err := s.repo.AddQueue(&queue); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, &queue)
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	// Extract queue name from path: /api/v1/queues/{name}
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/queues/")
	if name == "" {
		writeError(w, "queue name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getQueue(w, r, name)
	case http.MethodPut:
		s.updateQueue(w, r, name)
	case http.MethodDelete:
		s.deleteQueue(w, r, name)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getQueue(w http.ResponseWriter, r *http.Request, name string) {
	queue, err := s.repo.GetQueue(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, queue)
}

func (s *Server) updateQueue(w http.ResponseWriter, r *http.Request, name string) {
	existing, err := s.repo.GetQueue(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	var queue domain.Queue
	if err := json.NewDecoder(r.Body).Decode(&queue); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	queue.Name = name
	queue.JobsQueued = existing.JobsQueued
	queue.JobsRunning = existing.JobsRunning
	if err := normalizeQueueForWrite(&queue, true); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.repo.AddQueue(&queue); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, &queue)
}

func (s *Server) deleteQueue(w http.ResponseWriter, r *http.Request, name string) {
	if _, err := s.repo.GetQueue(name); err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}
	jobs, err := s.repo.ListJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, job := range jobs {
		if job.Queue == name && !isQueueDeletionSafeJobStatus(job.Status) {
			writeError(w, fmt.Sprintf("queue %s still has active job %s", name, job.ID), http.StatusConflict)
			return
		}
	}
	if err := s.repo.DeleteQueue(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
}

func normalizeQueueForWrite(queue *domain.Queue, updating bool) error {
	queue.Name = strings.TrimSpace(queue.Name)
	queue.Project = strings.TrimSpace(queue.Project)
	if queue.Name == "" {
		return fmt.Errorf("queue name is required")
	}
	if strings.Contains(queue.Name, "/") {
		return fmt.Errorf("queue name cannot contain /")
	}
	if queue.Status == "" {
		queue.Status = domain.QueueStatusActive
	}
	if queue.Status != domain.QueueStatusActive && queue.Status != domain.QueueStatusPaused {
		return fmt.Errorf("queue status must be Active or Paused")
	}
	if queue.MaxGPUs <= 0 {
		return fmt.Errorf("maxGpus must be positive")
	}
	if queue.SoftGPUs < 0 || queue.MaxGPUsPerJob < 0 || queue.MaxQueuedJobs < 0 || queue.MaxRunningJobs < 0 {
		return fmt.Errorf("queue limits cannot be negative")
	}
	if queue.SoftGPUs > queue.MaxGPUs {
		return fmt.Errorf("softGpus cannot exceed maxGpus")
	}
	if queue.MaxGPUsPerJob > queue.MaxGPUs {
		return fmt.Errorf("maxGpusPerJob cannot exceed maxGpus")
	}
	if !updating {
		queue.JobsQueued = 0
		queue.JobsRunning = 0
	}
	return nil
}

func isQueueDeletionSafeJobStatus(status domain.JobStatus) bool {
	switch status {
	case domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCanceled, domain.JobStatusStopped:
		return true
	default:
		return false
	}
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
