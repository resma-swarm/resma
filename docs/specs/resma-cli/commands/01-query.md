# RESMA CLI — Commands: Query (Read-Only)

> **Status:** Referência de comandos de consulta — 38 comandos read-only

Este documento descreve todos os comandos **read-only** do `resma-cli` — aqueles
que apenas consultam dados do RESMA API sem modificar estado. Todos os comandos
de consulta exigem autenticação **JWT** e estão disponíveis para qualquer
usuário autenticado (`RBAC: any`), salvo onde indicado.

Nomes de comandos, flags e parâmetros estão em **American English**. Descrições
prose estão em **Português (Brasil)**.

---

## Sumário

| # | Command | Endpoint | Auth | Description |
|---|---------|----------|------|-------------|
| 1 | `resma dashboard` | `GET /api/dashboard` | JWT | Visão geral consolidada do cluster |
| 2 | `resma services list` | `GET /api/services` | JWT | Lista todos os serviços monitorados |
| 3 | `resma services inspect <name>` | `GET /api/services/{name}/stats` | JWT | Estatísticas detalhadas de um serviço |
| 4 | `resma services metrics <name>` | `GET /api/services/{name}/metrics` | JWT | Séries temporais de métricas de um serviço |
| 5 | `resma services containers <name>` | `GET /api/services/{name}/containers` | JWT | Containers associados a um serviço |
| 6 | `resma services sparklines` | `GET /api/services/sparklines` | JWT | Sparklines de CPU/memória por serviço |
| 7 | `resma services health` | `GET /api/services/health` | JWT | Health-check de todos os serviços |
| 8 | `resma containers inspect <id>` | `GET /api/services/containers/{id}/stats` | JWT | Estatísticas de um container específico |
| 9 | `resma containers metrics <id>` | `GET /api/services/containers/{id}/metrics` | JWT | Métricas históricas de um container |
| 10 | `resma containers network <id>` | `GET /api/services/containers/{id}/network-info` | JWT | Informações de rede de um container |
| 11 | `resma nodes list` | `GET /api/nodes` | JWT | Lista todos os nodes do Swarm |
| 12 | `resma nodes cluster` | `GET /api/nodes/cluster` | JWT | Visão consolidada do cluster de nodes |
| 13 | `resma nodes inspect <node-id>` | `GET /api/nodes/{node_id}` | JWT | Detalhes de um node específico |
| 14 | `resma nodes metrics <node-id>` | `GET /api/nodes/{node_id}/metrics` | JWT | Métricas de um node |
| 15 | `resma nodes services <node-id>` | `GET /api/nodes/{node_id}/services` | JWT | Serviços rodando em um node |
| 16 | `resma agents list` | `GET /api/agents` | JWT | Lista todos os RESMA Agents |
| 17 | `resma agents inspect <node-id>` | `GET /api/agents/{node_id}` | JWT | Detalhes de um agent por node |
| 18 | `resma tasks list` | `GET /api/tasks` | JWT | Lista tasks do Swarm |
| 19 | `resma tasks show <service>` | `GET /api/tasks/{service}` | JWT | Tasks de um serviço específico |
| 20 | `resma tasks history <service>` | `GET /api/tasks/{service}/history` | JWT | Histórico de tasks de um serviço |
| 21 | `resma recommendations list` | `GET /api/recommendations` | JWT | Lista recomendações de limites |
| 22 | `resma recommendations show <service>` | `GET /api/recommendations/{service}` | JWT | Recomendações de um serviço |
| 23 | `resma recommendations triggers` | `GET /api/recommendations/triggers` | JWT | Gatilhos de recomendações |
| 24 | `resma recommendations storage` | `GET /api/recommendations/storage` | JWT | Recomendações de storage |
| 25 | `resma recommendations simulate` | `POST /api/recommendations/simulate` | JWT | Simulação de recomendações por tier |
| 26 | `resma rollback-watches list` | `GET /api/rollback-watches` | JWT | Lista monitors de rollback |
| 27 | `resma rollback-watches inspect <id>` | `GET /api/rollback-watches/{id}` | JWT | Detalhes de um monitor de rollback |
| 28 | `resma schedules list` | `GET /api/schedules` | JWT | Lista agendamentos |
| 29 | `resma schedules pending` | `GET /api/schedules/pending` | JWT | Agendamentos pendentes |
| 30 | `resma schedules history` | `GET /api/schedules/history` | JWT | Histórico de agendamentos |
| 31 | `resma templates list` | `GET /api/templates` | JWT | Lista templates de recursos |
| 32 | `resma templates inspect <name>` | `GET /api/templates/{name}` | JWT | Detalhes de um template |
| 33 | `resma storage summary` | `GET /api/storage/summary` | JWT | Resumo de uso de storage |
| 34 | `resma storage trend` | `GET /api/storage/trend` | JWT | Tendência de storage ao longo do tempo |
| 35 | `resma storage volumes` | `GET /api/storage/volumes/growth` | JWT | Crescimento de todos os volumes |
| 36 | `resma storage volume <name>` | `GET /api/storage/volumes/{name}/growth` | JWT | Crescimento de um volume específico |
| 37 | `resma alerts` | `GET /api/alerts` | JWT | Lista alertas ativos |
| 38 | `resma oom-events` | `GET /api/oom-events` | JWT | Lista eventos de OOM |
| 39 | `resma change-log` | `GET /api/change-log` | JWT | Log de mudanças de configuração |

