package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
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
	mux.HandleFunc("/api/v1/gpu-telemetry", s.handleGPUTelemetry)
	mux.HandleFunc("/api/v1/system/reserved-env", s.handleReservedEnv)
	mux.HandleFunc("/api/v1/jobs", s.handleJobs)
	mux.HandleFunc("/api/v1/jobs/", s.handleJobDetail)
	mux.HandleFunc("/api/v1/queues", s.handleQueues)
	mux.HandleFunc("/api/v1/queues/", s.handleQueue)
	mux.HandleFunc("/api/v1/reservations", s.handleReservations)
	mux.HandleFunc("/api/v1/reservations/", s.handleReservationDetail)
	mux.HandleFunc("/api/v1/cluster/summary", s.handleClusterSummary)

	// Serve static files from web/ directory. Keep the console uncached during rapid UI iteration.
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		fs.ServeHTTP(w, r)
	}))

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) handleGPUTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snapshot, err := collectNvidiaSMITelemetry()
	if err != nil {
		snapshot = fallbackRepositoryTelemetry(s.repo, err)
	} else {
		snapshot = mergeRepositoryTelemetry(s.repo, snapshot)
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleReservedEnv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"reservedEnv": reservedEnvCatalog()})
}

func collectNvidiaSMITelemetry() (*domain.GPUUsageSnapshot, error) {
	query := "index,uuid,name,memory.total,memory.used,utilization.gpu,power.draw,temperature.gpu"
	output, err := exec.Command("nvidia-smi", "--query-gpu="+query, "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(strings.NewReader(string(output)))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	usage := make([]domain.GPUUsage, 0, len(records))
	for _, record := range records {
		if len(record) < 8 {
			continue
		}
		index := atoiSafe(record[0])
		usage = append(usage, domain.GPUUsage{
			Index:         index,
			UUID:          strings.TrimSpace(record[1]),
			NodeName:      "A00",
			Name:          strings.TrimSpace(record[2]),
			MemoryTotalMB: atoiSafe(record[3]),
			MemoryUsedMB:  atoiSafe(record[4]),
			Utilization:   atoiSafe(record[5]),
			PowerWatts:    atoiSafe(record[6]),
			TemperatureC:  atoiSafe(record[7]),
		})
	}
	uuidToIndex := make(map[string]int, len(usage))
	for _, gpu := range usage {
		uuidToIndex[gpu.UUID] = gpu.Index
	}
	processes := collectNvidiaSMIProcesses(uuidToIndex)
	for index := range usage {
		usage[index].Processes = processes[usage[index].Index]
	}
	return &domain.GPUUsageSnapshot{GeneratedAt: time.Now(), Source: "nvidia-smi", GPUs: usage}, nil
}

func collectNvidiaSMIProcesses(uuidToIndex map[string]int) map[int][]domain.GPUProcess {
	processes := map[int][]domain.GPUProcess{}
	query := "gpu_uuid,pid,process_name,used_memory"
	output, err := exec.Command("nvidia-smi", "--query-compute-apps="+query, "--format=csv,noheader,nounits").Output()
	if err != nil {
		return processes
	}
	reader := csv.NewReader(strings.NewReader(string(output)))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return processes
	}
	for _, record := range records {
		if len(record) < 4 {
			continue
		}
		gpuUUID := strings.TrimSpace(record[0])
		gpuIndex, exists := uuidToIndex[gpuUUID]
		if !exists {
			continue
		}
		processes[gpuIndex] = append(processes[gpuIndex], domain.GPUProcess{
			GPUIndex:     gpuIndex,
			GPUUUID:      gpuUUID,
			PID:          atoiSafe(record[1]),
			ProcessName:  strings.TrimSpace(record[2]),
			UsedMemoryMB: atoiSafe(record[3]),
		})
	}
	return processes
}

func fallbackRepositoryTelemetry(repo repository.Repository, sourceErr error) *domain.GPUUsageSnapshot {
	gpus, err := repo.ListGPUs("")
	if err != nil {
		return &domain.GPUUsageSnapshot{GeneratedAt: time.Now(), Source: "repository", Error: sourceErr.Error() + "; " + err.Error()}
	}
	usage := make([]domain.GPUUsage, 0, len(gpus))
	for _, gpu := range gpus {
		usage = append(usage, domain.GPUUsage{
			Index:         gpu.Index,
			UUID:          gpu.UUID,
			NodeName:      gpu.NodeName,
			Name:          gpu.Model,
			MemoryTotalMB: gpu.MemoryMB,
			MemoryUsedMB:  0,
			Utilization:   0,
		})
	}
	return &domain.GPUUsageSnapshot{GeneratedAt: time.Now(), Source: "repository", GPUs: usage, Error: sourceErr.Error()}
}

