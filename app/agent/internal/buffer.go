// Package internal — ring buffer circular em memória para métricas pendentes.
//
// O agent NÃO pode perder dados se o manager cair. Estratégia:
//   - Buffer circular de N pontos por container (cap default 1000)
//   - Quando enche, descarta o ponto mais antigo (FIFO)
//   - Cap total ~10MB para 100 containers x 1000 pontos
//   - OOM events são persistidos em disco (events.go) — não entram aqui
package internal

import (
	"sync"
)

// MetricPoint é um ponto de métrica coletado localmente, pronto para push.
// Mantém os mesmos campos do db.MetricRow do server para 1:1 mapping.
type MetricPoint struct {
	TS                  string  `json:"ts"` // RFC3339Nano
	Service             string  `json:"service"`
	ContainerID         string  `json:"container_id"`
	CPUPercent          float64 `json:"cpu_percent"`
	CPUUsage            int64   `json:"cpu_usage"`
	CPUSystem           int64   `json:"cpu_system"`
	MemUsage            int64   `json:"mem_usage"`
	MemLimit            int64   `json:"mem_limit"`
	MemPercent          float64 `json:"mem_percent"`
	NetRX               int64   `json:"net_rx"`
	NetTX               int64   `json:"net_tx"`
	BlockRead           int64   `json:"block_read"`
	BlockWrite          int64   `json:"block_write"`
	MemWorkingSet       int64   `json:"mem_working_set"`
	CPUThrottledPeriods int64   `json:"cpu_throttled_periods"`
	CPUThrottledTime    int64   `json:"cpu_throttled_time"`
}

// OOMEvent é um evento OOM detectado localmente, pronto para push.
type OOMEvent struct {
	TS          string `json:"ts"` // RFC3339Nano
	Service     string `json:"service"`
	ContainerID string `json:"container_id"`
	ExitCode    int    `json:"exit_code"`
}

// Heartbeat é o sinal de vida periódico do agent.
type Heartbeat struct {
	NodeID          string `json:"node_id"`
	Hostname        string `json:"hostname"`
	ContainersCount int    `json:"containers_count"`
	Version         string `json:"version"`
}

// MetricsBatch é o payload enviado para POST /api/agent/ingest/metrics.
type MetricsBatch struct {
	NodeID  string        `json:"node_id"`
	Metrics []MetricPoint `json:"metrics"`
}

// Buffer é um ring buffer thread-safe de MetricPoints.
// Quando atinge a capacidade, descarta o ponto mais antigo (FIFO).
type Buffer struct {
	mu   sync.Mutex
	data []MetricPoint
	cap  int
	len  int
	head int // próximo índice a escrever
}

// NewBuffer cria um novo ring buffer com capacidade cap.
func NewBuffer(cap int) *Buffer {
	if cap < 1 {
		cap = 1
	}
	return &Buffer{
		data: make([]MetricPoint, cap),
		cap:  cap,
	}
}

// Add adiciona um ponto ao buffer. Se cheio, sobrescreve o mais antigo.
func (b *Buffer) Add(p MetricPoint) {
	b.mu.Lock()
	b.data[b.head] = p
	b.head = (b.head + 1) % b.cap
	if b.len < b.cap {
		b.len++
	}
	b.mu.Unlock()
}

// Len retorna o número de pontos atualmente no buffer.
func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.len
}

// Cap retorna a capacidade máxima do buffer.
func (b *Buffer) Cap() int {
	return b.cap
}

// Drain retorna todos os pontos em ordem FIFO e limpa o buffer.
// O caller deve chamar Ack() após confirmar o push bem-sucedido.
func (b *Buffer) Drain() []MetricPoint {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.len == 0 {
		return nil
	}
	out := make([]MetricPoint, b.len)
	// head aponta para o próximo slot a escrever; o mais antigo está em
	// (head - len) % cap quando len == cap, ou em 0 quando len < cap.
	start := (b.head - b.len + b.cap) % b.cap
	for i := 0; i < b.len; i++ {
		out[i] = b.data[(start+i)%b.cap]
	}
	return out
}

// Ack remove os primeiros n pontos do buffer (após push bem-sucedido).
// Se n >= len, limpa tudo.
func (b *Buffer) Ack(n int) {
	b.mu.Lock()
	if n >= b.len {
		b.len = 0
		b.head = 0
	} else {
		b.len -= n
		// head não muda — os n pontos mais antigos são "liberados"
		// e start avança. Como é ring, basta decrementar len.
	}
	b.mu.Unlock()
}

// IsFull retorna true se o buffer está na capacidade máxima.
func (b *Buffer) IsFull() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.len >= b.cap
}
