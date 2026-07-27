# SSE Pattern — Padrão Único para Todas as Páginas

> Documento de referência que define o padrão SSE (Server-Sent Events) para o frontend RESMA.
> Todas as specs de páginas SSE devem seguir este padrão.

## 1. Estratégia Híbrida

```
┌─────────────┐     EventSource      ┌─────────────┐     build*      ┌────────┐
│  Frontend    │ ←──────────────────→ │  Go API     │ ──────────────→ │ DuckDB │
│  (React)     │   cookie auth (SSE)  │  + Broker   │                │        │
│              │                      │  + Collector│                └────────┘
│  useQuery    │  GET inicial (HTTP)  │             │
│  + SSE cache │  setQueryData (SSE)  │             │
│  + polling   │  invalidate (SSE)    │             │
└─────────────┘                      └─────────────┘
```

**3 camadas de atualização:**

1. **GET inicial** — `useQuery` faz HTTP GET na carga da página
2. **SSE real-time** — `useEventSource` recebe payload completo e atualiza cache via `setQueryData` (zero refetch)
3. **Polling fallback** — `refetchInterval` ativa apenas quando SSE está desconectado

## 2. Regras Obrigatórias

### 2.1 Hook centralizado

Toda página que precisa de real-time usa `useEventSource` de `@/hooks/use-event-source`.

```tsx
const { isConnected: sseConnected } = useEventSource({
  topic: "services",
  invalidateQueries: [["services"], ["recs-summary"]],
})
```

**Proibido:** criar `EventSource` manualmente.

### 2.2 Polling fallback

```tsx
const fallbackInterval = sseConnected ? false : refreshInterval
```

- SSE ativo → `refetchInterval: false` (zero polling)
- SSE inativo → `refetchInterval: refreshInterval` (polling normal)

**Exceção:** páginas de detalhe com muitas queries derivadas podem usar `refetchInterval: 300_000` (5min) como safety net extra quando SSE está ativo. Mas o padrão é `false`.

### 2.3 setQueryData vs invalidateQueries

| Cenário | Mecanismo | Condição |
|---------|-----------|----------|
| Payload SSE = mesmo do GET | `setQueryData` via `EVENT_QUERY_MAP` | Query montada (`oldData !== undefined`) |
| Payload SSE ≠ dado da query | `invalidateQueries` | Sempre |
| Sem payload SSE para a query | `invalidateQueries` | Sempre |
| Query não montada | Bail out (não criar ghost cache) | `oldData === undefined` |

### 2.4 invalidateQueries da página vs TOPIC_QUERY_MAP

- **`invalidateQueries` (página):** queries específicas da página que precisam de refetch
- **`TOPIC_QUERY_MAP` (hook):** queries globais do tópico (definido no hook)
- **`RECONCILE_QUERY_MAP` (hook):** queries derivadas/lentas que só precisam de refresh 30s

A página só deve listar em `invalidateQueries` as queries que:
1. Não estão cobertas por `setQueryData` (não estão no `EVENT_QUERY_MAP`)
2. São específicas da página (ex: `["node-detail", nodeId]`)
3. Precosam de refetch a cada evento SSE

### 2.5 Coalescing global

Invalidações de todos os hooks são agrupadas em 1 flush a cada 2s (`COALESCE_INTERVAL_MS`). Não é necessário coalescing por página.

### 2.6 Reconciliação 30s

O hook faz reconciliação periódica (30s) que invalida `TOPIC_QUERY_MAP[topic] + RECONCILE_QUERY_MAP[topic]`. Isso é um safety net para corrigir drift entre SSE e DB.

### 2.7 Auth via cookie

SSE usa cookie HttpOnly (`sse_session`, 10min TTL). O `sseSessionManager` faz refresh proativo antes de expirar e reativo antes de reconectar.

### 2.8 Exponential backoff

Reconexão com backoff exponencial: 1s, 2s, 4s, 8s, 16s, max 30s. Antes de reconectar, renova o cookie SSE.

## 3. Tópicos SSE e Event Types

| Tópico SSE | Event type(s) | Payload | Cadência |
|------------|---------------|---------|----------|
| `metrics` | `metrics` | `buildDashboardData` | ~5s |
| `dashboard` | `cluster` | `buildDashboardData` | ~30s |
| `dashboard` | `storage` | `buildStorageSummary` | ~60s |
| `services` | `services` | `buildServicesList` | ~5s |
| `nodes` | `nodes` | `buildNodesList` | ~5s |
| `tasks` | `tasks` | `buildTasksList` | ~15s (só se houver mudança) |
| `agents` | `stale` | `buildAgentsList` | ~15s (só se houver stale) |
| `events` | `oom` | `{service, container}` (signal) | on-demand |
| `change-log` | `schedule` | `buildSchedulesList` | on-demand (scheduler) |
| `change-log` | `change-log` | `buildChangeLog` | on-demand (scheduler) |
| `service-detail/{name}` | `service-detail` | `BuildServiceDetailData` | ~5s (só se subscriber) |
| `container-detail/{id}` | `container-detail` | `BuildContainerDetailData` | ~5s (só se subscriber) |

