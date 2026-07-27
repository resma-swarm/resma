# PROMPT — Corrigir bug crítico do SSE no RESMA

## Contexto

Você está trabalhando no projeto **RESMA** (D:\allt\resma) — um gerenciador de recursos para Docker Swarm em Go + React. O projeto usa SSE (Server-Sent Events) para entregar atualizações em tempo real ao frontend, mas uma auditoria profunda revelou que **o SSE está completamente quebrado** — nenhum tópico tem publisher e consumer ao mesmo tempo.

## Stack relevante

- **Backend:** Go 1.26 + net/http + DuckDB + Docker SDK (em `app/api/`)
- **Frontend:** React 19 + Vite 6 + TanStack Query + shadcn/ui (em `frontend/src/`)
- **SSE:** Broker custom em `app/api/internal/sse/` com pub/sub por tópico
- **Build do Go:** SEMPRE em Docker (`docker compose exec go-dev go build ./...`) — go-duckdb requer CGO com gcc, não builda no Windows nativo
- **Regra de ouro:** NUNCA alterar componentes originais do shadcn em `frontend/src/components/ui/`

## Revisões aplicadas (análise técnica pré-implementação)

> Esta seção documenta correções ao prompt original, identificadas validando
> o código real antes de implementar. Estas revisões são **normativas** — a
> implementação segue estes ajustes, não o texto original abaixo delas.

- **Gap A — Ordem em `main.go`:** hoje o collector é instanciado (linha 75)
  **antes** do server (linha 85). A correção exige inverter: criar o server
  antes do collector para passar `srv.SSEHandler()`.
- **Gap B — `TOPIC_QUERY_MAP` estava errado:** o mapa invalidava
  `["recommendations"]` e `["services-sparklines"]`, mas as queryKeys reais em
  `Services.tsx` são `["recs-summary"]` e `["service-sparklines"]` (singular).
  Corrigido no mapa.
- **Gap C — Schedules real-time:** o tópico `events` só dispara em OOM. Schedules
  reagem a **change-log / execução de schedule**, conceito distinto. Adicionado
  novo tópico `change-log` com publisher no scheduler. Schedules.tsx assina
  `change-log` (não `events`).
- **Gap D — Cadência do Dashboard:** o Dashboard passa a assinar **também** o
  tópico `metrics` (publicado a cada ~5s pelo `collectLoop`), além de
  `dashboard` (publicado pelo `clusterCollectLoop`). Assim o dashboard
  atualiza em tempo real de métricas, não só no ritmo do cluster.
- **Gap E — `syncServiceRegistry`:** não é goroutine própria; é chamado dentro
  de `collectLoop` (linha 172). Publica `services` a cada ciclo de coleta.
- **Gap F — Smoke test não cobre SSE:** a validação de "eventos chegam" é
  manual via DevTools/Playwright. O smoke test (32 testes) valida regressão
  geral, não SSE especificamente.

## O bug — descrição completa

### Sintoma
O frontend mostra `sseConnected = true` (a conexão SSE está aberta), mas **as telas só atualizam a cada 5 minutos via polling de fallback**. Nenhum evento SSE chega de fato para as 4 telas principais (Dashboard, Nodes, Services, Tasks).

### Causa raiz
O **Collector** (`app/api/internal/collector/collector.go`) é o componente que coleta métricas, nodes, services, tasks e cluster info do Docker Swarm e escreve no DuckDB. Ele é instanciado em `app/api/cmd/server/main.go` **sem receber referência ao SSE handler**:

```go
col := collector.New(cfg, store, dockerCli)  // Sem SSE handler
col.Start(rootCtx)
```

Como o Collector não tem acesso ao broker SSE, ele **só escreve no DB e nunca notifica o frontend**. Os tópicos que as telas assinam nunca recebem eventos.

### Matriz do problema (validada pelo Technical Council)

