// Package client implementa o API client reutilizável do RESMA CLI.
//
// O APIClient é usado por todos os comandos do CLI (resma services, resma nodes,
// resma monitor, etc) para falar com o Go API. Ele gerencia:
//   - Autenticação JWT (login, refresh, persistência)
//   - Requests REST com auto-refresh de token expirado
//   - SSE streaming para o TUI monitor
//
// Tokens são persistidos em ~/.config/resma/credentials.json (XDG-compatible).
package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Credentials representa os tokens persistidos no disco.
type Credentials struct {
	ServerURL    string    `json:"server_url"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsExpired retorna true se o access token já expirou.
func (c *Credentials) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsRefreshable retorna true se há um refresh token disponível.
func (c *Credentials) IsRefreshable() bool {
	return c.RefreshToken != ""
}

// configDir retorna o diretório de config do RESMA (XDG-compatible).
//   - Linux/macOS: $XDG_CONFIG_HOME/resma ou ~/.config/resma
//   - Windows: %APPDATA%\resma
func configDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "resma"), nil
	}
	if appdata := os.Getenv("APPDATA"); appdata != "" {
		return filepath.Join(appdata, "resma"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home dir: %w", err)
	}
	return filepath.Join(home, ".config", "resma"), nil
}

// credentialsPath retorna o caminho completo do arquivo de credenciais.
func credentialsPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// SaveCredentials persiste as credenciais no disco.
// Cria o diretório se necessário. O arquivo tem permissão 0600 (owner-only).
func SaveCredentials(creds *Credentials) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	path := filepath.Join(dir, "credentials.json")
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

// LoadCredentials lê as credenciais do disco.
// Retorna nil, nil se o arquivo não existe (usuário não fez login).
func LoadCredentials() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // não logado — não é erro
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return &creds, nil
}

// ClearCredentials remove o arquivo de credenciais (logout).
// Não retorna erro se o arquivo não existe.
func ClearCredentials() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove credentials: %w", err)
	}
	return nil
}
