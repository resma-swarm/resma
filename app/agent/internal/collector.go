// Package internal — coleta de stats de containers locais via Docker socket.
//
// Mantém o mesmo padrão do server (app/api/internal/docker/stats.go):
//   - Uma goroutine por container mantém um stream persistente de stats
//   - O cache em memória é lido a cada CollectInterval para montar o batch
//   - Reconecta com backoff em caso de erro
//
// Diferenças vs server:
//   - Sem DuckDB — apenas cache em memória + Buffer para push
//   - Sem TaskList (Swarm metadata é coletada pelo manager)
//   - node_id/hostname preenchidos via Docker Info no startup (fallback
//     quando Swarm template vars não estão disponíveis)
package internal

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

// Stats é a versão parseada de Docker stats (mesma estrutura do server).
type Stats struct {
	Read                time.Time
	CPUPercent          float64
	CPUUsage            int64
	CPUSystem           int64
	OnlineCPUs          int64
	CPUThrottledPeriods int64
	CPUThrottledTime    int64
	MemUsage            int64
	MemLimit            int64
	MemPercent          float64
	MemWorkingSet       int64
	NetRX               int64
	NetTX               int64
	BlockRead           int64
	BlockWrite          int64
}

// ContainerInfo é a versão parseada de um container (mínimo para o agent).
type ContainerInfo struct {
	ID      string
	Name    string
	Service string
	State   string
	Image   string
}

// Collector coordena a coleta de stats de containers locais.
type Collector struct {
	cli          *client.Client
	log          *slog.Logger
	cfg          collectorConfig
	mu           sync.RWMutex
	cache        map[string]ContainerInfo
	statsCache   map[string]Stats
	streamCancel map[string]context.CancelFunc
}

type collectorConfig interface {
	ExcludedImages() []string
	Debug() bool
}

// NewCollector cria um novo Collector.
func NewCollector(cli *client.Client, cfg collectorConfig) *Collector {
	return &Collector{
		cli:          cli,
		log:          slog.Default().With("component", "agent-collector"),
		cfg:          cfg,
		cache:        make(map[string]ContainerInfo),
		statsCache:   make(map[string]Stats),
		streamCancel: make(map[string]context.CancelFunc),
	}
}

// InitContainers popula o cache inicial e inicia streams para containers running.
func (c *Collector) InitContainers(ctx context.Context) error {
	result, err := c.cli.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return err
	}
	c.mu.Lock()
	for _, cnt := range result.Items {
		ci := parseContainer(cnt)
		c.cache[cnt.ID] = ci
	}
	count := len(c.cache)
	c.mu.Unlock()
	c.log.Info("container cache inicializado", "count", count)

	for _, cnt := range result.Items {
		if string(cnt.State) != "running" {
			continue
		}
		if c.shouldExclude(cnt.Image) {
			continue
		}
		c.StartStatsStream(cnt.ID)
	}
	return nil
}

// ListContainers retorna todos os containers do cache.
func (c *Collector) ListContainers() []ContainerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ContainerInfo, 0, len(c.cache))
	for _, ci := range c.cache {
		out = append(out, ci)
	}
	return out
}

// AddContainer faz inspect e adiciona ao cache (evento start).
func (c *Collector) AddContainer(ctx context.Context, id string) {
	resp, err := c.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		c.log.Warn("erro ao inspecionar container", "id", shortID(id), "err", err)
		return
	}
	ci := parseInspect(resp.Container)
	c.mu.Lock()
	c.cache[id] = ci
	c.mu.Unlock()
	c.log.Info("container adicionado", "name", ci.Name, "id", shortID(id))
}

// RemoveContainer remove do cache e para o stream (evento stop/die/destroy).
func (c *Collector) RemoveContainer(id string) {
	c.mu.Lock()
	delete(c.cache, id)
	delete(c.statsCache, id)
	if cancel, ok := c.streamCancel[id]; ok {
		cancel()
		delete(c.streamCancel, id)
	}
	c.mu.Unlock()
}

