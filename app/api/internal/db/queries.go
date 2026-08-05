// Package db — queries portadas de backend/core/db.py.
//
// Mantém paridade com os métodos da classe Database do Python. Cada método
// retorna structs tipadas (em vez de dicts) para type-safety nos handlers Go.
// Timestamps são retornados como time.Time e convertidos para ISO 8601 na
// camada de serialização JSON (handlers).
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/marcboeker/go-duckdb"
)

// ---------------------------------------------------------------------------
// app_settings
// ---------------------------------------------------------------------------

// GetSetting retorna o valor de uma chave em app_settings, ou "" se não existir.
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var val string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM app_settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// SetSetting faz upsert de uma chave em app_settings.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO app_settings (key, value) VALUES (?, ?)", key, value)
	return err
}

// CountSettings retorna o número de chaves em app_settings.
// Usado para determinar se o onboarding de settings já foi feito.
func (s *Store) CountSettings(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM app_settings").Scan(&n)
	return n, err
}

// ---------------------------------------------------------------------------
// retention
// ---------------------------------------------------------------------------

// RunRetention deleta dados antigos conforme retentionDays.
// Fase 8: expandido de 3 para 7 tabelas time-series.
func (s *Store) RunRetention(ctx context.Context, retentionDays int) error {
	// Tabelas com coluna ts (time-series)
	tsTables := []string{"metrics", "oom_events", "node_metrics", "volume_metrics", "storage_summary", "task_history"}
	for _, table := range tsTables {
		// DuckDB não suporta placeholder em INTERVAL — usamos concat segura
		// (retentionDays é int controlado pela config, não user input).
		stmt := fmt.Sprintf("DELETE FROM %s WHERE ts < now()::TIMESTAMP - INTERVAL %d DAYS", table, retentionDays)
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("retention %s: %w", table, err)
		}
	}
	// change_log usa created_at (não ts)
	stmt := fmt.Sprintf("DELETE FROM change_log WHERE created_at < now()::TIMESTAMP - INTERVAL %d DAYS", retentionDays)
	if _, err := s.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("retention change_log: %w", err)
	}
	return nil
}

// CleanupExpiredRefreshTokens remove refresh tokens expirados ou revogados.
// Fase 8: novo — evita acúmulo indefinido de tokens.
func (s *Store) CleanupExpiredRefreshTokens(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < now() OR revoked = true`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ---------------------------------------------------------------------------
// stale-marking (Fase 8)
// ---------------------------------------------------------------------------

// MarkStaleServices marca services cujo last_seen é mais antigo que o threshold
// como 'stale'. Retorna o número de services marcados.
//
// Cast explícito now()::TIMESTAMP porque last_seen é TIMESTAMP (sem timezone)
// e now() retorna TIMESTAMP WITH TIME ZONE — DuckDB não suporta a subtração
// direta (TIMESTAMP WITH TIME ZONE - INTERVAL).
func (s *Store) MarkStaleServices(ctx context.Context, days int) (int64, error) {
	res, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`UPDATE service_registry SET status = 'stale', updated_at = now()
		 WHERE status = 'active' AND last_seen < now()::TIMESTAMP - INTERVAL %d DAYS`, days))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// MarkStaleNodes marca nodes cujo updated_at é mais antigo que o threshold
// como 'stale'. Retorna o número de nodes marcados.
func (s *Store) MarkStaleNodes(ctx context.Context, days int) (int64, error) {
	res, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`UPDATE nodes SET status = 'stale', updated_at = now()
		 WHERE status != 'stale' AND updated_at < now()::TIMESTAMP - INTERVAL %d DAYS`, days))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PruneContainerMap remove entradas de container_node_map para containers que
// não existem mais em metrics (órfãos). Retorna o número de linhas removidas.
func (s *Store) PruneContainerMap(ctx context.Context, days int) (int64, error) {
	res, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`DELETE FROM container_node_map WHERE updated_at < now()::TIMESTAMP - INTERVAL %d DAYS`, days))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ---------------------------------------------------------------------------
// prune helpers (Fase 8)
// ---------------------------------------------------------------------------

// PruneStaleServices remove services com status='stale' do service_registry.
// Retorna o número de linhas removidas.
func (s *Store) PruneStaleServices(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM service_registry WHERE status = 'stale'`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PruneStaleNodes remove nodes com status='stale'.
// Retorna o número de linhas removidas.
func (s *Store) PruneStaleNodes(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM nodes WHERE status = 'stale'`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PruneOrphanTasks remove tasks que não existem mais no Swarm (status='removed').
// Retorna o número de linhas removidas.
func (s *Store) PruneOrphanTasks(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE status = 'removed' OR status = 'orphaned'`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PruneMetrics remove todas as métricas (usado com cautela — prune manual).
// Retorna o número de linhas removidas.
func (s *Store) PruneMetrics(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM metrics`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PruneChangeLog remove todo o change_log (usado com cautela).
// Retorna o número de linhas removidas.
func (s *Store) PruneChangeLog(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM change_log`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// PruneVolumeMetrics remove todas as volume_metrics (usado com cautela).
// Retorna o número de linhas removidas.
func (s *Store) PruneVolumeMetrics(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM volume_metrics`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// CountStaleServices retorna o número de services com status='stale'.
func (s *Store) CountStaleServices(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM service_registry WHERE status = 'stale'`).Scan(&n)
	return n, err
}

// CountStaleNodes retorna o número de nodes com status='stale'.
func (s *Store) CountStaleNodes(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM nodes WHERE status = 'stale'`).Scan(&n)
	return n, err
}

// CountOrphanTasks retorna o número de tasks com status='removed' ou 'orphaned'.
func (s *Store) CountOrphanTasks(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM tasks WHERE status = 'removed' OR status = 'orphaned'`).Scan(&n)
	return n, err
}

// CountMetrics retorna o número total de linhas em metrics.
func (s *Store) CountMetrics(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM metrics`).Scan(&n)
	return n, err
}

// CountChangeLog retorna o número total de linhas em change_log.
func (s *Store) CountChangeLog(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM change_log`).Scan(&n)
	return n, err
}

