package k8s

import (
	"context"
	"fmt"

	"github.com/microsoft/kuafu/internal/domain"
)

type InventoryClient interface {
	ListNodes(ctx context.Context) ([]*domain.Node, error)
	ListGPUs(ctx context.Context) ([]*domain.GPU, error)
}

type InventoryRepository interface {
	AddNode(*domain.Node) error
	AddGPU(*domain.GPU) error
}

func SyncInventory(ctx context.Context, client InventoryClient, repo InventoryRepository) error {
	nodes, err := client.ListNodes(ctx)
	if err != nil {
		return fmt.Errorf("list Kubernetes nodes: %w", err)
	}
	for _, node := range nodes {
		if err := repo.AddNode(node); err != nil {
			return fmt.Errorf("store Kubernetes node %s: %w", node.Name, err)
		}
	}

	gpus, err := client.ListGPUs(ctx)
	if err != nil {
		return fmt.Errorf("list Kubernetes GPUs: %w", err)
	}
	for _, gpu := range gpus {
		if err := repo.AddGPU(gpu); err != nil {
			return fmt.Errorf("store Kubernetes GPU %s: %w", gpu.ID, err)
		}
	}

	return nil
}
