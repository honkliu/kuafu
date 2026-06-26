package controller

import (
	"context"
	"fmt"

	"github.com/microsoft/kuafu/internal/domain"
	kuafuerrors "github.com/microsoft/kuafu/internal/errors"
	"github.com/microsoft/kuafu/internal/repository"
	"github.com/microsoft/kuafu/internal/retry"
	"github.com/microsoft/kuafu/internal/scheduler"
)

type SyncController struct {
	repo    repository.JobRepository
	runtime scheduler.RuntimeAdapter
	retry   retry.Config
}

type ReconcileResult struct {
	Submitted int
	Updated   int
}

func NewSyncController(repo repository.JobRepository, runtime scheduler.RuntimeAdapter) *SyncController {
	return &SyncController{repo: repo, runtime: runtime, retry: retry.DefaultConfig()}
}

func (c *SyncController) Reconcile(ctx context.Context) (*ReconcileResult, error) {
	jobs, err := c.repo.ListJobs()
	if err != nil {
		return nil, fmt.Errorf("list job intents: %w", err)
	}

	result := &ReconcileResult{}
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		switch {
		case job.Status == domain.JobStatusQueued && job.RuntimeID == "":
			if err := c.submitJob(ctx, job); err != nil {
				return result, err
			}
			result.Submitted++
		case job.RuntimeID != "" && !isTerminalStatus(job.Status):
			updated, err := c.syncRuntimeStatus(ctx, job)
			if err != nil {
				return result, err
			}
			if updated {
				result.Updated++
			}
		}
	}

	return result, nil
}

func (c *SyncController) submitJob(ctx context.Context, job *domain.Job) error {
	var submission *scheduler.RuntimeSubmission
	err := retry.Do(ctx, c.retry, kuafuerrors.IsTransient, func() error {
		var submitErr error
		submission, submitErr = c.runtime.Submit(ctx, job.RuntimeSpec())
		return submitErr
	})
	if err != nil {
		return fmt.Errorf("submit runtime job %s: %w", job.ID, err)
	}

	job.RuntimeID = submission.RuntimeID
	job.Status = domain.JobStatusRunning
	job.Logs = append(job.Logs, "runtime object submitted")
	if err := c.repo.AddJob(job); err != nil {
		return fmt.Errorf("store submitted job %s: %w", job.ID, err)
	}
	return nil
}

func (c *SyncController) syncRuntimeStatus(ctx context.Context, job *domain.Job) (bool, error) {
	var status *scheduler.RuntimeStatus
	err := retry.Do(ctx, c.retry, kuafuerrors.IsTransient, func() error {
		var statusErr error
		status, statusErr = c.runtime.Status(ctx, job.ID)
		return statusErr
	})
	if err != nil {
		return false, fmt.Errorf("read runtime status for job %s: %w", job.ID, err)
	}
	if status == nil || status.Status == job.Status {
		return false, nil
	}

	job.Status = status.Status
	job.RuntimeID = status.RuntimeID
	if status.Message != "" {
		job.Logs = append(job.Logs, status.Message)
	}
	if err := c.repo.AddJob(job); err != nil {
		return false, fmt.Errorf("store runtime status for job %s: %w", job.ID, err)
	}
	return true, nil
}

func isTerminalStatus(status domain.JobStatus) bool {
	switch status {
	case domain.JobStatusCompleted, domain.JobStatusFailed, domain.JobStatusCanceled, domain.JobStatusStopped:
		return true
	default:
		return false
	}
}