---

## 1. `resma dashboard`

**Syntax:**
```bash
resma dashboard
```

**Description (PT-BR):** Exibe uma visão geral consolidada do cluster, incluindo
número de serviços, nodes, agents ativos, alertas pendentes e recomendações
recentes. É o ponto de entrada padrão para inspeção rápida do estado do Swarm.

**API Endpoint:** `GET /api/dashboard`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma dashboard
```

**Example output:**
```
RESMA Dashboard — 2025-01-15 14:32
────────────────────────────────────
Services:    12   Nodes:      5
Agents:       5   Alerts:     3
Recommendations (pending): 7

Top CPU:    api (78%)  worker-3 (62%)
Top Memory: ml (89%)   api (71%)
```

---

## 2. `resma services list`

**Syntax:**
```bash
resma services list
```

**Description (PT-BR):** Lista todos os serviços monitorados pelo RESMA no
Swarm, com informações resumidas de CPU, memória e status atual.

**API Endpoint:** `GET /api/services`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma services list
```

**Example output:**
```
NAME              REPLICAS  CPU%   MEM%   STATUS
api               3/3       45.2   62.1   running
ml                1/1       12.0   89.0   running
frontend-dev      1/1       3.1    18.5   running
worker-3          2/2       62.0   40.0   running
```

---

## 3. `resma services inspect <name>`

**Syntax:**
```bash
resma services inspect <name>
```

**Description (PT-BR):** Exibe estatísticas detalhadas de um serviço específico,
incluindo limites configurados, uso atual, percentis e histórico recente de
CPU/memória.

**API Endpoint:** `GET /api/services/{name}/stats`

**Path params:** `name` — nome do serviço no Swarm.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma services inspect api
```

**Example output:**
```
Service: api
Replicas: 3/3
CPU:  45.2% (limit: 2.0 cores)  p95: 78.0%
MEM:  62.1% (limit: 4Gi)       p95: 71.0%
OOMs (7d): 0
Last updated: 2025-01-15 14:30
```

---

## 4. `resma services metrics <name>`

**Syntax:**
```bash
resma services metrics <name> [--range 7d]
```

**Description (PT-BR):** Retorna séries temporais de métricas (CPU, memória,
rede) de um serviço dentro do intervalo especificado. Útil para análise de
tendências e diagnóstico de gargalos.

**API Endpoint:** `GET /api/services/{name}/metrics`

**Path params:** `name` — nome do serviço.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--range` | duration | `7d` | Intervalo de tempo da consulta (ex: `1d`, `7d`, `30d`) |

**Query params:** `range` ← `--range`

**Example usage:**
```bash
resma services metrics api --range 1d
```

**Example output:**
```
Service: api  (range: 1d)
TIMESTAMP            CPU%   MEM%   NET_RX    NET_TX
2025-01-14 14:00     40.1   58.0   1.2MB/s   0.8MB/s
2025-01-14 15:00     52.3   61.0   1.5MB/s   1.0MB/s
...
2025-01-15 14:00     45.2   62.1   1.1MB/s   0.9MB/s
```

---

## 5. `resma services containers <name>`

**Syntax:**
```bash
resma services containers <name>
```

**Description (PT-BR):** Lista todos os containers (tasks) associados a um
serviço, com node de execução, status e uso de recursos por container.

**API Endpoint:** `GET /api/services/{name}/containers`

