# RESMA CLI — Commands: Auth, Streaming, TUI, Operational, Infra

> **Status:** Referência de comandos de auth, streaming, TUI, operacional e infra — 25 comandos

Este documento é a referência detalhada dos comandos de **autenticação** (7),
**streaming SSE** (1 comando, 10 tópicos), **TUI** (1 comando), **operacional**
(3 comandos) e **infra** (4 comandos) do `resma-cli`. Cada comando é documentado
com sintaxe, descrição, endpoint/ação local, autenticação exigida, exemplo de
uso e exemplo de saída.

> **Convenção de naming:** todos os nomes de comandos, subcomandos, flags e
> parâmetros estão em **American English**. Descrições em prosa estão em
> **Português Brasileiro**.

---

## Sumário

| # | Command | Category | Endpoint/Action | Auth | Description |
|---|---------|----------|-----------------|------|-------------|
| 1 | `resma auth status` | Auth | `GET /api/auth/status` | None | Verifica se o sistema está inicializado (existe usuário admin) |
| 2 | `resma auth login` | Auth | `POST /api/auth/login` | None (public) | Autentica e armazena JWT em `~/.config/resma/credentials.json` |
| 3 | `resma auth logout` | Auth | `POST /api/auth/logout` | JWT | Invalida o refresh token |
| 4 | `resma auth me` | Auth | `GET /api/auth/me` | JWT | Exibe informações do usuário autenticado |
| 5 | `resma auth change-password` | Auth | `POST /api/auth/change-password` | JWT | Troca a senha do usuário autenticado |
| 6 | `resma auth profile` | Auth | `PUT /api/auth/profile` | JWT | Atualiza o perfil (display name) do usuário |
| 7 | `resma auth onboarding` | Auth | `POST /api/auth/onboarding` | None | Cria o primeiro usuário owner (bootstrap) |
| 8 | `resma stream <topic>` | Streaming | `GET /api/sse/{topic}` | Bearer (JWT ou API Key) | Stream SSE inline — Ctrl+C para parar |
| 9 | `resma monitor [--service <name>]` | TUI | SSE (múltiplos tópicos) | Bearer | Dashboard Bubble Tea alt-screen com 6 abas (componentes TUI custom — sem bubbles/list, bubbles/table, bubbles/viewport) |
| 10 | `resma agent status` | Operational | `GET localhost:8082/info` | None | Consulta o HTTP server local do agent |
| 11 | `resma agent health` | Operational | `GET localhost:8082/health` | None | Health check do agent local |
| 12 | `resma test smoke` | Operational | 32 endpoints da API | JWT | Port de `cmd/smoke-test` — 32 testes end-to-end |
| 13 | `resma health` | Infra | `GET /health` | None | Liveness probe da API |
| 14 | `resma ready` | Infra | `GET /ready` | None | Readiness probe da API (deps: DuckDB, Docker) |
| 15 | `resma version` | Infra | Local (sem API call) | None | Exibe versão do CLI, build date e commit |
| 16 | `resma completion <shell>` | Infra | Local | None | Gera script de shell completion |

---

## AUTH commands (7)

Os comandos de autenticação gerenciam o ciclo de vida da sessão do usuário no
`resma-cli`: desde o bootstrap inicial (onboarding), login, consulta de perfil,
troca de senha, até logout. O JWT obtido via `auth login` é armazenado em
`~/.config/resma/credentials.json` e reutilizado automaticamente por todos os comandos
subsequentes que exigem autenticação.

### 1. `resma auth status`

#### Syntax

```bash
resma auth status
```

#### Description

Verifica se o sistema RESMA já foi inicializado — ou seja, se existe pelo menos
um usuário criado no banco. Este é o primeiro comando a ser executado em uma
instalação nova: se retornar `initialized: false`, o próximo passo é rodar
`resma auth onboarding` para criar o primeiro usuário owner.

#### API Endpoint

`GET /api/auth/status`

#### Auth

Nenhuma (endpoint público).

#### Example usage

```bash
resma auth status
```

#### Example output

```json
{
  "initialized": true
}
```

---

### 2. `resma auth login`

#### Syntax

```bash
resma auth login [--username <user>] [--password <pass>] [--url <api-url>]
```

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--username` | string | (prompt interativo) | Nome de usuário |
| `--password` | string | (prompt interativo, mascarado) | Senha |
| `--url` | string | `http://localhost:8080` | URL base da API (sobrescreve `--api-url` global) |

