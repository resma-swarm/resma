# RESMA — Próximos Passos (Pós-Fase 7)

> **Status:** As Fases 0 a 7 (specs em `done/`) foram implementadas. Este documento lista os ajustes pendentes identificados na auditoria de fechamento, validada pelo Technical Council.
>
> **Quadro geral:** 4 fases completas (0, 0b, 2, 4), 4 parciais (1, 3, 5, 7), 1 não implementada (6).

---

## Ajustes pendentes — visão macro

### Prioridade Alta — Segurança e Funcionalidade

**Fase 7 — Resiliência do Agent**
- Compressão gzip no payload de métricas
- Rate limiting no server (tamanho de payload, frequência por node)
- Proteção contra replay (validação de timestamp skew)

**Fase 5 → Fase 7 — CI do Agent**
- Incluir build e push da imagem do agent no CI e release workflow
- Adicionar testes do agent no pipeline

**Fase 7 — Dashboard**
- Card "Service Health" no Dashboard (agregado desired vs running)

### Prioridade Média — Completude

**Fase 7 — Frontend**
- Pie chart de distribuição de estados na Tasks page
- Restart history chart na tab Tasks do ServiceDetail
- Ajustar configurações do agent para bater com a spec (heartbeat, buffer, backoff, stale threshold, task poll interval)

**Fase 7 — Documentação**
- Integrar `docs/multi-node.md` e `docs/task-monitoring.md` no site Docusaurus

**Fase 1 — Arquivos legais faltantes**
- `CHANGELOG.md` na raiz (formato Keep a Changelog)
- `.github/CODEOWNERS`
- Converter ISSUE_TEMPLATE para formato GitHub Issue Forms (.yml)

**Fase 3 — Conteúdo do Docusaurus**
- Seção `guides/` (getting-started, recommendations, memory-leaks, apply-limits, schedules, templates)
- Seção `contributing/` (development-setup, code-style, adding-handlers, testing, documentation)
- Páginas `architecture.md` e `api-keys.md` separadas
- Plugin OpenAPI para auto-gerar API reference a partir do swagger.json

**Fase 5 — Testes**
- Ampliar suite de testes Go (auth, handlers, db, docker)
- Criar suite de testes ML (pytest com conftest, test_health, test_analyze, test_forecast)
- Adicionar job `swag-check` no CI
- Completar `.dockerignore`

### Prioridade Baixa — Nice-to-have

**Fase 6 — Benchmarking de Swarm**
- Implementar scripts de benchmark Swarm vs bare metal (`swarm-benchmark.sh`, `resma-overhead.sh`, `stack-benchmark.yml`)
- Documentar baseline de referência no Docusaurus
- Guia de integração com swarm-hpa

**Fase 1 — Comunidade**
- `.github/FUNDING.yml` (opcional)
- Tabela comparativa com Swarmpit/Portainer no README
- Roadmap público no README

**Fase 4 — Deploy**
- `docker-compose.override.yml` para dev
- Suporte ARM completo (multi-arch manifest no docker-stack)

**Fase 0 — Documentação**
- Seção "Estrutura do projeto" no README
- Path filtering explícito no CI

---

## Gaps identificados pós-auditoria (validados em revisão com o usuário)

### Auth & Usuários — RBAC completo (Prioridade Alta)
- Tela de CRUD de usuários no frontend (listar, criar, editar, deletar)
- Implementar 3 perfis: owner (único, primeiro usuário via onboarding), admin (mesmo acesso do owner), user (somente leitura — não agenda nem aplica recomendações)
- Middleware `RequireRole` no backend para proteger rotas de escrita (schedules, apply templates, restart services, CRUD usuários)
- Fluxo de onboarding no frontend: se 0 usuários, criar owner; depois owner cadastra admin/user
- Endpoint de gestão de usuários (`GET/POST/PATCH/DELETE /api/auth/users`)
- Frontend: esconder/mostrar ações conforme role do usuário logado

### API Keys — Tela de gestão no frontend (Prioridade Alta)
- Página "API Keys" no frontend para criar, listar, revogar e editar keys
- Mostrar plaintext uma única vez na criação com aviso de copiar antes de fechar
- Exibir prefixo, nome, scopes, last_used, created_at, revoked_at na listagem

### Área de Configuração — UI (Prioridade Alta)
- Menu "Configurações" com ícone de engrenagem no rodapé do menu lateral, acima do user account nav, destacado visualmente dos itens operacionais acima
- Página de configuração com subnavegação interna para cada seção (ex: Usuários, API Keys, Settings gerais)
- Estrutura preparada para crescer (futuro: webhooks, integrações, etc.) sem inflar o menu principal

