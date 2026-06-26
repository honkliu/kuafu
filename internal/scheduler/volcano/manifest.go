package volcano

import (
	"fmt"
	"strings"

	"github.com/microsoft/kuafu/internal/domain"
)

const (
	APIVersion = "batch.volcano.sh/v1alpha1"
	Kind       = "Job"
)

type JobManifest struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       JobSpec  `json:"spec"`
}

type Metadata struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

type JobSpec struct {
	Queue     string `json:"queue"`
	MinMember int    `json:"minMember"`
	Tasks     []Task `json:"tasks"`
}

type Task struct {
	Replicas int         `json:"replicas"`
	Name     string      `json:"name"`
	Template PodTemplate `json:"template"`
}

type PodTemplate struct {
	Metadata Metadata `json:"metadata"`
	Spec     PodSpec  `json:"spec"`
}

type PodSpec struct {
	RestartPolicy string      `json:"restartPolicy"`
	Containers    []Container `json:"containers"`
}

type Container struct {
	Name      string               `json:"name"`
	Image     string               `json:"image"`
	Command   []string             `json:"command"`
	Resources ResourceRequirements `json:"resources"`
}

type ResourceRequirements struct {
	Limits map[string]int `json:"limits"`
}

func BuildJobManifest(spec domain.RuntimeJobSpec) (*JobManifest, error) {
	if spec.JobID == "" {
		return nil, fmt.Errorf("job id is required")
	}
	if spec.Name == "" {
		return nil, fmt.Errorf("job name is required")
	}
	if spec.Queue == "" {
		return nil, fmt.Errorf("queue is required")
	}
	if spec.Image == "" {
		return nil, fmt.Errorf("image is required")
	}
	if len(spec.Command) == 0 {
		return nil, fmt.Errorf("command is required")
	}
	if spec.GPUCount <= 0 {
		return nil, fmt.Errorf("gpu count must be positive")
	}

	labels := cloneLabels(spec.Labels)
	labels["kuafu.dev/scheduler"] = "volcano"

	return &JobManifest{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata: Metadata{
			Name:   dnsName(spec.JobID),
			Labels: labels,
		},
		Spec: JobSpec{
			Queue:     spec.Queue,
			MinMember: 1,
			Tasks: []Task{
				{
					Replicas: 1,
					Name:     "worker",
					Template: PodTemplate{
						Metadata: Metadata{Labels: labels},
						Spec: PodSpec{
							RestartPolicy: "Never",
							Containers: []Container{
								{
									Name:    "main",
									Image:   spec.Image,
									Command: spec.Command,
									Resources: ResourceRequirements{Limits: map[string]int{
										"nvidia.com/gpu": spec.GPUCount,
									}},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func cloneLabels(labels map[string]string) map[string]string {
	cloned := make(map[string]string, len(labels)+1)
	for key, value := range labels {
		if value != "" {
			cloned[key] = value
		}
	}
	return cloned
}

func dnsName(value string) string {
	name := strings.ToLower(value)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}