#### Description

Autentica o usuário junto à API e armazena o JWT (access token e refresh token)
no arquivo `~/.config/resma/credentials.json`. Se `--username` ou `--password` não forem
fornecidos via flags, o CLI abre um prompt interativo (senha mascarada). O
`--url` permite apontar para uma API remota sem alterar a config global.

#### API Endpoint

`POST /api/auth/login`

Body: `{ "username": "...", "password": "..." }`

#### Auth

Nenhuma (endpoint público — login é o ato de obter credenciais).

#### Example usage

```bash
resma auth login --username owner --password owner123 --url http://10.0.0.10:8080
```

#### Example output

```
Login successful.
  User:    owner
  Role:    owner
  Token:   eyJhbGciOiJIUzI1NiIs... (stored in ~/.config/resma/credentials.json)
  Expires: 2026-01-15T15:30:00Z
```

---

### 3. `resma auth logout`

#### Syntax

```bash
resma auth logout
```

#### Description

Invalida o refresh token armazenado em `~/.config/resma/credentials.json` junto à API e
remove o arquivo de credenciais local. Após o logout, comandos que exigem JWT
retornarão erro de autenticação até que um novo login seja executado.

#### API Endpoint

`POST /api/auth/logout`

Body: `{ "refresh_token": "..." }` (lido de `~/.config/resma/credentials.json`)

#### Auth

JWT (Bearer token no header `Authorization`).

#### Example usage

```bash
resma auth logout
```

#### Example output

```
Logged out. Credentials removed from ~/.config/resma/credentials.json.
```

---

### 4. `resma auth me`

#### Syntax

```bash
resma auth me
```

#### Description

Exibe as informações do usuário atualmente autenticado: ID, username, role e
display name. Útil para verificar qual sessão está ativa e confirmar que o
token armazenado em `~/.config/resma/credentials.json` ainda é válido.

#### API Endpoint

`GET /api/auth/me`

#### Auth

JWT (Bearer token no header `Authorization`).

#### Example usage

```bash
resma auth me
```

#### Example output

```json
{
  "id": "1",
  "username": "owner",
  "role": "owner",
  "name": "Daniel"
}
```

---

### 5. `resma auth change-password`

#### Syntax

```bash
resma auth change-password --current <current-password> --new <new-password>
```

#### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--current` | string | Senha atual (obrigatório) |
| `--new` | string | Nova senha (obrigatório) |

#### Description

Troca a senha do usuário autenticado. A senha atual é validada no servidor
antes de aceitar a nova. Após a troca, todos os tokens emitidos são
invalidados — o usuário precisa fazer login novamente com a nova senha.

#### API Endpoint

`POST /api/auth/change-password`

Body: `{ "current_password": "...", "new_password": "..." }`

#### Auth

JWT (Bearer token no header `Authorization`).

#### Example usage

```bash
resma auth change-password --current owner123 --new newSecurePass456
```

#### Example output

```
Password changed. Please login again.
```

---

### 6. `resma auth profile`

#### Syntax

```bash
resma auth profile --name <display-name>
```

#### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--name` | string | Novo display name do usuário |

#### Description

Atualiza o perfil do usuário autenticado. Atualmente, apenas o campo `name`
(display name) é editável. O nome é exibido no frontend e na saída de
`resma auth me`.

#### API Endpoint

`PUT /api/auth/profile`

Body: `{ "name": "..." }`

#### Auth

JWT (Bearer token no header `Authorization`).

#### Example usage

```bash
resma auth profile --name "Daniel Souza"
```

#### Example output

```json
{
  "name": "Daniel Souza"
}
```

---

### 7. `resma auth onboarding`

#### Syntax

```bash
resma auth onboarding --username <user> --password <pass>
```

#### Flags

| Flag | Type | Description |
|------|------|-------------|
| `--username` | string | Nome de usuário do primeiro owner (obrigatório) |
| `--password` | string | Senha do primeiro owner (obrigatório) |

#### Description

Cria o primeiro usuário com role `owner` — o bootstrap inicial do sistema.
Só funciona se o sistema ainda não foi inicializado (i.e., `resma auth status`
retorna `initialized: false`). Se já existe um usuário, retorna erro 403.
Após o onboarding, o usuário já pode fazer login com `resma auth login`.

#### API Endpoint

`POST /api/auth/onboarding`

