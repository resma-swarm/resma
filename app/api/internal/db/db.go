// Package db encapsula o acesso ao DuckDB via database/sql + go-duckdb.
//
// O arquivo resma.duckdb é compartilhado entre a API Go e o sidecar Python ML
// (mesma engine C embarcada — sem migração de schema). A inicialização do
// schema (CREATE TABLE IF NOT EXISTS ...) será portada na tarefa 0b.2.
package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/marcboeker/go-duckdb"
)

// Store wraps a *sql.DB backed by DuckDB.
type Store struct {
	db *sql.DB
}

// New abre (ou cria) o arquivo DuckDB em path, inicializa o schema e valida
// a conexão com Ping. O pool é configurado com no máximo uma conexão escritora
// (DuckDB é single-writer) e múltiplas leitoras concorrentes. O schema init é
// idempotente (CREATE TABLE IF NOT EXISTS) e aplica migrações in-place via
// ALTER TABLE ADD COLUMN IF NOT EXISTS — arquivos .duckdb existentes abrem
// sem necessidade de dump/restore.
func New(ctx context.Context, path string) (*Store, error) {
	dsn := path
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("duckdb open %q: %w", dsn, err)
	}

	// DuckDB suporta leitura concorrente mas escrita single-threaded.
	// Limitamos conexões para evitar contenção no arquivo .duckdb.
	db.SetMaxOpenConns(8)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("duckdb ping %q: %w", dsn, err)
	}

	s := &Store{db: db}
	if err := s.initSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// DB expõe o *sql.DB para uso pelos handlers/collector. Camadas superiores
// devem preferir os métodos de Store quando existirem.
func (s *Store) DB() *sql.DB { return s.db }

// QueryContext é um atalho para s.db.QueryContext.
func (s *Store) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

// QueryRowContext é um atalho para s.db.QueryRowContext.
func (s *Store) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

// Close fecha o pool e libera o arquivo DuckDB.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Health executa um SELECT 1 para probes de liveness/readiness.
func (s *Store) Health(ctx context.Context) error {
	var n int
	return s.db.QueryRowContext(ctx, "SELECT 1").Scan(&n)
}

// garante que o driver duckdb seja registrado no binário.
var _ = duckdb.Driver{}