// CountVolumeMetrics retorna o número total de linhas em volume_metrics.
func (s *Store) CountVolumeMetrics(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM volume_metrics`).Scan(&n)
	return n, err
}

// ---------------------------------------------------------------------------
// users
// ---------------------------------------------------------------------------

// User representa um usuário autenticável.
type User struct {
	ID           int32
	Username     string
	PasswordHash string
	Role         string
	Name         string // opcional (display name)
}

// GetUserCount retorna o número de usuários cadastrados.
func (s *Store) GetUserCount(ctx context.Context) (int32, error) {
	var n int32
	err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&n)
	return n, err
}

// GetUserByUsername busca um usuário pelo username.
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, username, password_hash, role, COALESCE(name, '') FROM users WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// GetUserByID busca um usuário pelo id (sem password_hash).
func (s *Store) GetUserByID(ctx context.Context, userID int32) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx,
		"SELECT id, username, role, COALESCE(name, '') FROM users WHERE id = ?", userID).
		Scan(&u.ID, &u.Username, &u.Role, &u.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// CreateUser insere um novo usuário admin e retorna o id.
// Deprecated: Fase 8 prefere CreateUserWithRole para especificar role explicitamente.
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (int32, error) {
	var id int32
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin') RETURNING id",
		username, passwordHash).Scan(&id)
	return id, err
}

// CreateUserWithRole insere um novo usuário com role específico e retorna o id.
// Fase 8: substitui CreateUser para suportar roles owner/admin/user.
// name é opcional (display name); string vazia = NULL no banco.
func (s *Store) CreateUserWithRole(ctx context.Context, username, passwordHash, role, name string) (int32, error) {
	var id int32
	var nameArg any
	if name == "" {
		nameArg = nil
	} else {
		nameArg = name
	}
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO users (username, password_hash, role, name) VALUES (?, ?, ?, ?) RETURNING id",
		username, passwordHash, role, nameArg).Scan(&id)
	return id, err
}

// UpdateUserPassword atualiza o hash de senha de um usuário.
func (s *Store) UpdateUserPassword(ctx context.Context, userID int32, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = ?, updated_at = now() WHERE id = ?",
		passwordHash, userID)
	return err
}

// ---------------------------------------------------------------------------
// users — Fase 8: RBAC + User CRUD
// ---------------------------------------------------------------------------

// ListUsers retorna todos os usuários (sem password_hash) ordenados por created_at.
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, role, COALESCE(name, '') FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Name); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// GetOwner retorna o usuário com role 'owner', ou nil se não existir.
// Usado para validação "apenas 1 owner" no User CRUD.
func (s *Store) GetOwner(ctx context.Context) (*User, error) {
	u := &User{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, role FROM users WHERE role = 'owner' LIMIT 1`).
		Scan(&u.ID, &u.Username, &u.Role)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// UpdateUserRole atualiza o role de um usuário.
func (s *Store) UpdateUserRole(ctx context.Context, userID int32, role string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET role = ?, updated_at = now() WHERE id = ?`, role, userID)
	return err
}

// UpdateUserName atualiza o name (display name) de um usuário.
// String vazia define name como NULL.
func (s *Store) UpdateUserName(ctx context.Context, userID int32, name string) error {
	var nameArg any
	if name == "" {
		nameArg = nil
	} else {
		nameArg = name
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET name = ?, updated_at = now() WHERE id = ?`, nameArg, userID)
	return err
}

// DeleteUser remove um usuário e revoga todos os seus refresh tokens.
func (s *Store) DeleteUser(ctx context.Context, userID int32) error {
	// Revogar todos os refresh tokens primeiro
	_, _ = s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked = true WHERE user_id = ?`, userID)
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	return err
}

// ---------------------------------------------------------------------------
// refresh_tokens
// ---------------------------------------------------------------------------

// RefreshToken representa um refresh token persistido.
type RefreshToken struct {
	Token     string
	UserID    int32
	ExpiresAt time.Time
	Revoked   bool
}

// SaveRefreshToken persiste um novo refresh token.
func (s *Store) SaveRefreshToken(ctx context.Context, token string, userID int32, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, userID, expiresAt)
	return err
}

// GetRefreshToken busca um refresh token pelo valor.
func (s *Store) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := s.db.QueryRowContext(ctx,
		"SELECT token, user_id, expires_at, revoked FROM refresh_tokens WHERE token = ?", token).
		Scan(&rt.Token, &rt.UserID, &rt.ExpiresAt, &rt.Revoked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rt, err
}

// RevokeRefreshToken marca um token como revogado.
func (s *Store) RevokeRefreshToken(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = true WHERE token = ?", token)
	return err
}

// RevokeAllUserTokens revoga todos os tokens de um usuário.
func (s *Store) RevokeAllUserTokens(ctx context.Context, userID int32) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked = true WHERE user_id = ?", userID)
	return err
}

// ---------------------------------------------------------------------------
// service_configs
// ---------------------------------------------------------------------------

// ServiceConfig representa a config de limites de um serviço.
type ServiceConfig struct {
	Service        string
	CPULimit       sql.NullFloat64
	MemLimit       sql.NullInt64
	CPUReservation sql.NullFloat64
	MemReservation sql.NullInt64
	Template       sql.NullString
	UpdatedAt      time.Time
}

// UpsertServiceConfig faz upsert da config de um serviço.
func (s *Store) UpsertServiceConfig(ctx context.Context, c ServiceConfig) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO service_configs
		 (service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, template, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, now())`,
		c.Service, c.CPULimit, c.MemLimit, c.CPUReservation, c.MemReservation, c.Template)
	return err
}

// ---------------------------------------------------------------------------
// templates
// ---------------------------------------------------------------------------

