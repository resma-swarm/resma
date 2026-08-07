# RESMA — Intervalos de Coleta e Refresh: Alinhamento com Boas Práticas Grafana/Prometheus

> **Status:** Proposta (não implementada) · **Domínio:** Infrastructure Intelligence + Product & Design
> **Baseado em:** Pesquisa no GitHub do Grafana (`TimeSrv.ts`, `SceneRefreshPicker.tsx`), docs grafana.com (context7), docs prometheus.io, PRs #1557/#1995/#70479 do node_exporter/grafana, Red Hat OpenShift blog, Robust Perception, prometheus-users
> **Objetivo:** alinhar intervalos de coleta do RESMA aos benchmarks da comunidade observabilidade e corrigir o dropdown de refresh do frontend

> **INSTRUÇÃO GLOBAL (ambiente de validação):** O Docker local deste projeto roda em
> **modo Swarm**. Toda validação, teste e manipulação de containers deve ser feita
> como **stack Swarm** (`docker stack ls`, `docker stack services`, `docker stack ps`,
> `docker service logs`, `docker exec <service>.<rep>.<id>`), **NUNCA** via
> `docker compose up`. Não há container `api` do compose com `air`/hot reload — o
> serviço `api` roda como réplica da stack `resma` (imagem runtime, sem toolchain Go).
> Para build/test do Go, usar `docker run` temporário com a imagem dev
> (`golang:1.26-bookworm` + gcc) montando o código, ou buildar uma imagem dev e
> executar nela. Os comandos `docker compose exec api go build/test` da spec original
> **não se aplicam** neste ambiente — substituí-los pelos equivalentes Swarm.

---

## 1. Contexto e Motivação

O RESMA coleta métricas de containers Docker Swarm via agents (intervalo configurável por env var) e atualiza o frontend via **SSE (push)**, com polling como fallback. O usuário relatou que o **dropdown de refresh no header não funciona**.

A investigação revelou dois problemas distintos:

1. **Intervalos de coleta fragmentados e inconsistentes** — defaults diferentes entre API/Agent/compose/stack, valores acima do limite de staleness recomendado pela comunidade Prometheus
2. **Dropdown de refresh conceitualmente confuso** — mistura "refresh do frontend" com "taxa de coleta do backend", e é ignorado quando SSE está ativo (o que é o caso default)

A pesquisa no Grafana (arquitetura pull/polling) e Prometheus (scrape_interval) forneceu benchmarks concretos para alinhar os intervalos do RESMA.

### 1.1 Modelos de arquitetura: Grafana vs RESMA

| Aspecto | Grafana | RESMA |
|---------|---------|-------|
| Transporte de dados | **Pull** (frontend pergunta a cada Ns) | **Push** (SSE empurra na cadência da coleta) |
| Dropdown de refresh | Controla re-query do frontend | Hoje ignorado quando SSE ativo |
| Taxa de coleta | Config do datasource (fora do Grafana) | Env vars + ParametersPage |
| Relação coleta/refresh | Totalmente desacoplados | Hoje acoplados (modo "auto" = `collect_interval`) |

**Conclusão:** o RESMA não deve copiar o dropdown do Grafana (arquitetura diferente), mas deve adotar o mesmo princípio: **coleta e refresh são camadas separadas**.

---

## 2. Evidência Coletada (fatos, não opinião)

### 2.1 Benchmarks de intervalos de coleta (Prometheus/Grafana community)

| Tipo de coleta | Benchmark | Fonte |
|----------------|-----------|-------|
| Container metrics (cAdvisor) | `5s`–`15s` | prometheus.io/docs/guides/cadvisor (exemplo oficial: 5s); Alloy docs: 10s |
| Node/host metrics (node_exporter) | `15s` recomendado; `30s`–`60s` comum | PR #1557 node_exporter: "recommended 15s"; Robust Perception: "10-60s é bom" |
| Cluster/k8s health | `30s`–`60s` | Red Hat OpenShift blog: defaults 30s; kube-prometheus: 30s |
| Storage/filesystem (df) | `60s`–`5m`; **máx 2m recomendado** | prometheus-users: staleness após 5m; comunidade: 1m é "grande mas sem impacto" |
| Alert evaluation | `15s`–`30s` crítico; `1m` default; `5m` não-urgente | grafana.com/docs alerting |
| Heartbeat/health | `15s`–`30s` | Prometheus instrumentation docs: heartbeat pattern |
| Scrape máximo (staleness) | **≤2m** (ideal ≤2.5m) | prometheus-users: "max 5m, recommended 2-2.5m" |

### 2.2 Como o Grafana lida com refresh (evidência de código)

- `SceneRefreshPicker` (`@grafana/scenes`): `setInterval` dedicado + `clearInterval` ao mudar interval
- `visibilitychange` pausa refresh em aba oculta e retoma ao voltar
- Interval sincronizado na URL (`?refresh=5s`) — bookmarkable
- "Auto" = calculado pela **janela de tempo + largura do browser** (`calculateInterval`), não pela taxa de coleta
- Cancela queries em execução ao refresh manual
- Mostra `isLoading` no botão (`queryController.isRunning`)
- **Dropdown controla só re-fetch do frontend, NUNCA a taxa de coleta do backend**

