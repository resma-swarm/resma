// Package docker — Swarm nodes e tasks.
package docker

import (
	"context"

	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
)

// GetNodes lista todos os nós do Swarm.
func (c *Client) GetNodes(ctx context.Context) ([]NodeInfo, error) {
	result, err := c.cli.NodeList(ctx, client.NodeListOptions{})
	if err != nil {
		c.log.Error("erro ao listar nós", "err", err)
		return nil, err
	}
	out := make([]NodeInfo, 0, len(result.Items))
	for _, node := range result.Items {
		out = append(out, parseNode(node))
	}
	return out, nil
}

// GetNodeDetail retorna detalhes de um nó pelo id.
func (c *Client) GetNodeDetail(ctx context.Context, nodeID string) (*NodeDetail, error) {
	result, err := c.cli.NodeInspect(ctx, nodeID, client.NodeInspectOptions{})
	if err != nil {
		c.log.Error("erro ao inspecionar nó", "id", nodeID, "err", err)
		return nil, err
	}
	ni := parseNode(result.Node)
	return &NodeDetail{
		NodeInfo:  ni,
		CreatedAt: result.Node.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: result.Node.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// parseNode converte um swarm.Node em NodeInfo.
// Note: Node.Description é um valor (não pointer) no SDK v0.5.0.
func parseNode(node swarm.Node) NodeInfo {
	desc := node.Description
	cpuTotal := float64(desc.Resources.NanoCPUs) / 1e9
	memTotal := int64(desc.Resources.MemoryBytes)

	isLeader := false
	reachability := ""
	if node.ManagerStatus != nil {
		isLeader = node.ManagerStatus.Leader
		reachability = string(node.ManagerStatus.Reachability)
	}

	labels := node.Spec.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	return NodeInfo{
		ID:            node.ID,
		Hostname:      desc.Hostname,
		Role:          string(node.Spec.Role),
		Availability:  string(node.Spec.Availability),
		Status:        string(node.Status.State),
		Address:       node.Status.Addr,
		CPUTotal:      cpuTotal,
		MemTotal:      memTotal,
		OS:            desc.Platform.OS,
		Architecture:  desc.Platform.Architecture,
		EngineVersion: desc.Engine.EngineVersion,
		IsLeader:      isLeader,
		Reachability:  reachability,
		Labels:        labels,
	}
}

// GetTasksByNode retorna um map node_id -> []TaskInfo.
func (c *Client) GetTasksByNode(ctx context.Context) (map[string][]TaskInfo, error) {
	tasks, err := c.GetTasks(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]TaskInfo)
	for _, t := range tasks {
		if t.NodeID == "" {
			continue
		}
		result[t.NodeID] = append(result[t.NodeID], t)
	}
	return result, nil
}

// GetTasks retorna a lista flat de todas as tasks do Swarm com service name
// resolvido, slot, desired_state e container_id (quando disponível).
// Usado pelo collector para popular a tabela tasks (Fase 7 — task lifecycle).
func (c *Client) GetTasks(ctx context.Context) ([]TaskInfo, error) {
	svcResult, err := c.cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		c.log.Error("erro ao listar serviços para tasks", "err", err)
		return nil, err
	}
	serviceNames := make(map[string]string, len(svcResult.Items))
	for _, svc := range svcResult.Items {
		serviceNames[svc.ID] = svc.Spec.Name
	}

	taskResult, err := c.cli.TaskList(ctx, client.TaskListOptions{})
	if err != nil {
		c.log.Error("erro ao listar tasks", "err", err)
		return nil, err
	}
	out := make([]TaskInfo, 0, len(taskResult.Items))
	for _, task := range taskResult.Items {
		var containerID string
		if task.Status.ContainerStatus != nil {
			containerID = task.Status.ContainerStatus.ContainerID
		}
		out = append(out, TaskInfo{
			ID:           task.ID,
			ServiceID:    task.ServiceID,
			ServiceName:  serviceNames[task.ServiceID],
			State:        string(task.Status.State),
			DesiredState: string(task.DesiredState),
			NodeID:       task.NodeID,
			Slot:         int32(task.Slot),
			ContainerID:  containerID,
		})
	}
	return out, nil
}

// GetContainerNodeMapping retorna mapeamentos container->nó->serviço
// apenas para tasks running com ContainerID.
func (c *Client) GetContainerNodeMapping(ctx context.Context) ([]ContainerNodeMapping, error) {
	svcResult, err := c.cli.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		c.log.Error("erro ao listar serviços para mapping", "err", err)
		return nil, err
	}
	serviceNames := make(map[string]string, len(svcResult.Items))
	for _, svc := range svcResult.Items {
		serviceNames[svc.ID] = svc.Spec.Name
	}

	taskResult, err := c.cli.TaskList(ctx, client.TaskListOptions{})
	if err != nil {
		c.log.Error("erro ao listar tasks para mapping", "err", err)
		return nil, err
	}
	var out []ContainerNodeMapping
	for _, task := range taskResult.Items {
		if task.Status.State != swarm.TaskStateRunning {
			continue
		}
		if task.Status.ContainerStatus == nil || task.Status.ContainerStatus.ContainerID == "" {
			continue
		}
		out = append(out, ContainerNodeMapping{
			ContainerID: task.Status.ContainerStatus.ContainerID,
			NodeID:      task.NodeID,
			Service:     serviceNames[task.ServiceID],
		})
	}
	return out, nil
}

// GetSwarmInfo retorna info do swarm atual.
func (c *Client) GetSwarmInfo(ctx context.Context) (*SwarmInfo, error) {
	infoResult, err := c.cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		c.log.Error("erro ao obter swarm info", "err", err)
		return &SwarmInfo{}, err
	}
	sw := infoResult.Info.Swarm
	si := &SwarmInfo{
		NodeID:           sw.NodeID,
		ControlAvailable: sw.ControlAvailable,
		Warnings:         sw.Warnings,
		NodesTotal:       int32(sw.Nodes),
		ManagersTotal:    int32(sw.Managers),
		WorkersTotal:     int32(sw.Nodes - sw.Managers),
	}
	if sw.Cluster != nil {
		si.ClusterID = sw.Cluster.ID
	}
	for _, m := range sw.RemoteManagers {
		si.RemoteManagers = append(si.RemoteManagers, m.Addr)
	}
	return si, nil
}
