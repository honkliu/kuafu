package k8s

import (
	"context"

	"github.com/microsoft/kuafu/internal/domain"
)

type FakeInventoryClient struct {
	Nodes   []*domain.Node
	GPUs    []*domain.GPU
	NodeErr error
	GPUErr  error
}

func (c *FakeInventoryClient) ListNodes(ctx context.Context) ([]*domain.Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.NodeErr != nil {
		return nil, c.NodeErr
	}
	return cloneNodes(c.Nodes), nil
}

func (c *FakeInventoryClient) ListGPUs(ctx context.Context) ([]*domain.GPU, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.GPUErr != nil {
		return nil, c.GPUErr
	}
	return cloneGPUs(c.GPUs), nil
}

func cloneNodes(nodes []*domain.Node) []*domain.Node {
	cloned := make([]*domain.Node, 0, len(nodes))
	for _, node := range nodes {
		nodeCopy := *node
		if node.Labels != nil {
			nodeCopy.Labels = make(map[string]string, len(node.Labels))
			for key, value := range node.Labels {
				nodeCopy.Labels[key] = value
			}
		}
		if node.InfiniBandIPs != nil {
			nodeCopy.InfiniBandIPs = append([]string(nil), node.InfiniBandIPs...)
		}
		cloned = append(cloned, &nodeCopy)
	}
	return cloned
}

func cloneGPUs(gpus []*domain.GPU) []*domain.GPU {
	cloned := make([]*domain.GPU, 0, len(gpus))
	for _, gpu := range gpus {
		gpuCopy := *gpu
		cloned = append(cloned, &gpuCopy)
	}
	return cloned
}
