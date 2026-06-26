package policy

import (
	"testing"

	"github.com/microsoft/kuafu/internal/domain"
	kuafuerrors "github.com/microsoft/kuafu/internal/errors"
)

func TestAdmitJobAcceptsDefaultQueue(t *testing.T) {
	queue := &domain.Queue{Name: "default", MaxGPUs: 8}
	job := &domain.Job{ID: "job-1", Queue: "default", GPUCount: 2}
	if err := AdmitJob(queue, job, QueueUsage{}); err != nil {
		t.Fatalf("AdmitJob failed: %v", err)
	}
}

func TestAdmitJobRejectsMaxGPUsPerJob(t *testing.T) {
	queue := &domain.Queue{Name: "training", MaxGPUs: 16, MaxGPUsPerJob: 4}
	job := &domain.Job{ID: "job-1", Queue: "training", GPUCount: 8}
	err := AdmitJob(queue, job, QueueUsage{})
	if err == nil || !kuafuerrors.IsKind(err, kuafuerrors.KindQuota) {
		t.Fatalf("expected quota error, got %v", err)
	}
}

func TestAdmitJobRejectsQueueMismatch(t *testing.T) {
	queue := &domain.Queue{Name: "training", MaxGPUs: 16}
	job := &domain.Job{ID: "job-1", Queue: "default", GPUCount: 1}
	err := AdmitJob(queue, job, QueueUsage{})
	if err == nil || !kuafuerrors.IsKind(err, kuafuerrors.KindPermanent) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestAdmitJobAllowsBurstOverSoftQuota(t *testing.T) {
	queue := &domain.Queue{Name: "burst", MaxGPUs: 16, SoftGPUs: 4, AllowBurst: true}
	job := &domain.Job{ID: "job-1", Queue: "burst", GPUCount: 8}
	if err := AdmitJob(queue, job, QueueUsage{RunningGPUs: 4}); err != nil {
		t.Fatalf("expected burst to be admitted, got %v", err)
	}
}

func TestAdmitJobRejectsHardGPUQuotaByUsage(t *testing.T) {
	queue := &domain.Queue{Name: "training", MaxGPUs: 8}
	job := &domain.Job{ID: "job-1", Queue: "training", GPUCount: 5}
	err := AdmitJob(queue, job, QueueUsage{RunningGPUs: 4})
	if err == nil || !kuafuerrors.IsKind(err, kuafuerrors.KindQuota) {
		t.Fatalf("expected hard quota error, got %v", err)
	}
}

func TestUsageForQueueCountsGPUs(t *testing.T) {
	usage := UsageForQueue("training", []*domain.Job{
		{Queue: "training", Status: domain.JobStatusRunning, GPUCount: 4},
		{Queue: "training", Status: domain.JobStatusQueued, GPUCount: 2},
		{Queue: "other", Status: domain.JobStatusRunning, GPUCount: 8},
	})
	if usage.RunningGPUs != 4 || usage.QueuedGPUs != 2 || usage.RunningJobs != 1 || usage.QueuedJobs != 1 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
}

func TestAdmitJobRejectsProjectMismatch(t *testing.T) {
	queue := &domain.Queue{Name: "training", Project: "vision", MaxGPUs: 8}
	job := &domain.Job{ID: "job-1", Project: "nlp", Queue: "training", GPUCount: 1}
	err := AdmitJob(queue, job, QueueUsage{})
	if err == nil || !kuafuerrors.IsKind(err, kuafuerrors.KindPermanent) {
		t.Fatalf("expected permanent project mismatch, got %v", err)
	}
}

func TestCanSubmitAllowsSubmitter(t *testing.T) {
	user := &domain.User{Alias: "ada", Memberships: []domain.ProjectMembership{{Project: "vision", Role: domain.ProjectRoleSubmitter}}}
	if err := CanSubmit(user, "vision"); err != nil {
		t.Fatalf("CanSubmit failed: %v", err)
	}
}

func TestCanSubmitRejectsViewer(t *testing.T) {
	user := &domain.User{Alias: "ada", Memberships: []domain.ProjectMembership{{Project: "vision", Role: domain.ProjectRoleViewer}}}
	err := CanSubmit(user, "vision")
	if err == nil || !kuafuerrors.IsKind(err, kuafuerrors.KindPermanent) {
		t.Fatalf("expected viewer rejection, got %v", err)
	}
}