func mergeRepositoryTelemetry(repo repository.Repository, snapshot *domain.GPUUsageSnapshot) *domain.GPUUsageSnapshot {
	gpus, err := repo.ListGPUs("")
	if err != nil {
		snapshot.Error = err.Error()
		return snapshot
	}
	byNodeIndex := make(map[string]int, len(snapshot.GPUs))
	for index, gpu := range snapshot.GPUs {
		if gpu.NodeName == "" {
			gpu.NodeName = "A00"
			snapshot.GPUs[index].NodeName = gpu.NodeName
		}
		byNodeIndex[fmt.Sprintf("%s/%d", gpu.NodeName, gpu.Index)] = index
	}
	for _, gpu := range gpus {
		key := fmt.Sprintf("%s/%d", gpu.NodeName, gpu.Index)
		if index, ok := byNodeIndex[key]; ok {
			if snapshot.GPUs[index].Name == "" {
				snapshot.GPUs[index].Name = gpu.Model
			}
			if snapshot.GPUs[index].MemoryTotalMB == 0 {
				snapshot.GPUs[index].MemoryTotalMB = gpu.MemoryMB
			}
			continue
		}
		snapshot.GPUs = append(snapshot.GPUs, domain.GPUUsage{
			Index:         gpu.Index,
			UUID:          gpu.UUID,
			NodeName:      gpu.NodeName,
			Name:          gpu.Model,
			MemoryTotalMB: gpu.MemoryMB,
			MemoryUsedMB:  0,
			Utilization:   0,
		})
	}
	sort.Slice(snapshot.GPUs, func(i, j int) bool {
		if snapshot.GPUs[i].NodeName == snapshot.GPUs[j].NodeName {
			return snapshot.GPUs[i].Index < snapshot.GPUs[j].Index
		}
		return snapshot.GPUs[i].NodeName < snapshot.GPUs[j].NodeName
	})
	if snapshot.Source == "nvidia-smi" {
		snapshot.Source = "nvidia-smi+repository"
	}
	return snapshot
}

func atoiSafe(value string) int {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") || strings.EqualFold(value, "[Not Supported]") {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err == nil {
		return parsed
	}
	floatParsed, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return int(floatParsed)
	}
	return 0
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
	req = requestFromLauncherSpec(req)

	// Validate required fields
	if req.Name == "" || req.Queue == "" || (req.Command == "" && len(req.TaskTemplates) == 0) {
		writeError(w, "name, queue, and command or taskTemplates are required", http.StatusBadRequest)
		return
	}

	if len(req.TaskTemplates) > 0 {
		req.GPUCount = totalRequestedGPUs(req)
	}
	if req.GPUCount <= 0 {
		req.GPUCount = totalRequestedGPUs(req)
		if req.GPUCount <= 0 {
			req.GPUCount = 1
		}
	}
	if err := validateUserEnv(req.SharedEnv, req.TaskTemplates); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
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
		EntrypointScript: req.EntrypointScript,
		Image:       req.Image,
		GPUCount:    req.GPUCount,
		Status:      domain.JobStatusQueued,
		SubmittedAt: time.Now(),
		LauncherSpec:  normalizeLauncherSpec(req),
		TaskTemplates: normalizeTaskTemplates(req),
		SharedEnv:     normalizeEnv(req.SharedEnv),
		Logs:        []string{fmt.Sprintf("[%s] Job submitted to queue %s", time.Now().Format("15:04:05"), req.Queue)},
	}
	job.Tasks = renderTaskInstances(job)
	applyReviewedTasks(job.Tasks, req.Tasks)

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

