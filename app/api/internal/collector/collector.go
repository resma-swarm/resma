// Package collector implementa a coleta periódica de métricas do Docker Swarm.
//
// Portado de backend/services/collector.py. Usa goroutines em vez de
// asyncio tasks. 7 loops:
//   - collectLoop: coleta stats de containers a cada CollectInterval
//   - containerEventLoop: escuta eventos Docker (start/stop/die/destroy)
//   - oomListener: detecta OOM (exit code 137) e registra no DB
//   - retentionLoop: roda retention job diariamente
//   - nodeCollectLoop: coleta info de nós a cada CollectInterval
//   - clusterCollectLoop: coleta info do cluster a cada ClusterInterval
//   - storageCollectLoop: coleta system df a cada StorageInterval
package collector

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/resma-swarm/resma/app/api/internal/config"
	"github.com/resma-swarm/resma/app/api/internal/db"
	"github.com/resma-swarm/resma/app/api/internal/docker"
)

// SSEPublisher é a interface que o collector usa para notificar o frontend
// via Server-Sent Events. É satisfeita por *sse.Handler (Publish(topic, eventType, payload)).
// Definida aqui para não acoplar o collector ao pacote sse diretamente e
// permitir testes com um publisher fake.
type SSEPublisher interface {
	Publish(topic, eventType string, payload any)
	SubscriberCount(topic string) int
	SubscribedTopicsByPrefix(prefix string) []string
}

// DataBuilder constrói payloads completos para publicação via SSE.
// É satisfeita por *server.Server (métodos build*). Permite que o collector
// publique os mesmos dados dos endpoints GET, eliminando a necessidade de
// refetch no frontend quando SSE está ativo.
type DataBuilder interface {
	BuildDashboardData(ctx context.Context) (map[string]any, error)
	BuildServicesList(ctx context.Context) ([]map[string]any, error)
	BuildServiceSparklines(ctx context.Context, points int) (map[string][]map[string]any, error)
	BuildNodesList(ctx context.Context) ([]map[string]any, error)
	BuildClusterInfo(ctx context.Context) (map[string]any, error)
	BuildStorageSummary(ctx context.Context) (map[string]any, error)
	BuildTasksList(ctx context.Context) ([]map[string]any, error)
	BuildServicesHealth(ctx context.Context, days int) (any, error)
	BuildAgentsList(ctx context.Context) ([]map[string]any, error)
	BuildRecommendations(ctx context.Context) (any, error)
	BuildServiceDetailData(ctx context.Context, service string) (map[string]any, error)
	BuildContainerDetailData(ctx context.Context, containerID string) (map[string]any, error)
}

// nopDataBuilder é um builder que retorna dados vazios (usado quando builder é nil).
type nopDataBuilder struct{}