### 2.3 Como o SSE do RESMA funciona hoje (evidência de código)

**Backend (`app/api/internal/sse/`):**
- `broker.go`: fan-out via channels, buffer 64 eventos por subscriber, keepalive ping 15s
- `handlers.go`: auth via cookie `sse_session` (TTL 10min) OU header Authorization
- `Publish` é non-blocking: se buffer cheio, descarta evento (subscriber lento não trava)
- Tópicos: `metrics`, `dashboard`, `events`, `services`, `nodes`, `tasks`, `agents`, `change-log`, `service-detail/{name}`, `container-detail/{id}`

**Frontend (`frontend/src/hooks/use-event-source.ts`):**
- Reconexão com backoff exponencial: 1s→2s→4s→8s→16s→30s + renova cookie antes de reconectar
- `setQueryData` direto para eventos com payload completo (zero refetch HTTP)
- Fallback para `invalidateQueries` quando `setQueryData` faz bail out
- **Reconciliação 30s safety-net** (hardcoded): mesmo com SSE ativo, invalida queries a cada 30s para corrigir drift
- Coalescing global: agrupa invalidações de múltiplos hooks em 1 flush a cada 2s

**Frontend (`frontend/src/hooks/use-collection-status.ts`):**
- Detecta fonte: "sse" vs "polling" (janela de 3s para distinguir invalidação por SSE de polling real)
- Estados: `collecting` (dados < 30s), `idle` (SSE ok mas sem dados > 30s), `reconnecting` (sem dados > 30s + SSE off)
- Badge visual no header mostra status em tempo real

---

## 3. Problemas Identificados

### 3.1 Intervalos de coleta inconsistentes (CRÍTICO)

**Onde:** `app/api/internal/config/config.go`, `app/agent/internal/config.go`, `docker-compose*.yml`, `docker-stack.yml`, `scripts/swarm-hpa-demo.yml`

| Env var | API (config.go) | Agent (config.go) | docker-compose.yml | docker-compose.swarm.yml | docker-compose.standalone.yml | docker-stack.yml | swarm-hpa-demo.yml |
|---------|-----------------|-------------------|--------------------|--------------------------|-------------------------------|------------------|--------------------|
| `RESMA_COLLECT_INTERVAL` | 10s | 15s | 5s | 10s (`${...:-10}`) | 5s | 15s | 5s |
| `RESMA_CLUSTER_INTERVAL` | 60s | — | 60s | 60s | 60s | 300s | 30s |
| `RESMA_STORAGE_INTERVAL` | 300s | — | 300s | 300s | 300s | 600s | 300s |
| `RESMA_AGENT_TASK_POLL_INTERVAL` | 15s | — | 15s | 15s | 15s | 30s | — |
| `RESMA_ROLLBACK_POLL_INTERVAL` | 30s | — | *(não definido)* | 10s (`${...:-10}`) | *(não definido)* | *(não definido)* | — |

**Problemas:**
- `COLLECT_INTERVAL` tem **5 valores diferentes** (10s/15s/5s/10s/5s) — mesmo conceito, defaults divergentes
- `STORAGE_INTERVAL` 300s/600s excede o limite de staleness recomendado (≤2m)
- `CLUSTER_INTERVAL` 300s no stack é excessivo para cluster health (benchmark 30s)
- `ROLLBACK_POLL_INTERVAL` só existe em `docker-compose.swarm.yml` (10s) — ausente nos demais composes

### 3.2 Intervalos hardcoded (não configuráveis) (ALTO)

**Onde:** código Go

| Loop | Arquivo | Valor | Problema |
|------|---------|-------|----------|
| Scheduler poll | `app/api/internal/scheduler/scheduler.go:19` | `15s` const | Não é env var |
| Stale service marking | `app/api/internal/collector/collector.go:511` | `1h` hardcoded | Não é env var |
| Retention purge | `app/api/internal/collector/collector.go:479` | `24h` hardcoded | Não é env var (aceitável, mas documentar) |
| SSE keepalive | `app/api/internal/sse/broker.go:161` | `15s` hardcoded | Não é env var |

### 3.3 Dropdown de refresh ignorado quando SSE ativo (ALTO)

**Onde:** 10 páginas do frontend

O padrão varia — todas desativam polling quando SSE está ativo, mas o valor de fallback difere:

| Página | Padrão | Fallback |
|--------|--------|----------|
| Dashboard, Services, Alerts, Nodes, Tasks, ServiceDetail | `sseConnected ? false : refreshInterval` | Usa dropdown |
| NodeDetail, ContainerDetail | `sseConnected ? 300_000 : refreshInterval` | Hardcoded 5m quando SSE ativo |
| Schedules | `sseConnected ? false : 30000` | Hardcoded 30s (ignora dropdown) |
| RollbackWatches | `sseEvents ? false : 30000` | Hardcoded 30s (ignora dropdown) |

- SSE ativo (default) → `refetchInterval = false` (ou hardcoded) → **dropdown tem efeito zero**
- A atualização vem via SSE na cadência do backend, ignorando o usuário
- O dropdown só funciona se SSE cair (fallback), e mesmo assim só em 6 das 10 páginas

