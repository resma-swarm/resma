package main

// mockLogEntry representa uma linha de log de um serviço.
type mockLogEntry struct {
	timestamp string
	level     string // INFO, WARN, ERROR, DEBUG
	message   string
}

// mockLogsFor retorna logs mockados para qualquer item (service, node, agent, task).
func mockLogsFor(name string) []mockLogEntry {
	switch name {
	// Services
	case "api":
		return apiLogs
	case "ml-inference":
		return mlLogs
	case "worker-3":
		return workerLogs
	case "postgres":
		return postgresLogs
	case "redis-cache":
		return redisLogs
	case "nginx-proxy":
		return nginxLogs
	case "frontend-dev":
		return frontendLogs
	case "batch-processor":
		return batchLogs
	// Nodes
	case "node-1":
		return node1Logs
	case "node-2":
		return node2Logs
	case "node-3":
		return node3Logs
	case "node-4":
		return node4Logs
	case "node-5":
		return node5Logs
	// Agents (usam mesmos IDs que nodes)
	case "agent-1":
		return agentLogs
	// Default
	default:
		return defaultLogs
	}
}

var apiLogs = []mockLogEntry{
	{"2026-01-15T14:32:01Z", "INFO", "Starting RESMA API server on :8080"},
	{"2026-01-15T14:32:01Z", "INFO", "Connected to DuckDB at /data/resma.duckdb"},
	{"2026-01-15T14:32:02Z", "INFO", "Docker client initialized (swarm: resma-swarm)"},
	{"2026-01-15T14:32:05Z", "INFO", "Metrics collector started (interval: 30s)"},
	{"2026-01-15T14:32:05Z", "INFO", "SSE broker initialized with 3 topics"},
	{"2026-01-15T14:32:10Z", "DEBUG", "GET /api/services -> 200 (12ms, 8 services)"},
	{"2026-01-15T14:32:15Z", "DEBUG", "GET /api/nodes -> 200 (8ms, 5 nodes)"},
	{"2026-01-15T14:32:20Z", "INFO", "Agent heartbeat from node-2 (version 7.8.1)"},
	{"2026-01-15T14:32:22Z", "INFO", "Agent heartbeat from node-3 (version 7.8.1)"},
	{"2026-01-15T14:32:25Z", "WARN", "High CPU on ml-inference: 89% (threshold: 85%)"},
	{"2026-01-15T14:32:25Z", "WARN", "High MEM on ml-inference: 89% (threshold: 85%)"},
	{"2026-01-15T14:32:28Z", "DEBUG", "POST /api/agent/ingest/metrics -> 201 (3ms)"},
	{"2026-01-15T14:32:30Z", "INFO", "Agent heartbeat from node-4 (version 7.8.0)"},
	{"2026-01-15T14:32:31Z", "WARN", "Agent version mismatch on node-4: 7.8.0 vs 7.8.1"},
	{"2026-01-15T14:32:35Z", "ERROR", "Agent heartbeat failed from node-5: connection refused"},
	{"2026-01-15T14:32:35Z", "ERROR", "Node node-5 is unreachable (last seen: 15m ago)"},
	{"2026-01-15T14:32:40Z", "DEBUG", "GET /api/services/health -> 200 (15ms)"},
	{"2026-01-15T14:32:42Z", "INFO", "SSE: 3 clients connected to [services] topic"},
	{"2026-01-15T14:32:45Z", "DEBUG", "GET /api/tasks -> 200 (18ms, 12 tasks)"},
	{"2026-01-15T14:32:50Z", "WARN", "batch-processor: all replicas stopped (0/2)"},
	{"2026-01-15T14:32:50Z", "ERROR", "Task task-7 failed on node-2: OOMKilled"},
	{"2026-01-15T14:32:50Z", "ERROR", "Task task-8 failed on node-3: OOMKilled"},
	{"2026-01-15T14:32:55Z", "INFO", "Recommendation engine tick (6 services analyzed)"},
	{"2026-01-15T14:33:00Z", "DEBUG", "GET /api/recommendations -> 200 (22ms, 6 recs)"},
	{"2026-01-15T14:33:02Z", "INFO", "SSE: broadcast [alerts] to 2 clients"},
	{"2026-01-15T14:33:05Z", "DEBUG", "POST /api/agent/ingest/metrics -> 201 (2ms)"},
	{"2026-01-15T14:33:10Z", "INFO", "Metrics flush: 240 rows written to DuckDB"},
	{"2026-01-15T14:33:15Z", "WARN", "Disk usage on node-4: 71% (threshold: 70%)"},
	{"2026-01-15T14:33:20Z", "DEBUG", "GET /api/agents -> 200 (10ms, 5 agents)"},
	{"2026-01-15T14:33:25Z", "INFO", "Agent heartbeat from node-1 (version 7.8.1)"},
}

