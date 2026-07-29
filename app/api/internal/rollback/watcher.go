// Package rollback implementa o watcher de rollback automático pós-apply.
//
// Segue o padrão do scheduler: 1 goroutine com polling a cada 30s (configurável).
// Monitora entradas de rollback_watches com status='monitoring' e avalia
// critérios de incidente. Se um critério dispara, reverte via Docker SDK
// e registra no change_log com action='rollback', source='auto'.
package rollback

import (
	"context"
	"log/slog"
	"time"

	"github.com/resma/api/internal/config"
	"github.com/resma/api/internal/db"
	"github.com/resma/api/internal/docker"
)

// SSEPublisher — mesma interface do scheduler.
type SSEPublisher interface {
	Publish(topic, eventType string, payload any)
}

// DataBuilder — para publicar payloads completos via SSE (change-log).
type DataBuilder interface {
	BuildChangeLog(ctx context.Context, service string, limit int32) (any, error)
}

// nopPublisher e nopBuilder — no-ops quando SSE/builder não disponíveis.
type nopPublisher struct{}

func (nopPublisher) Publish(string, string, any) {}

type nopBuilder struct{}

func (nopBuilder) BuildChangeLog(context.Context, string, int32) (any, error) { return nil, nil }

// Watcher monitora applies recentes e reverte se critérios de incidente disparam.
type Watcher struct {
	cfg     *config.Config
	db      *db.Store
	docker  *docker.Client
	sse     SSEPublisher
	builder DataBuilder
	log     *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New cria um novo Watcher. sse e builder podem ser nil.
func New(cfg *config.Config, database *db.Store, dc *docker.Client,
	sse SSEPublisher, builder DataBuilder) *Watcher {
	if sse == nil {
		sse = nopPublisher{}
	}
	if builder == nil {
		builder = nopBuilder{}
	}
	return &Watcher{
		cfg:     cfg,
		db:      database,
		docker:  dc,
		sse:     sse,
		builder: builder,
		log:     slog.Default().With("component", "rollback-watcher"),
		done:    make(chan struct{}),
	}
}

// Start inicia a goroutine de polling. No-op se RollbackEnabled=false.
func (w *Watcher) Start(ctx context.Context) {
	if !w.cfg.RollbackEnabled {
		w.log.Info("rollback watcher desabilitado (RESMA_ROLLBACK_ENABLED=false)")
		return
	}
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.log.Info("rollback watcher iniciado",
		"poll_interval", w.cfg.RollbackPollInterval,
		"default_window_h", w.cfg.RollbackDefaultWindow)
	go w.loop()
}

// Stop para a goroutine graciosamente.
func (w *Watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
		<-w.done
	}
	w.log.Info("rollback watcher parado")
}

func (w *Watcher) loop() {
	defer close(w.done)
	ticker := time.NewTicker(w.cfg.RollbackPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.processActiveWatches()
		}
	}
}

// processActiveWatches busca watches ativos e avalia cada um.
func (w *Watcher) processActiveWatches() {
	watches, err := w.db.GetActiveRollbackWatches(w.ctx)
	if err != nil {
		w.log.Error("erro ao buscar watches ativos", "err", err)
		return
	}
	if len(watches) == 0 {
		return
	}
	w.log.Debug("processando watches ativos", "count", len(watches))
	for i := range watches {
		w.evaluateOne(watches[i])
	}
}
