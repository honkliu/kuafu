package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	kuafuerrors "github.com/microsoft/kuafu/internal/errors"
	"github.com/microsoft/kuafu/internal/repository"
	"github.com/microsoft/kuafu/internal/scheduler"
)

func TestSyncControllerSubmitsQueuedJob(t *testing.T) {
	repo := repository.NewMemoryRepository()
	job := testJob("job-1", domain.JobStatusQueued)
	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	runtime := &fakeRuntimeAdapter{submissions: map[string]string{"job-1": "runtime-1"}}
	controller := NewSyncController(repo, runtime)

	result, err := controller.Reconcile(context.Background())
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Submitted != 1 {
		t.Fatalf("expected 1 submission, got %#v", result)
	}

	stored, err := repo.GetJob("job-1")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if stored.RuntimeID != "runtime-1" || stored.Status != domain.JobStatusRunning {
		t.Fatalf("unexpected stored job: %#v", stored)
	}
}

func TestSyncControllerUpdatesRuntimeStatus(t *testing.T) {
	repo := repository.NewMemoryRepository()
	job := testJob("job-1", domain.JobStatusRunning)
	job.RuntimeID = "runtime-1"
	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	runtime := &fakeRuntimeAdapter{statuses: map[string]*scheduler.RuntimeStatus{
		"job-1": {JobID: "job-1", RuntimeID: "runtime-1", Status: domain.JobStatusCompleted, Message: "completed in runtime"},
	}}
	controller := NewSyncController(repo, runtime)

	result, err := controller.Reconcile(context.Background())
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Updated != 1 {
		t.Fatalf("expected 1 update, got %#v", result)
	}

	stored, err := repo.GetJob("job-1")
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if stored.Status != domain.JobStatusCompleted {
		t.Fatalf("expected completed job, got %#v", stored)
	}
}

func TestSyncControllerSkipsTerminalJobs(t *testing.T) {
	repo := repository.NewMemoryRepository()
	job := testJob("job-1", domain.JobStatusCompleted)
	job.RuntimeID = "runtime-1"
	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	runtime := &fakeRuntimeAdapter{}
	controller := NewSyncController(repo, runtime)

	result, err := controller.Reconcile(context.Background())
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Submitted != 0 || result.Updated != 0 || runtime.statusCalls != 0 {
		t.Fatalf("expected terminal job to be skipped, result=%#v statusCalls=%d", result, runtime.statusCalls)
	}
}

func TestSyncControllerRetriesTransientSubmit(t *testing.T) {
	repo := repository.NewMemoryRepository()
	job := testJob("job-1", domain.JobStatusQueued)
	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	runtime := &fakeRuntimeAdapter{submissions: map[string]string{"job-1": "runtime-1"}, transientSubmitFailures: 1}
	controller := NewSyncController(repo, runtime)

	if _, err := controller.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if runtime.submitCalls != 2 {
		t.Fatalf("expected 2 submit attempts, got %d", runtime.submitCalls)
	}
}

func TestSyncControllerDoesNotRetryPermanentSubmit(t *testing.T) {
	repo := repository.NewMemoryRepository()
	job := testJob("job-1", domain.JobStatusQueued)
	if err := repo.AddJob(job); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	runtime := &fakeRuntimeAdapter{permanentSubmitFailure: true}
	controller := NewSyncController(repo, runtime)

	if _, err := controller.Reconcile(context.Background()); err == nil {
		t.Fatal("expected permanent submit error")
	}
	if runtime.submitCalls != 1 {
		t.Fatalf("expected 1 submit attempt, got %d", runtime.submitCalls)
	}
}

func testJob(id string, status domain.JobStatus) *domain.Job {
	return &domain.Job{ID: id, Name: "test", Queue: "default", Command: "echo ok", GPUCount: 1, Status: status, SubmittedAt: time.Now()}
}

type fakeRuntimeAdapter struct {
	submissions             map[string]string
	statuses                map[string]*scheduler.RuntimeStatus
	submitCalls             int
	statusCalls             int
	transientSubmitFailures int
	permanentSubmitFailure  bool
}

func (a *fakeRuntimeAdapter) Submit(ctx context.Context, spec domain.RuntimeJobSpec) (*scheduler.RuntimeSubmission, error) {
	a.submitCalls++
	if a.permanentSubmitFailure {
		return nil, kuafuerrors.New(kuafuerrors.KindPermanent, "invalid job spec")
	}
	if a.transientSubmitFailures > 0 {
		a.transientSubmitFailures--
		return nil, kuafuerrors.New(kuafuerrors.KindTransient, "runtime api unavailable")
	}
	runtimeID, ok := a.submissions[spec.JobID]
	if !ok {
		return nil, fmt.Errorf("missing submission for %s", spec.JobID)
	}
	return &scheduler.RuntimeSubmission{JobID: spec.JobID, RuntimeID: runtimeID}, nil
}

func (a *fakeRuntimeAdapter) Cancel(ctx context.Context, jobID string) error {
	return nil
}

func (a *fakeRuntimeAdapter) Status(ctx context.Context, jobID string) (*scheduler.RuntimeStatus, error) {
	a.statusCalls++
	status, ok := a.statuses[jobID]
	if !ok {
		return nil, nil
	}
	return status, nil
}
