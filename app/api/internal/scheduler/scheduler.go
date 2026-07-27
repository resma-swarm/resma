// Package scheduler implementa o executor de agendamentos pendentes.
//
// Portado de backend/services/scheduler.py. Usa uma goroutine com polling
// a cada 15s. Máximo de 3 tentativas por schedule.
package scheduler

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/resma/api/internal/db"
	"github.com/resma/api/internal/docker"
)

const (
	maxAttempts  = 3
	pollInterval = 15 * time.Second
)

// SSEPublisher é a interface que o scheduler usa para notificar o frontend
// via Server-Sent Events quando um schedule executa. Satisfeita por *sse.Handler.
type SSEPublisher interface {
	Publish(topic, eventType string, payload any)
}

// DataBuilder constrói payloads completos para publicação via SSE.
// Satisfeita por *server.CollectorDataBuilder (subset de schedules/change-log).
type DataBuilder interface {
	BuildSchedulesList(ctx context.Context, status string) (any, error)
	BuildChangeLog(ctx context.Context, service string, limit int32) (any, error)
}

// nopDataBuilder retorna nil para todos os builds.
type nopDataBuilder struct{}

func (nopDataBuilder) BuildSchedulesList(context.Context, string) (any, error) {
	return nil, nil
}
func (nopDataBuilder) BuildChangeLog(context.Context, string, int32) (any, error) {
	return nil, nil
}

// nopSSEPublisher descarta eventos (usado quando SSE é nil).
type nopSSEPublisher struct{}

func (nopSSEPublisher) Publish(string, string, any) {}

// Executor coordena a execução de schedules pendentes.
type Executor struct {
	db      *db.Store
	docker  *docker.Client
	sse     SSEPublisher
	builder DataBuilder
	log     *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New cria um novo Executor. sse e builder podem ser nil (usa nop*).
func New(database *db.Store, dc *docker.Client, sse SSEPublisher, builder DataBuilder) *Executor {
	if sse == nil {
		sse = nopSSEPublisher{}
	}
	if builder == nil {
		builder = nopDataBuilder{}
	}
	return &Executor{
		db:      database,
		docker:  dc,
		sse:     sse,
		builder: builder,
		log:     slog.Default().With("component", "scheduler"),
		done:    make(chan struct{}),
	}
}

// publishChangeLog notifica o frontend via SSE (tópico change-log) que um
// schedule mudou de status. Publica payload completo de schedules + change-log
// para o frontend usar setQueryData — zero refetch.
func (e *Executor) publishChangeLog(status, service string, scheduleID int32) {
	// Publicar payload completo de schedules (status atualizado)
	if schedules, err := e.builder.BuildSchedulesList(e.ctx, ""); err == nil && schedules != nil {
		e.sse.Publish("change-log", "schedules", schedules)
	}
	// Publicar payload completo de change-log (nova entrada)
	if changeLog, err := e.builder.BuildChangeLog(e.ctx, "", 100); err == nil && changeLog != nil {
		e.sse.Publish("change-log", "change-log", changeLog)
	}
}

// Start inicia a goroutine de polling.
func (e *Executor) Start(ctx context.Context) {
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.log.Info("scheduler iniciado")
	go e.loop()
}

// Stop para a goroutine graciosamente.
func (e *Executor) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	<-e.done
	e.log.Info("scheduler parado")
}

// loop é o loop principal de polling.
func (e *Executor) loop() {
	defer close(e.done)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.processPending()
		}
	}
}

// processPending busca schedules pendentes e executa cada um.
func (e *Executor) processPending() {
	pending, err := e.db.GetPendingSchedules(e.ctx)
	if err != nil {
		e.log.Error("erro ao obter schedules pendentes", "err", err)
		return
	}
	for _, item := range pending {
		e.executeOne(item)
	}
}

// executeOne executa um schedule individual.
func (e *Executor) executeOne(item db.Schedule) {
	ctx := e.ctx
	scheduleID := item.ID
	service := item.Service

	if item.Attempts >= maxAttempts {
		_ = e.db.UpdateScheduleStatus(ctx, scheduleID, "failed", "Max attempts reached", nil)
		e.log.Warn("schedule excedeu max tentativas", "id", scheduleID, "service", service)
		e.publishChangeLog("failed", service, scheduleID)
		return
	}

	_ = e.db.UpdateScheduleStatus(ctx, scheduleID, "running", "", nil)
	_ = e.db.IncrementScheduleAttempts(ctx, scheduleID)
	e.publishChangeLog("running", service, scheduleID)

	e.log.Info("aplicando config agendada", "service", service, "schedule_id", scheduleID)

	current, err := e.docker.GetServiceResources(ctx, service)
	if err != nil {
		e.failSchedule(ctx, item, "failed to get current resources: "+err.Error())
		return
	}

	// Verificar se valores já estão aplicados
	cpuLimit := nullFloat(item.CPULimit)
	memLimit := nullInt64(item.MemLimit)
	cpuRes := nullFloat(item.CPUReservation)
	memRes := nullInt64(item.MemReservation)

	if current.CPULimit == cpuLimit && current.MemLimit == memLimit &&
		current.CPUReservation == cpuRes && current.MemReservation == memRes {
		now := time.Now()
		_ = e.db.UpdateScheduleStatus(ctx, scheduleID, "completed", "", &now)
		e.log.Info("schedule skipped — valores já aplicados", "id", scheduleID)
		e.publishChangeLog("completed", service, scheduleID)
		return
	}

	// Aplicar via Docker
	var cpuLimitPtr *float64
	if item.CPULimit.Valid {
		v := item.CPULimit.Float64
		cpuLimitPtr = &v
	}
	var memLimitPtr *float64
	if item.MemLimit.Valid {
		v := float64(item.MemLimit.Int64)
		memLimitPtr = &v
	}
	var cpuResPtr *int64
	if item.CPUReservation.Valid {
		v := int64(item.CPUReservation.Float64 * 1e9)
		cpuResPtr = &v
	}
	var memResPtr *int64
	if item.MemReservation.Valid {
		v := item.MemReservation.Int64
		memResPtr = &v
	}

	result, err := e.docker.UpdateServiceResources(ctx, service, cpuLimitPtr, memLimitPtr, cpuResPtr, memResPtr)
	if err != nil {
		e.failSchedule(ctx, item, err.Error())
		return
	}

	if result.Success {
		now := time.Now()
		_ = e.db.UpdateScheduleStatus(ctx, scheduleID, "completed", "", &now)
		e.log.Info("schedule aplicado com sucesso", "id", scheduleID, "service", service)
		e.publishChangeLog("completed", service, scheduleID)
	} else {
		e.failSchedule(ctx, item, result.Error)
	}
}

// failSchedule marca um schedule como failed e registra no change_log.
func (e *Executor) failSchedule(ctx context.Context, item db.Schedule, errMsg string) {
	_ = e.db.UpdateScheduleStatus(ctx, item.ID, "failed", errMsg, nil)
	e.log.Error("schedule falhou", "id", item.ID, "service", item.Service, "err", errMsg)
	e.publishChangeLog("failed", item.Service, item.ID)
}

// nullFloat extrai o valor de sql.NullFloat64 (0 se inválido).
func nullFloat(n sql.NullFloat64) float64 {
	if n.Valid {
		return n.Float64
	}
	return 0
}

// nullInt64 extrai o valor de sql.NullInt64 (0 se inválido).
func nullInt64(n sql.NullInt64) int64 {
	if n.Valid {
		return n.Int64
	}
	return 0
}
