# RESMA CLI — Contrato de API

> **Status:** Documento de referência — endpoints da API Go consumidos pelo CLI
> **Fonte:** Código-fonte em `app/api/internal/server/` (rotas extraídas dos handlers Go)
> **API Go:** `http://<host>:8080` (dev e produção)

---

## 1. Visão Geral de Autenticação

A API Go tem 5 grupos de rotas com diferentes esquemas de auth:

| Grupo | Prefixo | Auth | Middleware | CLI usa? |
|-------|---------|------|------------|----------|
| **Infra** | `/health`, `/ready` | Nenhuma | — | ✅ Health check |
| **Auth público** | `/api/auth/status`, `/api/auth/onboarding`, `/api/auth/login`, `/api/auth/refresh` | Nenhuma | CORS + Logging + Recovery | ✅ Login (futuro) |
| **SSE** | `/api/sse/*` | Cookie `sse_session` OU `Authorization: Bearer` | Auth própria SSE | ✅ Streaming |
| **Interno/UI** | `/api/*` (exceto v1, sse, agent, internal, auth público) | JWT | JWTMiddleware | ✅ Comandos principais |
| **Público v1** | `/api/v1/*` | API Key + scopes | APIKeyMiddleware | ✅ Automação/scripts |
| **Agent ingest** | `/api/agent/*` | Agent token (constant-time) | agentTokenMiddleware | ❌ (agent only) |
| **Internal ML** | `/api/internal/*` | Nenhuma (Docker network only) | — | ❌ (ML sidecar only) |

### 1.1 Auth no CLI

O `resma-cli` suporta dois modos de auth:

| Modo | Header | Endpoint group | Uso |
|------|--------|----------------|-----|
| **JWT** | `Authorization: Bearer <jwt>` | `/api/*` (interno) | Login interativo (futuro) |
| **API Key** | `Authorization: Bearer <api-key>` | `/api/v1/*` (público) | Automação/scripts (recomendado) |

> **Nota:** O CLI usa `Authorization: Bearer` para SSE (não cookie). O endpoint SSE aceita ambos.

> **Credenciais:** O CLI armazena credenciais em `~/.config/resma/credentials.json` (XDG-compatible).

---

## 2. Endpoints Infra (sem auth)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/health` | Liveness check | `resma health` |
| GET | `/ready` | Readiness check | `resma ready` |

### GET /health
**Response 200:**
```json
{"status": "ok"}
```

### GET /ready
**Response 200:**
```json
{"status": "ready", "db": true, "docker": true}
```

---

## 3. Endpoints Auth

### 3.1 Auth público (sem JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/auth/status` | Verifica se onboarding foi feito | `resma auth status` |
| POST | `/api/auth/onboarding` | Cria primeiro usuário admin | `resma auth onboarding` |
| POST | `/api/auth/login` | Login → JWT | `resma auth login` (futuro) |
| POST | `/api/auth/refresh` | Renova JWT | `resma auth refresh` (futuro) |

### GET /api/auth/status
**Response 200:**
```json
{"initialized": true}
```

### POST /api/auth/login
**Request body:**
```json
{"username": "admin", "password": "secret"}
```
**Response 200:**
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_in": 3600,
  "user": {"id": 1, "username": "admin", "role": "owner"}
}
```

### 3.2 Auth com JWT

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| POST | `/api/auth/logout` | Logout (invalida token) | `resma auth logout` (futuro) |
| GET | `/api/auth/me` | Usuário atual | `resma auth me` |
| POST | `/api/auth/change-password` | Alterar senha | `resma auth change-password` |
| PUT | `/api/auth/profile` | Atualizar perfil | `resma auth profile` |

### 3.3 API Keys CRUD (JWT, owner/admin)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/auth/api-keys` | Lista API keys | `resma api-keys list` |
| POST | `/api/auth/api-keys` | Cria API key | `resma api-keys create` |
| DELETE | `/api/auth/api-keys/{id}` | Revoga API key | `resma api-keys revoke <id>` |
| PATCH | `/api/auth/api-keys/{id}` | Atualiza scopes/nome | `resma api-keys update <id>` |

---

## 4. Endpoints SSE (cookie ou Authorization)

