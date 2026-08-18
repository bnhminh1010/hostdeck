package proxmox

import (
	"context"
	"testing"
	"time"
)

func TestClient_GetClusterResources(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), token, true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	resources, err := client.GetClusterResources(ctx)
	if err != nil {
		t.Fatalf("GetClusterResources: %v", err)
	}

	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	// Check first resource (nginx-proxy)
	if resources[0].VMID != 100 {
		t.Errorf("expected VMID 100, got %d", resources[0].VMID)
	}
	if resources[0].Name != "nginx-proxy" {
		t.Errorf("expected name nginx-proxy, got %s", resources[0].Name)
	}
	if resources[0].Type != "qemu" {
		t.Errorf("expected type qemu, got %s", resources[0].Type)
	}
	if resources[0].Status != "running" {
		t.Errorf("expected status running, got %s", resources[0].Status)
	}

	// Check LXC
	if resources[1].Type != "lxc" {
		t.Errorf("expected type lxc for second resource, got %s", resources[1].Type)
	}
}

func TestClient_GetNodeStatus(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), token, true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	status, err := client.GetNodeStatus(ctx, "pve")
	if err != nil {
		t.Fatalf("GetNodeStatus: %v", err)
	}

	if status.PVEVersion != "8.2.7" {
		t.Errorf("expected PVE version 8.2.7, got %s", status.PVEVersion)
	}
	if status.CPU != 0.183 {
		t.Errorf("expected CPU 0.183, got %f", status.CPU)
	}
	if status.Memory.Total != 34359738368 {
		t.Errorf("expected memory total 34359738368, got %d", status.Memory.Total)
	}
}

func TestClient_GetNodeStorage(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), token, true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	storages, err := client.GetNodeStorage(ctx, "pve")
	if err != nil {
		t.Fatalf("GetNodeStorage: %v", err)
	}

	if len(storages) != 2 {
		t.Fatalf("expected 2 storages, got %d", len(storages))
	}
	if storages[0].Storage != "local-lvm" {
		t.Errorf("expected storage local-lvm, got %s", storages[0].Storage)
	}
	if storages[1].Storage != "local" {
		t.Errorf("expected storage local, got %s", storages[1].Storage)
	}
}

func TestClient_GetVzdumpTasks(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), token, true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	tasks, err := client.GetVzdumpTasks(ctx, "pve", 10)
	if err != nil {
		t.Fatalf("GetVzdumpTasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Fatalf("expected 3 vzdump tasks, got %d", len(tasks))
	}
	if tasks[0].Status != "OK" {
		t.Errorf("expected task status OK, got %s", tasks[0].Status)
	}
	if tasks[2].Status != "error" {
		t.Errorf("expected task status error, got %s", tasks[2].Status)
	}
}