### 3.4 Reconciliação 30s hardcoded competindo com dropdown (MÉDIO)

**Onde:** `frontend/src/hooks/use-event-source.ts:327`

```typescript
const RECONCILE_INTERVAL_MS = 30000 // hardcoded
```

- Mesmo com SSE ativo, a cada 30s invalida queries (safety-net)
- Este timer é invisível ao usuário e **não respeita o dropdown**
- Segundo timer competindo com o dropdown

### 3.5 Valores hardcoded de refetchInterval em 6 páginas (MÉDIO)

| Página | Linha | Valor | Respeita SSE? | Problema |
|--------|-------|-------|---------------|----------|
| `ServiceDetail.tsx` | 202, 209 | `30000` | Não (hardcoded puro) | Ignora dropdown e SSE |
| `Recommendations.tsx` | 886 | `30000` | Não (hardcoded puro) | Ignora dropdown e SSE |
| `RollbackWatches.tsx` | 106 | `30000` | Sim (`sseEvents ? false : 30000`) | Ignora dropdown |
| `Schedules.tsx` | 149 | `30000` | Sim (`sseConnected ? false : 30000`) | Ignora dropdown |
| `NodeDetail.tsx` | 127 | `300_000` | Sim (`sseConnected ? 300_000 : refreshInterval`) | Hardcoded 5m quando SSE ativo |
| `ContainerDetail.tsx` | 77 | `300_000` | Sim (`sseConnected ? 300_000 : refreshInterval`) | Hardcoded 5m quando SSE ativo |

### 3.7 Modo "auto" conceitualmente errado (MÉDIO)

**Onde:** `frontend/src/stores/refresh-store.ts:30-31`

```typescript
export function getIntervalMs(mode: RefreshMode, collectInterval: number): number | false {
  if (mode === "auto") return collectInterval * 1000
  return INTERVAL_MAP[mode]
}
```

- "auto" usa `collect_interval` do backend → mistura refresh do frontend com taxa de coleta
- No Grafana, "auto" é calculado pela resolução visual (largura do painel + janela de tempo)
- `useRefreshInterval` faz query a `/config` só para obter `collect_interval` — acoplamento desnecessário

### 3.8 Tickers do backend não cobertos pelas env vars (MÉDIO)

**Tickers em `collector.go` já configuráveis (OK):**
- Linha 211: `collectLoop` → `c.cfg.CollectInterval` ✓
- Linha 549: `nodeCollectLoop` → `c.cfg.CollectInterval` ✓
- Linha 636: `clusterCollectLoop` → `c.cfg.ClusterInterval` ✓
- Linha 700: `storageCollectLoop` → `c.cfg.StorageInterval` ✓
- Linha 806: `taskCollectLoop` → `c.cfg.AgentTaskPollInterval` ✓
- Linha 916: `agentHealthLoop` → `c.cfg.AgentTaskPollInterval` ✓

**Tickers no agent (`app/agent/cmd/agent/main.go`) já configuráveis (OK):**
- Linha 174: collectLoop → `cfg.CollectInterval` ✓
- Linha 205: retry ticker → `cfg.CollectInterval` ✓
- Linha 232: oomFlushLoop → interval próprio ✓
- Linha 248: heartbeatLoop → interval próprio ✓

**Tickers hardcoded (cobertos na seção 3.2):**
- `scheduler.go:19` → `15s` const
- `collector.go:511` → `1h` (staleLoop)
- `collector.go:479` → `24h` (retentionLoop)
- `broker.go:161` → `15s` (SSE keepalive)

### 3.9 Riscos operacionais das mudanças propostas (MÉDIO)

| Mudança | Risco | Mitigação |
|---------|-------|-----------|
| `STORAGE_INTERVAL` 300s→60s | I/O no disco 5x mais frequente em produção | `df` é barato; volumes Docker são poucos; monitorar |
| `CLUSTER_INTERVAL` 60s/300s→30s | Chamadas Docker API 2x–10x mais frequentes em produção | Cluster info é leve (node list, não stats); aceitável |
| `COLLECT_INTERVAL` 5s→15s (dev) | Devs podem achar sistema "mais lento" | Override via `.env` local (ver seção 4.1.5); dev não é o cenário alvo dos defaults |
| `ROLLBACK_POLL_INTERVAL` 10s→30s | Detecção de rollback 20s mais lenta | Rollback é reativo, não tempo real; 30s é alert eval padrão; override via `.env` em dev |

---

## 4. Solução Proposta

### 4.1 Frente A — Backend: intervalos de coleta alinhados aos benchmarks

> **Princípio:** os defaults em `config.go` (API e Agent) e em todos os arquivos de deploy (composes, stack) são **orientados a produção** — o cenário majoritário de usuários. Desenvolvedores podem sobrescrever via env vars locais se precisarem de coleta mais rápida (ver seção 4.1.5).

#### 4.1.1 Defaults de produção (alinhados aos benchmarks)