**Path params:** `name` — nome do serviço.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma services containers api
```

**Example output:**
```
Service: api
ID            NODE       STATUS    CPU%   MEM%
a1b2c3d4      node-01    running   15.0   20.0
e5f6g7h8      node-02    running   18.0   22.0
i9j0k1l2      node-03    running   12.2   20.1
```

---

## 6. `resma services sparklines`

**Syntax:**
```bash
resma services sparklines [--points 20]
```

**Description (PT-BR):** Gera sparklines (mini-gráficos de texto) de CPU e
memória para todos os serviços, permitindo visualização rápida de tendências em
uma única tela.

**API Endpoint:** `GET /api/services/sparklines`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Max | Description |
|------|------|---------|-----|-------------|
| `--points` | int | `20` | `100` | Número de pontos de dados em cada sparkline |

**Query params:** `points` ← `--points`

**Example usage:**
```bash
resma services sparklines --points 30
```

**Example output:**
```
SERVICE        CPU  ▁▂▃▅▇▆▄▃▂▁▂▃▅▇█▇▅▃▂▁▂▃▅▆▄▃▂▁▂▃
api
SERVICE        MEM  ▃▄▅▆▆▇▇█▇▇▆▅▅▄▄▅▆▇▇█▇▆▅▄▃▄▅▆▇█
api
SERVICE        CPU  ▁▁▁▁▂▂▃▃▄▄▅▅▆▆▇▇█▇▆▅▄▃▂▁▁▂▃▄▅▆
ml
```

---

## 7. `resma services health`

**Syntax:**
```bash
resma services health [--days 7]
```

**Description (PT-BR):** Exibe o health-check de todos os serviços monitorados,
incluindo número de OOMs, restarts e disponibilidade no período especificado.

**API Endpoint:** `GET /api/services/health`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--days` | int | `7` | Janela em dias para o health-check |

**Query params:** `days` ← `--days`

**Example usage:**
```bash
resma services health --days 14
```

**Example output:**
```
SERVICE        OOMs(14d)  RESTARTS  UPTIME%   STATUS
api            0          2         99.8%     healthy
ml             1          0         98.5%     warning
worker-3       0          0         100.0%    healthy
```

---

## 8. `resma containers inspect <id>`

**Syntax:**
```bash
resma containers inspect <id>
```

**Description (PT-BR):** Exibe estatísticas detalhadas de um container
específico, incluindo PID, limites de cgroup, uso atual e node de execução.

**API Endpoint:** `GET /api/services/containers/{id}/stats`

**Path params:** `id` — ID do container.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma containers inspect a1b2c3d4
```

**Example output:**
```
Container: a1b2c3d4
Node:      node-01
Service:   api
CPU:       15.0% (limit: 2.0 cores)
MEM:       20.0% (limit: 4Gi / 820MB used)
PID:       12345
Status:    running (up 3d 2h)
```

---

## 9. `resma containers metrics <id>`

**Syntax:**
```bash
resma containers metrics <id>
```

**Description (PT-BR):** Retorna métricas históricas de CPU, memória e rede de
um container específico ao longo do tempo.

**API Endpoint:** `GET /api/services/containers/{id}/metrics`

**Path params:** `id` — ID do container.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma containers metrics a1b2c3d4
```

**Example output:**
```
Container: a1b2c3d4 (api @ node-01)
TIMESTAMP            CPU%   MEM%   NET_RX
2025-01-15 13:00     14.0   19.0   1.0MB/s
2025-01-15 14:00     15.0   20.0   1.1MB/s
```

---

## 10. `resma containers network <id>`

**Syntax:**
```bash
resma containers network <id>
```

**Description (PT-BR):** Exibe informações de rede de um container, incluindo
IPs, portas mapeadas, redes conectadas e estatísticas de throughput.

**API Endpoint:** `GET /api/services/containers/{id}/network-info`

**Path params:** `id` — ID do container.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma containers network a1b2c3d4
```

**Example output:**
```
Container: a1b2c3d4
Networks:  resma_overlay (10.0.1.5)
           ingress       (10.0.0.12)
Ports:     8080:8080/tcp
RX:        1.1MB/s   TX: 0.9MB/s
```

---

## 11. `resma nodes list`

**Syntax:**
```bash
resma nodes list
```

**Description (PT-BR):** Lista todos os nodes do Swarm com role (manager/
worker), disponibilidade, CPU/memória total e número de serviços rodando.

**API Endpoint:** `GET /api/nodes`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma nodes list
```

