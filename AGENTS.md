# RESMA — AGENTS.md

> **RESMA** — RESource MAnager for Docker Swarm.
> App para gerenciar recursos (CPU/memória) de containers no Docker Swarm com coleta de métricas, análise estatística, ML para recomendações de limites e detecção de memory leaks.

> **AMBIENTE DE DEPLOY (OBRIGATÓRIO):** o RESMA roda em **Docker Swarm**. Em produção,
> o deploy é feito como **stack** via `docker stack deploy` (ver `scripts/deploy-swarm.ps1`),
> **NUNCA** como container isolado ou standalone via `docker compose up`. O
> `docker-compose.yml` (com profiles `dev`/`prod`) e o `docker-compose.standalone.yml`
> são **apenas para desenvolvimento local** — não são a forma de publicação. Qualquer
> instrução de deploy, documentação ou script deve tratar o sistema como uma **stack
> Swarm** (serviços replicados/global, multi-node, rede overlay), não como container
> único ou compose-up standalone.
>
> **AMBIENTE LOCAL (dev):** o dev local também roda como **stack Swarm** — não usar
> `docker compose up`. Para subir/verificar: `docker stack ls`, `docker stack services
> resma`, `docker stack ps resma`. A stack `resma` tem 3 serviços (api, ml, frontend-dev)
> e a stack `resma-test` tem 9 serviços (incluindo agents e serviços de teste).

## AI Engineering Framework

**Bootstrap obrigatório:** ler [.ai/AGENTS.md](.ai/AGENTS.md) e executar `.ai/skills/orchestrator/SKILL.md` antes de qualquer tarefa.

O Orchestrator é o único agente que conversa com o usuário. Nenhuma skill auto-inicia.

## Stack resumida (pós-Fase 0b — migração Go em andamento)

| Camada | Tecnologia | Status |
|--------|-----------|--------|
| API | Go 1.26 + net/http + DuckDB (go-duckdb) + Docker SDK (moby/moby) | Scaffold 0b.1 pronto, migração em andamento |
| ML sidecar | Python 3.12 + scikit-learn + scipy + numpy + httpx (FastAPI minimal) | Sem acesso direto ao DuckDB — obtém dados via HTTP do Go API |
| Frontend | React 19 + Vite 6 + TailwindCSS 4 + shadcn/ui | Sem mudanças até 0b.11 |
| Auth | JWT (golang-jwt) + bcrypt + API key com scopes | A implementar em 0b.4 |
| Real-time | SSE (Server-Sent Events) via stdlib net/http + http.Flusher | A implementar em 0b.7 |
| Deploy | Docker (docker-compose.yml com profiles dev/prod, Docker Swarm) | Funcional |
| Backend Python (legacy) | Python 3.12 + FastAPI + aiodocker + DuckDB | **Referência para portar — NÃO recebe alterações** |
| **RESMA Agent (Fase 7)** | **Go binary leve (~6 MB) — coleta stats multi-node via Docker socket local e faz push HTTP para o Go API** | **Implementado (7.1-7.8)** |

> **Importante:** O backend Python em `backend/` é a referência para portar durante a Fase 0b. Ele não deve ser modificado. Toda implementação nova acontece em `app/api/` (Go) e `app/ml/` (Python sidecar).

### Arquitetura de acesso ao DuckDB

**O Go API é o único owner do DuckDB.** O ML sidecar NÃO acessa o DuckDB diretamente — DuckDB tem lock exclusivo e não suporta multi-processo.

```
┌─────────────┐     HTTP      ┌─────────────┐     DuckDB     ┌────────┐
│  ML sidecar  │ ───────────→ │   Go API    │ ────────────→ │ DuckDB │
│  (Python)    │  /api/internal/* │  (Go)    │   (exclusive)  │        │
└─────────────┘              └─────────────┘                └────────┘
```

O Go API expõe endpoints internos (`/api/internal/*`) sem JWT (apenas rede Docker) para o ML sidecar:
- `GET /api/internal/services/with-metrics` — lista de serviços com métricas
- `GET /api/internal/services/{service}/metrics` — série temporal bruta
- `GET /api/internal/services/{service}/oom-count` — contagem de OOMs
- `GET /api/internal/services/{service}/config` — config de recursos
- `GET /api/internal/storage/volumes/metrics` — métricas de volume