// Template representa um template YAML de limites.
type Template struct {
	ID          int32
	Name        string
	Description string
	YAMLContent string
	Stacks      string // JSON array as string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ListTemplates retorna todos os templates ordenados por nome.
func (s *Store) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, yaml_content, stacks, created_at, updated_at
		 FROM templates ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.YAMLContent, &t.Stacks, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTemplateByID busca um template pelo id.
func (s *Store) GetTemplateByID(ctx context.Context, id int32) (*Template, error) {
	t := &Template{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, yaml_content, stacks, created_at, updated_at
		 FROM templates WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.Description, &t.YAMLContent, &t.Stacks, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// GetTemplateByName busca um template pelo nome.
func (s *Store) GetTemplateByName(ctx context.Context, name string) (*Template, error) {
	t := &Template{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, yaml_content, stacks, created_at, updated_at
		 FROM templates WHERE name = ?`, name).
		Scan(&t.ID, &t.Name, &t.Description, &t.YAMLContent, &t.Stacks, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// GetTemplateByStack busca o primeiro template cujo array stacks contém `stack`.
// Se nenhum match, retorna o template "default".
func (s *Store) GetTemplateByStack(ctx context.Context, stack string) (*Template, error) {
	templates, err := s.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range templates {
		var stacks []string
		if t.Stacks != "" {
			_ = json.Unmarshal([]byte(t.Stacks), &stacks)
		}
		for _, st := range stacks {
			if st == stack {
				return &t, nil
			}
		}
	}
	return s.GetTemplateByName(ctx, "default")
}

// CreateTemplate insere um novo template e retorna o id.
func (s *Store) CreateTemplate(ctx context.Context, name, description, yamlContent, stacks string) (int32, error) {
	var id int32
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO templates (name, description, yaml_content, stacks, created_at, updated_at)
		 VALUES (?, ?, ?, ?, now(), now()) RETURNING id`,
		name, description, yamlContent, stacks).Scan(&id)
	return id, err
}

// UpdateTemplate atualiza campos de um template existente.
// Campos nil não são atualizados.
//
// Caso especial para `name`: o DuckDB tem uma limitação conhecida de
// "over-eager unique constraint checking" que faz UPDATE de colunas UNIQUE
// falhar com duplicate key (delete+insert interno não remove o índice a tempo).
// Ver: https://duckdb.org/docs/sql/indexes#over-eager-unique-constraint-checking
//
// Quando `name` muda, usamos delete+insert numa transação. O novo id é
// auto-gerado (não preservamos o id antigo) para evitar conflito no índice
// da PRIMARY KEY — o id antigo ainda está no índice até o commit. Isso é
// seguro porque nada referencia o template id externamente (service_config
// armazena o template NAME, não o id).
func (s *Store) UpdateTemplate(ctx context.Context, id int32, name, description, yamlContent, stacks *string) error {
	if name != nil {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer tx.Rollback() //nolint:errcheck

		var curDesc, curYAML, curStacks string
		var createdAt time.Time
		err = tx.QueryRowContext(ctx,
			`SELECT description, yaml_content, stacks, created_at FROM templates WHERE id = ?`, id).
			Scan(&curDesc, &curYAML, &curStacks, &createdAt)
		if err != nil {
			return err
		}
		newDesc := curDesc
		if description != nil {
			newDesc = *description
		}
		newYAML := curYAML
		if yamlContent != nil {
			newYAML = *yamlContent
		}
		newStacks := curStacks
		if stacks != nil {
			newStacks = *stacks
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM templates WHERE id = ?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO templates (name, description, yaml_content, stacks, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, now())`,
			*name, newDesc, newYAML, newStacks, createdAt); err != nil {
			return err
		}
		return tx.Commit()
	}

	// Sem rename: UPDATE normal dos campos não-UNIQUE.
	setClauses := []string{"updated_at = now()"}
	var args []any
	if description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *description)
	}
	if yamlContent != nil {
		setClauses = append(setClauses, "yaml_content = ?")
		args = append(args, *yamlContent)
	}
	if stacks != nil {
		setClauses = append(setClauses, "stacks = ?")
		args = append(args, *stacks)
	}
	args = append(args, id)
	query := "UPDATE templates SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteTemplate remove um template pelo id.
func (s *Store) DeleteTemplate(ctx context.Context, id int32) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM templates WHERE id = ?", id)
	return err
}

// ---------------------------------------------------------------------------
// service_registry
// ---------------------------------------------------------------------------

// ServiceRegistryEntry representa o estado de um serviço no registry.
type ServiceRegistryEntry struct {
	Status   string
	LastSeen time.Time
}

// UpsertServiceRegistry faz upsert do estado de um serviço.
func (s *Store) UpsertServiceRegistry(ctx context.Context, service, status string, lastSeen time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO service_registry (service, status, last_seen, updated_at)
		 VALUES (?, ?, ?, now())
		 ON CONFLICT(service) DO UPDATE SET
		     status = excluded.status,
		     last_seen = excluded.last_seen,
		     updated_at = now()`,
		service, status, lastSeen)
	return err
}

// GetServiceRegistry retorna um map service -> entry.
func (s *Store) GetServiceRegistry(ctx context.Context) (map[string]ServiceRegistryEntry, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT service, status, last_seen FROM service_registry")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]ServiceRegistryEntry)
	for rows.Next() {
		var svc, status string
		var lastSeen sql.NullTime
		if err := rows.Scan(&svc, &status, &lastSeen); err != nil {
			return nil, err
		}
		e := ServiceRegistryEntry{Status: status}
		if lastSeen.Valid {
			e.LastSeen = lastSeen.Time
		}
		out[svc] = e
	}
	return out, rows.Err()
}

// SetServiceArchived marca um serviço como archived/active.
func (s *Store) SetServiceArchived(ctx context.Context, service string, archived bool) error {
	status := "active"
	if archived {
		status = "archived"
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE service_registry SET status = ?, updated_at = now() WHERE service = ?",
		status, service)
	return err
}

// ---------------------------------------------------------------------------
// nodes
// ---------------------------------------------------------------------------

// Node representa um nó do Swarm.
type Node struct {
	NodeID        string
	Hostname      string
	Role          string
	Availability  string
	Status        string
	Address       string
	CPUTotal      float64
	MemTotal      int64
	OS            string
	Architecture  string
	EngineVersion string
	IsLeader      bool
	Reachability  string
	Labels        string // JSON
	TasksRunning  int32
	UpdatedAt     time.Time
}

// UpsertNode faz upsert de um nó.
func (s *Store) UpsertNode(ctx context.Context, n Node) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO nodes
		 (node_id, hostname, role, availability, status, address, cpu_total, mem_total,
		  os, architecture, engine_version, is_leader, reachability, labels, tasks_running, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())`,
		n.NodeID, n.Hostname, n.Role, n.Availability, n.Status, n.Address,
		n.CPUTotal, n.MemTotal, n.OS, n.Architecture, n.EngineVersion,
		n.IsLeader, n.Reachability, n.Labels, n.TasksRunning)
	return err
}

// GetNodes retorna todos os nós ordenados por role, hostname.
func (s *Store) GetNodes(ctx context.Context) ([]Node, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT * FROM nodes ORDER BY role, hostname")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Node
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.NodeID, &n.Hostname, &n.Role, &n.Availability, &n.Status,
			&n.Address, &n.CPUTotal, &n.MemTotal, &n.OS, &n.Architecture, &n.EngineVersion,
			&n.IsLeader, &n.Reachability, &n.Labels, &n.TasksRunning, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// GetNodeByID busca um nó pelo id.
func (s *Store) GetNodeByID(ctx context.Context, nodeID string) (*Node, error) {
	n := &Node{}
	err := s.db.QueryRowContext(ctx, "SELECT * FROM nodes WHERE node_id = ?", nodeID).
		Scan(&n.NodeID, &n.Hostname, &n.Role, &n.Availability, &n.Status,
			&n.Address, &n.CPUTotal, &n.MemTotal, &n.OS, &n.Architecture, &n.EngineVersion,
			&n.IsLeader, &n.Reachability, &n.Labels, &n.TasksRunning, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return n, err
}

// ---------------------------------------------------------------------------
// cluster
// ---------------------------------------------------------------------------

// Cluster representa o snapshot do cluster Swarm.
type Cluster struct {
	ID            string
	NodesTotal    int32
	ManagersTotal int32
	WorkersTotal  int32
	NodesReady    int32
	NodesDown     int32
	QuorumHealthy bool
	SelfNodeID    string
	Warnings      string // JSON
	UpdatedAt     time.Time
}

// UpsertCluster faz upsert do snapshot do cluster.
func (s *Store) UpsertCluster(ctx context.Context, c Cluster) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO cluster
		 (id, nodes_total, managers_total, workers_total, nodes_ready, nodes_down,
		  quorum_healthy, self_node_id, warnings, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now())`,
		c.ID, c.NodesTotal, c.ManagersTotal, c.WorkersTotal, c.NodesReady, c.NodesDown,
		c.QuorumHealthy, c.SelfNodeID, c.Warnings)
	return err
}

// GetCluster retorna o snapshot mais recente do cluster.
func (s *Store) GetCluster(ctx context.Context) (*Cluster, error) {
	c := &Cluster{}
	err := s.db.QueryRowContext(ctx, "SELECT * FROM cluster ORDER BY updated_at DESC LIMIT 1").
		Scan(&c.ID, &c.NodesTotal, &c.ManagersTotal, &c.WorkersTotal, &c.NodesReady, &c.NodesDown,
			&c.QuorumHealthy, &c.SelfNodeID, &c.Warnings, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// ---------------------------------------------------------------------------
// container_node_map
// ---------------------------------------------------------------------------

// ContainerNodeMapRow é uma linha para upsert em container_node_map.
type ContainerNodeMapRow struct {
	ContainerID string
	NodeID      string
	Service     string
}

// UpsertContainerNodeMapBatch faz upsert em lote de mapeamentos container→nó.
//
// Pattern: temp table + Appender + MERGE INTO (bypassa SQL parser no insert,
// faz upsert em single statement). ~10x-180x mais rápido que INSERT 1-por-1.
// Ver docs/specs/oss/phase-perf-ingest/spec.md seção 3.1.
func (s *Store) UpsertContainerNodeMapBatch(ctx context.Context, rows []ContainerNodeMapRow) error {
	if len(rows) == 0 {
		return nil
	}
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn for upsert: %w", err)
	}
	defer conn.Close()

	// Temp table é session-scoped (dropped on conn close). Criar/clear por batch.
	if _, err := conn.ExecContext(ctx,
		`CREATE TEMP TABLE IF NOT EXISTS _cnm_staging (
			container_id VARCHAR,
			node_id      VARCHAR,
			service      VARCHAR,
			updated_at   TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("create staging: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `DELETE FROM _cnm_staging`); err != nil {
		return fmt.Errorf("clear staging: %w", err)
	}

	// Appender para popular staging (ultra rápido, bypassa SQL parser).
	now := time.Now()
	if err := func() error {
		var app *duckdb.Appender
		if err := conn.Raw(func(driverConn any) error {
			dc, ok := driverConn.(*duckdb.Conn)
			if !ok {
				return fmt.Errorf("driver conn is %T, not *duckdb.Conn", driverConn)
			}
			var aerr error
			app, aerr = duckdb.NewAppenderFromConn(dc, "", "_cnm_staging")
			return aerr
		}); err != nil {
			return fmt.Errorf("new appender for staging: %w", err)
		}
		defer app.Close()
		for _, r := range rows {
			if err := app.AppendRow(r.ContainerID, r.NodeID, r.Service, now); err != nil {
				return fmt.Errorf("append staging row: %w", err)
			}
		}
		return app.Flush()
	}(); err != nil {
		return err
	}

	// MERGE: insere novos, atualiza existentes (single statement, usa PK).
	_, err = conn.ExecContext(ctx,
		`INSERT INTO container_node_map (container_id, node_id, service, updated_at)
		 SELECT container_id, node_id, service, updated_at FROM _cnm_staging
		 ON CONFLICT (container_id) DO UPDATE SET
		     node_id = excluded.node_id,
		     service = excluded.service,
		     updated_at = excluded.updated_at`)
	if err != nil {
		return fmt.Errorf("merge into container_node_map: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// node_metrics + node consumption
// ---------------------------------------------------------------------------

// NodeMetricPoint é um ponto de métrica de nó.
type NodeMetricPoint struct {
	TS           time.Time `json:"ts"`
	TasksRunning int32     `json:"tasks_running"`
	CPUTotal     float64   `json:"cpu_total"`
	MemTotal     int64     `json:"mem_total"`
}

// GetNodeMetrics retorna métricas históricas de um nó.
func (s *Store) GetNodeMetrics(ctx context.Context, nodeID string, days int) ([]NodeMetricPoint, error) {
	stmt := fmt.Sprintf(
		`SELECT ts, tasks_running, cpu_total, mem_total FROM node_metrics
		 WHERE node_id = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS ORDER BY ts`, days)
	rows, err := s.db.QueryContext(ctx, stmt, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NodeMetricPoint
	for rows.Next() {
		var p NodeMetricPoint
		if err := rows.Scan(&p.TS, &p.TasksRunning, &p.CPUTotal, &p.MemTotal); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// NodeConsumption agrega consumo de um nó.
type NodeConsumption struct {
	CPUP95        float64
	MemP99        int64
	MemTotalUsage int64
	Containers    int32
}

// GetNodeConsumption retorna agregados de consumo de um nó.
func (s *Store) GetNodeConsumption(ctx context.Context, nodeID string, analysisWindowDays int) (NodeConsumption, error) {
	stmt := fmt.Sprintf(`
		SELECT
			quantile(m.cpu_percent, 0.95) as cpu_p95,
			quantile(m.mem_usage, 0.99) as mem_p99,
			sum(m.mem_usage) as mem_total_usage,
			count(DISTINCT m.container_id) as containers
		FROM metrics m
		JOIN container_node_map c ON m.container_id = c.container_id
		WHERE c.node_id = ? AND m.ts > now()::TIMESTAMP - INTERVAL %d DAYS`, analysisWindowDays)
	var c NodeConsumption
	var cpuP95, memP99 sql.NullFloat64
	var memTotal sql.NullInt64
	var containers sql.NullInt32
	err := s.db.QueryRowContext(ctx, stmt, nodeID).
		Scan(&cpuP95, &memP99, &memTotal, &containers)
	if err == sql.ErrNoRows {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	c.CPUP95 = round2(cpuP95.Float64)
	c.MemP99 = int64(memP99.Float64)
	c.MemTotalUsage = memTotal.Int64
	c.Containers = containers.Int32
	return c, nil
}

// NodeService agrega um serviço rodando em um nó.
type NodeService struct {
	Service    string  `json:"service"`
	Containers int32   `json:"containers"`
	CPUP95     float64 `json:"cpu_p95"`
	MemP99     int64   `json:"mem_p99"`
}

// GetNodeServices retorna os serviços rodando em um nó com agregados.
func (s *Store) GetNodeServices(ctx context.Context, nodeID string, analysisWindowDays int) ([]NodeService, error) {
	stmt := fmt.Sprintf(`
		SELECT
			c.service,
			count(DISTINCT m.container_id) as containers,
			quantile(m.cpu_percent, 0.95) as cpu_p95,
			quantile(m.mem_usage, 0.99) as mem_p99
		FROM metrics m
		JOIN container_node_map c ON m.container_id = c.container_id
		WHERE c.node_id = ? AND m.ts > now()::TIMESTAMP - INTERVAL %d DAYS
		GROUP BY c.service ORDER BY cpu_p95 DESC`, analysisWindowDays)
	rows, err := s.db.QueryContext(ctx, stmt, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NodeService
	for rows.Next() {
		var ns NodeService
		var containers sql.NullInt32
		var cpuP95, memP99 sql.NullFloat64
		if err := rows.Scan(&ns.Service, &containers, &cpuP95, &memP99); err != nil {
			return nil, err
		}
		ns.Containers = containers.Int32
		ns.CPUP95 = round2(cpuP95.Float64)
		ns.MemP99 = int64(memP99.Float64)
		out = append(out, ns)
	}
	return out, rows.Err()
}

// ClusterSummary agrega capacidade e consumo do cluster.
type ClusterSummary struct {
	CPUTotal   float64 `json:"cpu_total"`
	MemTotal   int64   `json:"mem_total"`
	TasksTotal int32   `json:"tasks_total"`
	CPUP95     float64 `json:"cpu_p95"`
	MemUsage   int64   `json:"mem_usage"`
}

// GetClusterSummary retorna capacidade total + consumo agregado do cluster.
func (s *Store) GetClusterSummary(ctx context.Context, analysisWindowDays int) (ClusterSummary, error) {
	var cs ClusterSummary
	var cpuTotal sql.NullFloat64
	var memTotal sql.NullInt64
	var tasksTotal sql.NullInt32
	err := s.db.QueryRowContext(ctx,
		"SELECT sum(cpu_total), sum(mem_total), sum(tasks_running) FROM nodes WHERE status = 'ready'").
		Scan(&cpuTotal, &memTotal, &tasksTotal)
	if err != nil {
		return cs, err
	}
	cs.CPUTotal = round1(cpuTotal.Float64)
	cs.MemTotal = memTotal.Int64
	cs.TasksTotal = tasksTotal.Int32

	stmt := fmt.Sprintf(`
		SELECT quantile(m.cpu_percent, 0.95), avg(m.mem_usage)
		FROM metrics m
		JOIN container_node_map c ON m.container_id = c.container_id
		WHERE m.ts > now()::TIMESTAMP - INTERVAL %d DAYS`, analysisWindowDays)
	var cpuP95 sql.NullFloat64
	var memUsage sql.NullFloat64
	err = s.db.QueryRowContext(ctx, stmt).Scan(&cpuP95, &memUsage)
	if err != nil {
		return cs, err
	}
	cs.CPUP95 = round2(cpuP95.Float64)
	cs.MemUsage = int64(memUsage.Float64)
	return cs, nil
}

// ---------------------------------------------------------------------------
// schedules
// ---------------------------------------------------------------------------

// Schedule representa um agendamento de aplicação de limites.
type Schedule struct {
	ID             int32
	Service        string
	CPULimit       sql.NullFloat64
	MemLimit       sql.NullInt64
	CPUReservation sql.NullFloat64
	MemReservation sql.NullInt64
	ScheduledAt    time.Time
	Status         string
	AppliedAt      sql.NullTime
	Error          sql.NullString
	Attempts       int32
	CreatedAt      time.Time
}

// CreateSchedule insere um novo schedule e retorna o id.
func (s *Store) CreateSchedule(ctx context.Context, sch Schedule) (int32, error) {
	var id int32
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO schedules (service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'pending', now(), now()) RETURNING id`,
		sch.Service, sch.CPULimit, sch.MemLimit, sch.CPUReservation, sch.MemReservation, sch.ScheduledAt).
		Scan(&id)
	return id, err
}

// ListSchedules lista schedules, opcionalmente filtrando por status.
func (s *Store) ListSchedules(ctx context.Context, status string) ([]Schedule, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, applied_at, error, attempts, created_at
			 FROM schedules WHERE status = ? ORDER BY scheduled_at`, status)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, applied_at, error, attempts, created_at
			 FROM schedules ORDER BY scheduled_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		var sch Schedule
		if err := rows.Scan(&sch.ID, &sch.Service, &sch.CPULimit, &sch.MemLimit, &sch.CPUReservation,
			&sch.MemReservation, &sch.ScheduledAt, &sch.Status, &sch.AppliedAt, &sch.Error, &sch.Attempts, &sch.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sch)
	}
	return out, rows.Err()
}

// GetPendingSchedules retorna schedules pendentes cujo scheduled_at <= now.
func (s *Store) GetPendingSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, attempts
		 FROM schedules WHERE status = 'pending' AND scheduled_at <= now() ORDER BY scheduled_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		var sch Schedule
		if err := rows.Scan(&sch.ID, &sch.Service, &sch.CPULimit, &sch.MemLimit, &sch.CPUReservation,
			&sch.MemReservation, &sch.ScheduledAt, &sch.Status, &sch.Attempts); err != nil {
			return nil, err
		}
		out = append(out, sch)
	}
	return out, rows.Err()
}

// GetScheduleByID busca um schedule pelo id.
func (s *Store) GetScheduleByID(ctx context.Context, id int32) (*Schedule, error) {
	sch := &Schedule{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, applied_at, error, attempts, created_at
		 FROM schedules WHERE id = ?`, id).
		Scan(&sch.ID, &sch.Service, &sch.CPULimit, &sch.MemLimit, &sch.CPUReservation,
			&sch.MemReservation, &sch.ScheduledAt, &sch.Status, &sch.AppliedAt, &sch.Error, &sch.Attempts, &sch.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sch, err
}

// GetPendingScheduleForService retorna o schedule pendente mais antigo de um serviço.
func (s *Store) GetPendingScheduleForService(ctx context.Context, service string) (*Schedule, error) {
	sch := &Schedule{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, attempts
		 FROM schedules WHERE service = ? AND status = 'pending' ORDER BY scheduled_at LIMIT 1`, service).
		Scan(&sch.ID, &sch.Service, &sch.CPULimit, &sch.MemLimit, &sch.CPUReservation,
			&sch.MemReservation, &sch.ScheduledAt, &sch.Status, &sch.Attempts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sch, err
}

// UpdateScheduleStatus atualiza status/error/applied_at de um schedule.
func (s *Store) UpdateScheduleStatus(ctx context.Context, id int32, status string, errMsg string, appliedAt *time.Time) error {
	if appliedAt != nil {
		_, err := s.db.ExecContext(ctx,
			"UPDATE schedules SET status = ?, error = ?, applied_at = ?, updated_at = now() WHERE id = ?",
			status, errMsg, *appliedAt, id)
		return err
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE schedules SET status = ?, error = ?, updated_at = now() WHERE id = ?",
		status, errMsg, id)
	return err
}

// IncrementScheduleAttempts incrementa o contador de tentativas.
func (s *Store) IncrementScheduleAttempts(ctx context.Context, id int32) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE schedules SET attempts = attempts + 1, updated_at = now() WHERE id = ?", id)
	return err
}

// CancelSchedule marca um schedule pendente como cancelled. Retorna false se
// não havia schedule pendente com esse id.
func (s *Store) CancelSchedule(ctx context.Context, id int32) (bool, error) {
	var returnedID int32
	err := s.db.QueryRowContext(ctx,
		`UPDATE schedules SET status = 'cancelled', updated_at = now()
		 WHERE id = ? AND status = 'pending' RETURNING id`, id).Scan(&returnedID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetScheduleHistory retorna o histórico de schedules, opcionalmente por serviço.
func (s *Store) GetScheduleHistory(ctx context.Context, service string, limit int32) ([]Schedule, error) {
	var rows *sql.Rows
	var err error
	if service != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, applied_at, error, attempts, created_at
			 FROM schedules WHERE service = ? ORDER BY scheduled_at DESC LIMIT ?`, service, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, applied_at, error, attempts, created_at
			 FROM schedules ORDER BY scheduled_at DESC LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		var sch Schedule
		if err := rows.Scan(&sch.ID, &sch.Service, &sch.CPULimit, &sch.MemLimit, &sch.CPUReservation,
			&sch.MemReservation, &sch.ScheduledAt, &sch.Status, &sch.AppliedAt, &sch.Error, &sch.Attempts, &sch.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sch)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// change_log
// ---------------------------------------------------------------------------

// ChangeLogEntry representa uma entrada no change_log.
type ChangeLogEntry struct {
	ID                   int32
	Service              string
	Action               string
	Source               string
	ScheduleID           sql.NullInt32
	CPULimitBefore       sql.NullFloat64
	MemLimitBefore       sql.NullInt64
	CPUReservationBefore sql.NullFloat64
	MemReservationBefore sql.NullInt64
	CPULimitAfter        sql.NullFloat64
	MemLimitAfter        sql.NullInt64
	CPUReservationAfter  sql.NullFloat64
	MemReservationAfter  sql.NullInt64
	User                 sql.NullString
	Status               string
	Error                sql.NullString
	DockerResponse       sql.NullString
	CreatedAt            time.Time
}

// AddChangeLog insere uma entrada no change_log e retorna o id.
func (s *Store) AddChangeLog(ctx context.Context, e ChangeLogEntry) (int32, error) {
	var id int32
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO change_log (service, action, source, schedule_id,
			cpu_limit_before, mem_limit_before, cpu_reservation_before, mem_reservation_before,
			cpu_limit_after, mem_limit_after, cpu_reservation_after, mem_reservation_after,
			"user", status, error, docker_response, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now()) RETURNING id`,
		e.Service, e.Action, e.Source, e.ScheduleID,
		e.CPULimitBefore, e.MemLimitBefore, e.CPUReservationBefore, e.MemReservationBefore,
		e.CPULimitAfter, e.MemLimitAfter, e.CPUReservationAfter, e.MemReservationAfter,
		e.User, e.Status, e.Error, e.DockerResponse).Scan(&id)
	return id, err
}

// GetChangeLog retorna entradas do change_log, opcionalmente por serviço.
func (s *Store) GetChangeLog(ctx context.Context, service string, limit int32) ([]ChangeLogEntry, error) {
	cols := `id, service, action, source, schedule_id,
		cpu_limit_before, mem_limit_before, cpu_reservation_before, mem_reservation_before,
		cpu_limit_after, mem_limit_after, cpu_reservation_after, mem_reservation_after,
		"user", status, error, docker_response, created_at`
	var rows *sql.Rows
	var err error
	if service != "" {
		rows, err = s.db.QueryContext(ctx,
			fmt.Sprintf(`SELECT %s FROM change_log WHERE service = ? ORDER BY created_at DESC LIMIT ?`, cols),
			service, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			fmt.Sprintf(`SELECT %s FROM change_log ORDER BY created_at DESC LIMIT ?`, cols),
			limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChangeLogEntry
	for rows.Next() {
		var e ChangeLogEntry
		if err := rows.Scan(&e.ID, &e.Service, &e.Action, &e.Source, &e.ScheduleID,
			&e.CPULimitBefore, &e.MemLimitBefore, &e.CPUReservationBefore, &e.MemReservationBefore,
			&e.CPULimitAfter, &e.MemLimitAfter, &e.CPUReservationAfter, &e.MemReservationAfter,
			&e.User, &e.Status, &e.Error, &e.DockerResponse, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// storage
// ---------------------------------------------------------------------------

// StorageSummary representa um snapshot de storage_summary.
type StorageSummary struct {
	TS                 time.Time
	ImagesCount        int32
	ImagesSize         int64
	ImagesReclaimable  int64
	ContainersCount    int32
	ContainersSize     int64
	VolumesCount       int32
	VolumesSize        int64
	VolumesReclaimable int64
	VolumesOrphanCount int32
	VolumesOrphanSize  int64
}

// GetLatestStorageSummary retorna o snapshot mais recente de storage_summary.
func (s *Store) GetLatestStorageSummary(ctx context.Context) (*StorageSummary, error) {
	ss := &StorageSummary{}
	err := s.db.QueryRowContext(ctx, "SELECT * FROM storage_summary ORDER BY ts DESC LIMIT 1").
		Scan(&ss.TS, &ss.ImagesCount, &ss.ImagesSize, &ss.ImagesReclaimable,
			&ss.ContainersCount, &ss.ContainersSize,
			&ss.VolumesCount, &ss.VolumesSize, &ss.VolumesReclaimable,
			&ss.VolumesOrphanCount, &ss.VolumesOrphanSize)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ss, err
}

// StorageTrendPoint é um ponto do trend de storage.
type StorageTrendPoint struct {
	TS                 time.Time
	ImagesSize         int64
	ContainersSize     int64
	VolumesSize        int64
	VolumesReclaimable int64
	VolumesOrphanCount int32
}

// GetStorageTrend retorna o trend de storage dos últimos `days` dias.
func (s *Store) GetStorageTrend(ctx context.Context, days int) ([]StorageTrendPoint, error) {
	stmt := fmt.Sprintf(
		`SELECT ts, images_size, containers_size, volumes_size, volumes_reclaimable, volumes_orphan_count
		 FROM storage_summary WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS ORDER BY ts ASC`, days)
	rows, err := s.db.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StorageTrendPoint
	for rows.Next() {
		var p StorageTrendPoint
		if err := rows.Scan(&p.TS, &p.ImagesSize, &p.ContainersSize, &p.VolumesSize,
			&p.VolumesReclaimable, &p.VolumesOrphanCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// VolumeGrowthPoint é um ponto do growth de um volume.
type VolumeGrowthPoint struct {
	TS          time.Time
	Size        int64
	Reclaimable int64
	InUse       bool
}

// GetVolumeGrowth retorna o growth de um volume específico.
func (s *Store) GetVolumeGrowth(ctx context.Context, volumeName string, days int) ([]VolumeGrowthPoint, error) {
	stmt := fmt.Sprintf(
		`SELECT ts, size_bytes, reclaimable_bytes, in_use
		 FROM volume_metrics WHERE volume_name = ? AND ts > now()::TIMESTAMP - INTERVAL %d DAYS ORDER BY ts ASC`, days)
	rows, err := s.db.QueryContext(ctx, stmt, volumeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VolumeGrowthPoint
	for rows.Next() {
		var p VolumeGrowthPoint
		if err := rows.Scan(&p.TS, &p.Size, &p.Reclaimable, &p.InUse); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// VolumeGrowthAllPoint é um ponto do growth de todos os volumes.
type VolumeGrowthAllPoint struct {
	VolumeName string
	TS         time.Time
	Size       int64
	InUse      bool
}

// GetVolumeGrowthAll retorna o growth de todos os volumes.
func (s *Store) GetVolumeGrowthAll(ctx context.Context, days int) ([]VolumeGrowthAllPoint, error) {
	stmt := fmt.Sprintf(
		`SELECT volume_name, ts, size_bytes, in_use
		 FROM volume_metrics WHERE ts > now()::TIMESTAMP - INTERVAL %d DAYS ORDER BY ts ASC`, days)
	rows, err := s.db.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []VolumeGrowthAllPoint
	for rows.Next() {
		var p VolumeGrowthAllPoint
		if err := rows.Scan(&p.VolumeName, &p.TS, &p.Size, &p.InUse); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// api_keys (0b.4 — novo, não existe no Python)
// ---------------------------------------------------------------------------

// APIKey representa uma API key cadastrada.
type APIKey struct {
	ID         int32        `json:"id"`
	KeyHash    string       `json:"-"`
	KeyPrefix  string       `json:"prefix"`
	Name       string       `json:"name"`
	Scopes     string       `json:"scopes"` // "read", "write", "read,write"
	CreatedAt  time.Time    `json:"created_at"`
	LastUsedAt sql.NullTime `json:"last_used_at"`
	RevokedAt  sql.NullTime `json:"revoked_at"`
}

// CreateAPIKey insere uma nova API key e retorna o id.
func (s *Store) CreateAPIKey(ctx context.Context, keyHash, keyPrefix, name, scopes string) (int32, error) {
	var id int32
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO api_keys (key_hash, key_prefix, name, scopes, created_at)
		 VALUES (?, ?, ?, ?, now()) RETURNING id`,
		keyHash, keyPrefix, name, scopes).Scan(&id)
	return id, err
}

// GetAPIKeyByHash busca uma API key pelo hash (para validação de middleware).
func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	k := &APIKey{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, key_hash, key_prefix, name, scopes, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE key_hash = ? AND revoked_at IS NULL`, keyHash).
		Scan(&k.ID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Scopes, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return k, err
}

// ListAPIKeys retorna todas as API keys (incluindo revogadas), sem o hash.
func (s *Store) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, key_prefix, name, scopes, created_at, last_used_at, revoked_at
		 FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.KeyPrefix, &k.Name, &k.Scopes, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// GetAPIKeyByID busca uma API key pelo id (sem o hash, para edição).
func (s *Store) GetAPIKeyByID(ctx context.Context, id int32) (*APIKey, error) {
	k := &APIKey{}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, key_prefix, name, scopes, created_at, last_used_at, revoked_at
		 FROM api_keys WHERE id = ?`, id).
		Scan(&k.ID, &k.KeyPrefix, &k.Name, &k.Scopes, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return k, err
}

// UpdateAPIKeyLastUsed atualiza o last_used_at de uma API key.
func (s *Store) UpdateAPIKeyLastUsed(ctx context.Context, id int32) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE api_keys SET last_used_at = now() WHERE id = ?", id)
	return err
}

// RevokeAPIKey marca uma API key como revogada (soft delete).
func (s *Store) RevokeAPIKey(ctx context.Context, id int32) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE api_keys SET revoked_at = now() WHERE id = ? AND revoked_at IS NULL", id)
	return err
}

// UpdateAPIKeyName atualiza o nome de uma API key.
func (s *Store) UpdateAPIKeyName(ctx context.Context, id int32, name string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE api_keys SET name = ? WHERE id = ?", name, id)
	return err
}

// ---------------------------------------------------------------------------
// internal endpoints (para ML sidecar — sem acesso direto ao DuckDB)
// ---------------------------------------------------------------------------

// GetServiceMetricsRaw retorna série temporal bruta de um serviço.
// Reutiliza MetricRow (definido em appender.go); apenas os campos TS,
// CPUPercent, MemUsage e MemLimit são preenchidos para o ML sidecar.
func (s *Store) GetServiceMetricsRaw(ctx context.Context, service string, days int) ([]MetricRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT ts, cpu_percent, mem_usage, mem_limit FROM metrics
		 WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL `+fmt.Sprintf("%d", days)+` DAYS
		 ORDER BY ts`, service)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []MetricRow
	for rows.Next() {
		var r MetricRow
		if err := rows.Scan(&r.TS, &r.CPUPercent, &r.MemUsage, &r.MemLimit); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// GetOOMCountByService retorna contagem de OOMs de um serviço nos últimos N dias.
func (s *Store) GetOOMCountByService(ctx context.Context, service string, days int) (int32, error) {
	var count int32
	err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM oom_events
		 WHERE service = ? AND ts > now()::TIMESTAMP - INTERVAL `+fmt.Sprintf("%d", days)+` DAYS`, service).Scan(&count)
	return count, err
}

// GetOOMCountSince retorna contagem de OOMs de um serviço desde um timestamp.
func (s *Store) GetOOMCountSince(ctx context.Context, service string, since time.Time) (int32, error) {
	var count int32
	err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM oom_events WHERE service = ? AND ts > ?`, service, since).Scan(&count)
	return count, err
}

// GetServicesWithMetrics retorna lista de serviços com métricas nos últimos N dias.
func (s *Store) GetServicesWithMetrics(ctx context.Context, days int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT service FROM metrics
		 WHERE ts > now()::TIMESTAMP - INTERVAL `+fmt.Sprintf("%d", days)+` DAYS`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var services []string
	for rows.Next() {
		var svc string
		if err := rows.Scan(&svc); err != nil {
			return nil, err
		}
		services = append(services, svc)
	}
	return services, nil
}

// GetActiveServices retorna serviços com métricas nos últimos N minutos.
// Usado pelo ML sidecar para filtrar apenas serviços ativos antes de buscar
// métricas, reduzindo chamadas HTTP desnecessárias a serviços parados.
func (s *Store) GetActiveServices(ctx context.Context, minutes int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT service FROM metrics
		 WHERE ts > now()::TIMESTAMP - INTERVAL '`+fmt.Sprintf("%d", minutes)+`' MINUTE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var services []string
	for rows.Next() {
		var svc string
		if err := rows.Scan(&svc); err != nil {
			return nil, err
		}
		services = append(services, svc)
	}
	return services, rows.Err()
}

// GetServiceConfigRow retorna a config de recursos de um serviço (ou nil se não existir).
func (s *Store) GetServiceConfigRow(ctx context.Context, service string) (*ServiceConfig, error) {
	var c ServiceConfig
	var cpuLimit, cpuRes sql.NullFloat64
	var memLimit, memRes sql.NullInt64
	var tmpl sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, template, updated_at
		 FROM service_configs WHERE service = ?`, service).Scan(
		&c.Service, &cpuLimit, &memLimit, &cpuRes, &memRes, &tmpl, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.CPULimit = cpuLimit
	c.MemLimit = memLimit
	c.CPUReservation = cpuRes
	c.MemReservation = memRes
	c.Template = tmpl
	return &c, nil
}

// GetVolumeMetricsRaw retorna métricas brutas de todos os volumes.
// Reutiliza VolumeMetricRow (definido em appender.go); apenas os campos
// VolumeName, TS e SizeBytes são preenchidos para o ML sidecar.
func (s *Store) GetVolumeMetricsRaw(ctx context.Context, days int) ([]VolumeMetricRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT volume_name, ts, size_bytes FROM volume_metrics
		 WHERE ts > now()::TIMESTAMP - INTERVAL `+fmt.Sprintf("%d", days)+` DAYS
		 ORDER BY volume_name, ts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []VolumeMetricRow
	for rows.Next() {
		var r VolumeMetricRow
		if err := rows.Scan(&r.VolumeName, &r.TS, &r.SizeBytes); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}

func round1(f float64) float64 {
	return float64(int64(f*10+0.5)) / 10
}

// ---------------------------------------------------------------------------
// rollback_watches (Right-Sizing Studio R5 — watcher de rollback automático)
// ---------------------------------------------------------------------------

// RollbackWatch representa uma entrada de rollback_watches.
type RollbackWatch struct {
	ID                   int32
	ChangeLogID          int32
	Service              string
	CPULimitBefore       sql.NullFloat64
	MemLimitBefore       sql.NullInt64
	CPUReservationBefore sql.NullFloat64
	MemReservationBefore sql.NullInt64
	CPULimitAfter        sql.NullFloat64
	MemLimitAfter        sql.NullInt64
	CPUReservationAfter  sql.NullFloat64
	MemReservationAfter  sql.NullInt64
	Strategy             string
	ObservationWindow    int
	Criteria             string // JSON
	Status               string // monitoring | optimized | rolled_back | expired | cancelled
	TriggeredCriteria    sql.NullString
	StartedAt            time.Time
	ExpiresAt            time.Time
	RolledBackAt         sql.NullTime
	CreatedAt            time.Time
}

const rollbackWatchCols = `id, change_log_id, service,
	cpu_limit_before, mem_limit_before, cpu_reservation_before, mem_reservation_before,
	cpu_limit_after, mem_limit_after, cpu_reservation_after, mem_reservation_after,
	strategy, observation_window, criteria, status, triggered_criteria,
	started_at, expires_at, rolled_back_at, created_at`

func scanRollbackWatch(rows *sql.Rows) (RollbackWatch, error) {
	var w RollbackWatch
	err := rows.Scan(&w.ID, &w.ChangeLogID, &w.Service,
		&w.CPULimitBefore, &w.MemLimitBefore, &w.CPUReservationBefore, &w.MemReservationBefore,
		&w.CPULimitAfter, &w.MemLimitAfter, &w.CPUReservationAfter, &w.MemReservationAfter,
		&w.Strategy, &w.ObservationWindow, &w.Criteria, &w.Status, &w.TriggeredCriteria,
		&w.StartedAt, &w.ExpiresAt, &w.RolledBackAt, &w.CreatedAt)
	return w, err
}

// CreateRollbackWatch insere um novo watch e retorna o id.
// expires_at é calculado como started_at + observation_window horas.
func (s *Store) CreateRollbackWatch(ctx context.Context, w RollbackWatch) (int32, error) {
	var id int32
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO rollback_watches (change_log_id, service,
			cpu_limit_before, mem_limit_before, cpu_reservation_before, mem_reservation_before,
			cpu_limit_after, mem_limit_after, cpu_reservation_after, mem_reservation_after,
			strategy, observation_window, criteria, status, started_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'monitoring', now(),
			cast(now() as TIMESTAMP) + INTERVAL '`+fmt.Sprintf("%d", w.ObservationWindow)+`' HOUR)
		 RETURNING id`,
		w.ChangeLogID, w.Service,
		w.CPULimitBefore, w.MemLimitBefore, w.CPUReservationBefore, w.MemReservationBefore,
		w.CPULimitAfter, w.MemLimitAfter, w.CPUReservationAfter, w.MemReservationAfter,
		w.Strategy, w.ObservationWindow, w.Criteria).Scan(&id)
	return id, err
}

