package mongodb

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
)

func TestNewRequiresURIAndDatabase(t *testing.T) {
	if _, err := New(context.Background(), "", "kuafu"); err == nil {
		t.Fatal("expected empty uri to fail")
	}
	if _, err := New(context.Background(), "mongodb://localhost:27017", ""); err == nil {
		t.Fatal("expected empty database to fail")
	}
}

func TestRepositoryIntegration(t *testing.T) {
	uri := os.Getenv("KUAFU_MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("KUAFU_MONGODB_TEST_URI is not set")
	}

	repo, err := New(context.Background(), uri, "kuafu_test")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer repo.Close(context.Background())

	now := time.Now()
	node := &domain.Node{Name: "test-node", Hostname: "test-node", GPUCount: 1, Status: domain.NodeStatusReady, CreatedAt: now}
	if err := repo.AddNode(node); err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}
	gotNode, err := repo.GetNode("test-node")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}
	if gotNode.Name != node.Name {
		t.Fatalf("unexpected node: %#v", gotNode)
	}

	gpu := &domain.GPU{ID: "test-gpu", NodeName: "test-node", Status: domain.GPUStatusAvailable, CreatedAt: now}
	if err := repo.AddGPU(gpu); err != nil {
		t.Fatalf("AddGPU failed: %v", err)
	}
	if err := repo.AllocateGPUs([]string{"test-gpu"}, "job-1"); err != nil {
		t.Fatalf("AllocateGPUs failed: %v", err)
	}
	allocatedGPU, err := repo.GetGPU("test-gpu")
	if err != nil {
		t.Fatalf("GetGPU failed: %v", err)
	}
	if allocatedGPU.Status != domain.GPUStatusAllocated || allocatedGPU.AllocatedTo != "job-1" {
		t.Fatalf("unexpected allocated GPU: %#v", allocatedGPU)
	}
}
