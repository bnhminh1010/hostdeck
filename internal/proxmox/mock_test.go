package proxmox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MockServer provides a fake Proxmox API for testing.
type MockServer struct {
	server *httptest.Server
	token  string
	mu     sync.Mutex
	data   MockData
}

// MockData holds the test data served by the mock.
type MockData struct {
	Nodes       map[string]MockNode
	ClusterRes  []ClusterResourceItem
	Tasks       map[string][]TaskEntry
}

type MockNode struct {
	Name        string
	Version     string
	Status      string
	CPU         float64
	MemoryTotal uint64
	MemoryUsed  uint64
	Storage     []StorageEntry
	RRDData     []RRDEntry
}

type RRDEntry struct {
	Time     int64   `json:"time"`
	CPU      float64 `json:"cpu"`
	MemUsed  uint64  `json:"memused"`
	MemTotal uint64  `json:"memtotal"`
}

// NewMockServer creates a new mock Proxmox API server.
func NewMockServer(token string) *MockServer {
	m := &MockServer{token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/", m.handleAPI)
	m.server = httptest.NewServer(mux)
	return m
}

// URL returns the base URL of the mock server.
func (m *MockServer) URL() string {
	return m.server.URL
}

// Close shuts down the mock server.
func (m *MockServer) Close() {
	m.server.Close()
}

// SetData configures the mock data.
func (m *MockServer) SetData(data MockData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = data
}

func (m *MockServer) handleAPI(w http.ResponseWriter, r *http.Request) {
	// Check auth
	auth := r.Header.Get("Authorization")
	expectedAuth := "PVEAPIToken=" + m.token
	if auth != expectedAuth {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{"errors": "invalid token"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api2/json/")
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "cluster/resources" && r.URL.Query().Get("type") == "vm":
		m.handleClusterResources(w)
	case path == "version":
		m.handleVersion(w)
	case strings.HasPrefix(path, "nodes/") && strings.HasSuffix(path, "/status"):
		m.handleNodeStatus(w, path)
	case strings.HasPrefix(path, "nodes/") && strings.HasSuffix(path, "/storage"):
		m.handleNodeStorage(w, path)
	case strings.HasPrefix(path, "nodes/") && strings.HasSuffix(path, "/tasks"):
		m.handleTasks(w, path, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"errors": "not found: " + path})
	}
}

func (m *MockServer) handleClusterResources(w http.ResponseWriter) {
	m.mu.Lock()
	data := m.data.ClusterRes
	m.mu.Unlock()
	if data == nil {
		data = []ClusterResourceItem{}
	}
	writeJSON(w, map[string]any{"data": data})
}

func (m *MockServer) handleVersion(w http.ResponseWriter) {
	writeJSON(w, map[string]any{"data": map[string]string{
		"version": "8.2.7",
		"release": "pve-manager/8.2.7",
		"repoid":  "pve-no-subscription",
	}})
}

func (m *MockServer) handleNodeStatus(w http.ResponseWriter, path string) {
	// Extract node name: nodes/{node}/status
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.NotFound(w, nil)
		return
	}
	nodeName := parts[1]

	m.mu.Lock()
	node := m.data.Nodes[nodeName]
	m.mu.Unlock()

	if node.Name == "" {
		http.NotFound(w, nil)
		return
	}

	writeJSON(w, map[string]any{"data": NodeStatus{
		CPU:        node.CPU,
		KVersion:   "Linux 6.8.12-4-pve",
		PVEVersion: node.Version,
		Uptime:     172800,
		Idle:       141120,
		Wait:       0.0,
		LoadAvg:    [3]float64{0.84, 0.71, 0.66},
		CPUInfo: NodeCPUInfo{
			Cores: 8, CPUs: 8, Sockets: 1,
			Model: "AMD Ryzen 7 7840HS", MHz: "3792.587",
			Flags: "fp asimd evtstrm", HVM: "1", UserHz: 100,
		},
		Memory: NodeMemory{Total: node.MemoryTotal, Used: node.MemoryUsed, Free: node.MemoryTotal - node.MemoryUsed},
		RootFS: NodeDisk{Total: 500107862016, Used: 214748364800, Free: 285359497216, Avail: 285359497216},
		Swap:   NodeSwap{Total: 8589934592, Used: 0, Free: 8589934592},
		KSM:    NodeKSM{Shared: 0},
	}})
}

func (m *MockServer) handleNodeStorage(w http.ResponseWriter, path string) {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.NotFound(w, nil)
		return
	}
	nodeName := parts[1]

	m.mu.Lock()
	node := m.data.Nodes[nodeName]
	m.mu.Unlock()

	if node.Name == "" {
		http.NotFound(w, nil)
		return
	}

	if node.Storage == nil {
		node.Storage = []StorageEntry{}
	}
	writeJSON(w, map[string]any{"data": node.Storage})
}

