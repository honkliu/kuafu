package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
	"github.com/microsoft/kuafu/pkg/fixtures"
)

var (
	apiURL  string
	fixture bool
)

func main() {
	root := flag.NewFlagSet("kuafu", flag.ExitOnError)
	root.StringVar(&apiURL, "api-url", getEnv("KUAFU_API_URL", "http://localhost:8080"), "Kuafu API server URL")
	root.BoolVar(&fixture, "fixture", false, "Use offline fixture mode (no API calls)")
	root.Usage = printUsage
	if err := root.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(2)
	}

	if root.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	command := root.Arg(0)
	args := root.Args()[1:]

	switch command {
	case "nodes":
		handleNodesCommand(args)
	case "gpus":
		handleGPUsCommand(args)
	case "jobs":
		handleJobsCommand(args)
	case "queues":
		handleQueuesCommand(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleNodesCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "nodes subcommand required: list, describe")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		listNodes()
	case "describe":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "node name required")
			os.Exit(1)
		}
		describeNode(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown nodes subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func handleGPUsCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "gpus subcommand required: list")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		listGPUs()
	default:
		fmt.Fprintf(os.Stderr, "Unknown gpus subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func handleJobsCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "jobs subcommand required: submit, list, describe, start, stop, restart, cancel, logs")
		os.Exit(1)
	}

	switch args[0] {
	case "submit":
		submitJob(args[1:])
	case "list":
		listJobs()
	case "describe":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job ID required")
			os.Exit(1)
		}
		describeJob(args[1])
	case "cancel":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job ID required")
			os.Exit(1)
		}
		doJobAction(args[1], "cancel")
	case "start", "stop", "restart":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job ID required")
			os.Exit(1)
		}
		doJobAction(args[1], args[0])
	case "logs":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "job ID required")
			os.Exit(1)
		}
		getJobLogs(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown jobs subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func handleQueuesCommand(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "queues subcommand required: list, describe, create, update, pause, resume, delete")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		listQueues()
	case "describe":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "queue name required")
			os.Exit(1)
		}
		describeQueue(args[1])
	case "create":
		createQueue(args[1:])
	case "update":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "queue name required")
			os.Exit(1)
		}
		updateQueue(args[1], args[2:])
	case "pause":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "queue name required")
			os.Exit(1)
		}
		setQueueStatus(args[1], domain.QueueStatusPaused)
	case "resume":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "queue name required")
			os.Exit(1)
		}
		setQueueStatus(args[1], domain.QueueStatusActive)
	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "queue name required")
			os.Exit(1)
		}
		deleteQueue(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown queues subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func listNodes() {
	var nodes []*domain.Node

	if fixture {
		nodes = getFixtureNodes()
	} else {
		var err error
		nodes, err = fetchNodes()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching nodes: %v\n", err)
			os.Exit(1)
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tHOSTNAME\tSTATUS\tCPUs\tMEMORY(GB)\tGPUs\tPRIVATE IP")
	for _, node := range nodes {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\n",
			node.Name,
			node.Hostname,
			node.Status,
			node.CPUCount,
			node.MemoryGB,
			node.GPUCount,
			node.PrivateIP,
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing nodes: %v\n", err)
		os.Exit(1)
	}
}

func describeNode(name string) {
	var node *domain.Node

	if fixture {
		nodes := getFixtureNodes()
		for _, n := range nodes {
			if n.Name == name {
				node = n
				break
			}
		}
		if node == nil {
			fmt.Fprintf(os.Stderr, "Node %s not found\n", name)
			os.Exit(1)
		}
	} else {
		var err error
		node, err = fetchNode(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching node: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Name:          %s\n", node.Name)
	fmt.Printf("Hostname:      %s\n", node.Hostname)
	fmt.Printf("Status:        %s\n", node.Status)
	fmt.Printf("Private IP:    %s\n", node.PrivateIP)
	if len(node.InfiniBandIPs) > 0 {
		fmt.Printf("InfiniBand:    %s\n", strings.Join(node.InfiniBandIPs, ", "))
	}
	fmt.Printf("OS:            %s\n", node.OS)
	fmt.Printf("Kernel:        %s\n", node.Kernel)
	fmt.Printf("CPUs:          %d\n", node.CPUCount)
	fmt.Printf("Memory:        %d GB\n", node.MemoryGB)
	fmt.Printf("Disk:          %d GB\n", node.DiskGB)
	fmt.Printf("GPUs:          %d\n", node.GPUCount)
	if len(node.Labels) > 0 {
		fmt.Println("Labels:")
		for k, v := range node.Labels {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
	fmt.Printf("Created:       %s\n", node.CreatedAt.Format("2006-01-02 15:04:05"))
}

func listGPUs() {
	var gpus []*domain.GPU

	if fixture {
		gpus = getFixtureGPUs()
	} else {
		var err error
		gpus, err = fetchGPUs()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching GPUs: %v\n", err)
			os.Exit(1)
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNODE\tINDEX\tMODEL\tMEMORY(MB)\tSTATUS\tALLOCATED TO")
	for _, gpu := range gpus {
		allocated := gpu.AllocatedTo
		if allocated == "" {
			allocated = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%d\t%s\t%s\n",
			gpu.ID,
			gpu.NodeName,
			gpu.Index,
			gpu.Model,
			gpu.MemoryMB,
			gpu.Status,
			allocated,
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing GPUs: %v\n", err)
		os.Exit(1)
	}
}

func submitJob(args []string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Job submission not supported in fixture mode")
		os.Exit(1)
	}

	// Parse job submission arguments
	var name, queue, command, image string
	var gpuCount int

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--name":
			if i+1 < len(args) {
				name = args[i+1]
				i++
			}
		case "-q", "--queue":
			if i+1 < len(args) {
				queue = args[i+1]
				i++
			}
		case "-c", "--command":
			if i+1 < len(args) {
				command = args[i+1]
				i++
			}
		case "-i", "--image":
			if i+1 < len(args) {
				image = args[i+1]
				i++
			}
		case "-g", "--gpus":
			if i+1 < len(args) {
				parsedGPUCount, err := strconv.Atoi(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "invalid GPU count %q: %v\n", args[i+1], err)
					os.Exit(1)
				}
				gpuCount = parsedGPUCount
				i++
			}
		}
	}

	// Validate required fields
	if name == "" || command == "" {
		fmt.Fprintln(os.Stderr, "Error: --name and --command are required")
		fmt.Fprintln(os.Stderr, "Usage: kuafu jobs submit --name NAME --command COMMAND [--queue QUEUE] [--gpus COUNT] [--image IMAGE]")
		os.Exit(1)
	}

	if queue == "" {
		queue = "default"
	}
	if gpuCount == 0 {
		gpuCount = 1
	}
	if image == "" {
		image = "nvidia/cuda:12.0-runtime"
	}

	req := domain.JobSubmitRequest{
		Name:     name,
		Queue:    queue,
		Command:  command,
		Image:    image,
		GPUCount: gpuCount,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding job request: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.Post(apiURL+"/api/v1/jobs", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting job: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}

	var job domain.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding submitted job: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Job %s submitted successfully\n", job.ID)
	fmt.Printf("Status: %s\n", job.Status)
	fmt.Printf("Queue: %s\n", job.Queue)
	fmt.Printf("GPUs: %d\n", job.GPUCount)
}

func listJobs() {
	if fixture {
		fmt.Fprintln(os.Stderr, "Job listing not supported in fixture mode")
		os.Exit(1)
	}

	resp, err := http.Get(apiURL + "/api/v1/jobs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching jobs: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result struct {
		Jobs []*domain.Job `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding jobs: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tQUEUE\tSTATUS\tGPUs\tSUBMITTED")
	for _, job := range result.Jobs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			job.ID,
			job.Name,
			job.Queue,
			job.Status,
			job.GPUCount,
			job.SubmittedAt.Format("15:04:05"),
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing jobs: %v\n", err)
		os.Exit(1)
	}
}

func describeJob(jobID string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Job describe not supported in fixture mode")
		os.Exit(1)
	}

	resp, err := http.Get(apiURL + "/api/v1/jobs/" + jobID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching job: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}

	var job domain.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding job: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Job ID:        %s\n", job.ID)
	fmt.Printf("Name:          %s\n", job.Name)
	fmt.Printf("Queue:         %s\n", job.Queue)
	fmt.Printf("Status:        %s\n", job.Status)
	fmt.Printf("Command:       %s\n", job.Command)
	fmt.Printf("Image:         %s\n", job.Image)
	fmt.Printf("GPUs:          %d\n", job.GPUCount)
	fmt.Printf("Submitted:     %s\n", job.SubmittedAt.Format("2006-01-02 15:04:05"))
	if !job.StartedAt.IsZero() {
		fmt.Printf("Started:       %s\n", job.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if !job.CompletedAt.IsZero() {
		fmt.Printf("Completed:     %s\n", job.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if len(job.AllocatedGPUs) > 0 {
		fmt.Printf("Allocated GPUs: %s\n", strings.Join(job.AllocatedGPUs, ", "))
	}
	if job.ExitCode != nil {
		fmt.Printf("Exit Code:     %d\n", *job.ExitCode)
	}
	if job.ErrorMsg != "" {
		fmt.Printf("Error:         %s\n", job.ErrorMsg)
	}
}

func doJobAction(jobID string, action string) {
	if fixture {
		fmt.Fprintf(os.Stderr, "Job %s not supported in fixture mode\n", action)
		os.Exit(1)
	}

	method := http.MethodPost
	path := apiURL + "/api/v1/jobs/" + jobID + "/" + action
	if action == "cancel" {
		method = http.MethodDelete
		path = apiURL + "/api/v1/jobs/" + jobID
	}
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating job %s request: %v\n", action, err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running job %s: %v\n", action, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}

	var job domain.Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding job %s response: %v\n", action, err)
		os.Exit(1)
	}
	fmt.Printf("Job %s %s: %s\n", job.ID, action, job.Status)
}

func getJobLogs(jobID string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Job logs not supported in fixture mode")
		os.Exit(1)
	}

	resp, err := http.Get(apiURL + "/api/v1/jobs/" + jobID + "/logs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching logs: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}

	var result struct {
		Logs []string `json:"logs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding logs: %v\n", err)
		os.Exit(1)
	}

	for _, log := range result.Logs {
		fmt.Println(log)
	}
}

func listQueues() {
	if fixture {
		fmt.Fprintln(os.Stderr, "Queue listing not supported in fixture mode")
		os.Exit(1)
	}

	resp, err := http.Get(apiURL + "/api/v1/queues")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching queues: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result struct {
		Queues []*domain.Queue `json:"queues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding queues: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tPROJECT\tMAX GPUs\tPRIORITY\tQUEUED\tRUNNING")
	for _, q := range result.Queues {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%d\n",
			q.Name,
			queueStatusOrDefault(q.Status),
			valueOrDash(q.Project),
			q.MaxGPUs,
			q.Priority,
			q.JobsQueued,
			q.JobsRunning,
		)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing queues: %v\n", err)
		os.Exit(1)
	}
}

func describeQueue(name string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Queue describe not supported in fixture mode")
		os.Exit(1)
	}

	resp, err := http.Get(apiURL + "/api/v1/queues/" + name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching queue: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}

	var queue domain.Queue
	if err := json.NewDecoder(resp.Body).Decode(&queue); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding queue: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Name:          %s\n", queue.Name)
	fmt.Printf("Status:        %s\n", queueStatusOrDefault(queue.Status))
	fmt.Printf("Project:       %s\n", valueOrDash(queue.Project))
	fmt.Printf("Max GPUs:      %d\n", queue.MaxGPUs)
	fmt.Printf("Soft GPUs:     %d\n", queue.SoftGPUs)
	fmt.Printf("Max/job GPUs:  %d\n", queue.MaxGPUsPerJob)
	fmt.Printf("Max queued:    %d\n", queue.MaxQueuedJobs)
	fmt.Printf("Max running:   %d\n", queue.MaxRunningJobs)
	fmt.Printf("Allow burst:   %v\n", queue.AllowBurst)
	fmt.Printf("Priority:      %d\n", queue.Priority)
	fmt.Printf("Queued Jobs:   %d\n", queue.JobsQueued)
	fmt.Printf("Running Jobs:  %d\n", queue.JobsRunning)
}

func createQueue(args []string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Queue create not supported in fixture mode")
		os.Exit(1)
	}
	queue := domain.Queue{Status: domain.QueueStatusActive, MaxGPUs: 8, Priority: 100}
	applyQueueArgs(&queue, args, true)
	if queue.Name == "" {
		fmt.Fprintln(os.Stderr, "queue --name is required")
		os.Exit(1)
	}
	saveQueue(http.MethodPost, apiURL+"/api/v1/queues", queue)
	fmt.Printf("Queue %s created\n", queue.Name)
}

func updateQueue(name string, args []string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Queue update not supported in fixture mode")
		os.Exit(1)
	}
	queue := fetchQueueOrExit(name)
	applyQueueArgs(&queue, args, false)
	queue.Name = name
	saveQueue(http.MethodPut, apiURL+"/api/v1/queues/"+name, queue)
	fmt.Printf("Queue %s updated\n", name)
}

func setQueueStatus(name string, status domain.QueueStatus) {
	if fixture {
		fmt.Fprintf(os.Stderr, "Queue %s not supported in fixture mode\n", strings.ToLower(string(status)))
		os.Exit(1)
	}
	queue := fetchQueueOrExit(name)
	queue.Status = status
	saveQueue(http.MethodPut, apiURL+"/api/v1/queues/"+name, queue)
	fmt.Printf("Queue %s is now %s\n", name, status)
}

func deleteQueue(name string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Queue delete not supported in fixture mode")
		os.Exit(1)
	}
	req, err := http.NewRequest(http.MethodDelete, apiURL+"/api/v1/queues/"+name, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating queue delete request: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting queue: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}
	fmt.Printf("Queue %s deleted\n", name)
}

func applyQueueArgs(queue *domain.Queue, args []string, allowName bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n", "--name":
			if !allowName {
				fmt.Fprintln(os.Stderr, "queue name is positional for update")
				os.Exit(1)
			}
			queue.Name = nextArg(args, &i, "queue name")
		case "--status":
			queue.Status = domain.QueueStatus(nextArg(args, &i, "queue status"))
		case "--project":
			queue.Project = nextArg(args, &i, "project")
		case "--max-gpus":
			queue.MaxGPUs = parseIntArg(nextArg(args, &i, "max GPUs"), "max GPUs")
		case "--soft-gpus":
			queue.SoftGPUs = parseIntArg(nextArg(args, &i, "soft GPUs"), "soft GPUs")
		case "--max-gpus-per-job":
			queue.MaxGPUsPerJob = parseIntArg(nextArg(args, &i, "max GPUs per job"), "max GPUs per job")
		case "--max-queued-jobs":
			queue.MaxQueuedJobs = parseIntArg(nextArg(args, &i, "max queued jobs"), "max queued jobs")
		case "--max-running-jobs":
			queue.MaxRunningJobs = parseIntArg(nextArg(args, &i, "max running jobs"), "max running jobs")
		case "--priority":
			queue.Priority = parseIntArg(nextArg(args, &i, "priority"), "priority")
		case "--allow-burst":
			queue.AllowBurst = true
		case "--no-allow-burst":
			queue.AllowBurst = false
		default:
			fmt.Fprintf(os.Stderr, "Unknown queue option: %s\n", args[i])
			os.Exit(1)
		}
	}
}

func nextArg(args []string, index *int, name string) string {
	if *index+1 >= len(args) {
		fmt.Fprintf(os.Stderr, "missing value for %s\n", name)
		os.Exit(1)
	}
	*index = *index + 1
	return args[*index]
}

func parseIntArg(value string, name string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s %q: %v\n", name, value, err)
		os.Exit(1)
	}
	return parsed
}

func saveQueue(method string, url string, queue domain.Queue) {
	reqBody, err := json.Marshal(queue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding queue request: %v\n", err)
		os.Exit(1)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating queue request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving queue: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}
}

func fetchQueueOrExit(name string) domain.Queue {
	resp, err := http.Get(apiURL + "/api/v1/queues/" + name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching queue: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: %s\n", readResponseBody(resp.Body))
		os.Exit(1)
	}
	var queue domain.Queue
	if err := json.NewDecoder(resp.Body).Decode(&queue); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding queue: %v\n", err)
		os.Exit(1)
	}
	return queue
}

func queueStatusOrDefault(status domain.QueueStatus) domain.QueueStatus {
	if status == "" {
		return domain.QueueStatusActive
	}
	return status
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func fetchNodes() ([]*domain.Node, error) {
	resp, err := http.Get(apiURL + "/api/v1/nodes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", readResponseBody(resp.Body))
	}

	var result struct {
		Nodes []*domain.Node `json:"nodes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Nodes, nil
}

func fetchNode(name string) (*domain.Node, error) {
	resp, err := http.Get(apiURL + "/api/v1/nodes/" + name)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", readResponseBody(resp.Body))
	}

	var node domain.Node
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, err
	}

	return &node, nil
}

func fetchGPUs() ([]*domain.GPU, error) {
	resp, err := http.Get(apiURL + "/api/v1/gpus")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", readResponseBody(resp.Body))
	}

	var result struct {
		GPUs []*domain.GPU `json:"gpus"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.GPUs, nil
}

func getFixtureNodes() []*domain.Node {
	repo := seedFixtureRepository()
	nodes, err := repo.ListNodes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing fixture nodes: %v\n", err)
		os.Exit(1)
	}
	return nodes
}

func getFixtureGPUs() []*domain.GPU {
	repo := seedFixtureRepository()
	gpus, err := repo.ListGPUs("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing fixture GPUs: %v\n", err)
		os.Exit(1)
	}
	return gpus
}

func seedFixtureRepository() *repository.MemoryRepository {
	repo := repository.NewMemoryRepository()
	if err := fixtures.SeedTestbedA00A01(repo.AddNode, repo.AddGPU, repo.AddQueue, repo.AddJob); err != nil {
		fmt.Fprintf(os.Stderr, "Error seeding fixture data: %v\n", err)
		os.Exit(1)
	}
	return repo
}

func readResponseBody(body io.Reader) string {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Sprintf("failed to read response body: %v", err)
	}
	return string(data)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func printUsage() {
	fmt.Println("Kuafu - GPU Management CLI")
	fmt.Println()
	fmt.Println("Usage: kuafu [--api-url URL] [--fixture] <command> <subcommand> [args]")
	fmt.Println()
	fmt.Println("Global Flags:")
	fmt.Println("  --api-url string    API server URL (default: http://localhost:8080 or KUAFU_API_URL env)")
	fmt.Println("  --fixture          Use offline fixture mode")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  nodes list                      List all nodes")
	fmt.Println("  nodes describe <name>           Describe a specific node")
	fmt.Println("  gpus list                       List all GPUs")
	fmt.Println("  jobs submit [options]           Submit a new job")
	fmt.Println("  jobs list                       List all jobs")
	fmt.Println("  jobs describe <id>              Describe a specific job")
	fmt.Println("  jobs start <id>                 Requeue a stopped/terminal job")
	fmt.Println("  jobs stop <id>                  Stop a queued/running job")
	fmt.Println("  jobs restart <id>               Stop and requeue a job")
	fmt.Println("  jobs cancel <id>                Cancel a job")
	fmt.Println("  jobs logs <id>                  Get job logs")
	fmt.Println("  queues list                     List all queues")
	fmt.Println("  queues describe <name>          Describe a specific queue")
	fmt.Println("  queues create [options]         Create a queue")
	fmt.Println("  queues update <name> [options]  Update a queue")
	fmt.Println("  queues pause <name>             Pause scheduling for a queue")
	fmt.Println("  queues resume <name>            Resume scheduling for a queue")
	fmt.Println("  queues delete <name>            Delete an inactive queue")
	fmt.Println()
	fmt.Println("Job Submit Options:")
	fmt.Println("  -n, --name string      Job name (required)")
	fmt.Println("  -c, --command string   Command to run (required)")
	fmt.Println("  -q, --queue string     Queue name (default: default)")
	fmt.Println("  -g, --gpus int         Number of GPUs (default: 1)")
	fmt.Println("  -i, --image string     Container image (default: nvidia/cuda:12.0-runtime)")
	fmt.Println()
	fmt.Println("Queue Options:")
	fmt.Println("  -n, --name string              Queue name (create only)")
	fmt.Println("  --status string                Active or Paused")
	fmt.Println("  --project string               Owning project")
	fmt.Println("  --max-gpus int                 Hard GPU quota")
	fmt.Println("  --soft-gpus int                Soft GPU quota")
	fmt.Println("  --max-gpus-per-job int         Per-job GPU limit")
	fmt.Println("  --max-queued-jobs int          Queued job limit")
	fmt.Println("  --max-running-jobs int         Running job limit")
	fmt.Println("  --priority int                 Queue priority")
	fmt.Println("  --allow-burst                  Allow exceeding soft quota")
	fmt.Println("  --no-allow-burst               Disable soft quota burst")
}