var mlLogs = []mockLogEntry{
	{"2026-01-15T14:30:00Z", "INFO", "ML sidecar started on :8081"},
	{"2026-01-15T14:30:01Z", "INFO", "Loading scikit-learn model: resource_recommender_v3.pkl"},
	{"2026-01-15T14:30:02Z", "INFO", "Model loaded (accuracy: 0.94, features: 12)"},
	{"2026-01-15T14:30:05Z", "DEBUG", "Fetching services from Go API: /api/internal/services/with-metrics"},
	{"2026-01-15T14:30:06Z", "DEBUG", "Received 8 services with metrics"},
	{"2026-01-15T14:30:10Z", "INFO", "Running recommendation analysis for 8 services"},
	{"2026-01-15T14:30:15Z", "WARN", "ml-inference: MEM 89% approaching OOM threshold (90%)"},
	{"2026-01-15T14:30:15Z", "WARN", "Recommendation: increase ml-inference memory to 8Gi"},
	{"2026-01-15T14:30:20Z", "INFO", "Analysis complete: 6 recommendations generated"},
	{"2026-01-15T14:30:25Z", "DEBUG", "POST recommendations to Go API: /api/internal/recommendations"},
	{"2026-01-15T14:30:26Z", "INFO", "Go API accepted 6 recommendations"},
	{"2026-01-15T14:31:00Z", "DEBUG", "Fetching metrics for ml-inference"},
	{"2026-01-15T14:31:01Z", "WARN", "ml-inference CPU trend: sustained >80% for 15min"},
	{"2026-01-15T14:31:10Z", "INFO", "Memory leak detection: no leak detected (p-value: 0.12)"},
	{"2026-01-15T14:31:30Z", "DEBUG", "Fetching metrics for worker-3"},
	{"2026-01-15T14:31:35Z", "INFO", "worker-3: CPU 62% — recommend 3 replicas (current: 2)"},
	{"2026-01-15T14:32:00Z", "INFO", "Scheduled analysis tick (every 60s)"},
	{"2026-01-15T14:32:05Z", "DEBUG", "Fetching OOM count for batch-processor"},
	{"2026-01-15T14:32:06Z", "ERROR", "batch-processor: 2 OOM events in last 24h"},
	{"2026-01-15T14:32:10Z", "WARN", "batch-processor: all replicas stopped — investigate before scaling"},
}

var workerLogs = []mockLogEntry{
	{"2026-01-15T14:25:00Z", "INFO", "Worker-3 started (replica 1/2 on node-2)"},
	{"2026-01-15T14:25:01Z", "INFO", "Connected to Redis at redis-cache:6379"},
	{"2026-01-15T14:25:02Z", "INFO", "Connected to PostgreSQL at postgres:5432"},
	{"2026-01-15T14:25:05Z", "DEBUG", "Polling queue: jobs:pending (0 items)"},
	{"2026-01-15T14:25:30Z", "DEBUG", "Polling queue: jobs:pending (3 items)"},
	{"2026-01-15T14:25:31Z", "INFO", "Processing job #4521: cleanup_expired_sessions"},
	{"2026-01-15T14:25:35Z", "INFO", "Job #4521 complete (4.2s, deleted 152 sessions)"},
	{"2026-01-15T14:25:36Z", "INFO", "Processing job #4522: aggregate_metrics"},
	{"2026-01-15T14:25:48Z", "INFO", "Job #4522 complete (12.1s, 8400 rows)"},
	{"2026-01-15T14:25:49Z", "INFO", "Processing job #4523: send_alerts"},
	{"2026-01-15T14:25:52Z", "WARN", "Job #4523: alert delivery failed for 1 recipient"},
	{"2026-01-15T14:25:55Z", "INFO", "Job #4523 complete (6.3s, 5/6 delivered)"},
	{"2026-01-15T14:26:00Z", "DEBUG", "Polling queue: jobs:pending (0 items)"},
	{"2026-01-15T14:27:00Z", "WARN", "CPU sustained >60% for 2min"},
	{"2026-01-15T14:28:00Z", "WARN", "CPU sustained >60% for 3min"},
	{"2026-01-15T14:29:00Z", "WARN", "CPU sustained >60% for 4min"},
	{"2026-01-15T14:30:00Z", "WARN", "CPU sustained >60% for 5min"},
}