## 4. EVENT_QUERY_MAP (setQueryData)

| Event type | Query key | Payload = GET? |
|------------|-----------|----------------|
| `metrics` | `["dashboard"]` | Sim |
| `agent_metrics` | `["dashboard"]` | Sim |
| `services` | `["services"]` | Sim |
| `nodes` | `["nodes"]` | Sim |
| `cluster` | `["dashboard"]` | Sim |
| `storage` | `["storage-summary"]` | Sim |
| `tasks` | `["tasks"]` | Sim |
| `stale` | `["agents"]` | Sim |
| `heartbeat` | `["agents"]` | Sim |
| `oom` | `["oom-events"]` | ⚠️ Verificar — signal vs payload completo |
| `schedule` | `["schedules"]` | Sim |
| `change-log` | `["change-log"]` | Sim |
| `service-detail` | Múltiplas (ver `applyServiceDetailPayload`) | Sim |
| `container-detail` | Múltiplas (ver `applyContainerDetailPayload`) | Sim |

## 5. TOPIC_QUERY_MAP (invalidate por tópico)

| Tópico | Queries invalidadas |
|--------|---------------------|
| `metrics` | `["dashboard"]` |
| `dashboard` | `["dashboard"], ["storage-summary"]` |
| `events` | `["oom-events"], ["change-log"]` |
| `services` | `["services"]` |
| `nodes` | `["nodes"], ["cluster"]` |
| `tasks` | `["tasks"], ["services-health"]` |
| `agents` | `["agents"], ["nodes"]` |
| `change-log` | `["schedules"], ["change-log"]` |
| `service-detail` | `[]` (tudo via setQueryData) |
| `container-detail` | `[]` (tudo via setQueryData) |

## 6. RECONCILE_QUERY_MAP (refresh 30s)

| Tópico | Queries adicionais 30s |
|--------|------------------------|
| `services` | `["recs-summary"]` |

## 7. Checklist de Conformidade

Para cada página, verificar:

- [ ] **HOOK** — Usa `useEventSource` (não cria EventSource manualmente)
- [ ] **GET INICIAL** — `useQuery` faz HTTP GET na carga
- [ ] **POLLING FALLBACK** — `sseConnected ? false : refreshInterval`
- [ ] **INVALIDATE QUERIES** — Lista queries da página não cobertas por setQueryData
- [ ] **QUERY MONTADA** — Queries da página têm `refetchInterval: fallbackInterval`
- [ ] **SEM GHOST CACHE** — Não chama `setQueryData` manualmente (delegado ao hook)
- [ ] **SEM console.debug** — Sem logs de debug em produção
- [ ] **TOPIC CORRETO** — Assina o tópico SSE correto para os dados da página

## 8. Padrão por Tipo de Página

### 8.1 Página de lista (List Page)

```tsx
const { isConnected: sseConnected } = useEventSource({
  topic: "services",
  invalidateQueries: [["services"], ["recs-summary"]],
})
const fallbackInterval = sseConnected ? false : refreshInterval

const { data } = useQuery({
  queryKey: ["services"],
  queryFn: () => api.get("/services"),
  refetchInterval: fallbackInterval,
})
```

### 8.2 Página de detalhe (Detail Page)

```tsx
const { isConnected: sseConnected } = useEventSource({
  topic: `service-detail/${serviceName}`,
  enabled: !!serviceName,
})
const fallbackInterval = sseConnected ? false : refreshInterval

const { data } = useQuery({
  queryKey: ["service-stats", serviceName],
  queryFn: () => api.get(`/services/${serviceName}/stats`),
  refetchInterval: fallbackInterval,
})
```

### 8.3 Página derivada (Derived Page)

Páginas cujos dados são derivados de métricas (ex: Alerts) não têm payload SSE próprio. Assinam tópicos relacionados e usam `invalidateQueries` para refetch.

```tsx
const { isConnected: sseDashboard } = useEventSource({
  topic: "dashboard",
  invalidateQueries: [["alerts"]],
})
const { isConnected: sseMetrics } = useEventSource({
  topic: "metrics",
  invalidateQueries: [["alerts"]],
})
const sseConnected = sseDashboard || sseMetrics
const fallbackInterval = sseConnected ? false : refreshInterval
```