**Example output:**
```
NODE-ID    ROLE     AVAIL    CPU    MEM      SERVICES
node-01    manager  active   8.0    32Gi     4
node-02    worker   active   8.0    32Gi     3
node-03    worker   active   4.0    16Gi     2
```

---

## 12. `resma nodes cluster`

**Syntax:**
```bash
resma nodes cluster
```

**Description (PT-BR):** Exibe uma visão consolidada do cluster de nodes,
incluindo capacidade total vs. alocada, distribuição de serviços e nodes em
estado degradado.

**API Endpoint:** `GET /api/nodes/cluster`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma nodes cluster
```

**Example output:**
```
Cluster Overview — 5 nodes
────────────────────────────
Total CPU:  32.0 cores  Allocated: 18.5 (57%)
Total MEM:  128Gi       Allocated: 72Gi  (56%)
Managers:   3   Workers: 2
Degraded:   0
```

---

## 13. `resma nodes inspect <node-id>`

**Syntax:**
```bash
resma nodes inspect <node-id>
```

**Description (PT-BR):** Exibe detalhes de um node específico, incluindo
hardware, labels, estado do Docker Engine e agentes associados.

**API Endpoint:** `GET /api/nodes/{node_id}`

**Path params:** `node_id` — ID do node no Swarm.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma nodes inspect node-01
```

**Example output:**
```
Node: node-01
Role:        manager
Availability: active
CPU:         8.0 cores   MEM: 32Gi
Engine:      27.3.1
Labels:      zone=us-east, disk=ssd
Agent:       online (last seen 30s ago)
Services:    4
```

---

## 14. `resma nodes metrics <node-id>`

**Syntax:**
```bash
resma nodes metrics <node-id>
```

**Description (PT-BR):** Retorna métricas históricas de CPU, memória, disco e
rede de um node específico.

**API Endpoint:** `GET /api/nodes/{node_id}/metrics`

**Path params:** `node_id` — ID do node.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma nodes metrics node-01
```

**Example output:**
```
Node: node-01
TIMESTAMP            CPU%   MEM%   DISK%   NET
2025-01-15 13:00     55.0   60.0   45.0    2.1MB/s
2025-01-15 14:00     58.0   62.0   46.0    2.3MB/s
```

---

## 15. `resma nodes services <node-id>`

**Syntax:**
```bash
resma nodes services <node-id>
```

**Description (PT-BR):** Lista todos os serviços (e suas tasks) rodando em um
node específico, com uso de recursos por serviço.

**API Endpoint:** `GET /api/nodes/{node_id}/services`

**Path params:** `node_id` — ID do node.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma nodes services node-01
```

**Example output:**
```
Node: node-01
SERVICE        TASKS  CPU%   MEM%
api            1      15.0   20.0
ml             1      12.0   89.0
frontend-dev   1      3.0    18.0
```

---

## 16. `resma agents list`

**Syntax:**
```bash
resma agents list
```

**Description (PT-BR):** Lista todos os RESMA Agents ativos no cluster, com
node associado, versão do agent, último heartbeat e status de conexão.

**API Endpoint:** `GET /api/agents`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma agents list
```

**Example output:**
```
NODE-ID    VERSION   LAST SEEN   STATUS
node-01    1.2.0     30s ago     online
node-02    1.2.0     25s ago     online
node-03    1.2.0     2m ago      online
node-04    1.1.5     5m ago      degraded
```

---

## 17. `resma agents inspect <node-id>`

**Syntax:**
```bash
resma agents inspect <node-id>
```

**Description (PT-BR):** Exibe detalhes de um RESMA Agent específico, incluindo
versão, configuração de coleta, buffer de métricas pendentes e histórico de
heartbeats.

**API Endpoint:** `GET /api/agents/{node_id}`

**Path params:** `node_id` — ID do node onde o agent roda.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma agents inspect node-01
```

**Example output:**
```
Agent: node-01
Version:       1.2.0
Status:        online
Last heartbeat: 30s ago
Buffer:        0 pending metrics
Collect interval: 15s
Push interval:   30s
```

---

## 18. `resma tasks list`

**Syntax:**
```bash
resma tasks list
```

**Description (PT-BR):** Lista todas as tasks do Swarm monitoradas pelo RESMA,
com serviço, node, estado e timestamp da última atualização.

