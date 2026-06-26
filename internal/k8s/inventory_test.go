package k8s

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/kuafu/internal/domain"
	"github.com/microsoft/kuafu/internal/repository"
)

func TestSyncInventoryStoresNodesAndGPUs(t *testing.T) {
	now := time.Now()
	client := &FakeInventoryClient{
		Nodes: []*domain.Node{
			{
				Name:          "a00",
				Hostname:      "hpcdev000000",
				PrivateIP:     "10.0.0.4",
				InfiniBandIPs: []string{"172.16.3.146"},
				CPUCount:      96,
				GPUCount:      8,
				Status:        domain.NodeStatusReady,
				Labels:        map[string]string{"gpu.vendor": "nvidia"},
				CreatedAt:     now,
			},
		},
		GPUs: []*domain.GPU{
			{
				ID:        "a00-gpu-0",
				NodeName:  "a00",
				Index:     0,
				Model:     "NVIDIA A100-SXM4-80GB",
				MemoryMB:  81920,
				Status:    domain.GPUStatusAvailable,
				CreatedAt: now,
			},
		},
	}
	repo := repository.NewMemoryRepository()

	if err := SyncInventory(context.Background(), client, repo); err != nil {
		t.Fatalf("SyncInventory failed: %v", err)
	}

	nodes, err := repo.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes failed: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Name != "a00" {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}

	gpus, err := repo.ListGPUs("a00")
	if err != nil {
		t.Fatalf("ListGPUs failed: %v", err)
	}
	if len(gpus) != 1 || gpus[0].ID != "a00-gpu-0" {
		t.Fatalf("unexpected GPUs: %#v", gpus)
	}
}

func TestSyncInventoryReturnsNodeError(t *testing.T) {
	client := &FakeInventoryClient{NodeErr: errors.New("api unavailable")}
	repo := repository.NewMemoryRepository()

	err := SyncInventory(context.Background(), client, repo)
	if err == nil || !strings.Contains(err.Error(), "list Kubernetes nodes") {
		t.Fatalf("expected node list error, got %v", err)
	}
}

func TestFakeInventoryClientRespectsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &FakeInventoryClient{}
	if _, err := client.ListNodes(ctx); err == nil {
		t.Fatal("expected canceled context error")
	}
}