func requestFromLauncherSpec(req domain.JobSubmitRequest) domain.JobSubmitRequest {
	if req.LauncherSpec == nil {
		return req
	}
	spec := normalizeJobCentricSpec(*req.LauncherSpec)
	req.LauncherSpec = &spec
	if req.Name == "" {
		req.Name = strings.TrimSpace(spec.Name)
	}
	if req.Project == "" {
		req.Project = strings.TrimSpace(spec.Project)
	}
	if req.Queue == "" {
		req.Queue = strings.TrimSpace(spec.Queue)
	}
	if req.Image == "" {
		req.Image = strings.TrimSpace(spec.Docker.Image)
	}
	if req.EntrypointScript == "" {
		req.EntrypointScript = strings.TrimSpace(spec.EntrypointScript)
	}
	if len(req.SharedEnv) == 0 {
		req.SharedEnv = spec.Env
	}
	if len(req.TaskTemplates) == 0 {
		req.TaskTemplates = spec.Tasks
	}
	if req.Command == "" && len(req.TaskTemplates) > 0 {
		req.Command = req.TaskTemplates[0].Command
	}
	return req
}

func normalizeJobCentricSpec(spec domain.LauncherSpec) domain.LauncherSpec {
	if spec.Name == "" {
		spec.Name = spec.Metadata.Name
	}
	if spec.Project == "" {
		spec.Project = spec.Metadata.Project
	}
	if spec.Queue == "" {
		spec.Queue = spec.Spec.Queue
	}
	if spec.ReplicaPolicy == "" {
		spec.ReplicaPolicy = spec.Spec.ReplicaPolicy
	}
	if spec.WorkingDirectory == "" {
		spec.WorkingDirectory = spec.Spec.WorkingDirectory
	}
	if spec.EntrypointScript == "" {
		spec.EntrypointScript = spec.Spec.EntrypointScript
	}
	if spec.Docker.Image == "" && spec.Spec.Docker.Image != "" {
		spec.Docker = spec.Spec.Docker
	}
	if len(spec.Env) == 0 {
		spec.Env = spec.Spec.Env
	}
	if len(spec.Tasks) == 0 {
		spec.Tasks = spec.Spec.Tasks
	}
	if spec.ReplicaPolicy == "" {
		spec.ReplicaPolicy = "fixed"
	}
	return spec
}

func normalizeLauncherSpec(req domain.JobSubmitRequest) *domain.LauncherSpec {
	if req.LauncherSpec != nil {
		spec := normalizeJobCentricSpec(*req.LauncherSpec)
		if spec.Name == "" {
			spec.Name = req.Name
		}
		if spec.Project == "" {
			spec.Project = req.Project
		}
		if spec.Queue == "" {
			spec.Queue = req.Queue
		}
		if spec.Docker.Image == "" {
			spec.Docker.Image = req.Image
		}
		if spec.EntrypointScript == "" {
			spec.EntrypointScript = req.EntrypointScript
		}
		if len(spec.Env) == 0 {
			spec.Env = normalizeEnv(req.SharedEnv)
		}
		if len(spec.Tasks) == 0 {
			spec.Tasks = normalizeTaskTemplates(req)
		}
		return &spec
	}
	return &domain.LauncherSpec{
		Name:          req.Name,
		Project:       req.Project,
		Queue:         req.Queue,
		ReplicaPolicy: "fixed",
		EntrypointScript: req.EntrypointScript,
		Docker:        domain.LauncherDocker{Image: req.Image},
		Env:           normalizeEnv(req.SharedEnv),
		Tasks:         normalizeTaskTemplates(req),
	}
}

func totalRequestedGPUs(req domain.JobSubmitRequest) int {
	if len(req.TaskTemplates) == 0 {
		return req.GPUCount
	}
	total := 0
	for _, template := range req.TaskTemplates {
		replicas := template.Replicas
		if replicas <= 0 {
			replicas = 1
		}
		gpuCount := template.GPUCount
		if gpuCount < 0 {
			gpuCount = 0
		}
		total += replicas * gpuCount
	}
	return total
}