| Tópico SSE | Rota existe? | Publisher? | Consumer (página)? | Status |
|------------|-------------|-----------|-------------------|--------|
| `metrics` | ✅ `/api/sse/metrics` | ✅ `agent_handlers.go:167` | ❌ NENHUM | **DESPERDÍCIO** — agent publica mas ninguém assina |
| `events` | ✅ `/api/sse/events` | ✅ `agent_handlers.go:206` | ❌ NENHUM | **DESPERDÍCIO** — agent publica OOM mas ninguém assina |
| `agents` | ✅ `/api/sse/agents` | ✅ `agent_handlers.go:240` | ❌ NENHUM | **DESPERDÍCIO** — agent publica heartbeat mas ninguém assina |
| `dashboard` | ✅ `/api/sse/dashboard` | ❌ NENHUM | ✅ `Dashboard.tsx` | **BUG CRÍTICO** — assina mas nada chega |
| `services` | ✅ `/api/sse/services` | ❌ NENHUM | ✅ `Services.tsx` | **BUG CRÍTICO** — assina mas nada chega |
| `nodes` | ✅ `/api/sse/nodes` | ❌ NENHUM | ✅ `Nodes.tsx` | **BUG CRÍTICO** — assina mas nada chega |
| `tasks` | ✅ `/api/sse/tasks` | ❌ NENHUM | ✅ `Tasks.tsx` | **BUG CRÍTICO** — assina mas nada chega |
| `change-log` | ❌ rota a criar | ❌ NENHUM | ✅ `Schedules.tsx` (após Gap C) | **BUG** — sem publisher nem rota |

**Resumo:** NENHUM tópico SSE tem publisher E consumer ao mesmo tempo. O SSE está conectado mas silencioso.

## Evidência no código

### Backend — publishers atuais (apenas em agent_handlers.go)

```go
// app/api/internal/server/agent_handlers.go:167
s.sse.Publish("metrics", "agent_metrics", map[string]any{"node_id": batch.NodeID, "count": len(rows)})

// app/api/internal/server/agent_handlers.go:206
s.sse.Publish("events", "oom", map[string]any{"service": ev.Service, "node_id": nodeID})

// app/api/internal/server/agent_handlers.go:240
s.sse.Publish("agents", "heartbeat", map[string]any{"node_id": hb.NodeID})
```

### Backend — rotas SSE registradas (sse/handlers.go:68-75)

```go
mux.HandleFunc("GET /api/sse/metrics", h.handleSSE("metrics"))
mux.HandleFunc("GET /api/sse/dashboard", h.handleSSE("dashboard"))
mux.HandleFunc("GET /api/sse/events", h.handleSSE("events"))
mux.HandleFunc("GET /api/sse/services", h.handleSSE("services"))
mux.HandleFunc("GET /api/sse/nodes", h.handleSSE("nodes"))
mux.HandleFunc("GET /api/sse/tasks", h.handleSSE("tasks"))
mux.HandleFunc("GET /api/sse/agents", h.handleSSE("agents"))
```

### Backend — Collector não tem SSE (collector.go)

O struct `Collector` (linhas 28-43) NÃO tem campo para SSE handler. O `collector.New()` (linha 56) não recebe SSE. As 9 goroutines de coleta (`collectLoop`, `containerEventLoop`, `oomListener`, `retentionLoop`, `nodeCollectLoop`, `clusterCollectLoop`, `storageCollectLoop`, `taskCollectLoop`, `agentHealthLoop`) apenas escrevem no DB.

### Frontend — hook use-event-source.ts

O hook usa estratégia híbrida: SSE para invalidação de cache + TanStack Query para refetch. Quando um evento SSE chega, o hook invalida as queryKeys mapeadas para aquele tópico:

```typescript
// frontend/src/hooks/use-event-source.ts:38-47
const TOPIC_QUERY_MAP: Record<string, string[][]> = {
  metrics: [["dashboard"], ["services"], ["services-sparklines"]],
  dashboard: [["dashboard"], ["storage-summary"]],
  events: [["oom-events"], ["change-log"]],
  services: [["services"], ["recommendations"], ["services-sparklines"]],
  nodes: [["nodes"], ["cluster"]],
  tasks: [["tasks"], ["services-health"]],
  agents: [["agents"], ["nodes"]],
}
```

### Frontend — páginas que assinam SSE

- **Dashboard.tsx** (linha 77): `useEventSource({ topic: "dashboard", invalidateQueries: [["dashboard"], ["storage-summary"]] })`
- **Nodes.tsx** (linha 63): `useEventSource({ topic: "nodes", invalidateQueries: [["nodes"], ["cluster"]] })`
- **Services.tsx** (linha 123): `useEventSource({ topic: "services", invalidateQueries: [["services"], ["recs-summary"], ["service-sparklines"]] })`
- **Tasks.tsx** (linha 54): `useEventSource({ topic: "tasks", invalidateQueries: [["tasks"], ["services-health"]] })`

### Frontend — páginas que NÃO assinam SSE (mas deveriam)

- **NodeDetail.tsx** — usa apenas polling 30s via `useRefreshInterval()`
- **ServiceDetail.tsx** — usa apenas polling 30s via `useRefreshInterval()`
- **ContainerDetail.tsx** — usa apenas polling 30s via `useRefreshInterval()`
- **Schedules.tsx** — usa polling fixo 30s

