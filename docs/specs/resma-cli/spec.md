# RESMA CLI — Especificação Técnica

> **Status:** Implementação em andamento — monitor TUI + auth + SSE funcionais · **Domínio:** Infrastructure Intelligence + Development
> **Baseado em:** Benchmark validado por Technical Council (Charmbracelet ecosystem + Cobra + Viper + termenv + asciigraph)
> **Análise completa:** 105 endpoints da API Go + 21 páginas do frontend + agent local mapeados
> **Stack:** Go 1.26 (alinhado com `app/api/` e `app/agent/`)
> **Objetivo:** CLI/TUI híbrido para interagir com o RESMA API — consultar dados, executar ações e exibir métricas em tempo real (SSE) com gráficos no terminal, com máxima compatibilidade cross-shell/cross-terminal.

> **Diretriz de naming:** Todos os comandos, subcomandos, flags, parâmetros e valores de flags são em **inglês americano** (ex: `--color`, `--dry-run`, `--output`, `services list`, `recommendations apply`).

---

## 1. Contexto e Motivação

O RESMA hoje oferece uma API Go (REST + SSE) com **105 endpoints** e um frontend React com **21 páginas**. Operadores de Swarm, no entanto, vivem no terminal — SSH em managers, `docker stack` commands, logs em `tail -f`. Um CLI nativo permite:

- **Consultas rápidas** sem abrir browser (`resma services list`, `resma nodes list`)
- **Monitoramento em tempo real** direto no shell (`resma monitor`, `resma stream metrics`)
- **Automação** em scripts e pipelines CI/CD (`resma agents list --output json | jq ...`)
- **Operações admin** sem UI (`resma prune metrics --confirm`, `resma users create`)
- **Diagnóstico operacional** (`resma agent status`, `resma test smoke`)
- **DX de primeira classe** com shell completions (bash/zsh/fish/PowerShell)

### 1.1 Princípios de design

| Princípio | Descrição |
|-----------|-----------|
| **Híbrido (3 modos)** | Inline (one-shot) + Inline streaming (SSE) + TUI (alt-screen dashboard) |
| **Máxima compatibilidade** | Windows Terminal, PowerShell, cmd (Win10+), bash, zsh, fish, iTerm2, Linux VT |
| **Graceful degradation** | TrueColor → ANSI256 → ANSI16 → ASCII (automático via termenv/colorprofile) |
| **Zero deps externas para SSE** | stdlib `net/http` — sem libs SSE mortas |
| **No hardcode** | URL da API, porta, token via Viper (env vars, config file, flags) |
| **Binary leve** | ~6-8MB (alinhado com o agent Go ~6MB) |
| **Não altera o core** | CLI é consumidor novo de `app/api/` — não modifica API, agent, backend ou frontend |
| **American English naming** | Todos os comandos, flags e parâmetros em inglês americano |
| **Safe writes** | Comandos destrutivos exigem `--confirm`; prune usa `--dry-run` por padrão |
| **Dual auth** | JWT (interativo) ou API Key (automação) — flag `--api-key` seleciona `/api/v1/*` |

### 1.2 Não-metas (fora de escopo)

- Não substitui o frontend React
- Não coleta métricas diretamente (usa a API; o agent continua sendo o collector)
- Não faz deploy de stacks via Docker SDK (isso é `docker stack deploy` / `scripts/deploy-swarm.ps1`)
- Não gerencia usuários do Docker Swarm (isso é `docker node`/`docker service`)

---

## 2. Stack Tecnológica

### 2.1 Dependências (go.mod)

| Dependência | Papel | Versão | Justificativa |
|-------------|-------|--------|---------------|
| `github.com/spf13/cobra` | CLI framework (comandos, flags, completions) | v1.9.1 | Padrão da indústria (kubectl, gh, hugo); completions bash/zsh/fish/PowerShell |
| `github.com/spf13/viper` | Config (env, file, flags, defaults) | v1.21.0 | Integração nativa com Cobra; 12-factor app |
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm Architecture) | v1.3.10 | Model-Update-View; Cmd/Msg + channels para streaming; usado por k9s, dry, lazydocker |
| `github.com/charmbracelet/lipgloss` | Styling/layout declarativo (CSS-like) | v1.1.0 | Color downsampling automático; borders, padding, layout |

> **Nota sobre dependências transitivas:** `lipgloss` já inclui `colorprofile` e `termenv`
> como dependências transitivas — não são declaradas separadamente em `go.mod`. A detecção
> de perfil de cor (TrueColor → ANSI256 → ANSI16 → ASCII) e o VT processing Windows
> (`EnableVirtualTerminalProcessing` para ConHost Win10+) são fornecidos por essas libs
> transitivas via lipgloss.
>
> **Componentes TUI custom (sem bubbles):** A implementação real NÃO usa `bubbles`.
> Todos os componentes TUI (table, sparkline, flash field, tabbar, header, breadcrumbs)
> são construídos custom sobre Bubble Tea + Lipgloss. Isso dá controle total sobre
> layout, renderização e comportamento, sem o overhead de abstrações genéricas.
>
> **Gráficos braille Unicode custom (sem asciigraph):** Gráficos e sparklines usam
> caracteres braille Unicode (ex: `▁▂▃▄▅▆▇█`, `╭─╮╰─╯`) implementados custom em
> `braille_chart.go` e `sparkline.go` — sem dependência de `asciigraph`.

### 2.2 Libs avaliadas e rejeitadas

