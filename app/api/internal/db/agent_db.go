// Package db — queries para agents e tasks (Fase 7 — multi-node).
//
// agents: 1 row por agent ativo (upsert on heartbeat).
// tasks: Swarm task lifecycle (populada pelo manager via TaskList).
// task_history: append-only log de mudanças de status (para restarts per slot).
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Agent é uma linha da tabela agents.
type Agent struct {
	NodeID          string     `json:"node_id"`
	Hostname        string     `json:"hostname"`
	Version         string     `json:"version"`
	ContainersCount int32      `json:"containers_count"`
	LastHeartbeat   *time.Time `json:"last_heartbeat"`
	Status          string     `json:"status"`
	FirstSeen       *time.Time `json:"first_seen"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// UpsertAgent insere ou atualiza um agent com base no heartbeat.
// status="active" e last_heartbeat=now(). Se já existe, preserva first_seen.
func (s *Store) UpsertAgent(ctx context.Context, nodeID, hostname, version string, containersCount int32) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agents (node_id, hostname, version, containers_count, last_heartbeat, status, first_seen, updated_at)
		 VALUES (?, ?, ?, ?, now(), 'active', now(), now())
		 ON CONFLICT (node_id) DO UPDATE SET
		   hostname = COALESCE(excluded.hostname, agents.hostname),
		   version = COALESCE(excluded.version, agents.version),
		   containers_count = excluded.containers_count,
		   last_heartbeat = now(),
		   status = 'active',
		   updated_at = now()`,
		nodeID, hostname, version, containersCount)
	return err
}

