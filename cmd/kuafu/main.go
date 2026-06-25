package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
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
	root.Parse(os.Args[1:])

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
		fmt.Fprintln(os.Stderr, "jobs subcommand required: submit, list, describe, cancel, logs")
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
		cancelJob(args[1])
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
		fmt.Fprintln(os.Stderr, "queues subcommand required: list, describe")
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
	w.Flush()
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
	w.Flush()
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
				fmt.Sscanf(args[i+1], "%d", &gpuCount)
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

	reqBody, _ := json.Marshal(req)
	resp, err := http.Post(apiURL+"/api/v1/jobs", "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error submitting job: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	var job domain.Job
	json.NewDecoder(resp.Body).Decode(&job)
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
	json.NewDecoder(resp.Body).Decode(&result)

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
	w.Flush()
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
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	var job domain.Job
	json.NewDecoder(resp.Body).Decode(&job)

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

func cancelJob(jobID string) {
	if fixture {
		fmt.Fprintln(os.Stderr, "Job cancel not supported in fixture mode")
		os.Exit(1)
	}

	req, _ := http.NewRequest(http.MethodDelete, apiURL+"/api/v1/jobs/"+jobID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error canceling job: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	fmt.Printf("Job %s canceled\n", jobID)
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
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	var result struct {
		Logs []string `json:"logs"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

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
	json.NewDecoder(resp.Body).Decode(&result)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tMAX GPUs\tPRIORITY\tQUEUED\tRUNNING")
	for _, q := range result.Queues {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\n",
			q.Name,
			q.MaxGPUs,
			q.Priority,
			q.JobsQueued,
			q.JobsRunning,
		)
	}
	w.Flush()
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
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	var queue domain.Queue
	json.NewDecoder(resp.Body).Decode(&queue)

	fmt.Printf("Name:          %s\n", queue.Name)
	fmt.Printf("Max GPUs:      %d\n", queue.MaxGPUs)
	fmt.Printf("Priority:      %d\n", queue.Priority)
	fmt.Printf("Queued Jobs:   %d\n", queue.JobsQueued)
	fmt.Printf("Running Jobs:  %d\n", queue.JobsRunning)
}

func fetchNodes() ([]*domain.Node, error) {
	resp, err := http.Get(apiURL + "/api/v1/nodes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
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
	repo := repository.NewMemoryRepository()
	fixtures.SeedTestbedA00A01(repo.AddNode, repo.AddGPU, repo.AddQueue, repo.AddJob)
	nodes, _ := repo.ListNodes()
	return nodes
}

func getFixtureGPUs() []*domain.GPU {
	repo := repository.NewMemoryRepository()
	fixtures.SeedTestbedA00A01(repo.AddNode, repo.AddGPU, repo.AddQueue, repo.AddJob)
	gpus, _ := repo.ListGPUs("")
	return gpus
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
	fmt.Println("  jobs cancel <id>                Cancel a job")
	fmt.Println("  jobs logs <id>                  Get job logs")
	fmt.Println("  queues list                     List all queues")
	fmt.Println("  queues describe <name>          Describe a specific queue")
	fmt.Println()
	fmt.Println("Job Submit Options:")
	fmt.Println("  -n, --name string      Job name (required)")
	fmt.Println("  -c, --command string   Command to run (required)")
	fmt.Println("  -q, --queue string     Queue name (default: default)")
	fmt.Println("  -g, --gpus int         Number of GPUs (default: 1)")
	fmt.Println("  -i, --image string     Container image (default: nvidia/cuda:12.0-runtime)")
	fmt.Println()
}