**API Endpoint:** `GET /api/tasks`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma tasks list
```

**Example output:**
```
ID            SERVICE        NODE       STATE     UPDATED
a1b2c3d4      api            node-01    running   2025-01-15 14:00
e5f6g7h8      api            node-02    running   2025-01-15 14:00
i9j0k1l2      ml             node-01    running   2025-01-15 13:55
```

---

## 19. `resma tasks show <service>`

**Syntax:**
```bash
resma tasks show <service>
```

**Description (PT-BR):** Exibe todas as tasks de um serviço específico, com
detalhes de estado, node, erro (se houver) e histórico de restarts.

**API Endpoint:** `GET /api/tasks/{service}`

**Path params:** `service` — nome do serviço.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma tasks show api
```

**Example output:**
```
Service: api (3 tasks)
ID            NODE       STATE     RESTARTS   ERROR
a1b2c3d4      node-01    running   0          -
e5f6g7h8      node-02    running   1          -
i9j0k1l2      node-03    running   0          -
```

---

## 20. `resma tasks history <service>`

**Syntax:**
```bash
resma tasks history <service> [--days 7]
```

**Description (PT-BR):** Exibe o histórico de tasks de um serviço ao longo do
período especificado, incluindo eventos de criação, remoção, falha e restart.

**API Endpoint:** `GET /api/tasks/{service}/history`

**Path params:** `service` — nome do serviço.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--days` | int | `7` | Janela em dias do histórico |

**Query params:** `days` ← `--days`

**Example usage:**
```bash
resma tasks history api --days 14
```

**Example output:**
```
Service: api (history: 14d)
TIMESTAMP            EVENT      NODE       TASK
2025-01-10 08:00     created    node-01    a1b2c3d4
2025-01-12 14:30     failed     node-02    e5f6g7h8
2025-01-12 14:31     restarted  node-02    e5f6g7h8
2025-01-14 09:00     created    node-03    i9j0k1l2
```

---

## 21. `resma recommendations list`

**Syntax:**
```bash
resma recommendations list
```

**Description (PT-BR):** Lista todas as recomendações de limites de recursos
geradas pelo ML sidecar, com nível de confiança, serviço e tipo de ajuste
(aumento/redução).

**API Endpoint:** `GET /api/recommendations`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma recommendations list
```

**Example output:**
```
SERVICE     TYPE         CPU_SUGG    MEM_SUGG    CONFIDENCE
api         reduce       1.5 cores   3Gi         92%
ml          increase     2.0 cores   12Gi        85%
worker-3    maintain     2.0 cores   4Gi         78%
```

---

## 22. `resma recommendations show <service>`

**Syntax:**
```bash
resma recommendations show <service>
```

**Description (PT-BR):** Exibe a recomendação detalhada para um serviço
específico, incluindo justificativa estatística, percentis usados e histórico
de recomendações anteriores.

**API Endpoint:** `GET /api/recommendations/{service}`

**Path params:** `service` — nome do serviço.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma recommendations show api
```

**Example output:**
```
Service: api
Recommendation: reduce
  CPU: 2.0 → 1.5 cores  (p95: 78%, p99: 85%)
  MEM: 4Gi → 3Gi        (p95: 71%, p99: 75%)
Confidence: 92%
Reason: p95 below 80% of limit for 7+ days
Generated: 2025-01-15 12:00
```

---

## 23. `resma recommendations triggers`

**Syntax:**
```bash
resma recommendations triggers
```

**Description (PT-BR):** Lista os gatilhos que disparam recomendações (ex:
OOMs repetidos, uso sustentado acima do limite, memory leak detectado), com
serviços afetados e severidade.

**API Endpoint:** `GET /api/recommendations/triggers`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma recommendations triggers
```

**Example output:**
```
TRIGGER              SERVICE    SEVERITY    DETECTED
oom_repeat            ml         high        2025-01-15 10:00
mem_above_limit       worker-3   medium      2025-01-14 22:00
mem_leak_suspected    ml         high        2025-01-15 11:00
```

---

## 24. `resma recommendations storage`

**Syntax:**
```bash
resma recommendations storage
```

**Description (PT-BR):** Exibe recomendações de storage, incluindo volumes com
crescimento anômalo, sugestões de cleanup e projeção de saturação.