func (m *MockServer) handleTasks(w http.ResponseWriter, path string, r *http.Request) {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		http.NotFound(w, nil)
		return
	}
	nodeName := parts[1]

	m.mu.Lock()
	tasks := m.data.Tasks[nodeName]
	m.mu.Unlock()

	if tasks == nil {
		tasks = []TaskEntry{}
	}

	// Apply filters
	typeFilter := r.URL.Query().Get("typefilter")
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	var filtered []TaskEntry
	for _, t := range tasks {
		if typeFilter != "" && t.Type != typeFilter {
			continue
		}
		filtered = append(filtered, t)
		if len(filtered) >= limit {
			break
		}
	}

	writeJSON(w, map[string]any{"data": filtered})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// ============ Test Helpers ============

// DefaultMockData returns a reasonable test dataset.
func DefaultMockData(token string) MockData {
	return MockData{
		Nodes: map[string]MockNode{
			"pve": {
				Name:        "pve",
				Version:     "8.2.7",
				Status:      "online",
				CPU:         0.183,
				MemoryTotal: 34359738368,
				MemoryUsed:  12582912000,
				Storage: []StorageEntry{
					{Storage: "local-lvm", Type: "lvmthin", Content: "images,rootdir", Enabled: 1, Active: 1, Shared: 0, Total: 1000000000000, Used: 400000000000, Avail: 600000000000, UsedFraction: 0.4, Format: "raw"},
					{Storage: "local", Type: "dir", Content: "backup,vztmpl,iso", Enabled: 1, Active: 1, Shared: 0, Total: 500000000000, Used: 100000000000, Avail: 400000000000, UsedFraction: 0.2, Format: "raw"},
				},
			},
		},
		ClusterRes: []ClusterResourceItem{
			{ID: "qemu/100", Type: "qemu", Node: "pve", VMID: 100, Name: "nginx-proxy", Status: "running", CPU: 0.042, MaxCPU: 2, Mem: 2147483648, MaxMem: 4294967296, Disk: 8388608000, MaxDisk: 34359738368, NetIn: 123456789, NetOut: 987654321, DiskRead: 0, DiskWrite: 0, Uptime: 86400, Template: 0, Lock: "", Tags: "prod", Pool: "", HAState: "", MemHost: 2199023256, RunningQemu: "8.1.5", RunningMachine: "pc-q35-8.1", Agent: 1, QMPStatus: "running"},
			{ID: "lxc/200", Type: "lxc", Node: "pve", VMID: 200, Name: "plex", Status: "running", CPU: 0.12, MaxCPU: 4, Mem: 4294967296, MaxMem: 8589934592, Disk: 50000000000, MaxDisk: 100000000000, NetIn: 5000000000, NetOut: 2000000000, DiskRead: 100000000, DiskWrite: 50000000, Uptime: 172800, Template: 0, Lock: "", Tags: "media", Pool: "", HAState: "", MemHost: 4398046512, Agent: 1},
			{ID: "qemu/101", Type: "qemu", Node: "pve", VMID: 101, Name: "homeassistant", Status: "stopped", CPU: 0, MaxCPU: 2, Mem: 0, MaxMem: 4294967296, Disk: 15000000000, MaxDisk: 34359738368, NetIn: 0, NetOut: 0, DiskRead: 0, DiskWrite: 0, Uptime: 0, Template: 0, Lock: "", Tags: "iot", Pool: "", HAState: "", MemHost: 0, Agent: 0},
		},
		Tasks: map[string][]TaskEntry{
			"pve": {
				{UPID: "UPID:pve:00001234:00001234:68A1B2C3:vzdump:100:root@pam:", Type: "vzdump", Status: "OK", StartTime: time.Now().Add(-2*time.Hour).Unix(), EndTime: time.Now().Add(-118*time.Minute).Unix(), User: "root@pam", Node: "pve", PID: 1234, PStart: 1234, ID: "qemu/100"},
				{UPID: "UPID:pve:00001235:00001235:68A1B2C4:vzdump:200:root@pam:", Type: "vzdump", Status: "OK", StartTime: time.Now().Add(-24*time.Hour).Unix(), EndTime: time.Now().Add(-23*time.Hour).Unix(), User: "root@pam", Node: "pve", PID: 1235, PStart: 1235, ID: "lxc/200"},
				{UPID: "UPID:pve:00001236:00001236:68A1B2C5:vzdump:101:root@pam:", Type: "vzdump", Status: "error", StartTime: time.Now().Add(-48*time.Hour).Unix(), EndTime: time.Now().Add(-47*time.Hour).Unix(), User: "root@pam", Node: "pve", PID: 1236, PStart: 1236, ID: "qemu/101"},
			},
		},
	}
}