| Env var | Default atual | Default produção | Benchmark |
|---------|---------------|------------------|-----------|
| `RESMA_COLLECT_INTERVAL` | 10s/15s/5s (inconsistente) | **15s** (unificar) | cAdvisor 5s–15s; node_exporter 15s recomendado |
| `RESMA_CLUSTER_INTERVAL` | 60s/300s (inconsistente) | **30s** | kube-prometheus 30s; cluster health muda rápido |
| `RESMA_STORAGE_INTERVAL` | 300s/600s (arriscado) | **60s** | staleness ≤2m; df é barato |
| `RESMA_AGENT_TASK_POLL_INTERVAL` | 15s | **15s** (manter) | heartbeat 15s padrão |
| `RESMA_ROLLBACK_POLL_INTERVAL` | 30s/10s (inconsistente) | **30s** (unificar) | alert evaluation 30s crítico |

**Onde aplicar (todos os arquivos usam os mesmos defaults de produção):**
- `config.go` (API) — fallback quando env var não está setada
- `config.go` (Agent) — fallback quando env var não está setada
- `docker-compose.yml`, `docker-compose.swarm.yml`, `docker-compose.standalone.yml`, `docker-stack.yml`, `scripts/swarm-hpa-demo.yml` — todos explicitam os mesmos valores de produção

#### 4.1.2 Novas env vars (tornar hardcoded configuráveis)

| Env var nova | Default | Substitui |
|--------------|---------|-----------|
| `RESMA_SCHEDULER_POLL` | `15s` | `pollInterval` const em `scheduler.go:19` |
| `RESMA_SSE_KEEPALIVE` | `15s` | `ticker` em `broker.go:161` |

> **Nota:** `staleLoop` (1h) e `retentionLoop` (24h) permanecem hardcoded — são cadências operacionais que não precisam de ajuste fino.

#### 4.1.3 Arquivos a alterar

- `app/api/internal/config/config.go` — ajustar defaults + adicionar 2 novas env vars
- `app/agent/internal/config.go` — unificar `CollectInterval` default para 15s
- `docker-compose.yml` — alinhar para 15s/30s/60s (não tem ROLLBACK_POLL_INTERVAL hoje)
- `docker-compose.swarm.yml` — alinhar para 15s/30s/60s + ROLLBACK_POLL_INTERVAL 30s
- `docker-compose.standalone.yml` — alinhar para 15s/30s/60s (não tem ROLLBACK_POLL_INTERVAL hoje)
- `docker-stack.yml` — alinhar para 15s/30s/60s
- `scripts/swarm-hpa-demo.yml` — alinhar para 15s/30s/60s (demo de HPA, manter consistente)
- `app/api/internal/scheduler/scheduler.go` — ler `cfg.SchedulerPoll` em vez de `pollInterval` const
- `app/api/internal/sse/broker.go` — keepalive via config (parâmetro no construtor)
- `docs/multi-node.md`, `docs/TECH_SPEC.md`, `README.md` — atualizar defaults documentados

> **Nota sobre test-env/:** os arquivos em `test-env/docker-compose.yml` e `test-env/docker-stack.yml` não definem intervalos RESMA (apenas serviços de teste), portanto não precisam ser alterados.

#### 4.1.4 Validação

```bash
docker compose exec api go build ./...
docker compose exec api go vet ./...
docker compose exec api gofmt -l .
docker compose exec api go test ./...
```

#### 4.1.5 Override para desenvolvimento local

Os defaults de produção (15s/30s/60s) são seguros para dev, mas desenvolvedores podem querer coleta mais rápida para iterar mais rápido. O override é via env vars locais — **não alterar os arquivos de deploy**.

**Opção 1 — env vars no shell antes de subir a stack:**
```bash
# PowerShell (dev local com coleta rápida)
$env:RESMA_COLLECT_INTERVAL="5"
$env:RESMA_ROLLBACK_POLL_INTERVAL="10"
docker compose up -d
```

**Opção 2 — arquivo `.env` na raiz do projeto (não commitado):**
```env
# .env (gitignored) — override local para dev
RESMA_COLLECT_INTERVAL=5
RESMA_ROLLBACK_POLL_INTERVAL=10
```

**Valores sugeridos para dev:**

| Env var | Produção | Dev (override local) | Motivo |
|---------|----------|----------------------|--------|
| `RESMA_COLLECT_INTERVAL` | 15s | **5s** | Ver dados chegando rápido durante desenvolvimento |
| `RESMA_CLUSTER_INTERVAL` | 30s | 30s (mesmo) | Não muda significativamente em dev |
| `RESMA_STORAGE_INTERVAL` | 60s | 60s (mesmo) | Não muda significativamente em dev |
| `RESMA_AGENT_TASK_POLL_INTERVAL` | 15s | 15s (mesmo) | Não muda significativamente em dev |
| `RESMA_ROLLBACK_POLL_INTERVAL` | 30s | **10s** | Testar rollback mais rápido em dev |

> **Importante:** o arquivo `.env` já é gitignored pelo RESMA (ver `.gitignore`). Desenvolvedores que precisarem de coleta rápida devem usar este mecanismo, não alterar os defaults dos arquivos de deploy.

---

### 4.2 Frente B — Frontend: desacoplar modo "auto" de coleta

#### 4.2.1 Redefinir `getIntervalMs` (remover `collectInterval`)

**Arquivo:** `frontend/src/stores/refresh-store.ts`

