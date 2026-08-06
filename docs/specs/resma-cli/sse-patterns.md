# RESMA CLI — Padrões de SSE em Go

> **Status:** Padrões de implementação — SSE via stdlib + Bubble Tea

Este documento descreve os padrões de implementação de **Server-Sent Events (SSE)** utilizados pelo `resma-cli`. O foco é em código idiomático em Go, usando apenas a biblioteca padrão (`net/http`) para o cliente SSE e integrando o fluxo de eventos com o framework TUI [Bubble Tea](https://github.com/charmbracelet/bubbletea).

> **Nota sobre Bubble Tea:** os componentes TUI do `resma-cli` são **custom** (sem uso de `bubbles` — a biblioteca de componentes prontos do ecossistema Charm). No entanto, o **framework Elm Architecture** do Bubble Tea (`Model`/`Update`/`View` + `Cmd`/`Msg`) é usado como base arquitetural da TUI. Isso significa: modelos próprios, views renderizadas manualmente com `lipgloss`, mas o loop de eventos e o sistema de `Cmd` vêm do `bubbletea`.

---

## 1. Visão geral do protocolo SSE

SSE é um protocolo unidirecional (servidor → cliente) construído sobre HTTP/1.1. O cliente abre uma conexão HTTP `GET` com `Accept: text/event-stream` e o servidor mantém a conexão aberta, enviando eventos no formato `text/event-stream`.

### Campos de um evento

Cada evento é composto por linhas no formato `campo: valor`. Um evento é delimitado por uma linha em branco (`\n\n`). Os campos definidos pela especificação são:

| Campo   | Descrição                                                                 | Exemplo                  |
|---------|---------------------------------------------------------------------------|--------------------------|
| `event` | Nome/tipo do evento (opcional; default `message`)                         | `event: task.updated`    |
| `data`  | Carga útil do evento (pode haver múltiplas linhas `data`, concatenadas)   | `data: {"id":42}`        |
| `id`    | Identificador do evento (usado para retomar via `Last-Event-ID`)          | `id: 1234`               |
| `retry` | Intervalo de reconexão sugerido pelo servidor (em milissegundos)          | `retry: 5000`            |

### Exemplo de payload bruto

```
event: task.updated
id: 1234
data: {"id":42,"status":"running"}

event: metrics.sample
data: {"cpu":0.42,"mem":0.71}

: comentários são ignorados (heartbeat/comment line)
```

### `Last-Event-ID`

Quando o cliente reconecta, o navegador (ou, no nosso caso, o cliente Go) envia o último `id` recebido no cabeçalho `Last-Event-ID`. O servidor usa esse valor para retomar o fluxo a partir do evento seguinte, evitando perda de eventos durante quedas.

### `retry`

O servidor pode sugerir um intervalo de reconexão via `retry`. Clientes SSE nativos dos navegadores respeitam esse valor automaticamente. No `resma-cli`, implementamos o mesmo comportamento no cliente Go, com backoff exponencial como fallback.

---

## 2. Parser SSE via stdlib `net/http`

O exemplo abaixo é um cliente SSE completo, compilável, usando apenas a biblioteca padrão. Ele:

- Usa `context.Context` para cancelamento (timeout, Ctrl+C, fechamento da TUI).
- Usa `bufio.Scanner` com um split customizado para detectar limites de evento (`\n\n`).
- Faz dispatch por tipo de evento (`event`).
- Suporta reconexão com `Last-Event-ID`.

```go
// sse_client.go
package sse

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Event representa um evento SSE decodificado.
type Event struct {
	ID    string // campo "id"
	Type  string // campo "event" (default "message")
	Data  string // campo(s) "data" concatenados
	Retry int    // campo "retry" em ms (0 = não definido)
}

// Handler é uma função que processa um evento recebido.
// Retornar erro não-fatal encerra o loop principal.
type Handler func(Event) error

// NOTA: A implementação real do resma-cli usa este pattern de Handler callback
// (func(Event) error), NÃO channels. O Handler é invocado síncronamente dentro
// do loop de parse para cada evento decodificado. O pattern de canal mostrado
// na seção 4 é apenas uma camada de adaptação para o Bubble Tea — o cliente SSE
// em si não usa canais internamente.

// Client é um cliente SSE mínimo baseado em stdlib.
type Client struct {
	URL     string
	Headers map[string]string
	Handler Handler
}

// Connect abre a conexão SSE e processa eventos até o contexto ser cancelado
// ou ocorrer um erro fatal. Reconecta automaticamente em caso de queda,
// enviando o último ID recebido via Last-Event-ID.
func (c *Client) Connect(ctx context.Context) error {
	var lastEventID string

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		retryAfter, err := c.connectOnce(ctx, &lastEventID)
		if err != nil {
			// Contexto cancelado: encerra sem reconectar.
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Erro de rede: aplica backoff e reconecta.
			fmt.Printf("sse: conexão perdida (%v), reconectando em %s\n", err, retryAfter)
		}

		select {
		case <-time.After(retryAfter):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// connectOnce abre uma única conexão SSE e processa eventos até erro/EOF.
// Retorna o intervalo sugerido para reconexão e o erro (se houver).
func (c *Client) connectOnce(ctx context.Context, lastEventID *string) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return 2 * time.Second, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}
	if *lastEventID != "" {
		req.Header.Set("Last-Event-ID", *lastEventID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 2 * time.Second, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 5 * time.Second, fmt.Errorf("status inesperado: %d", resp.StatusCode)
	}

	return c.parseStream(ctx, resp.Body, lastEventID)
}

// parseStream lê o corpo da resposta e decodifica eventos SSE.
func (c *Client) parseStream(ctx context.Context, body io.Reader, lastEventID *string) (time.Duration, error) {
	scanner := bufio.NewScanner(body)
	// Buffer maior para eventos com payloads grandes (ex.: snapshots de métricas).
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	scanner.Split(splitSSEEvent)

	var retryHint time.Duration = 3 * time.Second

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return retryHint, err
		}

		raw := scanner.Bytes()
		if len(bytes.TrimSpace(raw)) == 0 {
			continue // linha em branco extra entre eventos
		}

		ev, retry, err := decodeEvent(raw)
		if err != nil {
			// Erro de parse não-fatal: loga e continua.
			fmt.Printf("sse: erro de parse (%v)\n", err)
			continue
		}
		if retry > 0 {
			retryHint = time.Duration(retry) * time.Millisecond
		}
		if ev.ID != "" {
			*lastEventID = ev.ID
		}
		if ev.Type == "" {
			ev.Type = "message"
		}
		if c.Handler != nil {
			if err := c.Handler(ev); err != nil {
				return retryHint, err
			}
		}
	}

	// scanner.Err() captura erros de I/O (ex.: conexão fechada pelo servidor).
	return retryHint, scanner.Err()
}

// splitSSEEvent divide o stream em blocos separados por "\n\n" (limite de evento).
func splitSSEEvent(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// decodeEvent decodifica um bloco de linhas "campo: valor" em um Event.
func decodeEvent(block []byte) (Event, int, error) {
	var ev Event
	var retry int

	for _, line := range bytes.Split(block, []byte("\n")) {
		// Remove carriage return de finais de linha CRLF.
		line = bytes.TrimRight(line, "\r")

		// Linhas em branco dentro do bloco são ignoradas.
		if len(line) == 0 {
			continue
		}
		// Comentários (linhas iniciadas por ':') são ignorados.
		if line[0] == ':' {
			continue
		}

		idx := bytes.IndexByte(line, ':')
		var field, value []byte
		if idx < 0 {
			field = line
		} else {
			field = line[:idx]
			value = line[idx+1:]
		}
		// Convenção SSE: um espaço após ':' é removido (se presente).
		value = bytes.TrimPrefix(value, []byte(" "))

		switch string(field) {
		case "id":
			ev.ID = string(value)
		case "event":
			ev.Type = string(value)
		case "data":
			if len(ev.Data) > 0 {
				ev.Data += "\n"
			}
			ev.Data += string(value)
		case "retry":
			var r int
			if _, err := fmt.Sscanf(string(value), "%d", &r); err == nil {
				retry = r
			}
		}
	}

	return ev, retry, nil
}

// --- Exemplo de uso ---
//
// func main() {
//     ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
//     defer cancel()
//
//     c := &sse.Client{
//         URL: "http://localhost:8080/api/sse/tasks",
//         Headers: map[string]string{"Authorization": "Bearer TOKEN"},
//         Handler: func(ev sse.Event) error {
//             fmt.Printf("[%s] %s\n", ev.Type, ev.Data)
//             return nil
//         },
//     }
//     _ = c.Connect(ctx)
// }
//
// Para compilar isoladamente, ajuste o package para `main` e descomente o main().
```

---

## 3. SSE inline mode (streaming sem TUI)

No modo inline, o `resma-cli` imprime eventos diretamente no terminal (sem Bubble Tea). O cancelamento via **Ctrl+C** é tratado com `signal.NotifyContext`, que cancela o contexto e encerra a conexão SSE de forma limpa.

```go
// cmd/resma/sse_inline.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"resma-cli/internal/sse"
)

// RunInlineSSE abre um stream SSE e imprime eventos no stdout até Ctrl+C.
// Exemplo: resma sse tasks --inline
func RunInlineSSE(url, token string) error {
	// signal.NotifyContext cancela o contexto ao receber SIGINT (Ctrl+C) ou SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := &sse.Client{
		URL: url,
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Handler: func(ev sse.Event) error {
			// Pretty-print de payloads JSON, se aplicável.
			var pretty map[string]any
			if err := json.Unmarshal([]byte(ev.Data), &pretty); err == nil {
				out, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Printf("[%s] %s\n", ev.Type, string(out))
			} else {
				fmt.Printf("[%s] %s\n", ev.Type, ev.Data)
			}
			return nil
		},
	}

	fmt.Printf("Conectado a %s (Ctrl+C para encerrar)\n", url)
	return client.Connect(ctx)
}
```

### Pontos-chave do modo inline

- **`signal.NotifyContext`** converte Ctrl+C em cancelamento de contexto — sem goroutines manuais nem `os.Exit`.
- O `defer stop()` restaura o handler de sinal padrão ao final.
- O `Handler` é simples: apenas imprime. Toda a lógica de reconexão vive no `sse.Client`.
- Não há estado TUI; o terminal é tratado como um sink append-only.

---

## 4. SSE + Bubble Tea — padrão "channel bridge"

A integração entre uma goroutine produtora (cliente SSE) e o loop de eventos do Bubble Tea usa um **canal como ponte**. A goroutine produtora envia eventos SSE no canal; o Bubble Tea os consome via um `Cmd` que retorna uma `tea.Msg`. Após consumir, o modelo **rearma** o consumidor emitindo um novo `Cmd`.

### Arquitetura

```
  [Servidor SSE]
        │  HTTP stream
        ▼
  goroutine produtora (sse.Client.Handler)
        │  ev := <-events
        ▼
     canal events (chan sse.Event)
        │  waitForEvent(events) -> tea.Cmd
        ▼
  Bubble Tea Update() -> SSEMsg -> View()
        │  rearma: waitForEvent(events)
        ▼
     (loop)
```

### Código completo

```go
// tui/sse_model.go
package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"resma-cli/internal/sse"
)

// SSEMsg é a mensagem Bubble Tea que carrega um evento SSE.
type SSEMsg struct {
	Event sse.Event
}

// SSEErrMsg sinaliza um erro fatal do stream SSE.
type SSEErrMsg struct {
	Err error
}

// waitForEvent é um Cmd que bloqueia até receber um evento do canal e o
// converte em SSEMsg. Deve ser "rearmado" a cada Update para continuar consumindo.
func waitForEvent(events <-chan sse.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			// Canal fechado: stream encerrado.
			return tea.Quit
		}
		return SSEMsg{Event: ev}
	}
}

// waitForError retorna um Cmd que aguarda um erro fatal.
func waitForError(errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		err, ok := <-errs
		if !ok {
			return nil
		}
		return SSEErrMsg{Err: err}
	}
}

// Model é o modelo Bubble Tea que consome eventos SSE.
type Model struct {
	events   chan sse.Event
	errs     chan error
	cancel   context.CancelFunc
	lastType string
	lastData string
	quitting bool
}

// NewModel cria o modelo, abre o canal-ponte e dispara a goroutine produtora.
func NewModel(url, token string) *Model {
	events := make(chan sse.Event, 16)
	errs := make(chan error, 1)

	// Contexto ligado ao ciclo de vida da TUI: cancelado em Quit().
	ctx, cancel := context.WithCancel(context.Background())

	client := &sse.Client{
		URL: url,
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Handler: func(ev sse.Event) error {
			select {
			case events <- ev:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	// Goroutine produtora: roda o cliente SSE até o contexto ser cancelado.
	go func() {
		defer close(events)
		defer close(errs)
		if err := client.Connect(ctx); err != nil {
			errs <- err
		}
	}()

	return &Model{
		events: events,
		errs:   errs,
		cancel: cancel,
	}
}

// Init inicia os consumidores (rearmáveis) de eventos e erros.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		waitForEvent(m.events),
		waitForError(m.errs),
	)
}

// Update processa mensagens Bubble Tea.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SSEMsg:
		m.lastType = msg.Event.Type
		m.lastData = msg.Event.Data
		// Rearma o consumidor: emite um novo Cmd que aguarda o próximo evento.
		return m, waitForEvent(m.events)

	case SSEErrMsg:
		m.lastData = fmt.Sprintf("erro: %v", msg.Err)
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			// Cancela o contexto: a goroutine produtora encerra e fecha o canal.
			m.cancel()
			return m, tea.Quit
		}

	case tea.QuitMsg:
		m.cancel()
		return m, tea.Quit
	}

	return m, nil
}

// View renderiza o estado atual.
func (m *Model) View() string {
	if m.quitting {
		return "Encerrando...\n"
	}
	return fmt.Sprintf(
		"RESMA CLI — Monitor SSE\n\n  evento: %s\n  dados:  %s\n\n  [q/Ctrl+C] sair\n",
		m.lastType, m.lastData,
	)
}

// --- Ponto de entrada (exemplo) ---
//
// func Run(url, token string) error {
//     p := tea.NewProgram(NewModel(url, token))
//     _, err := p.Run()
//     return err
// }
```

### Padrão de rearme (re-arm)

O ponto crítico da integração é o **rearme**: a cada `SSEMsg` recebida, `Update` retorna `waitForEvent(m.events)` como `Cmd`. Sem isso, apenas o primeiro evento seria consumido. O `tea.Batch` em `Init` inicia os consumidores; cada `Update` mantém o ciclo vivo.

### Reconnect com backoff dentro da TUI

A reconexão já é tratada dentro de `sse.Client.Connect` (seção 2). A TUI não precisa implementar reconexão: do ponto de vista do modelo, o stream simplesmente continua produzindo eventos após uma queda. Erros fatais (ex.: HTTP 401) chegam via `SSEErrMsg` e encerram a TUI.

### FlashField — feedback visual de mudanças via SSE

Valores atualizados via SSE podem usar o componente **`FlashField`** do TUI para feedback visual. O `FlashField` exibe um **flash de ~800ms** quando o valor muda — o campo alterna para uma cor de destaque (ex.: verde/branco) e retorna à cor normal após o intervalo, indicando ao usuário que aquele valor foi atualizado pelo stream SSE.

```go
// tui/flash_field.go
type FlashField struct {
    value     string
    flashTime time.Time
    duration  time.Duration // ~800ms
}

func (f *FlashField) Set(value string) {
    if value != f.value {
        f.flashTime = time.Now()
    }
    f.value = value
}

func (f *FlashField) Render() string {
    if time.Since(f.flashTime) < f.duration {
        // Estilo de destaque (flash)
        return flashStyle.Render(f.value)
    }
    return normalStyle.Render(f.value)
}
```

No `Update` do modelo, ao receber um `SSEMsg`, os campos correspondentes chamam `Set(novoValor)`, disparando o flash automaticamente.

---

## 5. Fetch inicial REST (startup do `resma monitor`)

O `resma monitor` faz um **GET `/api/dashboard`** no startup, em uma goroutine paralela à conexão SSE. Isso é necessário porque o tópico SSE `dashboard` só publica a cada **60 segundos** — sem o fetch inicial, o usuário ficaria até 60s sem dados na tela.

### Fluxo de startup

```
resma monitor
  ├─ goroutine 1: GET /api/dashboard  →  payload JSON imediato (snapshot)
  └─ goroutine 2: GET /api/sse/dashboard  →  stream SSE (updates incrementais a cada 60s)
```

- O resultado do GET REST é renderizado imediatamente na TUI (estado inicial completo).
- A conexão SSE roda em paralelo e envia updates incrementais que **substituem** ou **mesclam** com o snapshot inicial.
- Se o GET REST falhar (ex.: rede instável), a TUI ainda funciona — apenas inicia vazia e aguarda o primeiro evento SSE (até 60s).

> **Por que não apenas SSE?** O SSE é otimizado para updates incrementais, não para snapshots completos. O servidor publica no tópico `dashboard` apenas mudanças/deltas a cada 60s. O fetch REST garante que o usuário veja dados imediatamente ao abrir o monitor, sem esperar o próximo ciclo de publicação.

---

## 6. Tratamento de erros

### Queda de conexão

Tratada dentro de `sse.Client.Connect`: ao detectar `scanner.Err()` ou status não-200, o cliente aplica backoff e reconecta, enviando `Last-Event-ID`. O `Handler` não precisa lidar com isso.

### Erros de parse

Erros de decodificação (JSON inválido, campo desconhecido) são **não-fatais**: logados e descartados, o stream continua. Apenas erros de I/O ou contexto disparam reconexão/encerramento.

### Cancelamento de contexto

O contexto é a única fonte de verdade para encerramento. Regras:

- **Modo inline:** `signal.NotifyContext` cancela ao receber Ctrl+C.
- **Modo TUI:** `context.WithCancel` criado em `NewModel`; `cancel()` chamado em `Update` ao receber `tea.Quit` ou `ctrl+c`.
- O `Handler` sempre faz `select` entre `events <- ev` e `<-ctx.Done()`, garantindo que não bloqueie para sempre se a TUI encerrar enquanto o canal estiver cheio.

### Prevenção de vazamento de goroutines

A goroutine produtora (seção 4) segue três regras para evitar leaks:

1. **Contexto compartilhado:** `client.Connect(ctx)` respeita `ctx.Done()`.
2. **Canais fechados no defer:** `defer close(events)` e `defer close(errs)` garantem que os consumidores (`waitForEvent`, `waitForError) recebam `tea.Quit`/nil ao encerrar.
3. **Handler não-bloqueante:** o `select` em `Handler` evita que a goroutine fique presa se a TUI parar de consumir.

Checklist anti-leak:

- [ ] Toda goroutine tem um caminho de saída via `<-ctx.Done()`.
- [ ] Canais de saída são fechados com `defer` ao final da goroutine.
- [ ] Sends em canais usam `select` com `<-ctx.Done()` como fallback.
- [ ] `http.DefaultClient` respeita `Request.Context()` (cancelamento fecha o body).

---

## 7. Estratégia de reconexão

O `resma-cli` implementa reconexão automática com **backoff exponencial** e **retomada via `Last-Event-ID`**.

### Algoritmo

1. Ao detectar perda de conexão, aguarda `retryAfter` (definido pelo campo `retry` do servidor, ou backoff local como fallback).
2. Backoff exponencial: `base * 2^attempt`, limitado a `maxBackoff`, com jitter.
3. A cada reconexão, envia `Last-Event-ID` com o último `id` recebido.
4. O contador de tentativas é zerado ao receber o primeiro evento com sucesso.

### Implementação

```go
// internal/sse/backoff.go
package sse

import (
	"math/rand"
	"time"
)

// Backoff calcula o próximo intervalo de reconexão com exponential backoff
// e jitter. attempt começa em 0.
type Backoff struct {
	Base       time.Duration // intervalo inicial (ex.: 1s)
	Max        time.Duration // teto (ex.: 30s)
	Multiplier float64       // fator de crescimento (ex.: 2.0)
}

// Next retorna o próximo intervalo. Após atingir Max, permanece em Max.
func (b Backoff) Next(attempt int) time.Duration {
	if b.Base <= 0 {
		b.Base = time.Second
	}
	if b.Max <= 0 {
		b.Max = 30 * time.Second
	}
	if b.Multiplier <= 0 {
		b.Multiplier = 2.0
	}

	d := float64(b.Base)
	for i := 0; i < attempt; i++ {
		d *= b.Multiplier
		if d > float64(b.Max) {
			d = float64(b.Max)
			break
		}
	}

	// Jitter completo: [0, d). Evita thundering herd em reconexões simultâneas.
	jitter := time.Duration(rand.Int63n(int64(d)))
	return jitter
}
```

### Uso dentro de `Connect`

O trecho relevante (integrado ao `Connect` da seção 2) seria:

```go
// Exemplo de integração do backoff no loop de reconexão.
bo := Backoff{Base: time.Second, Max: 30 * time.Second, Multiplier: 2.0}
attempt := 0

for {
    if err := ctx.Err(); err != nil {
        return err
    }

    retryAfter, err := c.connectOnce(ctx, &lastEventID)
    if err != nil {
        if ctx.Err() != nil {
            return ctx.Err()
        }
        // Usa o retry hint do servidor se houver; senão, backoff exponencial.
        if retryAfter <= 0 {
            retryAfter = bo.Next(attempt)
            attempt++
        }
        fmt.Printf("sse: reconectando em %s (tentativa %d)\n", retryAfter, attempt)
    } else {
        // Sucesso: zera o contador de tentativas.
        attempt = 0
    }

    select {
    case <-time.After(retryAfter):
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### Retomada via `Last-Event-ID`

O último `id` recebido é mantido em `lastEventID` (ponteiro compartilhado entre iterações de reconexão). A cada `connectOnce`, se `lastEventID != ""`, ele é enviado no cabeçalho `Last-Event-ID`. O servidor resma responde retomando o fluxo a partir do evento seguinte àquele ID.

---

## 8. Autenticação para SSE

O `resma-cli` usa o cabeçalho **`Authorization: Bearer <token>`** para autenticar a conexão SSE. Isso é preferido a cookies pelos motivos abaixo.

### Bearer header vs cookie

| Aspecto              | `Authorization: Bearer` (usado pelo CLI)        | Cookie de sessão                       |
|----------------------|-------------------------------------------------|----------------------------------------|
| Origem do token      | Config do CLI (`~/.config/resma/token`)         | Browser-only; exige fluxo de login web |
| Envio cross-origin   | Sem restrição de CORS para clientes não-browser | Sujeito a `SameSite`, `Secure`, etc.   |
| Suporte em CLI/TUI   | Nativo via `req.Header.Set`                     | Exige `http.CookieJar` e estado        |
| Expiração            | Token JWT com refresh explícito                 | Sessão server-side com expiração opaca |
| Revogação            | Stateless (JWT) ou denylist                     | Server-side (destrói sessão)           |

### Exemplo de envio do header

```go
// Trecho já presente em sse.Client.connectOnce (seção 2).
req.Header.Set("Accept", "text/event-stream")
req.Header.Set("Cache-Control", "no-cache")
req.Header.Set("Authorization", "Bearer "+token)
```

### Observação sobre `EventSource`

A API `EventSource` dos navegadores **não suporta** headers customizados — por isso navegadores usam cookies. Como o `resma-cli` implementa o cliente SSE manualmente com `net/http`, não há essa limitação: o Bearer header é a escolha natural.

---

## 9. Comparação: SSE vs WebSocket vs Polling

| Critério              | SSE                              | WebSocket                          | Polling                          |
|-----------------------|----------------------------------|------------------------------------|----------------------------------|
| Direção               | Servidor → cliente (unidirecional) | Bidirecional                      | Cliente → servidor (por requisição) |
| Protocolo             | HTTP/1.1 (`text/event-stream`)   | Upgrade para WS (frame-based)      | HTTP request/response normal     |
| Complexidade cliente  | Baixa (stdlib `net/http`)        | Alta (exige `gorilla/websocket` ou similar) | Mínima (HTTP GET em loop) |
| Auto-reconnect        | Nativo na spec (`retry`, `Last-Event-ID`) | Manual (sem garantia de ordem) | N/A (cada poll é independente)   |
| Compatibilidade proxy | Alta (HTTP comum)                | Média (upgrade pode ser bloqueado) | Alta                             |
| Overhead por evento   | Baixo (HTTP keep-alive, texto)   | Mínimo (frames binários)           | Alto (header HTTP por request)   |
| Melhor para o CLI     | **Sim** — updates de metrics, tasks, events | Quando há comandos cliente→servidor frequentes | Apenas fallback/degrade |

### Justificativa para SSE no `resma-cli`

O `resma-cli` consome **atualizações** (métricas, status de agentes, tarefas, eventos do Docker) — um padrão estritamente servidor→cliente. SSE oferece:

- Reconexão e retomada gratuitas (spec-level).
- Implementação trivial com stdlib.
- Compatibilidade total com proxies HTTP corporativos.
- Overhead mínimo comparado a polling.

WebSocket seria exagero: não há fluxo cliente→servidor frequente (comandos do CLI são requisições REST pontuais, não stream contínuo).

---

## 10. Tópicos SSE consumidos pelo `resma-cli`

O backend resma expõe endpoints SSE sob `/api/sse/{topic}`. O CLI consome os 10 tópicos abaixo.

| Tópico              | Endpoint                          | Descrição                                                              | Exemplo de `event`        |
|---------------------|-----------------------------------|------------------------------------------------------------------------|---------------------------|
| `metrics`           | `/api/sse/metrics`                | Stream periódico de métricas do sistema (CPU, memória, disco).         | `event: metrics.sample`   |
| `dashboard`         | `/api/sse/dashboard`              | Dados agregados para o dashboard (visão geral do cluster).             | `event: dashboard.update` |
| `events`            | `/api/sse/events`                 | Eventos do Docker, incluindo OOM e rollback.                           | `event: docker.oom`       |
| `services`          | `/api/sse/services`               | Mudanças de estado de serviços do Swarm (create, update, remove).      | `event: service.updated`  |
| `nodes`             | `/api/sse/nodes`                  | Mudanças de estado de nodes do Swarm (join, leave, drain).             | `event: node.state`       |
| `tasks`             | `/api/sse/tasks`                  | Atualizações de tarefas (created, running, completed, failed).         | `event: task.updated`     |
| `agents`            | `/api/sse/agents`                 | Mudanças de estado de agentes (start, stop, heartbeat, erro).          | `event: agent.state`      |
| `change-log`        | `/api/sse/change-log`             | Log de mudanças de configuração aplicadas no cluster.                  | `event: change.applied`   |
| `service-detail`    | `/api/sse/service-detail/{name}`  | Detalhe de um serviço específico (stats, tasks, config).               | `event: service.detail`   |
| `container-detail`  | `/api/sse/container-detail/{id}`  | Detalhe de um container específico (stats, events, inspect).           | `event: container.detail` |

> **Nota:** os alertas (OOM, disk, memory leak) **não** possuem um tópico SSE próprio. Eles chegam através dos tópicos `events` (eventos do Docker, como OOM e rollback) e `metrics` (limiares de CPU/memória/disco).

> **Nota sobre filtro de eventos no tópico `dashboard`:** o tópico `dashboard` recebe **múltiplos tipos de evento** (ex.: `cluster.update`, `storage.update`). O cliente deve **filtrar por `event.Type`** antes de deserializar o payload, para não tentar decodificar um payload de storage em uma struct de cluster (ou vice-versa). Exemplo:
>
> ```go
> Handler: func(ev sse.Event) error {
>     switch ev.Type {
>     case "cluster.update":
>         var c ClusterPayload
>         json.Unmarshal([]byte(ev.Data), &c)
>         // ...
>     case "storage.update":
>         var s StoragePayload
>         json.Unmarshal([]byte(ev.Data), &s)
>         // ...
>     default:
>         // Tipo desconhecido: ignorar (não deserializar).
>     }
>     return nil
> },
> ```

### Mapeamento para comandos do CLI

```
resma sse metrics            →  GET /api/sse/metrics
resma sse dashboard          →  GET /api/sse/dashboard
resma sse events             →  GET /api/sse/events
resma sse services           →  GET /api/sse/services
resma sse nodes              →  GET /api/sse/nodes
resma sse tasks              →  GET /api/sse/tasks
resma sse agents             →  GET /api/sse/agents
resma sse change-log         →  GET /api/sse/change-log
resma sse service-detail     →  GET /api/sse/service-detail/{name}
resma sse container-detail   →  GET /api/sse/container-detail/{id}
```

Cada comando aceita a flag `--inline` (modo direto no terminal, seção 3) ou roda no modo TUI padrão (seção 4). A flag `--last-event-id <id>` permite retomar manualmente a partir de um ID específico, útil para scripts.

### Exemplo de evento `task.updated`

```
event: task.updated
id: 9821
data: {"id":42,"status":"running","progress":0.67,"agent":"worker-3"}

```

### Exemplo de evento `docker.oom`

```
event: docker.oom
id: 441
data: {"container":"worker-3","service":"api","msg":"OOM killed","ts":"2025-01-15T12:34:56Z"}

```

---

## Referências

- [SSE spec — W3C](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [MDN: Using Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events)
- [Bubble Tea — tutorial](https://github.com/charmbracelet/bubbletea#tutorial)
