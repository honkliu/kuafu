package policy

import (
	"fmt"

	"github.com/microsoft/kuafu/internal/domain"
	kuafuerrors "github.com/microsoft/kuafu/internal/errors"
)

type QueueUsage struct {
	QueuedJobs  int
	RunningJobs int
	QueuedGPUs  int
	RunningGPUs int
}

func AdmitJob(queue *domain.Queue, job *domain.Job, usage QueueUsage) error {
	if queue == nil {
		return kuafuerrors.New(kuafuerrors.KindNotFound, "queue not found")
	}
	if queue.Status == domain.QueueStatusPaused {
		return kuafuerrors.New(kuafuerrors.KindPermanent, fmt.Sprintf("queue %q is paused", queue.Name))
	}
	if job == nil {
		return kuafuerrors.New(kuafuerrors.KindPermanent, "job cannot be nil")
	}
	if job.Queue != queue.Name {
		return kuafuerrors.New(kuafuerrors.KindPermanent, fmt.Sprintf("job queue %q does not match queue %q", job.Queue, queue.Name))
	}
	if queue.Project != "" && job.Project == "" {
		return kuafuerrors.New(kuafuerrors.KindPermanent, fmt.Sprintf("queue %q requires project %q", queue.Name, queue.Project))
	}
	if queue.Project != "" && job.Project != queue.Project {
		return kuafuerrors.New(kuafuerrors.KindPermanent, fmt.Sprintf("job project %q does not match queue project %q", job.Project, queue.Project))
	}
	if job.GPUCount <= 0 {
		return kuafuerrors.New(kuafuerrors.KindPermanent, "job GPU count must be positive")
	}
	if queue.MaxGPUsPerJob > 0 && job.GPUCount > queue.MaxGPUsPerJob {
		return kuafuerrors.New(kuafuerrors.KindQuota, fmt.Sprintf("job requests %d GPUs, queue allows %d GPUs per job", job.GPUCount, queue.MaxGPUsPerJob))
	}
	if queue.MaxGPUs > 0 && usage.RunningGPUs+usage.QueuedGPUs+job.GPUCount > queue.MaxGPUs {
		return kuafuerrors.New(kuafuerrors.KindQuota, fmt.Sprintf("queue %s has insufficient hard GPU quota", queue.Name))
	}
	if queue.SoftGPUs > 0 && !queue.AllowBurst && usage.RunningGPUs+usage.QueuedGPUs+job.GPUCount > queue.SoftGPUs {
		return kuafuerrors.New(kuafuerrors.KindQuota, fmt.Sprintf("queue %s has insufficient soft GPU quota", queue.Name))
	}
	if queue.MaxQueuedJobs > 0 && usage.QueuedJobs >= queue.MaxQueuedJobs {
		return kuafuerrors.New(kuafuerrors.KindQuota, fmt.Sprintf("queue %s queued job limit reached", queue.Name))
	}
	if queue.MaxRunningJobs > 0 && usage.RunningJobs >= queue.MaxRunningJobs {
		return kuafuerrors.New(kuafuerrors.KindQuota, fmt.Sprintf("queue %s running job limit reached", queue.Name))
	}
	return nil
}

func CanSubmit(user *domain.User, project string) error {
	role, ok := user.RoleForProject(project)
	if !ok {
		return kuafuerrors.New(kuafuerrors.KindPermanent, fmt.Sprintf("user is not a member of project %q", project))
	}
	switch role {
	case domain.ProjectRoleAdmin, domain.ProjectRoleSubmitter:
		return nil
	case domain.ProjectRoleViewer:
		return kuafuerrors.New(kuafuerrors.KindPermanent, fmt.Sprintf("user has viewer role in project %q", project))
	default:
		return kuafuerrors.New(kuafuerrors.KindPermanent, fmt.Sprintf("unsupported project role %q", role))
	}
}

func UsageForQueue(queueName string, jobs []*domain.Job) QueueUsage {
	usage := QueueUsage{}
	for _, job := range jobs {
		if job.Queue != queueName {
			continue
		}
		switch job.Status {
		case domain.JobStatusQueued:
			usage.QueuedJobs++
			usage.QueuedGPUs += job.GPUCount
		case domain.JobStatusRunning:
			usage.RunningJobs++
			usage.RunningGPUs += job.GPUCount
		}
	}
	return usage
}
