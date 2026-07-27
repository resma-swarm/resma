// Package docker — tipos compartilhados.
package docker

import "time"

// ContainerInfo é a versão parseada de um container (do cache).
type ContainerInfo struct {
	ID       string
	Name     string
	Service  string
	State    string
	Image    string
	Networks map[string]string // network_name -> ip
}

// Stats é a versão parseada de Docker stats (uma leitura point-in-time).
type Stats struct {
	// Timestamp da leitura
	Read time.Time

	// CPU
	CPUPercent float64
	CPUUsage   int64
	CPUSystem  int64
	OnlineCPUs int64

	// CPU throttling (0b.2)
	CPUThrottledPeriods int64
	CPUThrottledTime    int64

	// Memory
	MemUsage      int64
	MemLimit      int64
	MemPercent    float64
	MemWorkingSet int64 // working_set = usage - inactive_file/cache (0b.2)

	// Network
	NetRX int64
	NetTX int64

	// Block IO
	BlockRead  int64
	BlockWrite int64
}

// ServiceInfo é a versão parseada de um Swarm service.
type ServiceInfo struct {
	ID       string
	Name     string
	Image    string
	Replicas int64
	Labels   map[string]string
}

// ServiceResources representa limites/reservations de um serviço.
type ServiceResources struct {
	CPULimit       float64
	MemLimit       int64
	CPUReservation float64
	MemReservation int64
}

// ServiceStatusMap é um map service_name -> "online"/"offline".
type ServiceStatusMap map[string]string

// NodeInfo é a versão parseada de um nó do Swarm.
type NodeInfo struct {
	ID            string
	Hostname      string
	Role          string
	Availability  string
	Status        string
	Address       string
	CPUTotal      float64
	MemTotal      int64
	OS            string
	Architecture  string
	EngineVersion string
	IsLeader      bool
	Reachability  string
	Labels        map[string]string
}

// NodeDetail é NodeInfo + timestamps.
type NodeDetail struct {
	NodeInfo
	CreatedAt string
	UpdatedAt string
}

// TaskInfo é a versão parseada de uma task do Swarm.
type TaskInfo struct {
	ID           string
	ServiceID    string
	ServiceName  string
	State        string
	DesiredState string
	NodeID       string
	Slot         int32
	ContainerID  string
}

// SwarmInfo é a versão parseada do swarm info.
type SwarmInfo struct {
	ClusterID        string
	NodeID           string
	NodesTotal       int32
	ManagersTotal    int32
	WorkersTotal     int32
	ControlAvailable bool
	Warnings         []string
	RemoteManagers   []string
}

// ContainerNodeMapping mapeia container -> nó -> serviço.
type ContainerNodeMapping struct {
	ContainerID string
	NodeID      string
	Service     string
}

// SystemDF é o resultado de `docker system df`.
type SystemDF struct {
	Images        DFImages
	Containers    DFContainers
	Volumes       DFVolumes
	TopVolumes    []DFVolume
	OrphanVolumes []DFVolume
}

type DFImages struct {
	Count       int32
	TotalSize   int64
	Reclaimable int64
}

type DFContainers struct {
	Count     int32
	TotalSize int64
}

type DFVolumes struct {
	Count       int32
	TotalSize   int64
	Reclaimable int64
	OrphanCount int32
	OrphanSize  int64
}

type DFVolume struct {
	Name    string
	Size    int64
	InUse   bool
	Driver  string
	Created string
}

// VolumeInfo é a versão parseada de um volume.
type VolumeInfo struct {
	Name        string
	Driver      string
	Mountpoint  string
	Size        int64
	Reclaimable int64
	InUse       bool
	Labels      map[string]string
	Options     map[string]string
	Scope       string
	Created     string
}

// NetworkInfo é a versão parseada de uma network.
type NetworkInfo struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	Attachable bool
	Ingress    bool
	Subnets    []string
	Gateways   []string
	Containers []NetworkContainer
	Created    string
	Labels     map[string]string
	Options    map[string]string
}

// NetworkContainer é um container dentro de uma network.
type NetworkContainer struct {
	ID   string
	Name string
	IPv4 string
	IPv6 string
}

// ContainerNetwork é a info de network de um container.
type ContainerNetwork struct {
	Network     string
	IPAddress   string
	IPv6Address string
	MacAddress  string
	Gateway     string
	EndpointID  string
}

// UpdateServiceResult é o resultado de update_service_resources.
type UpdateServiceResult struct {
	Success       bool
	Warnings      []string
	Error         string
	VersionBefore int64
	VersionAfter  int64
}