| Método | Path | Tópico | Descrição | CLI command |
|--------|------|--------|-----------|-------------|
| POST | `/api/sse/session` | — | Troca JWT por cookie SSE | ❌ (CLI usa Bearer) |
| DELETE | `/api/sse/session` | — | Invalida cookie SSE | ❌ |
| GET | `/api/sse/metrics` | `metrics` | Stream de métricas | `resma stream metrics` |
| GET | `/api/sse/dashboard` | `dashboard` | Stream de dashboard | `resma stream dashboard` |
| GET | `/api/sse/events` | `events` | Stream de eventos Docker | `resma stream events` |
| GET | `/api/sse/services` | `services` | Stream de mudanças de serviços | `resma stream services` |
| GET | `/api/sse/nodes` | `nodes` | Stream de mudanças de nós | `resma stream nodes` |
| GET | `/api/sse/tasks` | `tasks` | Stream de tasks (Fase 7) | `resma stream tasks` |
| GET | `/api/sse/agents` | `agents` | Stream de agents (Fase 7) | `resma stream agents` |
| GET | `/api/sse/change-log` | `change-log` | Stream de change-log | `resma stream change-log` |
| GET | `/api/sse/service-detail/{name}` | `service-detail/{name}` | Detail de serviço (subscriber-triggered) | `resma stream service-detail <name>` |
| GET | `/api/sse/container-detail/{id}` | `container-detail/{id}` | Detail de container (subscriber-triggered) | `resma stream container-detail <id>` |

> **Nota:** CLI NÃO usa os endpoints `/api/sse/session` (POST/DELETE) — usa `Authorization: Bearer` direto para SSE, sem troca de cookie.

### SSE Auth no CLI

O CLI envia `Authorization: Bearer <token>` no header da conexão SSE. Não usa cookie.

```
GET /api/sse/metrics HTTP/1.1
Host: localhost:8080
Accept: text/event-stream
Cache-Control: no-cache
Authorization: Bearer resma_xxxxxxxxxxxx
```

### SSE Event format

```
event: metrics
data: {"service":"api","cpu":12.4,"mem":45.2,"timestamp":"2026-08-06T14:32:15Z"}

event: metrics
data: {"service":"ml","cpu":2.1,"mem":78.9,"timestamp":"2026-08-06T14:32:15Z"}
```

---

## 5. Endpoints Internos/UI (JWT)

### 5.1 Dashboard, Config, Alerts, OOM, Change-Log

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/dashboard` | Dados agregados do dashboard | `resma dashboard` |
| GET | `/api/config` | Config de recursos do cluster | `resma config` |
| GET | `/api/alerts` | Lista de alertas ativos | `resma alerts` |
| GET | `/api/oom-events` | Eventos de OOM | `resma oom-events` |
| GET | `/api/change-log` | Histórico de mudanças | `resma change-log` |
| GET | `/api/change-log/{service}` | Change-log de um serviço | `resma change-log <service>` |

### 5.2 Services (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/services` | Lista serviços com métricas resumidas | `resma services list` |
| GET | `/api/services/sparklines` | Sparklines de todos os serviços | `resma services sparklines` |
| GET | `/api/services/{name}/metrics` | Métricas temporais de um serviço | `resma metrics cpu/mem <service>` |
| GET | `/api/services/{name}/stats` | Estatísticas (avg, p95, max) | `resma services stats <name>` |
| GET | `/api/services/{name}/containers` | Containers de um serviço | `resma services containers <name>` |
| GET | `/api/services/containers/{container_id}/metrics` | Métricas de um container | `resma metrics container <id>` |
| GET | `/api/services/containers/{container_id}/stats` | Stats de um container | `resma stats container <id>` |
| GET | `/api/services/containers/{container_id}/network-info` | Info de rede do container | `resma net container <id>` |
| PATCH | `/api/services/{name}/archive` | Arquivar serviço (owner/admin) | `resma services archive <name>` |
| PATCH | `/api/services/{name}/restore` | Restaurar serviço (owner/admin) | `resma services restore <name>` |

### 5.3 Nodes (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/nodes` | Lista nós com consumo agregado | `resma nodes list` |
| GET | `/api/nodes/cluster` | Info do cluster Swarm | `resma nodes cluster` |
| GET | `/api/nodes/{node_id}` | Detalhe de um nó | `resma nodes inspect <id>` |
| GET | `/api/nodes/{node_id}/metrics` | Métricas de um nó | `resma metrics node <id>` |
| GET | `/api/nodes/{node_id}/services` | Serviços rodando em um nó | `resma nodes services <id>` |

