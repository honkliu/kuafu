package scheduler

import (
	"context"
	"strings"
	"testing"

	"github.com/microsoft/kuafu/internal/domain"
)

func TestJobRuntimeSpec(t *testing.T) {
	job := &domain.Job{
		ID:       "job-1",
		Name:     "train",
		Queue:    "training",
		Image:    "pytorch:latest",
		Command:  "python train.py",
		GPUCount: 8,
	}

	spec := job.RuntimeSpec()
	if spec.JobID != job.ID || spec.Name != job.Name || spec.Queue != job.Queue {
		t.Fatalf("runtime spec identity mismatch: %#v", spec)
	}
	if len(spec.Command) != 1 || spec.Command[0] != job.Command {
		t.Fatalf("runtime spec command mismatch: %#v", spec.Command)
	}
	if spec.Labels["kuafu.dev/job-id"] != job.ID {
		t.Fatalf("runtime spec missing job label: %#v", spec.Labels)
	}
}

func TestUnsupportedRuntimeAdapter(t *testing.T) {
	adapter := UnsupportedRuntimeAdapter{Backend: "frameworkcontroller"}
	_, err := adapter.Submit(context.Background(), domain.RuntimeJobSpec{JobID: "job-1"})
	if err == nil || !strings.Contains(err.Error(), "frameworkcontroller") {
		t.Fatalf("expected unsupported adapter error, got %v", err)
	}
}