## Proposta de solução

### Parte 1 — Backend: injetar SSE handler no Collector

1. **Modificar `collector.New()`** para receber o SSE handler:
   ```go
   func New(cfg *config.Config, database *db.Store, dc *docker.Client, ssePublisher SSEPublisher) *Collector
   ```
   (usar uma interface `SSEPublisher` com método `Publish(topic, eventType string, payload any)` para não acoplar o collector ao pacote sse diretamente)

2. **Modificar `app/api/cmd/server/main.go`** — **inverter a ordem de
   instanciação** (Gap A): criar o server **antes** do collector para poder
   passar o SSE handler:
   ```go
   srv := server.New(cfg, store, dockerCli, authSvc)
   col := collector.New(cfg, store, dockerCli, srv.SSEHandler())
   col.Start(rootCtx)
   defer col.Stop()
   // scheduler também recebe o SSE handler (para publicar change-log)
   sch := scheduler.New(store, dockerCli, srv.SSEHandler())
   ```

3. **Adicionar publishers no Collector** nos loops apropriados:
   - `collectLoop()` (via `collectOnce`) → publicar `metrics` após inserir métricas no DB (este é o principal — dispara a cada ciclo de coleta, ~5s)
   - `syncServiceRegistry()` → publicar `services` após upsert de serviços (roda dentro de `collectLoop`)
   - `nodeCollectLoop()` (via `collectNodes`) → publicar `nodes` após atualizar nodes
   - `taskCollectLoop()` (via `collectTasks`) → publicar `tasks` após detectar mudança em tasks (usar o flag `changed` que já existe)
   - `clusterCollectLoop()` (via `collectCluster`) → publicar `dashboard` após atualizar cluster info
   - `oomListener()` → publicar `events` ao detectar OOM local (hoje só agent remoto publica)
   - `agentHealthLoop()` → publicar `agents` ao marcar agent como stale
   - `storageCollectLoop()` (via `collectStorage`) → publicar `dashboard` após atualizar storage (storage-summary é invalidado pelo tópico dashboard)

4. **Adicionar publisher `change-log` no Scheduler** (Gap C):
   - `scheduler.executeOne()` → publicar `change-log` após concluir/falhar/pular
     um schedule (qualquer mudança de status). O scheduler recebe o SSE handler
     via `scheduler.New(db, docker, ssePublisher)`.
   - Registrar rota SSE `GET /api/sse/change-log` em `sse/handlers.go`.

5. **Decisão sobre os publishers existentes em `agent_handlers.go`:**
   - Manter os publishers de `metrics`, `events`, `agents` em `agent_handlers.go` (são para dados que vêm de agents remotos, não do collector local)
   - O Collector local também publica nos mesmos tópicos (para dados coletados localmente no manager)
   - Ambos publicam no mesmo broker — não há conflito, é pub/sub

### Parte 2 — Frontend: conectar consumers aos tópicos ativos

1. **Fazer as páginas existentes também assinarem os tópicos que têm publishers:**
   - Dashboard.tsx → passa a assinar **também** `metrics` (Gap D) além de `dashboard`. Invalida `["dashboard"]`, `["storage-summary"]`.
   - Nodes.tsx já assina `nodes` — vai passar a receber eventos
   - Services.tsx já assina `services` — vai passar a receber eventos
   - Tasks.tsx já assina `tasks` — vai passar a receber eventos

2. **Adicionar SSE nas páginas de detalhe:**
   - NodeDetail.tsx → assinar `nodes` e `metrics` (invalidar `["node-detail", nodeId]`, `["node-metrics", nodeId]`, `["node-services", nodeId]`, `["node-agent", nodeId]`, `["service-sparklines"]`, `["storage-summary"]`)
   - ServiceDetail.tsx → assinar `services` e `metrics` (invalidar `["service-stats", serviceName]`, `["service-metrics", serviceName]`, `["service-containers", serviceName]`, `["service-tasks", serviceName]`, `["services-health"]`, `["service-recommendation", serviceName]`, `["schedules", serviceName]`, `["change-log", serviceName]`)
   - ContainerDetail.tsx → assinar `metrics` (invalidar `["container-stats", cid]`, `["container-metrics", cid]`, `["container-network-info", cid]`)
   - Schedules.tsx → assinar `change-log` (Gap C — **não** `events`) (invalidar `["schedules"]`, `["change-log"]`)

