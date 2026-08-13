package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Limpar env vars relevantes
	for _, k := range []string{"RESMA_DB_PATH", "RESMA_HTTP_ADDR", "RESMA_JWT_SECRET", "RESMA_ENV"} {
		_ = os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.DBPath != "data/resma.duckdb" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "data/resma.duckdb")
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want %q", cfg.Env, "dev")
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost = %d, want 12", cfg.BcryptCost)
	}
}

func TestLoadCORSOrigins(t *testing.T) {
	_ = os.Setenv("RESMA_CORS_ORIGINS", "http://localhost:5173,http://localhost:8080,https://resma.example.com")
	defer func() { _ = os.Unsetenv("RESMA_CORS_ORIGINS") }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(cfg.CORSOrigins) != 3 {
		t.Fatalf("CORSOrigins len = %d, want 3", len(cfg.CORSOrigins))
	}
	if cfg.CORSOrigins[0] != "http://localhost:5173" {
		t.Errorf("CORSOrigins[0] = %q, want %q", cfg.CORSOrigins[0], "http://localhost:5173")
	}
}

func TestJWTSecretFile(t *testing.T) {
	// Criar arquivo temporário com secret
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "jwt_secret")
	secretValue := "my-secret-from-file"
	if err := os.WriteFile(secretFile, []byte(secretValue), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_ = os.Unsetenv("RESMA_JWT_SECRET")
	_ = os.Setenv("RESMA_JWT_SECRET_FILE", secretFile)
	defer func() { _ = os.Unsetenv("RESMA_JWT_SECRET_FILE") }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.JWTSecret != secretValue {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, secretValue)
	}
}

func TestProductionRejectsDefaultSecret(t *testing.T) {
	_ = os.Setenv("RESMA_ENV", "production")
	_ = os.Setenv("RESMA_JWT_SECRET", "dev-secret-change-me")
	defer func() {
		_ = os.Unsetenv("RESMA_ENV")
		_ = os.Unsetenv("RESMA_JWT_SECRET")
	}()

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail in production with default secret")
	}
}

func TestProductionRejectsEmptySecret(t *testing.T) {
	_ = os.Setenv("RESMA_ENV", "production")
	_ = os.Unsetenv("RESMA_JWT_SECRET")
	_ = os.Unsetenv("RESMA_JWT_SECRET_FILE")
	defer func() { _ = os.Unsetenv("RESMA_ENV") }()

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail in production with empty secret")
	}
}

func TestProductionAcceptsRealSecret(t *testing.T) {
	_ = os.Setenv("RESMA_ENV", "production")
	_ = os.Setenv("RESMA_JWT_SECRET", "a-real-production-secret-32-bytes-long!!")
	defer func() {
		_ = os.Unsetenv("RESMA_ENV")
		_ = os.Unsetenv("RESMA_JWT_SECRET")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.JWTSecret != "a-real-production-secret-32-bytes-long!!" {
		t.Errorf("JWTSecret = %q, want production secret", cfg.JWTSecret)
	}
}
