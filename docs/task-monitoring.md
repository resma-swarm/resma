# Task Monitoring — Lifecycle de Tasks do Docker Swarm

> **Fase 7** — Monitoramento de tasks do Swarm em tempo real.

## Visão Geral

O Docker Swarm gerencia serviços como coleções de **tasks**. Cada task
representa uma unidade de trabalho (1 container) que o Swarm agenda em um node.
O RESMA monitora o lifecycle dessas tasks para detectar:

- **Restarts** — tasks que reiniciaram (transições para `running`)
- **Falhas** — tasks com status `failed` ou `rejected`
- **Pending** — tasks aguardando agendamento
- **Distribuição** — quantas tasks cada node está rodando

## Lifecycle de uma Task

```
pending → assigned → accepted → preparing → starting → running → complete
                    ↓                                        ↓
                rejected                                  shutdown
                    ↓                                        ↓
                  failed                                  orphaned
```

### Status do Swarm

| Status | Descrição | Badge UI |
|--------|-----------|----------|
| `running` | Container rodando normalmente | Verde "Running" |
| `pending` | Aguardando agendamento | Amarelo "Pending" |
| `assigned` | Atribuída a um node | Amarelo "Assigned" |
| `accepted` | Node aceitou a task | Amarelo "Accepted" |
| `preparing` | Preparando container | Azul "Preparing" |
| `starting` | Iniciando container | Azul "Starting" |
| `complete` | Finalizada com sucesso | Verde "Complete" |
| `failed` | Falhou | Vermelho "Failed" |
| `rejected` | Rejeitada pelo node | Vermelho "Rejected" |
| `shutdown` | Encerrada | Cinza "Shutdown" |
| `orphaned` | Órfã (node indisponível) | Laranja "Orphaned" |

## Coleta de Tasks

O collector do Go API faz poll de tasks via Docker SDK a cada
`RESMA_AGENT_TASK_POLL_INTERVAL` (default: 15s):

```go
// app/api/internal/collector/collector.go
func (c *Collector) taskCollectLoop(ctx) {
    ticker := time.NewTicker(c.cfg.TaskPollInterval)
    for {
        tasks, _ := c.docker.GetTasks(ctx)
        c.updateTaskCache(tasks)
        c.publishTaskEvent(tasks)  // SSE "tasks" topic
    }
}
```

### Task Cache

O collector mantém um cache em memória das tasks ativas para enriquecer
métricas com `node_id`, `task_id` e `slot`:

```go
type taskCache struct {
    mu    sync.RWMutex
    tasks map[string]TaskInfo  // container_id -> TaskInfo
}

type TaskInfo struct {
    TaskID       string
    Service      string
    NodeID       string
    Slot         int
    DesiredState string
}
```

Quando o collector coleta métricas locais (modo híbrido), ele consulta o cache
para anexar `node_id` e `task_id` a cada métrica.

## Service Health

O endpoint `GET /api/services/health` agrega o health de cada serviço:

```json
[
  {
    "service": "my-app",
    "tasks_running": 3,
    "tasks_failed": 0,
    "tasks_pending": 0,
    "restarts": 2,
    "last_restart_at": "2026-07-25T06:20:00Z"
  }
]
```

### Cálculo de Restarts

Restarts são calculados contando transições para `running` nos últimos N dias
(default: 7). Cada vez que uma task muda de status para `running` (após ter
sido `failed`, `shutdown` ou `complete`), é contada como 1 restart.

```sql
-- Simplificado: conta transições para running nos últimos 7 dias
SELECT service, COUNT(*) as restarts
FROM task_history
WHERE status = 'running'
  AND ts > now() - INTERVAL '7 days'
GROUP BY service
```

## Endpoints

### Listar Tasks

```
GET /api/tasks
```

Retorna todas as tasks com hostname do node (LEFT JOIN nodes):

```json
[
  {
    "task_id": "abc123def456",
    "service": "my-app",
    "node_id": "worker-node-1",
    "node_hostname": "worker-1",
    "slot": 1,
    "status": "running",
    "desired_state": "running",
    "created_at": "2026-07-25T06:00:00Z",
    "updated_at": "2026-07-25T06:05:00Z"
  }
]
```

### Tasks por Serviço

```
GET /api/tasks/{service}
```

Retorna apenas as tasks do serviço especificado.

### Histórico de Tasks

```
GET /api/tasks/{service}/history?days=7
```

Retorna o histórico de mudanças de status das tasks do serviço:

```json
[
  {
    "ts": "2026-07-25T06:05:00Z",
    "task_id": "abc123def456",
    "status": "running",
    "slot": 1,
    "node_id": "worker-node-1"
  }
]
```

### Service Health

```
GET /api/services/health?days=7
```

Retorna o health agregado de todos os serviços com tasks ativas.

## Frontend

### Página Tasks

A página `/tasks` mostra todas as tasks do Swarm em uma tabela com:
- Serviço, Slot, Status, Desired State, Node, Task ID
- Filtros por serviço e status
- Busca por serviço, node ou task ID
- Stats cards: Tasks total, Running, Failed, Serviços

### Tab Tasks no ServiceDetail

A página de detalhe de serviço (`/services/{name}`) tem uma tab "Tasks"
condicional que aparece apenas se o serviço tiver tasks ativas. A tab mostra:
- Service health summary (running/failed/pending/restarts)
- Tabela de tasks do serviço

### SSE

- Tópico `tasks` — notifica mudanças no lifecycle de tasks
- Invalida queries `["tasks"]` e `["services-health"]`
- O collector publica eventos quando detecta mudanças no status das tasks

## Debugging

### Verificar tasks no banco

```bash
# Login (owner criado via onboarding — ver docs/README.md)
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"owner","password":"owner123"}' | jq -r .access_token

# Listar tasks
curl -s -H "Authorization: Bearer <token>" http://localhost:8080/api/tasks | jq .

# Service health
curl -s -H "Authorization: Bearer <token>" http://localhost:8080/api/services/health | jq .
```

### Verificar agents

```bash
# Listar agents
curl -s -H "Authorization: Bearer <token>" http://localhost:8080/api/agents | jq .

# Detalhe de um agent
curl -s -H "Authorization: Bearer <token>" http://localhost:8080/api/agents/<node_id> | jq .
```
