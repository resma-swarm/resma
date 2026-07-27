# Multi-Node Agent — Coleta de Stats em Swarm com N Workers

> **Fase 7** — RESMA Agent para coleta de métricas em todos os nodes do Docker Swarm.

## Visão Geral

O RESMA roda como 1 container no manager node do Docker Swarm com acesso ao
Docker socket local (`/var/run/docker.sock`). O collector do Go API usa
`ContainerList` + `ContainerStats` da Docker API, que são **chamadas locais** —
só retornam containers do daemon onde o socket está montado.

Em um Swarm com 1 manager + N workers, o RESMA sem o Agent coleta stats de
apenas 1/(N+1) dos containers. As recomendações ML seriam baseadas em dados
parciais e não confiáveis. OOMs em workers são invisíveis.

### Solução: RESMA Agent

Um binary Go leve (~6 MB) que roda como **global service** no Swarm (1 por node).
Cada agent coleta stats dos containers locais via Docker socket e faz HTTP POST
para o Go API no manager node.

```
┌─────────────────────────────────────────────────────────────────┐
│                    Docker Swarm Cluster                         │
│                                                                 │
│  ┌──────────────┐         ┌──────────────┐                      │
│  │  Manager     │         │  Worker 1    │                      │
│  │  Go API      │◄────────│  Agent       │ HTTP POST /api/agent │
│  │  Collector   │         │  (local sock)│ /heartbeat           │
│  │  DuckDB      │         └──────────────┘ /ingest/metrics      │
│  │  Frontend    │                         /ingest/oom           │
│  └──────┬───────┘         ┌──────────────┐                      │
│         │                 │  Worker 2    │                      │
│        ◄──────────────────│  Agent       │                      │
│         │                 └──────────────┘                      │
│         │                 ┌──────────────┐                      │
│        ◄──────────────────│  Worker N    │                      │
│                           └──────────────┘                      │
└─────────────────────────────────────────────────────────────────┘
```

## Arquitetura

### Componentes

| Componente | Localização | Função |
|------------|-------------|--------|
| **RESMA Agent** | `app/agent/` | Binary Go que coleta stats locais e faz push para o API |
| **Ingestion API** | `app/api/internal/server/agent_handlers.go` | Endpoints `/api/agent/*` para receber dados dos agents |
| **Agent DB** | `app/api/internal/db/agent_db.go` | Métodos para agents e tasks no DuckDB |
| **Task Handlers** | `app/api/internal/server/task_handlers.go` | Endpoints admin `/api/agents/*` e `/api/tasks/*` |
| **Collector híbrido** | `app/api/internal/collector/collector.go` | Modo híbrido: coleta local + agent health check |

### Protocolo

Cada agent faz 3 tipos de chamada para o Go API:

1. **Heartbeat** — `POST /api/agent/heartbeat` (a cada 60s)
   - Body: `{node_id, hostname, containers_count, version}`
   - Atualiza `agents` table com `last_heartbeat` e `status=active`

2. **Metrics** — `POST /api/agent/ingest/metrics` (a cada 1-5s)
   - Body: `{node_id, metrics: [{ts, service, container_id, cpu_percent, mem_usage, mem_limit, ...}]}`
   - Insere em `metrics` table com `node_id` e `task_id`

3. **OOM Events** — `POST /api/agent/ingest/oom` (event-driven)
   - Body: `{node_id, events: [{ts, container_id, service, exit_code}]}`
   - Insere em `oom_events` table com `node_id`

### Autenticação

- Agent token via header `Authorization: Bearer <token>` ou `X-Agent-Token: <token>`
- Token configurado via env `RESMA_AGENT_TOKEN`
- `agentTokenMiddleware` valida o token em todas as rotas `/api/agent/*`
- Rate limit: 100 req/s por agent (configurável)

### Schema (DuckDB)

Tabelas novas (Fase 7):

```sql
-- Agents registrados (1 por node)
CREATE TABLE IF NOT EXISTS agents (
    node_id VARCHAR PRIMARY KEY,
    hostname VARCHAR NOT NULL,
    version VARCHAR DEFAULT '',
    containers_count INTEGER DEFAULT 0,
    last_heartbeat TIMESTAMP,
    status VARCHAR DEFAULT 'active',  -- active | stale | offline
    first_seen TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

-- Tasks do Swarm (lifecycle)
CREATE TABLE IF NOT EXISTS tasks (
    task_id VARCHAR PRIMARY KEY,
    service VARCHAR NOT NULL,
    node_id VARCHAR,
    slot INTEGER DEFAULT 0,
    status VARCHAR DEFAULT 'pending',
    desired_state VARCHAR DEFAULT 'running',
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

Colunas adicionadas em `metrics`:
- `node_id VARCHAR` — ID do node onde a métrica foi coletada
- `task_id VARCHAR` — ID da task do Swarm (se aplicável)
- `slot INTEGER` — Slot da task no serviço

Coluna adicional em `oom_events`:
- `node_id VARCHAR` — ID do node onde o OOM ocorreu

## Deploy

### Docker Swarm (produção)

O `docker-stack.yml` inclui o serviço `resma-agent` com `mode: global` (1 por node):

```yaml
resma-agent:
  image: resma/agent:latest
  deploy:
    mode: global  # 1 agent por node
    placement:
      constraints: [node.role == worker]
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock:ro
  environment:
    - RESMA_API_URL=http://resma-api:8080
    - RESMA_AGENT_TOKEN={{.AgentToken}}
  networks:
    - resma-net
