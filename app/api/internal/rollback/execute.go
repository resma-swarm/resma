package rollback

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/resma/api/internal/db"
)

// evaluateOne avalia um watch: se critério disparou → rollback; se expirou → otimizado.
func (w *Watcher) evaluateOne(watch db.RollbackWatch) {
	ctx := w.ctx

	// 1. Verificar se a janela expirou sem incidente → marcar otimizado
	if time.Now().After(watch.ExpiresAt) {
		w.markOptimized(ctx, watch)
		return
	}

	// 2. Parsear critérios habilitados (JSON do DB)
	criteria, err := parseCriteria(watch.Criteria)
	if err != nil {
		w.log.Error("erro ao parsear critérios", "watch_id", watch.ID, "err", err)
		return
	}

	// 3. Avaliar critérios
	result := w.evaluate(ctx, watch, criteria)
	if !result.Triggered {
		return // ainda monitorando
	}

	// 4. Critério disparou → executar rollback
	w.log.Warn("critério de rollback disparado",
		"watch_id", watch.ID,
		"service", watch.Service,
		"reason", result.Reason)

	w.executeRollback(ctx, watch, result.Reason)
}

// executeRollback reverte o serviço para os valores before via Docker SDK.
func (w *Watcher) executeRollback(ctx context.Context, watch db.RollbackWatch, reason string) {
	// 1. Reverter via Docker (valores before = snapshot)
	var cpuLimit *float64
	if watch.CPULimitBefore.Valid {
		v := watch.CPULimitBefore.Float64
		cpuLimit = &v
	}
	var memLimit *float64
	if watch.MemLimitBefore.Valid {
		v := float64(watch.MemLimitBefore.Int64)
		memLimit = &v
	}
	var cpuRes *int64
	if watch.CPUReservationBefore.Valid {
		v := int64(watch.CPUReservationBefore.Float64 * 1e9)
		cpuRes = &v
	}
	var memRes *int64
	if watch.MemReservationBefore.Valid {
		v := watch.MemReservationBefore.Int64
		memRes = &v
	}

	result, err := w.docker.UpdateServiceResources(ctx, watch.Service,
		cpuLimit, memLimit, cpuRes, memRes)
	if err != nil {
		w.log.Error("falha ao executar rollback via Docker",
			"watch_id", watch.ID, "service", watch.Service, "err", err)
		return
	}

	// 2. Registrar no change_log (action=rollback, source=auto)
	status := "completed"
	if !result.Success {
		status = "failed"
	}
	dockerResp := fmt.Sprintf("rollback reason: %s", reason)
	_, _ = w.db.AddChangeLog(ctx, db.ChangeLogEntry{
		Service:              watch.Service,
		Action:               "rollback",
		Source:               "auto",
		CPULimitBefore:       watch.CPULimitAfter, // after do apply = before do rollback
		MemLimitBefore:       watch.MemLimitAfter,
		CPUReservationBefore: watch.CPUReservationAfter,
		MemReservationBefore: watch.MemReservationAfter,
		CPULimitAfter:        watch.CPULimitBefore, // before do apply = after do rollback
		MemLimitAfter:        watch.MemLimitBefore,
		CPUReservationAfter:  watch.CPUReservationBefore,
		MemReservationAfter:  watch.MemReservationBefore,
		Status:               status,
		DockerResponse:       sql.NullString{String: dockerResp, Valid: true},
	})

	// 3. Atualizar status do watch
	now := time.Now()
	_ = w.db.UpdateRollbackWatchStatus(ctx, watch.ID, "rolled_back", reason, &now)

	// 4. Notificar frontend via SSE
	w.publishRollbackEvent(ctx, watch.Service, reason)

	w.log.Info("rollback executado",
		"watch_id", watch.ID,
		"service", watch.Service,
		"reason", reason,
		"docker_success", result.Success)
}

// markOptimized marca um watch como otimizado (janela expirou sem incidente).
func (w *Watcher) markOptimized(ctx context.Context, watch db.RollbackWatch) {
	_ = w.db.UpdateRollbackWatchStatus(ctx, watch.ID, "optimized", "", nil)
	w.log.Info("watch expirado sem incidente — serviço otimizado",
		"watch_id", watch.ID, "service", watch.Service)
	w.publishOptimizedEvent(ctx, watch.Service)
}

// publishRollbackEvent notifica o frontend via SSE.
func (w *Watcher) publishRollbackEvent(ctx context.Context, service, reason string) {
	// Tópico change-log: payload completo para o frontend atualizar sem refetch
	if changeLog, err := w.builder.BuildChangeLog(ctx, "", 100); err == nil && changeLog != nil {
		w.sse.Publish("change-log", "change-log", changeLog)
	}
	// Tópico events: alerta de rollback para destacar na UI
	w.sse.Publish("events", "rollback", map[string]any{
		"service": service,
		"reason":  reason,
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (w *Watcher) publishOptimizedEvent(ctx context.Context, service string) {
	if changeLog, err := w.builder.BuildChangeLog(ctx, "", 100); err == nil && changeLog != nil {
		w.sse.Publish("change-log", "change-log", changeLog)
	}
	w.sse.Publish("events", "optimized", map[string]any{
		"service": service,
		"time":    time.Now().Format(time.RFC3339),
	})
}