```typescript
// ANTES:
export function getIntervalMs(mode: RefreshMode, collectInterval: number): number | false {
  if (mode === "auto") return collectInterval * 1000
  return INTERVAL_MAP[mode]
}

// DEPOIS:
export function getIntervalMs(mode: RefreshMode): number | false {
  if (mode === "auto") return 30_000 // fixo 30s (Grafana troubleshooting: "1m ou mais")
  return INTERVAL_MAP[mode]
}
```

**Justificativa:** "auto" deixa de depender do backend. 30s fixo é o recomendado pelo Grafana troubleshooting docs para dados que mudam devagar.

#### 4.2.2 Simplificar `useRefreshInterval`

**Arquivo:** `frontend/src/hooks/use-refresh.ts`

- Remover query a `/config` (não precisa mais de `collect_interval`)
- `getIntervalMs(mode)` sem parâmetro de coleta
- Manter `useCollectInterval` export separado (para display no ParametersPage, se necessário)

#### 4.2.3 Arquivos a alterar

- `frontend/src/stores/refresh-store.ts` — `getIntervalMs` sem `collectInterval`
- `frontend/src/hooks/use-refresh.ts` — remover query `/config` de `useRefreshInterval`

#### 4.2.4 Validação

```bash
cd frontend && pnpm build  # deve compilar sem erros
```

---

### 4.3 Frente C — Dropdown: disabled temporariamente (fase final)

> **Decisão do usuário:** o dropdown fica **disabled** até finalizarmos as Frentes A e B. A implementação completa do dropdown (Frente C) fica documentada abaixo para execução posterior.

#### 4.3.1 Disabled temporário

**Arquivo:** `frontend/src/components/Layout.tsx`

- Adicionar `disabled` no `DropdownMenuTrigger` button
- Tooltip: "Ajuste de intervalo em implementação — use o refresh manual"
- Manter o botão de refresh manual funcional (se existir) ou adicionar um

#### 4.3.2 Plano completo do dropdown (para implementar depois)

**Princípio:** o dropdown controla a **reconciliação safety-net** (hoje hardcoded 30s), não competindo com SSE.

| Cenário | Comportamento |
|---------|---------------|
| SSE ativo + dropdown "5s" | Reconciliação safety-net a cada 5s (em vez de 30s fixo) |
| SSE ativo + dropdown "off" | Sem reconciliação (só SSE push) |
| SSE inativo + dropdown "5s" | Polling puro a cada 5s (fallback) |
| SSE inativo + dropdown "off" | Sem atualização automática (só manual) |

**Implementação:**

1. **Novo hook `useRefreshTimer`** (substitui reconciliação 30s hardcoded):
   ```typescript
   function useRefreshTimer(queryKeys: string[][]) {
     const mode = useRefreshStore(s => s.mode)
     const intervalMs = getIntervalMs(mode)
     useEffect(() => {
       if (!intervalMs) return
       const id = setInterval(() => {
         queryClient.invalidateQueries({ queryKey: queryKeys })
       }, intervalMs)
       return () => clearInterval(id)
     }, [intervalMs])
     // visibilitychange: pausa quando hidden
   }
   ```

2. **Remover reconciliação 30s hardcoded** de `use-event-source.ts:325-338`

3. **Remover `fallbackInterval` pattern** de 10 páginas — substituir por `useRefreshTimer`

4. **Remover hardcoded `30000`/`300_000`** de 6 páginas (ver seção 3.5)

5. **UX (do Grafana):**
   - Spinner no botão RefreshCw durante queries
   - Sync do modo na URL (`?refresh=5s`)
   - `visibilitychange` pausa/retoma
   - Mostrar intervalo calculado no modo Auto

#### 4.3.3 Arquivos a alterar (Frente C — fase final)

- `frontend/src/components/Layout.tsx` — re-enable dropdown + URL sync + spinner
- `frontend/src/hooks/use-refresh.ts` — novo `useRefreshTimer`
- `frontend/src/hooks/use-event-source.ts` — remover reconciliação 30s hardcoded
- 10 páginas com `fallbackInterval`: Dashboard, Services, Alerts, Nodes, Tasks, NodeDetail, ContainerDetail, ServiceDetail, Schedules, RollbackWatches
- 6 páginas com hardcoded `30000`/`300_000`: ServiceDetail, Recommendations, RollbackWatches, Schedules, NodeDetail, ContainerDetail
- 6 páginas com hardcoded `30000`/`300_000` — substituir por `useRefreshTimer`

---

## 5. Ordem de Implementação

| Ordem | Frente | Descrição | Risco | Status |
|-------|--------|-----------|-------|--------|
| 1 | A | Backend: alinhar intervalos de coleta via env vars | Baixo (só defaults) | ✅ Concluído |
| 2 | D | Installer/Upgrader: expor todas as env vars como parâmetros | Baixo (scripts bash) | Pendente |
| 3 | B | Frontend: desacoplar modo "auto" de `collect_interval` | Baixo (só store/hook) | Pendente |
| 4 | C | Dropdown: disabled temporário | Baixo (1 arquivo) | Pendente |
| 5 | C (futura) | Dropdown: implementação completa | Médio (11 páginas) | Pendente |

