// Package internal — detecção e persistência de eventos OOM.
//
// OOM events são CRÍTICOS — não podem ser perdidos mesmo se o manager cair
// por longos períodos. Estratégia:
//  1. Detectar via Docker events stream (action "die" + exitCode 137)
//  2. Persistir imediatamente em disco (events.pending.jsonl — append-only)
//  3. Tentar push para o server em background
//  4. Após ACK do server, remover do arquivo
//
// Formato do arquivo: JSON Lines (um evento por linha).
// Replay no startup: lê o arquivo, faz push dos pendentes, remove os aceitos.
package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventStore persiste OOM events em disco e coordena o push.
type EventStore struct {
	mu      sync.Mutex
	log     *slog.Logger
	path    string
	pending []OOMEvent
	pusher  *Pusher
}

// NewEventStore cria um EventStore que persiste em dir/events.pending.jsonl.
func NewEventStore(dir string, pusher *Pusher) *EventStore {
	if dir == "" {
		dir = "/tmp"
	}
	es := &EventStore{
		log:    slog.Default().With("component", "agent-events"),
		path:   filepath.Join(dir, "events.pending.jsonl"),
		pusher: pusher,
	}
	return es
}

// LoadPending lê o arquivo de pendentes no startup e devolve a lista.
func (es *EventStore) LoadPending() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	f, err := os.Open(es.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev OOMEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			es.log.Warn("linha inválida em events.pending", "err", err)
			continue
		}
		es.pending = append(es.pending, ev)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if len(es.pending) > 0 {
		es.log.Info("eventos OOM pendentes recuperados do disco", "count", len(es.pending))
	}
	return nil
}

// Add persiste um OOM event em disco e adiciona à fila de push.
func (es *EventStore) Add(ev OOMEvent) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Append ao arquivo (cria se não existe)
	f, err := os.OpenFile(es.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("abrir events.pending: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	if _, err := f.Write(line); err != nil {
		return err
	}
	es.pending = append(es.pending, ev)
	return nil
}

// FlushAll tenta fazer push de todos os pendentes. Remove os aceitos do arquivo.
// Chamado periodicamente (ex: a cada CollectInterval) e no startup.
func (es *EventStore) FlushAll(ctx context.Context) {
	es.mu.Lock()
	if len(es.pending) == 0 {
		es.mu.Unlock()
		return
	}
	pending := make([]OOMEvent, len(es.pending))
	copy(pending, es.pending)
	es.mu.Unlock()

	accepted := 0
	for _, ev := range pending {
		if ctx.Err() != nil {
			break
		}
		if err := es.pusher.PushOOM(ctx, ev); err != nil {
			es.log.Warn("push OOM falhou, mantendo pendente", "service", ev.Service, "err", err)
			break
		}
		accepted++
	}
	if accepted > 0 {
		es.mu.Lock()
		es.pending = es.pending[accepted:]
		es.mu.Unlock()
		if err := es.rewriteFile(); err != nil {
			es.log.Error("erro ao reescrever events.pending", "err", err)
		}
		es.log.Info("OOM events enviados", "count", accepted, "remaining", len(es.pending)-0)
	}
}

// rewriteFile reescreve o arquivo apenas com os pendentes restantes.
// Deve ser chamado com es.mu NÃO segurada (chama Lock internamente).
func (es *EventStore) rewriteFile() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	f, err := os.Create(es.path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, ev := range es.pending {
		line, err := json.Marshal(ev)
		if err != nil {
			continue
		}
		w.Write(line)
		w.WriteByte('\n')
	}
	return w.Flush()
}

// PendingCount retorna o número de eventos aguardando push.
func (es *EventStore) PendingCount() int {
	es.mu.Lock()
	defer es.mu.Unlock()
	return len(es.pending)
}

// IsOOM verifica se um evento "die" com exitCode é um OOM (137 = SIGKILL pelo kernel).
// Docker também emite action "oom" para containers que excederam memória.
func IsOOM(action, exitCode string) bool {
	if action == "oom" {
		return true
	}
	if action == "die" && exitCode == "137" {
		return true
	}
	return false
}

// NewOOMEvent cria um OOMEvent a partir de um ContainerEvent.
func NewOOMEvent(ce ContainerEvent) OOMEvent {
	exitCode := 137
	if ce.ExitCode != "" {
		fmt.Sscanf(ce.ExitCode, "%d", &exitCode)
	}
	return OOMEvent{
		TS:          time.Now().UTC().Format(time.RFC3339Nano),
		Service:     ce.Service,
		ContainerID: ce.ContainerID,
		ExitCode:    exitCode,
	}
}
