// Package internal — carrega configuração do RESMA Agent via variáveis de ambiente.
//
// O agent roda como `mode: global` no Swarm (1 por node). Ele coleta stats
// locais via Docker socket e envia via HTTP POST para o RESMA API no manager.
// Configuração 100% via env vars — sem arquivo de config.
package internal

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all runtime configuration for the RESMA Agent.
type Config struct {
	// Conexão com o server
	APIURL string // RESMA_API_URL — ex: http://api:8080
	Token  string // RESMA_AGENT_TOKEN — bearer token compartilhado

	// Identificação do node
	NodeID       string // RESMA_NODE_ID — template var {{.Node.ID}} do Swarm
	NodeHostname string // RESMA_NODE_HOSTNAME — template var {{.Node.Hostname}}

	// Coleta
	CollectInterval time.Duration // RESMA_COLLECT_INTERVAL (ex: 15s)
	ExcludedImages  []string      // RESMA_EXCLUDED_IMAGES (csv)

	// Health/Info server (porta local do agent)
	HTTPAddr string // RESMA_AGENT_HTTP_ADDR — default :8082

	// Debug
	Debug bool // RESMA_AGENT_DEBUG
}

// Load lê as variáveis de ambiente e devolve uma Config pronta para uso.
// Falha se RESMA_AGENT_TOKEN ou RESMA_API_URL estiverem vazios (obrigatórios).
func Load() (*Config, error) {
	cfg := &Config{
		APIURL:          getenv("RESMA_API_URL", "http://api:8080"),
		Token:           getenvOrFile("RESMA_AGENT_TOKEN", "RESMA_AGENT_TOKEN_FILE", ""),
		NodeID:          getenv("RESMA_NODE_ID", ""),
		NodeHostname:    getenv("RESMA_NODE_HOSTNAME", ""),
		CollectInterval: getDuration("RESMA_COLLECT_INTERVAL", 15*time.Second),
		ExcludedImages:  getCSV("RESMA_EXCLUDED_IMAGES", []string{"resma-agent:latest"}),
		HTTPAddr:        getenv("RESMA_AGENT_HTTP_ADDR", ":8082"),
		Debug:           getBool("RESMA_AGENT_DEBUG", false),
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("RESMA_AGENT_TOKEN é obrigatório (compartilhado com o server)")
	}
	if cfg.APIURL == "" {
		return nil, fmt.Errorf("RESMA_API_URL é obrigatório (ex: http://api:8080)")
	}
	// Tenta descobrir node_id/hostname via Docker Info se não vieram do Swarm template
	if cfg.NodeID == "" || cfg.NodeHostname == "" {
		// Best-effort: o collector.go tenta preencher via Docker Info depois.
		// Não falha aqui — o agent pode rodar em standalone sem Swarm.
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

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
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

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		// fallback: inteiro = segundos
		var secs int
		if _, err := fmt.Sscanf(v, "%d", &secs); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return def
}
