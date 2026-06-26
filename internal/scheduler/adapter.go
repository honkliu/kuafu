package scheduler

import (
	"context"
	"fmt"

	"github.com/microsoft/kuafu/internal/domain"
)

type RuntimeAdapter interface {
	// Submit must be idempotent by RuntimeJobSpec.JobID. Re-submitting the same
	// Kuafu job should return the existing runtime object instead of creating a
	// duplicate Kubernetes/FrameworkController/Volcano object.
	Submit(ctx context.Context, spec domain.RuntimeJobSpec) (*RuntimeSubmission, error)
	Cancel(ctx context.Context, jobID string) error
	Status(ctx context.Context, jobID string) (*RuntimeStatus, error)
}

type RuntimeSubmission struct {
	JobID     string `json:"jobId"`
	RuntimeID string `json:"runtimeId"`
}

type RuntimeStatus struct {
	JobID     string           `json:"jobId"`
	RuntimeID string           `json:"runtimeId"`
	Status    domain.JobStatus `json:"status"`
	Message   string           `json:"message,omitempty"`
}

type UnsupportedRuntimeAdapter struct {
	Backend string
}

func (a UnsupportedRuntimeAdapter) Submit(ctx context.Context, spec domain.RuntimeJobSpec) (*RuntimeSubmission, error) {
	return nil, a.unsupportedError()
}

func (a UnsupportedRuntimeAdapter) Cancel(ctx context.Context, jobID string) error {
	return a.unsupportedError()
}

func (a UnsupportedRuntimeAdapter) Status(ctx context.Context, jobID string) (*RuntimeStatus, error) {
	return nil, a.unsupportedError()
}

func (a UnsupportedRuntimeAdapter) unsupportedError() error {
	if a.Backend == "" {
		return fmt.Errorf("production runtime adapter is not configured")
	}
	return fmt.Errorf("production runtime adapter %q is not wired yet", a.Backend)
}