### 5.4 Recommendations (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/recommendations` | Lista recomendações | `resma recommendations list` |
| GET | `/api/recommendations/triggers` | Triggers de recomendações | `resma recommendations triggers` |
| GET | `/api/recommendations/storage` | Recomendações de storage | `resma recommendations storage` |
| GET | `/api/recommendations/{service}` | Recomendação de um serviço | `resma recommendations <service>` |
| POST | `/api/recommendations/recalculate` | Recalcular todas | `resma recommendations recalculate` |
| POST | `/api/recommendations/{service}/recalculate` | Recalcular um serviço | `resma recommendations recalculate <service>` |
| POST | `/api/recommendations/{service}/apply` | Aplicar recomendação (owner/admin) | `resma recommendations apply <service>` |
| POST | `/api/recommendations/simulate` | Simular batch por tier | `resma recommendations simulate` |

### 5.5 Rollback Watches (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/rollback-watches` | Lista watches | `resma rollback-watches list` |
| GET | `/api/rollback-watches/{id}` | Detalhe de um watch | `resma rollback-watches inspect <id>` |
| POST | `/api/rollback-watches/{id}/rollback` | Rollback manual (owner/admin) | `resma rollback-watches rollback <id>` |
| POST | `/api/rollback-watches/{id}/cancel` | Cancelar watch (owner/admin) | `resma rollback-watches cancel <id>` |

### 5.6 Schedules (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/schedules` | Lista agendamentos | `resma schedules list` |
| GET | `/api/schedules/pending` | Agendamentos pendentes | `resma schedules pending` |
| GET | `/api/schedules/history` | Histórico de execuções | `resma schedules history` |
| POST | `/api/schedules` | Criar agendamento (owner/admin) | `resma schedules create` |
| DELETE | `/api/schedules/{schedule_id}` | Cancelar agendamento (owner/admin) | `resma schedules cancel <id>` |

### 5.7 Templates (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/templates` | Lista templates | `resma templates list` |
| GET | `/api/templates/{name}` | Detalhe de um template | `resma templates inspect <name>` |
| POST | `/api/templates` | Criar template (owner/admin) | `resma templates create` |
| PUT | `/api/templates/{template_id}` | Atualizar template (owner/admin) | `resma templates update <id>` |
| DELETE | `/api/templates/{template_id}` | Deletar template (owner/admin) | `resma templates delete <id>` |
| POST | `/api/templates/{name}/apply/{service}` | Aplicar template (owner/admin) | `resma templates apply <name> <service>` |

### 5.8 Storage (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/storage/summary` | Resumo de storage (system df) | `resma storage summary` |
| GET | `/api/storage/trend` | Tendência de storage | `resma storage trend` |
| GET | `/api/storage/volumes/growth` | Crescimento de volumes | `resma storage volumes` |
| GET | `/api/storage/volumes/{volume_name}/growth` | Crescimento de um volume | `resma storage volume <name>` |

### 5.9 Users (JWT, owner/admin)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/users` | Lista usuários | `resma users list` |
| POST | `/api/users` | Criar usuário | `resma users create` |
| PATCH | `/api/users/{id}` | Atualizar usuário | `resma users update <id>` |
| DELETE | `/api/users/{id}` | Deletar usuário | `resma users delete <id>` |

### 5.10 Settings (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/settings` | Lista settings operacionais | `resma settings list` |
| PUT | `/api/settings` | Atualizar settings (owner/admin) | `resma settings update` |

### 5.11 Prune (JWT, owner/admin)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/prune/preview` | Preview de dados stale/orphan | `resma prune preview` |
| POST | `/api/prune/services-stale` | Prune serviços stale | `resma prune services` |
| POST | `/api/prune/nodes-stale` | Prune nós stale | `resma prune nodes` |
| POST | `/api/prune/tasks-orphan` | Prune tasks órfãs | `resma prune tasks` |
| POST | `/api/prune/metrics` | Prune métricas antigas | `resma prune metrics` |
| POST | `/api/prune/change-log` | Prune change-log | `resma prune change-log` |
| POST | `/api/prune/volume-metrics` | Prune volume metrics | `resma prune volume-metrics` |