func (nopDataBuilder) BuildDashboardData(context.Context) (map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildServicesList(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildServiceSparklines(context.Context, int) (map[string][]map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildNodesList(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildClusterInfo(context.Context) (map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildStorageSummary(context.Context) (map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildTasksList(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildServicesHealth(context.Context, int) (any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildAgentsList(context.Context) ([]map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildRecommendations(context.Context) (any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildServiceDetailData(context.Context, string) (map[string]any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildContainerDetailData(context.Context, string) (map[string]any, error) {
	return nil, nil
}

// nopSSEPublisher é um publisher que descarta eventos (usado quando SSE é nil).
type nopSSEPublisher struct{}

func (nopSSEPublisher) Publish(string, string, any)              {}
func (nopSSEPublisher) SubscriberCount(string) int               { return 0 }
func (nopSSEPublisher) SubscribedTopicsByPrefix(string) []string { return nil }

// Collector coordena 9 goroutines de coleta.
type Collector struct {
	cfg     *config.Config
	db      *db.Store
	docker  *docker.Client
	sse     SSEPublisher
	builder DataBuilder
	log     *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// taskByContainer — cache read-only de container_id -> TaskInfo,
	// populado pelo taskCollectLoop e consumido pelo collectOnce.
	// Protegido por taskMu (RWMutex). Pode estar vazio no startup.
	taskMu          sync.RWMutex
	taskByContainer map[string]docker.TaskInfo
}

// New cria um novo Collector. sse e builder podem ser nil (usa nop*).
func New(cfg *config.Config, database *db.Store, dc *docker.Client, sse SSEPublisher, builder DataBuilder) *Collector {
	if sse == nil {
		sse = nopSSEPublisher{}
	}
	if builder == nil {
		builder = nopDataBuilder{}
	}
	return &Collector{
		cfg:     cfg,
		db:      database,
		docker:  dc,
		sse:     sse,
		builder: builder,
		log:     slog.Default().With("component", "collector"),
	}
}

// publish notifica o frontend via SSE. Falhas de publish são silenciosas
// (o broker é non-blocking e nunca retorna erro).
func (c *Collector) publish(topic, eventType string, payload any) {
	c.sse.Publish(topic, eventType, payload)
}

// publishPayload constrói o payload completo via DataBuilder e publica via SSE.
// Se o builder retornar erro ou nil, faz fallback para publish com signal simples.
// Isso elimina a necessidade de refetch no frontend: o SSE carrega os dados
// completos (mesmo payload do GET), e o frontend usa setQueryData.
func (c *Collector) publishPayload(topic, eventType string, ctx context.Context, build func() (any, error)) {
	payload, err := build()
	if err != nil || payload == nil {
		// Fallback: publica signal simples (comportamento anterior)
		c.sse.Publish(topic, eventType, map[string]any{"signal": true})
		return
	}
	c.sse.Publish(topic, eventType, payload)
}

// Start inicia as 9 goroutines de coleta.
func (c *Collector) Start(ctx context.Context) {
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.log.Info("collector iniciado",
		"interval", c.cfg.CollectInterval,
		"cluster_interval", c.cfg.ClusterInterval,
		"storage_interval", c.cfg.StorageInterval,
		"task_poll_interval", c.cfg.AgentTaskPollInterval,
	)

	c.wg.Add(10)
	go c.collectLoop()
	go c.containerEventLoop()
	go c.oomListener()
	go c.retentionLoop()
	go c.nodeCollectLoop()
	go c.clusterCollectLoop()
	go c.storageCollectLoop()
	go c.taskCollectLoop()
	go c.agentHealthLoop()
	go c.staleLoop() // Fase 8
}

// Stop para todas as goroutines graciosamente.
func (c *Collector) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	c.log.Info("collector parado")
}

// shouldExclude verifica se o container deve ser excluído da coleta.
func (c *Collector) shouldExclude(image string) bool {
	if image == "" {
		return false
	}
	for _, excluded := range c.cfg.ExcludedImages {
		if strings.Contains(image, excluded) {
			return true
		}
	}
	return false
}

// collectLoop coleta stats de containers a cada CollectInterval.
func (c *Collector) collectLoop() {
	defer c.wg.Done()
	c.ensureStreams()

	ticker := time.NewTicker(c.cfg.CollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectOnce()
		}
	}
}

// ensureStreams inicia stats streams para todos os containers running.
func (c *Collector) ensureStreams() {
	containers := c.docker.ListContainers()
	for _, cnt := range containers {
		if cnt.State != "running" {
			continue
		}
		if c.shouldExclude(cnt.Image) {
			continue
		}
		c.docker.StartStatsStream(cnt.ID)
	}
}

// collectOnce faz uma rodada de coleta de métricas.
// Fase 7 — modo híbrido: anexa node_id/task_id/slot às métricas usando
// o snapshot de tasks do Swarm (taskByContainer) populado pelo taskCollectLoop.
func (c *Collector) collectOnce() {
	containers := c.docker.ListContainers()
	rows := make([]db.MetricRow, 0, len(containers))
	taskByContainer := c.taskSnapshot()
	for _, cnt := range containers {
		if cnt.State != "running" {
			continue
		}
		if c.shouldExclude(cnt.Image) {
			continue
		}
		stats := c.docker.GetCachedStats(cnt.ID)
		if stats == nil {
			// Tentar leitura point-in-time
			s, err := c.docker.GetStats(c.ctx, cnt.ID)
			if err != nil || s == nil {
				continue
			}
			c.docker.StartStatsStream(cnt.ID)
			stats = s
		}
		row := c.parseStats(stats, cnt)
		if row == nil {
			continue
		}
		// Enriquecer com node_id/task_id/slot do snapshot de tasks
		if ti, ok := taskByContainer[cnt.ID]; ok {
			row.NodeID = ti.NodeID
			row.TaskID = ti.ID
			row.Slot = ti.Slot
		}
		rows = append(rows, *row)
	}
	if len(rows) > 0 {
		if err := c.db.InsertMetricsBatch(c.ctx, rows); err != nil {
			c.log.Error("erro ao inserir métricas", "err", err)
		}
	}
	// Notificar frontend via SSE (tópico metrics) — payload completo.
	// O frontend usa setQueryData(["dashboard"], payload) — zero refetch HTTP.
	c.publishPayload("metrics", "metrics", c.ctx, func() (any, error) {
		return c.builder.BuildDashboardData(c.ctx)
	})
	c.syncServiceRegistry()
	// Publicar ServiceDetail para serviços com subscriber SSE ativo.
	// O collector só constrói o payload se alguém estiver vendo a página
	// de detalhe do serviço — evita trabalho desnecessário.
	c.publishServiceDetails()
	// Publicar ContainerDetail para containers com subscriber SSE ativo.
	// Mesmo princípio: só constrói o payload se alguém estiver vendo a página
	// de detalhe do container — elimina 3 GETs HTTP do frontend.
	c.publishContainerDetails()
}

// publishServiceDetails publica o payload do ServiceDetail para cada serviço
// que tem subscriber SSE ativo no tópico "service-detail/{name}".
// Isso elimina 6 GETs HTTP do frontend (zero refetch com SSE).
func (c *Collector) publishServiceDetails() {
	if c.builder == nil {
		return
	}
	// Listar serviços ativos para verificar subscribers
	services, _ := c.db.GetServiceRegistry(c.ctx)
	for svcName := range services {
		topic := "service-detail/" + svcName
		if c.sse.SubscriberCount(topic) == 0 {
			continue
		}
		// Há subscriber — construir e publicar payload
		c.publishPayload(topic, "service-detail", c.ctx, func() (any, error) {
			return c.builder.BuildServiceDetailData(c.ctx, svcName)
		})
	}
}

// publishContainerDetails publica o payload do ContainerDetail para cada
// container que tem subscriber SSE ativo no tópico "container-detail/{id}".
// Usa SubscribedTopicsByPrefix para descobrir apenas os containers que alguém
// está vendo — sem iterar sobre todos os containers conhecidos.
// Isso elimina 3 GETs HTTP do frontend (zero refetch com SSE).
func (c *Collector) publishContainerDetails() {
	if c.builder == nil {
		return
	}
	const prefix = "container-detail/"
	topics := c.sse.SubscribedTopicsByPrefix(prefix)
	for _, topic := range topics {
		cid := topic[len(prefix):]
		if cid == "" {
			continue
		}
		// Há subscriber — construir e publicar payload
		c.publishPayload(topic, "container-detail", c.ctx, func() (any, error) {
			return c.builder.BuildContainerDetailData(c.ctx, cid)
		})
	}
}

// parseStats converte docker.Stats + ContainerInfo em db.MetricRow.
func (c *Collector) parseStats(stats *docker.Stats, cnt docker.ContainerInfo) *db.MetricRow {
	service := cnt.Service
	if service == "" {
		service = cnt.Name
	}
	service = strings.TrimPrefix(service, "/")

	return &db.MetricRow{
		TS:                  stats.Read,
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
	}
}

// syncServiceRegistry sincroniza o registry de serviços com o Docker.
func (c *Collector) syncServiceRegistry() {
	statusMap, err := c.docker.GetServiceStatusMap(c.ctx)
	if err != nil {
		c.log.Error("erro ao obter status map", "err", err)
		return
	}
	registry, _ := c.db.GetServiceRegistry(c.ctx)

	query := "SELECT service, MAX(ts) as last_seen FROM metrics WHERE ts > now()::TIMESTAMP - INTERVAL " +
		itoa(c.cfg.RetentionDays) + " DAYS GROUP BY service"
	rows, err := c.db.QueryContext(c.ctx, query)
	if err != nil {
		c.log.Error("erro ao query services", "err", err)
		return
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var svcName string
		var lastSeen time.Time
		if err := rows.Scan(&svcName, &lastSeen); err != nil {
			continue
		}
		if reg, ok := registry[svcName]; ok && reg.Status == "archived" {
			continue
		}
		status := "legado"
		if s, ok := statusMap[svcName]; ok {
			status = s
		}
		_ = c.db.UpsertServiceRegistry(c.ctx, svcName, status, lastSeen)
	}
	// Notificar frontend via SSE (tópico services) — payload completo.
	c.publishPayload("services", "services", c.ctx, func() (any, error) {
		return c.builder.BuildServicesList(c.ctx)
	})
}

// containerEventLoop escuta eventos Docker para adicionar/remover containers.
func (c *Collector) containerEventLoop() {
	defer c.wg.Done()
	c.log.Info("container event loop iniciado")
	for {
		if c.ctx.Err() != nil {
			return
		}
		events, err := c.docker.ListenEvents(c.ctx)
		if err != nil {
			c.log.Error("erro ao escutar eventos", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for event := range events {
			if c.ctx.Err() != nil {
				return
			}
			switch event.Type {
			case docker.EventContainerStart:
				_ = c.docker.AddContainerToCache(c.ctx, event.ContainerID)
				c.docker.StartStatsStream(event.ContainerID)
			case docker.EventContainerStop, docker.EventContainerDie, docker.EventContainerDestroy:
				c.docker.RemoveContainerFromCache(event.ContainerID)
			}
		}
	}
}

// oomListener detecta OOM (exit code 137) e registra no DB.
func (c *Collector) oomListener() {
	defer c.wg.Done()
	c.log.Info("OOM listener iniciado")
	for {
		if c.ctx.Err() != nil {
			return
		}
		events, err := c.docker.ListenEvents(c.ctx)
		if err != nil {
			c.log.Error("OOM listener erro", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for event := range events {
			if c.ctx.Err() != nil {
				return
			}
			if event.Type != docker.EventContainerDie {
				continue
			}
			if event.ExitCode != "137" {
				continue
			}
			service := strings.TrimPrefix(event.Service, "/")
			_ = c.db.InsertOOMEvent(c.ctx, time.Now().UTC(), service, event.ContainerID, 137)
			cid := event.ContainerID
			if len(cid) > 12 {
				cid = cid[:12]
			}
			c.log.Warn("OOM detectado", "service", service, "container", cid)
			// Notificar frontend via SSE (tópico events) — OOM detectado.
			// OOM é um evento pontual, não payload de lista — mantém signal.
			c.publish("events", "oom", map[string]any{"service": service, "container": cid})
		}
	}
}

// retentionLoop roda retention job diariamente.
// Fase 8: também limpa refresh tokens expirados.
func (c *Collector) retentionLoop() {
	defer c.wg.Done()
	// Roda imediatamente ao iniciar, depois a cada 24h
	c.runRetentionOnce()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.runRetentionOnce()
		}
	}
}

// runRetentionOnce executa retention + cleanup de tokens uma vez.
func (c *Collector) runRetentionOnce() {
	if err := c.db.RunRetention(c.ctx, c.cfg.RetentionDays); err != nil {
		c.log.Error("retention erro", "err", err)
	} else {
		c.log.Info("retention job completed")
	}
	if n, err := c.db.CleanupExpiredRefreshTokens(c.ctx); err != nil {
		c.log.Error("cleanup expired tokens erro", "err", err)
	} else if n > 0 {
		c.log.Info("cleanup expired tokens", "removed", n)
	}
}

// staleLoop roda stale-marking a cada 1h (Fase 8).
// Marca services e nodes sem heartbeat por mais de StaleServiceDays como 'stale'.
func (c *Collector) staleLoop() {
	defer c.wg.Done()
	// Roda imediatamente ao iniciar, depois a cada 1h
	c.runStaleOnce()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.runStaleOnce()
		}
	}
}

// runStaleOnce executa stale-marking uma vez.
func (c *Collector) runStaleOnce() {
	days := c.cfg.StaleServiceDays
	if days <= 0 {
		days = 7
	}
	if n, err := c.db.MarkStaleServices(c.ctx, days); err != nil {
		c.log.Error("mark stale services erro", "err", err)
	} else if n > 0 {
		c.log.Info("marked stale services", "count", n)
	}
	if n, err := c.db.MarkStaleNodes(c.ctx, days); err != nil {
		c.log.Error("mark stale nodes erro", "err", err)
	} else if n > 0 {
		c.log.Info("marked stale nodes", "count", n)
	}
	if n, err := c.db.PruneContainerMap(c.ctx, days); err != nil {
		c.log.Error("prune container map erro", "err", err)
	} else if n > 0 {
		c.log.Info("pruned container map", "count", n)
	}
}

// nodeCollectLoop coleta info de nós a cada CollectInterval.
func (c *Collector) nodeCollectLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.cfg.CollectInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectNodes()
		}
	}
}

// collectNodes faz uma rodada de coleta de nós.
func (c *Collector) collectNodes() {
	ctx := c.ctx
	nodes, err := c.docker.GetNodes(ctx)
	if err != nil {
		c.log.Error("erro ao listar nós", "err", err)
		return
	}
	tasksByNode, _ := c.docker.GetTasksByNode(ctx)
	ts := time.Now().UTC()

	metricRows := make([]db.NodeMetricRow, 0, len(nodes))
	for _, n := range nodes {
		tasks := tasksByNode[n.ID]
		runningCount := int32(0)
		for _, t := range tasks {
			if t.State == "running" {
				runningCount++
			}
		}
		labelsJSON, _ := json.Marshal(n.Labels)
		_ = c.db.UpsertNode(ctx, db.Node{
			NodeID:        n.ID,
			Hostname:      n.Hostname,
			Role:          n.Role,
			Availability:  n.Availability,
			Status:        n.Status,
			Address:       n.Address,
			CPUTotal:      n.CPUTotal,
			MemTotal:      n.MemTotal,
			OS:            n.OS,
			Architecture:  n.Architecture,
			EngineVersion: n.EngineVersion,
			IsLeader:      n.IsLeader,
			Reachability:  n.Reachability,
			Labels:        string(labelsJSON),
			TasksRunning:  runningCount,
		})
		metricRows = append(metricRows, db.NodeMetricRow{
			TS:           ts,
			NodeID:       n.ID,
			Hostname:     n.Hostname,
			Role:         n.Role,
			Availability: n.Availability,
			Status:       n.Status,
			CPUTotal:     n.CPUTotal,
			MemTotal:     n.MemTotal,
			TasksRunning: runningCount,
		})
	}
	if len(metricRows) > 0 {
		_ = c.db.InsertNodeMetricsBatch(ctx, metricRows)
	}

	mappings, _ := c.docker.GetContainerNodeMapping(ctx)
	if len(mappings) > 0 {
		mapRows := make([]db.ContainerNodeMapRow, 0, len(mappings))
		for _, m := range mappings {
			mapRows = append(mapRows, db.ContainerNodeMapRow{
				ContainerID: m.ContainerID,
				NodeID:      m.NodeID,
				Service:     m.Service,
			})
		}
		_ = c.db.UpsertContainerNodeMapBatch(ctx, mapRows)
	}
	// Notificar frontend via SSE (tópico nodes) — payload completo.
	c.publishPayload("nodes", "nodes", c.ctx, func() (any, error) {
		return c.builder.BuildNodesList(c.ctx)
	})
}

// clusterCollectLoop coleta info do cluster a cada ClusterInterval.
func (c *Collector) clusterCollectLoop() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.cfg.ClusterInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectCluster()
		}
	}
}

// collectCluster faz uma rodada de coleta do cluster.
func (c *Collector) collectCluster() {
	ctx := c.ctx
	swarm, err := c.docker.GetSwarmInfo(ctx)
	if err != nil {
		c.log.Error("erro ao obter swarm info", "err", err)
		return
	}
	nodes, _ := c.docker.GetNodes(ctx)
	nodesReady := int32(0)
	nodesDown := int32(0)
	managers := int32(0)
	workers := int32(0)
	for _, n := range nodes {
		if n.Status == "ready" {
			nodesReady++
		} else {
			nodesDown++
		}
		if n.Role == "manager" {
			managers++
		} else {
			workers++
		}
	}
	quorumHealthy := managers >= 1 && swarm.ControlAvailable
	warningsJSON, _ := json.Marshal(swarm.Warnings)
	clusterID := swarm.ClusterID
	if clusterID == "" {
		clusterID = "local"
	}
	_ = c.db.UpsertCluster(ctx, db.Cluster{
		ID:            clusterID,
		NodesTotal:    swarm.NodesTotal,
		ManagersTotal: managers,
		WorkersTotal:  workers,
		NodesReady:    nodesReady,
		NodesDown:     nodesDown,
		QuorumHealthy: quorumHealthy,
		SelfNodeID:    swarm.NodeID,
		Warnings:      string(warningsJSON),
	})
	// Notificar frontend via SSE (tópico dashboard) — payload completo do dashboard.
	c.publishPayload("dashboard", "cluster", c.ctx, func() (any, error) {
		return c.builder.BuildDashboardData(c.ctx)
	})
}

// storageCollectLoop coleta system df a cada StorageInterval.
func (c *Collector) storageCollectLoop() {
	defer c.wg.Done()
	c.log.Info("storage collect loop iniciado", "interval", c.cfg.StorageInterval)
	ticker := time.NewTicker(c.cfg.StorageInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectStorage()
		}
	}
}

// collectStorage faz uma rodada de coleta de storage.
func (c *Collector) collectStorage() {
	ctx := c.ctx
	df, err := c.docker.GetSystemDF(ctx)
	if err != nil {
		c.log.Error("erro ao obter system df", "err", err)
		return
	}
	ts := time.Now().UTC()
	_ = c.db.InsertStorageSummary(ctx, ts,
		df.Images.Count, df.Images.TotalSize, df.Images.Reclaimable,
		df.Containers.Count, df.Containers.TotalSize,
		df.Volumes.Count, df.Volumes.TotalSize, df.Volumes.Reclaimable,
		df.Volumes.OrphanCount, df.Volumes.OrphanSize,
	)

	volumes, err := c.docker.GetVolumes(ctx)
	if err != nil {
		c.log.Error("erro ao listar volumes", "err", err)
		return
	}
	volRows := make([]db.VolumeMetricRow, 0, len(volumes))
	for _, v := range volumes {
		volRows = append(volRows, db.VolumeMetricRow{
			TS:               ts,
			VolumeName:       v.Name,
			SizeBytes:        v.Size,
			ReclaimableBytes: v.Reclaimable,
			InUse:            v.InUse,
		})
	}
	if len(volRows) > 0 {
		_ = c.db.InsertVolumeMetricsBatch(ctx, volRows)
	}
	// Notificar frontend via SSE (tópico dashboard) — payload completo de storage.
	c.publishPayload("dashboard", "storage", c.ctx, func() (any, error) {
		return c.builder.BuildStorageSummary(c.ctx)
	})
}

// itoa converte int para string (sem import strconv para manter imports mínimos).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// ---------------------------------------------------------------------------
// Fase 7 — task lifecycle + agent health
// ---------------------------------------------------------------------------

// taskSnapshot retorna uma cópia read-only do cache container_id -> TaskInfo.
// Pode ser chamado concorrentemente pelo collectOnce.
func (c *Collector) taskSnapshot() map[string]docker.TaskInfo {
	c.taskMu.RLock()
	defer c.taskMu.RUnlock()
	if c.taskByContainer == nil {
		return nil
	}
	out := make(map[string]docker.TaskInfo, len(c.taskByContainer))
	for k, v := range c.taskByContainer {
		out[k] = v
	}
	return out
}

// taskCollectLoop faz poll de TaskList a cada AgentTaskPollInterval e:
//   - atualiza a tabela tasks (snapshot do Swarm)
//   - registra mudanças de status em task_history (append-only)
//   - atualiza o cache taskByContainer para o collectOnce enriquecer métricas
//   - faz prune de tasks que sumiram do Swarm
func (c *Collector) taskCollectLoop() {
	defer c.wg.Done()
	c.log.Info("task collect loop iniciado", "interval", c.cfg.AgentTaskPollInterval)

	// Rodar imediatamente ao iniciar para popular o cache rápido
	c.collectTasks()

	ticker := time.NewTicker(c.cfg.AgentTaskPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectTasks()
		}
	}
}

// collectTasks faz uma rodada de coleta de tasks do Swarm.
func (c *Collector) collectTasks() {
	ctx := c.ctx
	tasks, err := c.docker.GetTasks(ctx)
	if err != nil {
		c.log.Error("erro ao listar tasks para lifecycle", "err", err)
		return
	}

	// Snapshot atual de task_ids para prune
	seen := make(map[string]bool, len(tasks))
	newCache := make(map[string]docker.TaskInfo, len(tasks))
	anyChanged := false

	for _, t := range tasks {
		seen[t.ID] = true
		if t.ContainerID != "" {
			newCache[t.ContainerID] = t
		}

		dbTask := db.Task{
			TaskID:       t.ID,
			Service:      t.ServiceName,
			NodeID:       t.NodeID,
			Slot:         t.Slot,
			Status:       t.State,
			DesiredState: t.DesiredState,
		}
		changed, err := c.db.UpsertTask(ctx, dbTask)
		if err != nil {
			c.log.Error("erro ao upsert task", "task_id", t.ID, "err", err)
			continue
		}
		if changed {
			anyChanged = true
			if err := c.db.InsertTaskHistory(ctx, dbTask); err != nil {
				c.log.Error("erro ao inserir task_history", "task_id", t.ID, "err", err)
			}
			c.log.Info("task status mudou",
				"task_id", t.ID, "service", t.ServiceName,
				"slot", t.Slot, "status", t.State, "node_id", t.NodeID)
		}
	}

	// Prune: remover tasks que sumiram do Swarm
	pruned := c.pruneTasks(ctx, seen)
	if pruned > 0 {
		anyChanged = true
	}

	// Atualizar cache atomicamente
	c.taskMu.Lock()
	c.taskByContainer = newCache
	c.taskMu.Unlock()

	// Notificar frontend via SSE (tópico tasks) apenas quando houve mudança
	// de status ou prune — evita invalidação desnecessária a cada poll.
	if anyChanged {
		// Notificar frontend via SSE (tópico tasks) — payload completo.
		c.publishPayload("tasks", "tasks", c.ctx, func() (any, error) {
			return c.builder.BuildTasksList(c.ctx)
		})
	}
}

// pruneTasks remove da tabela tasks os IDs que não estão no snapshot atual.
// Retorna o número de tasks removidas.
func (c *Collector) pruneTasks(ctx context.Context, seen map[string]bool) int {
	existing, err := c.db.GetTasks(ctx)
	if err != nil {
		return 0
	}
	removed := 0
	for _, t := range existing {
		if !seen[t.TaskID] {
			if err := c.db.DeleteTask(ctx, t.TaskID); err != nil {
				c.log.Error("erro ao deletar task obsoleta", "task_id", t.TaskID, "err", err)
			} else {
				removed++
			}
		}
	}
	return removed
}

// agentHealthLoop marca agents como "stale" se o heartbeat não chegar dentro
// do threshold (3x o interval esperado). O agent envia heartbeat a cada 60s,
// então o threshold default é 180s (3 minutos).
func (c *Collector) agentHealthLoop() {
	defer c.wg.Done()
	c.log.Info("agent health loop iniciado")

	// Threshold: 3x o task poll interval (mínimo 60s)
	threshold := c.cfg.AgentTaskPollInterval * 3
	if threshold < 60*time.Second {
		threshold = 60 * time.Second
	}

	ticker := time.NewTicker(c.cfg.AgentTaskPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			n, err := c.db.MarkAgentsStale(c.ctx, threshold)
			if err != nil {
				c.log.Error("erro ao marcar agents stale", "err", err)
				continue
			}
			if n > 0 {
				c.log.Warn("agents marcados como stale", "count", n, "threshold", threshold)
				// Notificar frontend via SSE (tópico agents) — payload completo.
				c.publishPayload("agents", "stale", c.ctx, func() (any, error) {
					return c.builder.BuildAgentsList(c.ctx)
				})
			}
		}
	}
}