## Portas

| Serviço | Porta | Ambiente |
|---------|-------|----------|
| API Go | 8080 | Dev (Docker) e produção |
| ML sidecar | 8081 | Interno (Docker network apenas) |
| Frontend dev (vite) | 5173 | Dev (Docker ou host, proxy /api → 8080) |
| Docs-site | 3001 | Dev (host) |

## Comandos

| Área | Comando | Onde rodar |
|------|---------|------------|
| **Tudo (dev multi-node)** | `docker compose up -d` | Host — sobe API + ML + frontend + 5 agents (default) |
| API Go dev (Docker) | `docker compose up -d api` | Host — api inicia automaticamente com air (hot reload) |
| API Go build (Docker) | `docker compose exec api go build ./...` | Container api |
| API Go test (Docker) | `docker compose exec api go test ./...` | Container api |
| API Go logs | `docker compose logs -f api` | Host |
| Frontend logs | `docker compose logs -f frontend-dev` | Host |
| API Go prod (Swarm) | `.\scripts\deploy-swarm.ps1` | Host — build + docker stack deploy |
| Frontend dev (host) | `pnpm dev` (em `frontend/`) | Host — alternativa ao container frontend-dev |
| Frontend build | `pnpm build` (em `frontend/`) | Host |
| Docs-site dev | `pnpm start` (em `docs-site/`) | Host |
| Docs-site build | `pnpm build` (em `docs-site/`) | Host |
| **RESMA Agent dev** | `docker compose up -d agent-dev` | Host — agent-dev coleta stats locais e faz push para api |
| **Dev standalone** | `docker compose -f docker-compose.standalone.yml up -d` | Host — dev sem workers (1 agent apenas) |
| **Smoke test** | `docker compose exec api go run ./cmd/smoke-test` | Container api — 32 testes end-to-end |

> **Go é buildado e executado PRIMARIAMENTE em Docker** (container `api` no profile `dev`). O Go instalado no host Windows (`C:\Program Files\Go\bin\go.exe`) é apenas fallback para validações rápidas. O motivo: go-duckdb requer CGO + gcc com ABI GNU libstdc++ (Debian), que não está disponível no Windows nativo. Sempre prefira Docker para build/test/run do Go.

## Contexto do projeto

Ver `.ai/context/` para arquitetura, domínio, padrões e convenções detalhadas.

## Fase 7 — Multi-Node Agent (implementada)

A Fase 7 resolve o gap de coleta multi-node: o RESMA Agent é um binary Go leve
(~6 MB) que roda como global service no Swarm (1 por node), coleta stats dos
containers locais via Docker socket e faz push HTTP para o Go API no manager.

**Documentação:** [docs/multi-node.md](docs/multi-node.md) | [docs/task-monitoring.md](docs/task-monitoring.md)

**Componentes:**
- `app/agent/` — binary Go (collector, events, pusher, buffer, heartbeat)
- `app/api/internal/server/agent_handlers.go` — ingestion API (`/api/agent/*`)
- `app/api/internal/server/task_handlers.go` — admin endpoints (`/api/agents/*`, `/api/tasks/*`)
- `app/api/internal/db/agent_db.go` — DB methods para agents e tasks
- `frontend/src/pages/Tasks.tsx` — página de tasks do Swarm
- `frontend/src/pages/Agents.tsx` — página de agents

**Endpoints de ingestão (agent token):** `POST /api/agent/heartbeat`,
`POST /api/agent/ingest/metrics`, `POST /api/agent/ingest/oom`

**Endpoints admin (JWT):** `GET /api/agents`, `GET /api/agents/{node_id}`,
`GET /api/tasks`, `GET /api/tasks/{service}`, `GET /api/tasks/{service}/history`,
`GET /api/services/health`

**SSE topics:** `agents`, `tasks` (em `/api/sse/agents`, `/api/sse/tasks`)

---

## Orientações para Agentes de IA — Implementação OSS

> **LEIA ESTA SEÇÃO ANTES DE INICIAR QUALQUER IMPLEMENTAÇÃO.**
> Adicione estas orientações à memória do chat antes de começar a trabalhar em qualquer tarefa das Fases 0b–6.