### UserNav + Sidebar — migrar para padrão shadcn oficial (Prioridade Alta)
- **Regra de ouro:** NUNCA alterar os componentes originais do shadcn em `frontend/src/components/ui/` — usar sempre como estão, compondo em volta deles
- Adicionar o componente `sidebar.tsx` do shadcn (não existe no projeto hoje — o Layout é custom, o que viola o padrão de usar sempre shadcn)
- Migrar o `Layout.tsx` para usar `Sidebar/SidebarHeader/SidebarContent/SidebarFooter/SidebarMenu/SidebarMenuItem/SidebarMenuButton/SidebarRail` do shadcn
- Criar componente `NavUser` seguindo o `nav-user.tsx` oficial do shadcn (sidebar-07): Avatar + nome/email em grid + `ChevronsUpDown`, dropdown com `side="right"` + `align="end"` para abrir para o lado
- Itens do dropdown do usuário: Perfil, Mudar senha, Configurações (link para área de config), Sair
- Criar página de Perfil (`/profile`) onde o usuário visualiza seus dados e altera a senha
- Referência oficial: `github.com/shadcn-ui/ui` — `apps/v4/registry/new-york-v4/blocks/sidebar-07/components/nav-user.tsx` (código completo já coletado)
- PR #9321 do shadcn-ui: `DropdownMenuLabel` deve ser envolvido em `DropdownMenuGroup` para evitar erro runtime "MenuGroupRootContext is missing" (Base UI)
- O ícone de engrenagem para Configurações deve ficar no `SidebarFooter` acima do `NavUser`, conforme padrão shadcn

### Settings gerais — two-tier config (Prioridade Alta)
- Mover parâmetros operacionais (intervalos de coleta, thresholds, janela de análise) de env vars para a tabela `app_settings` (key-value, já existe no DuckDB)
- Env vars passam a ser apenas o default na primeira inicialização; depois o valor no DB prevalece
- Sub-rota "Parâmetros gerais" na área de configuração para ajustar via UI sem reiniciar o container
- Env vars permanecem para configs de infra (porta, JWT secret, DB path, CORS, agent token) — estas não mudam sem restart
- Padrão two-tier validado: Grafana e Datadog usam o mesmo modelo (env var = infra, DB/UI = operacional)

### Data Retention — expansão do retention para todas as tabelas (Prioridade Alta)
- **Status atual:** `RunRetention` (queries.go) cobre apenas `metrics`, `oom_events`, `node_metrics` com `RESMA_RETENTION_DAYS` (default 30d). Roda ao iniciar + a cada 24h
- **Gap:** 17 tabelas sem retention — acumulam indefinidamente: `task_history`, `task_errors`, `change_log`, `volume_metrics`, `storage_summary`, `service_registry`, `container_node_map`, `agents`, `tasks`, `nodes`, `cluster`, `schedules`, `templates`, `service_configs`, `api_keys`, `refresh_tokens`, `app_settings`
- **Tabelas que NÃO devem ter retention** (dados de configuração/estado, não time-series): `app_settings`, `templates`, `api_keys`, `users`, `service_configs`, `schedules`
- **Tabelas que DEVEM ter retention por idade** (ts-based, mesma lógica de `metrics`):
  - `task_history` — manter por `RESMA_RETENTION_DAYS` (eventos de lifecycle de tasks)
  - `task_errors` — manter por `RESMA_RETENTION_DAYS`
  - `change_log` — manter por `RESMA_RETENTION_DAYS` (auditoria de mudanças aplicadas)
  - `volume_metrics` — manter por `RESMA_RETENTION_DAYS`
  - `storage_summary` — manter por `RESMA_RETENTION_DAYS`
  - `refresh_tokens` — expirar tokens com `expires_at < now()` (não ts-based, mas cleanup)
- **Tabelas com stale-marking** (não deletar, marcar como stale):
  - `service_registry` — marcar `status='stale'` quando service sem métricas há **>7 dias** (threshold `RESMA_STALE_SERVICE_DAYS`, default 7). Swarm faz scale up/down; service pode voltar. Não deletar — apenas marcar
  - `agents` — já tem lógica de stale via heartbeat (manter como está)
  - `nodes` — marcar `status='stale'` quando node não reporta há **>7 dias** (mesmo threshold)
  - `container_node_map` — cleanup de mapeamentos de containers que não existem mais (prune via Docker API diff)
- **Stale threshold:** `RESMA_STALE_SERVICE_DAYS` (default 7) — separado de `RESMA_RETENTION_DAYS` (30) porque stale é sobre visibilidade operacional, retention é sobre armazenamento. Netdata usa 24h-7d para nodes offline; Prometheus ignora tasks com `desired_state != running`
- **Referências:** Prometheus `--storage.tsdb.retention.time`; TimescaleDB `add_retention_policy`; Netdata `cleanup ephemeral hosts after`; InfluxDB retention enforcement service

