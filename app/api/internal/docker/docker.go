// Package docker wraps o Docker SDK oficial (moby/moby/client) para uso
// pela API RESMA.
//
// Portado de backend/core/docker_client.py. Mantém o mesmo padrão:
//   - Container cache em memória (evita re-listar a cada ciclo)
//   - Stats cache em memória (atualizado por goroutines de streaming)
//   - Stats streaming por container (conexão persistente)
//   - Event-driven container discovery via Docker events
//
// Melhoria 0b.3 vs Python: GetAllServiceResources() usa ServiceList() que
// retorna Spec.Resources em uma única chamada (elimina N+1 do Python que
// fazia inspect() por serviço).
package docker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/moby/moby/client"
)

// Client wraps *client.Client do moby/moby + caches em memória.
type Client struct {
	cli *client.Client

	mu             sync.RWMutex
	statsCache     map[string]Stats // container_id -> latest stats
	containerCache map[string]ContainerInfo
	streamCancel   map[string]context.CancelFunc // container_id -> cancel stats stream

	log *slog.Logger
}

// New cria o client Docker usando as variáveis padrão (DOCKER_HOST etc.).
// Em produção dentro do Swarm, o socket /var/run/docker.sock é montado ro.
func New(ctx context.Context) (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()) //nolint:staticcheck // SA1019: migrate to client.New in a dedicated PR
	if err != nil {
		return nil, fmt.Errorf("docker client init: %w", err)
	}
	c := &Client{
		cli:            cli,
		statsCache:     make(map[string]Stats),
		containerCache: make(map[string]ContainerInfo),
		streamCancel:   make(map[string]context.CancelFunc),
		log:            slog.Default().With("component", "docker"),
	}
	if err := c.initContainerCache(ctx); err != nil {
		c.log.Warn("container cache init falhou", "err", err)
	}
	return c, nil
}

// CLI expõe o client subjacente para as camadas que precisam de APIs
// específicas não cobertas pelos wrappers.
func (c *Client) CLI() *client.Client { return c.cli }

// Health valida conectividade com o daemon via ping.
func (c *Client) Health(ctx context.Context) error {
	_, err := c.cli.Ping(ctx, client.PingOptions{})
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Close libera recursos do client e cancela todos os stats streams.
func (c *Client) Close() error {
	c.mu.Lock()
	for _, cancel := range c.streamCancel {
		cancel()
	}
	c.streamCancel = nil
	c.statsCache = nil
	c.containerCache = nil
	c.mu.Unlock()
	if c.cli == nil {
		return nil
	}
	return c.cli.Close()
}