Body: `{ "username": "...", "password": "..." }`

#### Auth

Nenhuma (endpoint público — só funciona antes do primeiro usuário existir).

#### Example usage

```bash
resma auth onboarding --username owner --password owner123
```

#### Example output

```json
{
  "id": 1,
  "username": "owner",
  "role": "owner",
  "message": "Owner account created"
}
```

---

## STREAMING commands (1 command, 10 topics)

O comando `resma stream` abre uma conexão SSE (Server-Sent Events) com a API e
exibe os eventos inline no terminal, em tempo real. A conexão permanece aberta
até o usuário pressionar **Ctrl+C**. O cliente SSE é implementado via stdlib
`net/http` (sem libs externas), com reconexão automática e backoff exponencial.
O cliente usa **callback pattern** (`func(event SSEEvent) error`) — não channels
Go. Cada evento recebido invoca o callback, que processa e exibe o evento.

### 8. `resma stream <topic>`

#### Syntax

```bash
resma stream <topic>
```

#### Arguments

| Argument | Type | Description |
|----------|------|-------------|
| `topic` | string | Tópico SSE para assinar (ver tabela abaixo) |

#### Description

Abre uma conexão SSE com o tópico especificado e imprime cada evento recebido
no terminal. O formato de saída depende do `--output` global: `text` (default,
um evento por linha com timestamp e tipo) ou `json` (cada evento como JSON
completo). Pressione **Ctrl+C** para encerrar o stream.

#### API Endpoint

`GET /api/sse/{topic}`

O header `Accept: text/event-stream` é enviado automaticamente. Para tópicos
dinâmicos (`service-detail/<name>`, `container-detail/<id>`), o path inclui o
parâmetro de path.

#### Auth

Bearer token (JWT ou API Key) no header `Authorization`. O CLI envia o token
armazenado em `~/.config/resma/credentials.json` ou o `--api-key` global.

#### Tópicos SSE disponíveis (10)

| # | Topic | SSE Event Type(s) | Endpoint | Description |
|---|-------|-------------------|----------|-------------|
| 1 | `metrics` | `metrics.sample` | `GET /api/sse/metrics` | Stream de amostras de métricas (CPU/memória) de todos os serviços |
| 2 | `dashboard` | `dashboard.update` | `GET /api/sse/dashboard` | Snapshot completo do dashboard (cluster overview) a cada coleta |
| 3 | `events` | `docker.event` | `GET /api/sse/events` | Eventos do Docker (container start/stop/die, service create/update, etc.) |
| 4 | `services` | `services.changed` | `GET /api/sse/services` | Mudanças em serviços (replicas, status, config de recursos) |
| 5 | `nodes` | `nodes.changed` | `GET /api/sse/nodes` | Mudanças em nós do Swarm (join/leave, availability, recursos) |
| 6 | `tasks` | `task.updated` | `GET /api/sse/tasks` | Mudanças em tasks do Swarm (state transitions: running/complete/failed) |
| 7 | `agents` | `agent.changed` | `GET /api/sse/agents` | Mudanças em agents multi-node (heartbeat, online/offline, container count) |
| 8 | `change-log` | `change-log.entry` | `GET /api/sse/change-log` | Novas entradas no change-log (schedule executado, config aplicada, etc.) |
| 9 | `service-detail/<name>` | `service-detail.update` | `GET /api/sse/service-detail/{name}` | Detalhe completo de um serviço (stats + metrics + containers + tasks + health) |
| 10 | `container-detail/<id>` | `container-detail.update` | `GET /api/sse/container-detail/{id}` | Detalhe completo de um container (stats + metrics + network) |

#### Example usage

```bash
# Stream de métricas de todos os serviços
resma stream metrics

# Stream de mudanças de tasks do Swarm
resma stream tasks

# Stream de detalhe de um serviço específico
resma stream service-detail/api-gateway

# Stream de detalhe de um container específico
resma stream container-detail/abc123def456

# Output JSON (para pipe em jq)
resma stream metrics --output json | jq .
```

#### Example output — `metrics`

```
[2026-01-15 14:32:07] event: metrics.sample
  data: {"service":"api-gateway","cpu":12.4,"mem":34.1,"ts":"2026-01-15T14:32:07Z"}

[2026-01-15 14:32:08] event: metrics.sample
  data: {"service":"data-worker","cpu":78.3,"mem":91.2,"ts":"2026-01-15T14:32:08Z"}
```