3. **Corrigir o `TOPIC_QUERY_MAP` no hook `use-event-source.ts`** (Gap B — o mapa
   atual **estava errado**, contrariando o prompt original):
   - `services`: trocar `["recommendations"]` → `["recs-summary"]`; trocar `["services-sparklines"]` → `["service-sparklines"]`
   - `metrics`: trocar `["services-sparklines"]` → `["service-sparklines"]`
   - Adicionar entrada `change-log`: `[["schedules"], ["change-log"]]`

### Parte 3 — Validação

1. **Build Go:** `docker compose exec go-dev go build ./...`
2. **Vet:** `docker compose exec go-dev go vet ./...`
3. **Gofmt:** `docker compose exec go-dev gofmt -l .`
4. **Frontend build:** `cd frontend && pnpm build`
5. **Smoke test:** `docker compose exec go-dev go run ./cmd/smoke-test`
6. **Teste manual:** abrir o Dashboard no browser, verificar no DevTools (tab Network) que eventos SSE chegam no `/api/sse/dashboard` (não só `: ping`)
7. **Teste multi-node:** subir `docker compose --profile multi-node -f docker-compose.yml -f docker-compose.multi-node.yml up -d` e verificar que eventos de `metrics`, `events`, `agents` chegam quando agents remotos publicam

## Arquivos principais envolvidos

### Backend (modificar)
- `app/api/internal/collector/collector.go` — adicionar SSE handler, publicar em todos os loops
- `app/api/cmd/server/main.go` — passar SSE handler para o Collector
- `app/api/internal/server/server.go` — expor getter do SSE handler (já existe `SSEHandler()` na linha 61-64)

### Backend (não modificar — já está correto)
- `app/api/internal/sse/broker.go` — broker pub/sub está correto
- `app/api/internal/sse/handlers.go` — rotas SSE estão corretas
- `app/api/internal/server/agent_handlers.go` — publishers de agent estão corretos

### Frontend (modificar)
- `frontend/src/hooks/use-event-source.ts` — revisar TOPIC_QUERY_MAP se necessário
- `frontend/src/pages/NodeDetail.tsx` — adicionar `useEventSource`
- `frontend/src/pages/ServiceDetail.tsx` — adicionar `useEventSource`
- `frontend/src/pages/ContainerDetail.tsx` — adicionar `useEventSource`
- `frontend/src/pages/Schedules.tsx` — adicionar `useEventSource`

### Frontend (não modificar — já está correto)
- `frontend/src/pages/Dashboard.tsx` — já assina `dashboard`
- `frontend/src/pages/Nodes.tsx` — já assina `nodes`
- `frontend/src/pages/Services.tsx` — já assina `services`
- `frontend/src/pages/Tasks.tsx` — já assina `tasks`

## Observações importantes

1. **Não quebrar o backward compatibility:** single-node sem agents deve continuar funcionando — o Collector local publica eventos mesmo sem agents remotos
2. **Não alterar componentes shadcn** em `frontend/src/components/ui/`
3. **Payloads SSE são mínimos** (ex: `{"node_id":"abc","count":42}`) — isso é intencional, o SSE é só um sinal de "dados mudaram", o frontend faz refetch via React Query para buscar os dados completos
4. **O broker tem backpressure:** buffer de 64 eventos por subscriber, drop silencioso se cheio — não precisa mudar isso
5. **Keepalive de 15s** já está implementado no broker — conexões não caem por inatividade
6. **Coalescing no frontend:** o hook já faz coalescing de 500ms — múltiplos eventos SSE viram uma única invalidação
7. **Commits locais são permitidos** — o repo não tem remote configurado, commit livremente com mensagem descritiva

## Critério de aceite

- [ ] Collector publica eventos SSE em `metrics`, `services`, `nodes`, `tasks`, `dashboard`, `events`, `agents` após cada ciclo de coleta relevante
- [ ] Scheduler publica `change-log` ao concluir/falhar/pular schedule; rota `GET /api/sse/change-log` registrada
- [ ] `TOPIC_QUERY_MAP` corrigido (`recs-summary`, `service-sparklines`) com entrada `change-log`
- [ ] Dashboard assina `metrics` + `dashboard`; Nodes, Services, Tasks recebem eventos SSE em tempo real (verificar no DevTools/Playwright — eventos `data: {...}` chegam, não só `: ping`)
- [ ] NodeDetail, ServiceDetail, ContainerDetail assinam SSE e atualizam em tempo real
- [ ] Schedules assina `change-log` e atualiza quando um schedule executa (não só em OOM)
- [ ] Build Go passa sem erros
- [ ] Frontend build passa sem erros
- [ ] Smoke test (32 testes) passa
- [ ] Single-node sem agents continua funcionando (Collector local publica eventos)
- [ ] Multi-node com agents funciona (agents remotos + collector local ambos publicam)
