# Especificação Técnica — Nodes & Cluster Swarm

> Feature: Swarm Nodes & Cluster Info | Autor: Technical Council (Orchestrator) | Data: 2026-07-21
> Framework: AI Engineering Framework — Skill: technical-spec

---

## 1. Contexto do Projeto (para agente novo)

### O que é o RESMA

RESMA (RESource MAnager) é um aplicativo customizado para gerenciar recursos (CPU/memória) de containers no Docker Swarm. Roda como um único container no node manager com acesso ao Docker socket.

### Princípios

- "O básico que funciona" — sem over-engineering
- Self-contained — zero dependências externas (sem Prometheus, sem cAdvisor, sem Grafana)
- Coleta direto do Docker API via `aiodocker`
- Foco em gestão de recursos, não monitoramento genérico

### Stack

| Camada | Tecnologia |
|--------|-----------|
| Backend | Python 3.12+, FastAPI, aiodocker, DuckDB (embedded OLAP) |
| Frontend | React + Vite + TailwindCSS v4 + shadcn/ui + recharts |
| ML | scikit-learn, scipy, numpy (Z-score, Linear Regression, KMeans) |
| Deploy | Single container no Swarm manager node, acesso ao Docker socket |

### Arquitetura

```
RESMA (1 container)
├── Collector (aiodocker → Docker socket, coleta a cada RESMA_COLLECT_INTERVAL segundos)
├── DuckDB (embedded, retention 30 dias)
├── Recommender (percentis + ML simples, on-demand)
├── Templates (YAML versionado)
├── FastAPI (API + serve frontend SPA)
└── React SPA (dashboard, services, recommendations, templates)
```

### Estrutura de pastas

```
resma/
├── backend/
│   ├── core/
│   │   ├── config.py          # Settings (env vars com prefixo RESMA_)
│   │   ├── db.py              # Database (DuckDB, schema, queries)
│   │   └── docker_client.py   # DockerClient (aiodocker wrapper)
│   ├── routers/
│   │   ├── dashboard.py       # GET /api/dashboard
│   │   ├── services.py        # GET/PATCH /api/services/*
│   │   ├── recommendations.py # GET/POST /api/recommendations/*
│   │   ├── templates.py       # CRUD /api/templates
│   │   ├── oom.py             # GET /api/oom
│   │   ├── auth.py            # POST /api/auth/*
│   │   └── config.py          # GET /api/config
│   ├── services/
│   │   ├── collector.py       # MetricsCollector (loop de coleta)
│   │   └── auth.py            # Auth service (JWT, bcrypt)
│   ├── models.py              # Pydantic models
│   └── main.py                # FastAPI app entry point
├── frontend/
│   ├── src/
│   │   ├── api/client.ts      # API client (fetch wrapper)
│   │   ├── components/
│   │   │   ├── Layout.tsx     # Sidebar + breadcrumbs + refresh
│   │   │   ├── ui/            # shadcn/ui components
│   │   │   ├── empty-state.tsx
│   │   │   └── page-header.tsx
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Services.tsx
│   │   │   ├── ServiceDetail.tsx
│   │   │   ├── ContainerDetail.tsx
│   │   │   ├── Recommendations.tsx
│   │   │   └── Templates.tsx
│   │   ├── hooks/use-refresh.ts
│   │   ├── stores/refresh-store.ts
│   │   ├── contexts/AuthContext.tsx
│   │   └── lib/utils.ts       # formatBytes, formatCPU
│   └── src/App.tsx            # React Router setup
└── docs/
    └── IMPLEMENTATION_PROMPT.md
```

### Padrões existentes (SEGUIR estes)

**Backend:**
- Routers em `backend/routers/`, prefixo `/api/`, com `Depends(get_current_user)`
- Queries DuckDB via `db.fetchall()` / `db.fetchone()` com f-string para INTERVAL e `?` para params
- DockerClient em `backend/core/docker_client.py` com métodos async, singleton `docker_client`
- Collector em `backend/services/collector.py` com loop async, tasks criadas em `start()`
- Config em `backend/core/config.py` com `pydantic_settings.BaseSettings`, prefixo `RESMA_`
- Logger: `logging.getLogger("resma.<module>")`

**Frontend:**
- Páginas em `frontend/src/pages/`, componentes em `frontend/src/components/`
- API client: `api.get<T>(path)`, `api.post(path, body)`, `api.patch(path)`
- Data fetching: `useQuery` do `@tanstack/react-query` com `refetchInterval` do `useRefreshInterval()`
- Charts: `recharts` (BarChart, AreaChart, ResponsiveContainer)
- UI: shadcn/ui (Card, Table, Badge, Progress, Tabs, Skeleton, Alert, DropdownMenu, Tooltip)
- Ícones: `lucide-react`
- Toast: `sonner`
- Cores CSS vars: `--chart-1` (CPU/laranja), `--chart-2` (memória/azul), `--chart-3` (drift), `--chart-4` (leak), `--chart-5` (templates), `--primary`, `--destructive`, `--warning`, `--success`
- Navegação: `react-router-dom` com `NavLink` no sidebar (`Layout.tsx`), rotas em `App.tsx`
- Breadcrumbs: função `buildBreadcrumbs()` em `Layout.tsx`
- Formatação: `formatBytes()`, `formatCPU()` de `@/lib/utils`

### Schema DuckDB existente (NÃO alterar)

```sql
-- Time-series (append-only, retention 30d)
CREATE TABLE metrics (
    ts TIMESTAMP, service VARCHAR, container_id VARCHAR,
    cpu_percent DOUBLE, cpu_usage BIGINT, cpu_system BIGINT,
    mem_usage BIGINT, mem_limit BIGINT, mem_percent DOUBLE,
    net_rx BIGINT, net_tx BIGINT, block_read BIGINT, block_write BIGINT
);

-- Event log (append-only, retention 30d)
CREATE TABLE oom_events (
    ts TIMESTAMP, service VARCHAR, container_id VARCHAR, exit_code INTEGER
);

-- State (upsert)
CREATE TABLE service_configs (
    service VARCHAR PRIMARY KEY, cpu_limit DOUBLE, mem_limit BIGINT,
    cpu_reservation DOUBLE, mem_reservation BIGINT, template VARCHAR, updated_at TIMESTAMP
);

-- State (upsert)
CREATE TABLE service_registry (
    service VARCHAR PRIMARY KEY, status VARCHAR DEFAULT 'active',
    last_seen TIMESTAMP, updated_at TIMESTAMP DEFAULT now()
);

-- Entity
CREATE TABLE users (id INTEGER PRIMARY KEY, username VARCHAR, password_hash VARCHAR, role VARCHAR, ...);
CREATE TABLE refresh_tokens (token VARCHAR PRIMARY KEY, user_id INTEGER, ...);
CREATE TABLE app_settings (key VARCHAR PRIMARY KEY, value VARCHAR);
CREATE TABLE templates (id INTEGER PRIMARY KEY, name VARCHAR, description VARCHAR, yaml_content TEXT, stacks VARCHAR, ...);
```