#### Example output — `dashboard`

```
[2026-01-15 14:32:07] event: dashboard.update
  data: {"services":12,"nodes":3,"containers":45,"cpu_avg":34.2,"mem_avg":56.7,
         "alerts":2,"healthy":10,"warning":1,"critical":1}
```

#### Example output — `events`

```
[2026-01-15 14:32:10] event: docker.event
  data: {"Type":"container","Action":"die","Actor":{"ID":"abc123","Attributes":
         {"name":"data-worker.1","com.docker.swarm.service.name":"data-worker"}},
         "time":"2026-01-15T14:32:10Z"}

[2026-01-15 14:32:15] event: docker.event
  data: {"Type":"service","Action":"update","Actor":{"ID":"svc456","Attributes":
         {"name":"api-gateway"}},"time":"2026-01-15T14:32:15Z"}
```

#### Example output — `services`

```
[2026-01-15 14:32:12] event: services.changed
  data: {"name":"data-worker","replicas_ready":1,"replicas_total":2,
         "cpu_limit":2.0,"mem_limit":1073741824,"status":"warning"}
```

#### Example output — `nodes`

```
[2026-01-15 14:32:14] event: nodes.changed
  data: {"hostname":"worker-01","role":"worker","availability":"active",
         "status":"ready","cpu_used":4.2,"mem_used":8192}
```

#### Example output — `tasks`

```
[2026-01-15 14:32:16] event: task.updated
  data: {"id":"abc123","service":"data-worker","node":"worker-02",
         "state":"complete","desired":"running","error":""}
```

#### Example output — `agents`

```
[2026-01-15 14:32:18] event: agent.changed
  data: {"node_id":"worker-01","hostname":"worker-01","online":true,
         "containers_count":12,"version":"7.1.0","last_seen":"2026-01-15T14:32:17Z"}
```

#### Example output — `change-log`

```
[2026-01-15 14:32:20] event: change-log.entry
  data: {"id":42,"action":"schedule.executed","service":"data-worker",
         "user":"scheduler","detail":"Applied memory limit: 2Gi","ts":"2026-01-15T14:32:20Z"}
```

#### Example output — `service-detail/<name>`

```
[2026-01-15 14:32:22] event: service-detail.update
  data: {"name":"api-gateway","replicas_ready":3,"replicas_total":3,
         "cpu":12.4,"mem":34.1,"cpu_limit":2.0,"mem_limit":2147483648,
         "containers":[{"id":"abc","cpu":12.1,"mem":33.0,"state":"running"},
                       {"id":"def","cpu":12.8,"mem":35.2,"state":"running"}],
         "tasks":[{"id":"t1","state":"running"},{"id":"t2","state":"running"}],
         "health":"healthy"}
```

#### Example output — `container-detail/<id>`

```
[2026-01-15 14:32:24] event: container-detail.update
  data: {"id":"abc123def456","service":"api-gateway","node":"manager-01",
         "cpu":12.1,"mem_usage":34603008,"mem_limit":2147483648,
         "net_rx":1024,"net_tx":512,"state":"running","uptime":"2h14m"}
```

---

## TUI commands (1)

O comando `resma monitor` abre um dashboard TUI em modo **alt-screen** (tela
cheia) construído com **Bubble Tea** (Charmbracelet). O dashboard consome
múltiplos tópicos SSE simultaneamente e apresenta métricas em tempo real com
gráficos/sparklines inline (asciigraph), navegação totalmente por teclado e
layout responsivo.

> **Nota de implementação TUI:** os componentes TUI são **custom** — não usam
> `bubbles/list`, `bubbles/table` ou `bubbles/viewport`. A implementação usa
> `TableModel` custom, `braille_chart` custom, `FlashField` custom e outros
> componentes próprios sobre o runtime do Bubble Tea, sem depender dos
> componentes pré-construídos da biblioteca bubbles.

> **FlashField:** o componente reutilizável `FlashField` aplica um efeito visual
> de **flash** (destaque temporário) em valores atualizados via SSE — quando um
> campo recebe um novo valor de um evento SSE, ele pisca/muda de cor
> brevemente para indicar a atualização ao usuário. É usado em métricas de CPU,
> MEM, contadores de cluster e qualquer campo que muda em tempo real.

### 9. `resma monitor [--service <name>]`

#### Syntax

