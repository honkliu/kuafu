package volcano

import (
	"testing"

	"github.com/microsoft/kuafu/internal/domain"
)

func TestBuildJobManifest(t *testing.T) {
	manifest, err := BuildJobManifest(domain.RuntimeJobSpec{
		JobID:    "Job_One",
		Name:     "train",
		Project:  "vision",
		Queue:    "training",
		Image:    "pytorch:latest",
		Command:  []string{"python", "train.py"},
		GPUCount: 4,
		Labels: map[string]string{
			"kuafu.dev/job-id":  "Job_One",
			"kuafu.dev/project": "vision",
		},
	})
	if err != nil {
		t.Fatalf("BuildJobManifest failed: %v", err)
	}
	if manifest.APIVersion != APIVersion || manifest.Kind != Kind {
		t.Fatalf("unexpected type metadata: %#v", manifest)
	}
	if manifest.Metadata.Name != "job-one" {
		t.Fatalf("expected DNS-safe name, got %s", manifest.Metadata.Name)
	}
	if manifest.Spec.Queue != "training" {
		t.Fatalf("expected training queue, got %s", manifest.Spec.Queue)
	}
	container := manifest.Spec.Tasks[0].Template.Spec.Containers[0]
	if container.Resources.Limits["nvidia.com/gpu"] != 4 {
		t.Fatalf("expected 4 GPUs, got %#v", container.Resources.Limits)
	}
}

func TestBuildJobManifestRejectsInvalidSpec(t *testing.T) {
	if _, err := BuildJobManifest(domain.RuntimeJobSpec{}); err == nil {
		t.Fatal("expected invalid spec error")
	}
}
