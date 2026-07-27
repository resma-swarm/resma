# Fase 7 — Multi-Node Agent (Coleta de Stats em Swarm com N Workers)

> **Prioridade:** Alta
> **Esforço:** Alto (novo binary Go + protocolo + endpoints de ingestão + testes multi-node)
> **Bloqueador:** Não (Fase 0b pode continuar em paralelo)
> **Dependências:** Fase 0b (Go API + collector + DuckDB), Fase 4 (docker-stack.yml)

## Contexto do problema

### Limitação atual

O RESMA roda como 1 container no manager node do Docker Swarm com acesso ao Docker socket local (`/var/run/docker.sock`). O collector atual usa `ContainerList` + `ContainerStats` da Docker API, que são **chamadas locais** — só retornam containers do daemon onde o socket está montado.

Em um Swarm com 1 manager + N workers:

| Categoria de dado | API Docker usada | Funciona multi-node? |
|-------------------|------------------|----------------------|
| Swarm metadata (nós, tasks, serviços, cluster) | `NodeList`, `TaskList`, `ServiceList`, `Info` | **Sim** — manager tem visão global |
| Container stats (CPU%, memória, rede, I/O) | `ContainerList`, `ContainerStats` | **NÃO** — só containers do node local |
| OOM events | `Events` (filter `die` + exit 137) | **NÃO** — só eventos do node local |

**Resultado em 1 manager + 5 workers:** O RESMA coleta stats de apenas 1/6 dos containers. As recomendações ML seriam baseadas em dados parciais e não confiáveis. OOMs em workers são invisíveis.

### Gap já identificado na spec 0b

A spec `phase-0b-go-migration` já listava "Swarm task monitoring" como feature futura de prioridade média, mas não resolvia a coleta de stats de workers — que é o problema fundamental.

## Pesquisa de mercado (benchmark)

Foram investigadas 4 categorias de soluções usadas pelo mercado para monitoramento multi-node em Docker Swarm:

### 1. Swarmpit Agent — modelo push HTTP

**Repositório:** github.com/swarmpit/agent

| Aspecto | Detalhe |
|---------|---------|
| Linguagem | Go (89%) |
| Imagem | ~6.3 MB (base `scratch`, binário estático) |
| RAM | 64 MB limite / 32 MB reserva |
| CPU | 0.10 cores limite / 0.05 reserva |
| Protocolo | HTTP POST para `http://app:8080/events` |
| Frequência | 30s (configurável via `STATS_FREQUENCY`) |
| Deploy | `mode: global` no Swarm (1 por node) |
| Socket | `/var/run/docker.sock:/var/run/docker.sock:ro` |
| Cluster | Nenhum — cada agent fala diretamente com o servidor |
| Auth | Nenhuma — confia na rede overlay |
| Multi-arch | amd64, arm64, arm/v7 |

**Modelo:** Cada agent coleta stats locais via Docker socket e faz HTTP POST para o servidor no manager. Sem clusterização entre agents.

**Pontos fortes:** Extremamente leve, simples, sem dependências.
**Pontos fracos:** Sem TLS, sem auth, projeto em manutenção limitada (último commit 2020).

### 2. Portainer Agent — modelo proxy transparente

**Repositório:** github.com/portainer/agent

| Aspecto | Detalhe |
|---------|---------|
| Linguagem | Go |
| Imagem | ~50 MB (base alpine) |
| RAM | ~20 MB por agent |
| CPU | ~0.1% de 1 core |
| Protocolo | HTTP/HTTPS (porta 9001) + WebSocket + Serf (gossip) |
| Deploy | `mode: global` no Swarm ou DaemonSet no K8s |
| Socket | `/var/run/docker.sock` + `/var/lib/docker/volumes` |
| Cluster | Serf (HashiCorp) para descoberta e membership |
| Auth | Assinatura digital via headers (`X-PortainerAgent-Signature`) |
| Compatibilidade | 100% compatível com Docker API padrão |

**Modelo:** O agent é um **proxy transparente** da Docker API. O Portainer Server faz requisições Docker API padrão para qualquer agent, que roteia para o node apropriado e agrega respostas de múltiplos nodes. Não coleta nada por iniciativa própria — é pull sob demanda.

