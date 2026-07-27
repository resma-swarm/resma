// Command resma-agent coleta métricas locais de containers e envia para o RESMA API.
//
// Deploy: global service no Docker Swarm (1 agent por node).
//
//	docker service create --mode global --name resma-agent \
//	  --env RESMA_API_URL=http://api:8080 \
//	  --env RESMA_AGENT_TOKEN=<shared-secret> \
//	  --env RESMA_NODE_ID={{.Node.ID}} \
//	  --env RESMA_NODE_HOSTNAME={{.Node.Hostname}} \
//	  -v /var/run/docker.sock:/var/run/docker.sock:ro \
//	  ghcr.io/resma/resma-agent:latest
//
// O agent NÃO acessa DuckDB. Ele lê do Docker socket local e faz HTTP POST
// para o manager. Resiliência: ring buffer para métricas, persistência em
// disco para OOM events, exponential backoff para retries.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/moby/moby/client"

	"github.com/resma/agent/internal"
)

// Version é sobrescrita via -ldflags no build do Dockerfile.
var Version = "dev"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := internal.Load()
	if err != nil {
		logger.Error("config load falhou", "err", err)
		os.Exit(1)
	}
	logger.Info("config carregada",
		"api_url", cfg.APIURL,
		"node_id", cfg.NodeID,
		"node_hostname", cfg.NodeHostname,
		"collect_interval", cfg.CollectInterval,
		"http_addr", cfg.HTTPAddr,
		"version", Version,
	)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Docker SDK (socket local)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error("docker client init falhou", "err", err)
		os.Exit(1)
	}
	defer cli.Close()

	// Preencher node_id/hostname via Docker Info se não vieram do Swarm template
	if cfg.NodeID == "" || cfg.NodeHostname == "" {
		info, err := cli.Info(rootCtx, client.InfoOptions{})
		if err != nil {
			logger.Warn("docker info falhou (continuando com node_id vazio)", "err", err)
		} else {
			if cfg.NodeID == "" {
				cfg.NodeID = info.Info.Swarm.NodeID
			}
			if cfg.NodeHostname == "" {
				cfg.NodeHostname = info.Info.Name
			}
			logger.Info("node identificado via docker info", "node_id", cfg.NodeID, "hostname", cfg.NodeHostname)
		}
	}
	if cfg.NodeID == "" {
		logger.Error("não foi possível determinar node_id — defina RESMA_NODE_ID via Swarm template")
		os.Exit(1)
	}

	// Componentes
	collector := internal.NewCollector(cli, &cfgAdapter{cfg: cfg})
	pusher := internal.NewPusher(&cfgAdapter{cfg: cfg})

	// Event store (OOM persistence) — dir /data no container, fallback /tmp
	eventsDir := getenv("RESMA_AGENT_DATA_DIR", "/data")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		logger.Warn("não foi possível criar data dir, usando /tmp", "dir", eventsDir, "err", err)
		eventsDir = "/tmp"
	}
	eventStore := internal.NewEventStore(eventsDir, pusher)
	if err := eventStore.LoadPending(); err != nil {
		logger.Warn("erro ao carregar OOM pendentes", "err", err)
	}

	// Buffer de métricas (ring buffer, cap 1000 por container ~ total ~10MB)
	metricsBuf := internal.NewBuffer(10000)

	// Init containers cache + start stats streams
	if err := collector.InitContainers(rootCtx); err != nil {
		logger.Error("init containers falhou", "err", err)
		os.Exit(1)
	}

	// Events listener (container start/stop/die/destroy + OOM detection)
	eventsCh, err := collector.ListenEvents(rootCtx)
	if err != nil {
		logger.Error("docker events listen falhou", "err", err)
		os.Exit(1)
	}
	go eventLoop(rootCtx, collector, eventStore, eventsCh, logger)

	// Collect loop — coleta stats a cada CollectInterval, adiciona ao buffer
	go collectLoop(rootCtx, collector, metricsBuf, cfg.CollectInterval, logger)

	// Push loop — drena o buffer e faz push a cada CollectInterval (offset 5s)
	go pushLoop(rootCtx, pusher, metricsBuf, cfg.CollectInterval, logger)

	// OOM flush loop — tenta enviar pendentes a cada 30s
	go oomFlushLoop(rootCtx, eventStore, 30*time.Second, logger)

	// Heartbeat loop — a cada 60s
	go heartbeatLoop(rootCtx, pusher, collector, cfg, 60*time.Second, Version, logger)

	// Health/Info HTTP server (porta local :8082)
	go infoServer(rootCtx, cfg.HTTPAddr, collector, eventStore, metricsBuf, Version, logger)

	<-rootCtx.Done()
	logger.Info("shutdown solicitado")
	// Best-effort flush final
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	eventStore.FlushAll(ctx)
	logger.Info("bye")
}

