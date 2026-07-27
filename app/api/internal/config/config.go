// Package config carrega configuração do RESMA via variáveis de ambiente.
//
// Mantém paridade com backend/core/config.py do RESMA Python. Todos os valores
// possuem defaults seguros para desenvolvimento local; em produção devem ser
// sobrepostos via env (docker-compose.yml / Swarm).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the RESMA API.
type Config struct {
	// Database
	DBPath string // RESMA_DB_PATH — caminho do arquivo resma.duckdb

	// Collector
	CollectInterval  time.Duration // RESMA_COLLECT_INTERVAL (segundos)
	RetentionDays    int           // RESMA_RETENTION_DAYS
	OutlierThreshold float64       // RESMA_OUTLIER_THRESHOLD

	// Leak detection
	LeakR2Threshold      float64 // RESMA_LEAK_R2_THRESHOLD
	LeakDailyMBThreshold float64 // RESMA_LEAK_DAILY_MB_THRESHOLD
	AnalysisWindowDays   int     // RESMA_ANALYSIS_WINDOW_DAYS

	// Auth / JWT
	JWTSecret      string        // RESMA_JWT_SECRET (ou RESMA_JWT_SECRET_FILE)
	JWTAccessTTL   time.Duration // RESMA_JWT_ACCESS_TTL_MINUTES
	JWTRefreshTTL  time.Duration // RESMA_JWT_REFRESH_TTL_DAYS
	BcryptCost     int           // RESMA_BCRYPT_COST
	LoginRateLimit int           // RESMA_LOGIN_RATE_LIMIT (tentativas/min)

	// ML sidecar
	MLURL     string // RESMA_ML_URL — ex: http://resma-ml:8081
	MLEnabled bool   // RESMA_ML_ENABLED

	// Docker
	ExcludedImages []string // RESMA_EXCLUDED_IMAGES (csv)

	// Coleta auxiliar
	ClusterInterval time.Duration // RESMA_CLUSTER_INTERVAL (segundos)
	StorageInterval time.Duration // RESMA_STORAGE_INTERVAL (segundos)

	// Stale-marking (Fase 8)
	StaleServiceDays int // RESMA_STALE_SERVICE_DAYS — services/nodes sem heartbeat por N dias viram 'stale'

	// Server
	HTTPAddr string // RESMA_HTTP_ADDR — default :8080

	// Security (Fase 2)
	Env                  string   // RESMA_ENV — "production" | "dev" (default)
	CORSOrigins          []string // RESMA_CORS_ORIGINS (csv) — default dev: localhost:5173,localhost:8080
	DefaultAdminPassword string   // RESMA_DEFAULT_ADMIN_PASSWORD — se vazio, gerar aleatório no startup
	APIKeyDefaultScopes  string   // RESMA_API_KEY_DEFAULT_SCOPES — default "read"
	APIKeyRateLimit      int      // RESMA_API_KEY_RATE_LIMIT (req/min) — default 100

	// Multi-node Agent (Fase 7)
	AgentToken            string        // RESMA_AGENT_TOKEN (ou RESMA_AGENT_TOKEN_FILE) — bearer compartilhado
	AgentTaskPollInterval time.Duration // RESMA_AGENT_TASK_POLL_INTERVAL (segundos) — default 15
}

