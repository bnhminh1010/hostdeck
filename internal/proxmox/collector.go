package proxmox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bnhminh1010/homelab-dashboard/internal/metrics"
	"github.com/bnhminh1010/homelab-dashboard/internal/model"
)

// Collector implements metrics.ProxmoxSource by polling the Proxmox API.
type Collector struct {
	client     *Client
	pollInterval time.Duration
	mu         sync.RWMutex
	lastNodes  []model.ProxmoxNode
	lastErr    error
	lastFetch  time.Time
}

// NewCollector creates a new Proxmox collector.
func NewCollector(client *Client, pollInterval time.Duration) *Collector {
	if pollInterval <= 0 {
		pollInterval = 30 * time.Second
	}
	return &Collector{
		client:       client,
		pollInterval: pollInterval,
	}
}

// ProxmoxNodes returns the last successfully collected Proxmox nodes.
// It triggers a background refresh if data is stale.
// If no data has been fetched yet, it blocks until the first fetch completes.
func (c *Collector) ProxmoxNodes(ctx context.Context) ([]model.ProxmoxNode, error) {
	c.mu.RLock()
	stale := time.Since(c.lastFetch) > c.pollInterval
	lastNodes := c.lastNodes
	lastErr := c.lastErr
	c.mu.RUnlock()

	if stale {
		// Fire-and-forget refresh
		go c.refresh(context.Background())
	}

	// If no data yet, wait for first fetch
	if lastNodes == nil && lastErr == nil {
		// Trigger fetch and wait
		done := make(chan struct{})
		go func() {
			c.refresh(context.Background())
			close(done)
		}()
		select {
		case <-done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		// Re-read after fetch
		c.mu.RLock()
		lastNodes = c.lastNodes
		lastErr = c.lastErr
		c.mu.RUnlock()
	}

	if lastErr != nil && lastNodes == nil {
		return nil, lastErr
	}
	return lastNodes, nil
}

// refresh fetches fresh data from Proxmox API.
func (c *Collector) refresh(ctx context.Context) {
	nodes, err := c.fetchAll(ctx)
	c.mu.Lock()
	c.lastFetch = time.Now()
	if err != nil {
		c.lastErr = err
	} else {
		c.lastNodes = nodes
		c.lastErr = nil
	}
	c.mu.Unlock()
}

// fetchAll collects all data needed for Proxmox nodes.
func (c *Collector) fetchAll(ctx context.Context) ([]model.ProxmoxNode, error) {
	// 1. Get cluster resources (VMs + LXC)
	resources, err := c.client.GetClusterResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("cluster/resources: %w", err)
	}

	// Group by node
	byNode := make(map[string][]ClusterResourceItem)
	for _, r := range resources {
		byNode[r.Node] = append(byNode[r.Node], r)
	}

	// 2. Fetch node status for each node
	nodeStatuses := make(map[string]NodeStatus)
	for nodeName := range byNode {
		ns, err := c.client.GetNodeStatus(ctx, nodeName)
		if err != nil {
			// Don't fail entirely if one node is unreachable
			continue
		}
		nodeStatuses[nodeName] = ns
	}

	// 3. Fetch storage for each node
	nodeStorages := make(map[string][]StorageEntry)
	for nodeName := range byNode {
		storages, err := c.client.GetNodeStorage(ctx, nodeName)
		if err != nil {
			continue
		}
		nodeStorages[nodeName] = storages
	}

	// 4. Convert to model
	return ToModelProxmoxNodes(resources, nodeStatuses, nodeStorages), nil
}

// ProxmoxSourceFunc wraps a Collector as a ProxmoxSource.
func ProxmoxSourceFunc(c *Collector) metrics.ProxmoxSourceFunc {
	return func(ctx context.Context) ([]model.ProxmoxNode, error) {
		return c.ProxmoxNodes(ctx)
	}
}