### Config existente (env vars)

| Variável | Default | Descrição |
|----------|---------|-----------|
| `RESMA_DB_PATH` | `./data/resma.duckdb` | Caminho do DuckDB |
| `RESMA_COLLECT_INTERVAL` | `15` | Intervalo de coleta em segundos |
| `RESMA_RETENTION_DAYS` | `30` | Dias de retenção |
| `RESMA_ANALYSIS_WINDOW_DAYS` | `7` | Janela de análise |
| `RESMA_ML_ENABLED` | `True` | Habilita ML nas recomendações |
| `RESMA_EXCLUDED_IMAGES` | `resma:latest` | Images excluídas da coleta |
| `RESMA_JWT_SECRET` | `""` | Secret do JWT |
| ... | ... | (ver `backend/core/config.py`) |

---

## 2. Visão Geral da Feature

Adicionar visibilidade de nodes e cluster do Docker Swarm ao RESMA, mantendo o foco em gestão de recursos. Hoje o RESMA coleta métricas de containers e recomenda limits por serviço, mas não enxerga o cluster — não há visão de nodes, capacidade total, distribuição de carga, nem status do Swarm.

### Objetivos

- **Visibilidade do cluster:** quantos nodes, managers vs workers, quorum status
- **Capacidade real por node:** CPU e memória total/disponível de cada servidor
- **Consumo agregado por node:** CPU/memória consumida pelos containers rodando em cada node (cruzando métricas existentes com mapping container→node)
- **Distribuição de carga:** tasks/containers por node — identifica desequilíbrio
- **Disponibilidade:** status (ready/down) + availability (active/pause/drain) por node
- **Placement:** labels por node para entender constraints

### Fora de escopo (Regression Boundary)

- **NÃO alterar** schema da tabela `metrics` existente
- **NÃO alterar** lógica de coleta de containers existente
- **NÃO alterar** recomendações existentes
- **NÃO alterar** auth, templates, ou qualquer router existente
- **NÃO implementar** topology visualizer (grid de boxes como Portainer)
- **NÃO implementar** rebalance suggestions
- **NÃO implementar** disk usage per node (Docker API não expõe confiavelmente)
- **NÃO implementar** network traffic per node (fora do foco CPU/memória)
- **NÃO implementar** log viewer

---

## 3. Modelo de Dados

### 3.1. Novas tabelas DuckDB

Adicionar em `backend/core/db.py` no método `_init_schema()`, **após** as tabelas existentes, **antes** de `_seed_templates()`.

#### Tabela: `nodes` (estado atual — upsert)

Segue o padrão de `service_registry`: PK + estado atual + `updated_at`.

```sql
CREATE TABLE IF NOT EXISTS nodes (
    node_id         VARCHAR PRIMARY KEY,
    hostname        VARCHAR,
    role            VARCHAR,
    availability    VARCHAR,
    status          VARCHAR,
    address         VARCHAR,
    cpu_total       DOUBLE,
    mem_total       BIGINT,
    os              VARCHAR,
    architecture    VARCHAR,
    engine_version  VARCHAR,
    is_leader       BOOLEAN,
    reachability    VARCHAR,
    labels          VARCHAR,
    tasks_running   INTEGER,
    updated_at      TIMESTAMP DEFAULT now()
)
```

**Campos:**
- `node_id`: Docker node ID (string longa, ex: `e216jshn25ckzbvmwlnh5jr3g`)
- `hostname`: `Description.Hostname`
- `role`: `Spec.Role` — `"manager"` ou `"worker"`
- `availability`: `Spec.Availability` — `"active"`, `"pause"`, `"drain"`
- `status`: `Status.State` — `"ready"`, `"down"`, `"disconnected"`, `"unknown"`
- `address`: `Status.Addr` (IP)
- `cpu_total`: `Description.Resources.NanoCPUs` / 1e9 (em cores)
- `mem_total`: `Description.Resources.MemoryBytes` (em bytes)
- `os`: `Description.Platform.OS`
- `architecture`: `Description.Platform.Architecture`
- `engine_version`: `Description.Engine.EngineVersion`
- `is_leader`: `ManagerStatus.Leader` se existir, senão `false`
- `reachability`: `ManagerStatus.Reachability` se existir, senão `null`
- `labels`: JSON string de `Spec.Labels` (ex: `{"storage":"ssd","env":"prod"}`)
- `tasks_running`: count de tasks com `NodeID == node_id` e `Status.State == "running"`
- `updated_at`: timestamp do último upsert

#### Tabela: `node_metrics` (time-series — append + retention 30d)

Segue o padrão de `metrics`: `ts` + dados por coleta, append-only.

```sql
CREATE TABLE IF NOT EXISTS node_metrics (
    ts              TIMESTAMP,
    node_id         VARCHAR,
    hostname        VARCHAR,
    role            VARCHAR,
    availability    VARCHAR,
    status          VARCHAR,
    cpu_total       DOUBLE,
    mem_total       BIGINT,
    tasks_running   INTEGER
)
```

**Racional:** Mesmos dados de estado + timestamp. Permite ver histórico de:
- Quando um node foi drenado (availability mudou de active → drain)
- Quando um node caiu (status mudou de ready → down)
- Evolução da distribuição de tasks (tasks_running ao longo do tempo)
- Capacidade do cluster ao longo do tempo (nodes adicionados/removidos)

#### Tabela: `cluster` (estado do Swarm — upsert)

Segue o padrão de `service_registry`: PK + estado atual + `updated_at`.

```sql
CREATE TABLE IF NOT EXISTS cluster (
    id              VARCHAR PRIMARY KEY,
    nodes_total     INTEGER,
    managers_total  INTEGER,
    workers_total   INTEGER,
    nodes_ready     INTEGER,
    nodes_down      INTEGER,
    quorum_healthy  BOOLEAN,
    self_node_id    VARCHAR,
    warnings        VARCHAR,
    updated_at      TIMESTAMP DEFAULT now()
)
```

