package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bnhminh1010/homelab-dashboard/internal/model"
)

// Client is a minimal Proxmox VE API client for monitoring (read-only).
type Client struct {
	baseURL    string
	token      string
	insecure   bool
	httpClient *http.Client
}

// NewClient creates a new Proxmox client.
func NewClient(baseURL, token string, insecure bool) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Proxmox URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("Proxmox URL must be http or https")
	}
	if u.Path != "" && u.Path != "/" {
		// Ensure base URL ends with /api2/json/
		if u.Path != "/api2/json/" {
			return nil, fmt.Errorf("Proxmox base URL should not include path (use host:port only), got %q", u.Path)
		}
	}
	base := u.Scheme + "://" + u.Host + "/api2/json/"

	return &Client{
		baseURL:  base,
		token:    token,
		insecure: insecure,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

func (c *Client) doReq(ctx context.Context, method, path string, query url.Values) (json.RawMessage, error) {
	reqURL := c.baseURL + path
	if query != nil {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "PVEAPIToken="+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("proxmox API %s %s: %d %v", method, path, resp.StatusCode, errResp)
	}

	var envelope map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return envelope["data"], nil
}

// getRaw does a GET and returns raw data field as json.RawMessage.
func (c *Client) getRaw(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	return c.doReq(ctx, "GET", path, query)
}

// ============ Cluster Resources ============

// ClusterResources represents the response from /cluster/resources?type=vm
type ClusterResources []ClusterResourceItem

type ClusterResourceItem struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Node         string  `json:"node"`
	VMID         int     `json:"vmid"`
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	CPU          float64 `json:"cpu"`
	MaxCPU       int     `json:"maxcpu"`
	Mem          uint64  `json:"mem"`
	MaxMem       uint64  `json:"maxmem"`
	Disk         uint64  `json:"disk"`
	MaxDisk      uint64  `json:"maxdisk"`
	NetIn        uint64  `json:"netin"`
	NetOut       uint64  `json:"netout"`
	DiskRead     uint64  `json:"diskread"`
	DiskWrite    uint64  `json:"diskwrite"`
	Uptime       uint64  `json:"uptime"`
	Template     int     `json:"template"`
	Lock         string  `json:"lock"`
	Tags         string  `json:"tags"`
	Pool         string  `json:"pool"`
	HAState      string  `json:"hastate"`
	MemHost      uint64  `json:"memhost"`
	RunningQemu  string  `json:"running-qemu"`
	RunningMachine string `json:"running-machine"`
	Agent        int     `json:"agent"`
	QMPStatus    string  `json:"qmpstatus"`
}

// GetClusterResources fetches all VMs and LXC containers in the cluster.
func (c *Client) GetClusterResources(ctx context.Context) (ClusterResources, error) {
	q := url.Values{}
	q.Set("type", "vm")
	data, err := c.getRaw(ctx, "cluster/resources", q)
	if err != nil {
		return nil, err
	}
	var res ClusterResources
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("unmarshal cluster/resources: %w", err)
	}
	return res, nil
}

// ============ Node Status ============

// NodeStatus represents /nodes/{node}/status
type NodeStatus struct {
	CPU       float64          `json:"cpu"`
	KVersion  string           `json:"kversion"`
	PVEVersion string          `json:"pveversion"`
	Uptime    uint64           `json:"uptime"`
	Idle      uint64           `json:"idle"`
	Wait      float64          `json:"wait"`
	LoadAvg   [3]float64       `json:"loadavg"`
	CPUInfo   NodeCPUInfo      `json:"cpuinfo"`
	Memory    NodeMemory       `json:"memory"`
	RootFS    NodeDisk         `json:"rootfs"`
	Swap      NodeSwap         `json:"swap"`
	KSM       NodeKSM          `json:"ksm"`
}

type NodeCPUInfo struct {
	Cores  int    `json:"cores"`
	CPUs   int    `json:"cpus"`
	Sockets int   `json:"sockets"`
	Model  string `json:"model"`
	MHz    string `json:"mhz"`
	Flags  string `json:"flags"`
	HVM    string `json:"hvm"`
	UserHz int    `json:"user_hz"`
}

type NodeMemory struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

type NodeDisk struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
	Avail uint64 `json:"avail"`
}

type NodeSwap struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
}

type NodeKSM struct {
	Shared uint64 `json:"shared"`
}

// GetNodeStatus fetches status for a single node.
func (c *Client) GetNodeStatus(ctx context.Context, node string) (NodeStatus, error) {
	data, err := c.getRaw(ctx, "nodes/"+url.PathEscape(node)+"/status", nil)
	if err != nil {
		return NodeStatus{}, err
	}
	var res NodeStatus
	if err := json.Unmarshal(data, &res); err != nil {
		return NodeStatus{}, fmt.Errorf("unmarshal node status: %w", err)
	}
	return res, nil
}

// ============ Storage ============

// StorageEntry represents a storage from /nodes/{node}/storage
type StorageEntry struct {
	Storage      string  `json:"storage"`
	Type         string  `json:"type"`
	Content      string  `json:"content"`
	Enabled      int     `json:"enabled"`
	Active       int     `json:"active"`
	Shared       int     `json:"shared"`
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Avail        uint64  `json:"avail"`
	UsedFraction float64 `json:"used_fraction"`
	Format       string  `json:"format"`
}