### 5.12 Agents Admin (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/agents` | Lista agents com status | `resma agents list` |
| GET | `/api/agents/{node_id}` | Detalhe de um agent | `resma agents inspect <node_id>` |

### 5.13 Tasks (JWT)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/tasks` | Lista tasks do Swarm | `resma tasks list` |
| GET | `/api/tasks/{service}` | Tasks de um serviço | `resma tasks <service>` |
| GET | `/api/tasks/{service}/history` | Histórico de tasks | `resma tasks history <service>` |
| GET | `/api/services/health` | Health de todos os serviços | `resma services health` |

---

## 6. Endpoints Públicos v1 (API Key + scopes)

> **Nota:** As rotas v1 são registradas SEM o prefixo `/v1` — o `http.StripPrefix("/api/v1", ...)` remove-o antes de delegar. O CLI deve usar o path completo `/api/v1/services`, etc.

### 6.1 Services (leitura)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/v1/services` | Lista serviços | `resma services list --api-key` |
| GET | `/api/v1/services/{name}/metrics` | Métricas de um serviço | `resma metrics cpu <name> --api-key` |
| GET | `/api/v1/services/{name}/stats` | Stats de um serviço | `resma services stats <name> --api-key` |
| GET | `/api/v1/services/{name}/containers` | Containers de um serviço | `resma services containers <name> --api-key` |
| GET | `/api/v1/containers/{id}/metrics` | Métricas de um container | `resma metrics container <id> --api-key` |
| GET | `/api/v1/containers/{id}/stats` | Stats de um container | `resma stats container <id> --api-key` |

### 6.2 Nodes (leitura)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/v1/nodes` | Lista nós | `resma nodes list --api-key` |
| GET | `/api/v1/nodes/{id}` | Detalhe de um nó | `resma nodes inspect <id> --api-key` |
| GET | `/api/v1/nodes/{id}/metrics` | Métricas de um nó | `resma metrics node <id> --api-key` |
| GET | `/api/v1/nodes/{id}/services` | Serviços de um nó | `resma nodes services <id> --api-key` |
| GET | `/api/v1/cluster` | Info do cluster | `resma nodes cluster --api-key` |

### 6.3 OOM, Storage, Recommendations, Change-Log (leitura)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/v1/oom-events` | Eventos de OOM | `resma oom-events --api-key` |
| GET | `/api/v1/storage/trend` | Tendência de storage | `resma storage trend --api-key` |
| GET | `/api/v1/storage/volumes/growth` | Crescimento de volumes | `resma storage volumes --api-key` |
| GET | `/api/v1/storage/volumes/{name}/growth` | Crescimento de um volume | `resma storage volume <name> --api-key` |
| GET | `/api/v1/recommendations` | Lista recomendações | `resma recommendations list --api-key` |
| GET | `/api/v1/recommendations/{service}` | Recomendação de um serviço | `resma recommendations <service> --api-key` |
| GET | `/api/v1/recommendations/storage` | Recomendações de storage | `resma recommendations storage --api-key` |
| GET | `/api/v1/change-log` | Histórico de mudanças | `resma change-log --api-key` |

### 6.4 Agents + Tasks + Health (leitura, Fase 7)

| Método | Path | Descrição | CLI command |
|--------|------|-----------|-------------|
| GET | `/api/v1/agents` | Lista agents | `resma agents list --api-key` |
| GET | `/api/v1/agents/{node_id}` | Detalhe de um agent | `resma agents inspect <node_id> --api-key` |
| GET | `/api/v1/tasks` | Lista tasks | `resma tasks list --api-key` |
| GET | `/api/v1/tasks/{service}` | Tasks de um serviço | `resma tasks <service> --api-key` |
| GET | `/api/v1/tasks/{service}/history` | Histórico de tasks | `resma tasks history <service> --api-key` |
| GET | `/api/v1/services/health` | Health de todos os serviços | `resma services health --api-key` |

---

## 7. Endpoints Agent Ingest (agent token — NÃO usado pelo CLI)