```bash
resma monitor [--service <name>]
```

#### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | (none) | Nome do serviço para destacar no detail panel e gráfico ao vivo |

#### Description

Abre o dashboard TUI em alt-screen. O dashboard consome múltiplos tópicos SSE
em paralelo (um por aba ativa) e renderiza métricas, tabelas, listas e gráficos
em tempo real. Ao sair (tecla `q`), o terminal é restaurado ao estado anterior.

Se `--service <name>` for fornecido, a aba **Services** abre com o serviço
especificado selecionado, e o painel de detail + gráfico ao vivo focam nele.

#### API Endpoint

SSE (múltiplos tópicos consumidos em paralelo). O CLI abre uma conexão SSE por
tópico ativo. O cliente SSE usa **callback pattern** (`func(event SSEEvent) error`)
— não channels Go. Cada evento SSE invoca o callback diretamente, e o callback
atualiza o estado do modelo TUI. Antes de conectar SSE, o monitor faz um **fetch
inicial via REST** `GET /api/dashboard` para popular o dashboard com dados
imediatos (evita tela vazia até o primeiro evento SSE chegar).

#### Auth

Bearer token (JWT ou API Key) no header `Authorization` de cada conexão SSE.

#### Tabs (6 abas)

| Tab | Key | Title | Description |
|-----|-----|-------|-------------|
| 1 | `1` | Services | Tabela de serviços com CPU%, MEM%, sparkline e status |
| 2 | `2` | Nodes | Lista de nós do Swarm com recursos e availability |
| 3 | `3` | Agents | Lista de agents multi-node com status online/offline |
| 4 | `4` | Tasks | Lista de tasks do Swarm com state transitions |
| 5 | `5` | Alerts | Feed de alertas (OOM events, recomendações críticas) |
| 6 | `6` | Recommendations | Cards de recomendações de limites (tier + risk + delta) |

#### Keybindings

| Key | Action |
|-----|--------|
| `q` | Sair do TUI (restaura terminal) |
| `1`–`6` | Trocar para a aba N (1=Services, 2=Nodes, 3=Agents, 4=Tasks, 5=Alerts, 6=Recommendations) |
| `Tab` | Trocar para a próxima aba (circular) |
| `s` | Selecionar item destacado (abre detail panel / gráfico ao vivo) |
| `r` | Forçar refresh (reabre conexões SSE) |
| `/` | Focar no campo de busca/filtro |
| `?` | Toggle help overlay (mostra todos os keybindings) |

#### Tab → SSE topics mapping

| Tab | SSE Topics Consumed |
|-----|---------------------|
| 1 — Services | `metrics`, `services`, `service-detail/<selected>`, `dashboard` |
| 2 — Nodes | `nodes`, `dashboard` |
| 3 — Agents | `agents`, `dashboard` |
| 4 — Tasks | `tasks`, `dashboard` |
| 5 — Alerts | `events`, `change-log`, `dashboard` |
| 6 — Recommendations | `services`, `change-log`, `dashboard` |

> **Nota:** O tópico `dashboard` é consumido em todas as abas para manter o
> header (status da conexão, contadores de cluster) sempre atualizado. Tópicos
> dinâmicos (`service-detail/<name>`) só são assinados quando um serviço está
> selecionado na aba Services.

#### Example usage

```bash
# Abrir dashboard TUI
resma monitor

# Abrir com serviço pré-selecionado
resma monitor --service data-worker
```

#### Example output (mockup do alt-screen)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  RESMA Monitor                                          14:32:07  ● Online    │
│  [1] Services  [2] Nodes  [3] Agents  [4] Tasks  [5] Alerts  [6] Recs       │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─ Services ──────────────────────────────────────────────────────────────┐ │
│  │ NAME          REPLICAS   CPU%   MEM%   SPARKLINE        STATUS           │ │
│  │ api-gateway   3/3        12.4   34.1   ▁▂▃▂▁▂▃▄▃▂▁     ● healthy         │ │
│  │ auth-service  2/2         8.1   22.0   ▁▁▂▁▁▂▁▁▂▁▁     ● healthy         │ │
│  │ data-worker   1/2        78.3   91.2   ▆▇█▇▆▇█▇▆▇█▇     ▲ warning         │ │
│  │ billing       0/1         0.0    0.0   ▁▁▁▁▁▁▁▁▁▁▁▁     ✖ critical        │ │
│  └──────────────────────────────────────────────────────────────────────────┘ │
│                                                                              │
│  ┌─ CPU ao vivo — data-worker ──────────────────────────────────────────────┐ │
│  │   100 ┤                            ╭─                                    │ │
│  │    50 ┤                  ╭────╯                                          │ │
│  │     0 ┤───────────╯                                                     │ │
│  └──────────────────────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────────────────────┤
│  q sair │ s selecionar │ r refresh │ Tab próximo │ / buscar │ ? help         │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## OPERATIONAL commands (3)