**API Endpoint:** `GET /api/recommendations/storage`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma recommendations storage
```

**Example output:**
```
VOLUME            GROWTH (7d)   PROJECTED FULL    ACTION
pg_data           +12Gi         2025-02-10        cleanup suggested
redis_data        +0.5Gi        -                 maintain
ml_cache           +3Gi         2025-03-01        review retention
```

---

## 25. `resma recommendations simulate`

**Syntax:**
```bash
resma recommendations simulate [--tier conservative|balanced|aggressive] [--services <csv>]
```

**Description (PT-BR):** Simula a aplicação de recomendações em diferentes níveis
de agressividade (tiers) para um conjunto de serviços, sem aplicar mudanças.
Retorna os limites sugeridos e o impacto estimado. Embora use `POST` (por
enviar um payload de simulação), é read-only — nenhuma alteração de estado é
persistida.

**API Endpoint:** `POST /api/recommendations/simulate`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only (simulação)

**Flags:**

| Flag | Type | Default | Values | Description |
|------|------|---------|--------|-------------|
| `--tier` | string | `balanced` | `conservative`, `balanced`, `aggressive` | Nível de agressividade da simulação |
| `--services` | CSV | all | — | Lista de serviços a simular (ex: `api,ml`) |

**Request body:** `{ "tier": "<tier>", "services": ["<svc>", ...] }`

**Example usage:**
```bash
resma recommendations simulate --tier aggressive --services api,ml
```

**Example output:**
```
Simulation: aggressive tier
SERVICE     CPU_NOW → CPU_SUGG   MEM_NOW → MEM_SUGG   EST. SAVINGS
api         2.0 → 1.0 cores      4Gi → 2Gi            1.0 core, 2Gi
ml          2.0 → 3.0 cores      8Gi → 12Gi           -1.0 core, -4Gi
```

---

## 26. `resma rollback-watches list`

**Syntax:**
```bash
resma rollback-watches list [--status <status>] [--service <name>] [--limit 200]
```

**Description (PT-BR):** Lista os monitors de rollback (rollback-watches)
configurados, que observam serviços após uma mudança de limites para detectar
degradação e disparar rollback automático se necessário.

**API Endpoint:** `GET /api/rollback-watches`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Max | Description |
|------|------|---------|-----|-------------|
| `--status` | string | — | — | Filtra por status (`active`, `completed`, `rolled_back`) |
| `--service` | string | — | — | Filtra por nome de serviço |
| `--limit` | int | `200` | `200` | Número máximo de resultados |

**Query params:** `status` ← `--status`, `service` ← `--service`, `limit` ← `--limit`

**Example usage:**
```bash
resma rollback-watches list --status active --service api
```

**Example output:**
```
ID         SERVICE    STATUS    STARTED              DURATION
42         api        active    2025-01-15 14:00     30m
43         ml         active    2025-01-15 13:30     1h
```

---

## 27. `resma rollback-watches inspect <id>`

**Syntax:**
```bash
resma rollback-watches inspect <id>
```

**Description (PT-BR):** Exibe detalhes de um monitor de rollback específico,
incluindo limites anteriores e novos, métricas observadas e decisão (maintain
ou rollback).

**API Endpoint:** `GET /api/rollback-watches/{id}`

**Path params:** `id` — ID do rollback-watch.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma rollback-watches inspect 42
```

**Example output:**
```
Watch #42
Service:    api
Status:     active
Started:    2025-01-15 14:00
Duration:   30m (window: 1h)
Old limits: CPU 2.0  MEM 4Gi
New limits: CPU 1.5  MEM 3Gi
Metrics:    CPU p95 72%  MEM p95 65%  (within threshold)
Decision:   pending (monitoring)
```

---

## 28. `resma schedules list`

**Syntax:**
```bash
resma schedules list [--status <status>]
```

**Description (PT-BR):** Lista todos os agendamentos (schedules) de mudanças de
limites, com status, serviço, próxima execução e tipo de operação.

**API Endpoint:** `GET /api/schedules`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--status` | string | — | Filtra por status (`pending`, `executed`, `cancelled`, `failed`) |

**Query params:** `status` ← `--status`

**Example usage:**
```bash
resma schedules list --status pending
```

**Example output:**
```
ID    SERVICE    STATUS    NEXT RUN             ACTION
10    api        pending   2025-01-16 02:00     apply recommendation #77
11    ml         pending   2025-01-16 02:00     apply recommendation #78
```

---

## 29. `resma schedules pending`

**Syntax:**
```bash
resma schedules pending
```

**Description (PT-BR):** Atalho para listar apenas agendamentos pendentes
(status = `pending`), ordenados por próxima execução.

**API Endpoint:** `GET /api/schedules/pending`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma schedules pending
```