// MarkAgentsStale marca agents cujo last_heartbeat é mais antigo que o threshold
// como "stale". Retorna o número de agents marcados.
//
// Cast explícito now()::TIMESTAMP porque last_heartbeat é TIMESTAMP (sem timezone)
// e now() retorna TIMESTAMP WITH TIME ZONE — DuckDB não suporta a subtração
// direta (TIMESTAMP WITH TIME ZONE - INTERVAL).
func (s *Store) MarkAgentsStale(ctx context.Context, threshold time.Duration) (int64, error) {
	secs := int(threshold / time.Second)
	res, err := s.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE agents SET status = 'stale', updated_at = now()
		 WHERE status = 'active' AND last_heartbeat < now()::TIMESTAMP - INTERVAL '%d' SECOND`, secs))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// GetAgents retorna todos os agents ordenados por hostname.
func (s *Store) GetAgents(ctx context.Context) ([]Agent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT node_id, hostname, version, containers_count, last_heartbeat, status, first_seen, updated_at
		 FROM agents ORDER BY hostname`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Agent
	for rows.Next() {
		var a Agent
		if err := rows.Scan(&a.NodeID, &a.Hostname, &a.Version, &a.ContainersCount,
			&a.LastHeartbeat, &a.Status, &a.FirstSeen, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAgentByNode retorna um agent pelo node_id, ou nil se não existir.
func (s *Store) GetAgentByNode(ctx context.Context, nodeID string) (*Agent, error) {
	var a Agent
	err := s.db.QueryRowContext(ctx,
		`SELECT node_id, hostname, version, containers_count, last_heartbeat, status, first_seen, updated_at
		 FROM agents WHERE node_id = ?`, nodeID).
		Scan(&a.NodeID, &a.Hostname, &a.Version, &a.ContainersCount,
			&a.LastHeartbeat, &a.Status, &a.FirstSeen, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Task é uma linha da tabela tasks (snapshot atual do Swarm).
type Task struct {
	TaskID       string     `json:"task_id"`
	Service      string     `json:"service"`
	NodeID       string     `json:"node_id"`
	Slot         int32      `json:"slot"`
	Status       string     `json:"status"`
	DesiredState string     `json:"desired_state"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

// UpsertTask insere ou atualiza uma task (snapshot do Swarm).
// Retorna true se o status mudou (para o caller registrar em task_history).
func (s *Store) UpsertTask(ctx context.Context, t Task) (statusChanged bool, err error) {
	var prevStatus sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT status FROM tasks WHERE task_id = ?`, t.TaskID).Scan(&prevStatus)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO tasks (task_id, service, node_id, slot, status, desired_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, now(), now())
		 ON CONFLICT (task_id) DO UPDATE SET
		   service = COALESCE(excluded.service, tasks.service),
		   node_id = COALESCE(excluded.node_id, tasks.node_id),
		   slot = excluded.slot,
		   status = excluded.status,
		   desired_state = COALESCE(excluded.desired_state, tasks.desired_state),
		   updated_at = now()`,
		t.TaskID, t.Service, t.NodeID, t.Slot, t.Status, t.DesiredState)
	if err != nil {
		return false, err
	}
	if prevStatus.Valid && prevStatus.String != t.Status {
		return true, nil
	}
	if !prevStatus.Valid && t.Status != "" {
		return true, nil
	}
	return false, nil
}

// DeleteTask remove uma task que não existe mais no Swarm (prune).
func (s *Store) DeleteTask(ctx context.Context, taskID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM tasks WHERE task_id = ?", taskID)
	return err
}

// GetTasks retorna todas as tasks ordenadas por service, slot.
func (s *Store) GetTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT task_id, service, node_id, slot, status, desired_state, created_at, updated_at
		 FROM tasks ORDER BY service, slot`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Task
	for rows.Next() {
		var t Task
		var nodeID sql.NullString
		if err := rows.Scan(&t.TaskID, &t.Service, &nodeID, &t.Slot, &t.Status,
			&t.DesiredState, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.NodeID = nodeID.String
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTasksByService retorna as tasks de um serviço.
func (s *Store) GetTasksByService(ctx context.Context, service string) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT task_id, service, node_id, slot, status, desired_state, created_at, updated_at
		 FROM tasks WHERE service = ? ORDER BY slot`, service)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Task
	for rows.Next() {
		var t Task
		var nodeID sql.NullString
		if err := rows.Scan(&t.TaskID, &t.Service, &nodeID, &t.Slot, &t.Status,
			&t.DesiredState, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.NodeID = nodeID.String
		out = append(out, t)
	}
	return out, rows.Err()
}

// InsertTaskHistory registra uma mudança de status de task (append-only).
func (s *Store) InsertTaskHistory(ctx context.Context, t Task) error {
	var nodeVal any
	if t.NodeID != "" {
		nodeVal = t.NodeID
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO task_history (task_id, service, node_id, slot, status, ts)
		 VALUES (?, ?, ?, ?, ?, now())`,
		t.TaskID, t.Service, nodeVal, t.Slot, t.Status)
	return err
}

// TaskRestartCount retorna o número de vezes que uma task (por service+slot)
// passou por um estado de "running" após "failed"/"rejected" — proxy para restarts.
// Conta transições para "running" nos últimos N dias.
func (s *Store) TaskRestartCount(ctx context.Context, service string, slot int32, days int) (int32, error) {
	var count int32
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM task_history
		 WHERE service = ? AND slot = ? AND status = 'running'
		   AND ts > now()::TIMESTAMP - INTERVAL '%d' DAYS`, days),
		service, slot).Scan(&count)
	// Subtrai 1 para não contar o primeiro start como restart.
	if count > 0 {
		count--
	}
	return count, err
}

// ServiceRestartCount retorna o total de restarts de um serviço (soma de todos os slots).
func (s *Store) ServiceRestartCount(ctx context.Context, service string, days int) (int32, error) {
	var count int32
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM task_history
		 WHERE service = ? AND status = 'running'
		   AND ts > now()::TIMESTAMP - INTERVAL '%d' DAYS`, days),
		service).Scan(&count)
	// Não subtrai aqui — o caller decide. Retorna o total de "running" transitions.
	return count, err
}

// TaskHistoryPoint é um ponto do histórico de uma task.
type TaskHistoryPoint struct {
	TS     time.Time `json:"ts"`
	TaskID string    `json:"task_id"`
	Status string    `json:"status"`
	Slot   int32     `json:"slot"`
	NodeID string    `json:"node_id"`
}

// GetTaskHistory retorna o histórico de tasks de um serviço nos últimos N dias.
func (s *Store) GetTaskHistory(ctx context.Context, service string, days int) ([]TaskHistoryPoint, error) {
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT ts, task_id, status, slot, node_id FROM task_history
		 WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL '%d' DAYS
		 ORDER BY ts`, days), service)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []TaskHistoryPoint
	for rows.Next() {
		var p TaskHistoryPoint
		var nodeID sql.NullString
		if err := rows.Scan(&p.TS, &p.TaskID, &p.Status, &p.Slot, &nodeID); err != nil {
			return nil, err
		}
		p.NodeID = nodeID.String
		out = append(out, p)
	}
	return out, rows.Err()
}