```

O agent token é injetado via Docker secret `resma_agent_token`.

### Dev (single-node)

```bash
# Sobe API + frontend + agent-dev (simula 1 worker node)
docker compose up -d
```

O `agent-dev` coleta stats dos containers locais (do host Docker) e faz push
para o `go-dev`. Em dev single-node, o agent-dev e o go-dev veem o mesmo
`docker.sock` — útil para validar o fluxo end-to-end.

### Dev (multi-node simulado)

```bash
# Sobe API + frontend + agent-dev + 2 workers adicionais
docker compose --profile multi-node \
  -f docker-compose.yml -f docker-compose.multi-node.yml \
  up -d agent-dev agent-worker-1 agent-worker-2
```

Cada worker agent usa um `node_id` distinto (`worker-node-1`, `worker-node-2`),
permitindo testar a UI com múltiplos agents.

## Configuração

### Variáveis de ambiente (Agent)

| Variável | Default | Descrição |
|----------|---------|-----------|
| `RESMA_API_URL` | `http://localhost:8080` | URL do Go API |
| `RESMA_AGENT_TOKEN` | (obrigatório) | Token de autenticação do agent |
| `RESMA_NODE_ID` | auto-detectado | ID do node (hostname em dev) |
| `RESMA_NODE_HOSTNAME` | auto-detectado | Hostname do node |
| `RESMA_COLLECT_INTERVAL` | `1` | Intervalo de coleta (segundos) |
| `RESMA_HEARTBEAT_INTERVAL` | `60` | Intervalo de heartbeat (segundos) |
| `RESMA_OOM_EVENTS_FILE` | `/data/events.pending.jsonl` | Buffer de OOM events (persistente) |

### Variáveis de ambiente (Go API)

| Variável | Default | Descrição |
|----------|---------|-----------|
| `RESMA_AGENT_TOKEN` | (obrigatório se agents ativos) | Token esperado pelos agents |
| `RESMA_AGENT_TASK_POLL_INTERVAL` | `15` | Intervalo de poll de tasks (segundos) |

## Endpoints

### Agent ingestion (token auth, `/api/agent/*`)

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/agent/heartbeat` | Heartbeat do agent |
| POST | `/api/agent/ingest/metrics` | Lote de métricas |
| POST | `/api/agent/ingest/oom` | Lote de OOM events |

### Admin (JWT auth, `/api/*`)

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/agents` | Lista todos os agents |
| GET | `/api/agents/{node_id}` | Detalhe de um agent |
| GET | `/api/tasks` | Lista todas as tasks |
| GET | `/api/tasks/{service}` | Tasks de um serviço |
| GET | `/api/tasks/{service}/history` | Histórico de mudanças de status |
| GET | `/api/services/health` | Health agregado por serviço |

### SSE (cookie auth, `/api/sse/*`)

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/sse/agents` | Stream de mudanças de agents |
| GET | `/api/sse/tasks` | Stream de mudanças de tasks |

## Frontend

### Páginas novas

- **Tasks** (`/tasks`) — Lista de tasks do Swarm com service, slot, status,
  desired_state, node_hostname. Filtros por serviço e status, busca, stats cards.
- **Agents** (`/agents`) — Lista de RESMA Agents com hostname, status
  (active/stale), containers_count, version, last_heartbeat (relative time),
  node_id. Filtro por status, busca, stats cards.

### Modificações

- **ServiceDetail** — Tab "Tasks" condicional (só mostra se há tasks do serviço),
  com service health summary (running/failed/pending/restarts) e tabela de tasks.
- **NodeDetail** — Card "RESMA Agent" condicional no tab Overview (status,
  containers, version, last heartbeat).

### SSE Topics

- `agents` — invalida queries `["agents"]` e `["nodes"]`
- `tasks` — invalida queries `["tasks"]` e `["services-health"]`

## Status do Agent

| Status | Condição | UI |
|--------|----------|-----|
| `active` | Heartbeat nos últimos 2x `heartbeat_interval` | Badge verde "Active" |
| `stale` | Sem heartbeat há mais de 2x `heartbeat_interval` | Badge laranja "Stale" |
| `offline` | Sem heartbeat há mais de 10x `heartbeat_interval` | Badge vermelho "Offline" |

O `agentHealthLoop` no collector verifica a cada 30s se algum agent está stale
e atualiza o status no banco.