**Campos:**
- `id`: `Swarm.ClusterID` ou `"default"` se não houver
- `nodes_total`: count total de nodes
- `managers_total`: count de nodes com `role == "manager"`
- `workers_total`: count de nodes com `role == "worker"`
- `nodes_ready`: count de nodes com `status == "ready"`
- `nodes_down`: count de nodes com `status != "ready"`
- `quorum_healthy`: `true` se `managers_total >= 1` e `Reachability == "reachable"` para o leader
- `self_node_id`: `Swarm.NodeID` (o node onde o RESMA roda)
- `warnings`: JSON string de `Swarm.Warnings` ou `null`
- `updated_at`: timestamp do último upsert

#### Tabela: `container_node_map` (mapping — upsert)

Mapeia container → node para permitir agregar métricas de container por node.

```sql
CREATE TABLE IF NOT EXISTS container_node_map (
    container_id    VARCHAR PRIMARY KEY,
    node_id         VARCHAR,
    service         VARCHAR,
    updated_at      TIMESTAMP DEFAULT now()
)
```

**Racional:** A tabela `metrics` existente tem `container_id` mas não `node_id`. Em vez de alterar o schema de `metrics` (breaking change), criamos uma tabela de mapping que é atualizada a cada coleta via `tasks.list()`. Isso permite queries como:

```sql
SELECT m.node_id, quantile(cpu_percent, 0.95) as cpu_p95
FROM metrics m
JOIN container_node_map c ON m.container_id = c.container_id
WHERE m.ts > now() - INTERVAL 7 DAYS
GROUP BY m.node_id
```

### 3.2. Retention

Adicionar em `db.run_retention()`:

```python
self._conn.execute(
    f"DELETE FROM node_metrics WHERE ts < now() - INTERVAL {settings.retention_days} DAYS"
)
```

**NOTA:** `nodes`, `cluster` e `container_node_map` são state tables (upsert) — **sem retention**. Sempre estado atual.

---

## 4. Configuração

### 4.1. Nova env var

Adicionar em `backend/core/config.py` na classe `Settings`:

```python
cluster_interval: int = 60
```

**Nome completo:** `RESMA_CLUSTER_INTERVAL` (default: 60 segundos)

**Racional:** Cluster info (quorum, manager count, warnings) muda raramente. Não precisa coletar a cada 15s como containers. 60s é suficiente e reduz carga na Docker API.

### 4.2. Env vars existentes (reutilizar, NÃO criar novas)

| Variável | Uso nesta feature |
|----------|-------------------|
| `RESMA_COLLECT_INTERVAL` | Intervalo para coleta de node state + node_metrics + container_node_map |
| `RESMA_RETENTION_DAYS` | Retention para `node_metrics` (mesmo valor do `metrics`) |
| `RESMA_ANALYSIS_WINDOW_DAYS` | Janela para queries de agregação por node |

---

## 5. Backend — DockerClient

Adicionar em `backend/core/docker_client.py` os seguintes métodos:

### 5.1. `get_nodes()`

```python
async def get_nodes(self) -> list[dict]:
```

**Docker API:** `self.client.nodes.list()`

**Retorno:** lista de dicts com:
```python
{
    "id": node["ID"],
    "hostname": node["Description"]["Hostname"],
    "role": node["Spec"]["Role"],
    "availability": node["Spec"]["Availability"],
    "status": node["Status"]["State"],
    "address": node["Status"].get("Addr", ""),
    "cpu_total": node["Description"]["Resources"]["NanoCPUs"] / 1e9,
    "mem_total": node["Description"]["Resources"]["MemoryBytes"],
    "os": node["Description"]["Platform"]["OS"],
    "architecture": node["Description"]["Platform"]["Architecture"],
    "engine_version": node["Description"]["Engine"]["EngineVersion"],
    "is_leader": node.get("ManagerStatus", {}).get("Leader", False),
    "reachability": node.get("ManagerStatus", {}).get("Reachability"),
    "labels": node["Spec"].get("Labels", {}) or {},
}
```

### 5.2. `get_node_detail(node_id)`

```python
async def get_node_detail(self, node_id: str) -> dict | None:
```

**Docker API:** `self.client.nodes.inspect(node_id)` ou buscar na lista

**Retorno:** dict completo do node (todos os campos de `get_nodes()` + `created_at`, `updated_at`)

### 5.3. `get_swarm_info()`

```python
async def get_swarm_info(self) -> dict:
```

**Docker API:** `self.client.system.info()` — extrair campos de Swarm

**Retorno:**
```python
{
    "cluster_id": info.get("Swarm", {}).get("Cluster", {}).get("ID", ""),
    "node_id": info.get("Swarm", {}).get("NodeID", ""),
    "nodes_total": info.get("Swarm", {}).get("Nodes", 0),
    "managers_total": info.get("Swarm", {}).get("Managers", 0),
    "workers_total": info.get("Swarm", {}).get("Nodes", 0) - info.get("Swarm", {}).get("Managers", 0),
    "control_available": info.get("Swarm", {}).get("ControlAvailable", False),
    "warnings": info.get("Swarm", {}).get("Warnings", []) or [],
    "remote_managers": [m.get("Addr") for m in info.get("Swarm", {}).get("RemoteManagers", []) or []],
}
```

### 5.4. `get_tasks_by_node()`

```python
async def get_tasks_by_node(self) -> dict[str, list[dict]]:
```

**Docker API:** `self.client.tasks.list()`

**Retorno:** dict mapeando `node_id → lista de tasks`:
```python
{
    "node_id_1": [
        {"id": "...", "service_id": "...", "service_name": "...", "state": "running", ...},
        ...
    ],
    ...
}
```

**Notas:**
- Cada task tem `NodeID` e `ServiceID`
- Para obter o service name, cruzar com `self.client.services.list()` ou usar o `service_registry` do DuckDB
- Filtrar apenas tasks com `Status.State == "running"` para o count de `tasks_running`

### 5.5. `get_container_node_mapping()`

```python
async def get_container_node_mapping(self) -> list[dict]:
```

**Docker API:** `self.client.tasks.list()` cruzado com containers

**Retorno:** lista de dicts:
```python
[
    {"container_id": "...", "node_id": "...", "service": "service_name"},
    ...
]
```