// StartStatsStream inicia uma goroutine de streaming de stats para um container.
// Idempotente: se já existe stream ativo, não faz nada.
func (c *Collector) StartStatsStream(id string) {
	c.mu.Lock()
	if _, exists := c.streamCancel[id]; exists {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.streamCancel[id] = cancel
	c.mu.Unlock()

	go c.statsStreamLoop(ctx, id)
}

func (c *Collector) statsStreamLoop(ctx context.Context, id string) {
	for {
		if ctx.Err() != nil {
			return
		}
		err := c.streamStatsOnce(ctx, id)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			c.log.Warn("stats stream erro, reconectando em 3s", "id", shortID(id), "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func (c *Collector) streamStatsOnce(ctx context.Context, id string) error {
	result, err := c.cli.ContainerStats(ctx, id, client.ContainerStatsOptions{Stream: true})
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
		c.statsCache[id] = parsed
		c.mu.Unlock()
	}
}

// GetCachedStats retorna as stats mais recentes do cache, ou nil.
func (c *Collector) GetCachedStats(id string) *Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.statsCache[id]
	if !ok {
		return nil
	}
	return &s
}

// CollectOnce faz uma rodada de coleta e devolve os MetricPoints prontos para push.
func (c *Collector) CollectOnce() []MetricPoint {
	containers := c.ListContainers()
	out := make([]MetricPoint, 0, len(containers))
	for _, cnt := range containers {
		if cnt.State != "running" {
			continue
		}
		if c.shouldExclude(cnt.Image) {
			continue
		}
		stats := c.GetCachedStats(cnt.ID)
		if stats == nil {
			continue
		}
		service := cnt.Service
		if service == "" {
			service = strings.TrimPrefix(cnt.Name, "/")
		}
		out = append(out, MetricPoint{
			TS:                  stats.Read.UTC().Format(time.RFC3339Nano),
			Service:             service,
			ContainerID:         cnt.ID,
			CPUPercent:          stats.CPUPercent,
			CPUUsage:            stats.CPUUsage,
			CPUSystem:           stats.CPUSystem,
			MemUsage:            stats.MemUsage,
			MemLimit:            stats.MemLimit,
			MemPercent:          stats.MemPercent,
			NetRX:               stats.NetRX,
			NetTX:               stats.NetTX,
			BlockRead:           stats.BlockRead,
			BlockWrite:          stats.BlockWrite,
			MemWorkingSet:       stats.MemWorkingSet,
			CPUThrottledPeriods: stats.CPUThrottledPeriods,
			CPUThrottledTime:    stats.CPUThrottledTime,
		})
	}
	return out
}

// shouldExclude verifica se o container deve ser excluído da coleta.
func (c *Collector) shouldExclude(image string) bool {
	if image == "" {
		return false
	}
	for _, excluded := range c.cfg.ExcludedImages() {
		if strings.Contains(image, excluded) {
			return true
		}
	}
	return false
}

// parseContainer converte container.Summary em ContainerInfo.
func parseContainer(cnt container.Summary) ContainerInfo {
	service := ""
	if cnt.Labels != nil {
		service = cnt.Labels["com.docker.swarm.service.name"]
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

// parseInspect converte container.InspectResponse em ContainerInfo.
func parseInspect(info container.InspectResponse) ContainerInfo {
	service := ""
	if info.Config != nil && info.Config.Labels != nil {
		service = info.Config.Labels["com.docker.swarm.service.name"]
	}
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
	return ContainerInfo{
		ID:      info.ID,
		Name:    name,
		Service: strings.TrimPrefix(service, "/"),
		State:   state,
		Image:   image,
	}
}

// parseStats converte container.StatsResponse em Stats (mesma lógica do server).
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

	memWorkingSet := memUsage
	if v, ok := memStats.Stats["inactive_file"]; ok && int64(v) > 0 {
		memWorkingSet = memUsage - int64(v)
	} else if v, ok := memStats.Stats["cache"]; ok && int64(v) > 0 {
		memWorkingSet = memUsage - int64(v)
	}
	if memWorkingSet < 0 {
		memWorkingSet = memUsage
	}

	throttledPeriods := int64(cpuStats.ThrottlingData.ThrottledPeriods)
	throttledTime := int64(cpuStats.ThrottlingData.ThrottledTime)

	var netRX, netTX int64
	for _, n := range raw.Networks {
		netRX += int64(n.RxBytes)
		netTX += int64(n.TxBytes)
	}

	var blockRead, blockWrite int64
	for _, entry := range raw.BlkioStats.IoServiceBytesRecursive {
		switch entry.Op {
		case "read", "Read":
			blockRead += int64(entry.Value)
		case "write", "Write":
			blockWrite += int64(entry.Value)
		}
	}

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

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// ContainerEvent é um evento de container parseado.
type ContainerEvent struct {
	Type        string // "start", "stop", "die", "destroy"
	ContainerID string
	ExitCode    string
	Service     string
}

// ListenEvents abre um stream de eventos Docker e envia ContainerEvents para o channel.
// Mesmo padrão do server (app/api/internal/docker/events.go).
func (c *Collector) ListenEvents(ctx context.Context) (<-chan ContainerEvent, error) {
	f := make(client.Filters).Add("type", "container")
	result := c.cli.Events(ctx, client.EventsListOptions{Filters: f})

	out := make(chan ContainerEvent, 64)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-result.Messages:
				if !ok {
					select {
					case err := <-result.Err:
						if err != nil && ctx.Err() == nil {
							c.log.Warn("docker events stream encerrado", "err", err)
						}
					default:
					}
					return
				}
				ce := parseEvent(msg)
				if ce.ContainerID != "" {
					select {
					case out <- ce:
					case <-ctx.Done():
						return
					}
				}
			case err := <-result.Err:
				if err != nil && ctx.Err() == nil {
					c.log.Warn("docker events erro", "err", err)
				}
				return
			}
		}
	}()
	return out, nil
}

// parseEvent converte um events.Message em ContainerEvent.
func parseEvent(msg events.Message) ContainerEvent {
	ce := ContainerEvent{
		Type:        string(msg.Action),
		ContainerID: msg.Actor.ID,
	}
	if msg.Actor.Attributes != nil {
		ce.ExitCode = msg.Actor.Attributes["exitCode"]
		ce.Service = msg.Actor.Attributes["com.docker.swarm.service.name"]
	}
	return ce
}