var postgresLogs = []mockLogEntry{
	{"2026-01-15T14:20:00Z", "INFO", "PostgreSQL 16.1 started on :5432"},
	{"2026-01-15T14:20:01Z", "INFO", "Database: resma, schema: public"},
	{"2026-01-15T14:20:02Z", "INFO", "WAL archive enabled (s3://resma-wal/)"},
	{"2026-01-15T14:20:05Z", "DEBUG", "Autovacuum running on table: metrics"},
	{"2026-01-15T14:20:10Z", "DEBUG", "Autovacuum complete (2400 dead tuples removed)"},
	{"2026-01-15T14:25:00Z", "INFO", "Checkpoint started"},
	{"2026-01-15T14:25:05Z", "INFO", "Checkpoint complete (42 buffers written)"},
	{"2026-01-15T14:30:00Z", "WARN", "Memory usage: 71% (threshold: 70%)"},
	{"2026-01-15T14:30:05Z", "DEBUG", "Slow query detected (1.2s): SELECT * FROM metrics WHERE..."},
	{"2026-01-15T14:31:00Z", "INFO", "Connection pool: 8/20 active"},
	{"2026-01-15T14:32:00Z", "DEBUG", "Connection pool: 5/20 active"},
}

var redisLogs = []mockLogEntry{
	{"2026-01-15T14:20:00Z", "INFO", "Redis 7.2 started on :6379"},
	{"2026-01-15T14:20:01Z", "INFO", "Persistence: AOF enabled (fsync: everysec)"},
	{"2026-01-15T14:20:05Z", "DEBUG", "Memory: 35% used (2.8GB / 8GB)"},
	{"2026-01-15T14:25:00Z", "DEBUG", "Stats: 1240 ops/s, 15 clients"},
	{"2026-01-15T14:30:00Z", "DEBUG", "Stats: 980 ops/s, 12 clients"},
	{"2026-01-15T14:32:00Z", "INFO", "Replica sync complete (node-2 -> node-3)"},
}

var nginxLogs = []mockLogEntry{
	{"2026-01-15T14:32:00Z", "INFO", "10.0.0.1 GET /api/services 200 4ms"},
	{"2026-01-15T14:32:05Z", "INFO", "10.0.0.2 GET /api/nodes 200 3ms"},
	{"2026-01-15T14:32:10Z", "INFO", "10.0.0.1 GET /api/services/health 200 5ms"},
	{"2026-01-15T14:32:15Z", "INFO", "10.0.0.3 GET / 200 2ms"},
	{"2026-01-15T14:32:20Z", "WARN", "10.0.0.4 POST /api/agent/ingest 413 (payload too large)"},
	{"2026-01-15T14:32:25Z", "INFO", "10.0.0.1 GET /api/tasks 200 8ms"},
	{"2026-01-15T14:32:30Z", "INFO", "10.0.0.2 GET /api/recommendations 200 12ms"},
	{"2026-01-15T14:32:35Z", "ERROR", "10.0.0.5 GET /api/agents 502 (upstream timeout)"},
}

var frontendLogs = []mockLogEntry{
	{"2026-01-15T14:30:00Z", "INFO", "Vite dev server started on :5173"},
	{"2026-01-15T14:30:01Z", "INFO", "HMR enabled, watching /src"},
	{"2026-01-15T14:30:05Z", "DEBUG", "Compiled /src/App.tsx in 245ms"},
	{"2026-01-15T14:30:10Z", "DEBUG", "Compiled /src/pages/Dashboard.tsx in 180ms"},
	{"2026-01-15T14:30:15Z", "INFO", "Client connected (10.0.0.1:5173)"},
	{"2026-01-15T14:32:00Z", "DEBUG", "Hot reload: /src/pages/Services.tsx updated"},
}

var batchLogs = []mockLogEntry{
	{"2026-01-15T14:10:00Z", "INFO", "Batch processor started (replica 1/2)"},
	{"2026-01-15T14:10:05Z", "INFO", "Loading batch config from /config/batch.yaml"},
	{"2026-01-15T14:10:10Z", "ERROR", "OOMKilled: exceeded memory limit of 2Gi"},
	{"2026-01-15T14:10:10Z", "ERROR", "Process exited with code 137 (SIGKILL)"},
	{"2026-01-15T14:10:22Z", "INFO", "Batch processor started (replica 2/2)"},
	{"2026-01-15T14:10:25Z", "ERROR", "OOMKilled: exceeded memory limit of 2Gi"},
	{"2026-01-15T14:10:25Z", "ERROR", "Process exited with code 137 (SIGKILL)"},
	{"2026-01-15T14:10:30Z", "ERROR", "All replicas stopped (0/2 running)"},
}

var defaultLogs = []mockLogEntry{
	{"2026-01-15T14:30:00Z", "INFO", "Service started"},
	{"2026-01-15T14:30:05Z", "INFO", "Health check passed"},
	{"2026-01-15T14:30:10Z", "DEBUG", "Ready"},
}