| Lib | Motivo da rejeição |
|-----|-------------------|
| `gizak/termui` | Maintainer explicitamente ausente; README nota "fluctuating availability" |
| `rivo/tview` | Maintenance score 1/10; maintainer pede colaboradores (issue #1124) |
| `jroimartin/gocui` (original) | Unmaintained desde 2016 |
| `awesome-gocui` (fork) | Viável mas ecossistema menor que Charmbracelet |
| `gosuri/uiprogress` | Maintenance 0/10; 30 issues abertas sem atividade |
| `briandowns/spinner` | Maintenance 0/10; 0 commits em 90 dias |
| `r3labs/sse` | Maintenance 0/10; usar stdlib `net/http` |
| `pterm/pterm` | Boa para output inline, mas overlap com Bubbles; não é TUI full |

### 2.3 Por que Charmbracelet (Bubble Tea + Lipgloss)?

1. **Elm Architecture** — estado previsível e testável; essencial para streaming SSE sem race conditions
2. **Cmd/Msg + channels** — casamento natural entre concurrency Go e event loop TUI
3. **Color downsampling automático** — via `colorprofile.Detect()` + `lipgloss.Complete()`
4. **Windows VT processing** — `termenv.EnableVirtualTerminalProcessing()` resolve ConHost legacy
5. **Provas reais em produção** — k9s (23K stars), dry (Docker monitor), lazydocker (31K stars)
6. **Ecossistema coeso** — Bubble Tea + Lipgloss + Bubbles + termenv da mesma equipe (Charm)

---

## 3. Arquitetura

### 3.1 Estrutura de diretórios

```
app/cli/                              # resma-cli (novo módulo Go)
├── cmd/resma-cli/
│   └── main.go                       # entry point — inicializa Cobra root cmd
├── internal/
│   ├── cli/                          # Cobra commands (um arquivo por grupo)
│   │   ├── root.go                   # root cmd + flags globais
│   │   ├── services.go               # services list/inspect/metrics/containers/sparklines/health/archive/restore
│   │   ├── containers.go             # containers inspect/metrics/network
│   │   ├── nodes.go                  # nodes list/cluster/inspect/metrics/services
│   │   ├── agents.go                 # agents list/inspect
│   │   ├── tasks.go                  # tasks list/show/history
│   │   ├── recommendations.go        # recommendations list/show/triggers/storage/recalculate/simulate/apply
│   │   ├── rollback_watches.go       # rollback-watches list/inspect/rollback/cancel
│   │   ├── schedules.go              # schedules list/pending/history/create/cancel
│   │   ├── templates.go              # templates list/inspect/create/update/delete/apply
│   │   ├── storage.go                # storage summary/trend/volumes/volume
│   │   ├── alerts.go                 # alerts (list only — no ack endpoint)
│   │   ├── oom_events.go             # oom-events [--service]
│   │   ├── change_log.go             # change-log [--service]
│   │   ├── dashboard.go              # dashboard (cluster overview)
│   │   ├── users.go                  # users list/create/update/delete
│   │   ├── api_keys.go               # api-keys list/create/revoke/update
│   │   ├── settings.go               # settings list/update
│   │   ├── prune.go                  # prune preview/services/nodes/tasks/metrics/change-log/volume-metrics
│   │   ├── auth.go                   # auth status/login/logout/me/change-password/profile/onboarding
│   │   ├── agent.go                  # agent status/health (local :8082)
│   │   ├── test.go                   # test smoke (port de cmd/smoke-test)
│   │   ├── monitor.go                # monitor [--service] — Bubble Tea TUI (6 tabs)
│   │   ├── stream.go                 # stream <topic> — inline SSE (10 topics)
│   │   ├── health.go                 # health / ready (API infra)
│   │   ├── version.go                # version
│   │   └── completion.go             # completion bash|zsh|fish|powershell
│   ├── tui/                          # Bubble Tea models (modo TUI) — estrutura flat
│   │   ├── main.go                   # Entry point: Run() — fetch inicial REST + SSE
│   │   ├── model.go                  # Estado raiz (tabs, view modes, sort, flash, SSE)
│   │   ├── layout.go                 # Orquestra layout (header, tabs, content, crumbs)
│   │   ├── header.go                 # Header rico com cluster info, menu grid, logo
│   │   ├── styles.go                 # Paleta RESMA + estilos lipgloss
│   │   ├── keys.go                   # Handler de teclas globais e navegação
│   │   ├── components.go             # Table reutilizável (header, cursor, sort, colunas flex)
│   │   ├── flash_field.go            # Componente flash para valores atualizados via SSE
│   │   ├── flash_prompt.go           # Mensagens flash/toast + prompts command/filter
│   │   ├── braille_chart.go          # Gráficos braille Unicode (sparklines, charts)
│   │   ├── sparkline.go              # Helpers de sparkline e padding
│   │   ├── services_tab.go           # Tab [1] Services
│   │   ├── nodes_tab.go              # Tab [2] Nodes
│   │   ├── agents_tab.go             # Tab [3] Agents
│   │   ├── tasks_tab.go              # Tab [4] Tasks
│   │   ├── alerts_tab.go             # Tab [5] Alerts
│   │   ├── recommendations_tab.go   # Tab [6] Recommendations
│   │   ├── detail_view.go            # Views de detalhe (drill-down)
│   │   ├── logs_view.go              # View de logs inline com filtro e follow
│   │   ├── crumbs.go                 # Breadcrumbs com chips coloridos
│   │   ├── tabbar.go                 # Régua visual de abas
│   │   ├── menu.go                   # Keyhints dinâmicos por view mode e tab
│   │   ├── input_help.go             # Inputs de command/filter + help
│   │   ├── table_helpers.go          # Helpers de renderização e formatação
│   │   ├── mockdata.go               # Dados mock para wireframe
│   │   └── mocklogs.go               # Logs mockados para demonstração
│   ├── client/                       # HTTP client (REST + SSE)
│   │   ├── client.go                 # APIClient base com auth JWT e auto-refresh
│   │   ├── sse.go                    # SSE client reutilizável (callbacks, reconexão)
│   │   ├── auth.go                   # Login, Logout, LoadOrRefresh
│   │   ├── token_store.go            # Persistência em ~/.config/resma/credentials.json (XDG)
│   │   └── cluster.go                # Tipos do payload dashboard SSE
│   ├── config/                       # Viper config
│   │   ├── config.go                 # Load (file + env + flags), defaults, validation
│   │   └── defaults.go               # Defaults: RESMA_API_URL, RESMA_TOKEN, etc.
│   ├── output/                       # Renderização inline (modo não-TUI)
│   │   ├── table.go                  # Tabelas via Lipgloss (inline)
│   │   ├── chart.go                  # Gráficos via asciigraph (inline)
│   │   ├── json.go                   # Output JSON (--output json)
│   │   ├── yaml.go                   # Output YAML (--output yaml)
│   │   └── text.go                   # Output texto plano (--output text, default)
│   ├── confirm/                      # Confirmação interativa para writes
│   │   └── confirm.go                # Prompt sim/não (pula com --confirm/--yes)
│   └── version/                      # Build info (injected via ldflags)
│       └── version.go
├── go.mod
├── go.sum
└── Dockerfile                        # Build multi-stage (golang:1.26-bookworm → distroless)
```

### 3.2 Diagrama de fluxo

```
┌─────────────────────────────────────────────────────────────┐
│                       resma-cli (Go binary)                  │
│                                                              │
│  ┌──────────┐    ┌──────────────────┐    ┌───────────────┐  │
│  │  Cobra   │───→│  Viper Config    │───→│  API Client   │  │
│  │ commands │    │  (env/file/flags)│    │  (REST + SSE) │  │
│  └────┬─────┘    └──────────────────┘    └───────┬───────┘  │
│       │                                          │          │
│       ├──── inline mode ─────────────────────────┤          │
│       │     (Lipgloss + asciigraph)              │          │
│       │     output: table / chart / json / yaml  │          │
│       │                                          │          │
│       ├──── inline streaming ────────────────────┤          │
│       │     (stdlib SSE + Lipgloss)              │          │
│       │     Ctrl+C to stop                       │          │
│       │                                          │          │
│       ├──── TUI mode (`monitor`) ────────────────┤          │
│       │     (Bubble Tea + Bubbles, 6 tabs)       │          │
│       │     Model ←── Cmd/Msg ←── channel ←──────┘          │
│       │        ↓                                             │
│       │     View (Lipgloss render, alt-screen)               │
│       │                                                      │
│       └──── operational mode (local) ──────────────────────  │
│             agent status → localhost:8082                    │
│             test smoke → API endpoints (32 tests)            │
└─────────────────────────────────────────────────────────────┘
                         │
                         ▼  HTTP (REST + SSE)
              ┌─────────────────────┐
              │   RESMA Go API      │
              │   (app/api/)        │
              │   :8080             │
              └─────────────────────┘
```

### 3.3 Modos de operação

| Modo | Comando | Descrição | Stack usada |
|------|---------|-----------|-------------|
| **Inline** | `resma services list` | One-shot, output formatado, retorna ao shell | Cobra + Lipgloss + asciigraph |
| **Inline JSON** | `resma services list --output json` | One-shot, output estruturado para pipes/scripts | Cobra + encoding/json |
| **Inline streaming** | `resma stream metrics` | SSE contínuo, linhas impressas incrementalmente (Ctrl+C para sair) | Cobra + stdlib SSE + Lipgloss |
| **TUI** | `resma monitor` | Alt-screen, dashboard interativo com 6 tabs, navegação por teclado | Cobra + Bubble Tea + Bubbles + Lipgloss |
| **Operational** | `resma agent status` | Consulta local (agent :8082), não passa pela API | Cobra + net/http (localhost) |

---

## 4. Comandos — Árvore Completa

> **Diretriz:** Todos os nomes em inglês americano. Comandos de escrita (WRITE) exigem `--confirm`
> (ou `--yes`/`-y`). Prune usa `--dry-run` por padrão (executa de verdade apenas com `--confirm`).

### 4.1 Estrutura completa (~76 comandos)

```
resma
│
├── ═══════════════════════════════════════════════════
├── CONSULTA (read-only, HTTP API — JWT ou API Key)
├── ═══════════════════════════════════════════════════
│
├── dashboard                              # Cluster overview (GET /api/dashboard)
│
├── services
│   ├── list                               # List all services (GET /api/services)
│   ├── inspect <name>                     # Service detail: stats (GET /api/services/{name}/stats)
│   ├── metrics <name> [--range 7d]        # Time series metrics (GET /api/services/{name}/metrics)
│   ├── containers <name>                  # Service containers (GET /api/services/{name}/containers)
│   ├── sparklines [--points 20]           # All service sparklines (GET /api/services/sparklines)
│   ├── health [--days 7]                  # All services health (GET /api/services/health)
│   ├── archive <name>                     # Archive service [WRITE: owner/admin, --confirm]
│   └── restore <name>                     # Restore service [WRITE: owner/admin, --confirm]
│
├── containers
│   ├── inspect <id>                       # Container stats (GET /api/services/containers/{id}/stats)
│   ├── metrics <id>                       # Container metrics (GET /api/services/containers/{id}/metrics)
│   └── network <id>                       # Network info (GET /api/services/containers/{id}/network-info)
│
├── nodes
│   ├── list                               # List all nodes (GET /api/nodes)
│   ├── cluster                            # Cluster info (GET /api/nodes/cluster)
│   ├── inspect <node-id>                  # Node detail (GET /api/nodes/{node_id})
│   ├── metrics <node-id>                  # Node metrics (GET /api/nodes/{node_id}/metrics)
│   └── services <node-id>                 # Services on node (GET /api/nodes/{node_id}/services)
│
├── agents
│   ├── list                               # List all agents (GET /api/agents)
│   └── inspect <node-id>                  # Agent detail (GET /api/agents/{node_id})
│
├── tasks
│   ├── list                               # List all tasks (GET /api/tasks)
│   ├── show <service>                     # Tasks for service (GET /api/tasks/{service})
│   └── history <service> [--days 7]       # Task history/restarts (GET /api/tasks/{service}/history)
│
├── recommendations
│   ├── list                               # List all (GET /api/recommendations)
│   ├── show <service>                     # Service recommendation (GET /api/recommendations/{service})
│   ├── triggers                           # Triggers (GET /api/recommendations/triggers)
│   ├── storage                            # Storage recommendations (GET /api/recommendations/storage)
│   ├── recalculate [service]              # Recalculate [WRITE: any role]
│   ├── simulate                           # Simulate batch by tier (POST /api/recommendations/simulate)
│   └── apply <service>                    # Apply [WRITE: owner/admin, --confirm]
│
├── rollback-watches
│   ├── list [--status] [--service]        # List watches (GET /api/rollback-watches)
│   ├── inspect <id>                       # Watch detail (GET /api/rollback-watches/{id})
│   ├── rollback <id>                      # Manual rollback [WRITE: owner/admin, --confirm]
│   └── cancel <id>                        # Cancel watch [WRITE: owner/admin, --confirm]
│
├── schedules
│   ├── list [--status]                    # List (GET /api/schedules)
│   ├── pending                            # Pending (GET /api/schedules/pending)
│   ├── history [--service] [--limit 50]   # History (GET /api/schedules/history)
│   ├── create                             # Create [WRITE: owner/admin]
│   └── cancel <id>                        # Cancel [WRITE: owner/admin, --confirm]
│
├── templates
│   ├── list                               # List (GET /api/templates)
│   ├── inspect <name>                     # Detail (GET /api/templates/{name})
│   ├── create                             # Create [WRITE: owner/admin]
│   ├── update <id>                        # Update [WRITE: owner/admin]
│   ├── delete <id>                        # Delete [WRITE: owner/admin, --confirm]
│   └── apply <name> <service>             # Apply to service [WRITE: owner/admin, --confirm]
│
├── storage
│   ├── summary                            # Summary (GET /api/storage/summary)
│   ├── trend [--days 7]                   # Trend (GET /api/storage/trend)
│   ├── volumes [--days 7]                 # Volumes growth (GET /api/storage/volumes/growth)
│   └── volume <name> [--days 7]           # Volume detail (GET /api/storage/volumes/{name}/growth)
│
├── alerts                                 # List alerts (GET /api/alerts)
├── oom-events [--service] [--range 7d]    # OOM events (GET /api/oom-events)
├── change-log [--service] [--limit 100]   # Change log (GET /api/change-log)
│
├── ═══════════════════════════════════════════════════
├── ADMIN (write, JWT owner/admin)
├── ═══════════════════════════════════════════════════
│
├── users
│   ├── list                               # List (GET /api/users)
│   ├── create                             # Create [WRITE: owner/admin]
│   ├── update <id>                        # Change role [WRITE: owner/admin]
│   └── delete <id>                        # Delete [WRITE: owner/admin, --confirm]
│
├── api-keys
│   ├── list                               # List (GET /api/auth/api-keys)
│   ├── create                             # Create [WRITE: owner/admin]
│   ├── revoke <id>                        # Revoke [WRITE: owner/admin, --confirm]
│   └── update <id>                        # Update name/scopes [WRITE: owner/admin]
│
├── settings
│   ├── list                               # List (GET /api/settings)
│   └── update                             # Update [WRITE: owner/admin]
│
├── prune
│   ├── preview                            # Preview counts (GET /api/prune/preview)
│   ├── services [--dry-run]               # Prune stale services [WRITE, --dry-run default]
│   ├── nodes [--dry-run]                  # Prune stale nodes [WRITE, --dry-run default]
│   ├── tasks [--dry-run]                  # Prune orphan tasks [WRITE, --dry-run default]
│   ├── metrics [--dry-run]                # Prune old metrics [WRITE, --dry-run default]
│   ├── change-log [--dry-run]             # Prune change-log [WRITE, --dry-run default]
│   └── volume-metrics [--dry-run]         # Prune volume-metrics [WRITE, --dry-run default]
│
├── ═══════════════════════════════════════════════════
├── AUTH
├── ═══════════════════════════════════════════════════
│
├── auth
│   ├── status                             # System initialized? (GET /api/auth/status)
│   ├── login                              # Login → JWT (POST /api/auth/login)
│   ├── logout                             # Logout (POST /api/auth/logout)
│   ├── me                                 # Current user (GET /api/auth/me)
│   ├── change-password                    # Change password (POST /api/auth/change-password)
│   ├── profile                            # Update profile (PUT /api/auth/profile)
│   └── onboarding                         # Create initial admin (POST /api/auth/onboarding)
│
├── ═══════════════════════════════════════════════════
├── STREAMING (SSE — Bearer token)
├── ═══════════════════════════════════════════════════
│
├── stream <topic>                         # Inline SSE (Ctrl+C to stop)
│                                          # Topics:
│                                          #   metrics | dashboard | events | services
│                                          #   nodes | tasks | agents | change-log
│                                          #   service-detail/<name>
│                                          #   container-detail/<id>
│
├── ═══════════════════════════════════════════════════
├── TUI (Bubble Tea — alt-screen)
├── ═══════════════════════════════════════════════════
│
├── monitor [--service <name>]             # Interactive dashboard with 6 tabs:
│                                          #   [1] Services  [2] Nodes  [3] Agents
│                                          #   [4] Tasks     [5] Alerts [6] Recommendations
│
├── ═══════════════════════════════════════════════════
├── OPERATIONAL (local — não passa pela API)
├── ═══════════════════════════════════════════════════
│
├── agent
│   ├── status                             # Local agent status (GET localhost:8082/info)
│   └── health                             # Local agent health (GET localhost:8082/health)
│
├── test
│   └── smoke                              # Smoke test 32 endpoints (port of cmd/smoke-test)
│
├── ═══════════════════════════════════════════════════
├── INFRA
├── ═══════════════════════════════════════════════════
│
├── health                                 # API health (GET /health)
├── ready                                  # API ready (GET /ready)
├── version                                # CLI version
└── completion <shell>                     # Generate completions (bash|zsh|fish|powershell)
```

### 4.2 Flags globais (root command)

| Flag | Env var | Config key | Default | Descrição |
|------|---------|------------|---------|-----------|
| `--api-url` | `RESMA_API_URL` | `api.url` | `http://localhost:8080` | API base URL |
| `--token` | `RESMA_TOKEN` | `auth.token` | (empty) | JWT token (Bearer) |
| `--api-key` | `RESMA_API_KEY` | `auth.api_key` | (empty) | API key — switches to `/api/v1/*` |
| `--config` | `RESMA_CONFIG` | — | `~/.config/resma/config.yaml` | Config file path |
| `--output` | `RESMA_OUTPUT` | `output.format` | `text` | Format: `text\|json\|yaml\|table` |
| `--no-color` | `NO_COLOR` | `output.color` | `false` | Disable colors |
| `--timeout` | `RESMA_TIMEOUT` | `api.timeout` | `30s` | REST request timeout |
| `--debug` | `RESMA_DEBUG` | `debug` | `false` | Debug logs to stderr |

### 4.3 Flags de escrita (comandos WRITE)

| Flag | Descrição |
|------|-----------|
| `--confirm` / `-y` | Skip confirmation prompt (for scripts/CI) |
| `--dry-run` | Prune only: preview what would be deleted (default: true for prune) |

### 4.4 Precedência de config (Viper)

```
flag explícita > env var > config file > default
```

### 4.5 Config file padrão (`~/.config/resma/config.yaml`)

```yaml
api:
  url: http://localhost:8080
  timeout: 30s
auth:
  token: ""        # ou usar env RESMA_TOKEN
  api_key: ""      # ou usar env RESMA_API_KEY
output:
  format: text     # text | json | yaml | table
  color: true      # respeita NO_COLOR
debug: false
```

### 4.6 Dual auth — JWT vs API Key

| Modo | Condição | Endpoints usados | Header |
|------|----------|-----------------|--------|
| **JWT** | `--token` ou `RESMA_TOKEN` presente | `/api/*` (interno) | `Authorization: Bearer <jwt>` |
| **API Key** | `--api-key` ou `RESMA_API_KEY` presente | `/api/v1/*` (público, read-only) | `Authorization: Bearer <api-key>` |
| **Erro** | Nenhum dos dois | — | "authentication required: set --token or --api-key" |

> **Nota:** API Key só tem acesso a endpoints de leitura (`/api/v1/*`). Comandos WRITE exigem JWT.
> SSE aceita ambos (cookie ou Authorization header).
>
> **SSE no CLI:** O CLI usa `Authorization: Bearer` direto para SSE — NÃO usa os endpoints
> `/api/sse/session` (POST/DELETE) para troca de cookie. Esses endpoints existem para o
> frontend (browser), que precisa de cookie HttpOnly. O CLI envia o JWT diretamente no
> header `Authorization` da requisição SSE, sem sessão intermediária.

### 4.7 Exemplos de uso

```bash
# Consulta rápida (inline table)
resma services list
resma nodes list
resma agents list

# Métricas com sparkline
resma services metrics api --range 1h
resma nodes metrics node-1

# Output JSON para pipe
resma agents list --output json | jq '.[] | select(.status=="active")'
resma tasks list --output json | jq '.[] | select(.status=="failed")'

# Streaming SSE inline (Ctrl+C to stop)
resma stream metrics
resma stream agents
resma stream service-detail/api

# Dashboard TUI interativo (6 tabs)
resma monitor
resma monitor --service api

# Operações admin (pede confirmação)
resma recommendations apply api --confirm
resma prune metrics --confirm          # executa de verdade (não dry-run)
resma prune metrics                     # dry-run por padrão (preview only)
resma users create --role admin --username john

# Operacional (local, não passa pela API)
resma agent status                      # consulta localhost:8082/info
resma test smoke                        # 32 endpoint tests

# Shell completions
resma completion powershell | Out-File | Invoke-Expression
resma completion bash >> ~/.bashrc
```

---

## 5. SSE — Implementação

### 5.1 Tópicos SSE suportados (10)

| Tópico | Endpoint | Descrição | CLI command |
|--------|----------|-----------|-------------|
| `metrics` | `GET /api/sse/metrics` | Stream de métricas | `resma stream metrics` |
| `dashboard` | `GET /api/sse/dashboard` | Stream de dashboard | `resma stream dashboard` |
| `events` | `GET /api/sse/events` | Docker events (OOM, rollback) | `resma stream events` |
| `services` | `GET /api/sse/services` | Mudanças de serviços | `resma stream services` |
| `nodes` | `GET /api/sse/nodes` | Mudanças de nós | `resma stream nodes` |
| `tasks` | `GET /api/sse/tasks` | Mudanças de tasks | `resma stream tasks` |
| `agents` | `GET /api/sse/agents` | Mudanças de agents | `resma stream agents` |
| `change-log` | `GET /api/sse/change-log` | Change-log | `resma stream change-log` |
| `service-detail/<name>` | `GET /api/sse/service-detail/{name}` | Detail de serviço | `resma stream service-detail/api` |
| `container-detail/<id>` | `GET /api/sse/container-detail/{id}` | Detail de container | `resma stream container-detail/abc123` |

> **Bug corrigido:** A spec anterior listava `stream alerts` — esse tópico **não existe**. Alerts
> vêm via tópicos `events` (OOM, rollback) e `metrics` (resource drift).

### 5.2 Parser SSE via stdlib (sem deps externas)

```go
// internal/api/sse.go
package api

import (
    "bufio"
    "context"
    "net/http"
    "strings"
)

type SSEEvent struct {
    Event string
    Data  string
    ID    string
}

// StreamSSE connects to an SSE endpoint and sends events to the channel.
// Cancels via ctx.Done(). Returns error on connection failure.
func (c *Client) StreamSSE(ctx context.Context, path string, ch chan<- SSEEvent) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Accept", "text/event-stream")
    req.Header.Set("Cache-Control", "no-cache")
    if c.token != "" {
        req.Header.Set("Authorization", "Bearer "+c.token)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line

    var event SSEEvent
    for scanner.Scan() {
        line := scanner.Text()
        switch {
        case strings.HasPrefix(line, "event:"):
            event.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
        case strings.HasPrefix(line, "data:"):
            event.Data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
        case strings.HasPrefix(line, "id:"):
            event.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
        case line == "":
            // Event boundary — dispatch
            if event.Data != "" {
                select {
                case ch <- event:
                case <-ctx.Done():
                    return ctx.Err()
                }
            }
            event = SSEEvent{}
        }
    }
    return scanner.Err()
}
```

### 5.3 Integração com Bubble Tea (channel bridge)

```go
// internal/tui/dashboard.go
type metricMsg api.SSEEvent

type DashboardModel struct {
    client   *api.Client
    dataChan chan api.SSEEvent
    ctx      context.Context
    cancel   context.CancelFunc
    // ... tab state, selected service, etc.
}

func (m DashboardModel) Init() tea.Cmd {
    return tea.Batch(
        m.listenSSE(),      // producer: goroutine reads SSE → channel
        m.waitForMetric(),  // consumer: blocks on channel → Msg
    )
}

func (m DashboardModel) listenSSE() tea.Cmd {
    return func() tea.Msg {
        m.client.StreamSSE(m.ctx, "/api/sse/metrics", m.dataChan)
        return sseErrorMsg{}
    }
}

func (m DashboardModel) waitForMetric() tea.Cmd {
    return func() tea.Msg { return metricMsg(<-m.dataChan) }
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case metricMsg:
        m.processMetric(msg)
        return m, m.waitForMetric() // re-arm consumer
    case sseErrorMsg:
        return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return reconnectMsg{} })
    case reconnectMsg:
        return m, tea.Batch(m.listenSSE(), m.waitForMetric())
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            m.cancel()
            return m, tea.Quit
        }
    }
    return m, nil
}
```

### 5.4 SSE inline (modo não-TUI)

```go
// internal/cli/stream.go
func runStream(cmd *cobra.Command, args []string) error {
    topic := args[0] // metrics | dashboard | events | services | nodes | tasks | agents | change-log
    path := fmt.Sprintf("/api/sse/%s", topic)

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    ch := make(chan api.SSEEvent, 100)
    errCh := make(chan error, 1)

    go func() { errCh <- client.StreamSSE(ctx, path, ch) }()

    for {
        select {
        case event := <-ch:
            fmt.Println(renderSSELine(event)) // Lipgloss styled
        case err := <-errCh:
            return err
        case <-ctx.Done():
            return nil
        }
    }
}
```

---

## 6. TUI Dashboard — `resma monitor`

### 6.1 Visão geral

O `monitor` abre um alt-screen TUI com **6 tabs** correspondentes aos domínios do RESMA. Cada tab
consome um ou mais tópicos SSE e renderiza dados em tempo real.

### 6.2 Tabs

| Tab | Tecla | Tópicos SSE | Conteúdo |
|-----|-------|-------------|----------|
| **Services** | `1` | `services`, `metrics` | Tabela de serviços com sparklines de CPU/mem |
| **Nodes** | `2` | `nodes` | Lista de nós com consumo agregado |
| **Agents** | `3` | `agents` | Lista de agents com status (active/stale) |
| **Tasks** | `4` | `tasks` | Lista de tasks com lifecycle status |
| **Alerts** | `5` | `events`, `metrics` | Feed de alertas (OOM, memory leak, drift) |
| **Recommendations** | `6` | `services` | Cards de recomendação com tier + risk + delta |

### 6.3 Keybindings

| Tecla | Ação |
|-------|------|
| `1`-`6` | Switch tab |
| `Tab` / `Shift+Tab` | Next/prev tab |
| `s` | Select service (no tab Services) |
| `r` | Refresh |
| `/` | Filter/search |
| `?` | Help overlay |
| `q` / `Ctrl+C` | Quit |

### 6.4 Mockup

```
┌─ RESMA Monitor ────────────────────────────── 14:32:15 ─┐
│ [1]Services  [2]Nodes  [3]Agents  [4]Tasks  [5]Alerts  [6]Recs │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  SERVICES                                                │
│  ┌──────────────────────────────────────────┐           │
│  │ NAME        REPLICAS  CPU%   MEM%   STATUS│           │
│  │ api         3/3       12.4   45.2  running│           │
│  │ ml          1/1        2.1   78.9  running│           │
│  │ frontend    1/1        0.3   12.1  running│           │
│  └──────────────────────────────────────────┘           │
│                                                          │
│  CPU — service: api (live)                              │
│  100% ┤      ╭──╮                                        │
│   50% ┤ ╭────╯  ╰──╮                                     │
│    0% ┼─╯         ╰─────────────────────────             │
│                                                          │
│  [q]quit  [s]select  [1-6]tabs  [/]filter  [?]help      │
└──────────────────────────────────────────────────────────┘
```

> Detalhes completos do TUI: [tui-design.md](./tui-design.md)

### 6.5 Efeito Flash — FlashField

Quando um valor de métrica muda via SSE, ele pisca em destaque (bold + cor de destaque) por ~800ms e volta à cor normal. Feedback visual de que o dado é live.

- Componente reutilizável: `FlashField` (flash_field.go)
- Cada métrica tem seu próprio FlashField (CPU, MEM independentes)
- `Update(value)` aciona o flash se o valor mudou
- `Flashing()` retorna true se dentro da janela de flash
- Tick de 500ms re-renderiza para expirar o flash

### 6.6 Fetch inicial REST

Ao entrar no `resma monitor`, um GET /api/dashboard é feito imediatamente (goroutine paralela ao SSE) para carregar dados sem esperar até 60s pelo próximo evento SSE.

---

## 7. Compatibilidade Cross-Terminal

### 7.1 Detecção de capacidades (termenv + colorprofile)

> **Nota de implementação:** A implementação real NÃO chama `termenv` ou
> `colorprofile` diretamente. O `lipgloss` já inclui essas deps como transitivas
> e cuida da detecção de perfil de cor e VT processing automaticamente. O código
> abaixo é referência técnica de como funciona internamente.

```go
// internal/config/config.go
func SetupTerminal() (*termenv.Output, colorprofile.Profile) {
    // Enable VT processing on Windows (safe no-op on Unix)
    restore, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
    if err == nil {
        defer restore()
    }
    output := termenv.NewOutput(os.Stdout)
    profile := colorprofile.Detect(os.Stdout, os.Environ())
    return output, profile
}
```

### 7.2 Matriz de compatibilidade

| Terminal | Plataforma | Cor | VT Processing | TUI | Notas |
|----------|-----------|-----|---------------|-----|-------|
| Windows Terminal | Windows 10+ | TrueColor | ✅ nativo | ✅ | Recomendado |
| PowerShell (Win10+) | Windows | TrueColor/256 | ✅ via termenv | ✅ | |
| cmd.exe (Win10+) | Windows | ANSI256 | ✅ via termenv | ✅ | ConHost |
| cmd.exe (pré-Win10) | Windows | ASCII | ❌ | ❌ | Fallback sem cor |
| bash (GNOME Terminal) | Linux | TrueColor | ✅ nativo | ✅ | |
| zsh (iTerm2) | macOS | TrueColor | ✅ nativo | ✅ | |
| fish | Linux/macOS | TrueColor | ✅ nativo | ✅ | |
| SSH (dumb) | Qualquer | ASCII | ❌ | ❌ | `NO_COLOR` detectado |
| CI (GitHub Actions) | Linux | ANSI | ✅ | ❌ | Non-TTY: texto plano |

### 7.3 Fallback gracioso

```go
// Se não for TTY (pipe/redirecionamento), desabilitar cores e TUI
if !isatty.IsTerminal(os.Stdout.Fd()) {
    cfg.NoColor = true
    // `resma monitor` recusa rodar em non-TTY, sugere `resma stream`
}
```

### 7.4 Env vars respeitadas

| Env var | Comportamento |
|---------|---------------|
| `NO_COLOR` | Desabilita todas as cores (https://no-color.org/) |
| `CLICOLOR=0` | Desabilita cores |
| `CLICOLOR_FORCE=1` | Força cores mesmo em non-TTY |
| `TERM=dumb` | Fallback ASCII |

> Detalhes completos: [compatibility-matrix.md](./compatibility-matrix.md)

---

## 8. Output — Formatos

### 8.1 Text (default) — tabela Lipgloss

```
$ resma services list

  SERVICE        REPLICAS    CPU%    MEM%    STATUS     NODES
  api            3/3         12.4    45.2    running    3
  ml             1/1         2.1     78.9    running    1
  frontend-dev   1/1         0.3     12.1    running    1
```

### 8.2 JSON (para pipes/scripts)

```bash
$ resma services list --output json
[
  {"name":"api","replicas":"3/3","cpu":12.4,"mem":45.2,"status":"running","nodes":3}
]
```

### 8.3 Sparkline (braille Unicode custom)

> **Nota de implementação:** A implementação real usa gráficos **braille Unicode
> custom** (`internal/tui/braille_chart.go`, `sparkline.go`) em vez de `asciigraph`.
> O `asciigraph` foi avaliado e rejeitado em favor de um componente próprio que
> se integra melhor com o Bubble Tea e suporta multi-series com gradiente de cores.

```
$ resma services metrics api --range 1h

  CPU usage — service: api (last 1h)

  100% ┤                    ╭──╮
   80% ┤              ╭─────╯  ╰──╮
   60% ┤         ╭────╯           ╰──
   40% ┤    ╭────╯
   20% ┤────╯
    0% ┼──────────────────────────────────
       13:00    13:15    13:30    13:45    14:00

  avg: 34.2%  |  p95: 78.1%  |  max: 89.3%
```

### 8.4 Table (explicit)

```
$ resma agents list --output table

  NODE ID     HOSTNAME          STATUS    CONTAINERS    LAST HEARTBEAT      VERSION
  node-1      swarm-manager     active    12            2026-08-06 14:32    0.7.2
  node-2      swarm-worker-1    active    8             2026-08-06 14:32    0.7.2
  node-3      swarm-worker-2    stale     5             2026-08-06 14:10    0.7.2
```

---

## 9. Confirmação Interativa (writes)

Comandos de escrita (WRITE) pedem confirmação interativa por padrão:

```
$ resma recommendations apply api

  Service: api
  Current:  CPU 1.0  Mem 1GB
  Recommended: CPU 0.5  Mem 512MB
  Savings:  CPU 50%  Mem 50%

  Proceed with apply? [y/N]:
```

Com `--confirm` (ou `-y`), pula o prompt (para scripts/CI):

```bash
resma recommendations apply api --confirm
resma prune metrics --confirm
resma users delete 5 --confirm
```

### Prune — dry-run por padrão

Prune é destrutivo. Sem `--confirm`, executa em modo `--dry-run` (preview only):

```
$ resma prune metrics

  DRY RUN — no data will be deleted

  Metrics older than retention (30 days): 1,234,567 rows
  Estimated space freed: ~450MB

  Run with --confirm to execute.
```

```
$ resma prune metrics --confirm

  Deleting metrics older than retention (30 days)...
  Deleted 1,234,567 rows
  Space freed: ~450MB
```

---

## 10. Autenticação

### 10.1 Métodos de auth

| Método | Header | Endpoints | Uso |
|--------|--------|-----------|-----|
| JWT | `Authorization: Bearer <jwt>` | `/api/*` (interno, read+write) | Interativo (login) |
| API Key | `Authorization: Bearer <api-key>` | `/api/v1/*` (público, read-only) | Automação/scripts |

### 10.2 Fluxo de auth

```bash
# API Key via env (recomendado para scripts)
export RESMA_API_KEY="resma_xxxxxxxxxxxx"
resma services list              # usa /api/v1/services

# JWT via login interativo (futuro)
resma auth login --url http://swarm-manager:8080
# → pede username/password
# → armazena JWT em ~/.config/resma/credentials.json (0600, XDG-compatible)
resma services list              # usa /api/services

# Token via flag
resma services list --token "eyJhbGci..."
```

### 10.3 RBAC — role requirements

| Role | Acesso |
|------|--------|
| `owner` | Tudo (incluindo write) |
| `admin` | Tudo exceto gerenciar owners |
| `user` | Read-only (sem write) |

> O CLI avisa se o token não tem role suficiente para um comando WRITE.

---

## 11. Build e Distribuição

### 11.1 Dockerfile (multi-stage, CGO_ENABLED=0)

```dockerfile
FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags)" \
    -o /resma-cli ./cmd/resma-cli

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /resma-cli /resma-cli
ENTRYPOINT ["/resma-cli"]
```

> **CGO_ENABLED=0** — o CLI não usa go-duckdb (só faz HTTP), então pode ser buildado sem gcc.
> Binário puramente estático, ~6-8MB.

### 11.2 GoReleaser

```yaml
project_name: resma-cli
builds:
  - main: ./cmd/resma-cli
    env: [CGO_ENABLED=0]
    goos: [linux, windows, darwin]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}}
archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
```

### 11.3 Instalação

```bash
# Via go install
go install github.com/resma/resma-cli/cmd/resma-cli@latest

# Via binary download
curl -L https://github.com/resma/resma-cli/releases/latest/download/resma-cli_linux_amd64.tar.gz | tar xz
sudo mv resma-cli /usr/local/bin/

# Via Docker
docker run --rm ghcr.io/resma/resma-cli:latest services list --api-url http://host:8080
```

---

## 12. Roadmap de Implementação

| Fase | Escopo | Comandos | Stack | Entregável |
|------|--------|----------|-------|------------|
| **MVP** | Consulta core + streaming + infra | ~20 | Cobra + Lipgloss + asciigraph + SSE stdlib + Viper | Binário funcional com output inline |
| **Fase 2** | TUI monitor (6 tabs) | +1 (complexo) | + Bubble Tea + Bubbles | Dashboard interativo alt-screen |
| **Fase 3** | Admin completo (users, api-keys, settings, prune, templates, schedules) | +22 | Cobra commands + confirm + output formats | Cobertura total de escrita |
| **Fase 4** | Auth completo + operacional (agent local, smoke test) | +10 | + auth login + localhost client | DX completo |
| **Fase 5** | Recomendações avançadas (simulate, rollback-watches) | +6 | + simulate UI + rollback watch | Right-sizing via CLI |
| **Fase 6** | Distribuição (GoReleaser, Docker, homebrew, scoop) | — | CI/CD | Release multi-plataforma |

### 12.1 MVP — detalhamento

| Comando | Endpoint | Prioridade |
|---------|----------|-----------|
| `resma services list` | `GET /api/services` | P0 |
| `resma services inspect <name>` | `GET /api/services/{name}/stats` | P0 |
| `resma services metrics <name>` | `GET /api/services/{name}/metrics` | P0 |
| `resma services health` | `GET /api/services/health` | P0 |
| `resma nodes list` | `GET /api/nodes` | P0 |
| `resma agents list` | `GET /api/agents` | P0 |
| `resma tasks list` | `GET /api/tasks` | P0 |
| `resma dashboard` | `GET /api/dashboard` | P1 |
| `resma stream metrics` | `GET /api/sse/metrics` | P0 |
| `resma stream agents` | `GET /api/sse/agents` | P1 |
| `resma stream tasks` | `GET /api/sse/tasks` | P1 |
| `resma health` | `GET /health` | P0 |
| `resma version` | (local) | P0 |
| `resma completion <shell>` | (local) | P1 |
| `resma agent status` | `GET localhost:8082/info` | P1 |
| `resma alerts` | `GET /api/alerts` | P1 |
| `resma storage summary` | `GET /api/storage/summary` | P1 |
| `resma recommendations list` | `GET /api/recommendations` | P1 |
| `resma change-log` | `GET /api/change-log` | P2 |
| `resma oom-events` | `GET /api/oom-events` | P2 |

---

## 13. Riscos e Mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Bubble Tea v2.0 em beta | Média | Médio | Pinar v1.3.x estável; migrar v2 após estabilização |
| Curva de aprendizado Elm Architecture | Alta | Baixo | MVP sem TUI (só inline); adiar Bubble Tea para Fase 2 |
| Goroutine leak no SSE | Média | Alto | `context.Cancel` + `defer resp.Body.Close()` + buffer channel |
| Windows legacy cmd.exe (pré-Win10) | Baixa | Médio | Declarar requisito Win10+; fallback ASCII via termenv |
| Manutenção Cobra (backlog) | Baixa | Baixo | Stable há anos; kubectl/gh dependem; risk aceitável |
| Breaking changes na API Go | Média | Alto | CLI versiona independente; types.go espelha payloads |
| Non-TTY detection falha | Baixa | Médio | `isatty` check + `--no-color` flag + `NO_COLOR` env |
| Comando WRITE executado sem querer | Média | Alto | `--confirm` obrigatório; prune `--dry-run` por padrão |
| API Key sem role para WRITE | Baixa | Baixo | CLI avisa antes de tentar; API rejeita com 403 |

---

## 14. Decisões do Technical Council

| Decisão | Votação | Justificativa |
|---------|---------|---------------|
| Charmbracelet ecosystem como stack principal | Unânime (4/4) | Provas em produção (k9s, dry, lazydocker); Elm Architecture; Windows VT |
| Cobra + Viper para CLI structure | Unânime (4/4) | Padrão da indústria; completions multi-shell; integração nativa |
| SSE via stdlib (sem r3labs/sse) | Unânime (4/4) | r3labs/sse maintenance 0/10; SSE é HTTP trivial |
| MVP sem Bubble Tea (só inline) | 3/4 (Critic abstain) | Reduz risco; valida API client antes do TUI |
| asciigraph para sparklines | Unânime (4/4) | Zero deps; real-time mode; ASCII puro compatível |
| termenv para Windows VT | Unânime (4/4) | `EnableVirtualTerminalProcessing` é explícito e documentado |
| Rejeitar termui, tview, gocui, uiprogress, spinner | Unânime (4/4) | Maintenance 0-1/10; risco de abandono |
| Expandir árvore para ~76 comandos | Unânime (4/4) | Cobertura total da API (105 endpoints) |
| Remover `alerts ack` (bug) | Unânime (4/4) | Endpoint não existe na API |
| Remover `stream alerts` (bug) | Unânime (4/4) | Tópico SSE não existe; alerts vêm via `events` |
| Prune sempre `--dry-run` por padrão | Unânime (4/4) | Operação destrutiva |
| Comandos write exigem `--confirm` | Unânime (4/4) | Prevenir acidentes |
| TUI com 6 tabs (não dashboard único) | Unânime (4/4) | Densidade de informação por domínio |
| `agent status` consulta localhost:8082 | Unânime (4/4) | Quick win, não precisa da API |
| Portar smoke-test para `resma test smoke` | Unânime (4/4) | Já existe como standalone, trivial de portar |
| American English para todos os nomes | Unânime (4/4) | Diretriz do usuário; padrão de CLIs |
| Dual auth (JWT + API Key) | Unânime (4/4) | JWT para interativo, API Key para automação |

---

## 15. Cobertura da API

### 15.1 Endpoints consumidos pelo CLI

| Grupo | Endpoints API | Comandos CLI | Status |
|-------|--------------|-------------|--------|
| Infra | 2 | `health`, `ready` | ✅ |
| Auth | 8 | `auth status/login/logout/me/change-password/profile/onboarding` | ✅ |
| API Keys | 4 | `api-keys list/create/revoke/update` | ✅ |
| Services | 10 | `services list/inspect/metrics/containers/sparklines/health/archive/restore` | ✅ |
| Containers | 3 | `containers inspect/metrics/network` | ✅ |
| Nodes | 5 | `nodes list/cluster/inspect/metrics/services` | ✅ |
| Agents | 2 | `agents list/inspect` | ✅ |
| Tasks | 4 | `tasks list/show/history` + `services health` | ✅ |
| Recommendations | 7 | `recommendations list/show/triggers/storage/recalculate/simulate/apply` | ✅ |
| Rollback Watches | 4 | `rollback-watches list/inspect/rollback/cancel` | ✅ |
| Schedules | 5 | `schedules list/pending/history/create/cancel` | ✅ |
| Templates | 6 | `templates list/inspect/create/update/delete/apply` | ✅ |
| Storage | 4 | `storage summary/trend/volumes/volume` | ✅ |
| Users | 4 | `users list/create/update/delete` | ✅ |
| Settings | 2 | `settings list/update` | ✅ |
| Prune | 7 | `prune preview/services/nodes/tasks/metrics/change-log/volume-metrics` | ✅ |
| Alerts | 1 | `alerts` | ✅ |
| OOM Events | 1 | `oom-events` | ✅ |
| Change-Log | 2 | `change-log` | ✅ |
| Dashboard | 1 | `dashboard` | ✅ |
| SSE | 10 tópicos | `stream <topic>` + `monitor` (6 tabs) | ✅ |
| Agent local | 2 | `agent status/health` | ✅ |
| Smoke test | 32 tests | `test smoke` | ✅ |
| **Total** | **105 + 10 SSE + 2 local** | **~76 comandos** | **100%** |

### 15.2 Endpoints NÃO consumidos pelo CLI (intencional)

| Grupo | Endpoints | Motivo |
|-------|-----------|--------|
| Agent Ingest | 3 (`POST /api/agent/*`) | Agent-only (binary Go, não CLI) |
| Internal ML | 6 (`GET /api/internal/*`) | ML sidecar only (Docker network) |
| SSE Session | 2 (`POST/DELETE /api/sse/session`) | CLI usa Bearer header, não cookie |
| Swagger | 1 (`/swagger/*`) | UI only, não CLI |

---

## 16. Referências

- **Bubble Tea:** https://github.com/charmbracelet/bubbletea
- **Lipgloss:** https://github.com/charmbracelet/lipgloss
- **Bubbles:** https://github.com/charmbracelet/bubbles
- **termenv:** https://github.com/muesli/termenv
- **colorprofile:** https://github.com/charmbracelet/colorprofile
- **Cobra:** https://github.com/spf13/cobra
- **Viper:** https://github.com/spf13/viper
- **asciigraph:** https://github.com/guptarohit/asciigraph
- **Fang (Cobra + Charm):** https://github.com/charmbracelet/fang
- **Provas reais:** k9s, dry, lazydocker
- **NO_COLOR spec:** https://no-color.org/
- **SSE spec:** https://html.spec.whatwg.org/multipage/server-sent-events.html

---

## 17. Documentos relacionados

- [stack-benchmark.md](./stack-benchmark.md) — Benchmark detalhado das 17 libs avaliadas
- [api-contract.md](./api-contract.md) — Contrato de todos os 105 endpoints da API
- [tui-design.md](./tui-design.md) — Design do dashboard TUI (6 tabs, two-column layout, command mode, skins, hotkeys, drill-down, filter, mouse)
- [sse-patterns.md](./sse-patterns.md) — Padrões de SSE em Go (stdlib + Bubble Tea)
- [compatibility-matrix.md](./compatibility-matrix.md) — Matriz de compatibilidade terminal
- [commands/](./commands/index.md) — Referência detalhada de cada comando (dividida em 3 arquivos):
  - [01-query.md](./commands/01-query.md) — 39 comandos read-only
  - [02-admin.md](./commands/02-admin.md) — 29 comandos write/admin
  - [03-auth-streaming-tui.md](./commands/03-auth-streaming-tui.md) — 25 comandos auth/streaming/TUI/ops/infra