func TestToModelProxmoxNodes(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), token, true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	resources, err := client.GetClusterResources(ctx)
	if err != nil {
		t.Fatalf("GetClusterResources: %v", err)
	}

	// Fetch node status and storage
	nodeStatuses := make(map[string]NodeStatus)
	nodeStorages := make(map[string][]StorageEntry)
	for _, r := range resources {
		if _, ok := nodeStatuses[r.Node]; !ok {
			ns, err := client.GetNodeStatus(ctx, r.Node)
			if err != nil {
				t.Logf("GetNodeStatus for %s: %v", r.Node, err)
				continue
			}
			nodeStatuses[r.Node] = ns
			storages, err := client.GetNodeStorage(ctx, r.Node)
			if err != nil {
				t.Logf("GetNodeStorage for %s: %v", r.Node, err)
				continue
			}
			nodeStorages[r.Node] = storages
		}
	}

	nodes := ToModelProxmoxNodes(resources, nodeStatuses, nodeStorages)

	if len(nodes) != 1 {
		t.Fatalf("expected 1 Proxmox node, got %d", len(nodes))
	}

	node := nodes[0]
	if node.Name != "pve" {
		t.Errorf("expected node name pve, got %s", node.Name)
	}
	if node.Version != "8.2.7" {
		t.Errorf("expected version 8.2.7, got %s", node.Version)
	}
	if node.Status != "online" {
		t.Errorf("expected status online, got %s", node.Status)
	}
	if len(node.Guests) != 3 {
		t.Errorf("expected 3 guests, got %d", len(node.Guests))
	}
	if len(node.Storage) != 2 {
		t.Errorf("expected 2 storages, got %d", len(node.Storage))
	}

	// Check guest conversion
	var nginxFound, plexFound, haFound bool
	for _, g := range node.Guests {
		if g.VMID == 100 && g.Name == "nginx-proxy" {
			nginxFound = true
			if g.Type != "qemu" {
				t.Errorf("nginx-proxy type should be qemu, got %s", g.Type)
			}
			if g.Status != "running" {
				t.Errorf("nginx-proxy status should be running, got %s", g.Status)
			}
			if g.MaxCPU != 2 {
				t.Errorf("nginx-proxy maxcpu should be 2, got %d", g.MaxCPU)
			}
		}
		if g.VMID == 200 && g.Name == "plex" {
			plexFound = true
			if g.Type != "lxc" {
				t.Errorf("plex type should be lxc, got %s", g.Type)
			}
		}
		if g.VMID == 101 && g.Name == "homeassistant" {
			haFound = true
			if g.Status != "stopped" {
				t.Errorf("homeassistant status should be stopped, got %s", g.Status)
			}
		}
	}
	if !nginxFound {
		t.Error("nginx-proxy not found in guests")
	}
	if !plexFound {
		t.Error("plex not found in guests")
	}
	if !haFound {
		t.Error("homeassistant not found in guests")
	}
}

func TestCollector_ProxmoxNodes(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), token, true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	collector := NewCollector(client, 100*time.Millisecond)
	source := ProxmoxSourceFunc(collector)

	ctx := context.Background()

	// First call should fetch fresh data
	nodes, err := source(ctx)
	if err != nil {
		t.Fatalf("first ProxmoxNodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	// Second call within poll interval should return cached
	nodes2, err := source(ctx)
	if err != nil {
		t.Fatalf("second ProxmoxNodes: %v", err)
	}
	if len(nodes2) != 1 {
		t.Fatalf("expected 1 node on second call, got %d", len(nodes2))
	}

	// Verify guests are present
	if len(nodes[0].Guests) != 3 {
		t.Errorf("expected 3 guests, got %d", len(nodes[0].Guests))
	}
}

func TestVzdumpTaskToBackupStatus(t *testing.T) {
	task := TaskEntry{
		UPID:      "UPID:pve:00001234:00001234:68A1B2C3:vzdump:100:root@pam:",
		Type:      "vzdump",
		Status:    "OK",
		StartTime: time.Now().Add(-2 * time.Hour).Unix(),
		EndTime:   time.Now().Add(-118 * time.Minute).Unix(),
		User:      "root@pam",
		Node:      "pve",
		PID:       1234,
		PStart:    1234,
		ID:        "qemu/100",
	}

	backup := VzdumpTaskToBackupStatus(task)

	if backup.Job != "vzdump:pve/qemu/100" {
		t.Errorf("expected job vzdump:pve/qemu/100, got %s", backup.Job)
	}
	if backup.Status != "success" {
		t.Errorf("expected status success, got %s", backup.Status)
	}
	if backup.CompletedAt.IsZero() {
		t.Error("completedAt should not be zero")
	}
	if backup.Message == "" {
		t.Error("message should not be empty")
	}

	// Test failed task
	task.Status = "error"
	backup2 := VzdumpTaskToBackupStatus(task)
	if backup2.Status != "failed" {
		t.Errorf("expected status failed for error task, got %s", backup2.Status)
	}
}

func TestClient_AuthFailure(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), "wrong-token", true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetClusterResources(ctx)
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	// Should get 401 Unauthorized
	t.Logf("Auth error (expected): %v", err)
}

func TestClient_NodeNotFound(t *testing.T) {
	token := "test-token"
	mock := NewMockServer(token)
	defer mock.Close()
	mock.SetData(DefaultMockData(token))

	client, err := NewClient(mock.URL(), token, true)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetNodeStatus(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected not found error for nonexistent node")
	}
}