**Lógica:**
1. Listar tasks: `self.client.tasks.list()`
2. Para cada task com `Status.State == "running"`:
   - `container_id` = `task["Status"]["ContainerStatus"]["ContainerID"]` (se disponível)
   - `node_id` = `task["NodeID"]`
   - `service` = nome do serviço (cruzar `task["ServiceID"]` com services list)
3. Retornar lista

**Importante:** Nem toda task tem `ContainerID` disponível. Filtrar `None`.

---

## 6. Backend — Database (db.py)

### 6.1. Schema (em `_init_schema()`)

Adicionar as 4 tabelas novas (ver seção 3.1) após as existentes, antes de `_seed_templates()`.

### 6.2. Métodos novos

#### `upsert_node()`

```python
def upsert_node(self, node_id, hostname, role, availability, status, address,
                cpu_total, mem_total, os, architecture, engine_version,
                is_leader, reachability, labels, tasks_running):
```

**SQL:**
```sql
INSERT OR REPLACE INTO nodes
(node_id, hostname, role, availability, status, address, cpu_total, mem_total,
 os, architecture, engine_version, is_leader, reachability, labels, tasks_running, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())
```

#### `insert_node_metrics_batch()`

```python
def insert_node_metrics_batch(self, rows: list[list]):
```

**SQL:** `INSERT INTO node_metrics VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)` via `executemany`

Segue o padrão de `insert_metrics_batch()`.

#### `upsert_cluster()`

```python
def upsert_cluster(self, cluster_id, nodes_total, managers_total, workers_total,
                   nodes_ready, nodes_down, quorum_healthy, self_node_id, warnings):
```

**SQL:**
```sql
INSERT OR REPLACE INTO cluster
(id, nodes_total, managers_total, workers_total, nodes_ready, nodes_down,
 quorum_healthy, self_node_id, warnings, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, now())
```

#### `upsert_container_node_map_batch()`

```python
def upsert_container_node_map_batch(self, rows: list[list]):
```

**SQL:** Para cada row: `INSERT OR REPLACE INTO container_node_map (container_id, node_id, service, updated_at) VALUES (?, ?, ?, now())`

#### `get_nodes()`

```python
def get_nodes(self) -> list[dict]:
```

**SQL:** `SELECT * FROM nodes ORDER BY role, hostname`

Retorna lista de dicts com todos os campos da tabela `nodes`.

#### `get_node_by_id()`

```python
def get_node_by_id(self, node_id: str) -> dict | None:
```

**SQL:** `SELECT * FROM nodes WHERE node_id = ?`

#### `get_cluster()`

```python
def get_cluster(self) -> dict | None:
```

**SQL:** `SELECT * FROM cluster ORDER BY updated_at DESC LIMIT 1`

#### `get_node_metrics()`

```python
def get_node_metrics(self, node_id: str, days: int = 7) -> list[dict]:
```

**SQL:**
```sql
SELECT ts, tasks_running, cpu_total, mem_total
FROM node_metrics
WHERE node_id = ? AND ts > now() - INTERVAL {days} DAYS
ORDER BY ts
```

#### `get_node_consumption()`

```python
def get_node_consumption(self, node_id: str) -> dict:
```

**SQL:** Agrega métricas de containers rodando no node, cruzando com `container_node_map`:
```sql
SELECT
    quantile(m.cpu_percent, 0.95) as cpu_p95,
    quantile(m.mem_usage, 0.99) as mem_p99,
    sum(m.mem_usage) as mem_total_usage,
    count(DISTINCT m.container_id) as containers
FROM metrics m
JOIN container_node_map c ON m.container_id = c.container_id
WHERE c.node_id = ? AND m.ts > now() - INTERVAL {settings.analysis_window_days} DAYS
```

**Retorno:**
```python
{
    "cpu_p95": round(row[0], 2),
    "mem_p99": int(row[1]),
    "mem_total_usage": int(row[2]),
    "containers": row[3],
}
```

#### `get_node_services()`

```python
def get_node_services(self, node_id: str) -> list[dict]:
```

**SQL:** Lista services rodando no node com agregação de consumo:
```sql
SELECT
    c.service,
    count(DISTINCT m.container_id) as containers,
    quantile(m.cpu_percent, 0.95) as cpu_p95,
    quantile(m.mem_usage, 0.99) as mem_p99
FROM metrics m
JOIN container_node_map c ON m.container_id = c.container_id
WHERE c.node_id = ? AND m.ts > now() - INTERVAL {settings.analysis_window_days} DAYS
GROUP BY c.service ORDER BY cpu_p95 DESC
```

#### `get_cluster_summary()`

```python
def get_cluster_summary(self) -> dict:
```

**SQL:** Agrega capacidade e consumo de todos os nodes:
```sql
SELECT
    sum(cpu_total) as total_cpu,
    sum(mem_total) as total_mem,
    sum(tasks_running) as total_tasks
FROM nodes
WHERE status = 'ready'
```

Mais consumo agregado:
```sql
SELECT
    quantile(m.cpu_percent, 0.95) as cpu_p95,
    sum(m.mem_usage) as mem_usage
FROM metrics m
JOIN container_node_map c ON m.container_id = c.container_id
WHERE m.ts > now() - INTERVAL {settings.analysis_window_days} DAYS
```

### 6.3. Retention (em `run_retention()`)

Adicionar após os DELETEs existentes:
```python
self._conn.execute(
    f"DELETE FROM node_metrics WHERE ts < now() - INTERVAL {settings.retention_days} DAYS"
)
```

---

## 7. Backend — Collector

Modificar `backend/services/collector.py` para adicionar 2 novos loops de coleta.

### 7.1. Node collection loop (a cada `RESMA_COLLECT_INTERVAL`)

Adicionar nova task em `start()`:
```python
self._tasks.append(asyncio.create_task(self._node_collect_loop()))
```