> **Frente C completa fica aguardando comando do usuário** após A, B e D.

---

## 6. Modelo Final

```
┌─────────────────────────────────────────────────────┐
│  BACKEND (coleta/ingestão)                          │
│  • COLLECT_INTERVAL=15s   (cAdvisor benchmark)      │
│  • CLUSTER_INTERVAL=30s   (kube-prometheus)         │
│  • STORAGE_INTERVAL=60s   (staleness <2m)           │
│  • SCHEDULER_POLL=15s     (env var nova)            │
│  • SSE_KEEPALIVE=15s      (env var nova)            │
│  • Defaults de PRODUÇÃO em todos os arquivos        │
│  • Dev sobrescreve via .env local (ver 4.1.5)       │
│  • Configurável via ParametersPage + env vars       │
│  • SSE publica eventos na cadência da coleta        │
└────────────────────┬────────────────────────────────┘
                     │ SSE push + HTTP fallback
┌────────────────────▼────────────────────────────────┐
│  FRONTEND (refresh/visualização)                    │
│  • Dropdown [disabled por enquanto]                 │
│  • Modo "auto" = 30s fixo (desacoplado de coleta)   │
│  • SSE ativo = transporte primário (push)           │
│  • Reconciliação 30s safety-net (hardcoded por ora) │
│  • [Futuro] dropdown controla reconciliação         │
└─────────────────────────────────────────────────────┘
```

---

## 7. Princípios Garantidos

- 📏 **Coleta ≠ refresh** — desacoplados como no Grafana/Prometheus
- ⏱️ **Intervalos via env vars** — nenhum hardcoded relevante (exceto retention/stale operacionais)
- 📊 **Defaults de produção** — 15s/30s/60s alinhados aos benchmarks da comunidade; dev sobrescreve via `.env`
- 🎛️ **Dropdown funciona sempre** (futura Frente C) — SSE on/off, safety-net respeita dropdown
- 🤖 **Auto faz sentido** — 30s fixo, não taxa de coleta
- 🔗 **URL bookmarkable** (futuro) — estado compartilhável

---

## 8. Installer/Upgrader: customização de intervalos via parâmetros

### 8.1 Contexto

O RESMA é implantado via script de install/upgrade que roda dentro de um container Docker (`app/installer/install.sh`, `app/installer/upgrade.sh`). O usuário invoca via one-liner:

```bash
docker run -it --rm \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest
```

Hoje o installer aceita apenas `STACK_NAME`, `APP_PORT`, `INTERACTIVE`, `RESMA_VERSION` e `RESMA_CORS_ORIGINS` via env vars (`-e VAR=value`). **Nenhuma env var de intervalo de coleta é exposta** — o usuário fica preso aos defaults do `docker-stack.yml`.

### 8.2 Padrão de mercado (evidência)

| Ferramenta | Mecanismo | Referência |
|------------|-----------|------------|
| Docker install script | Flags CLI (`--version`, `--channel`) + env vars | docker/docker-install DeepWiki |
| Docker Desktop MSI | Propriedades CLI (`ENABLEDESKTOPSHORTCUT`, `INSTALLFOLDER`) | docs.docker.com enterprise MSI |
| Grafana CLI | `--configOverrides` (age como env var) + env vars `GF_*` | grafana.com/docs/administration/cli |
| Grafana Helm | `--set key=value` ou `--values values.yaml` | grafana-community/helm-charts |

**Conclusão:** o padrão universal para installers de container é **env vars passadas via `-e VAR=value`** — é o mecanismo nativo do `docker run`, não requer parsing de flags no script, e é compatível com automação (CI/CD, Ansible, Terraform). O RESMA já usa este padrão; basta **expandir as env vars aceitas** para incluir todos os intervalos de coleta.

### 8.3 Solução: todas as env vars de intervalo como parâmetros de install/upgrade

O installer e o upgrader devem aceitar **todas as env vars de intervalo** via `-e VAR=value`. O script lê com `${VAR:-default}` e injeta no `docker stack deploy` via `--env-file` ou inline.

#### 8.3.1 Env vars expostas no install

```bash
docker run -it --rm \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  -e INTERACTIVE=0 \
  -e STACK_NAME=resma \
  -e APP_PORT=8080 \
  -e RESMA_COLLECT_INTERVAL=15 \
  -e RESMA_CLUSTER_INTERVAL=30 \
  -e RESMA_STORAGE_INTERVAL=60 \
  -e RESMA_AGENT_TASK_POLL_INTERVAL=15 \
  -e RESMA_ROLLBACK_POLL_INTERVAL=30 \
  -e RESMA_SCHEDULER_POLL=15 \
  -e RESMA_SSE_KEEPALIVE=15 \
  -e RESMA_RETENTION_DAYS=30 \
  -e RESMA_ANALYSIS_WINDOW_DAYS=7 \
  -e RESMA_STALE_SERVICE_DAYS=7 \
  resmaswarm/resma-install:latest
```

#### 8.3.2 Env vars expostas no upgrade