### 1. Go é executado em Docker, não no host

- **Build, test e run do Go acontecem dentro do container `api`** (profile `dev` do docker-compose.yml)
- O container `api` herda do target `dev` do `app/api/Dockerfile` (golang:1.26-bookworm + gcc para CGO)
- **Nunca** tente buildar o Go localmente no Windows como passo principal — go-duckdb requer CGO com ABI GNU libstdc++ que não existe no Windows nativo
- O Go no host (`C:\Program Files\Go\bin\go.exe`) é apenas para validações rápidas de sintaxe (`go vet`, `gofmt`) — não para build/test completos
- Comandos padrão:
  ```bash
  docker compose up -d api          # sobe container
  docker compose exec api go build ./...   # build
  docker compose exec api go test ./...    # test
  docker compose exec api go run ./cmd/server  # run
  ```

### 2. Use subagents sempre que possível

- **Tarefas paralelas independentes** devem ser delegadas a subagents em background (ex: portar 2 routers diferentes simultaneamente)
- **Pesquisa web** deve ser delegada a subagent (não bloqueia o agente principal)
- **Revisão de consistência** deve ser delegada a subagent após cada bloco de mudanças
- **Exploração de código** deve usar subagent `subagent_explore` (read-only) quando o agente principal precisa manter contexto limpo
- Subagents em background permitem que o agente principal continue trabalhando enquanto a pesquisa/análise roda

### 3. Auto-aprovação de comandos está habilitada — restrições obrigatórias

> **AVISO CRÍTICO:** A auto-aprovação de comandos shell está habilitada neste ambiente. Isso significa que comandos executam sem confirmação manual do usuário. Portanto:

- **PROIBIDO:** Comandos de exclusão fora de arquivos do projeto:
  - `rm -rf` em diretórios que não sejam artefatos de build do próprio projeto
  - `git reset --hard`, `git clean -fd` (podem apagar trabalho não commitado)
  - `docker system prune`, `docker volume prune` (podem apagar dados de outros projetos)
  - `drop database`, `truncate table` (banco de dados)
  - Qualquer comando que modifique arquivos fora de `D:\allt\resma\`
- **PROIBIDO:** Forçar push, reescrever git history, deletar branches
- **PROIBIDO:** Enviar emails, fazer pagamentos, chamar APIs com efeitos colaterais reais
- **PERMITIDO:** Criar/editar/deletar arquivos dentro de `D:\allt\resma\` (código, configs, docs)
- **PERMITIDO:** `docker compose` commands (up, down, build, exec) — não destrutivos
- **PERMITIDO:** `go build`, `go test`, `go vet`, `gofmt` dentro do container
- **PERMITIDO:** `git add`, `git commit`, `git status`, `git diff`, `git log` (não destrutivos)
- **SEMPRE que um comando potencialmente destrutivo for necessário:** PARAR e pedir confirmação explícita do usuário, descrevendo exatamente o que será executado e por quê

### 3a. Commits locais — criar sempre que fizer sentido

- O repo tem `git init` rodado mas **sem remote configurado** — todos os commits são locais por enquanto
- **Crie commits locais livremente** ao concluir uma tarefa, sub-tarefa ou bloco coeso de mudanças (ex: ao final de cada tarefa 0b.N, ao concluir uma fase, ao adicionar um conjunto de arquivos relacionado)
- Não pergunte permissão para commitar — apenas faça commits locais com mensagens descritivas no padrão do projeto
- **NUNCA** force-push, reescreva history ou delete branches (não há remote, mas mantenha a disciplina)
- **NUNCA** commite secrets, chaves JWT reais, credenciais ou `.env` com valores reais
- Formato sugerido de mensagem: `tipo(escopo): descrição` (ex: `feat(0b.2): portar DuckDB schema init e queries`), seguido de corpo explicativo quando relevante
- Use `git status` e `git diff` antes de commitar para garantir que só arquivos do projeto entram no commit
- Inclua o rodapé `Generated with [Devin](https://devin.ai)` e o Co-Authored-By nos commits

#### Como fazer commit neste ambiente (PowerShell) — OBRIGATÓRIO

> **O shell deste ambiente é PowerShell (Windows), NÃO bash.** Sintaxe bash como
> heredoc (`<<'EOF'`), `$(cat <<EOF ...)`, ou aspas duplas com quebras de linha
> **NÃO funcionam** e quebram o commit. Sempre use uma destas duas formas:

**Forma 1 — Mensagem curta (1 linha):** aspas duplas, sem quebras de linha.

```powershell
git commit -m "fix(sse): invalidar [agents] na pagina Nodes quando SSE ativo"
```

**Forma 2 — Mensagem com corpo (multi-linha):** escreva a mensagem em um arquivo
temporário com a tool `write` e use `-F` (recomendado, evita escaping de aspas/CR).

```powershell
# 1. Escrever a mensagem no arquivo (use a tool `write`, não echo/printf)
#    Caminho sugerido: .git/COMMIT_MSG.txt (fica dentro de .git, não é tracked)
# 2. Commitar com -F
git commit -F .git/COMMIT_MSG.txt
# 3. Remover o arquivo temporário
Remove-Item .git/COMMIT_MSG.txt
```

**NUNCA use** (quebra no PowerShell):
- `git commit -m "$(cat <<'EOF' ... EOF)"` — heredoc é sintaxe bash
- `git commit -m "linha1\nlinha2"` — `\n` não é interpretado como quebra de linha
- Aspas simples com `$` ou apóstrofos internos — escaping inconsistente

**Após commitar:** rode `git log --oneline -3` para confirmar que o commit foi
criado com a mensagem correta (sem `ParserError` ou mensagem truncada).

### 4. Ordem de implementação — respeitar dependências

As Fases 0b–6 têm dependências estritas. **Não pular tarefas:**

```
Fase 0 (monorepo) → Fase 0b (Go API + SSE) → Fase 2 (security) → Fase 4 (install) → Fase 3 (docs)
                                                                → Fase 5 (CI/CD)
                                                                → Fase 6 (benchmark)
Fase 1 (legal) — paralelo com 0b (não toca em código)
```

Dentro da Fase 0b, a ordem é:
1. 0b.1 (scaffold) ✅ concluído
2. 0b.2 (DuckDB layer) → 0b.3 (Docker client) → 0b.4 (auth + API key) → 0b.5 (routers com split público/interno)
3. 0b.6 (collector + scheduler) → 0b.7 (SSE broker com cookie auth)
4. 0b.8 (ML sidecar) → 0b.9 (Go ↔ ML integration)
5. 0b.10 (Dockerfile + compose finalização)
6. 0b.11 (frontend SSE) → 0b.12 (testes de equivalência)

**O frontend só é tocado em 0b.11** — antes disso, o frontend continua consumindo a API Python sem mudanças.

### 5. Backend Python é referência — não modificar

- O diretório `backend/` contém o código Python original que está sendo portado para Go
- **NÃO modificar arquivos em `backend/`** durante a Fase 0b
- Usar `backend/` apenas como referência para entender contratos, schemas, lógica de negócio
- O `backend/` só será removido após 0b.12 (testes de equivalência) confirmar paridade total

### 6. API split — respeitar a decisão

- `/api/v1/*` — público, API key + scopes, OpenAPI via swaggo
- `/api/*` — interno/UI, JWT, não-versionado (frontend consome aqui)
- `/api/sse/*` — streaming, cookie HttpOnly (browser) ou Authorization header (clients não-browser)
- `/health`, `/ready` — infra, sem auth

**O frontend NÃO migra para `/api/v1/*`** nesta fase. Ver seção "Frontend Impact" na [spec 0b](docs/specs/oss/phase-0b-go-migration/spec.md#frontend-impact--o-que-muda-no-react).

### 7. Validação obrigatória após cada tarefa

Após completar cada tarefa (0b.2, 0b.3, etc.):
1. `docker compose exec api go build ./...` — deve compilar sem erros
2. `docker compose exec api go vet ./...` — sem warnings
3. `docker compose exec api gofmt -l .` — sem arquivos não formatados
4. Smoke test dos endpoints implementados (curl dentro do container ou do host)
5. Atualizar o checkbox da tarefa na spec 0b

### 8. Documentação — manter specs atualizadas

- Se uma decisão arquitetural mudar durante a implementação, atualizar a spec correspondente em `docs/specs/oss/`
- Se um novo padrão for descoberto (ex: melhor forma de fazer X em Go), documentar em `AGENTS.md` ou na spec
- Specs são a fonte de verdade — código deve seguir specs, e mudanças de plano devem refletir nas specs

### 9. Comunicação com o usuário

- Reportar progresso após cada tarefa concluída
- Se encontrar bloqueadores, parar e reportar — não tentar contornar silenciosamente
- Se uma decisão da spec parecer errada durante implementação, parar e discutir com o usuário antes de prosseguir
- Usar todo_write para manter visibilidade do progresso

### 10. Frontend — regra de ouro shadcn/ui

- **NUNCA alterar os componentes originais do shadcn em `frontend/src/components/ui/`** — usar sempre como estão, compondo em volta deles
- Todo componente de UI deve vir do shadcn/ui — não criar componentes custom quando existe um equivalente no shadcn
- Se o shadcn não tem um componente necessário, compor com componentes shadcn existentes (ex: combinar `Button` + `DropdownMenu` em vez de criar um `SplitButton` custom)
- Ao adicionar novos componentes shadcn, usar `npx shadcn@latest add <component>` para garantir a versão oficial
- O `Layout.tsx` deve usar o componente `Sidebar` do shadcn (`Sidebar/SidebarHeader/SidebarContent/SidebarFooter/SidebarMenu/SidebarMenuItem/SidebarMenuButton/SidebarRail`), não um sidebar custom

### 11. Frontend — componentes reutilizáveis e padrões de UX

Antes de criar um novo componente ou duplicar código, verificar se já existe um componente reutilizável em `frontend/src/components/` que resolva o caso. Componentes reutilizáveis atuais:

- **`HelpIcon`** (`components/help-icon.tsx`) — ícone de ajuda (`HelpCircle`) com tooltip. Usar sempre que precisar exibir ajuda contextual sobre um parâmetro, métrica ou configuração. **NUNCA** duplicar o padrão `TooltipProvider + Tooltip + TooltipTrigger + HelpCircle + TooltipContent` manualmente — usar `<HelpIcon text="..." />` (ou `<HelpIcon title="..." text="..." />` para tooltip estruturado com título). Props: `text` (obrigatório), `title` (opcional), `side` (default `"top"`), `className` (opcional).
- **`EmptyState`** (`components/empty-state.tsx`) — estado vazio centralizado (ícone + mensagem) dentro de um `CardContent`. Usar quando uma lista/gráfico não tem dados.
- **`CollectingBadge`** (`components/collecting-badge.tsx`) — badge no header top que reflete o estado real da coleta de métricas (Coletando/Aguardando/Reconectando) via SSE + polling. **NÃO** criar badges "Coletando" duplicados nas páginas — o badge global no `Layout.tsx` é a fonte única.
- **`PageHeader`** (`components/page-header.tsx`) — cabeçalho padrão de página (título + descrição + actions). Todas as páginas devem usar.

**Padrão para textos de ajuda (tooltips):** ao escrever textos para o `HelpIcon`, seguir a estrutura: (1) o que é — definição direta, (2) exemplo real com valor concreto ("Ex: 15s significa que..."), (3) trade-off/impacto ("Valores baixos = X, valores altos = Y"). Manter linguagem simples e direta, sem jargão técnico desnecessário.

**Padrão para estados de erro:** páginas que dependem de dados da API devem usar um `Card` centralizado com ícone + título + descrição + botão "Tentar novamente" (ver `Dashboard.tsx` bloco `isError || !data` como referência). **NÃO** usar `Alert variant="destructive"` para erros de carregamento de página — esse padrão é reservado para erros inline em formulários (Login, Onboarding).

**Padrão para header top:** o `Layout.tsx` centraliza no header top: breadcrumb (esquerda) + `CollectingBadge` + dropdown de intervalo + separador + `NavUser` (direita). **NÃO** adicionar badges de status ou componentes de usuário em `PageHeader` das páginas — esses componentes são globais e ficam no header top.