**Método `_node_collect_loop()`:**
```python
async def _node_collect_loop(self):
    while self._running:
        try:
            nodes = await docker_client.get_nodes()
            tasks_by_node = await docker_client.get_tasks_by_node()

            node_rows = []
            metric_rows = []

            for node in nodes:
                node_tasks = tasks_by_node.get(node["id"], [])
                running_count = sum(1 for t in node_tasks if t.get("Status", {}).get("State") == "running")

                # Upsert node state
                db.upsert_node(
                    node_id=node["id"],
                    hostname=node["hostname"],
                    role=node["role"],
                    availability=node["availability"],
                    status=node["status"],
                    address=node["address"],
                    cpu_total=node["cpu_total"],
                    mem_total=node["mem_total"],
                    os=node["os"],
                    architecture=node["architecture"],
                    engine_version=node["engine_version"],
                    is_leader=node["is_leader"],
                    reachability=node["reachability"],
                    labels=json.dumps(node["labels"]),
                    tasks_running=running_count,
                )

                # Append node metrics
                ts = datetime.now(timezone.utc)
                metric_rows.append([
                    ts, node["id"], node["hostname"], node["role"],
                    node["availability"], node["status"],
                    node["cpu_total"], node["mem_total"], running_count
                ])

            if metric_rows:
                db.insert_node_metrics_batch(metric_rows)

            # Update container_node_map
            mapping = await docker_client.get_container_node_mapping()
            if mapping:
                db.upsert_container_node_map_batch(
                    [[m["container_id"], m["node_id"], m["service"]] for m in mapping]
                )

            logger.debug("Collected %d nodes", len(nodes))
        except Exception as e:
            logger.error("Node collect error: %s", e)
        await asyncio.sleep(settings.collect_interval)
```

### 7.2. Cluster collection loop (a cada `RESMA_CLUSTER_INTERVAL`)

Adicionar nova task em `start()`:
```python
self._tasks.append(asyncio.create_task(self._cluster_collect_loop()))
```

**Método `_cluster_collect_loop()`:**
```python
async def _cluster_collect_loop(self):
    while self._running:
        try:
            info = await docker_client.get_swarm_info()
            nodes = db.get_nodes()

            managers = sum(1 for n in nodes if n["role"] == "manager")
            workers = sum(1 for n in nodes if n["role"] == "worker")
            ready = sum(1 for n in nodes if n["status"] == "ready")
            down = sum(1 for n in nodes if n["status"] != "ready")
            quorum = managers >= 1 and any(n["reachability"] == "reachable" for n in nodes if n["role"] == "manager")

            db.upsert_cluster(
                cluster_id=info["cluster_id"] or "default",
                nodes_total=len(nodes),
                managers_total=managers,
                workers_total=workers,
                nodes_ready=ready,
                nodes_down=down,
                quorum_healthy=quorum,
                self_node_id=info["node_id"],
                warnings=json.dumps(info["warnings"]),
            )

            logger.debug("Cluster info updated")
        except Exception as e:
            logger.error("Cluster collect error: %s", e)
        await asyncio.sleep(settings.cluster_interval)
```

### 7.3. Imports necessários

Adicionar no topo de `collector.py`:
```python
import json
```

---

## 8. Backend — Router: nodes.py

Criar `backend/routers/nodes.py`.

### 8.1. Estrutura

```python
from fastapi import APIRouter, Depends, HTTPException, status

from backend.core.config import settings
from backend.core.db import db
from backend.core.docker_client import docker_client
from backend.services.auth import get_current_user

router = APIRouter(prefix="/api/nodes", tags=["nodes"])
```

### 8.2. Endpoints

#### `GET /api/nodes`

Lista todos os nodes do DuckDB (tabela `nodes`).

**Response:** lista de:
```json
[
    {
        "node_id": "e216jshn25ckzbvmwlnh5jr3g",
        "hostname": "swarm-manager",
        "role": "manager",
        "availability": "active",
        "status": "ready",
        "address": "168.0.32.137",
        "cpu_total": 4.0,
        "mem_total": 8367677440,
        "os": "linux",
        "architecture": "x86_64",
        "engine_version": "27.0.0",
        "is_leader": true,
        "reachability": "reachable",
        "labels": {"storage": "ssd"},
        "tasks_running": 12,
        "updated_at": "2026-07-21T12:00:00"
    }
]
```

**Query params opcionais:**
- `role=manager` ou `role=worker` — filtra por role
- `status=ready` ou `status=down` — filtra por status
- `availability=active` — filtra por availability

#### `GET /api/nodes/{node_id}`

Detalhe do node do DuckDB + consumo agregado + Docker API para dados não persistidos.

**Response:**
```json
{
    "node": { ...campos de nodes... },
    "consumption": {
        "cpu_p95": 45.2,
        "mem_p99": 536870912,
        "mem_total_usage": 1073741824,
        "containers": 8
    },
    "capacity": {
        "cpu_total": 4.0,
        "mem_total": 8367677440,
        "cpu_utilization_pct": 11.3,
        "mem_utilization_pct": 12.8
    }
}
```

**Lógica:**
1. `db.get_node_by_id(node_id)` — se não existir, 404
2. `db.get_node_consumption(node_id)` — agrega métricas de containers no node
3. Calcular `cpu_utilization_pct` = `consumption.cpu_p95 / node.cpu_total * 100`
4. Calcular `mem_utilization_pct` = `consumption.mem_total_usage / node.mem_total * 100`

#### `GET /api/nodes/{node_id}/metrics`

Time-series do node (tabela `node_metrics`).

**Query params:**
- `range=7d` (default) — janela em dias

**Response:**
```json
[
    {
        "ts": "2026-07-21T12:00:00",
        "tasks_running": 12,
        "cpu_total": 4.0,
        "mem_total": 8367677440
    }
]
```

#### `GET /api/nodes/{node_id}/services`

Lista services rodando no node com consumo agregado.

**Response:**
```json
[
    {
        "service": "my-api",
        "containers": 3,
        "cpu_p95": 45.2,
        "mem_p99": 536870912
    }
]
```

#### `GET /api/cluster`

Estado do cluster (tabela `cluster`).

**Response:**
```json
{
    "id": "v1b67c3a...",
    "nodes_total": 5,
    "managers_total": 3,
    "workers_total": 2,
    "nodes_ready": 5,
    "nodes_down": 0,
    "quorum_healthy": true,
    "self_node_id": "e216jshn25ckzbvmwlnh5jr3g",
    "warnings": null,
    "updated_at": "2026-07-21T12:00:00"
}
```

### 8.3. Registro do router

Adicionar em `backend/main.py`:
```python
from backend.routers import nodes
app.include_router(nodes.router)
```

---

## 9. Backend — Dashboard (enriquecimento)

Modificar `backend/routers/dashboard.py` para adicionar dados de cluster no response.

### 9.1. Novos campos no response do `GET /api/dashboard`

Adicionar ao dict de retorno:

```python
"cluster": {
    "nodes_total": cluster_info["nodes_total"] if cluster_info else 0,
    "nodes_ready": cluster_info["nodes_ready"] if cluster_info else 0,
    "managers_total": cluster_info["managers_total"] if cluster_info else 0,
    "quorum_healthy": cluster_info["quorum_healthy"] if cluster_info else False,
},
"nodes_distribution": [
    {"hostname": n["hostname"], "tasks_running": n["tasks_running"]}
    for n in db.get_nodes()
],
```

**Onde:**
- `cluster_info = db.get_cluster()` no início do handler
- `db.get_nodes()` para a distribuição

### 9.2. Query de capacidade do cluster

Adicionar aggregate de capacidade:
```python
cluster_capacity = db.fetchone(
    "SELECT sum(cpu_total) as total_cpu, sum(mem_total) as total_mem, sum(tasks_running) as total_tasks FROM nodes WHERE status = 'ready'"
)
```

Incluir no response:
```python
"cluster_capacity": {
    "cpu_total": round(cluster_capacity[0] or 0, 1),
    "mem_total": int(cluster_capacity[1] or 0),
    "tasks_total": int(cluster_capacity[2] or 0),
},
```

---

## 10. Frontend — Navegação

### 10.1. Sidebar (Layout.tsx)

Adicionar item no array `navItems` (após Dashboard, antes de Recomendações):

```typescript
{ to: "/nodes", label: "Nodes", icon: Server, activeColor: "text-chart-3", activeBg: "bg-chart-3/10", barColor: "bg-chart-3" },
```

**Import:** Adicionar `Server` do `lucide-react` nos imports existentes.

### 10.2. Breadcrumbs (Layout.tsx)

Adicionar em `buildBreadcrumbs()`:

```typescript
} else if (segments[0] === "nodes") {
    crumbs.push({ label: "Nodes", to: "/nodes" })
    if (segments[1]) {
        crumbs.push({ label: decodeURIComponent(segments[1]) })
    }
}
```

### 10.3. Rotas (App.tsx)

Adicionar rotas:

```tsx
<Route path="/nodes" element={<Layout><Nodes /></Layout>} />
<Route path="/nodes/:id" element={<Layout><NodeDetail /></Layout>} />
```

**Imports:**
```tsx
import Nodes from "@/pages/Nodes"
import NodeDetail from "@/pages/NodeDetail"
```

---

## 11. Frontend — Página: Nodes.tsx

### 11.1. Estrutura

Arquivo: `frontend/src/pages/Nodes.tsx`

Segue o padrão de `Services.tsx` (tabela com progress bars e status badges).

### 11.2. Stat cards no topo (4 cards)

Grid de 4 cards (padrão Dashboard), acima da tabela:

| Card | Valor | Ícone | Cor |
|------|-------|-------|-----|
| Nodes Online | `{nodes_ready} de {nodes_total}` | Server | `text-primary` se todos online, `text-destructive` se algum down |
| Capacidade CPU | `{cpu_total} cores` | Cpu | `text-chart-1` |
| Capacidade RAM | `{formatBytes(mem_total)}` | MemoryStick | `text-chart-2` |
| Tasks Ativas | `{tasks_total}` | Boxes | `text-chart-3` |

**Data source:** `GET /api/cluster` + `GET /api/nodes`

### 11.3. Tabela de nodes

Colunas (padrão Services com progress bars):

| Coluna | Largura | Conteúdo |
|--------|---------|----------|
| Hostname | auto | Texto, clickável → `/nodes/{node_id}` |
| Role | 100px | Badge: Manager (variant default) ou Worker (variant outline). Se `is_leader`, ícone coroa (Crown do lucide) ao lado |
| Status | 80px | Badge: Ready (variant success, dot verde) ou Down (variant destructive, dot vermelho) |
| Availability | 90px | Badge: Active (variant outline, dot verde), Pause (variant outline, dot amarelo), Drain (variant outline, dot vermelho) |
| CPU Total | 160px | `{cpu_total} cores` + Progress bar mostrando `cpu_utilization_pct` (verde <50%, amarelo 50-80%, vermelho >80%) |
| RAM Total | 160px | `{formatBytes(mem_total)}` + Progress bar mostrando `mem_utilization_pct` (mesma escala de cores) |
| Tasks | 60px | Badge com número, cor baseada em count (verde ≤5, amarelo 6-15, vermelho >15) |
| Engine | 80px | Texto muted, versão do Docker |
| Labels | 50px | Contador de labels com tooltip listando chave-valor |
| (action) | 40px | ChevronRight |

**Filtros** (DropdownMenu, padrão ServiceDetail):
- Por role: Todos / Manager / Worker
- Por status: Todos / Ready / Down
- Por availability: Todos / Active / Pause / Drain

### 11.4. Estados

- **Loading:** Skeleton (padrão Services)
- **Error:** Alert destructive (padrão Dashboard)
- **Empty:** EmptyState com ícone Server e mensagem "Nenhum node encontrado. Aguardando coleta..."

### 11.5. Refresh

Usar `useRefreshInterval()` como `refetchInterval` (padrão existente).

### 11.6. Interfaces TypeScript

```typescript
interface Node {
    node_id: string
    hostname: string
    role: string
    availability: string
    status: string
    address: string
    cpu_total: number
    mem_total: number
    os: string
    architecture: string
    engine_version: string
    is_leader: boolean
    reachability: string | null
    labels: string  // JSON string
    tasks_running: number
    updated_at: string
}

interface ClusterInfo {
    id: string
    nodes_total: number
    managers_total: number
    workers_total: number
    nodes_ready: number
    nodes_down: number
    quorum_healthy: boolean
    self_node_id: string
    warnings: string | null
    updated_at: string
}
```

---

## 12. Frontend — Página: NodeDetail.tsx

### 12.1. Estrutura

Arquivo: `frontend/src/pages/NodeDetail.tsx`

Segue o padrão de `ServiceDetail.tsx` (stat cards + tabs com charts).

### 12.2. Header

`PageHeader` com:
- Title: hostname do node
- Description: `{role} • {address} • {os}/{architecture}`
- Badges: Status (Ready/Down), Availability (Active/Pause/Drain), Leader (se is_leader, badge dourado)
- Button "Voltar" → `/nodes`

### 12.3. Stat cards (4 cards, padrão ServiceDetail)

| Card | Valor | Ícone |
|------|-------|-------|
| CPU Total | `{cpu_total} cores` | Cpu |
| RAM Total | `{formatBytes(mem_total)}` | MemoryStick |
| Tasks Rodando | `{tasks_running}` | Boxes |
| Containers | `{consumption.containers}` | Box |