var node1Logs = []mockLogEntry{
	{"2026-01-15T14:20:00Z", "INFO", "node-1: Swarm manager initialized (leader)"},
	{"2026-01-15T14:20:05Z", "INFO", "node-1: Overlay network resma-net ready"},
	{"2026-01-15T14:25:00Z", "INFO", "node-1: 8 containers running (api, ml, frontend, worker)"},
	{"2026-01-15T14:30:00Z", "DEBUG", "node-1: CPU 45%, MEM 52%, Disk 38%"},
	{"2026-01-15T14:32:00Z", "INFO", "node-1: Agent heartbeat received (v7.8.1)"},
	{"2026-01-15T14:32:30Z", "DEBUG", "node-1: 240 metrics points ingested"},
}

var node2Logs = []mockLogEntry{
	{"2026-01-15T14:20:00Z", "INFO", "node-2: Swarm worker joined cluster"},
	{"2026-01-15T14:25:00Z", "INFO", "node-2: 4 containers running (worker, postgres)"},
	{"2026-01-15T14:30:00Z", "WARN", "node-2: CPU 72% sustained for 5min"},
	{"2026-01-15T14:32:00Z", "ERROR", "node-2: Task task-7 OOMKilled (batch-processor)"},
	{"2026-01-15T14:32:50Z", "INFO", "node-2: Agent heartbeat received (v7.8.1)"},
}

var node3Logs = []mockLogEntry{
	{"2026-01-15T14:20:00Z", "INFO", "node-3: Swarm worker joined cluster"},
	{"2026-01-15T14:25:00Z", "INFO", "node-3: 3 containers running (redis, nginx)"},
	{"2026-01-15T14:30:00Z", "DEBUG", "node-3: CPU 38%, MEM 41%, Disk 45%"},
	{"2026-01-15T14:32:00Z", "ERROR", "node-3: Task task-8 OOMKilled (batch-processor)"},
	{"2026-01-15T14:32:50Z", "INFO", "node-3: Agent heartbeat received (v7.8.1)"},
}

var node4Logs = []mockLogEntry{
	{"2026-01-15T14:20:00Z", "INFO", "node-4: Swarm worker joined cluster"},
	{"2026-01-15T14:25:00Z", "INFO", "node-4: 2 containers running (ml-inference replica)"},
	{"2026-01-15T14:30:00Z", "WARN", "node-4: Disk 71% (threshold: 70%)"},
	{"2026-01-15T14:31:00Z", "WARN", "node-4: Agent version mismatch: 7.8.0 vs 7.8.1"},
	{"2026-01-15T14:32:50Z", "INFO", "node-4: Agent heartbeat received (v7.8.0)"},
}

var node5Logs = []mockLogEntry{
	{"2026-01-15T14:20:00Z", "INFO", "node-5: Swarm worker joined cluster"},
	{"2026-01-15T14:25:00Z", "WARN", "node-5: Agent heartbeat missed (last seen: 5m ago)"},
	{"2026-01-15T14:30:00Z", "ERROR", "node-5: Agent unreachable (connection refused)"},
	{"2026-01-15T14:32:00Z", "ERROR", "node-5: Node marked as down (last seen: 15m ago)"},
	{"2026-01-15T14:33:00Z", "ERROR", "node-5: 2 tasks orphaned, rescheduling..."},
}

var agentLogs = []mockLogEntry{
	{"2026-01-15T14:30:00Z", "INFO", "RESMA Agent v7.8.1 started"},
	{"2026-01-15T14:30:01Z", "INFO", "Connected to Docker socket at /var/run/docker.sock"},
	{"2026-01-15T14:30:02Z", "INFO", "Pushing metrics to http://api:8080/api/agent/ingest/metrics"},
	{"2026-01-15T14:30:05Z", "DEBUG", "Collecting stats for 4 local containers"},
	{"2026-01-15T14:30:10Z", "DEBUG", "Heartbeat sent to http://api:8080/api/agent/heartbeat"},
	{"2026-01-15T14:30:15Z", "INFO", "Metrics buffer: 16 points flushed (4 containers × 4 metrics)"},
	{"2026-01-15T14:30:30Z", "DEBUG", "OOM event detected: batch-processor (exit code 137)"},
	{"2026-01-15T14:30:31Z", "INFO", "OOM event pushed to API"},
	{"2026-01-15T14:31:00Z", "DEBUG", "Collecting stats for 4 local containers"},
	{"2026-01-15T14:31:05Z", "DEBUG", "Heartbeat sent (uptime: 5m, metrics: 32 total)"},
}