**Example output:**
```
ID    SERVICE    NEXT RUN             ACTION
10    api        2025-01-16 02:00     apply recommendation #77
11    ml         2025-01-16 02:00     apply recommendation #78
```

---

## 30. `resma schedules history`

**Syntax:**
```bash
resma schedules history [--service <name>] [--limit 50]
```

**Description (PT-BR):** Exibe o histórico de agendamentos executados, com
resultado, timestamp e serviço afetado. Permite filtrar por serviço.

**API Endpoint:** `GET /api/schedules/history`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | — | Filtra por nome de serviço |
| `--limit` | int | `50` | Número máximo de resultados |

**Query params:** `service` ← `--service`, `limit` ← `--limit`

**Example usage:**
```bash
resma schedules history --service api --limit 10
```

**Example output:**
```
ID    SERVICE    EXECUTED              RESULT
7     api        2025-01-14 02:00      success
8     api        2025-01-13 02:00      success
9     api        2025-01-12 02:00      rolled_back (OOM detected)
```

---

## 31. `resma templates list`

**Syntax:**
```bash
resma templates list
```

**Description (PT-BR):** Lista todos os templates de recursos disponíveis, com
limites de CPU/memória predefinidos que podem ser aplicados a serviços.

**API Endpoint:** `GET /api/templates`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma templates list
```

**Example output:**
```
NAME          CPU     MEM      DESCRIPTION
small         0.5     512Mi    Light workload
medium        1.0     1Gi      Standard workload
large         2.0     4Gi      Heavy workload
xlarge        4.0     8Gi      CPU-intensive workload
```

---

## 32. `resma templates inspect <name>`

**Syntax:**
```bash
resma templates inspect <name>
```

**Description (PT-BR):** Exibe os detalhes de um template específico, incluindo
limites de CPU, memória, reservas e serviços que o utilizam atualmente.

**API Endpoint:** `GET /api/templates/{name}`

**Path params:** `name` — nome do template.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma templates inspect medium
```

**Example output:**
```
Template: medium
CPU limit:      1.0 core
CPU reserved:   0.5 core
MEM limit:      1Gi
MEM reserved:   256Mi
Services using: api, worker-3
```

---

## 33. `resma storage summary`

**Syntax:**
```bash
resma storage summary
```

**Description (PT-BR):** Exibe um resumo consolidado do uso de storage do
cluster, incluindo capacidade total, usado, disponível e número de volumes.

**API Endpoint:** `GET /api/storage/summary`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma storage summary
```

**Example output:**
```
Storage Summary
────────────────────
Total capacity:  500Gi
Used:            180Gi (36%)
Available:       320Gi
Volumes:         12
Snapshots:       5
```

---

## 34. `resma storage trend`

**Syntax:**
```bash
resma storage trend [--days 7]
```

**Description (PT-BR):** Exibe a tendência de crescimento de storage ao longo
do período especificado, com taxa de crescimento diária e projeção.

**API Endpoint:** `GET /api/storage/trend`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--days` | int | `7` | Janela em dias da tendência |

**Query params:** `days` ← `--days`

**Example usage:**
```bash
resma storage trend --days 30
```

**Example output:**
```
Storage Trend (30d)
────────────────────
Day -30:  150Gi
Day -20:  162Gi
Day -10:  171Gi
Today:    180Gi
Growth:   +30Gi (+20%)  ~1.0Gi/day
Projected full: 2025-06-01
```

---

## 35. `resma storage volumes`

**Syntax:**
```bash
resma storage volumes [--days 7]
```

**Description (PT-BR):** Exibe o crescimento de todos os volumes do cluster ao
longo do período especificado, permitindo identificar volumes com crescimento
anômalo.

**API Endpoint:** `GET /api/storage/volumes/growth`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--days` | int | `7` | Janela em dias do crescimento |

**Query params:** `days` ← `--days`

**Example usage:**
```bash
resma storage volumes --days 14
```

**Example output:**
```
Volume Growth (14d)
VOLUME            SIZE     GROWTH    RATE/DAY
pg_data           45Gi     +12Gi     0.86Gi
redis_data        2Gi      +0.5Gi    0.04Gi
ml_cache          8Gi      +3Gi      0.21Gi
```

---

## 36. `resma storage volume <name>`

**Syntax:**
```bash
resma storage volume <name> [--days 7]
```

**Description (PT-BR):** Exibe o crescimento detalhado de um volume específico
ao longo do período, com série temporal de tamanho e taxa de crescimento.

**API Endpoint:** `GET /api/storage/volumes/{name}/growth`

**Path params:** `name` — nome do volume.

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--days` | int | `7` | Janela em dias do crescimento |