// GetActiveRollbackWatches retorna watches com status='monitoring' e não expirados.
func (s *Store) GetActiveRollbackWatches(ctx context.Context) ([]RollbackWatch, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+rollbackWatchCols+` FROM rollback_watches
		 WHERE status = 'monitoring' AND expires_at > now()
		 ORDER BY expires_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RollbackWatch
	for rows.Next() {
		w, err := scanRollbackWatch(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, rows.Err()
}

// GetRollbackWatchByID busca um watch pelo id.
func (s *Store) GetRollbackWatchByID(ctx context.Context, id int32) (*RollbackWatch, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+rollbackWatchCols+` FROM rollback_watches WHERE id = ?`, id)
	var w RollbackWatch
	if err := row.Scan(&w.ID, &w.ChangeLogID, &w.Service,
		&w.CPULimitBefore, &w.MemLimitBefore, &w.CPUReservationBefore, &w.MemReservationBefore,
		&w.CPULimitAfter, &w.MemLimitAfter, &w.CPUReservationAfter, &w.MemReservationAfter,
		&w.Strategy, &w.ObservationWindow, &w.Criteria, &w.Status, &w.TriggeredCriteria,
		&w.StartedAt, &w.ExpiresAt, &w.RolledBackAt, &w.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// ListRollbackWatches lista watches com filtros opcionais (status, service, limit).
func (s *Store) ListRollbackWatches(ctx context.Context, status, service string, limit int32) ([]RollbackWatch, error) {
	q := `SELECT ` + rollbackWatchCols + ` FROM rollback_watches WHERE 1=1`
	args := []any{}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	if service != "" {
		q += ` AND service = ?`
		args = append(args, service)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RollbackWatch
	for rows.Next() {
		w, err := scanRollbackWatch(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, rows.Err()
}

// UpdateRollbackWatchStatus atualiza status/triggered_criteria/rolled_back_at.
func (s *Store) UpdateRollbackWatchStatus(ctx context.Context, id int32,
	status, reason string, rolledBackAt *time.Time) error {
	if rolledBackAt != nil {
		_, err := s.db.ExecContext(ctx,
			`UPDATE rollback_watches SET status = ?, triggered_criteria = ?,
			 rolled_back_at = ?, updated_at = now() WHERE id = ?`,
			status, reason, *rolledBackAt, id)
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE rollback_watches SET status = ?, triggered_criteria = ?,
		 updated_at = now() WHERE id = ?`, status, reason, id)
	return err
}

// GetMetricsSince retorna métricas de um serviço desde um timestamp.
// Usado pelo watcher para avaliar critérios de throttle e mem_pressure.
func (s *Store) GetMetricsSince(ctx context.Context, service string, since time.Time) ([]MetricRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT ts, cpu_percent, mem_usage, mem_limit, mem_percent,
				cpu_throttled_periods, cpu_throttled_time
		 FROM metrics WHERE service = ? AND ts > ? ORDER BY ts`, service, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []MetricRow
	for rows.Next() {
		var r MetricRow
		var memPercent sql.NullFloat64
		var throttledPeriods, throttledTime sql.NullInt64
		if err := rows.Scan(&r.TS, &r.CPUPercent, &r.MemUsage, &r.MemLimit,
			&memPercent, &throttledPeriods, &throttledTime); err != nil {
			return nil, err
		}
		r.MemPercent = memPercent.Float64
		r.CPUThrottledPeriods = throttledPeriods.Int64
		r.CPUThrottledTime = throttledTime.Int64
		result = append(result, r)
	}
	return result, rows.Err()
}