### 12.4. Tabs (3 tabs, padrão ServiceDetail)

#### Tab 1 — Visão Geral

**Area chart CPU (esquerda):**
- Título: "CPU — Consumo vs. Capacidade"
- Data: `GET /api/nodes/{id}/metrics` (time-series)
- Eixo X: timestamp
- Eixo Y: % de utilização (consumo / capacidade * 100)
- Área laranja (`--chart-1`): consumo real agregado dos containers
- Linha tracejada: 100% (capacidade total do node)
- Padrão visual: AreaChart com gradient (igual ServiceDetail)

**Area chart Memória (direita):**
- Título: "Memória — Consumo vs. Capacidade"
- Mesmo padrão, área azul (`--chart-2`)
- Eixo Y: MB
- Linha tracejada: capacidade total

**Capacity utilization bars (abaixo dos charts):**
- Card com 2 Progress bars:
  - CPU: `cpu_utilization_pct` (label: "CPU Utilization", value: `{pct}%`, cor: verde/amarelo/vermelho)
  - RAM: `mem_utilization_pct` (label: "RAM Utilization", value: `{pct}%`, mesma escala)

#### Tab 2 — Services no Node

- Tabela (padrão Services): Service name (clickable → `/services/{name}`), Containers, CPU P95 (progress bar), Memória P99 (progress bar)
- Data: `GET /api/nodes/{id}/services`
- Empty state se nenhum service

#### Tab 3 — Info

- Card com lista chave-valor (estilo ServiceDetail "Estatísticas Detalhadas"):
  - Platform: `{os} / {architecture}`
  - Engine: `{engine_version}`
  - Address: `{address}`
  - Role: `{role}`
  - Leader: `{is_leader ? "Yes" : "No"}`
  - Reachability: `{reachability ?? "N/A"}`
  - Created at: (do Docker API, se disponível)
  - Updated at: `{updated_at}`

- Card com Labels (se existirem):
  - Lista de badges em formato `{key}: {value}`
  - Se vazio: "Nenhum label configurado"

### 12.5. Estados

- **Loading:** Skeleton (padrão ServiceDetail)
- **Error:** Alert destructive
- **Not found:** EmptyState "Node não encontrado"

### 12.6. Refresh

`useRefreshInterval()` em todas as queries.

### 12.7. Interfaces TypeScript

```typescript
interface NodeDetail {
    node: Node
    consumption: {
        cpu_p95: number
        mem_p99: number
        mem_total_usage: number
        containers: number
    }
    capacity: {
        cpu_total: number
        mem_total: number
        cpu_utilization_pct: number
        mem_utilization_pct: number
    }
}

interface NodeMetric {
    ts: string
    tasks_running: number
    cpu_total: number
    mem_total: number
}

interface NodeService {
    service: string
    containers: number
    cpu_p95: number
    mem_p99: number
}
```

---

## 13. Frontend — Dashboard (enriquecimento)

### 13.1. Novos stat cards

Modificar `Dashboard.tsx` para expandir o grid de 5 para 7 cards.

Adicionar após os cards existentes:

```typescript
{
    label: "Nodes",
    value: `${data.cluster.nodes_ready}/${data.cluster.nodes_total}`,
    icon: Server,
    iconBg: data.cluster.nodes_ready < data.cluster.nodes_total ? "bg-destructive/15" : "bg-chart-3/15",
    iconColor: data.cluster.nodes_ready < data.cluster.nodes_total ? "text-destructive" : "text-chart-3",
    valueColor: data.cluster.nodes_ready < data.cluster.nodes_total ? "text-destructive" : "text-foreground",
    clickable: true,
    onClick: () => navigate("/nodes"),
},
{
    label: "Cluster",
    value: data.cluster.quorum_healthy ? "Saudável" : "Quorum Lost",
    icon: ShieldCheck, // ou AlertTriangle se quorum lost
    iconBg: data.cluster.quorum_healthy ? "bg-success/15" : "bg-destructive/15",
    iconColor: data.cluster.quorum_healthy ? "text-success" : "text-destructive",
    valueColor: data.cluster.quorum_healthy ? "text-success" : "text-destructive",
    clickable: false,
},
```

**Grid:** Mudar de `md:grid-cols-5` para `md:grid-cols-7` (ou usar `md:grid-cols-4` + segunda linha com 3).

**Imports:** Adicionar `Server`, `ShieldCheck` do `lucide-react`.

### 13.2. Novo chart — Distribuição de Tasks por Node

Adicionar abaixo dos charts de Top CPU e Top Memory existentes:

```tsx
<Card>
    <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
            <Server className="h-4 w-4 text-chart-3" />
            Distribuição de Tasks por Node
        </CardTitle>
    </CardHeader>
    <CardContent>
        <div className="h-48">
            <ResponsiveContainer width="100%" height="100%">
                <BarChart data={nodesDistribution} layout="vertical">
                    <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" horizontal={false} />
                    <XAxis type="number" tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} />
                    <YAxis type="category" dataKey="name" tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} width={90} />
                    <RTooltip contentStyle={chartTooltipStyle} cursor={{ fill: "var(--muted)" }} />
                    <Bar dataKey="tasks" fill="var(--chart-3)" radius={[0, 4, 4, 0]} name="Tasks" />
                </BarChart>
            </ResponsiveContainer>
        </div>
    </CardContent>
</Card>
```

**Data:** `data.nodes_distribution` do `GET /api/dashboard`:
```typescript
const nodesDistribution = (data?.nodes_distribution ?? []).map((n) => ({
    name: n.hostname.length > 12 ? n.hostname.slice(0, 11) + "…" : n.hostname,
    tasks: n.tasks_running,
}))
```

### 13.3. Interface atualizada

```typescript
interface DashboardData {
    total_services: number
    total_containers: number
    top_cpu_consumers: Array<{ name: string; container_count: number; cpu_p95: number; mem_p99: number }>
    top_mem_consumers: Array<{ name: string; container_count: number; cpu_p95: number; mem_p99: number }>
    recent_ooms: Array<{ ts: string; service: string; oom_count: number }>
    leak_alerts: Array<{ service: string; daily_growth_mb: number; r_squared: number }>
    drift_alerts: Array<{ service: string; cpu_drift: number; mem_drift: number }>
    // NOVOS:
    cluster: {
        nodes_total: number
        nodes_ready: number
        managers_total: number
        quorum_healthy: boolean
    }
    cluster_capacity: {
        cpu_total: number
        mem_total: number
        tasks_total: number
    }
    nodes_distribution: Array<{ hostname: string; tasks_running: number }>
}
```

