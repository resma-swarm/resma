// Package db — Appender para batch insert de alta performance.
//
// O Appender do go-duckdb é a forma mais eficiente de inserir múltiplas
// linhas: bypassa o parser SQL e escreve diretamente em chunks columnar.
// Cada Appender exige uma conexão dedicada (não é thread-safe), então
// pegamos uma *sql.Conn do pool por batch.
//
// Portado de db.py insert_metrics_batch / insert_node_metrics_batch /
// insert_volume_metrics_batch — mas usando Appender em vez de executemany
// para aproveitar a performance do DuckDB bulk insert.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/marcboeker/go-duckdb"
)

// MetricRow é uma linha da tabela metrics (com colunas novas 0b.2 e Fase 7).
type MetricRow struct {
	TS                  time.Time
	Service             string
	ContainerID         string
	CPUPercent          float64
	CPUUsage            int64
	CPUSystem           int64
	MemUsage            int64
	MemLimit            int64
	MemPercent          float64
	NetRX               int64
	NetTX               int64
	BlockRead           int64
	BlockWrite          int64
	MemWorkingSet       int64  // 0b.2 — working set = usage - inactive_file/cache
	CPUThrottledPeriods int64  // 0b.2 — Docker stats throttling_data
	CPUThrottledTime    int64  // 0b.2 — Docker stats throttling_data
	NodeID              string // Fase 7 — node de origem (NULL se coletado pelo manager local)
	TaskID              string // Fase 7 — Swarm task ID (NULL se não for Swarm task)
	Slot                int32  // Fase 7 — Swarm task slot (NULL/0 se não aplicável)
}

// InsertMetricsBatch insere um lote de métricas via Appender.
// Retorna nil se rows estiver vazio.
func (s *Store) InsertMetricsBatch(ctx context.Context, rows []MetricRow) error {
	if len(rows) == 0 {
		return nil
	}
	return s.appendBatch(ctx, "metrics", func(a *duckdb.Appender) error {
		for _, r := range rows {
			// Colunas nullable (node_id, task_id, slot) usam nil quando vazias
			var nodeID, taskID any
			if r.NodeID != "" {
				nodeID = r.NodeID
			}
			if r.TaskID != "" {
				taskID = r.TaskID
			}
			var slot any
			if r.Slot != 0 {
				slot = r.Slot
			}
			if err := a.AppendRow(
				r.TS, r.Service, r.ContainerID,
				r.CPUPercent, r.CPUUsage, r.CPUSystem,
				r.MemUsage, r.MemLimit, r.MemPercent,
				r.NetRX, r.NetTX, r.BlockRead, r.BlockWrite,
				r.MemWorkingSet, r.CPUThrottledPeriods, r.CPUThrottledTime,
				nodeID, taskID, slot,
			); err != nil {
				return fmt.Errorf("append metric row: %w", err)
			}
		}
		return nil
	})
}

// NodeMetricRow é uma linha da tabela node_metrics.
type NodeMetricRow struct {
	TS           time.Time
	NodeID       string
	Hostname     string
	Role         string
	Availability string
	Status       string
	CPUTotal     float64
	MemTotal     int64
	TasksRunning int32
}

// InsertNodeMetricsBatch insere um lote de node_metrics via Appender.
func (s *Store) InsertNodeMetricsBatch(ctx context.Context, rows []NodeMetricRow) error {
	if len(rows) == 0 {
		return nil
	}
	return s.appendBatch(ctx, "node_metrics", func(a *duckdb.Appender) error {
		for _, r := range rows {
			if err := a.AppendRow(
				r.TS, r.NodeID, r.Hostname, r.Role, r.Availability, r.Status,
				r.CPUTotal, r.MemTotal, r.TasksRunning,
			); err != nil {
				return fmt.Errorf("append node metric row: %w", err)
			}
		}
		return nil
	})
}

// VolumeMetricRow é uma linha da tabela volume_metrics.
type VolumeMetricRow struct {
	TS               time.Time
	VolumeName       string
	SizeBytes        int64
	ReclaimableBytes int64
	InUse            bool
}

// InsertVolumeMetricsBatch insere um lote de volume_metrics via Appender.
func (s *Store) InsertVolumeMetricsBatch(ctx context.Context, rows []VolumeMetricRow) error {
	if len(rows) == 0 {
		return nil
	}
	return s.appendBatch(ctx, "volume_metrics", func(a *duckdb.Appender) error {
		for _, r := range rows {
			if err := a.AppendRow(
				r.TS, r.VolumeName, r.SizeBytes, r.ReclaimableBytes, r.InUse,
			); err != nil {
				return fmt.Errorf("append volume metric row: %w", err)
			}
		}
		return nil
	})
}

// appendBatch pega uma *sql.Conn dedicada do pool, cria um Appender para a
// tabela informada, executa a função de append e fecha o Appender (que faz
// flush automático). O Appender não é thread-safe — uma Conn por batch.
func (s *Store) appendBatch(ctx context.Context, table string, fn func(*duckdb.Appender) error) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn for appender: %w", err)
	}
	defer func() { _ = conn.Close() }()

	var app *duckdb.Appender
	if err := conn.Raw(func(driverConn any) error {
		dc, ok := driverConn.(*duckdb.Conn)
		if !ok {
			return fmt.Errorf("driver conn is %T, not *duckdb.Conn", driverConn)
		}
		var aerr error
		app, aerr = duckdb.NewAppenderFromConn(dc, "", table)
		return aerr
	}); err != nil {
		return fmt.Errorf("new appender for %q: %w", table, err)
	}
	// Close faz flush automático; erro de Close tem prioridade.
	defer func() { _ = app.Close() }()

	if err := fn(app); err != nil {
		return err
	}
	return app.Flush()
}

// InsertOOMEvent insere um único evento OOM (sem node_id — coletado pelo manager).
func (s *Store) InsertOOMEvent(ctx context.Context, ts time.Time, service, containerID string, exitCode int32) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO oom_events (ts, service, container_id, exit_code) VALUES (?, ?, ?, ?)",
		ts, service, containerID, exitCode,
	)
	return err
}

// InsertOOMEventWithNode insere um evento OOM com node_id (vindo de um agent).
// node_id pode ser "" → armazena NULL.
func (s *Store) InsertOOMEventWithNode(ctx context.Context, ts time.Time, service, containerID, nodeID string, exitCode int32) error {
	var nodeVal any
	if nodeID != "" {
		nodeVal = nodeID
	}
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO oom_events (ts, service, container_id, exit_code, node_id) VALUES (?, ?, ?, ?, ?)",
		ts, service, containerID, exitCode, nodeVal,
	)
	return err
}

// InsertStorageSummary insere um snapshot de storage_summary.
func (s *Store) InsertStorageSummary(ctx context.Context, ts time.Time,
	imagesCount int32, imagesSize, imagesReclaimable int64,
	containersCount int32, containersSize int64,
	volumesCount int32, volumesSize, volumesReclaimable int64,
	volumesOrphanCount int32, volumesOrphanSize int64,
) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO storage_summary VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ts, imagesCount, imagesSize, imagesReclaimable,
		containersCount, containersSize,
		volumesCount, volumesSize, volumesReclaimable,
		volumesOrphanCount, volumesOrphanSize,
	)
	return err
}

// garante que sql.Conn é referenciado (usado em appendBatch).
var _ *sql.Conn
