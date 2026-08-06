// Package db — schema init.
//
// Portado de backend/core/db.py (_init_schema). Mantém paridade total com o
// arquivo resma.duckdb existente: todas as tabelas usam CREATE TABLE IF NOT
// EXISTS, e colunas novas (0b.2) são adicionadas via ALTER TABLE ADD COLUMN
// IF NOT EXISTS para não questrar arquivos existentes.
//
// Novidades 0b.2 (vs Python):
//   - metrics.mem_working_set     — working set = usage - inactive_file/cache
//   - metrics.cpu_throttled_periods — Docker stats throttling_data
//   - metrics.cpu_throttled_time    — Docker stats throttling_data
package db

import (
	"context"
	"fmt"
)

// initSchema cria todas as tabelas/sequências e aplica migrações in-place.
// Deve ser chamado uma única vez logo após New.
func (s *Store) initSchema(ctx context.Context) error {
	stmts := []string{
		// --- metrics ---
		`CREATE TABLE IF NOT EXISTS metrics (
			ts                TIMESTAMP,
			service           VARCHAR,
			container_id      VARCHAR,
			cpu_percent       DOUBLE,
			cpu_usage         BIGINT,
			cpu_system        BIGINT,
			mem_usage         BIGINT,
			mem_limit         BIGINT,
			mem_percent       DOUBLE,
			net_rx            BIGINT,
			net_tx            BIGINT,
			block_read        BIGINT,
			block_write       BIGINT
		)`,
		// --- oom_events ---
		`CREATE TABLE IF NOT EXISTS oom_events (
			ts           TIMESTAMP,
			service      VARCHAR,
			container_id VARCHAR,
			exit_code    INTEGER
		)`,
		// --- service_configs ---
		`CREATE TABLE IF NOT EXISTS service_configs (
			service         VARCHAR PRIMARY KEY,
			cpu_limit       DOUBLE,
			mem_limit       BIGINT,
			cpu_reservation DOUBLE,
			mem_reservation BIGINT,
			template        VARCHAR,
			updated_at      TIMESTAMP
		)`,
		// --- users ---
		`CREATE SEQUENCE IF NOT EXISTS users_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS users (
			id           INTEGER PRIMARY KEY DEFAULT nextval('users_id_seq'),
			username     VARCHAR UNIQUE NOT NULL,
			password_hash VARCHAR NOT NULL,
			role         VARCHAR DEFAULT 'admin',
			name         VARCHAR,
			created_at   TIMESTAMP DEFAULT now(),
			updated_at   TIMESTAMP DEFAULT now()
		)`,
		// --- refresh_tokens ---
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			token      VARCHAR PRIMARY KEY,
			user_id    INTEGER REFERENCES users(id),
			expires_at TIMESTAMP,
			revoked    BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT now()
		)`,
		// --- app_settings ---
		`CREATE TABLE IF NOT EXISTS app_settings (
			key   VARCHAR PRIMARY KEY,
			value VARCHAR
		)`,
		// --- templates ---
		`CREATE TABLE IF NOT EXISTS templates (
			id          INTEGER PRIMARY KEY DEFAULT nextval('users_id_seq'),
			name        VARCHAR UNIQUE NOT NULL,
			description VARCHAR DEFAULT '',
			yaml_content TEXT NOT NULL,
			stacks      VARCHAR DEFAULT '[]',
			created_at  TIMESTAMP DEFAULT now(),
			updated_at  TIMESTAMP DEFAULT now()
		)`,
		// --- service_registry ---
		`CREATE TABLE IF NOT EXISTS service_registry (
			service    VARCHAR PRIMARY KEY,
			status     VARCHAR DEFAULT 'active',
			last_seen  TIMESTAMP,
			updated_at TIMESTAMP DEFAULT now()
		)`,
		// --- nodes ---
		`CREATE TABLE IF NOT EXISTS nodes (
			node_id       VARCHAR PRIMARY KEY,
			hostname      VARCHAR,
			role          VARCHAR,
			availability  VARCHAR,
			status        VARCHAR,
			address       VARCHAR,
			cpu_total     DOUBLE,
			mem_total     BIGINT,
			os            VARCHAR,
			architecture  VARCHAR,
			engine_version VARCHAR,
			is_leader     BOOLEAN,
			reachability  VARCHAR,
			labels        VARCHAR,
			tasks_running INTEGER,
			updated_at    TIMESTAMP DEFAULT now()
		)`,
		// --- node_metrics ---
		`CREATE TABLE IF NOT EXISTS node_metrics (
			ts            TIMESTAMP,
			node_id       VARCHAR,
			hostname      VARCHAR,
			role          VARCHAR,
			availability  VARCHAR,
			status        VARCHAR,
			cpu_total     DOUBLE,
			mem_total     BIGINT,
			tasks_running INTEGER
		)`,
		// --- cluster ---
		`CREATE TABLE IF NOT EXISTS cluster (
			id             VARCHAR PRIMARY KEY,
			nodes_total    INTEGER,
			managers_total INTEGER,
			workers_total  INTEGER,
			nodes_ready    INTEGER,
			nodes_down     INTEGER,
			quorum_healthy BOOLEAN,
			self_node_id   VARCHAR,
			warnings       VARCHAR,
			updated_at     TIMESTAMP DEFAULT now()
		)`,
		// --- container_node_map ---
		`CREATE TABLE IF NOT EXISTS container_node_map (
			container_id VARCHAR PRIMARY KEY,
			node_id      VARCHAR,
			service      VARCHAR,
			updated_at   TIMESTAMP DEFAULT now()
		)`,
		// --- schedules ---
		`CREATE SEQUENCE IF NOT EXISTS schedules_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS schedules (
			id              INTEGER PRIMARY KEY DEFAULT nextval('schedules_id_seq'),
			service         VARCHAR NOT NULL,
			cpu_limit       DOUBLE,
			mem_limit       BIGINT,
			cpu_reservation DOUBLE,
			mem_reservation BIGINT,
			scheduled_at    TIMESTAMP NOT NULL,
			status          VARCHAR DEFAULT 'pending',
			applied_at      TIMESTAMP,
			error           VARCHAR,
			attempts        INTEGER DEFAULT 0,
			created_at      TIMESTAMP DEFAULT now(),
			updated_at      TIMESTAMP DEFAULT now()
		)`,
		// --- change_log ---
		`CREATE SEQUENCE IF NOT EXISTS change_log_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS change_log (
			id                      INTEGER PRIMARY KEY DEFAULT nextval('change_log_id_seq'),
			service                 VARCHAR NOT NULL,
			action                  VARCHAR NOT NULL,
			source                  VARCHAR DEFAULT 'manual',
			schedule_id             INTEGER,
			cpu_limit_before        DOUBLE,
			mem_limit_before        BIGINT,
			cpu_reservation_before  DOUBLE,
			mem_reservation_before  BIGINT,
			cpu_limit_after         DOUBLE,
			mem_limit_after         BIGINT,
			cpu_reservation_after   DOUBLE,
			mem_reservation_after   BIGINT,
			"user"                  VARCHAR,
			status                  VARCHAR DEFAULT 'completed',
			error                   VARCHAR,
			docker_response         TEXT,
			created_at              TIMESTAMP DEFAULT now()
		)`,
		// --- volume_metrics ---
		`CREATE TABLE IF NOT EXISTS volume_metrics (
			ts               TIMESTAMP,
			volume_name      VARCHAR,
			size_bytes       BIGINT,
			reclaimable_bytes BIGINT,
			in_use           BOOLEAN
		)`,
		// --- storage_summary ---
		`CREATE TABLE IF NOT EXISTS storage_summary (
			ts                   TIMESTAMP,
			images_count         INTEGER,
			images_size          BIGINT,
			images_reclaimable   BIGINT,
			containers_count     INTEGER,
			containers_size      BIGINT,
			volumes_count        INTEGER,
			volumes_size         BIGINT,
			volumes_reclaimable  BIGINT,
			volumes_orphan_count INTEGER,
			volumes_orphan_size  BIGINT
		)`,
		// --- api_keys (0b.4 — novo, não existe no Python) ---
		`CREATE SEQUENCE IF NOT EXISTS api_keys_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id           INTEGER PRIMARY KEY DEFAULT nextval('api_keys_id_seq'),
			key_hash     VARCHAR UNIQUE NOT NULL,
			key_prefix   VARCHAR NOT NULL,
			name         VARCHAR NOT NULL,
			scopes       VARCHAR DEFAULT 'read',
			created_at   TIMESTAMP DEFAULT now(),
			last_used_at TIMESTAMP,
			revoked_at   TIMESTAMP
		)`,
		// --- agents (Fase 7 — multi-node: 1 row por agent ativo) ---
		`CREATE TABLE IF NOT EXISTS agents (
			node_id          VARCHAR PRIMARY KEY,
			hostname         VARCHAR,
			version          VARCHAR,
			containers_count INTEGER DEFAULT 0,
			last_heartbeat   TIMESTAMP,
			status           VARCHAR DEFAULT 'active',
			first_seen       TIMESTAMP DEFAULT now(),
			updated_at       TIMESTAMP DEFAULT now()
		)`,
		// --- tasks (Fase 7 — Swarm task lifecycle, populada pelo manager via TaskList) ---
		`CREATE TABLE IF NOT EXISTS tasks (
			task_id        VARCHAR PRIMARY KEY,
			service        VARCHAR,
			node_id        VARCHAR,
			slot           INTEGER,
			status         VARCHAR,
			desired_state  VARCHAR,
			created_at     TIMESTAMP DEFAULT now(),
			updated_at     TIMESTAMP DEFAULT now()
		)`,
		// --- task_history (Fase 7 — append-only log de mudanças de status de tasks) ---
		`CREATE SEQUENCE IF NOT EXISTS task_history_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS task_history (
			id         INTEGER PRIMARY KEY DEFAULT nextval('task_history_id_seq'),
			task_id    VARCHAR NOT NULL,
			service    VARCHAR,
			node_id    VARCHAR,
			slot       INTEGER,
			status     VARCHAR,
			ts         TIMESTAMP DEFAULT now()
		)`,
		// --- rollback_watches (Right-Sizing Studio R5 — watcher de rollback automático pós-apply) ---
		`CREATE SEQUENCE IF NOT EXISTS rollback_watches_id_seq START 1`,
		`CREATE TABLE IF NOT EXISTS rollback_watches (
			id                       INTEGER PRIMARY KEY DEFAULT nextval('rollback_watches_id_seq'),
			change_log_id            INTEGER NOT NULL,
			service                  VARCHAR NOT NULL,
			cpu_limit_before         DOUBLE,
			mem_limit_before         BIGINT,
			cpu_reservation_before   DOUBLE,
			mem_reservation_before   BIGINT,
			cpu_limit_after          DOUBLE,
			mem_limit_after          BIGINT,
			cpu_reservation_after    DOUBLE,
			mem_reservation_after    BIGINT,
			strategy                 VARCHAR DEFAULT 'deferred',
			observation_window       INTEGER DEFAULT 24,
			criteria                 VARCHAR DEFAULT '{}',
			status                   VARCHAR DEFAULT 'monitoring',
			triggered_criteria       VARCHAR,
			started_at               TIMESTAMP DEFAULT now(),
			expires_at               TIMESTAMP NOT NULL,
			rolled_back_at           TIMESTAMP,
			created_at               TIMESTAMP DEFAULT now(),
			updated_at               TIMESTAMP DEFAULT now()
		)`,
	}

	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("schema init: %w (stmt: %s)", err, truncate(stmt, 80))
		}
	}

	// --- migrações in-place (colunas novas vs Python) ---
	migrations := []string{
		// Python já tinha esta migração para change_log
		`ALTER TABLE change_log ADD COLUMN IF NOT EXISTS docker_response TEXT`,
		// 0b.2: memory working set + CPU throttling
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS mem_working_set BIGINT`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_throttled_periods BIGINT`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS cpu_throttled_time BIGINT`,
		// Fase 7: multi-node — node_id/task_id/slot em metrics + node_id em oom_events
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS node_id VARCHAR`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS task_id VARCHAR`,
		`ALTER TABLE metrics ADD COLUMN IF NOT EXISTS slot INTEGER`,
		`ALTER TABLE oom_events ADD COLUMN IF NOT EXISTS node_id VARCHAR`,
		// Fase 8: RBAC — onboarding cria 'owner' em vez de 'admin'.
		// Se existe exatamente 1 usuário com role='admin', promover a 'owner'.
		`UPDATE users SET role = 'owner' WHERE role = 'admin' AND (SELECT count(*) FROM users) = 1`,
		// Fase 8: campo name (opcional) para display name
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR`,
	}
	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			return fmt.Errorf("schema migration: %w (stmt: %s)", err, truncate(m, 80))
		}
	}

	if err := s.seedTemplates(ctx); err != nil {
		return fmt.Errorf("seed templates: %w", err)
	}
	return nil
}

// seedTemplates insere o template "default" se a tabela estiver vazia.
// Portado de db.py _seed_templates.
func (s *Store) seedTemplates(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM templates").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	const defaultYAML = "limits:\n" +
		"  cpus: '0.50'\n" +
		"  memory: 512M\n" +
		"reservations:\n" +
		"  cpus: '0.25'\n" +
		"  memory: 256M\n" +
		"mem_margin: 1.5\n" +
		"cpu_margin: 1.5\n" +
		"reservation_ratio: 0.75\n" +
		"leak_tolerance: 1.0\n"
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO templates (name, description, yaml_content, stacks, created_at, updated_at)
		 VALUES (?, ?, ?, '[]', now(), now())`,
		"default", "Template padrão — sem stack específica", defaultYAML,
	)
	return err
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
