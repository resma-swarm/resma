# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2026-08-06

### Added
- **Installer upgrade mode** — `MODE=upgrade` faz pull das novas imagens e
  atualiza os services do Swarm in-place via `docker service update --image`,
  preservando dados (DuckDB) e secrets. Suporta `RESMA_VERSION` para
  especificar versão (default: latest)

## [0.2.0] - 2026-08-06

Major release: Right-Sizing Studio (apply com rollback automático), RBAC com
3 roles, gestão de usuários e API keys, Settings area, installer container
com uninstall e upgrade in-place, redesign da landing page e dezenas de fixes
de UX.

### Added
- **Right-Sizing Studio** — nova página `/optimizations` (ex-`/recommendations`)
  com HeroMetric, Explainability, LayerToggle, simulação what-if, apply com
  rollback automático, simulação em lote e modo Template (combobox de templates
  YAML com preview e apply)
- **Bulk rollback** — apply em lote com seleção, countdown com progress bar,
  timestamps relativos, paginação, ARIA labels, row expansível (antes/depois +
  critérios + timeline), busca por serviço
- **RBAC** — 3 roles (owner/admin/user) com RequireRole middleware protegendo
  todos os write endpoints
- **User management UI** (CRUD) com onboarding flow criando 'owner'
- **API Keys management UI** com One-Time Read (OTR) — plaintext mostrado apenas
  uma vez na criação
- **Settings area** com nested routes (users, api-keys, parameters, data)
- **Two-tier config** — env var (infra) + DB operacional via `/api/settings`
- **Data retention expansion** — task_history, volume_metrics, storage_summary,
  change_log
- **Stale-marking** para services e nodes (soft delete, status='stale')
- **Data prune endpoints** com dry-run e audit logging
- **shadcn official Sidebar** migration (NavUser, NavSettings, NavMain)
- **Profile page** com change password
- **StaleServiceDays** config (RESMA_STALE_SERVICE_DAYS, default 7)
- **CleanupExpiredRefreshTokens** in retention loop
- **Installer container** no padrão SwarmPit — install + uninstall + upgrade
  mode (in-place, preserva dados e secrets), logo ASCII, intervals de produção,
  modo não-interativo
- **Frontend estático em produção** (SPA) servido pelo Go API
- **ML payload estendido** com tiers, risk, explainability, histograms
- **Landing page redesign** (uiux T1-T11) — hero split com terminal mockup,
  bento grid de features, dashboard mockup, code example com syntax highlight,
  tabela comparativa RESMA vs Portainer vs Swarmpit, CTA com social proof,
  scroll animations, SEO meta tags e OG image
- **Status 'observing' pós-apply** — distinção de OOMs antes/depois do apply
- **Scheduler** registra change_log + ícone de schedule nos cards

### Changed
- Onboarding agora cria 'owner' em vez de 'admin'
- Layout migrado de custom para shadcn Sidebar (SidebarProvider + SidebarInset)
- RunRetention expandido de 3 para 7 tabelas
- Rota `/studio` renomeada para `/optimizations` (alinha URL com o nome da página)
- Storage + agendamento migrados de `/recommendations` para `/optimizations`
- Intervalo de coleta 1s → 5s em todos os ambientes (swarm, standalone, hpa-demo)
- Removidos limits do compose para api/ml/agent — Studio é o único owner dos
  limits (evita `docker stack deploy` sobrescrever)
- dropdown-menu, tabs, avatar e select substituídos pelas versões oficiais shadcn
- Removida integração PostHog (R7) — OSS não pode depender de serviço pago
- Removido endpoint export-yaml da API (apply em lote com rollback é mais seguro)

### Fixed
- Studio: slider max em modo template usa capacidade real do cluster Swarm
  (busca cluster_capacity do /api/dashboard em vez de hardcoded)
- Studio: accordion "Ver YAML" reposicionado acima da seção Recursos com design
  consistente (uppercase muted, sem chevron default)
- Studio: template mode mantém sliders + WhatIfPanel visíveis (YAML do template
  alimenta os sliders via parse)
- Studio: combobox no topo, sliders visíveis, preview do YAML em accordion
- ScheduleDialog: resetar estado ao fechar (evita valores residuais)
- Sheet do Studio: fechar ao confirmar agendamento + invalidar schedules pending
- Alerts: timestamp vazia em leaks/drifts — usar time.Now() como ts de detecção
- Layout: header fixo com scroll apenas no body (h-svh + overflow-hidden)
- API: corrigir import e null unwrap em ExportYamlButton e RollbackWatches
- Server: usar s.cfg.WebDir em vez de cfg (fora de escopo)
- dockerignore: excluir frontend/pnpm-workspace.yaml do build context
- 6 achados do QA Playwright corrigidos

### Security
- RequireRole middleware protege todos os write endpoints (schedules, templates,
  services, recommendations apply, api-keys, users, settings, prune)
- API keys plaintext mostrado apenas uma vez na criação (OTR)
- Owner role é único (não pode ser criado via UI, apenas via onboarding)
- Owner não pode ser deletado nem ter role alterado
- Usuários não podem deletar a si mesmos

## [0.1.0] - 2026-07-27

Initial public release.

- Go API (net/http + DuckDB + Docker SDK) com auth JWT + API key com scopes
- ML sidecar Python (scikit-learn + scipy) acessando dados via HTTP do Go API
- Frontend React 19 + Vite 6 + TailwindCSS 4 + shadcn/ui
- Multi-node Agent (Go binary leve, global service no Swarm)
- Coleta de métricas CPU/memória, detecção de memory leaks, OOM tracking
- SSE (Server-Sent Events) para streaming em tempo real
- Docker Swarm deploy via installer container
- Pipeline de release CI/CD (multi-arch amd64+arm64, Docker Hub + GHCR)