---

## 14. Ordem de Implementação

Executar nesta ordem exata. Cada passo deve compilar sem erros antes do próximo.

### Fase 1: Backend — Dados

1. **`backend/core/config.py`**: Adicionar `cluster_interval: int = 60`
2. **`backend/core/db.py`**: Adicionar 4 tabelas em `_init_schema()` + métodos novos + retention
3. **`backend/core/docker_client.py`**: Adicionar 5 métodos novos (`get_nodes`, `get_node_detail`, `get_swarm_info`, `get_tasks_by_node`, `get_container_node_mapping`)
4. **`backend/services/collector.py`**: Adicionar `_node_collect_loop()` e `_cluster_collect_loop()` + registrar tasks em `start()` + import `json`
5. **`backend/routers/nodes.py`**: Criar router com 5 endpoints
6. **`backend/main.py`**: Importar e registrar `nodes.router`
7. **`backend/routers/dashboard.py`**: Adicionar `cluster`, `cluster_capacity`, `nodes_distribution` no response

### Fase 2: Frontend — Navegação

8. **`frontend/src/components/Layout.tsx`**: Adicionar nav item "Nodes" + breadcrumb
9. **`frontend/src/App.tsx`**: Adicionar rotas `/nodes` e `/nodes/:id`

### Fase 3: Frontend — Páginas

10. **`frontend/src/pages/Nodes.tsx`**: Criar página de listagem
11. **`frontend/src/pages/NodeDetail.tsx`**: Criar página de detalhe
12. **`frontend/src/pages/Dashboard.tsx`**: Adicionar 2 stat cards + chart de distribuição + atualizar interface

---

## 15. Validação

### 15.1. Backend

- Build sem erros: `cd backend && python -m py_compile main.py`
- Endpoints respondem:
  - `GET /api/nodes` retorna lista (pode ser vazia se collector ainda não rodou)
  - `GET /api/cluster` retorna estado do cluster
  - `GET /api/nodes/{id}` retorna detalhe
  - `GET /api/nodes/{id}/metrics` retorna time-series
  - `GET /api/nodes/{id}/services` retorna services no node
  - `GET /api/dashboard` inclui `cluster`, `cluster_capacity`, `nodes_distribution`
- DuckDB: tabelas criadas sem erro, `node_metrics` recebe rows a cada `RESMA_COLLECT_INTERVAL`
- Retention: `node_metrics` limpa registros > 30 dias

### 15.2. Frontend

- Build sem erros: `cd frontend && npm run build`
- Sidebar mostra "Nodes" entre Dashboard e Recomendações
- `/nodes` renderiza com estados loading/erro/empty
- `/nodes/{id}` renderiza com 3 tabs
- Dashboard mostra 7 stat cards + chart de distribuição
- Refresh funciona (auto/5s/30s/1m/5m/off)
- Navegação: Nodes → click node → NodeDetail → click service → ServiceDetail
- Breadcrumbs funcionam em `/nodes` e `/nodes/{id}`

### 15.3. Sem hardcode

- Todos os dados vêm da Docker API ou DuckDB
- Intervalos via env vars (`RESMA_COLLECT_INTERVAL`, `RESMA_CLUSTER_INTERVAL`)
- Sem IPs, hostnames, ou IDs hardcoded

---

## 16. Riscos Técnicos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| `tasks.list()` não retorna `ContainerID` para todas as tasks | Média | Baixo | Filtrar `None` no mapping; `container_node_map` terá menos rows mas `nodes` e `node_metrics` não dependem disso |
| Docker API em Swarm com muitos nodes é lenta | Baixa | Baixo | `nodes.list()` é leve (~1KB por node); `tasks.list()` já é chamada em `get_service_status_map()` sem problemas |
| `container_node_map` desatualizada se container reinicia | Média | Baixo | Atualizada a cada `RESMA_COLLECT_INTERVAL` (15s default); container_id muda a cada restart mas métricas novas usam o novo ID |
| Node removido do Swarm fica órfão na tabela `nodes` | Baixa | Baixo | O upsert só atualiza nodes que existem; nodes removidos não aparecem em `nodes.list()`. Considerar limpeza futura (não neste escopo) |
| DuckDB lock contention com 2 loops de coleta | Baixa | Médio | `_lock` é RLock (reentrante); loops rodam em threads diferentes mas o lock protege todas as operações |

---

## 17. Dependências

**Sem novas dependências.** Tudo usa os pacotes já instalados:

- Backend: `aiodocker` (já usado), `duckdb` (já usado), `fastapi` (já usado)
- Frontend: `recharts` (já usado), `lucide-react` (já usado), `@tanstack/react-query` (já usado), shadcn/ui (já usado)

---

## 18. Notas para o Agente Implementador

1. **Leia os arquivos existentes antes de começar.** Especialmente: `db.py`, `docker_client.py`, `collector.py`, `dashboard.py`, `services.py`, `Layout.tsx`, `App.tsx`, `Dashboard.tsx`, `Services.tsx`, `ServiceDetail.tsx`. O padrão visual e de código está lá.

2. **Siga o padrão existente.** Não invente novos patterns. Se `Services.tsx` usa `Progress` com `indicatorClassName`, faça igual. Se `ServiceDetail.tsx` usa `AreaChart` com gradient, faça igual.

3. **Não altere nada que não esteja nesta spec.** Se sentir necessidade de alterar algo fora do escopo, pare e pergunte.

4. **Teste cada passo.** Após cada item da ordem de implementação, verifique se compila e se o endpoint/tela funciona.

5. **Comentários apenas no cabeçalho do arquivo.** Sem comentários no corpo do código (padrão do projeto).

6. **Espaços entre grupos de código** para facilitar leitura (padrão do projeto).

7. **Use `formatBytes()` e `formatCPU()`** de `@/lib/utils` para formatação (já existem).

8. **Cores dos charts:** `--chart-1` (CPU), `--chart-2` (memória), `--chart-3` (nodes/distribuição). Não invente cores novas.

9. **Logger:** Use `logging.getLogger("resma.<module>")` — ex: `"resma.collector"`, `"resma.nodes"`.

10. **Auth:** Todos os endpoints novos usam `Depends(get_current_user)` — igual aos routers existentes.
