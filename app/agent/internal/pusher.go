// Package internal — pusher HTTP com retry exponencial e backoff.
//
// O agent envia 3 tipos de payload para o server:
//   - POST /api/agent/ingest/metrics  (MetricsBatch)
//   - POST /api/agent/ingest/oom      (OOMEvent)
//   - POST /api/agent/heartbeat       (Heartbeat)
//
// Estratégia de resiliência:
//   - Exponential backoff: 1s, 2s, 4s, 8s, 16s, 30s (cap)
//   - Métricas: ring buffer em memória (buffer.go) — descarta mais antigo se cheio
//   - OOM events: persistidos em disco (events.go) — nunca perdidos
//   - Heartbeat: best-effort, sem retry agressivo (próximo ciclo reenvia)
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"
)

// Pusher envia payloads para o RESMA API com retry exponencial.
type Pusher struct {
	cfg     pusherConfig
	log     *slog.Logger
	client  *http.Client
	mu      sync.Mutex
	backoff float64 // current backoff seconds (resets on success)
}

type pusherConfig interface {
	APIURL() string
	Token() string
	NodeID() string
}

// NewPusher cria um novo Pusher.
func NewPusher(cfg pusherConfig) *Pusher {
	return &Pusher{
		cfg:    cfg,
		log:    slog.Default().With("component", "agent-pusher"),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// PushMetrics envia um batch de métricas. Retorna nil se aceito (2xx).
// Em caso de falha, o caller deve manter os pontos no buffer para retry.
func (p *Pusher) PushMetrics(ctx context.Context, points []MetricPoint) error {
	if len(points) == 0 {
		return nil
	}
	batch := MetricsBatch{
		NodeID:  p.cfg.NodeID(),
		Metrics: points,
	}
	return p.postWithRetry(ctx, "/api/agent/ingest/metrics", batch, "metrics")
}

// PushOOM envia um evento OOM. Retorna nil se aceito.
func (p *Pusher) PushOOM(ctx context.Context, ev OOMEvent) error {
	return p.postWithRetry(ctx, "/api/agent/ingest/oom", ev, "oom")
}

// PushHeartbeat envia um heartbeat. Best-effort — loga warning em falha.
func (p *Pusher) PushHeartbeat(ctx context.Context, hb Heartbeat) error {
	return p.postWithRetry(ctx, "/api/agent/heartbeat", hb, "heartbeat")
}

// postWithRetry faz POST com exponential backoff até success ou ctx cancelado.
// Máximo 5 tentativas. Backoff: 1s, 2s, 4s, 8s, 16s (cap 30s).
func (p *Pusher) postWithRetry(ctx context.Context, path string, body interface{}, kind string) error {
	url := p.cfg.APIURL() + path
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", kind, err)
	}

	const maxAttempts = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := p.postOnce(ctx, url, payload)
		if err == nil {
			p.mu.Lock()
			p.backoff = 0
			p.mu.Unlock()
			return nil
		}
		lastErr = err
		// 4xx (exceto 429) não vale retry — payload inválido/token errado
		if isClientError(err) {
			p.log.Error("push rejeitado pelo server (não retentável)", "kind", kind, "err", err)
			return err
		}
		// backoff exponencial com jitter simples
		p.mu.Lock()
		p.backoff = math.Min(math.Pow(2, float64(attempt)), 30)
		delay := time.Duration(p.backoff*float64(time.Second)) + time.Duration(attempt*100)*time.Millisecond
		p.mu.Unlock()
		p.log.Warn("push falhou, retentando", "kind", kind, "attempt", attempt+1, "delay", delay, "err", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return fmt.Errorf("push %s falhou após %d tentativas: %w", kind, maxAttempts, lastErr)
}

// postOnce faz uma única tentativa de POST.
func (p *Pusher) postOnce(ctx context.Context, url string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.Token())
	req.Header.Set("X-RESMA-Agent", "1")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
		return clientError{code: resp.StatusCode, msg: fmt.Sprintf("server rejeitou: %d", resp.StatusCode)}
	}
	return fmt.Errorf("server status %d", resp.StatusCode)
}

// clientError indica erro 4xx (não retentável, exceto 429).
type clientError struct {
	code int
	msg  string
}

func (e clientError) Error() string { return e.msg }

func isClientError(err error) bool {
	_, ok := err.(clientError)
	return ok
}