```bash
docker run -it --rm \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  -e MODE=upgrade \
  -e INTERACTIVE=0 \
  -e STACK_NAME=resma \
  -e RESMA_VERSION=v0.2.0 \
  -e RESMA_COLLECT_INTERVAL=10 \
  -e RESMA_STORAGE_INTERVAL=120 \
  resmaswarm/resma-install:latest
```

> **No upgrade**, apenas as env vars passadas são aplicadas (as demais permanecem com o valor atual do service). Isso segue o padrão `docker service update --env-add` — só sobrescreve o que for explicitamente passado.

#### 8.3.3 Tabela completa de parâmetros

| Parâmetro (env var) | Default produção | Descrição | Install | Upgrade |
|---------------------|------------------|-----------|---------|---------|
| `RESMA_COLLECT_INTERVAL` | `15` (segundos) | Intervalo de coleta de métricas de containers | ✅ | ✅ |
| `RESMA_CLUSTER_INTERVAL` | `30` (segundos) | Intervalo de coleta de info do cluster Swarm | ✅ | ✅ |
| `RESMA_STORAGE_INTERVAL` | `60` (segundos) | Intervalo de coleta de métricas de storage (volumes/discos) | ✅ | ✅ |
| `RESMA_AGENT_TASK_POLL_INTERVAL` | `15` (segundos) | Intervalo de poll de tasks do Swarm + health de agents | ✅ | ✅ |
| `RESMA_ROLLBACK_POLL_INTERVAL` | `30` (segundos) | Intervalo de poll do rollback watcher | ✅ | ✅ |
| `RESMA_SCHEDULER_POLL` | `15` (segundos) | Intervalo de poll do scheduler de agendamentos | ✅ | ✅ |
| `RESMA_SSE_KEEPALIVE` | `15` (segundos) | Intervalo de keepalive ping do SSE broker | ✅ | ✅ |
| `RESMA_RETENTION_DAYS` | `30` (dias) | Dias de métricas mantidos antes da purga | ✅ | ✅ |
| `RESMA_ANALYSIS_WINDOW_DAYS` | `7` (dias) | Janela de dados usada pela análise de ML | ✅ | ✅ |
| `RESMA_STALE_SERVICE_DAYS` | `7` (dias) | Dias sem heartbeat para marcar serviço como stale | ✅ | ✅ |

#### 8.3.4 Implementação no installer

**`app/installer/install.sh`** — ler env vars e injetar no `docker stack deploy`:

```bash
# ---------- defaults de produção ----------
RESMA_COLLECT_INTERVAL="${RESMA_COLLECT_INTERVAL:-15}"
RESMA_CLUSTER_INTERVAL="${RESMA_CLUSTER_INTERVAL:-30}"
RESMA_STORAGE_INTERVAL="${RESMA_STORAGE_INTERVAL:-60}"
RESMA_AGENT_TASK_POLL_INTERVAL="${RESMA_AGENT_TASK_POLL_INTERVAL:-15}"
RESMA_ROLLBACK_POLL_INTERVAL="${RESMA_ROLLBACK_POLL_INTERVAL:-30}"
RESMA_SCHEDULER_POLL="${RESMA_SCHEDULER_POLL:-15}"
RESMA_SSE_KEEPALIVE="${RESMA_SSE_KEEPALIVE:-15}"
RESMA_RETENTION_DAYS="${RESMA_RETENTION_DAYS:-30}"
RESMA_ANALYSIS_WINDOW_DAYS="${RESMA_ANALYSIS_WINDOW_DAYS:-7}"
RESMA_STALE_SERVICE_DAYS="${RESMA_STALE_SERVICE_DAYS:-7}"

# ---------- inject no docker stack deploy ----------
# O docker-stack.yml já referencia estas env vars; o installer só precisa
# exportá-las antes do deploy (docker stack deploy herda env do shell).
export RESMA_COLLECT_INTERVAL RESMA_CLUSTER_INTERVAL RESMA_STORAGE_INTERVAL \
       RESMA_AGENT_TASK_POLL_INTERVAL RESMA_ROLLBACK_POLL_INTERVAL \
       RESMA_SCHEDULER_POLL RESMA_SSE_KEEPALIVE \
       RESMA_RETENTION_DAYS RESMA_ANALYSIS_WINDOW_DAYS RESMA_STALE_SERVICE_DAYS

docker stack deploy -c "$COMPOSE_FILE" "$STACK_NAME"
```

**`app/installer/upgrade.sh`** — aplicar apenas env vars passadas via `docker service update --env-add`:

```bash
# Para cada env var passada, atualizar o service correspondente.
# Se a env var não foi passada, não fazer nada (preserva valor atual).
update_env() {
  local svc="$1"      # ex: api
  local var="$2"      # ex: RESMA_COLLECT_INTERVAL
  local val="$3"      # ex: 10
  echo "  docker service update --env-add ${var}=${val} ${STACK_NAME}_${svc}"
  docker service update --env-add "${var}=${val}" "${STACK_NAME}_${svc}"
}

# Aplicar apenas se a env var foi explicitamente passada (não vazia)
[[ -n "${RESMA_COLLECT_INTERVAL:-}" ]] && update_env api    RESMA_COLLECT_INTERVAL    "$RESMA_COLLECT_INTERVAL"
[[ -n "${RESMA_CLUSTER_INTERVAL:-}" ]]  && update_env api    RESMA_CLUSTER_INTERVAL     "$RESMA_CLUSTER_INTERVAL"
[[ -n "${RESMA_STORAGE_INTERVAL:-}" ]]  && update_env api    RESMA_STORAGE_INTERVAL     "$RESMA_STORAGE_INTERVAL"
# ... (uma linha por env var)
```