### Data Prune — UI de limpeza manual (Prioridade Alta)
- **Sub-rota "Dados" na área de configuração** (aba junto a Usuários, API Keys, Parâmetros gerais)
- **Botão "Prune" por categoria** — owner/admin clica e remove dados sob demanda, sem esperar o retention automático:
  - **Services stale** — remove services marcados como `stale` em `service_registry` + suas métricas órfãs em `metrics` + `oom_events` + `service_configs`. Confirm dialog com lista de services afetados
  - **Nodes stale** — remove nodes marcados como `stale` em `nodes` + `node_metrics` órfãs + `container_node_map`. Confirm dialog
  - **Tasks órfãs** — remove tasks em `tasks`/`task_history`/`task_errors` cujo service não existe mais no Swarm. Confirm dialog
  - **Métricas antigas** — remove manualmente métricas mais velhas que N dias (input de dias, default = `RESMA_RETENTION_DAYS`). Útil para limpeza agressiva pontual
  - **Change log** — remove entradas de `change_log` mais velhas que N dias
  - **Volume metrics** — remove `volume_metrics` e `storage_summary` mais velhos que N dias
- **Endpoints:** `POST /api/prune/services-stale`, `POST /api/prune/nodes-stale`, `POST /api/prune/tasks-orphan`, `POST /api/prune/metrics?days=N`, `POST /api/prune/change-log?days=N`, `POST /api/prune/volumes?days=N` — todos requerem role owner/admin
- **Audit:** toda operação de prune registra em `change_log` (quem, o quê, quantas rows, quando)
- **Feedback:** toast com resumo pós-prune ("Removido: 3 services stale, 12.450 rows de metrics, 8 oom_events")
- **Proteção:** confirm dialog duplo para prune de services/nodes (ação irreversível); dry-run opcional (`?dry-run=true` retorna contagem sem deletar)
- **Referências:** Elasticsearch ILM delete phase; Kusto retention policy soft-delete; Docker system prune (mesmo padrão de confirm + dry-run)

### Outros gaps
- Role no JWT é cosmético hoje — nenhum middleware checa role (validar após implementar RBAC)
- Frontend não usa o role retornado por `/api/auth/me` para nada

### SSE — bug crítico: real-time completamente quebrado (Prioridade Alta)
- **Descoberta validada pelo Technical Council:** NENHUM tópico SSE tem publisher E consumer ao mesmo tempo. O SSE está conectado mas silencioso — `sseConnected=true` engana o usuário enquanto o polling de 5min é o que realmente atualiza as telas
- **Causa raiz:** o Collector não recebe referência ao SSE handler em `collector.New()` — ele só escreve no DB, nunca notifica o frontend
- **Tópicos com consumer mas SEM publisher (bug):** `dashboard`, `services`, `nodes`, `tasks` — Dashboard/Services/Nodes/Tasks assinam mas nada chega
- **Tópicos com publisher mas SEM consumer (desperdício):** `metrics`, `events`, `agents` — agents publicam mas nenhuma página assina
- **Correção:** injetar SSE handler no Collector e publicar em `collectLoop` (metrics), `syncServiceRegistry` (services), `nodeCollectLoop` (nodes), `taskCollectLoop` (tasks), `clusterCollectLoop` (dashboard); fazer frontend assinar `metrics`, `events`, `agents` nas páginas apropriadas
- **Páginas de detalhe sem SSE:** NodeDetail, ServiceDetail, ContainerDetail usam apenas polling 30s — deveriam assinar `nodes`/`services`/`metrics` respectivamente

### SSE — dados não exibidos mas disponíveis no backend (Prioridade Média)
- Métricas coletadas mas não exibidas: `mem_working_set`, `cpu_throttled_periods`, `cpu_throttled_time` (críticas para memory leaks e CPU contention)
- `task_history` (endpoint `/api/tasks/{service}/history`) não tem UI — deveria ter restart history chart no ServiceDetail
- OOM events não têm tela dedicada — só aparecem como contagem no Dashboard
- Storage trend e volume growth têm endpoints mas nenhuma tela consome
- `node_id`, `task_id`, `slot` coletados (Fase 7) mas não exibidos em detalhes de container/service

### SSE — benchmark de referência (Grafana Live, Datadog)
- Grafana Live usa WebSocket Pub/Sub por canal; SSE é adequado para 80% dos casos de dashboard (1-30s de latência)
- Padrão consolidado: push (SSE/WebSocket) para dados que mudam em <30s; polling para dados que mudam em >30s ou raramente
- RESMA já usa SSE (não WebSocket) — correto para o caso de uso (one-directional, server→client)
- Estratégia híbrida do frontend (SSE invalida React Query → refetch) é o padrão correto, só precisa dos publishers no backend

---

## Sobreposições já resolvidas (não requerem ação)

- `backend/` Python removido pela Fase 0b (correto — migração concluída)
- `docker-stack.yml` expandido pela Fase 7 com `resma-agent` (evolução natural)
- Collector refactorado pela Fase 7 para modo híbrido (aditivo, sem quebrar contratos)
- Endpoint `/api/agent/nodes` → `/api/agents` (mudança de contrato aceitável)
- Endpoint `/api/internal/services/{service}/restarts` integrado em `/api/services/health` (aceitável)

---

## Referências

- Specs originais (Fases 0-7): `done/`
- Auditoria completa: validada por 3 subagents + Technical Council
- Documentação multi-node: `docs/multi-node.md`, `docs/task-monitoring.md`
- Benchmark existente (Go vs Python): `scripts/benchmark.sh`, `docs/benchmarks.md`
