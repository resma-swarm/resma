package api

// Service represents a RESMA service payload.
type Service struct {
	Name     string `json:"name"`
	Replicas int    `json:"replicas"`
	CPU      string `json:"cpu"`
	Mem      string `json:"mem"`
	Status   string `json:"status"`
}

// Container represents a container payload.
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Service string `json:"service"`
	CPU     string `json:"cpu"`
	Mem     string `json:"mem"`
}

// Node represents a cluster node payload.
type Node struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	CPU      string `json:"cpu"`
	Mem      string `json:"mem"`
}

// Agent represents an agent payload.
type Agent struct {
	NodeID   string `json:"nodeId"`
	Status   string `json:"status"`
	Version  string `json:"version"`
	LastSeen string `json:"lastSeen"`
}

// Task represents a task payload.
type Task struct {
	ID           string `json:"id"`
	Service      string `json:"service"`
	Node         string `json:"node"`
	Status       string `json:"status"`
	DesiredState string `json:"desiredState"`
}

// Recommendation represents a right-sizing recommendation payload.
type Recommendation struct {
	Service string `json:"service"`
	Tier    string `json:"tier"`
	CPU     string `json:"cpu"`
	Mem     string `json:"mem"`
	Risk    string `json:"risk"`
	Reason  string `json:"reason"`
}

// RollbackWatch represents a rollback watch payload.
type RollbackWatch struct {
	ID        string `json:"id"`
	Service   string `json:"service"`
	Status    string `json:"status"`
	WatchType string `json:"watchType"`
}

// Schedule represents a scheduled action payload.
type Schedule struct {
	ID          string `json:"id"`
	Service     string `json:"service"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduledAt"`
}

// Template represents a service template payload.
type Template struct {
	Name   string   `json:"name"`
	CPU    string   `json:"cpu"`
	Mem    string   `json:"mem"`
	Stacks []string `json:"stacks"`
}

// Alert represents an alert payload.
type Alert struct {
	Level     string `json:"level"`
	Service   string `json:"service"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// OOMEvent represents an OOM event payload.
type OOMEvent struct {
	Service   string `json:"service"`
	Container string `json:"container"`
	Timestamp string `json:"timestamp"`
}

// ChangeLogEntry represents a change log entry payload.
type ChangeLogEntry struct {
	Service   string `json:"service"`
	Action    string `json:"action"`
	User      string `json:"user"`
	Timestamp string `json:"timestamp"`
}

// Dashboard represents a dashboard summary payload.
type Dashboard struct {
	Services        []Service        `json:"services"`
	Nodes           []Node           `json:"nodes"`
	Agents          []Agent          `json:"agents"`
	Alerts          []Alert          `json:"alerts"`
	Recommendations []Recommendation `json:"recommendations"`
}

// StorageSummary represents a storage summary payload.
type StorageSummary struct {
	Total     string `json:"total"`
	Used      string `json:"used"`
	Available string `json:"available"`
}

// User represents a user payload.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// APIKey represents an API key payload.
type APIKey struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	CreatedAt string   `json:"createdAt"`
}

// Settings represents global settings payload.
type Settings struct {
	CollectInterval  int     `json:"collectInterval"`
	RetentionDays    int     `json:"retentionDays"`
	OutlierThreshold float64 `json:"outlierThreshold"`
}

// PrunePreview represents a prune preview payload.
type PrunePreview struct {
	StaleServices []string `json:"staleServices"`
	StaleNodes    []string `json:"staleNodes"`
	OrphanTasks   []string `json:"orphanTasks"`
	OldMetrics    []string `json:"oldMetrics"`
}

// SSEEvent represents a server-sent event payload.
type SSEEvent struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}