func normalizeTaskTemplates(req domain.JobSubmitRequest) []domain.TaskTemplate {
	if len(req.TaskTemplates) > 0 {
		templates := make([]domain.TaskTemplate, 0, len(req.TaskTemplates))
		for index, template := range req.TaskTemplates {
			template.Name = strings.TrimSpace(template.Name)
			if template.Name == "" {
				template.Name = fmt.Sprintf("task%d", index)
			}
			template.Role = strings.TrimSpace(template.Role)
			if template.Role == "" {
				template.Role = template.Name
			}
			if template.Replicas <= 0 {
				template.Replicas = 1
			}
			if template.Image == "" {
				template.Image = req.Image
			}
			if template.Command == "" {
				template.Command = req.Command
			}
			if template.EntrypointScript == "" {
				template.EntrypointScript = req.EntrypointScript
			}
			if template.WorkingDirectory == "" && req.LauncherSpec != nil {
				template.WorkingDirectory = req.LauncherSpec.WorkingDirectory
			}
			if len(template.DockerOptions) == 0 && req.LauncherSpec != nil {
				template.DockerOptions = append([]string{}, req.LauncherSpec.Docker.Options...)
			}
			if template.GPUCount < 0 {
				template.GPUCount = 0
			}
			template.Env = normalizeEnv(template.Env)
			templates = append(templates, template)
		}
		return templates
	}
	workingDirectory := ""
	dockerOptions := []string(nil)
	if req.LauncherSpec != nil {
		workingDirectory = req.LauncherSpec.WorkingDirectory
		dockerOptions = req.LauncherSpec.Docker.Options
	}
	return []domain.TaskTemplate{
		{Name: "master", Role: "master", Replicas: 1, Image: req.Image, Command: req.Command, EntrypointScript: req.EntrypointScript, WorkingDirectory: workingDirectory, DockerOptions: dockerOptions, GPUCount: req.GPUCount},
		{Name: "worker", Role: "worker", Replicas: 1, Image: req.Image, Command: req.Command, EntrypointScript: req.EntrypointScript, WorkingDirectory: workingDirectory, DockerOptions: dockerOptions, GPUCount: 0},
	}
}

func normalizeEnv(values []domain.EnvVar) []domain.EnvVar {
	env := make([]domain.EnvVar, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			continue
		}
		env = append(env, domain.EnvVar{Name: name, Value: strings.TrimSpace(value.Value)})
	}
	return env
}

func validateUserEnv(shared []domain.EnvVar, templates []domain.TaskTemplate) error {
	for _, item := range shared {
		if isReservedEnvName(item.Name) {
			return fmt.Errorf("%s is a Kuafu reserved environment variable and cannot be set by users", item.Name)
		}
	}
	for _, template := range templates {
		for _, item := range template.Env {
			if isReservedEnvName(item.Name) {
				return fmt.Errorf("%s is a Kuafu reserved environment variable and cannot be set by users", item.Name)
			}
		}
	}
	return nil
}

func renderTaskInstances(job *domain.Job) []domain.TaskInstance {
	worldSize := 0
	for _, template := range job.TaskTemplates {
		worldSize += template.Replicas
	}
	if worldSize <= 0 {
		worldSize = 1
	}
	globalRank := 0
	addresses := make(map[int]string, worldSize)
	for rank := 0; rank < worldSize; rank++ {
		addresses[rank] = fmt.Sprintf("%s-task-%d", job.Name, rank)
	}
	tasks := make([]domain.TaskInstance, 0, worldSize)
	for _, template := range job.TaskTemplates {
		for ordinal := 0; ordinal < template.Replicas; ordinal++ {
			rank := globalRank
			if template.StartOrdinal > 0 {
				rank = template.StartOrdinal + ordinal
			}
			env := mergeTaskEnv(job.SharedEnv, template.Env, reservedTaskEnv(rank, ordinal, template.Role, worldSize, addresses))
			tasks = append(tasks, domain.TaskInstance{
				ID:               fmt.Sprintf("%s-task-%d", job.ID, rank),
				Name:             fmt.Sprintf("%s-%d", template.Name, ordinal),
				Role:             template.Role,
				Rank:             rank,
				Ordinal:          ordinal,
				Address:          addresses[rank],
				Image:            template.Image,
				Command:          renderTemplateCommand(template.Command, env),
				EntrypointScript: renderTemplateCommand(template.EntrypointScript, env),
				WorkingDirectory: template.WorkingDirectory,
				DockerOptions:    append([]string{}, template.DockerOptions...),
				GPUCount:         template.GPUCount,
				CPUCount:         template.CPUCount,
				MemoryGB:         template.MemoryGB,
				Status:           job.Status,
				Env:              env,
			})
			globalRank++
		}
	}
	return tasks
}