> Estes endpoints são para o RESMA Agent (binary Go que roda em cada node). O CLI **não consome** estes endpoints.

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/agent/ingest/metrics` | Push de métricas |
| POST | `/api/agent/ingest/oom` | Push de eventos OOM |
| POST | `/api/agent/heartbeat` | Heartbeat do agent |

---

## 8. Endpoints Internal ML (sem auth — NÃO usado pelo CLI)

> Estes endpoints são para o ML sidecar Python. O CLI **não consome** estes endpoints.

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/internal/services/with-metrics` | Serviços com métricas |
| GET | `/api/internal/services/{service}/metrics` | Série temporal bruta |
| GET | `/api/internal/services/{service}/oom-count` | Contagem de OOMs |
| GET | `/api/internal/services/{service}/config` | Config de recursos |
| GET | `/api/internal/services/{service}/last-apply` | Último apply |
| GET | `/api/internal/storage/volumes/metrics` | Métricas de volume |

---

## 9. Mapeamento CLI → API

### 9.1 MVP (Fase 1)

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma services list` | `GET /api/services` | JWT |
| `resma services list --api-key` | `GET /api/v1/services` | API Key |
| `resma services inspect <name>` | `GET /api/services/{name}/stats` | JWT |
| `resma services health` | `GET /api/services/health` | JWT |
| `resma metrics cpu <service>` | `GET /api/services/{name}/metrics` | JWT |
| `resma metrics mem <service>` | `GET /api/services/{name}/metrics` | JWT |
| `resma stream metrics` | `GET /api/sse/metrics` | Bearer |
| `resma stream agents` | `GET /api/sse/agents` | Bearer |
| `resma stream tasks` | `GET /api/sse/tasks` | Bearer |
| `resma stream services` | `GET /api/sse/services` | Bearer |
| `resma version` | (local — sem API call) | — |
| `resma health` | `GET /health` | — |

### 9.2 Fase 2 (TUI)

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma monitor` | `GET /api/sse/metrics` + `GET /api/sse/agents` + `GET /api/sse/tasks` | Bearer |
| `resma monitor --service <name>` | `GET /api/sse/service-detail/{name}` | Bearer |