Os comandos operacionais não seguem o fluxo padrão de API REST. Eles consultam
serviços locais (agent HTTP server em `:8082`) ou executam baterias de testes
contra a API. São usados para diagnóstico e validação.

### 10. `resma agent status`

#### Syntax

```bash
resma agent status
```

#### Description

Consulta o HTTP server local do RESMA Agent (porta `:8082`) e exibe informações
de runtime: versão do agent, número de containers em execução, tamanho do
buffer de métricas pendentes e contagem de OOMs aguardando envio. Este comando
é útil para diagnosticar se o agent está coletando e bufferizando corretamente.

#### API Endpoint / Local action

`GET http://localhost:8082/info`

O agent expõe um HTTP server minimalista na porta `8082` (configurável via
`RESMA_AGENT_HTTP_ADDR`) com dois endpoints: `/health` e `/info`.

#### Auth

Nenhuma (servidor local sem auth — acessível apenas via localhost).

#### Example usage

```bash
resma agent status
```

#### Example output

```json
{
  "version": "7.1.0",
  "containers": 12,
  "buffer_len": 0,
  "oom_pending": 0
}
```

---

### 11. `resma agent health`

#### Syntax

```bash
resma agent health
```

#### Description

Executa um health check no HTTP server local do RESMA Agent (porta `:8082`).
Retorna `ok` se o agent está respondendo. Usado por scripts de monitoramento
e Docker healthchecks.

#### API Endpoint / Local action

`GET http://localhost:8082/health`

#### Auth

Nenhuma (servidor local sem auth — acessível apenas via localhost).

#### Example usage

```bash
resma agent health
```

#### Example output

```
ok
```

---

### 12. `resma test smoke`

#### Syntax

```bash
resma test smoke
```

#### Description

Executa uma bateria de **32 testes end-to-end** contra a API RESMA, validando
todos os endpoints principais: infra, auth, endpoints internos com JWT,
endpoints públicos com API key, ingestão de agent (com e sem token), SSE
session e casos negativos (401 sem auth). É o port do comando `cmd/smoke-test`
do Go API para o CLI, permitindo rodar os mesmos testes sem precisar de `go
run` dentro do container.

**Pré-requisitos:**
- API Go rodando e acessível (default `http://localhost:8080`)
- Onboarding done (usuário owner criado)
- JWT válido (via `resma auth login` ou `--token`)
- `RESMA_AGENT_TOKEN` configurado (default: `dev-agent-token-change-me`)

#### API Endpoint / Local action

Executa 32 requisições HTTP contra a API (ver categorias abaixo).

#### Auth

JWT necessário (lido de `~/.config/resma/credentials.json` ou `--token`). O agent token é
lido da env var `RESMA_AGENT_TOKEN`.

#### 32 test categories