func applyReviewedTasks(tasks []domain.TaskInstance, reviewed []domain.TaskInstance) {
	if len(reviewed) == 0 {
		return
	}
	byRank := map[int]domain.TaskInstance{}
	for _, task := range reviewed {
		byRank[task.Rank] = task
	}
	for index := range tasks {
		if reviewedTask, ok := byRank[tasks[index].Rank]; ok && strings.TrimSpace(reviewedTask.Command) != "" {
			tasks[index].Command = reviewedTask.Command
			if strings.TrimSpace(reviewedTask.EntrypointScript) != "" {
				tasks[index].EntrypointScript = reviewedTask.EntrypointScript
			}
		}
	}
}

func reservedTaskEnv(rank int, ordinal int, role string, worldSize int, addresses map[int]string) []domain.EnvVar {
	return []domain.EnvVar{
		{Name: "RANK", Value: strconv.Itoa(rank)},
		{Name: "TASK_RANK", Value: strconv.Itoa(rank)},
		{Name: "TASK_INDEX", Value: strconv.Itoa(rank)},
		{Name: "TASK_ORDINAL", Value: strconv.Itoa(ordinal)},
		{Name: "TASK_ROLE", Value: role},
		{Name: "WORLD_SIZE", Value: strconv.Itoa(worldSize)},
		{Name: "TASK0_ADDRESS", Value: addresses[0]},
		{Name: "MASTER_ADDRESS", Value: addresses[0]},
		{Name: "MASTER_PORT", Value: "20000"},
		{Name: "KUAFU_TASK_ADDRESS", Value: addresses[rank]},
	}
}

func reservedEnvCatalog() []domain.ReservedEnvVar {
	return []domain.ReservedEnvVar{
		{Name: "RANK", Description: "Global zero-based task rank. Master is rank 0 by convention."},
		{Name: "TASK_RANK", Description: "Alias of RANK for launchers that prefer task-scoped naming."},
		{Name: "TASK_INDEX", Description: "Alias of RANK; useful for templates that need a unique per-task number."},
		{Name: "TASK_ORDINAL", Description: "Zero-based replica index within the task template role."},
		{Name: "TASK_ROLE", Description: "Task template role, such as master or worker."},
		{Name: "WORLD_SIZE", Description: "Total rendered task count across all templates."},
		{Name: "TASK0_ADDRESS", Description: "Stable address for rank 0; use as the default distributed rendezvous host."},
		{Name: "MASTER_ADDRESS", Description: "Alias of TASK0_ADDRESS."},
		{Name: "MASTER_PORT", Value: "20000", Description: "Default distributed rendezvous port selected by Kuafu."},
		{Name: "KUAFU_TASK_ADDRESS", Description: "Stable address assigned to this task instance."},
	}
}

func isReservedEnvName(name string) bool {
	name = strings.ToUpper(strings.TrimSpace(name))
	for _, item := range reservedEnvCatalog() {
		if item.Name == name {
			return true
		}
	}
	return false
}

func mergeTaskEnv(groups ...[]domain.EnvVar) []domain.EnvVar {
	merged := []domain.EnvVar{}
	seen := map[string]int{}
	for _, group := range groups {
		for _, item := range group {
			if idx, ok := seen[item.Name]; ok {
				merged[idx] = item
				continue
			}
			seen[item.Name] = len(merged)
			merged = append(merged, item)
		}
	}
	return merged
}

func renderTemplateCommand(command string, env []domain.EnvVar) string {
	for _, item := range env {
		if !isReservedEnvName(item.Name) {
			continue
		}
		command = strings.ReplaceAll(command, "$"+item.Name, item.Value)
		command = strings.ReplaceAll(command, "${"+item.Name+"}", item.Value)
	}
	return command
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
	usageByGPUID := make(map[string]map[string]interface{}, len(job.AllocatedGPUs))
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
		gpuUsage := map[string]interface{}{
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
		}
		usage = append(usage, gpuUsage)
		usageByGPUID[gpu.ID] = gpuUsage
	}
	taskUsage := buildTaskUsage(job, usageByGPUID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobId":       job.ID,
		"status":      job.Status,
		"generatedAt": time.Now(),
		"gpus":        usage,
		"tasks":       taskUsage,
	})
}