#### 8.3.5 Modo interativo

No modo interativo (`INTERACTIVE=1`, default), o installer deve **perguntar** se o usuário quer customizar intervalos antes de usar os defaults de produção:

```text
Application setup

  Enter stack name [resma]:
  Enter application port [8080]:

  Collection intervals (press Enter for production defaults):
    Collect interval in seconds [15]:
    Cluster interval in seconds [30]:
    Storage interval in seconds [60]:
    Agent task poll interval in seconds [15]:
    Rollback poll interval in seconds [30]:
    Retention days [30]:
    Analysis window days [7]:
    Stale service days [7]:

  Customize advanced intervals? (y/N) [N]:
    Scheduler poll interval in seconds [15]:
    SSE keepalive interval in seconds [15]:
```

> **UX:** intervalos avançados (scheduler, SSE keepalive) ficam atrás de um sub-prompt "Customize advanced? (y/N)" para não sobrecarregar o usuário comum. Defaults de produção são sugeridos entre colchetes.

#### 8.3.6 Validação de ranges

O installer deve validar ranges mínimos/máximos antes de aplicar (evita misconfiguration):

| Env var | Mínimo | Máximo | Validação |
|---------|--------|--------|-----------|
| `RESMA_COLLECT_INTERVAL` | 5s | 3600s (1h) | `if [[ $v -lt 5 || $v -gt 3600 ]]; then error; exit 1; fi` |
| `RESMA_CLUSTER_INTERVAL` | 5s | 3600s | idem |
| `RESMA_STORAGE_INTERVAL` | 10s | 3600s | mínimo 10s (df tem custo de I/O) |
| `RESMA_AGENT_TASK_POLL_INTERVAL` | 5s | 300s | máximo 300s (staleness de tasks) |
| `RESMA_ROLLBACK_POLL_INTERVAL` | 10s | 300s | mínimo 10s (rollback reativo) |
| `RESMA_SCHEDULER_POLL` | 5s | 300s | — |
| `RESMA_SSE_KEEPALIVE` | 5s | 60s | máximo 60s (proxies cortam idle) |
| `RESMA_RETENTION_DAYS` | 1 | 3650 (10 anos) | — |
| `RESMA_ANALYSIS_WINDOW_DAYS` | 1 | 365 | — |
| `RESMA_STALE_SERVICE_DAYS` | 1 | 365 | — |

#### 8.3.7 Arquivos a alterar

- `app/installer/install.sh` — ler 10 env vars + defaults de produção + export antes do deploy + prompts interativos + validação de ranges
- `app/installer/upgrade.sh` — aplicar apenas env vars passadas via `docker service update --env-add`
- `app/installer/docker-stack.yml` — garantir que todas as env vars são referenciadas com `${VAR:-default}` (já é o padrão atual)
- `app/installer/README.md` (se existir) ou `docs/installation.md` — documentar todos os parâmetros

#### 8.3.8 Validação

```bash
# Testar install não-interativo com intervals customizados
docker run -it --rm \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  -e INTERACTIVE=0 \
  -e RESMA_COLLECT_INTERVAL=10 \
  -e RESMA_STORAGE_INTERVAL=120 \
  resmaswarm/resma-install:latest

# Verificar se os valores foram aplicados
docker service inspect resma_api --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' | grep RESMA_
```

---

## 9. Referências

- Grafana `TimeSrv.ts`: https://github.com/grafana/grafana/blob/main/public/app/features/dashboard/services/TimeSrv.ts
- Grafana `SceneRefreshPicker.tsx`: https://github.com/grafana/scenes/blob/main/packages/scenes/src/components/SceneRefreshPicker.tsx
- Grafana docs (context7): dashboard refresh, scrape interval, alert evaluation
- Prometheus cAdvisor guide: https://prometheus.io/docs/guides/cadvisor/ (scrape_interval: 5s)
- Prometheus node_exporter guide: https://prometheus.io/docs/guides/node-exporter/ (scrape_interval: 15s)
- node_exporter PR #1557: "recommended 15s"
- node_exporter PR #1995: "default 60s, 5m rate for interoperability"
- Grafana issue #70479: "auto refresh based on query range"
- Robust Perception: "Keep it simple — 10-60s, pick one"
- Red Hat OpenShift blog: "scrape intervals 30s defaults"
- prometheus-users: "max 5m staleness, recommended 2-2.5m"
- Grafana troubleshooting: "refresh 1m or longer when data changes infrequently"
- Docker install script: DeepWiki docker/docker-install (flags CLI + env vars)
- Docker Desktop MSI: docs.docker.com/enterprise (propriedades CLI)
- Grafana CLI: grafana.com/docs/administration/cli (`--configOverrides` + env vars `GF_*`)
- Grafana Helm chart: grafana-community/helm-charts (`--set key=value`)
