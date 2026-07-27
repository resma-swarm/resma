// Package docker — system df, volumes e networks.
package docker

import (
	"context"
	"sort"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"
)

// GetSystemDF retorna o resultado de `docker system df`.
// Usa DiskUsageOptions para solicitar apenas os componentes necessários.
func (c *Client) GetSystemDF(ctx context.Context) (*SystemDF, error) {
	du, err := c.cli.DiskUsage(ctx, client.DiskUsageOptions{
		Containers: true,
		Images:     true,
		Volumes:    true,
	})
	if err != nil {
		c.log.Error("erro ao obter system df", "err", err)
		return &SystemDF{
			Images:     DFImages{},
			Containers: DFContainers{},
			Volumes:    DFVolumes{},
		}, err
	}

	// Images
	imgCount := int32(du.Images.TotalCount)
	imgTotal := du.Images.TotalSize
	imgReclaimable := du.Images.Reclaimable

	// Containers
	cntCount := int32(du.Containers.TotalCount)
	cntTotal := du.Containers.TotalSize

	// Volumes
	volCount := int32(du.Volumes.TotalCount)
	volTotal := du.Volumes.TotalSize
	volReclaimable := du.Volumes.Reclaimable

	var orphanVolumes []DFVolume
	for _, v := range du.Volumes.Items {
		size := volumeSize(v)
		inUse := volumeInUse(v)
		dv := DFVolume{
			Name:    v.Name,
			Size:    size,
			InUse:   inUse,
			Driver:  v.Driver,
			Created: v.CreatedAt,
		}
		if !inUse {
			orphanVolumes = append(orphanVolumes, dv)
		}
	}

	var orphanSize int64
	for _, ov := range orphanVolumes {
		orphanSize += ov.Size
	}

	// Top 10 volumes por size
	topVols := make([]DFVolume, 0, len(du.Volumes.Items))
	for _, v := range du.Volumes.Items {
		size := volumeSize(v)
		inUse := volumeInUse(v)
		topVols = append(topVols, DFVolume{
			Name:    v.Name,
			Size:    size,
			InUse:   inUse,
			Driver:  v.Driver,
			Created: v.CreatedAt,
		})
	}
	sort.Slice(topVols, func(i, j int) bool { return topVols[i].Size > topVols[j].Size })
	if len(topVols) > 10 {
		topVols = topVols[:10]
	}

	return &SystemDF{
		Images: DFImages{
			Count:       imgCount,
			TotalSize:   imgTotal,
			Reclaimable: imgReclaimable,
		},
		Containers: DFContainers{
			Count:     cntCount,
			TotalSize: cntTotal,
		},
		Volumes: DFVolumes{
			Count:       volCount,
			TotalSize:   volTotal,
			Reclaimable: volReclaimable,
			OrphanCount: int32(len(orphanVolumes)),
			OrphanSize:  orphanSize,
		},
		TopVolumes:    topVols,
		OrphanVolumes: orphanVolumes,
	}, nil
}

// GetVolumes retorna a lista de volumes com size/reclaimable.
func (c *Client) GetVolumes(ctx context.Context) ([]VolumeInfo, error) {
	volResult, err := c.cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		c.log.Error("erro ao listar volumes", "err", err)
		return nil, err
	}
	out := make([]VolumeInfo, 0, len(volResult.Items))
	for _, v := range volResult.Items {
		size := int64(0)
		reclaimable := int64(0)
		inUse := false
		if v.UsageData != nil {
			size = v.UsageData.Size
			if v.UsageData.RefCount > 0 {
				inUse = true
			}
		}
		out = append(out, VolumeInfo{
			Name:        v.Name,
			Driver:      v.Driver,
			Mountpoint:  v.Mountpoint,
			Size:        size,
			Reclaimable: reclaimable,
			InUse:       inUse,
			Labels:      v.Labels,
			Options:     v.Options,
			Scope:       v.Scope,
			Created:     v.CreatedAt,
		})
	}
	return out, nil
}

// GetNetworks lista todas as networks.
func (c *Client) GetNetworks(ctx context.Context) ([]NetworkInfo, error) {
	netResult, err := c.cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		c.log.Error("erro ao listar networks", "err", err)
		return nil, err
	}
	out := make([]NetworkInfo, 0, len(netResult.Items))
	for _, n := range netResult.Items {
		out = append(out, parseNetworkSummary(n))
	}
	return out, nil
}

// GetNetworkDetail retorna detalhes de uma network pelo id.
func (c *Client) GetNetworkDetail(ctx context.Context, networkID string) (*NetworkInfo, error) {
	result, err := c.cli.NetworkInspect(ctx, networkID, client.NetworkInspectOptions{})
	if err != nil {
		c.log.Error("erro ao inspecionar network", "id", networkID, "err", err)
		return nil, err
	}
	ni := parseNetworkInspect(result.Network)
	return &ni, nil
}

// parseNetworkSummary converte um network.Summary em NetworkInfo.
// network.Summary embute network.Network (sem Containers).
func parseNetworkSummary(n network.Summary) NetworkInfo {
	var subnets, gateways []string
	for _, cfg := range n.IPAM.Config {
		if cfg.Subnet.IsValid() {
			subnets = append(subnets, cfg.Subnet.String())
		}
		if cfg.Gateway.IsValid() {
			gateways = append(gateways, cfg.Gateway.String())
		}
	}
	return NetworkInfo{
		ID:         n.ID,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Attachable: n.Attachable,
		Ingress:    n.Ingress,
		Subnets:    subnets,
		Gateways:   gateways,
		Created:    n.Created.String(),
		Labels:     n.Labels,
		Options:    n.Options,
	}
}

// parseNetworkInspect converte um network.Inspect em NetworkInfo.
// network.Inspect embute network.Network + Containers map[string]EndpointResource.
func parseNetworkInspect(n network.Inspect) NetworkInfo {
	ni := parseNetworkSummary(network.Summary{Network: n.Network})
	var containers []NetworkContainer
	for cid, cdata := range n.Containers {
		containers = append(containers, NetworkContainer{
			ID:   cid,
			Name: cdata.Name,
			IPv4: cdata.IPv4Address.String(),
			IPv6: cdata.IPv6Address.String(),
		})
	}
	ni.Containers = containers
	return ni
}

// volumeSize extrai o size de um volume a partir de UsageData (se disponível).
// volume.Volume não tem campo Size direto — o size vem em UsageData.Size.
func volumeSize(v volume.Volume) int64 {
	if v.UsageData == nil {
		return 0
	}
	size := v.UsageData.Size
	if size < 0 {
		return 0
	}
	return size
}

// volumeInUse determina se um volume está em uso baseado em UsageData.RefCount.
func volumeInUse(v volume.Volume) bool {
	if v.UsageData == nil {
		return true // conservador: assume in use se não temos info
	}
	return v.UsageData.RefCount > 0
}