**Pontos fortes:** Compatibilidade total com Docker API, agregação transparente, clusterização automática via Serf.
**Pontos fracos:** Complexidade alta (Serf, assinaturas, proxy cluster), memory leak histórico (issue #2254), imagem maior.

### 3. cAdvisor + node-exporter — modelo pull Prometheus

**Repositórios:** github.com/google/cadvisor, github.com/prometheus/node_exporter

| Aspecto | cAdvisor | node-exporter |
|---------|----------|---------------|
| Linguagem | Go | Go |
| Imagem | ~70 MB | ~22 MB |
| RAM | ~100 MB (40 containers) | ~180 MiB |
| CPU | 0.2%-70% (varia muito) | 102m-250m |
| Protocolo | HTTP `/metrics` (Prometheus text) | HTTP `/metrics` (Prometheus text) |
| Porta | 8080 | 9100 |
| Deploy | DaemonSet (K8s) / global (Swarm) | DaemonSet (K8s) / global (Swarm) |
| Scrape | Prometheus faz pull a cada 15-30s | Prometheus faz pull a cada 15-30s |

**Modelo:** Cada node roda cAdvisor (containers) + node-exporter (host). Um Prometheus central faz scrape de todos via service discovery do Swarm (`dockerswarm_sd_configs`).

**Pontos fortes:** Padrão de facto do mercado, ecossistema maduro, métricas ricas.
**Pontos fracos:** Requer Prometheus como dependência externa (viola princípio "self-contained" do RESMA), overhead alto (cAdvisor pode usar 100MB+ RAM), formato Prometheus text não é ideal para OLAP/DuckDB.

### 4. Alternativas modernas

| Abordagem | Viabilidade | Segurança | Overhead | Compatibilidade RESMA |
|-----------|-------------|-----------|----------|----------------------|
| Docker TCP/TLS (porta 2376) | Alta | **Muito baixa** (acesso root equivalente) | Baixo | Média |
| gRPC + Protobuf | Alta | Alta (TLS nativo) | Baixo (3-10x menor que JSON) | Média |
| WebSocket bidirecional | Alta | Média | Médio | Alta |
| Overlay network (VXLAN) | Alta | Média (-24% throughput, +0.4-0.7ms) | Médio | Alta |
| SSH tunneling | Média | Alta | Médio | Média |
| Cetacean (manager-only) | Alta | Alta | Baixo | Alta — mas não coleta stats remotos |

**Descobertas chave:**
- **Docker TCP/TLS é proibido** pela OWASP ("RULE #1: Do not expose the Docker daemon socket"). Acesso root equivalente via rede.
- **A Swarm API NÃO tem endpoint de stats remotos.** `docker node inspect` retorna info do node mas não stats de containers. O endpoint `/containers/{id}/stats` só funciona no daemon local.
- **Cetacean e OneUptime** provam que manager-only funciona para inventário (services, tasks, nodes) mas **ambos precisam de agents para stats de containers em workers**.
- **Overlay network** é viável para comunicação agent→servidor mas tem -24% de throughput e criptografia nativa causa 99% de perda (issue #33133).

## Decisão arquitetural

### Modelo escolhido: Agent push HTTP (estilo Swarmpit, adaptado para RESMA)

**Justificativa:**

1. **Alinhamento com princípios do RESMA:** "O básico que funciona", self-contained, sem dependências externas (sem Prometheus, sem Serf, sem InfluxDB)
2. **Simplicidade:** O Swarmpit agent prova que um binary Go de ~6MB resolve o problema com HTTP simples
3. **Compatibilidade com arquitetura atual:** O RESMA já tem Go API + DuckDB + collector. O agent apenas envia dados para o collector ingestar no DuckDB
4. **Overhead mínimo:** ~10-15MB imagem, ~32-64MB RAM por agent, ~0.05-0.10 CPU
5. **Segurança adequada:** Overlay network do Swarm isola o tráfego; token compartilhado para auth simples

### Por que não o modelo Portainer (proxy transparente)?

- **Complexidade:** Serf cluster, assinaturas digitais, proxy aggregation — muito código para o benefício
- **Pull sob demanda:** O RESMA precisa de coleta contínua (streaming), não queries esporádicas
- **Incompatibilidade com streaming:** O Portainer proxy não suporta `ContainerStats(stream=true)` de forma eficiente para múltiplos nodes

### Por que não cAdvisor + Prometheus?

- **Viola "self-contained":** Prometheus é uma dependência externa pesada
- **Overhead:** cAdvisor usa 100MB+ RAM por node — mais que o próprio RESMA
- **Formato:** Prometheus text format não é ideal para DuckDB (columnar OLAP)
- **Escopo:** cAdvisor coleta métricas que o RESMA não precisa (TCP, disk I/O detalhado, labels de alta cardinalidade)

### Por que não gRPC?

- **Complexidade:** Requer `.proto`, codegen, servidor gRPC separado
- **Benefício marginal:** Para ~100 containers a cada 15s, HTTP/JSON é perfeitamente adequado
- **Decisão:** gRPC é uma otimização futura se a escala aumentar significativamente

## Arquitetura proposta

### Diagrama

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Docker Swarm Cluster                              │
│                                                                      │
│  ┌──────────────────────────┐                                       │
│  │    Manager Node           │                                       │
│  │  ┌─────────────────────┐  │                                       │
│  │  │  RESMA API (Go)     │  │                                       │
│  │  │  - Collector local  │  │  ← coleta stats do manager local     │
│  │  │  - Ingestion API    │  │  ← recebe dados dos agents           │
│  │  │  - DuckDB (owner)   │  │                                       │
│  │  │  - ML sidecar       │  │                                       │
│  │  └──────────┬──────────┘  │                                       │
│  │             │              │                                       │
│  │  ┌──────────▼──────────┐  │                                       │
│  │  │  Docker Socket      │  │  ← /var/run/docker.sock (local)      │
│  │  └─────────────────────┘  │                                       │
│  └──────────────────────────┘                                       │
│               ▲                                                      │
│               │ HTTP POST /api/agent/ingest                          │
│               │ (overlay network: resma-net)                         │
│               │ Auth: Bearer RESMA_AGENT_TOKEN                       │
│               │                                                      │
│  ┌────────────┴──────────────┐  ┌──────────────────────┐            │
│  │  Worker 1                  │  │  Worker N             │            │
│  │  ┌─────────────────────┐   │  │  ┌─────────────────┐  │            │
│  │  │  RESMA Agent (Go)   │   │  │  │  RESMA Agent    │  │            │
│  │  │  - Stats collector  │   │  │  │  (global svc)   │  │            │
│  │  │  - Event listener   │   │  │  └────────┬────────┘  │            │
│  │  │  - HTTP push        │   │  │           │           │            │
│  │  └──────────┬──────────┘   │  │  ┌────────▼────────┐  │            │
│  │             │              │  │  │  Docker Socket  │  │            │
│  │  ┌──────────▼──────────┐   │  │  │  (local, :ro)   │  │            │
│  │  │  Docker Socket      │   │  │  └─────────────────┘  │            │
│  │  │  (local, :ro)       │   │  └──────────────────────┘            │
│  │  └─────────────────────┘   │                                       │
│  └────────────────────────────┘                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### Componentes

#### 1. RESMA Agent (novo binary Go)

Binary Go leve que roda como `mode: global` no Swarm (1 por node).

**Responsabilidades:**
- Conectar ao Docker socket local (`/var/run/docker.sock:ro`)
- Coletar stats de containers locais via `ContainerStats(stream=true)` (mesmo padrão do collector atual)
- Escutar eventos Docker locais (start/stop/die/destroy)
- Detectar OOM (exit 137) e enviar imediatamente
- Enviar batches de métricas via HTTP POST para o RESMA API no manager
- Heartbeat a cada 30s para indicar que está vivo

**NÃO responsável por:**
- Swarm metadata (nodes, tasks, services, cluster) — o manager já coleta via Swarm API
- Storage info (`docker system df`) — o manager já coleta localmente
- Análise ML — o ML sidecar no manager faz isso

**Configuração via env vars:**

| Variável | Default | Descrição |
|----------|---------|-----------|
| `RESMA_API_URL` | `http://api:8080` | URL do RESMA API no manager (via overlay) |
| `RESMA_AGENT_TOKEN` | (obrigatório) | Token compartilhado para auth agent→server |
| `RESMA_COLLECT_INTERVAL` | `15s` | Intervalo de coleta de stats (igual ao server) |
| `RESMA_NODE_ID` | `{{.Node.ID}}` | ID do node (template var do Swarm) |
| `RESMA_NODE_HOSTNAME` | `{{.Node.Hostname}}` | Hostname do node (template var) |
| `RESMA_EXCLUDED_IMAGES` | (vazio) | Imagens a excluir da coleta (ex: `resma-agent:latest`) |
| `RESMA_AGENT_DEBUG` | `false` | Logs de debug |

**Endpoints do agent (HTTP server local, porta 8082):**
- `GET /health` — health check para Docker HEALTHCHECK
- `GET /info` — info do agent (versão, node_id, containers monitorados)

**Protocolo de push (HTTP POST para o server):**

| Endpoint no server | Método | Payload | Frequência |
|-------------------|--------|---------|------------|
| `POST /api/agent/ingest/metrics` | POST | `{"node_id":"...","metrics":[...]}` | A cada `RESMA_COLLECT_INTERVAL` |
| `POST /api/agent/ingest/oom` | POST | `{"node_id":"...","ts":"...","service":"...","container_id":"...","exit_code":137}` | Imediato (event-driven) |
| `POST /api/agent/heartbeat` | POST | `{"node_id":"...","hostname":"...","containers_count":N,"version":"..."}` | A cada 30s |

**Auth:** Header `Authorization: Bearer <RESMA_AGENT_TOKEN>`. O token é comparado com `RESMA_AGENT_TOKEN` configurado no server. Simples e eficaz para rede overlay isolada.

**Resiliência (buffer + retry):**

O agent **NÃO pode perder dados** se o manager cair (manutenção, crash, rolling update). Estratégia:

| Mecanismo | Detalhe |
|-----------|---------|
| Buffer em memória | Ring buffer circular de 1000 pontos por container (cap ~10MB total) |
| Retry com backoff | Exponencial: 3s → 6s → 12s → 30s → 60s → 60s (cap em 60s) |
| Persistência OOM | OOM events são escritos em `/tmp/resma-oom-<ts>.json` antes do POST; removidos após ACK 2xx |
| Drop policy | Se buffer encher, descarta pontos mais antigos de stats (mantém OOMs) |
| Health do server | Agent faz `GET /health` no server antes do POST; se falhar, espera backoff |
| Compression | Payload enviado com `Content-Encoding: gzip` (JSON ~200B → ~60B comprimido) |

**Fluxo de retry:**
```
1. Agent coleta stats → adiciona ao buffer
2. Timer dispara POST /api/agent/ingest/metrics (gzip)
3. Se 2xx → remove do buffer, reseta backoff
4. Se 5xx ou timeout → mantém no buffer, aplica backoff
5. Se buffer > 90% capacidade → log WARNING + drop stats antigos (mantém OOMs)
6. OOM events: escreve em disco → POST → se falhar, mantém em disco → retry
```

**Imagem Docker:**
- Base: `scratch` (binário estático, CGO_ENABLED=0) ou `alpine` se precisar de CA certs
- Tamanho esperado: ~10-15 MB
- Multi-arch: amd64, arm64

**Resource limits (docker-stack.yml):**
```yaml
deploy:
  resources:
    limits:
      cpus: '0.10'
      memory: 64M
    reservations:
      cpus: '0.05'
      memory: 32M
```

#### 2. Ingestion API (novos endpoints no RESMA API existente)

Endpoints no Go API existente para receber dados dos agents. Sem JWT — auth via `RESMA_AGENT_TOKEN`.

**Novos endpoints:**

| Endpoint | Auth | Descrição |
|----------|------|-----------|
| `POST /api/agent/ingest/metrics` | Bearer token | Recebe batch de métricas de um agent |
| `POST /api/agent/ingest/oom` | Bearer token | Recebe evento OOM de um agent |
| `POST /api/agent/heartbeat` | Bearer token | Recebe heartbeat de um agent |
| `GET /api/agent/nodes` | JWT (admin) | Lista nodes agents com último heartbeat |
| `GET /api/agent/nodes/{node_id}` | JWT (admin) | Detalhes de um agent (status, containers, versão) |

**Registro em middleware separado** (como `/api/internal/*` — sem JWT, apenas token check):

```go
// server.go — registrar rotas /api/agent/* sem JWT
agentMux := http.NewServeMux()
s.registerAgentRoutes(agentMux)
mux.Handle("/api/agent/", s.corsMiddleware(s.recoveryMiddleware(
    s.loggingMiddleware(s.agentTokenMiddleware(agentMux)))))
```

**Novo schema DuckDB:**

```sql
CREATE TABLE IF NOT EXISTS agent_heartbeats (
    node_id      VARCHAR PRIMARY KEY,
    hostname     VARCHAR,
    ip_address   VARCHAR,
    containers_count INTEGER,
    agent_version VARCHAR,
    last_seen    TIMESTAMP,
    status       VARCHAR DEFAULT 'active'
);
```

A tabela `metrics` existente ganha uma coluna `node_id` (nullable para compatibilidade com dados existentes do manager):

```sql
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS node_id VARCHAR;
ALTER TABLE oom_events ADD COLUMN IF NOT EXISTS node_id VARCHAR;
```

**Tratamento de dados antigos (node_id NULL):**

Dados existentes (pré-Fase 7) terão `node_id = NULL`. Para evitar que queries agregadas misturem nodes incorretamente:

1. **Manager local:** O collector do manager preenche `node_id` com `RESMA_NODE_ID` (template var `{{.Node.ID}}`) em todos os inserts locais — dados novos do manager terão node_id preenchido
2. **Dados antigos:** Permanecem com `NULL` — são tratados como "node local/manager" implicitamente
3. **Queries existentes:** Não quebram (sintaxe SQL válida), mas queries agregadas por service vão somar containers de múltiplos nodes. Isso é **comportamento desejado** para recomendações de limite (queremos o total de uso do service no cluster)
4. **Queries novas (opcional):** Endpoints `/api/internal/*` podem expor `node_id` para o ML sidecar se necessário. Para a Fase 7, o ML trata o service como unidade (soma de todos os containers em todos os nodes) — não precisa distinguir por node
5. **Backfill (opcional):** Job one-off para preencher `node_id = 'manager'` em dados antigos: `UPDATE metrics SET node_id = 'manager' WHERE node_id IS NULL` — executado manualmente após deploy

**Rate limiting no server:**

| Limite | Valor | Resposta |
|--------|-------|----------|
| Payload máximo | 5 MB por POST | 413 Payload Too Large |
| Métricas por batch | 10.000 | 413 Payload Too Large |
| Requests por node | 2/segundo (burst 5) | 429 Too Many Requests |
| Heartbeat | 1 a cada 20s (janela 60s) | 429 Too Many Requests |
| Token inválido/ausente | — | 401 Unauthorized |

Implementação via middleware `agentRateLimitMiddleware` com contador em memória (map[node_id]limiter). Para clusters grandes (>50 nodes), considerar Redis — fora de escopo da Fase 7.

#### 3. Collector refactor — modo híbrido

O collector atual (`app/api/internal/collector/collector.go`) precisa ser refatorado para:

1. **No manager:** Continuar coletando stats locais (containers do manager) + Swarm metadata (nodes, tasks, cluster, storage)
2. **Receber dados dos agents:** Os novos endpoints `/api/agent/ingest/*` inserem no DuckDB diretamente
3. **Tracking de agents:** O collector verifica heartbeats e marca agents como `inactive` se não reportarem em 90s

**Não há mudança na lógica ML** — o recommender já recebe dados via `/api/internal/*` do DuckDB, que agora terá dados de todos os nodes.

### Fluxo de dados completo

```
1. Agent em Worker N coleta stats locais (a cada 15s)
   ↓
2. Agent faz POST /api/agent/ingest/metrics para RESMA API no manager
   ↓
3. RESMA API valida token, insere no DuckDB (com node_id)
   ↓
4. Collector no manager continua coletando stats locais do manager
   ↓
5. DuckDB agora tem dados de TODOS os nodes (manager local + agents remotos)
   ↓
6. ML sidecar solicita dados via /api/internal/* (já existente)
   ↓
7. Recommender analisa todos os containers de todos os nodes
   ↓
8. Dashboard mostra dados agregados do cluster inteiro
```

### Segurança

| Camada | Mecanismo |
|--------|-----------|
| Rede | Overlay network do Swarm (isolada, não roteável externamente) |
| Overlay encryption | **Não habilitada por padrão** (causa 5-15% overhead). Documentar como opt-in via `--opt encrypted` para ambientes multi-tenant |
| Auth agent→server | Bearer token (`RESMA_AGENT_TOKEN`) via Swarm secret |
| Replay protection | Payload inclui `ts` (timestamp); server rejeita payloads com `ts` > 60s de skew |
| Socket | Montado como `:ro` (read-only) no agent |
| TLS | Opcional — overlay já é isolada em datacenter privado; mTLS é futuro se demandar |
| Rate limiting | Ver seção "Rate limiting no server" acima |
| Token rotation | Swarm secret permite rotação: `docker secret rm` + `docker secret create` + `docker service update --secret-rm --secret-add` |

**Modelo de ameaça assumido:**
- **Confiamos** na rede overlay do Swarm (datacenter privado, nodes controlados pelo mesmo admin)
- **Não confiamos** em containers de aplicação (podem ser comprometidos) — por isso o token é via secret, não env var hardcoded
- **Não protege contra** admin malicioso com acesso ao manager (esse já tem acesso root ao cluster)

**Swarm secret para o token:**
```bash
echo "seu-token-seguro-aqui" | docker secret create resma_agent_token -
```

```yaml
# docker-stack.yml
agent:
  environment:
    RESMA_AGENT_TOKEN_FILE: /run/secrets/resma_agent_token
  secrets:
    - resma_agent_token
```

### Task lifecycle monitoring (slot stability + degraded service detection)

> Resolve o gap documentado na spec 0b (linha 661): "Swarm task monitoring — Swarm opera em tasks (pending/accepted/running/failed). Container só existe quando task está running. Task stuck em pending = serviço degradado invisível para RESMA."

#### Contexto: tasks vs containers no Swarm

O Docker Swarm tem uma hierarquia de execução:

```
Service (nginx, replicas: 6)
  ├── Task slot 1 → container abc123 (node-1)
  ├── Task slot 2 → container def456 (node-2)
  ├── Task slot 3 → container ghi789 (node-3)
  └── ...
```

- **Service**: workload definido (nginx, 6 réplicas)
- **Task**: slot individual com **slot number estável** (1, 2, 3...). Se a task slot 2 morre, o Swarm cria uma **nova task com slot 2** — o slot persiste através de restarts
- **Container**: instância efêmera (container ID muda a cada restart). Só existe quando a task está em estado `running`

**Problema atual:** O RESMA monitora apenas containers. Se um container reinicia, o `container_id` muda e o histórico de métricas "se perde" — impossível rastrear continuidade do mesmo slot. Tasks em estado `pending`/`accepted` (sem container) são invisíveis.

#### O que muda

**1. Coleta de task lifecycle (no manager, sem agent):**

O manager já faz `TaskList` para inventário (`services.go:56`, `nodes.go:89,123`). A Fase 7 adiciona persistência periódica do estado das tasks:

```sql
CREATE TABLE IF NOT EXISTS task_states (
    ts              TIMESTAMP,
    task_id         VARCHAR,
    service_id      VARCHAR,
    service_name    VARCHAR,
    slot            INTEGER,
    node_id         VARCHAR,
    desired_state   VARCHAR,   -- running, shutdown, accepted, pending
    current_state   VARCHAR,   -- running, complete, failed, rejected, orphaned
    status_message  VARCHAR,
    created_at      TIMESTAMP,
    PRIMARY KEY (ts, task_id)
);
```

Poll de `TaskList` a cada `RESMA_TASK_POLL_INTERVAL` (default 30s) no collector do manager. Não precisa de agent — `TaskList` é uma Swarm API global visível do manager.

**2. Correlação task ↔ container via TaskList (não parse de labels):**

O collector já chama `TaskList` periodicamente. Em vez de parsear o label `com.docker.swarm.task.name` (frágil para service names com pontos como `my.app.v2`), usar o campo nativo `task.Slot` do Docker SDK:

```go
// Em collector.go — construir mapa container_id → (task_id, slot)
taskResult, _ := cli.TaskList(ctx, types.TaskListOptions{})
containerToTask := make(map[string]struct{ TaskID string; Slot int })
for _, task := range taskResult {
    if task.Status.ContainerID != "" {
        containerToTask[task.Status.ContainerID] = struct{ TaskID string; Slot int }{
            TaskID: task.ID,
            Slot:   task.Slot,
        }
    }
}
// Ao inserir métricas de um container, lookup no mapa
```

**Nota:** Serviços em `global` mode (1 task por node) não têm slot number — `task.Slot` será 0. Usar `task_id` como identificador único nesses casos.

**Nota:** Containers non-Swarm (standalone) não aparecem em `TaskList` — `task_id` e `slot` serão NULL, tratados como "sem slot info".

O collector preenche novas colunas na tabela `metrics`:

```sql
ALTER TABLE metrics ADD COLUMN IF NOT EXISTS task_id VARCHAR, ADD COLUMN IF NOT EXISTS slot INTEGER;
ALTER TABLE oom_events ADD COLUMN IF NOT EXISTS task_id VARCHAR, ADD COLUMN IF NOT EXISTS slot INTEGER;
```

**3. Detecção de serviços degradados:**

Nova query no collector que compara `desired_state` vs `current_state` diretamente em `task_states` (sem precisar de tabela `services` que não existe no schema atual):

```sql
-- Serviços com tasks running < desired (degradado)
SELECT service_name,
       COUNT(*) FILTER (current_state = 'running') AS running,
       COUNT(*) FILTER (desired_state = 'running') AS desired
FROM task_states
WHERE ts = (SELECT MAX(ts) FROM task_states)
GROUP BY service_name
HAVING COUNT(*) FILTER (current_state = 'running') < COUNT(*) FILTER (desired_state = 'running');
```

Resultado exposto em novo endpoint:
- `GET /api/services/{service}/health` — retorna `{desired: 6, running: 4, pending: 1, failed: 1, slots: [...]}`
- `GET /api/internal/services/{service}/task-history` — histórico de restarts por slot (para o ML)

**4. ML sidecar — detecção de memory leak por slot:**

O recommender pode agora rastrear continuidade do mesmo slot através de restarts:

```python
# Antes (Fase 7 pré-task monitoring): container_id muda a cada restart
# → ML perde histórico, trata como container novo
# Depois: slot é estável
# → ML pode ver "slot 2 reiniciou 5x em 24h com memória crescente" = memory leak
```

Novo endpoint interno para o ML:
- `GET /api/internal/services/{service}/restarts` — retorna `[{slot: 2, restarts: 5, last_restart_ts: "..."}, ...]`

O ML sidecar usa isso para:
- Detectar memory leaks (slot com restarts frequentes + memória crescente)
- Ajustar recomendação de `limits.memory` (se slot reinicia por OOM, aumentar limite)
- Marcar serviços como "unstable" no dashboard

#### O que NÃO muda

- **Métricas continuam vindo de containers** — CPU/memória só existem em containers rodando
- **Recomendações continuam por service** — limits/reservations são aplicados por service no Swarm
- **ML sidecar não quebra** — novos endpoints são aditivos; endpoints existentes continuam funcionando
- **Single-node continua funcionando** — `TaskList` funciona mesmo com 1 node; slots existem mesmo em serviços replicados locais

#### Backward compatibility (task monitoring)

- Dados antigos: `task_id` e `slot` serão `NULL` — tratados como "sem slot info"
- Queries existentes que não usam `task_id`/`slot` continuam funcionando
- Backfill não é possível (slot não é derivável do container_id) — dados antigos permanecem NULL
- Tabela `task_states` começa vazia e é populada a partir do deploy da Fase 7
- **Retention policy:** `task_states` retém 7 dias (poll a cada 30s = ~20K registros/dia por 100 tasks). Job de cleanup diário: `DELETE FROM task_states WHERE ts < now()::TIMESTAMP - INTERVAL '7' DAY`
- **Serviços global mode:** Não têm slot number (`task.Slot = 0`); usar `task_id` como identificador único
- **Containers non-Swarm (standalone):** Não aparecem em `TaskList`; `task_id` e `slot` serão NULL

### Frontend — visualização de tasks e multi-node

> Baseado em benchmark de Swarmpit, Portainer (Cluster Visualizer), Cetacean (topology views) e padrões de Grafana dashboards para Docker Swarm.

#### Decisão de UX: 3 pontos de integração

Em vez de uma única page, a visualização de tasks é distribuída em 3 locais para contexto natural (padrão Portainer + Cetacean):

| Local | URL | Componente | Propósito |
|-------|-----|------------|-----------|
| **Page Tasks (nova)** | `/tasks` | `src/pages/Tasks.tsx` | Visão global de todas as tasks do cluster com filtros |
| **Aba Tasks em Service Detail** | `/services/:name` (tab) | `TasksTab` em `ServiceDetail.tsx` | Tasks daquele serviço específico + restart history |
| **Card Service Health no Dashboard** | `/` (card) | `ServiceHealthCard` em `Dashboard.tsx` | Overview rápido de desired vs running |

#### 1. Page Tasks (`/tasks`) — nova

**Item de menu** em `src/components/Layout.tsx` (array `navItems`, linha 12-19):
```typescript
{ to: "/tasks", label: "Tasks", icon: ListTodo, activeColor: "text-chart-4", activeBg: "bg-chart-4/10", barColor: "bg-chart-4" },
```

**Rota** em `src/App.tsx` (linha 47-57): adicionar `<Route path="/tasks" element={<Tasks />} />`

**Breadcrumbs** em `src/components/Layout.tsx` (`buildBreadcrumbs`, linha 21-49): adicionar caso `tasks → "Tasks"`

**Estrutura da page** (seguir padrão de `Services.tsx`):

```
┌─────────────────────────────────────────────────────────────┐
│  Tasks                                                        │
│  Tasks do Docker Swarm com slot stability e restart history   │
├─────────────────────────────────────────────────────────────┤
│  [Filtro: Service ▼] [Filtro: Node ▼] [Filtro: State ▼]     │
│  [Buscar task ID...]                                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─ Task State Distribution ─────────────────────────────┐  │
│  │  [Pie chart: running 80% / pending 10% / failed 10%]  │  │
│  └───────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  ┌─ Tabela de Tasks ─────────────────────────────────────┐  │
│  │ Service    │ Slot │ Node      │ State   │ Age  │ Restarts│
│  │ nginx      │ 1    │ worker-01 │ running │ 2h   │ 0       │
│  │ nginx      │ 2    │ worker-02 │ running │ 2h   │ 0       │
│  │ nginx      │ 3    │ worker-03 │ failed  │ 5m   │ 3       │
│  │ api        │ 1    │ manager   │ pending │ 1m   │ 0       │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Tipos TypeScript** (inline na page, padrão do projeto):
```typescript
interface Task {
  task_id: string
  service_name: string
  slot: number          // 0 para global mode services
  node_id: string
  node_hostname: string
  desired_state: string // running, shutdown
  current_state: string // running, pending, failed, rejected, shutdown, orphaned
  status_message: string
  created_at: string
  restart_count: number
}

interface TaskStateSummary {
  running: number
  pending: number
  failed: number
  rejected: number
  shutdown: number
  orphaned: number
  total: number
}
```

**Fetch de dados** (React Query, padrão do projeto):
```typescript
const { data: tasks, isLoading } = useQuery<Task[]>({
  queryKey: ["tasks"],
  queryFn: () => api.get<Task[]>("/tasks"),
  refetchInterval: fallbackInterval,
})

const { data: summary } = useQuery<TaskStateSummary>({
  queryKey: ["tasks-summary"],
  queryFn: () => api.get<TaskStateSummary>("/tasks/summary"),
})
```

**SSE** — adicionar tópico `tasks` em `src/hooks/use-event-source.ts` (linha 38-44):
```typescript
const TOPIC_QUERY_MAP: Record<string, string[][]> = {
  metrics: [["dashboard"], ["services"], ["services-sparklines"]],
  dashboard: [["dashboard"], ["storage-summary"]],
  events: [["oom-events"], ["change-log"]],
  services: [["services"], ["recommendations"], ["services-sparklines"]],
  nodes: [["nodes"], ["cluster"]],
  tasks: [["tasks"], ["tasks-summary"]],  // NOVO
}
```

Na page Tasks:
```typescript
const { isConnected: sseConnected } = useEventSource({
  topic: "tasks",
  invalidateQueries: [["tasks"], ["tasks-summary"]],
})
```

**State badges** (padrão `STATUS_CONFIG` de Services.tsx):
```typescript
const TASK_STATE_CONFIG: Record<string, { label: string; variant: BadgeVariant; dot: string }> = {
  running:   { label: "Running",   variant: "outline",   dot: "bg-green-500" },
  pending:   { label: "Pending",   variant: "warning",   dot: "bg-amber-500" },
  failed:    { label: "Failed",    variant: "destructive", dot: "bg-red-500" },
  rejected:  { label: "Rejected",  variant: "destructive", dot: "bg-orange-500" },
  shutdown:  { label: "Shutdown",  variant: "secondary", dot: "bg-muted-foreground" },
  orphaned:  { label: "Orphaned",  variant: "secondary", dot: "bg-muted-foreground" },
}
```

**Pie chart de state distribution** (recharts, padrão do projeto):
```tsx
<TaskStateDistribution data={summary} />  // PieChart com cores: green/amber/red/gray
```

**Filtros** (Zustand `filter-store.ts`, padrão do projeto):
- Filtro por service (Select dropdown)
- Filtro por node (Select dropdown)
- Filtro por state (Select dropdown com opções de TASK_STATE_CONFIG)
- Busca por task_id (Input)

#### 2. Aba Tasks em Service Detail

`src/pages/ServiceDetail.tsx` já usa `Tabs` (shadcn/ui). Adicionar tab "Tasks" ao `TabsList`:

```
[Overview] [Containers] [Métricas] [Recomendações] [Tasks ← NOVO]
```

Conteúdo da tab:
- **Tabela de tasks** daquele serviço (mesmo componente `TaskTable` da page Tasks, filtrado por service)
- **Restart History Chart** (recharts LineChart): restarts por slot ao longo do tempo (últimas 24h)
- **Service Health Card**: desired vs running com Progress bar

```tsx
<ServiceHealthCard service={service} />  // desired: 6, running: 4, failed: 1, pending: 1
<RestartHistoryChart service={serviceName} />  // LineChart: x=tempo, y=restarts por slot
<TaskTable tasks={serviceTasks} />
```

**Restart History Chart** (recharts):
```tsx
<LineChart data={restartHistory}>
  <XAxis dataKey="ts" />
  <YAxis />
  <Line dataKey="restarts" stroke="var(--color-destructive)" />
</LineChart>
```

Dados do endpoint: `GET /api/services/{service}/restarts` → `[{slot: 1, restarts: 3, last_restart_ts: "..."}, ...]`

#### 3. Card Service Health no Dashboard

`src/pages/Dashboard.tsx` — adicionar card "Service Health" no grid de cards:

```
┌─────────────────────────────────────┐
│  Service Health                      │
│  ┌────────┐ ┌────────┐ ┌────────┐   │
│  │  24    │ │  22    │ │   2    │   │
│  │desired │ │running │ │degraded│   │
│  └────────┘ └────────┘ └────────┘   │
│  [████████████████░░░░] 91%          │
└─────────────────────────────────────┘
```

Mostra agregado de todos os serviços: total desired, total running, total degraded (running < desired). Link para page Tasks.

#### 4. Multi-node status no frontend — coluna + card no NodeDetail

Agents não existem sem nodes (1 agent por node — global service da Fase 7). Em vez de nova page ou tab, integrar na estrutura existente (master-detail):

**4a. Coluna "Agent" na tabela de Nodes** (`src/pages/Nodes.tsx`):
- Nova coluna mostrando badge `active` (verde) / `inactive` (vermelho) + `last_seen` relativo (ex: "há 12s")
- Scan rápido: operador vê qual node parou de reportar sem clicar em nada
- Dados merged no backend (ver "Implementação do merge" abaixo) — frontend apenas usa campos extras na resposta de `/api/nodes`

**4b. Card "Agent" no NodeDetail** (`src/pages/NodeDetail.tsx`):
- NodeDetail já existe em `/nodes/:nodeId` — adicionar um card na layout existente
- Mostra: status (active/inactive badge), agent_version, containers_count, last_seen, erro se houver
- Drill-down natural: operador já está investigando um node suspeito
- Dados merged no backend em `/api/nodes/{node_id}` (mesma abordagem da lista)

**Implementação do merge (backend):**
O backend faz LEFT JOIN de `nodes` com `agent_heartbeats` em `GetNodes()` e `GetNodeByID()`:

```sql
-- GetNodes() modificado
SELECT n.*, 
       h.status AS agent_status, 
       h.last_seen AS agent_last_seen,
       h.agent_version AS agent_version,
       h.containers_count AS agent_containers_count
FROM nodes n
LEFT JOIN agent_heartbeats h ON h.node_id = n.node_id
ORDER BY n.hostname;

-- GetNodeByID() modificado (mesmo LEFT JOIN com WHERE n.node_id = ?)
```

Campos `agent_*` são nullable (nodes sem agent = single-node ou agent ainda não deployado). Frontend trata NULL como "sem agent" (badge cinza).

**Por que backend merge e não frontend merge:**
- Frontend fica simples (1 fetch, sem lógica de merge)
- Evita 2 round-trips HTTP
- Padrão já usado em Services.tsx (recs merge era no backend via `/api/services` que já incluía `current` resources)
- Nodes sem agent (backward compatibility) retornam `agent_status = NULL` naturalmente

**Endpoint `/api/agent/nodes` mantido:**
- Ainda útil para debugging/admin (ver heartbeats crus sem dados de nodes)
- Não é usado pelo frontend principal, mas pode ser usado por tooling futuro

**Endpoint `/api/agent/nodes/{node_id}` removido:**
- Redundante — dados já incluídos em `/api/nodes/{node_id}` após o merge

**Por que não tab/page separada:**
- Há exatamente 1 agent por node (global service) — uma "lista de agents" seria redundante com a lista de nodes
- Tab no nível da página mostraria agents sem contexto de node (pior que a coluna)
- Nova rota `/agents` seria um item de menu para algo que é 1:1 com nodes (poluição de navegação)
- Coluna + card reusam infra existente (Nodes.tsx tabela, NodeDetail.tsx cards) sem refatoração estrutural

#### Endpoints de API necessários (resumo)

| Endpoint | Auth | Frontend consumer |
|----------|------|-------------------|
| `GET /api/tasks` | JWT | Page Tasks (lista com filtros via query params) |
| `GET /api/tasks/summary` | JWT | Page Tasks (pie chart de state distribution) |
| `GET /api/services/health-summary` | JWT | Dashboard (card Service Health — agregado desired vs running de todos os serviços) |
| `GET /api/services/{service}/tasks` | JWT | Aba Tasks em Service Detail |
| `GET /api/services/{service}/restarts` | JWT | Restart History Chart |
| `GET /api/services/{service}/health` | JWT | Service Health Card (em ServiceDetail) |
| `GET /api/agent/nodes` | JWT | Debugging/admin (heartbeats crus) — não usado pelo frontend principal |
| `/api/nodes` (modificado) | JWT | Coluna Agent na tabela de Nodes (LEFT JOIN com agent_heartbeats) |
| `/api/nodes/{node_id}` (modificado) | JWT | Card Agent no NodeDetail (LEFT JOIN com agent_heartbeats) |

**Query params para `GET /api/tasks`:**
- `service=nginx` — filtra por service
- `node=worker-01` — filtra por node hostname
- `state=failed` — filtra por current_state
- `search=abc123` — busca por task_id (LIKE)

#### SSE — novos eventos

O server emite eventos SSE quando:
- Task muda de estado (pending → running, running → failed): tópico `tasks`
- Agent envia heartbeat ou fica inativo: tópico `agents` (novo)

Adicionar em `src/hooks/use-event-source.ts`:
```typescript
agents: [["agents"], ["nodes"]],  // NOVO — agents afetam view de Nodes
```

#### Arquivos frontend a criar/modificar

| Arquivo | Ação | Esforço |
|---------|------|---------|
| `src/pages/Tasks.tsx` | NOVO | 4h |
| `src/components/task-table.tsx` | NOVO (componente reutilizável) | 2h |
| `src/components/task-state-badge.tsx` | NOVO | 0.5h |
| `src/components/service-health-card.tsx` | NOVO | 1h |
| `src/components/restart-history-chart.tsx` | NOVO | 1.5h |
| `src/components/task-state-distribution.tsx` | NOVO (pie chart) | 1h |
| `src/pages/Tasks.tsx` | NOVO — page de tasks com filtros + pie chart | 4h |
| `src/pages/ServiceDetail.tsx` | Modificar — adicionar tab Tasks | 2h |
| `src/pages/Dashboard.tsx` | Modificar — adicionar card Service Health | 1h |
| `src/pages/Nodes.tsx` | Modificar — adicionar coluna Agent (badge + last_seen) na tabela | 2h |
| `src/pages/NodeDetail.tsx` | Modificar — adicionar card Agent (status, version, containers, last_seen) | 1.5h |
| `src/components/Layout.tsx` | Modificar — adicionar item Tasks no menu + breadcrumbs | 0.5h |
| `src/App.tsx` | Modificar — adicionar rota /tasks | 0.2h |
| `src/hooks/use-event-source.ts` | Modificar — adicionar tópicos tasks + agents | 0.3h |
| `src/stores/filter-store.ts` | Modificar — adicionar filtros de tasks | 0.5h |
| **Total frontend** | | **17h** |

### Backward compatibility

- **Single-node (atual):** Se não houver agents, o RESMA continua funcionando exatamente como hoje — coleta apenas do manager. Os endpoints `/api/agent/*` simplesmente não recebem dados.
- **Dados existentes:** A coluna `node_id` é nullable — dados antigos (sem node_id) continuam válidos e são tratados como "node local/manager".
- **ML sidecar:** Nenhuma mudança necessária — já consome via `/api/internal/*`.
- **Frontend:** Mudanças são aditivas — nova page Tasks, nova tab em ServiceDetail, novo card no Dashboard, nova coluna em Nodes, novo card em NodeDetail. Pages existentes não são refatoradas estruturalmente.

### Rollback strategy

Se a Fase 7 causar problemas em produção, reverter nesta ordem:

1. **Remover agents:** `docker service rm resma-agent` — para coleta de workers imediatamente
2. **Server continua funcionando:** O Go API não quebra sem agents — endpoints `/api/agent/*` apenas não recebem dados
3. **Dados coletados:** Permanecem no DuckDB com `node_id` preenchido — não são apagados
4. **Reverter código do server (se necessário):** `git revert` do commit da Fase 7 — as colunas `node_id` extras são harmless (nullable, não usadas se não há agents)
5. **Schema:** NÃO fazer `ALTER TABLE DROP COLUMN` — DuckDB suporta mas é desnecessário e arriscado. Colunas nullable não usadas não causam overhead significativo
6. **Downtime:** Zero — remoção dos agents não afeta o manager

**Rollback é seguro porque:**
- O agent é aditivo (não modifica lógica existente do collector do manager)
- As colunas `node_id` são nullable (não quebram queries)
- Os endpoints `/api/agent/*` são isolados (não afetam `/api/*` ou `/api/internal/*`)

## Tarefas

### 7.1 — Criar RESMA Agent binary

- **Arquivo:** `app/agent/` (novo módulo Go no monorepo)
- **Estrutura:**
  ```
  app/agent/
  ├── cmd/agent/main.go          # Entry point
  ├── internal/
  │   ├── collector.go           # Stats collector (stream + cache, mesmo padrão do server)
  │   ├── events.go              # Event listener (OOM, start/stop) + persistência em disco
  │   ├── pusher.go              # HTTP client com gzip + retry backoff
  │   ├── buffer.go              # Ring buffer circular em memória
  │   ├── heartbeat.go           # Heartbeat periódico
  │   └── config.go              # Config via env vars
  ├── Dockerfile                 # Multi-stage, scratch base
  ├── go.mod                     # Módulo separado (sem DuckDB, sem CGO)
  └── .air.toml                  # Hot reload para dev
  ```
- **Dependências Go:** `moby/moby/client`, `net/http` (stdlib), `compress/gzip` (stdlib) — mínimo possível
- **CGO_ENABLED=0** — binário estático puro, sem dependência de gcc
- **Padrão de coleta:** Stream persistente (`ContainerStats(stream=true)`) + cache em memória (igual ao `app/api/internal/docker/stats.go`), lido a cada `RESMA_COLLECT_INTERVAL` para montar o batch
- **Esforço:** 7h (aumentou de 6h para incluir buffer + retry + gzip)

### 7.2 — Criar Ingestion API no Go API

- **Arquivo:** `app/api/internal/server/agent_handlers.go` (novo)
- **Endpoints:** `POST /api/agent/ingest/metrics`, `POST /api/agent/ingest/oom`, `POST /api/agent/heartbeat`
- **Auth middleware:** `agentTokenMiddleware` — valida `Authorization: Bearer <token>` contra `RESMA_AGENT_TOKEN`
- **Queries DuckDB:** `InsertMetricsBatchWithNode`, `InsertOOMEventWithNode`, `UpsertAgentHeartbeat`, `InsertTaskStates`, `GetServiceHealth`, `GetServiceRestartHistory`
- **Schema migration:** `ALTER TABLE metrics ADD COLUMN node_id, task_id, slot`, `ALTER TABLE oom_events ADD COLUMN node_id, task_id, slot`, `CREATE TABLE agent_heartbeats`, `CREATE TABLE task_states`
- **Registrar rotas em server.go** (sem JWT, com agentTokenMiddleware)
- **Rate limiting middleware:** `agentRateLimitMiddleware` com limites de payload (5MB), requests/s (2 por node), heartbeat (1/20s)
- **Replay protection:** Validar `ts` no payload; rejeitar skew > 60s
- **Esforço:** 5h

### 7.3 — Refatorar Collector para modo híbrido + task lifecycle

- **Arquivo:** `app/api/internal/collector/collector.go`
- **Mudança 1:** O `collectOnce()` adiciona `node_id` (do manager) aos MetricRows locais
- **Mudança 2:** Extrair `task_id` e `slot` via `TaskList` (campo nativo `task.Slot` do Docker SDK, não parse de labels) e correlacionar com `container_id` via `task.Status.ContainerID`. Preencher colunas novas em `metrics` e `oom_events`
- **Mudança 3:** Adicionar campo `Labels map[string]string` ao tipo `ContainerInfo` em `app/api/internal/docker/types.go` e atualizar `parseContainer()`/`parseInspect()` para populá-lo (necessário para correlação task↔container)
- **Nova goroutine 1:** `agentHealthCheckLoop` — verifica heartbeats a cada 30s, marca agents inativos após 90s sem heartbeat
- **Nova goroutine 2:** `taskStatePollLoop` — poll de `TaskList` a cada `RESMA_TASK_POLL_INTERVAL` (default 30s), persiste em `task_states`, detecta serviços degradados (running < desired)
- **Config:** `RESMA_NODE_ID`, `RESMA_NODE_HOSTNAME`, `RESMA_TASK_POLL_INTERVAL` no server (template vars do Swarm)
- **Esforço:** 4h (aumentou de 2h para incluir task lifecycle + slot extraction)

### 7.4 — Dockerfile + docker-compose + docker-stack

- **`app/agent/Dockerfile`:** Multi-stage Go build, `FROM scratch`, CGO_ENABLED=0
- **`docker-compose.yml` (dev):** Novo serviço `agent-dev` com `mode: global`, mount do socket `:ro`, `RESMA_API_URL=http://go-dev:8080`
- **`docker-stack.yml` (prod):** Novo serviço `resma-agent` com `mode: global`, placement sem constraint (roda em todos os nodes), secret para token
- **Rede overlay:** Criar rede `resma-net` (overlay) se não existir, conectar `api` e `agent`
- **Excluir agent da coleta:** Adicionar `resma-agent:latest` ao `RESMA_EXCLUDED_IMAGES` default
- **Esforço:** 3h

### 7.5 — Endpoints de status dos agents + task health + tasks (admin)

- **Arquivo:** `app/api/internal/server/agent_handlers.go` + `app/api/internal/server/service_handlers.go` + `app/api/internal/server/task_handlers.go` (novo)
- **Endpoints agents:** `GET /api/agent/nodes` (heartbeats crus para debugging/admin)
- **Modificação de endpoints existentes:** `GET /api/nodes` e `GET /api/nodes/{node_id}` fazem LEFT JOIN com `agent_heartbeats` para incluir campos `agent_status`, `agent_last_seen`, `agent_version`, `agent_containers_count` (nullable)
- **Endpoints task health:** `GET /api/services/{service}/health` (desired vs running), `GET /api/services/{service}/restarts` (restarts por slot para frontend)
- **Endpoints tasks (frontend):** `GET /api/tasks` (lista com filtros: service, node, state, search), `GET /api/tasks/summary` (agregado por state para pie chart), `GET /api/services/{service}/tasks` (tasks de um service)
- **Endpoints internos (ML):** `GET /api/internal/services/{service}/task-history`, `GET /api/internal/services/{service}/restarts`
- **Auth:** JWT (admin) para `/api/*`; sem auth (rede Docker) para `/api/internal/*`
- **SSE:** Emitir eventos tópicos `tasks` (mudança de estado) e `agents` (heartbeat/inactive)
- **Queries:** `SELECT * FROM agent_heartbeats`, `GetServiceHealth`, `GetServiceRestartHistory`, `GetTasksWithFilters`, `GetTaskStateSummary`, `GetServiceTasks`
- **Esforço:** 4h (aumentou de 2h para incluir endpoints de tasks + SSE)

### 7.6 — Testes multi-node

- **Setup:** Criar `docker-compose.multi-node.yml` com Docker-in-Docker simulando 1 manager + 2 workers (usando `docker:dind` images)
- **Testes:**
  - Agent em cada worker reporta stats corretamente
  - OOM em worker é detectado pelo agent e enviado ao server
  - Heartbeat mostra todos os nodes como `active`
  - ML recommender analisa dados de todos os nodes
  - Agent cai → server marca como `inactive` após 90s
  - Token inválido → server rejeita com 401
  - **Task monitoring:** service com 6 replicas, matar 1 container → task_states mostra `desired: running, current: failed` → endpoint `/api/services/{service}/health` mostra `running: 5, desired: 6`
  - **Slot stability:** restartar service → slot number persiste, container_id muda → métricas continuam associadas ao mesmo slot
  - **Memory leak detection:** simular slot com restarts frequentes → ML detecta via `/api/internal/services/{service}/restarts`
- **Smoke test:** Estender `cmd/smoke-test/main.go` com testes dos endpoints `/api/agent/*` e `/api/services/{service}/health`
- **Esforço:** 5h (aumentou de 4h para incluir testes de task lifecycle)

### 7.7 — Documentação

- **`docs-site/docs/guides/multi-node.md`:** Guia de deploy em cluster multi-node
- **`docs-site/docs/guides/task-monitoring.md`:** Guia de task monitoring e slot stability
- **`docs-site/docs/architecture.md`:** Atualizar diagrama de arquitetura com agent + tasks
- **`AGENTS.md`:** Atualizar stack table com "RESMA Agent (Go) — coleta stats de workers" + "Task lifecycle monitoring"
- **`README.md`:** Atualizar "2 containers" → "2 + N containers (1 agent por node)"
- **Esforço:** 2h

### 7.8 — Frontend: Tasks + Service Health + Agents

- **Page Tasks (nova):** `src/pages/Tasks.tsx` — tabela de tasks com filtros (service, node, state, search) + pie chart de state distribution
- **Componentes novos:**
  - `src/components/task-table.tsx` — tabela reutilizável (Task interface, state badges, slot, node, restarts)
  - `src/components/task-state-badge.tsx` — Badge com TASK_STATE_CONFIG (running/pending/failed/rejected/shutdown/orphaned)
  - `src/components/service-health-card.tsx` — Card com desired vs running + Progress bar
  - `src/components/restart-history-chart.tsx` — LineChart (recharts) de restarts por slot
  - `src/components/task-state-distribution.tsx` — PieChart (recharts) de distribuição de states
- **Modificações em pages existentes:**
  - `src/pages/ServiceDetail.tsx` — adicionar tab "Tasks" com TaskTable + RestartHistoryChart + ServiceHealthCard
  - `src/pages/Dashboard.tsx` — adicionar card "Service Health" (agregado de todos os serviços via `/api/services/health-summary`)
  - `src/pages/Nodes.tsx` — adicionar coluna "Agent" na tabela (badge active/inactive + last_seen relativo). Merge dos dados de `/api/agent/nodes` com a lista de nodes por `node_id`
  - `src/pages/NodeDetail.tsx` — adicionar card "Agent" (status, agent_version, containers_count, last_seen, erro). Endpoint `/api/agent/nodes/{node_id}`
- **Modificações em infra:**
  - `src/components/Layout.tsx` — adicionar item "Tasks" no navItems + caso em buildBreadcrumbs
  - `src/App.tsx` — adicionar rota `/tasks`
  - `src/hooks/use-event-source.ts` — adicionar tópicos `tasks` e `agents` ao TOPIC_QUERY_MAP
  - `src/stores/filter-store.ts` — adicionar filtros de tasks (service, node, state, search)
- **Padrões a seguir (verificados no código atual):**
  - Fetch: `useQuery` com `queryKey` + `refetchInterval` + SSE invalidation
  - Tabela: shadcn/ui Table (não @tanstack/react-table)
  - Gráficos: recharts (LineChart, PieChart)
  - Badges: STATUS_CONFIG pattern com `dot` color + `variant`
  - Loading: Skeleton; Empty: EmptyState
  - Filtros: Zustand filter-store
  - Tema: dark mode, tokens oklch (chart-4 para tasks)
- **Esforço:** 17h (ver tabela detalhada na seção "Frontend")

## Estrutura de arquivos resultante

```
app/
├── api/                    # Go API existente (manager)
│   └── internal/
│       ├── server/
│       │   ├── agent_handlers.go      # NOVO — endpoints /api/agent/* + task health
│       │   ├── task_handlers.go       # NOVO — endpoints /api/tasks, /api/tasks/summary
│       │   ├── service_handlers.go    # Modificado — endpoint /api/services/{service}/health + /tasks + /restarts
│       │   └── server.go              # Modificado — registrar rotas agent + tasks + SSE topics
│       ├── collector/
│       │   └── collector.go           # Modificado — node_id + slot extraction + task poll + health check
│       ├── db/
│       │   ├── queries.go             # Modificado — queries com node_id, task_id, slot + task_states + restarts + task filters
│       │   └── schema.go              # Modificado — ALTER TABLE (node_id, task_id, slot) + agent_heartbeats + task_states
│       └── config/
│           └── config.go              # Modificado — RESMA_AGENT_TOKEN + RESMA_TASK_POLL_INTERVAL
├── ml/                     # Python ML sidecar (sem mudanças obrigatórias; opcional: usar /api/internal/services/{service}/restarts)
├── agent/                  # NOVO — RESMA Agent
│   ├── cmd/agent/main.go
│   ├── internal/
│   │   ├── collector.go
│   │   ├── events.go
│   │   ├── buffer.go
│   │   ├── pusher.go
│   │   ├── heartbeat.go
│   │   └── config.go
│   ├── Dockerfile
│   ├── go.mod
│   └── .air.toml
└── frontend/               # React frontend (modificado)
    └── src/
        ├── pages/
        │   ├── Tasks.tsx              # NOVO — page de tasks com filtros + pie chart
        │   ├── ServiceDetail.tsx      # Modificado — tab Tasks + ServiceHealthCard + RestartHistoryChart
        │   ├── Dashboard.tsx          # Modificado — card Service Health
        │   ├── Nodes.tsx              # Modificado — coluna Agent (badge + last_seen) na tabela
        │   └── NodeDetail.tsx         # Modificado — card Agent (status, version, containers, last_seen)
        ├── components/
        │   ├── task-table.tsx         # NOVO — tabela reutilizável de tasks
        │   ├── task-state-badge.tsx   # NOVO — badge de state (running/pending/failed/...)
        │   ├── service-health-card.tsx # NOVO — card desired vs running
        │   ├── restart-history-chart.tsx # NOVO — LineChart de restarts por slot
        │   ├── task-state-distribution.tsx # NOVO — PieChart de distribuição de states
        │   ├── Layout.tsx             # Modificado — item Tasks no menu + breadcrumbs
        │   └── ...
        ├── App.tsx                    # Modificado — rota /tasks
        ├── hooks/
        │   └── use-event-source.ts    # Modificado — tópicos tasks + agents
        └── stores/
            └── filter-store.ts        # Modificado — filtros de tasks
```

## Docker Stack (produção)

```yaml
# docker-stack.yml (excerpt — serviço agent adicionado)
services:
  api:
    # ... existente ...
    environment:
      - RESMA_AGENT_TOKEN_FILE=/run/secrets/resma_agent_token
    secrets:
      - resma_agent_token
    networks:
      - resma-net

  resma-ml:
    # ... existente ...
    networks:
      - resma-net

  resma-agent:
    image: ghcr.io/${RESMA_REGISTRY:-user}/resma-agent:latest
    deploy:
      mode: global
      resources:
        limits:
          cpus: '0.10'
          memory: 64M
        reservations:
          cpus: '0.05'
          memory: 32M
      restart_policy:
        condition: on-failure
        max_attempts: 3
    environment:
      - RESMA_API_URL=http://api:8080
      - RESMA_AGENT_TOKEN_FILE=/run/secrets/resma_agent_token
      - RESMA_NODE_ID={{.Node.ID}}
      - RESMA_NODE_HOSTNAME={{.Node.Hostname}}
      - RESMA_EXCLUDED_IMAGES=ghcr.io/${RESMA_REGISTRY:-user}/resma-api:latest,ghcr.io/${RESMA_REGISTRY:-user}/resma-ml:latest,ghcr.io/${RESMA_REGISTRY:-user}/resma-agent:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    secrets:
      - resma_agent_token
    networks:
      - resma-net
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8082/health"]
      interval: 30s
      timeout: 5s
      retries: 3

networks:
  resma-net:
    driver: overlay

secrets:
  resma_agent_token:
    external: true
```

## Comparação final com mercado

| Aspecto | RESMA (proposto) | Swarmpit | Portainer | cAdvisor+Prometheus |
|---------|------------------|----------|-----------|---------------------|
| Agent imagem | ~10-15 MB | ~6.3 MB | ~50 MB | ~70+22 MB |
| Agent RAM | 64 MB | 64 MB | 20 MB | 100+180 MB |
| Agent CPU | 0.10 | 0.10 | 0.1% | 0.2-70% |
| Protocolo | HTTP POST | HTTP POST | HTTP proxy | Prometheus pull |
| Auth | Bearer token | Nenhuma | Assinatura digital | Nenhuma (rede) |
| Cluster entre agents | Não | Não | Serf | Não |
| Self-contained | **Sim** | Não (InfluxDB) | Não (DB interna) | **Não** (Prometheus) |
| Complexidade | **Baixa** | Baixa | Alta | Alta |
| Multi-arch | amd64, arm64 | amd64, arm64, arm/v7 | amd64, arm64, arm/v7, Windows | amd64, arm64 |

**Diferencial do RESMA:** Único que combina agent leve (estilo Swarmpit) + self-contained (sem Prometheus/InfluxDB) + ML recommendations + DuckDB embedded.

## Riscos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Overlay network latência afeta coleta | Baixa | Baixo | 0.4-0.7ms é negligenciável para 15s de intervalo |
| Agent consome mais RAM que esperado | Média | Médio | Limites hard de 64MB; monitorar com `docker stats` |
| Token comprometido | Baixa | Alto | Swarm secret (não env var); rotacionar periodicamente |
| Agent cai e não reinicia | Baixa | Médio | `restart_policy: on-failure` + healthcheck + alerta no dashboard |
| Schema migration quebra dados | Baixa | Alto | `ALTER TABLE ADD COLUMN IF NOT EXISTS` (nullable) |
| Docker API version mismatch | Média | Baixo | Agent usa `client.WithAPIVersionNegotiation()` |
| DinD (Docker-in-Docker) para testes | Média | Baixo | Usar `docker:dind` com privileged apenas em CI |

## Critérios de aceite

- [ ] Agent binary compila com `CGO_ENABLED=0` (sem dependência de gcc)
- [ ] Imagem do agent < 20 MB
- [ ] Agent coleta stats de containers locais e envia via HTTP POST
- [ ] Agent detecta OOM (exit 137) e envia evento imediatamente
- [ ] Agent envia heartbeat a cada 30s
- [ ] Server valida token do agent e rejeita tokens inválidos (401)
- [ ] Server insere métricas com `node_id` no DuckDB
- [ ] Collector no manager marca agents inativos após 90s sem heartbeat
- [ ] `GET /api/agent/nodes` retorna lista de agents com status
- [ ] ML recommender analisa dados de todos os nodes (não só manager)
- [ ] Dashboard mostra containers de todos os nodes
- [ ] `docker stack deploy` sobe 3 serviços: api + ml + agent (global)
- [ ] Single-node continua funcionando sem agents (backward compatible)
- [ ] Agent usa ring buffer circular (1000 pontos por container, cap ~10MB)
- [ ] Agent implementa retry com backoff exponencial (3s → 60s cap)
- [ ] Agent persiste OOM events em disco antes do POST (não perde se manager cair)
- [ ] Payload enviado com `Content-Encoding: gzip`
- [ ] Server aplica rate limiting (payload max 5MB, 2 req/s por node, heartbeat 1/20s)
- [ ] Server rejeita payloads com timestamp skew > 60s (replay protection)
- [ ] Rollback funciona: `docker service rm resma-agent` sem afetar o manager
- [ ] Collector extrai `task_id` e `slot` dos labels `com.docker.swarm.task.*` em containers
- [ ] Tabela `task_states` é populada a cada `RESMA_TASK_POLL_INTERVAL` (default 30s)
- [ ] Endpoint `GET /api/services/{service}/health` retorna desired vs running tasks
- [ ] Endpoint `GET /api/internal/services/{service}/restarts` retorna restarts por slot
- [ ] Slot stability: restart de service mantém slot number, muda container_id, métricas continuam associadas ao slot
- [ ] ML sidecar pode detectar memory leak por slot (restarts frequentes + memória crescente)
- [ ] Serviço degradado (running < desired) é detectado e exposto no endpoint de health
- [ ] Page Tasks (`/tasks`) renderiza tabela com filtros (service, node, state, search) + pie chart
- [ ] Tab "Tasks" em ServiceDetail mostra tasks do serviço + restart history chart + service health card
- [ ] Card "Service Health" no Dashboard mostra agregado desired vs running
- [ ] Coluna "Agent" na tabela de Nodes mostra badge active/inactive + last_seen relativo
- [ ] Card "Agent" no NodeDetail (`/nodes/:nodeId`) mostra status, agent_version, containers_count, last_seen
- [ ] State badges (running/pending/failed/rejected/shutdown/orphaned) com cores corretas
- [ ] SSE tópicos `tasks` e `agents` invalidam queries no frontend
- [ ] Item "Tasks" no menu lateral com ícone e cor distintos
- [ ] Breadcrumbs funcionam para `/tasks` e `/services/:name` (tab Tasks)
- [ ] Smoke test estendido cobre endpoints `/api/agent/*`
- [ ] Documentação de multi-node no Docusaurus

## Dependências

- **Depende de:** Fase 0b (Go API + collector + DuckDB + `/api/internal/*`), Fase 4 (docker-stack.yml)
- **Não bloqueia:** Nenhuma fase existente — pode ser implementada em paralelo com Fases 3, 5, 6
- **Desbloqueia:** Monitoramento real em clusters Swarm multi-node (o caso de uso principal do produto)

## Esforço total estimado

| Tarefa | Esforço |
|--------|---------|
| 7.1 Agent binary (com buffer + retry + gzip) | 7h |
| 7.2 Ingestion API (com rate limiting + task queries) | 5h |
| 7.3 Collector refactor (híbrido + task lifecycle + slot via TaskList) | 5h |
| 7.4 Docker + compose | 3h |
| 7.5 Endpoints (agents + task health + tasks + SSE) | 4h |
| 7.6 Testes multi-node (com task lifecycle + slot stability) | 5h |
| 7.7 Documentação (com task monitoring guide) | 2h |
| 7.8 Frontend (Tasks page + coluna Agent em Nodes + card em NodeDetail + Service Health) | 17h |
| **Total** | **48h (~6 dias)** |

## Revisão crítica (Critic) — resoluções

A spec passou por revisão crítica técnica. Problemas identificados e resolvidos:

| Problema | Severidade | Resolução |
|----------|-----------|-----------|
| Sem buffer local — perda de dados se manager cair | CRÍTICO | Seção "Resiliência" adicionada com ring buffer + retry backoff + persistência OOM em disco |
| Tratamento de node_id NULL — queries agregam cross-node | ALTO | Seção "Tratamento de dados antigos" — comportamento desejado para recomendações (soma do service no cluster) |
| Rate limiting incompleto | ALTO | Seção "Rate limiting no server" com limites em bytes, requests/s, e por-node |
| Sem rollback strategy | ALTO | Seção "Rollback strategy" adicionada — rollback é seguro e sem downtime |
| Overlay não criptografada por padrão | MÉDIO | Decisão documentada: não habilitada por padrão (5-15% overhead); opt-in para multi-tenant |
| Sem compressão | MÉDIO | `Content-Encoding: gzip` adicionado ao pusher |
| Replay protection | MÉDIO | Payload inclui `ts`; server rejeita skew > 60s |
| Stream vs point-in-time não especificado | BAIXO | Especificado: stream persistente + cache (mesmo padrão do server) |
| Heartbeat 30s lento | BAIXO | Mantido em 30s (aceitável para 90s timeout); ajustável via env var no futuro |
| Task monitoring gap (spec 0b linha 661) | ALTO | Seção "Task lifecycle monitoring" adicionada — coleta de `TaskList` + slot stability + detecção de serviços degradados + memory leak por slot |
| Slot extraction de containers | MÉDIO | Labels `com.docker.swarm.task.id` e `com.docker.swarm.task.name` extraídos em `collectOnce()`; colunas `task_id` e `slot` adicionadas a `metrics` e `oom_events` |
| Backward compatibility de task_id/slot NULL | MÉDIO | Colunas nullable; dados antigos tratados como "sem slot info"; queries existentes não quebram |
| Frontend não cobria tasks/multi-node | ALTO | Seção "Frontend" adicionada: page Tasks, tab Tasks em ServiceDetail, card Service Health no Dashboard, coluna Agent em Nodes + card Agent em NodeDetail (1 agent por node — não faz sentido page/tab separada), 5 componentes novos, SSE topics tasks+agents |
| Endpoints de tasks para frontend faltavam | ALTO | Tarefa 7.5 expandida: GET /api/tasks (com filtros), /api/tasks/summary, /api/services/health-summary (Dashboard card), /api/services/{service}/tasks, /api/services/{service}/restarts |
| Dashboard card precisava de desired vs running | MÉDIO | Endpoint `/api/services/health-summary` adicionado (não confundir com `/api/tasks/summary` que é agregado por state) |
| Merge de dados nodes+agents não especificado | ALTO | Backend faz LEFT JOIN em GetNodes() e GetNodeByID() com agent_heartbeats; frontend usa campos extras (agent_status, agent_last_seen, agent_version, agent_containers_count nullable). Endpoint /api/agent/nodes/{node_id} removido (redundante) |
| Esforço subestimado sem frontend | ALTO | Tarefa 7.8 adicionada (17h frontend); total 29h → 48h (~6 dias) |