### 9.3 Fase 3 (cobertura total)

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma agents list` | `GET /api/agents` | JWT |
| `resma agents inspect <id>` | `GET /api/agents/{node_id}` | JWT |
| `resma tasks list` | `GET /api/tasks` | JWT |
| `resma tasks <service>` | `GET /api/tasks/{service}` | JWT |
| `resma tasks history <service>` | `GET /api/tasks/{service}/history` | JWT |
| `resma recommendations list` | `GET /api/recommendations` | JWT |
| `resma recommendations apply <service>` | `POST /api/recommendations/{service}/apply` | JWT (owner/admin) |
| `resma alerts` | `GET /api/alerts` | JWT |
| `resma nodes list` | `GET /api/nodes` | JWT |
| `resma nodes inspect <id>` | `GET /api/nodes/{node_id}` | JWT |
| `resma storage summary` | `GET /api/storage/summary` | JWT |
| `resma dashboard` | `GET /api/dashboard` | JWT |
| `resma change-log` | `GET /api/change-log` | JWT |
| `resma oom-events` | `GET /api/oom-events` | JWT |

### 9.4 Mapeamento completo (~76 comandos)

> Referência completa de todos os comandos do CLI (spec.md §4.1). Comandos WRITE exigem `--confirm`.

#### Consulta (read-only)

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma dashboard` | `GET /api/dashboard` | JWT |
| `resma services list` | `GET /api/services` | JWT |
| `resma services inspect <name>` | `GET /api/services/{name}/stats` | JWT |
| `resma services metrics <name>` | `GET /api/services/{name}/metrics` | JWT |
| `resma services containers <name>` | `GET /api/services/{name}/containers` | JWT |
| `resma services sparklines` | `GET /api/services/sparklines` | JWT |
| `resma services health` | `GET /api/services/health` | JWT |
| `resma services archive <name>` | `PATCH /api/services/{name}/archive` | JWT (owner/admin) |
| `resma services restore <name>` | `PATCH /api/services/{name}/restore` | JWT (owner/admin) |
| `resma containers inspect <id>` | `GET /api/services/containers/{id}/stats` | JWT |
| `resma containers metrics <id>` | `GET /api/services/containers/{id}/metrics` | JWT |
| `resma containers network <id>` | `GET /api/services/containers/{id}/network-info` | JWT |
| `resma nodes list` | `GET /api/nodes` | JWT |
| `resma nodes cluster` | `GET /api/nodes/cluster` | JWT |
| `resma nodes inspect <node-id>` | `GET /api/nodes/{node_id}` | JWT |
| `resma nodes metrics <node-id>` | `GET /api/nodes/{node_id}/metrics` | JWT |
| `resma nodes services <node-id>` | `GET /api/nodes/{node_id}/services` | JWT |
| `resma agents list` | `GET /api/agents` | JWT |
| `resma agents inspect <node-id>` | `GET /api/agents/{node_id}` | JWT |
| `resma tasks list` | `GET /api/tasks` | JWT |
| `resma tasks show <service>` | `GET /api/tasks/{service}` | JWT |
| `resma tasks history <service>` | `GET /api/tasks/{service}/history` | JWT |
| `resma recommendations list` | `GET /api/recommendations` | JWT |
| `resma recommendations show <service>` | `GET /api/recommendations/{service}` | JWT |
| `resma recommendations triggers` | `GET /api/recommendations/triggers` | JWT |
| `resma recommendations storage` | `GET /api/recommendations/storage` | JWT |
| `resma recommendations recalculate` | `POST /api/recommendations/recalculate` | JWT |
| `resma recommendations recalculate <service>` | `POST /api/recommendations/{service}/recalculate` | JWT |
| `resma recommendations simulate` | `POST /api/recommendations/simulate` | JWT |
| `resma recommendations apply <service>` | `POST /api/recommendations/{service}/apply` | JWT (owner/admin) |
| `resma rollback-watches list` | `GET /api/rollback-watches` | JWT |
| `resma rollback-watches inspect <id>` | `GET /api/rollback-watches/{id}` | JWT |
| `resma rollback-watches rollback <id>` | `POST /api/rollback-watches/{id}/rollback` | JWT (owner/admin) |
| `resma rollback-watches cancel <id>` | `POST /api/rollback-watches/{id}/cancel` | JWT (owner/admin) |
| `resma schedules list` | `GET /api/schedules` | JWT |
| `resma schedules pending` | `GET /api/schedules/pending` | JWT |
| `resma schedules history` | `GET /api/schedules/history` | JWT |
| `resma schedules create` | `POST /api/schedules` | JWT (owner/admin) |
| `resma schedules cancel <id>` | `DELETE /api/schedules/{schedule_id}` | JWT (owner/admin) |
| `resma templates list` | `GET /api/templates` | JWT |
| `resma templates inspect <name>` | `GET /api/templates/{name}` | JWT |
| `resma templates create` | `POST /api/templates` | JWT (owner/admin) |
| `resma templates update <id>` | `PUT /api/templates/{template_id}` | JWT (owner/admin) |
| `resma templates delete <id>` | `DELETE /api/templates/{template_id}` | JWT (owner/admin) |
| `resma templates apply <name> <service>` | `POST /api/templates/{name}/apply/{service}` | JWT (owner/admin) |
| `resma storage summary` | `GET /api/storage/summary` | JWT |
| `resma storage trend` | `GET /api/storage/trend` | JWT |
| `resma storage volumes` | `GET /api/storage/volumes/growth` | JWT |
| `resma storage volume <name>` | `GET /api/storage/volumes/{name}/growth` | JWT |
| `resma alerts` | `GET /api/alerts` | JWT |
| `resma oom-events` | `GET /api/oom-events` | JWT |
| `resma change-log` | `GET /api/change-log` | JWT |
| `resma change-log <service>` | `GET /api/change-log/{service}` | JWT |

