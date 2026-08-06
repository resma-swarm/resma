package tui

// Mock data types e dados para o wireframe.

type mockService struct {
	name     string
	replicas string
	cpu      float64
	mem      float64
	status   string
	spark    []float64
}

type mockNode struct {
	id       string
	hostname string
	role     string
	cpu      float64
	mem      float64
	disk     float64
	status   string
}

type mockAgent struct {
	nodeID   string
	status   string
	version  string
	lastSeen string
	services int
}

type mockTask struct {
	id      string
	service string
	node    string
	status  string
	desired string
	uptime  string
}

type mockAlert struct {
	level   string
	service string
	message string
	time    string
}

type mockRec struct {
	service string
	tier    string
	cpu     string
	mem     string
	risk    string
	reason  string
}

var mockServices = []mockService{
	{"api", "3/3", 45.2, 62.1, "running", []float64{30, 35, 42, 38, 45, 50, 48, 45, 52, 48, 45}},
	{"ml-inference", "1/1", 12.0, 89.0, "running", []float64{8, 10, 12, 15, 11, 13, 12, 14, 12, 13, 12}},
	{"frontend-dev", "1/1", 3.1, 18.5, "running", []float64{2, 3, 4, 3, 2, 3, 4, 3, 3, 4, 3}},
	{"worker-3", "2/2", 62.0, 40.0, "running", []float64{40, 45, 50, 55, 60, 65, 62, 58, 62, 64, 62}},
	{"postgres", "1/1", 22.0, 71.0, "running", []float64{18, 20, 22, 25, 23, 21, 22, 24, 22, 23, 22}},
	{"redis-cache", "2/2", 8.0, 35.0, "running", []float64{5, 6, 8, 10, 7, 8, 9, 8, 7, 8, 8}},
	{"nginx-proxy", "3/3", 5.0, 12.0, "running", []float64{3, 4, 5, 6, 5, 4, 5, 6, 5, 5, 5}},
	{"batch-processor", "0/2", 0, 0, "stopped", []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
}

var mockNodes = []mockNode{
	{"node-1", "swarm-manager-01", "manager", 52.0, 68.0, 45.0, "ready"},
	{"node-2", "swarm-worker-01", "worker", 78.0, 72.0, 62.0, "ready"},
	{"node-3", "swarm-worker-02", "worker", 35.0, 55.0, 38.0, "ready"},
	{"node-4", "swarm-worker-03", "worker", 91.0, 88.0, 71.0, "ready"},
	{"node-5", "swarm-worker-04", "worker", 0, 0, 0, "down"},
}

var mockAgents = []mockAgent{
	{"node-1", "active", "7.8.1", "2s ago", 8},
	{"node-2", "active", "7.8.1", "1s ago", 6},
	{"node-3", "active", "7.8.1", "3s ago", 5},
	{"node-4", "active", "7.8.0", "5s ago", 7},
	{"node-5", "offline", "7.8.1", "15m ago", 0},
}

var mockTasks = []mockTask{
	{"task-1", "api", "node-2", "running", "running", "2h15m"},
	{"task-2", "api", "node-3", "running", "running", "2h15m"},
	{"task-3", "api", "node-4", "running", "running", "2h15m"},
	{"task-4", "ml-inference", "node-4", "running", "running", "5h30m"},
	{"task-5", "worker-3", "node-2", "running", "running", "1h45m"},
	{"task-6", "worker-3", "node-3", "failed", "running", "0"},
	{"task-7", "batch-processor", "node-2", "failed", "running", "0"},
	{"task-8", "batch-processor", "node-3", "failed", "running", "0"},
	{"task-9", "postgres", "node-1", "running", "running", "12h00m"},
	{"task-10", "redis-cache", "node-2", "running", "running", "12h00m"},
	{"task-11", "redis-cache", "node-3", "running", "running", "12h00m"},
	{"task-12", "nginx-proxy", "node-1", "running", "running", "8h20m"},
}

var mockAlerts = []mockAlert{
	{"critical", "ml-inference", "Memory 89% — approaching OOM (90%)", "14:32:01"},
	{"warning", "node-4", "Disk 71% — cleanup recommended", "14:28:15"},
	{"warning", "worker-3", "CPU 62% — sustained 15min", "14:25:03"},
	{"critical", "batch-processor", "All replicas stopped (0/2)", "14:10:22"},
	{"info", "node-5", "Agent offline — last seen 15m ago", "14:18:00"},
	{"warning", "postgres", "Memory 71% — monitor closely", "14:05:11"},
}

var mockRecs = []mockRec{
	{"ml-inference", "conservative", "4 cores", "8Gi", "high", "Memory 89%, OOM risk. Increase to 8Gi."},
	{"api", "balanced", "2 cores", "4Gi", "low", "CPU p95 78%. Limits adequate, no change."},
	{"worker-3", "aggressive", "1.5 cores", "2Gi", "medium", "CPU 62%. Consider 3 replicas."},
	{"postgres", "conservative", "2 cores", "6Gi", "medium", "Memory 71%. Increase to 6Gi."},
	{"batch-processor", "balanced", "1 core", "2Gi", "high", "All replicas failed. Investigate first."},
	{"redis-cache", "balanced", "0.5 cores", "1Gi", "low", "Stable. No changes."},
}