// eventLoop processa eventos de container do Docker.
func eventLoop(ctx context.Context, c *internal.Collector, es *internal.EventStore, ch <-chan internal.ContainerEvent, log *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			switch ev.Type {
			case "start":
				c.AddContainer(ctx, ev.ContainerID)
				c.StartStatsStream(ev.ContainerID)
			case "stop", "die", "destroy":
				c.RemoveContainer(ev.ContainerID)
			}
			// OOM detection: action "oom" ou "die" com exitCode 137
			if internal.IsOOM(ev.Type, ev.ExitCode) {
				oomEv := internal.NewOOMEvent(ev)
				if err := es.Add(oomEv); err != nil {
					log.Error("erro ao persistir OOM event", "service", oomEv.Service, "err", err)
				} else {
					log.Warn("OOM detectado e persistido", "service", oomEv.Service, "container", oomEv.ContainerID[:12])
				}
			}
		}
	}
}

// collectLoop coleta stats a cada interval e adiciona ao buffer.
func collectLoop(ctx context.Context, c *internal.Collector, buf *internal.Buffer, interval time.Duration, log *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// Primeira coleta imediata
	publish := func() {
		points := c.CollectOnce()
		for _, p := range points {
			buf.Add(p)
		}
		if len(points) > 0 {
			log.Debug("pontos coletados", "count", len(points), "buffer_len", buf.Len())
		}
	}
	publish()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publish()
		}
	}
}

// pushLoop drena o buffer e faz push a cada interval.
func pushLoop(ctx context.Context, p *internal.Pusher, buf *internal.Buffer, interval time.Duration, log *slog.Logger) {
	// Offset inicial de 5s para deixar o collect loop encher o buffer primeiro
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			points := buf.Drain()
			if len(points) == 0 {
				continue
			}
			if err := p.PushMetrics(ctx, points); err != nil {
				log.Error("push metrics falhou — pontos perdidos (buffer drenado)", "count", len(points), "err", err)
				// NOTA: não recolocamos no buffer para evitar duplicação.
				// O ring buffer já descarta os mais antigos quando cheio,
				// e o próximo ciclo de coleta adiciona novos pontos.
				// Para OOM (crítico) usamos persistência em disco.
				continue
			}
			buf.Ack(len(points))
			log.Info("metrics enviadas", "count", len(points))
		}
	}
}

// oomFlushLoop tenta enviar OOM pendentes periodicamente.
func oomFlushLoop(ctx context.Context, es *internal.EventStore, interval time.Duration, log *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// Flush inicial no startup (recuperados do disco)
	es.FlushAll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			es.FlushAll(ctx)
		}
	}
}

// heartbeatLoop envia heartbeat periódico.
func heartbeatLoop(ctx context.Context, p *internal.Pusher, c *internal.Collector, cfg *internal.Config, interval time.Duration, version string, log *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	send := func() {
		containers := c.ListContainers()
		running := 0
		for _, ci := range containers {
			if ci.State == "running" {
				running++
			}
		}
		hb := internal.Heartbeat{
			NodeID:          cfg.NodeID,
			Hostname:        cfg.NodeHostname,
			ContainersCount: running,
			Version:         version,
		}
		if err := p.PushHeartbeat(ctx, hb); err != nil {
			log.Warn("heartbeat falhou", "err", err)
		} else {
			log.Debug("heartbeat enviado", "containers", running)
		}
	}
	send()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			send()
		}
	}
}

// infoServer expõe /health e /info na porta local do agent.
func infoServer(ctx context.Context, addr string, c *internal.Collector, es *internal.EventStore, buf *internal.Buffer, version string, log *slog.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		containers := c.ListContainers()
		running := 0
		for _, ci := range containers {
			if ci.State == "running" {
				running++
			}
		}
		fmt.Fprintf(w, `{"version":%q,"containers":%d,"buffer_len":%d,"oom_pending":%d}`,
			version, running, buf.Len(), es.PendingCount())
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()
	log.Info("agent info server iniciado", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("info server falhou", "err", err)
	}
}

// cfgAdapter adapta *internal.Config às interfaces internas (collectorConfig, pusherConfig).
type cfgAdapter struct {
	cfg *internal.Config
}

func (a *cfgAdapter) ExcludedImages() []string { return a.cfg.ExcludedImages }
func (a *cfgAdapter) Debug() bool              { return a.cfg.Debug }
func (a *cfgAdapter) APIURL() string           { return a.cfg.APIURL }
func (a *cfgAdapter) Token() string            { return a.cfg.Token }
func (a *cfgAdapter) NodeID() string           { return a.cfg.NodeID }

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