#### Admin (write — JWT owner/admin)

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma users list` | `GET /api/users` | JWT (owner/admin) |
| `resma users create` | `POST /api/users` | JWT (owner/admin) |
| `resma users update <id>` | `PATCH /api/users/{id}` | JWT (owner/admin) |
| `resma users delete <id>` | `DELETE /api/users/{id}` | JWT (owner/admin) |
| `resma api-keys list` | `GET /api/auth/api-keys` | JWT (owner/admin) |
| `resma api-keys create` | `POST /api/auth/api-keys` | JWT (owner/admin) |
| `resma api-keys revoke <id>` | `DELETE /api/auth/api-keys/{id}` | JWT (owner/admin) |
| `resma api-keys update <id>` | `PATCH /api/auth/api-keys/{id}` | JWT (owner/admin) |
| `resma settings list` | `GET /api/settings` | JWT |
| `resma settings update` | `PUT /api/settings` | JWT (owner/admin) |
| `resma prune preview` | `GET /api/prune/preview` | JWT (owner/admin) |
| `resma prune services` | `POST /api/prune/services-stale` | JWT (owner/admin) |
| `resma prune nodes` | `POST /api/prune/nodes-stale` | JWT (owner/admin) |
| `resma prune tasks` | `POST /api/prune/tasks-orphan` | JWT (owner/admin) |
| `resma prune metrics` | `POST /api/prune/metrics` | JWT (owner/admin) |
| `resma prune change-log` | `POST /api/prune/change-log` | JWT (owner/admin) |
| `resma prune volume-metrics` | `POST /api/prune/volume-metrics` | JWT (owner/admin) |

#### Auth

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma auth status` | `GET /api/auth/status` | — |
| `resma auth login` | `POST /api/auth/login` | — |
| `resma auth logout` | `POST /api/auth/logout` | JWT |
| `resma auth me` | `GET /api/auth/me` | JWT |
| `resma auth change-password` | `POST /api/auth/change-password` | JWT |
| `resma auth profile` | `PUT /api/auth/profile` | JWT |
| `resma auth onboarding` | `POST /api/auth/onboarding` | — |

#### Streaming (SSE — Bearer)

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma stream metrics` | `GET /api/sse/metrics` | Bearer |
| `resma stream dashboard` | `GET /api/sse/dashboard` | Bearer |
| `resma stream events` | `GET /api/sse/events` | Bearer |
| `resma stream services` | `GET /api/sse/services` | Bearer |
| `resma stream nodes` | `GET /api/sse/nodes` | Bearer |
| `resma stream tasks` | `GET /api/sse/tasks` | Bearer |
| `resma stream agents` | `GET /api/sse/agents` | Bearer |
| `resma stream change-log` | `GET /api/sse/change-log` | Bearer |
| `resma stream service-detail <name>` | `GET /api/sse/service-detail/{name}` | Bearer |
| `resma stream container-detail <id>` | `GET /api/sse/container-detail/{id}` | Bearer |

#### TUI

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma monitor` | `GET /api/sse/metrics` + `/agents` + `/tasks` | Bearer |
| `resma monitor --service <name>` | `GET /api/sse/service-detail/{name}` | Bearer |

