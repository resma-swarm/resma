// Package docker — stats streaming e parsing.
//
// Mantém o padrão do Python: uma goroutine por container mantém um stream
// persistente de stats e atualiza o cache em memória. O collector lê do
// cache a cada ciclo (sem round-trip ao daemon).
//
// 0b.2: parsing de working set (usage - inactive_file/cache) e CPU
// throttling (throttled_periods, throttled_time).
package docker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// GetCachedStats retorna as stats mais recentes do cache, ou nil.
func (c *Client) GetCachedStats(containerID string) *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.statsCache[containerID]
	if !ok {
		return nil
	}
	return &s
}

// StartStatsStream inicia uma goroutine de streaming de stats para um container.
// Idempotente: se já existe stream ativo, não faz nada.
func (c *Client) StartStatsStream(containerID string) {
	c.mu.Lock()
	if _, exists := c.streamCancel[containerID]; exists {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.streamCancel[containerID] = cancel
	c.mu.Unlock()

	go c.statsStreamLoop(ctx, containerID)
}

// StopStatsStream para o stream de stats de um container.
func (c *Client) StopStatsStream(containerID string) {
	c.mu.Lock()
	cancel, ok := c.streamCancel[containerID]
	if ok {
		cancel()
		delete(c.streamCancel, containerID)
	}
	delete(c.statsCache, containerID)
	c.mu.Unlock()
}

// statsStreamLoop mantém um stream persistente de stats e atualiza o cache.
// Reconecta com backoff em caso de erro.
func (c *Client) statsStreamLoop(ctx context.Context, containerID string) {
	for {
		if ctx.Err() != nil {
			return
		}
		err := c.streamStatsOnce(ctx, containerID)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			c.log.Warn("stats stream erro, reconectando em 3s", "id", containerID[:12], "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}
}

// streamStatsOnce abre um stream de stats e lê até erro/cancel.
func (c *Client) streamStatsOnce(ctx context.Context, containerID string) error {
	result, err := c.cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
		Stream: true,
	})
	if err != nil {
		return err
	}
	defer result.Body.Close()

	dec := json.NewDecoder(result.Body)
	for {
		if ctx.Err() != nil {
			return nil
		}
		var raw container.StatsResponse
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		parsed := parseStats(raw)
		c.mu.Lock()
		c.statsCache[containerID] = parsed
		c.mu.Unlock()
	}
}

// GetStats faz uma leitura point-in-time (stream=false) das stats de um container.
func (c *Client) GetStats(ctx context.Context, containerID string) (*Stats, error) {
	result, err := c.cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
		Stream: false,
	})
	if err != nil {
		c.log.Error("erro ao obter stats", "id", containerID[:12], "err", err)
		return nil, err
	}
	defer result.Body.Close()

	var raw container.StatsResponse
	if err := json.NewDecoder(result.Body).Decode(&raw); err != nil {
		return nil, err
	}
	parsed := parseStats(raw)
	return &parsed, nil
}

// parseStats converte container.StatsResponse em Stats (com working set e throttling).
func parseStats(raw container.StatsResponse) Stats {
	cpuStats := raw.CPUStats
	cpuPrev := raw.PreCPUStats
	memStats := raw.MemoryStats

	cpuUsage := int64(cpuStats.CPUUsage.TotalUsage)
	cpuSystem := int64(cpuStats.SystemUsage)
	cpuPrevUsage := int64(cpuPrev.CPUUsage.TotalUsage)
	cpuPrevSystem := int64(cpuPrev.SystemUsage)
	onlineCPUs := int64(cpuStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = int64(len(cpuStats.CPUUsage.PercpuUsage))
	}
	if onlineCPUs == 0 {
		onlineCPUs = 1
	}

	cpuDelta := cpuUsage - cpuPrevUsage
	systemDelta := cpuSystem - cpuPrevSystem
	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = float64(cpuDelta) / float64(systemDelta) * float64(onlineCPUs) * 100.0
	}

	memUsage := int64(memStats.Usage)
	memLimit := int64(memStats.Limit)
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100.0
	}

	// 0b.2: working set = usage - inactive_file (cgroup v2) ou usage - cache (v1)
	memWorkingSet := memUsage
	if v, ok := memStats.Stats["inactive_file"]; ok && int64(v) > 0 {
		memWorkingSet = memUsage - int64(v)
	} else if v, ok := memStats.Stats["cache"]; ok && int64(v) > 0 {
		memWorkingSet = memUsage - int64(v)
	}
	if memWorkingSet < 0 {
		memWorkingSet = memUsage
	}

	// 0b.2: CPU throttling
	throttledPeriods := int64(cpuStats.ThrottlingData.ThrottledPeriods)
	throttledTime := int64(cpuStats.ThrottlingData.ThrottledTime)

	// Network
	var netRX, netTX int64
	for _, n := range raw.Networks {
		netRX += int64(n.RxBytes)
		netTX += int64(n.TxBytes)
	}

	// Block IO
	var blockRead, blockWrite int64
	for _, entry := range raw.BlkioStats.IoServiceBytesRecursive {
		switch entry.Op {
		case "read", "Read":
			blockRead += int64(entry.Value)
		case "write", "Write":
			blockWrite += int64(entry.Value)
		}
	}

	// Timestamp
	ts := raw.Read
	if ts.IsZero() {
		ts = time.Now()
	}

	return Stats{
		Read:                ts,
		CPUPercent:          round2(cpuPercent),
		CPUUsage:            cpuUsage,
		CPUSystem:           cpuSystem,
		OnlineCPUs:          onlineCPUs,
		CPUThrottledPeriods: throttledPeriods,
		CPUThrottledTime:    throttledTime,
		MemUsage:            memUsage,
		MemLimit:            memLimit,
		MemPercent:          round2(memPercent),
		MemWorkingSet:       memWorkingSet,
		NetRX:               netRX,
		NetTX:               netTX,
		BlockRead:           blockRead,
		BlockWrite:          blockWrite,
	}
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
