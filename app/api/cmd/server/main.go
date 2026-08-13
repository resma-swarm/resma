// @title RESMA API
// @version 1.0
// @description RESource MAnager for Docker Swarm — API Go com coleta de métricas, ML para recomendações e detecção de memory leaks.
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/resma-swarm/resma/app/api/docs/swagger"
	"github.com/resma-swarm/resma/app/api/internal/auth"
	"github.com/resma-swarm/resma/app/api/internal/collector"
	"github.com/resma-swarm/resma/app/api/internal/config"
	"github.com/resma-swarm/resma/app/api/internal/db"
	resmadocker "github.com/resma-swarm/resma/app/api/internal/docker"
	"github.com/resma-swarm/resma/app/api/internal/rollback"
	"github.com/resma-swarm/resma/app/api/internal/scheduler"
	"github.com/resma-swarm/resma/app/api/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load falhou", "err", err)
		os.Exit(1)
	}
	logger.Info("config carregada",
		"db", cfg.DBPath,
		"ml_url", cfg.MLURL,
		"ml_enabled", cfg.MLEnabled,
		"collect_interval", cfg.CollectInterval,
		"http_addr", cfg.HTTPAddr,
	)

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// DuckDB
	store, err := db.New(rootCtx, cfg.DBPath)
	if err != nil {
		logger.Error("duckdb init falhou", "err", err, "path", cfg.DBPath)
		os.Exit(1)
	}
	defer func() { _ = store.Close() }()
	logger.Info("duckdb conectado", "path", cfg.DBPath)

	// Docker SDK
	dockerCli, err := resmadocker.New(rootCtx)
	if err != nil {
		logger.Warn("docker client init falhou (continuando em modo sem docker)", "err", err)
	} else {
		defer func() { _ = dockerCli.Close() }()
		if err := dockerCli.Health(rootCtx); err != nil {
			logger.Warn("docker daemon inacessível", "err", err)
		} else {
			logger.Info("docker daemon conectado")
		}
	}

	// Auth service
	authSvc := auth.New(cfg, store)

	// HTTP Server — criado ANTES do collector/scheduler para que estes possam
	// receber o SSE handler e publicar eventos para o frontend.
	srv := server.New(cfg, store, dockerCli, authSvc)

	// Collector (9 goroutines de coleta de métricas + publicação SSE)
	// O CollectorDataBuilder permite que o collector publique payloads
	// completos via SSE (mesmo payload dos GETs), eliminando refetch no frontend.
	col := collector.New(cfg, store, dockerCli, srv.SSEHandler(), server.NewCollectorDataBuilder(srv))
	col.Start(rootCtx)
	defer col.Stop()

	// Scheduler (1 goroutine de polling de schedules pendentes + publicação SSE)
	// O CollectorDataBuilder permite que o scheduler publique payloads completos.
	sch := scheduler.New(store, dockerCli, srv.SSEHandler(), server.NewCollectorDataBuilder(srv), cfg.SchedulerPoll)
	sch.Start(rootCtx)
	defer sch.Stop()

	// Rollback watcher (Right-Sizing Studio R5 — 1 goroutine de monitoramento pós-apply)
	// Opt-in via RESMA_ROLLBACK_ENABLED. Monitora applies recentes e reverte
	// automaticamente se critérios de incidente (OOM/throttle/mem pressure) disparam.
	rbw := rollback.New(cfg, store, dockerCli, srv.SSEHandler(), server.NewCollectorDataBuilder(srv))
	rbw.Start(rootCtx)
	defer rbw.Stop()

	go func() {
		if err := srv.Start(rootCtx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("http server falhou", "err", err)
			stop()
		}
	}()

	<-rootCtx.Done()
	logger.Info("shutdown solicitado")
	logger.Info("bye")
}