| # | Test | Method | Endpoint | Expected | Auth |
|---|------|--------|----------|----------|------|
| 1 | Infra — liveness | `GET` | `/health` | 200 | None |
| 2 | Infra — readiness | `GET` | `/ready` | 200 | None |
| 3 | Auth — login | `POST` | `/api/auth/login` | 200 + token | None |
| 4 | Auth — me | `GET` | `/api/auth/me` | 200 | JWT |
| 5 | Auth — status | `GET` | `/api/auth/status` | 200 | None |
| 6 | Config | `GET` | `/api/config` | 200 | JWT |
| 7 | Services list | `GET` | `/api/services` | 200 | JWT |
| 8 | Services sparklines | `GET` | `/api/services/sparklines` | 200 | JWT |
| 9 | Dashboard | `GET` | `/api/dashboard` | 200 | JWT |
| 10 | Nodes list | `GET` | `/api/nodes` | 200 | JWT |
| 11 | Nodes cluster | `GET` | `/api/nodes/cluster` | 200 | JWT |
| 12 | Change log | `GET` | `/api/change-log` | 200 | JWT |
| 13 | OOM events | `GET` | `/api/oom-events` | 200 | JWT |
| 14 | Recommendations | `GET` | `/api/recommendations` | 200 | JWT |
| 15 | Recommendations triggers | `GET` | `/api/recommendations/triggers` | 200 | JWT |
| 16 | Recommendations storage | `GET` | `/api/recommendations/storage` | 200 | JWT |
| 17 | Schedules list | `GET` | `/api/schedules` | 200 | JWT |
| 18 | Schedules pending | `GET` | `/api/schedules/pending` | 200 | JWT |
| 19 | Schedules history | `GET` | `/api/schedules/history` | 200 | JWT |
| 20 | Templates | `GET` | `/api/templates` | 200 | JWT |
| 21 | Storage summary | `GET` | `/api/storage/summary` | 200 | JWT |
| 22 | Storage trend | `GET` | `/api/storage/trend` | 200 | JWT |
| 23 | Storage volumes growth | `GET` | `/api/storage/volumes/growth` | 200 | JWT |
| 24 | API keys list | `GET` | `/api/auth/api-keys` | 200 | JWT |
| 25 | Agents list | `GET` | `/api/agents` | 200 | JWT |
| 26 | Tasks list | `GET` | `/api/tasks` | 200 | JWT |
| 27 | Services health | `GET` | `/api/services/health` | 200 | JWT |
| 28 | Agent heartbeat | `POST` | `/api/agent/heartbeat` | 200 | Agent token |
| 29 | Agent ingest metrics | `POST` | `/api/agent/ingest/metrics` | 200 | Agent token |
| 30 | Agent heartbeat (no token) | `POST` | `/api/agent/heartbeat` | 401 | None (negative) |
| 31 | Config (no JWT) | `GET` | `/api/config` | 401 | None (negative) |
| 32 | SSE session create | `POST` | `/api/sse/session` | 200 | JWT |

#### Example usage

```bash
resma test smoke
```

#### Example output

```
=== RESMA API Smoke Test ===
================================================================================
[PASS] GET /health                                     status=200, body_len=15
[PASS] GET /ready                                      status=200, body_len=45
[PASS] POST /api/auth/login                            status=200, token_len=187
[PASS] GET /api/auth/me (JWT)                          status=200, body_len=52
[PASS] GET /api/auth/status (no auth)                  status=200, body_len=21
[PASS] GET /api/config                                 status=200, body_len=128
[PASS] GET /api/services                               status=200, body_len=2048
[PASS] GET /api/services/sparklines                    status=200, body_len=1024
[PASS] GET /api/dashboard                              status=200, body_len=512
[PASS] GET /api/nodes                                  status=200, body_len=384
[PASS] GET /api/nodes/cluster                          status=200, body_len=256
[PASS] GET /api/change-log                             status=200, body_len=128
[PASS] GET /api/oom-events                             status=200, body_len=64
[PASS] GET /api/recommendations                        status=200, body_len=512
[PASS] GET /api/recommendations/triggers               status=200, body_len=256
[PASS] GET /api/recommendations/storage                status=200, body_len=128
[PASS] GET /api/schedules                              status=200, body_len=64
[PASS] GET /api/schedules/pending                      status=200, body_len=32
[PASS] GET /api/schedules/history                      status=200, body_len=128
[PASS] GET /api/templates                              status=200, body_len=256
[PASS] GET /api/storage/summary                        status=200, body_len=64
[PASS] GET /api/storage/trend                          status=200, body_len=128
[PASS] GET /api/storage/volumes/growth                 status=200, body_len=256
[PASS] GET /api/auth/api-keys                          status=200, body_len=32
[PASS] GET /api/agents                                 status=200, body_len=128
[PASS] GET /api/tasks                                  status=200, body_len=64
[PASS] GET /api/services/health                        status=200, body_len=256
[PASS] POST /api/agent/heartbeat (agent token)         status=200
[PASS] POST /api/agent/ingest/metrics (agent token)    status=200
[PASS] POST /api/agent/heartbeat (no token — should 401) status=401
[PASS] GET /api/config (no JWT — should 401)           status=401
[PASS] POST /api/sse/session (create cookie)           status=200
================================================================================

All 32 tests PASSED
```

---

## INFRA commands (4)

Os comandos de infraestrutura não exigem autenticação e são usados para probes
de liveness/readiness, verificação de versão e geração de shell completions.