// Load lê as variáveis de ambiente e devolve uma Config pronta para uso.
// Falha apenas se um valor numérico obrigatório estiver malformado ou se
// a validação de segurança (Fase 2) rejeitar a config em produção.
func Load() (*Config, error) {
	cfg := &Config{
		DBPath:                getenv("RESMA_DB_PATH", "data/resma.duckdb"),
		CollectInterval:       getDurationSecs("RESMA_COLLECT_INTERVAL", 1),
		RetentionDays:         getInt("RESMA_RETENTION_DAYS", 30),
		OutlierThreshold:      getFloat("RESMA_OUTLIER_THRESHOLD", 3.0),
		LeakR2Threshold:       getFloat("RESMA_LEAK_R2_THRESHOLD", 0.7),
		LeakDailyMBThreshold:  getFloat("RESMA_LEAK_DAILY_MB_THRESHOLD", 10.0),
		AnalysisWindowDays:    getInt("RESMA_ANALYSIS_WINDOW_DAYS", 7),
		JWTSecret:             getenvOrFile("RESMA_JWT_SECRET", "RESMA_JWT_SECRET_FILE", ""),
		JWTAccessTTL:          getDurationMins("RESMA_JWT_ACCESS_TTL_MINUTES", 15),
		JWTRefreshTTL:         getDurationDays("RESMA_JWT_REFRESH_TTL_DAYS", 7),
		BcryptCost:            getInt("RESMA_BCRYPT_COST", 12),
		LoginRateLimit:        getInt("RESMA_LOGIN_RATE_LIMIT", 5),
		MLURL:                 getenv("RESMA_ML_URL", "http://localhost:8081"),
		MLEnabled:             getBool("RESMA_ML_ENABLED", true),
		ExcludedImages:        getCSV("RESMA_EXCLUDED_IMAGES", []string{"resma:latest"}),
		ClusterInterval:       getDurationSecs("RESMA_CLUSTER_INTERVAL", 60),
		StorageInterval:       getDurationSecs("RESMA_STORAGE_INTERVAL", 300),
		StaleServiceDays:      getInt("RESMA_STALE_SERVICE_DAYS", 7),
		HTTPAddr:              getenv("RESMA_HTTP_ADDR", ":8080"),
		Env:                   getenv("RESMA_ENV", "dev"),
		CORSOrigins:           getCSV("RESMA_CORS_ORIGINS", []string{"http://localhost:5173", "http://localhost:8080"}),
		DefaultAdminPassword:  getenv("RESMA_DEFAULT_ADMIN_PASSWORD", ""),
		APIKeyDefaultScopes:   getenv("RESMA_API_KEY_DEFAULT_SCOPES", "read"),
		APIKeyRateLimit:       getInt("RESMA_API_KEY_RATE_LIMIT", 100),
		AgentToken:            getenvOrFile("RESMA_AGENT_TOKEN", "RESMA_AGENT_TOKEN_FILE", ""),
		AgentTaskPollInterval: getDurationSecs("RESMA_AGENT_TASK_POLL_INTERVAL", 15),
	}

	// Validação de JWT secret em produção (Fase 2.4)
	if cfg.Env == "production" {
		if cfg.JWTSecret == "" || cfg.JWTSecret == "dev-secret-change-me" {
			return nil, fmt.Errorf(
				"RESMA_JWT_SECRET inválido para produção (vazio ou default). " +
					"Gere um secret com: openssl rand -base64 32",
			)
		}
	}

	if cfg.JWTSecret == "" {
		fmt.Fprintln(os.Stderr, "[config] AVISO: RESMA_JWT_SECRET vazio — usar apenas em dev")
	}
	return cfg, nil
}

// --- helpers ---

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvOrFile lê uma env var, ou se vazio, lê o conteúdo de um arquivo
// cujo path está em outra env var (sufixo _FILE). Suporte para Docker Swarm secrets.
func getenvOrFile(key, fileKey, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if filePath := os.Getenv(fileKey); filePath != "" {
		if data, err := os.ReadFile(filePath); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getCSV(key string, def []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if t := strings.TrimSpace(p); t != "" {
				out = append(out, t)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return def
}

func getDurationSecs(key string, def int) time.Duration {
	return time.Duration(getInt(key, def)) * time.Second
}

func getDurationMins(key string, def int) time.Duration {
	return time.Duration(getInt(key, def)) * time.Minute
}

func getDurationDays(key string, def int) time.Duration {
	return time.Duration(getInt(key, def)) * 24 * time.Hour
}