**Query params:** `days` ← `--days`

**Example usage:**
```bash
resma storage volume pg_data --days 30
```

**Example output:**
```
Volume: pg_data (growth: 30d)
TIMESTAMP            SIZE     DELTA
2025-01-01           33Gi     -
2025-01-08           38Gi     +5Gi
2025-01-15           45Gi     +7Gi
Rate: 0.86Gi/day
```

---

## 37. `resma alerts`

**Syntax:**
```bash
resma alerts
```

**Description (PT-BR):** Lista todos os alertas ativos no cluster, incluindo
severidade, serviço afetado, mensagem e timestamp de disparo.

**API Endpoint:** `GET /api/alerts`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:** Nenhuma.

**Query params:** Nenhum.

**Example usage:**
```bash
resma alerts
```

**Example output:**
```
SEVERITY    SERVICE    MESSAGE                         TRIGGERED
high        ml         OOM detected (3 in 1h)          2025-01-15 10:00
medium      worker-3   Memory above limit (92%)        2025-01-15 09:30
low         api        CPU p95 above 80%               2025-01-15 08:15
```

---

## 38. `resma oom-events`

**Syntax:**
```bash
resma oom-events [--service <name>] [--range 7d]
```

**Description (PT-BR):** Lista eventos de OOM (Out-Of-Memory) registrados no
período especificado, com serviço, container, timestamp e impacto. Permite
filtrar por serviço.

**API Endpoint:** `GET /api/oom-events`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | — | Filtra por nome de serviço |
| `--range` | duration | `7d` | Intervalo de tempo da consulta |

**Query params:** `service` ← `--service`, `range` ← `--range`

**Example usage:**
```bash
resma oom-events --service ml --range 1d
```

**Example output:**
```
OOM Events (ml, range: 1d)
TIMESTAMP            CONTAINER    EXIT_CODE   MESSAGE
2025-01-15 10:00     m1n2o3p4     137         OOMKilled
2025-01-15 10:30     m1n2o3p4     137         OOMKilled
2025-01-15 11:00     m1n2o3p4     137         OOMKilled
Total: 3 events
```

---

## 39. `resma change-log`

**Syntax:**
```bash
resma change-log [--service <name>] [--limit 100]
```

**Description (PT-BR):** Exibe o log de mudanças de configuração aplicadas
(limites de CPU/memória, templates, rollbacks), com autor, timestamp e
diferença entre valores anteriores e novos.

**API Endpoint:** `GET /api/change-log`

**Auth:** JWT · **RBAC:** any · **R/W:** read-only

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | — | Filtra por nome de serviço |
| `--limit` | int | `100` | Número máximo de resultados |

**Query params:** `service` ← `--service`, `limit` ← `--limit`

**Example usage:**
```bash
resma change-log --service api --limit 20
```

**Example output:**
```
Change Log (api, limit: 20)
TIMESTAMP            USER         ACTION          CHANGE
2025-01-15 14:00     admin        apply_reco      CPU 2.0→1.5  MEM 4Gi→3Gi
2025-01-14 02:00     scheduler    apply_reco      CPU 2.5→2.0  MEM 5Gi→4Gi
2025-01-12 02:00     admin        rollback        CPU 1.0→2.5  MEM 2Gi→5Gi
```

---

## Notas gerais

- **Autenticação:** Todos os 39 comandos exigem JWT válido. O token é obtido
  via `resma auth login` e armazenado no arquivo de configuração do CLI
  (`~/.config/resma/credentials.json`, XDG-compatible). Nenhum comando de consulta aceita API Key —
  API Keys são exclusivas da API pública versionada (`/api/v1/*`).
- **RBAC:** Todos os comandos de consulta estão disponíveis para qualquer
  usuário autenticado (`any`). Não há restrição por role para leitura.
- **Read-only:** Nenhum comando deste documento modifica estado do cluster ou
  do banco de dados. O comando `recommendations simulate` usa `POST` por
  enviar um payload, mas é semanticamente read-only — a simulação não é
  persistida.
- **Paginação:** Comandos com flag `--limit` suportam paginação no lado do
  servidor. O valor máximo varia por endpoint (ver tabela de flags de cada
  comando).
- **Output:** Todos os comandos suportam flag global `--json` para saída
  estruturada (machine-readable) e `--output <file>` para gravação em arquivo.
