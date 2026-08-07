// Package sse implementa um broker Server-Sent Events com fan-out via channels.
//
// Portado do conceito do backend Python (sse_manager.py) mas reescrito
// para Go usando channels e http.Flusher. O broker suporta:
//   - Múltiplos tópicos (metrics, dashboard, events, services, nodes)
//   - Fan-out para N subscribers por tópico
//   - Keepalive ping a cada 15s
//   - Disconnect detection via r.Context().Done()
//   - Buffer por subscriber para não bloquear publishers
package sse

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Event é um evento SSE com tipo e payload.
type Event struct {
	Type    string `json:"type"`    // tipo do evento (ex: "metrics", "dashboard")
	Payload any    `json:"payload"` // payload JSON
}

// subscriber é um cliente SSE conectado.
type subscriber struct {
	ch     chan Event // buffer channel para eventos
	closed bool
	mu     sync.Mutex
}

// Broker gerencia subscribers por tópico e faz fan-out de eventos.
type Broker struct {
	mu   sync.RWMutex
	subs map[string]map[*subscriber]struct{} // tópico -> set de subscribers
	log  *slog.Logger

	keepalive time.Duration // cadência do keepalive ping (RESMA_SSE_KEEPALIVE)
}

// New cria um novo Broker SSE. keepalive define o intervalo do ping de keepalive
// enviado a cada subscriber; se <= 0, usa 15s (default histórico).
func New(keepalive time.Duration) *Broker {
	if keepalive <= 0 {
		keepalive = 15 * time.Second
	}
	return &Broker{
		subs:      make(map[string]map[*subscriber]struct{}),
		log:       slog.Default().With("component", "sse"),
		keepalive: keepalive,
	}
}

// Subscribe registra um novo subscriber para um tópico e retorna o channel
// de eventos e uma função de cleanup.
func (b *Broker) Subscribe(topic string) (<-chan Event, func()) {
	sub := &subscriber{
		ch: make(chan Event, 64), // buffer 64 eventos
	}

	b.mu.Lock()
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[*subscriber]struct{})
	}
	b.subs[topic][sub] = struct{}{}
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		if subs, ok := b.subs[topic]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(b.subs, topic)
			}
		}
		b.mu.Unlock()
		sub.mu.Lock()
		if !sub.closed {
			sub.closed = true
			close(sub.ch)
		}
		sub.mu.Unlock()
	}

	return sub.ch, cleanup
}

// Publish envia um evento para todos os subscribers de um tópico.
// Non-blocking: se o buffer de um subscriber estiver cheio, o evento é descartado
// para não bloquear o publisher (subscriber lento não trava o sistema).
//
// O RLock é mantido durante toda a iteração para evitar data race com
// Subscribe/cleanup (que modificam b.subs[topic] sob b.mu.Lock()). Sem isso,
// o runtime aborta com "concurrent map iteration and map write" quando um
// subscriber desconecta enquanto um publisher itera — situação que passa a
// ser frequente com múltiplos publishers (collector) e subscribers (telas).
// O send é non-blocking (select/default), então segurar RLock é seguro e breve.
func (b *Broker) Publish(topic string, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for sub := range b.subs[topic] {
		sub.mu.Lock()
		if sub.closed {
			sub.mu.Unlock()
			continue
		}
		// Non-blocking send
		select {
		case sub.ch <- event:
		default:
			// Buffer cheio — descarta evento para subscriber lento
			b.log.Warn("sse buffer cheio, evento descartado", "topic", topic)
		}
		sub.mu.Unlock()
	}
}

// SubscriberCount retorna o número de subscribers ativos por tópico.
func (b *Broker) SubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[topic])
}

// SubscribedTopicsByPrefix retorna os tópicos ativos (com ao menos 1 subscriber)
// que começam com o prefixo dado. Usado pelo collector para descobrir quais
// tópicos dinâmicos "container-detail/{id}" têm subscribers sem precisar iterar
// sobre todos os containers conhecidos — apenas os que alguém está vendo.
func (b *Broker) SubscribedTopicsByPrefix(prefix string) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.subs))
	for topic := range b.subs {
		if len(topic) >= len(prefix) && topic[:len(prefix)] == prefix {
			out = append(out, topic)
		}
	}
	return out
}

// ServeHTTP implementa o handler HTTP para uma conexão SSE.
// O caller deve ter validado a auth (cookie ou Authorization) antes.
// O tópico é determinado pelo parâmetro topic.
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request, topic string) {
	// Headers SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx: desabilita buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Enviar evento inicial de conexão
	writeSSE(w, "connected", map[string]any{"topic": topic})
	flusher.Flush()

	ch, cleanup := b.Subscribe(topic)
	defer cleanup()

	ctx := r.Context()
	ticker := time.NewTicker(b.keepalive) // keepalive ping (RESMA_SSE_KEEPALIVE)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Keepalive ping
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case event, ok := <-ch:
			if !ok {
				return
			}
			payload, err := json.Marshal(event.Payload)
			if err != nil {
				continue
			}
			writeSSEData(w, event.Type, payload)
			flusher.Flush()
		}
	}
}

// writeSSE escreve um evento SSE no formato: event: <type>\ndata: <json>\n\n
func writeSSE(w http.ResponseWriter, eventType string, data any) {
	payload, _ := json.Marshal(data)
	writeSSEData(w, eventType, payload)
}

// writeSSEData escreve um evento SSE com payload já serializado.
func writeSSEData(w http.ResponseWriter, eventType string, payload []byte) {
	_, _ = w.Write([]byte("event: "))
	_, _ = w.Write([]byte(eventType))
	_, _ = w.Write([]byte("\n"))
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(payload)
	_, _ = w.Write([]byte("\n\n"))
}

// PublishFromContext publica um evento de forma segura (não bloqueia).
// Útil para goroutines do collector publicarem eventos.
func (b *Broker) PublishFromContext(_ context.Context, topic string, eventType string, payload any) {
	b.Publish(topic, Event{Type: eventType, Payload: payload})
}
