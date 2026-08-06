// Package client — tipos do payload do dashboard SSE.
//
// Estes tipos mapeiam o JSON publicado pelo collector no tópico "dashboard"
// (event type "cluster"). O payload é o mesmo de GET /api/dashboard.
package client

// DashboardPayload é o payload completo do tópico SSE "dashboard".
type DashboardPayload struct {
	TotalServices    int32             `json:"total_services"`
	TotalContainers  int32             `json:"total_containers"`
	Cluster          ClusterInfo       `json:"cluster"`
	ClusterCapacity  ClusterCapacity   `json:"cluster_capacity"`
	AlertsSummary    AlertsSummary     `json:"alerts_summary"`
	NodesDistribution []NodeDist       `json:"nodes_distribution"`
	TopCPUConsumers  []ServiceConsumer `json:"top_cpu_consumers"`
	TopMemConsumers  []ServiceConsumer `json:"top_mem_consumers"`
}

// ClusterInfo é a info do cluster Swarm.
type ClusterInfo struct {
	ID             string   `json:"id"`
	NodesTotal     int      `json:"nodes_total"`
	ManagersTotal  int      `json:"managers_total"`
	WorkersTotal   int      `json:"workers_total"`
	NodesReady     int      `json:"nodes_ready"`
	NodesDown      int      `json:"nodes_down"`
	QuorumHealthy  bool     `json:"quorum_healthy"`
	SelfNodeID     string   `json:"self_node_id"`
	Warnings       []string `json:"warnings"`
}

// ClusterCapacity é a capacidade total + consumo agregado do cluster.
type ClusterCapacity struct {
	CPUTotal   float64 `json:"cpu_total"`   // núcleos totais (ex: 16)
	MemTotal   int64   `json:"mem_total"`   // bytes totais (ex: 8170692608)
	TasksTotal int32   `json:"tasks_total"` // tasks rodando
	CPUP95     float64 `json:"cpu_p95"`     // P95 de CPU do cluster (%)
	MemUsage   int64   `json:"mem_usage"`   // média de memória usada (bytes)
}

// AlertsSummary é o resumo de alertas.
type AlertsSummary struct {
	Total      int `json:"total"`
	OOMCount   int `json:"oom_count"`
	LeakCount  int `json:"leak_count"`
	DriftCount int `json:"drift_count"`
}

// NodeDist é um nó na distribuição de nós.
type NodeDist struct {
	Hostname     string `json:"hostname"`
	NodeID       string `json:"node_id"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	TasksRunning int32  `json:"tasks_running"`
}

// ServiceConsumer é um top consumer de CPU ou memória.
type ServiceConsumer struct {
	Name           string  `json:"name"`
	ContainerCount int32   `json:"container_count"`
	CPUP95         float64 `json:"cpu_p95"`
	MemP99         int64   `json:"mem_p99"`
}

// CPUPercent calcula a % de CPU usada em relação ao total do cluster.
// Retorna 0 se CPUTotal for 0.
func (c *ClusterCapacity) CPUPercent() float64 {
	if c.CPUTotal == 0 {
		return 0
	}
	// cpu_p95 já é uma porcentagem (0-100) do total disponível
	return c.CPUP95
}

// MemPercent calcula a % de memória usada em relação ao total do cluster.
// Retorna 0 se MemTotal for 0.
func (c *ClusterCapacity) MemPercent() float64 {
	if c.MemTotal == 0 {
		return 0
	}
	return float64(c.MemUsage) / float64(c.MemTotal) * 100
}

// MemTotalGB retorna a memória total em GB (para display).
func (c *ClusterCapacity) MemTotalGB() float64 {
	return float64(c.MemTotal) / (1024 * 1024 * 1024)
}

// MemUsageGB retorna a memória usada em GB (para display).
func (c *ClusterCapacity) MemUsageGB() float64 {
	return float64(c.MemUsage) / (1024 * 1024 * 1024)
}

// MemUsageMB retorna a memória usada em MB (para display).
func (c *ClusterCapacity) MemUsageMB() float64 {
	return float64(c.MemUsage) / (1024 * 1024)
}