func buildTaskUsage(job *domain.Job, usageByGPUID map[string]map[string]interface{}) []map[string]interface{} {
	tasks := job.Tasks
	if len(tasks) == 0 {
		tasks = []domain.TaskInstance{{
			ID:            job.ID + "-master-0",
			Name:          "master-0",
			Role:          "master",
			Rank:          0,
			Command:       job.Command,
			GPUCount:      job.GPUCount,
			CPUCount:      4,
			MemoryGB:      16,
			AllocatedGPUs: job.AllocatedGPUs,
			Status:        job.Status,
		}}
	}
	result := make([]map[string]interface{}, 0, len(tasks))
	for _, task := range tasks {
		gpuUsage := make([]map[string]interface{}, 0, len(task.AllocatedGPUs))
		gpuMemoryUsedMB := 0
		gpuMemoryTotalMB := 0
		gpuUtilizationTotal := 0
		for _, gpuID := range task.AllocatedGPUs {
			if usage, ok := usageByGPUID[gpuID]; ok {
				gpuUsage = append(gpuUsage, usage)
				gpuMemoryUsedMB += intFromInterface(usage["memoryUsedMb"])
				gpuMemoryTotalMB += intFromInterface(usage["memoryTotalMb"])
				gpuUtilizationTotal += intFromInterface(usage["utilization"])
			}
		}
		avgGPUUtilization := 0
		if len(gpuUsage) > 0 {
			avgGPUUtilization = gpuUtilizationTotal / len(gpuUsage)
		}
		cpuUsage := task.CPUUsage
		if cpuUsage == 0 && job.Status == domain.JobStatusRunning {
			cpuUsage = 18 + (task.Rank*7)%40
		}
		memoryUsedMB := task.MemoryUsedMB
		if memoryUsedMB == 0 && job.Status == domain.JobStatusRunning && task.MemoryGB > 0 {
			memoryUsedMB = task.MemoryGB * 1024 * (28 + (task.Rank*5)%25) / 100
		}
		result = append(result, map[string]interface{}{
			"id":                task.ID,
			"name":              task.Name,
			"role":              task.Role,
			"rank":              task.Rank,
			"status":            task.Status,
			"cpuRequested":      task.CPUCount,
			"cpuUsage":          cpuUsage,
			"memoryRequestedMb": task.MemoryGB * 1024,
			"memoryUsedMb":      memoryUsedMB,
			"gpuRequested":      task.GPUCount,
			"allocatedGpus":     task.AllocatedGPUs,
			"gpuUtilization":    avgGPUUtilization,
			"gpuMemoryUsedMb":   gpuMemoryUsedMB,
			"gpuMemoryTotalMb":  gpuMemoryTotalMB,
			"gpus":              gpuUsage,
			"command":           task.Command,
		})
	}
	return result
}

func intFromInterface(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
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
	if len(req.RenderedCommands) > 0 {
		overrides, err := normalizeReservationCommandOverrides(reservation.NodeNames, req.RenderedCommands)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		command.RenderedCommands = overrides
	}
	reservation.Commands = append(reservation.Commands, command)
	if err := s.repo.AddReservation(reservation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, command)
}

func normalizeReservationCommandOverrides(nodeNames []string, commands []domain.ReservationNodeCommand) ([]domain.ReservationNodeCommand, error) {
	wanted := stringSet(nodeNames)
	seen := make(map[string]bool, len(commands))
	normalizedByNode := make(map[string]domain.ReservationNodeCommand, len(commands))
	for _, command := range commands {
		nodeName := strings.TrimSpace(command.NodeName)
		text := strings.TrimSpace(command.Command)
		if nodeName == "" || text == "" {
			return nil, fmt.Errorf("rendered command nodeName and command are required")
		}
		if !wanted[nodeName] {
			return nil, fmt.Errorf("rendered command targets node %s outside reservation", nodeName)
		}
		if seen[nodeName] {
			return nil, fmt.Errorf("duplicate rendered command for node %s", nodeName)
		}
		seen[nodeName] = true
		normalizedByNode[nodeName] = domain.ReservationNodeCommand{NodeName: nodeName, Command: text}
	}
	if len(seen) != len(wanted) {
		return nil, fmt.Errorf("rendered command must be provided for every reserved node")
	}
	normalized := make([]domain.ReservationNodeCommand, 0, len(nodeNames))
	for _, nodeName := range nodeNames {
		normalized = append(normalized, normalizedByNode[nodeName])
	}
	return normalized, nil
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
