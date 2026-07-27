// Package docker — container cache e listagem.
package docker

import (
	"context"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// initContainerCache popula o cache a partir dos containers atuais.
func (c *Client) initContainerCache(ctx context.Context) error {
	result, err := c.cli.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return err
	}
	c.mu.Lock()
	for _, cnt := range result.Items {
		c.containerCache[cnt.ID] = parseContainer(cnt)
	}
	count := len(c.containerCache)
	c.mu.Unlock()
	c.log.Info("container cache inicializado", "count", count)
	return nil
}

// ListContainers retorna todos os containers do cache.
func (c *Client) ListContainers() []ContainerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ContainerInfo, 0, len(c.containerCache))
	for _, ci := range c.containerCache {
		out = append(out, ci)
	}
	return out
}

// AddContainerToCache faz inspect de um container e adiciona ao cache.
func (c *Client) AddContainerToCache(ctx context.Context, containerID string) error {
	result, err := c.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		c.log.Error("erro ao adicionar container ao cache", "id", containerID[:12], "err", err)
		return err
	}
	ci := parseInspect(result.Container)
	c.mu.Lock()
	c.containerCache[containerID] = ci
	c.mu.Unlock()
	c.log.Info("container adicionado ao cache", "name", ci.Name, "id", containerID[:12])
	return nil
}

// RemoveContainerFromCache remove um container do cache e para seu stats stream.
func (c *Client) RemoveContainerFromCache(containerID string) {
	c.mu.Lock()
	delete(c.containerCache, containerID)
	cancel, ok := c.streamCancel[containerID]
	if ok {
		cancel()
		delete(c.streamCancel, containerID)
	}
	delete(c.statsCache, containerID)
	c.mu.Unlock()
	c.log.Info("container removido do cache", "id", containerID[:12])
}

// GetRunningContainerIDs retorna o set de IDs de containers running.
func (c *Client) GetRunningContainerIDs(ctx context.Context) (map[string]bool, error) {
	filters := make(client.Filters).Add("status", "running")
	result, err := c.cli.ContainerList(ctx, client.ContainerListOptions{
		Filters: filters,
	})
	if err != nil {
		c.log.Error("erro ao listar running containers", "err", err)
		return nil, err
	}
	out := make(map[string]bool, len(result.Items))
	for _, cnt := range result.Items {
		out[cnt.ID] = true
	}
	return out, nil
}

// GetContainerNetworks retorna as networks de um container.
func (c *Client) GetContainerNetworks(ctx context.Context, containerID string) ([]ContainerNetwork, error) {
	result, err := c.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		c.log.Error("erro ao inspecionar container para networks", "id", containerID[:12], "err", err)
		return nil, err
	}
	var out []ContainerNetwork
	if result.Container.NetworkSettings != nil {
		for netName, ep := range result.Container.NetworkSettings.Networks {
			if ep == nil {
				continue
			}
			out = append(out, ContainerNetwork{
				Network:     netName,
				IPAddress:   ep.IPAddress.String(),
				IPv6Address: ep.GlobalIPv6Address.String(),
				MacAddress:  ep.MacAddress.String(),
				Gateway:     ep.Gateway.String(),
				EndpointID:  ep.EndpointID,
			})
		}
	}
	return out, nil
}

// parseContainer converte um container.Summary do Docker SDK em ContainerInfo.
func parseContainer(cnt container.Summary) ContainerInfo {
	labels := cnt.Labels
	service := ""
	if labels != nil {
		service = labels["com.docker.swarm.service.name"]
	}
	name := ""
	if len(cnt.Names) > 0 {
		name = strings.TrimPrefix(cnt.Names[0], "/")
	}
	state := string(cnt.State)
	if state == "" {
		state = "unknown"
	}
	return ContainerInfo{
		ID:      cnt.ID,
		Name:    name,
		Service: strings.TrimPrefix(service, "/"),
		State:   state,
		Image:   cnt.Image,
	}
}

// parseInspect converte um container.InspectResponse em ContainerInfo.
func parseInspect(info container.InspectResponse) ContainerInfo {
	labels := map[string]string{}
	if info.Config != nil && info.Config.Labels != nil {
		labels = info.Config.Labels
	}
	service := labels["com.docker.swarm.service.name"]
	name := strings.TrimPrefix(info.Name, "/")
	state := "stopped"
	if info.State != nil {
		state = string(info.State.Status)
		if state == "" {
			if info.State.Running {
				state = "running"
			} else {
				state = "stopped"
			}
		}
	}
	image := ""
	if info.Config != nil {
		image = info.Config.Image
	}
	networks := make(map[string]string)
	if info.NetworkSettings != nil {
		for netName, ep := range info.NetworkSettings.Networks {
			if ep == nil {
				continue
			}
			ip := ep.IPAddress.String()
			if ip != "" && ep.IPAddress.IsValid() {
				networks[netName] = ip
			}
		}
	}
	return ContainerInfo{
		ID:       info.ID,
		Name:     name,
		Service:  strings.TrimPrefix(service, "/"),
		State:    state,
		Image:    image,
		Networks: networks,
	}
}