#### Operational (local — não passa pela API)

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma agent status` | `GET localhost:8082/info` | — |
| `resma agent health` | `GET localhost:8082/health` | — |
| `resma test smoke` | (32 endpoints da API) | JWT/API Key |

#### Infra

| CLI command | API endpoint | Auth |
|-------------|-------------|------|
| `resma health` | `GET /health` | — |
| `resma ready` | `GET /ready` | — |
| `resma version` | (local — sem API call) | — |
| `resma completion <shell>` | (local — sem API call) | — |

---

## 10. Query Parameters Comuns

| Parâmetro | Endpoints | Descrição | Default |
|-----------|-----------|-----------|---------|
| `window` | metrics, stats | Janela de tempo (ex: `1h`, `24h`, `7d`) | `24h` |
| `limit` | list endpoints | Máximo de resultados | `100` |
| `offset` | list endpoints | Paginação | `0` |
| `status` | tasks, alerts, schedules | Filtrar por status | (todos) |
| `service` | oom-events, change-log | Filtrar por serviço | (todos) |

---

## 11. Response Shapes — Exemplos

### 11.1 GET /api/services

```json
[
  {
    "name": "api",
    "stack": "resma",
    "image": "resma/api:latest",
    "replicas": "3/3",
    "cpu_percent": 12.4,
    "mem_percent": 45.2,
    "mem_usage": "452MB",
    "mem_limit": "1GB",
    "status": "running",
    "nodes": 3,
    "updated_at": "2026-08-06T14:32:15Z"
  }
]
```

### 11.2 GET /api/services/{name}/metrics

```json
{
  "service": "api",
  "window": "1h",
  "points": [
    {"timestamp": "2026-08-06T13:32:15Z", "cpu": 8.2, "mem": 42.1},
    {"timestamp": "2026-08-06T13:47:15Z", "cpu": 12.4, "mem": 45.2},
    ...
  ],
  "stats": {
    "cpu": {"avg": 10.3, "p95": 18.7, "max": 22.1},
    "mem": {"avg": 43.5, "p95": 48.2, "max": 50.1}
  }
}
```

### 11.3 GET /api/agents

```json
[
  {
    "node_id": "node-1",
    "hostname": "swarm-manager",
    "status": "active",
    "last_heartbeat": "2026-08-06T14:32:10Z",
    "version": "0.7.2",
    "containers": 12,
    "buffer_size": 1024
  }
]
```

### 11.4 GET /api/tasks

```json
[
  {
    "id": "task-abc123",
    "service": "api",
    "node": "node-1",
    "status": "running",
    "slot": 1,
    "created_at": "2026-08-06T10:00:00Z",
    "updated_at": "2026-08-06T14:32:15Z"
  }
]
```

### 11.5 GET /api/recommendations

```json
[
  {
    "service": "api",
    "tier": "balanced",
    "current": {"cpu_limit": "1.0", "mem_limit": "1GB"},
    "recommended": {"cpu_limit": "0.5", "mem_limit": "512MB"},
    "savings": {"cpu": "50%", "mem": "50%"},
    "confidence": 0.92,
    "trigger": "low-usage",
    "oom_count": 0
  }
]
```

### 11.6 Error response (todos endpoints)

```json
{
  "error": "service not found",
  "code": 404
}
```

---

## 12. Considerações de Implementação

### 12.1 Dual auth (JWT vs API Key)

O CLI deve suportar ambos os esquemas. A escolha é feita por:

1. Se `--api-key` flag ou `RESMA_API_KEY` env var presente → usa `/api/v1/*` com API Key
2. Senão se `--token` flag ou `RESMA_TOKEN` env var presente → usa `/api/*` com JWT
3. Senão → erro: "authentication required"

### 12.2 SSE no CLI

O CLI usa `Authorization: Bearer` para SSE (não cookie). O endpoint SSE valida ambos:

```go
// Pseudocódigo do handler SSE (app/api/internal/sse/handlers.go)
func (h *Handler) validateSSEAuth(r *http.Request) bool {
    // 1. Tentar cookie sse_session
    if cookie, err := r.Cookie("sse_session"); err == nil {
        // validar session ID
        ...
    }
    // 2. Tentar Authorization: Bearer
    if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
        token := strings.TrimPrefix(authHeader, "Bearer ")
        // validar JWT ou API key
        ...
    }
    return false
}
```

### 12.3 Rate limiting

A API pode retornar `429 Too Many Requests` com header `Retry-After`. O CLI deve respeitar e fazer backoff.

### 12.4 Versioning

- `/api/v1/*` — público, versionado (breaking changes só em v2)
- `/api/*` — interno, não versionado (pode mudar entre releases)
- O CLI deve preferir `/api/v1/*` para automação (estabilidade) e `/api/*` para uso interativo (mais endpoints)

---

## 13. Referências

- **Router setup:** `app/api/internal/server/server.go` — `Handler()` method
- **Service routes:** `app/api/internal/server/service_handlers.go`
- **Node routes:** `app/api/internal/server/node_handlers.go`
- **Task routes:** `app/api/internal/server/task_handlers.go`
- **Agent admin routes:** `app/api/internal/server/task_handlers.go` (registerAgentAdminRoutes)
- **Agent ingest routes:** `app/api/internal/server/agent_handlers.go`
- **Recommendation routes:** `app/api/internal/server/recommendation_handlers.go`
- **SSE routes:** `app/api/internal/sse/handlers.go` — `RegisterRoutes()`
- **Auth routes:** `app/api/internal/server/auth_handlers.go`
- **API key routes:** `app/api/internal/server/apikey_handlers.go`
- **Storage routes:** `app/api/internal/server/storage_handlers.go`
- **Schedule routes:** `app/api/internal/server/schedule_handlers.go`
- **Template routes:** `app/api/internal/server/template_handlers.go`
- **User routes:** `app/api/internal/server/user_handlers.go`
- **Settings routes:** `app/api/internal/server/settings_handlers.go`
- **Prune routes:** `app/api/internal/server/prune_handlers.go`
- **Rollback watch routes:** `app/api/internal/server/rollback_watch_handlers.go`
- **Internal ML routes:** `app/api/internal/server/internal_handlers.go`
- **Simulate routes:** `app/api/internal/server/simulate_handlers.go`