// GetNodeStorage fetches storage list for a node.
func (c *Client) GetNodeStorage(ctx context.Context, node string) ([]StorageEntry, error) {
	data, err := c.getRaw(ctx, "nodes/"+url.PathEscape(node)+"/storage", nil)
	if err != nil {
		return nil, err
	}
	var res []StorageEntry
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("unmarshal node storage: %w", err)
	}
	return res, nil
}

// ============ Tasks (Backup/Vzdump) ============

// TaskEntry represents a task from /nodes/{node}/tasks
type TaskEntry struct {
	UPID       string  `json:"upid"`
	Type       string  `json:"type"`
	Status     string  `json:"status"`
	StartTime  int64   `json:"starttime"`
	EndTime    int64   `json:"endtime"`
	User       string  `json:"user"`
	Node       string  `json:"node"`
	PID        int     `json:"pid"`
	PStart     int64   `json:"pstart"`
	ID         string  `json:"id"`
}

// GetVzdumpTasks fetches recent vzdump tasks for a node.
func (c *Client) GetVzdumpTasks(ctx context.Context, node string, limit int) ([]TaskEntry, error) {
	q := url.Values{}
	q.Set("typefilter", "vzdump")
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("source", "archive") // include finished tasks
	data, err := c.getRaw(ctx, "nodes/"+url.PathEscape(node)+"/tasks", q)
	if err != nil {
		return nil, err
	}
	var res []TaskEntry
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("unmarshal vzdump tasks: %w", err)
	}
	return res, nil
}

// VzdumpTaskToBackupStatus converts a vzdump task to model.BackupStatus.
func VzdumpTaskToBackupStatus(task TaskEntry) model.BackupStatus {
	status := "success"
	if task.Status != "OK" {
		status = "failed"
	}
	jobName := fmt.Sprintf("vzdump:%s/%s", task.Node, task.ID)
	var completedAt time.Time
	if task.EndTime > 0 {
		completedAt = time.Unix(task.EndTime, 0).UTC()
	}
	return model.BackupStatus{
		Job:         jobName,
		Status:      status,
		CompletedAt: completedAt,
		Message:     fmt.Sprintf("vzdump %s on %s", task.ID, task.Node),
	}
}

// ============ Convert to Model ============

// ToModelProxmoxNodes converts API responses to model.ProxmoxNode slice.
func ToModelProxmoxNodes(
	resources ClusterResources,
	nodeStatuses map[string]NodeStatus,
	nodeStorages map[string][]StorageEntry,
) []model.ProxmoxNode {
	// Group resources by node
	byNode := make(map[string][]ClusterResourceItem)
	for _, r := range resources {
		byNode[r.Node] = append(byNode[r.Node], r)
	}

	var nodes []model.ProxmoxNode
	for nodeName, items := range byNode {
		ns := nodeStatuses[nodeName]
		storages := nodeStorages[nodeName]

		// Build guests
		var guests []model.ProxmoxGuest
		for _, item := range items {
			if item.Template == 1 {
				continue // skip templates
			}
			g := model.ProxmoxGuest{
				VMID:           item.VMID,
				Type:           item.Type,
				Name:           item.Name,
				Status:         item.Status,
				CPU:            item.CPU,
				MaxCPU:         item.MaxCPU,
				MemoryBytes:    item.Mem,
				MaxMemoryBytes: item.MaxMem,
				DiskBytes:      item.Disk,
				MaxDiskBytes:   item.MaxDisk,
				NetInBytes:     item.NetIn,
				NetOutBytes:    item.NetOut,
				UptimeSeconds:  item.Uptime,
				Tags:           item.Tags,
				Node:           item.Node,
			}
			if item.Agent == 1 {
				// Agent info would be fetched separately if needed
				g.Agent = &model.ProxmoxGuestAgent{}
			}
			guests = append(guests, g)
		}

		// Build storages
		var modelStorages []model.ProxmoxStorage
		for _, s := range storages {
			modelStorages = append(modelStorages, model.ProxmoxStorage{
				Name:           s.Storage,
				Type:           s.Type,
				Content:        s.Content,
				Enabled:        s.Enabled,
				Active:         s.Active,
				Shared:         s.Shared,
				TotalBytes:     s.Total,
				UsedBytes:      s.Used,
				AvailBytes:     s.Avail,
				UsedFraction:   s.UsedFraction,
				Format:         s.Format,
			})
		}

		nodes = append(nodes, model.ProxmoxNode{
			Name:    nodeName,
			Version: ns.PVEVersion,
			Status:  "online",
			CPU:     ns.CPU,
			Memory: model.MemoryStats{
				TotalBytes:     ns.Memory.Total,
				UsedBytes:      ns.Memory.Used,
				AvailableBytes: ns.Memory.Free,
			},
			Storage: modelStorages,
			Guests:  guests,
		})
	}

	// Add offline nodes from nodeStatuses but not in resources
	for nodeName, ns := range nodeStatuses {
		if _, ok := byNode[nodeName]; !ok {
			nodes = append(nodes, model.ProxmoxNode{
				Name:   nodeName,
				Version: ns.PVEVersion,
				Status: "offline",
				Memory: model.MemoryStats{
					TotalBytes:     ns.Memory.Total,
					UsedBytes:      ns.Memory.Used,
					AvailableBytes: ns.Memory.Free,
				},
			})
		}
	}

	return nodes
}