### 13. `resma health`

#### Syntax

```bash
resma health
```

#### Description

Verifica se o processo da API está vivo (liveness probe). Retorna `ok` se o
servidor responde. Não verifica dependências — usado por load balancers e
Docker healthchecks para saber se o processo está up.

#### API Endpoint

`GET /health`

#### Auth

Nenhuma.

#### Example usage

```bash
resma health
```

#### Example output

```json
{
  "status": "ok"
}
```

---

### 14. `resma ready`

#### Syntax

```bash
resma ready
```

#### Description

Verifica se a API está pronta para receber tráfego (readiness probe). Diferente
do `health`, este endpoint valida dependências: DuckDB (banco de dados) e Docker
(client SDK). Retorna `ok` se todas as deps estão saudáveis, ou `degraded` com
detalhes se alguma falhou. Usado por Kubernetes/Swarm readiness probes.

#### API Endpoint

`GET /ready`

#### Auth

Nenhuma.

#### Example usage

```bash
resma ready
```

#### Example output (healthy)

```json
{
  "status": "ok",
  "deps": {
    "db": "ok",
    "docker": "ok"
  }
}
```

#### Example output (degraded)

```json
{
  "status": "degraded",
  "deps": {
    "db": "ok",
    "docker": "Cannot connect to the Docker daemon at unix:///var/run/docker.sock"
  }
}
```

---

### 15. `resma version`

#### Syntax

```bash
resma version
```

#### Description

Exibe a versão do `resma-cli` binary, incluindo build date e commit hash. Estes
valores são injetados via ldflags no momento do build. Não faz nenhuma chamada
de API — é puramente local.

#### API Endpoint / Local action

Local (sem API call). Build info injetado via `-ldflags`:

```
-X main.version=1.0.0
-X main.buildDate=2026-01-15T14:00:00Z
-X main.commit=abc123def
```

#### Auth

Nenhuma.

#### Example usage

```bash
resma version
```

#### Example output

```
resma-cli v1.0.0
  commit:     abc123def
  build date: 2026-01-15T14:00:00Z
  go version: go1.26.0
  platform:   linux/amd64
```

---

### 16. `resma completion <shell>`

#### Syntax

```bash
resma completion <shell>
```

#### Arguments

| Argument | Type | Description |
|----------|------|-------------|
| `shell` | string | Shell alvo: `bash`, `zsh`, `fish`, `powershell` |

#### Description

Gera o script de shell completion para o shell especificado. O script gerado
deve ser sourced (ou instalado no diretório de completions do shell) para
habilitar autocompletar de comandos, subcomandos e flags do `resma-cli`.

A geração de completions é nativa do **Cobra** (framework CLI usado pelo
`resma-cli`), suportando bash, zsh, fish e PowerShell.

#### API Endpoint / Local action

Local (geração de script via Cobra — sem API call).

#### Auth

Nenhuma.

#### Example usage

```bash
# Bash — instalar permanentemente
resma completion bash > /etc/bash_completion.d/resma

# Bash — sourced temporariamente
source <(resma completion bash)

# Zsh — instalar permanentemente
resma completion zsh > "${fpath[1]}/_resma"

# Fish — instalar permanentemente
resma completion fish > ~/.config/fish/completions/resma.fish

# PowerShell — sourced na sessão atual
resma completion powershell | Out-String | Invoke-Expression
```

#### Example output (bash, excerpt)

```bash
# bash completion for resma                              -*- shell-script -*-

__resma_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE:-} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

# ... (script de completion completo gerado pelo Cobra)
```

---

## Referências

- [Spec técnica do CLI](../spec.md) — arquitetura, stack, estrutura de diretórios
- [Padrões de SSE em Go](../sse-patterns.md) — parser SSE, integração Bubble Tea
- [Design do dashboard TUI](../tui-design.md) — mockup, componentes, modelo
- [API contract](../api-contract.md) — contratos REST da API Go
- [Código fonte do smoke-test](../../../../app/api/cmd/smoke-test/main.go) — implementação de referência dos 32 testes
- [Handlers SSE](../../../../app/api/internal/sse/handlers.go) — endpoints SSE e tópicos
- [Auth handlers](../../../../app/api/internal/server/auth_handlers.go) — endpoints de auth
- [Agent info server](../../../../app/agent/cmd/agent/main.go) — HTTP server local do agent (:8082)
