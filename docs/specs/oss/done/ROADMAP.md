# ROADMAP — RESMA Open-Source

> Timeline consolidada para transformar RESMA em projeto open-source no GitHub.

## Visão geral

| Fase | Tarefas | Prioridade | Esforço | Dependências |
|------|---------|------------|---------|--------------|
| **0 — Monorepo** | **5** | **Crítica** | **Médio** | **Nenhuma** |
| **0b — Go API + SSE** | **12** | **Crítica** | **Alto** | **Fase 0** |
| 1 — Legais/Comunidade | 10 | Crítica | Baixo | Fase 0 |
| 2 — Segurança | 9 | Crítica | Médio | Fase 0, 0b |
| 3 — Docs/Site | 13 | Alta | Alto | Fase 0, 1, 2 |
| 4 — Install Swarm | 7 | Alta | Médio | Fase 0, 0b, 2 |
| 5 — CI/CD | 8 | Alta | Médio | Fase 0, 0b, 2, 4 |
| 6 — Benchmark | 5 | Média | Médio | Fase 0, 0b, 4 |
| 7 — Multi-Node Agent | 7 | Alta | Alto | Fase 0b, 4 |
| **Total** | **76** | — | ~5-6 semanas | — |

## Milestones

### Milestone —1 — Reorganização do monorepo (pré-Sprint 1)

**Objetivo:** Estrutura de pastas consolidada, CI com path filtering, repo pronto para receber as demais fases.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 0.1 Limpeza de artefatos | 0 | 30 min |
| 0.2 Criar estrutura de pastas base | 0 | 30 min |
| 0.3 Documentar estrutura no README | 0 | 30 min |
| 0.4 Preparar CI com path filtering | 0 | 1h |
| 0.5 Validar docker-compose | 0 | 30 min |

**Entrega:** Repo estruturado, CI básico funcionando, pronto para Fase 0b+1+2.

### Milestone 0 — Migração Go API + SSE (Sprint 1)

**Objetivo:** API em Go com SSE, ML em Python sidecar, DuckDB compartilhado. Base para todas as fases subsequentes — evita retrabalho em código Python que seria migrado.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 0b.1 Setup Go + estrutura base | 0b | 2h |
| 0b.2 Migrar DuckDB layer | 0b | 4h |
| 0b.3 Migrar Docker client | 0b | 4h |
| 0b.4 Migrar Auth + API Key model | 0b | 3h |
| 0b.5 Migrar Routers (11 routers, split público/interno) | 0b | 7h |
| 0b.6 Migrar Collector + Scheduler | 0b | 3h |
| 0b.7 Implementar SSE broker | 0b | 4h |
| 0b.8 Criar Python ML sidecar | 0b | 2h |
| 0b.9 Integrar Go ↔ Python ML | 0b | 1h |
| 0b.10 Atualizar Dockerfile + compose | 0b | 2h |
| 0b.11 Atualizar frontend para SSE | 0b | 5h |
| 0b.12 Testes de equivalência | 0b | 3h |

**Entrega:** API Go funcional com SSE, ML sidecar Python, 2 containers no Swarm, frontend atualizado.

### Milestone 1 — Repo público pronto (Sprint 2)

**Objetivo:** Tornar o repo seguro para exposição pública.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 1.1 LICENSE | 1 | 5 min |
| 1.2 README.md raiz | 1 | 1h |
| 1.3 CONTRIBUTING.md | 1 | 30 min |
| 1.4 CODE_OF_CONDUCT.md | 1 | 5 min |
| 1.5 SECURITY.md | 1 | 15 min |
| 1.6 CHANGELOG.md | 1 | 15 min |
| 1.7 CODEOWNERS | 1 | 5 min |
| 1.8 PR template | 1 | 10 min |
| 1.9 Issue templates | 1 | 20 min |
| 2.1 Remover JWT secret | 2 | 15 min |
| 2.2 .env.example | 2 | 20 min |
| 2.3 Restringir CORS | 2 | 15 min |
| 2.4 Validar JWT secret | 2 | 15 min |
| 2.6 Parametrizar admin | 2 | 30 min |
| 2.7 Headers de segurança | 2 | 15 min |
| 2.8 API key security | 2 | 1h |
| 2.9 ML sidecar security | 2 | 30 min |

**Entrega:** Repo pode ser tornado público sem risco.

### Milestone 2 — Instalável (Sprint 3)

**Objetivo:** Qualquer pessoa pode instalar RESMA com 1 comando.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 4.1 docker-stack.yml | 4 | 30 min |
| 4.2 install.sh | 4 | 1h |
| 4.3 Swarm secrets | 4 | 30 min |
| 4.4 Healthcheck | 4 | 15 min |
| 4.5 docker-compose.override | 4 | 15 min |
| 4.6 README quickstart | 4 | 20 min |
| 4.7 GHCR publish | 4 | 30 min |
| 2.5 Swarm secrets config | 2 | 30 min |

**Entrega:** `curl ... | bash` instala RESMA em qualquer Swarm.

### Milestone 3 — CI/CD ativo (Sprint 3)

**Objetivo:** Pipeline automatizado funcionando.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 5.1 ci.yml (Go test + lint + Python ML test) | 5 | 30 min |
| 5.4 golangci-lint config | 5 | 15 min |
| 5.5 Go test suite + pytest ML | 5 | 2h |
| 5.6 pre-commit | 5 | 15 min |
| 5.7 dependabot | 5 | 10 min |
| 5.8 .dockerignore | 5 | 10 min |
| 5.3 release.yml | 5 | 30 min |

**Entrega:** Todo PR passa por CI, releases são automatizados.

### Milestone 4 — Documentação pública (Sprint 4)

**Objetivo:** Site de docs profissional no ar.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 3.1 Setup Docusaurus | 3 | 1h |
| 3.2 Homepage | 3 | 1h |
| 3.3 introduction.md | 3 | 30 min |
| 3.4 installation.md | 3 | 30 min |
| 3.5 configuration.md | 3 | 30 min |
| 3.6 API reference | 3 | 1h |
| 3.7 architecture.md | 3 | 1h |
| 3.8 Guias (6) | 3 | 3h |
| 3.9 Contributing docs | 3 | 1h |
| 3.10 GitHub Pages deploy | 3 | 20 min |
| 3.11 OpenAPI metadata (swaggo) | 3 | 30 min |
| 3.12 swaggo annotations handlers públicos | 3 | 2h |
| 3.13 docs/api-keys.md | 3 | 30 min |
| 5.2 docs.yml workflow | 5 | 20 min |

**Entrega:** `https://USER.github.io/resma` no ar com docs completas.

### Milestone 5 — Benchmarking (Sprint 4)

**Objetivo:** Ferramentas de benchmark disponíveis.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 6.1 swarm-benchmark.sh | 6 | 2h |
| 6.2 resma-overhead.sh | 6 | 1h |
| 6.3 stack-benchmark.yml | 6 | 30 min |
| 6.4 Docs benchmarking | 6 | 1h |
| 6.5 swarm-hpa integration | 6 | 30 min |

**Entrega:** Scripts de benchmark funcionais e documentados.

### Milestone 6 — Multi-Node Agent (Sprint 5)

**Objetivo:** Coleta de stats de containers em todos os nodes do Swarm (manager + workers), não apenas do manager.

| Tarefa | Fase | Esforço |
|--------|------|---------|
| 7.1 Agent binary (com buffer + retry + gzip) | 7 | 7h |
| 7.2 Ingestion API (com rate limiting + task queries) | 7 | 5h |
| 7.3 Collector refactor (híbrido + task lifecycle + slot via TaskList) | 7 | 5h |
| 7.4 Docker + compose | 7 | 3h |
| 7.5 Endpoints (agents + task health + tasks + SSE) | 7 | 4h |
| 7.6 Testes multi-node (com task lifecycle + slot stability) | 7 | 5h |
| 7.7 Documentação (com task monitoring guide) | 7 | 2h |
| 7.8 Frontend (Tasks page + coluna Agent em Nodes + card em NodeDetail + Service Health) | 7 | 17h |

**Entrega:** Agent deployado como global service em todos os nodes, coleta de stats cluster-wide funcionando, ML recommender analisa dados de todos os nodes.

## Timeline visual

```
Pré-Sprint 1        Sprint 1 (Semana 1)          Sprint 2 (Semana 2)          Sprint 3 (Semana 3)        Sprint 4 (Semana 4)
┌──────────────┐    ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐
│ Milestone -1 │    │ Milestone 0      │    │ Milestone 1      │    │ Milestone 2      │    │ Milestone 4      │
│ Reorganização│───→│ Migração Go + SSE │───→│ Legais + Segurança│───→│ Install + CI/CD  │───→│ Documentação     │
│              │    │                  │    │                  │    │                  │    │                  │
│ Fase 0       │    │ Fase 0b (12 tasks│    │ Fase 1 (10 tasks)│    │ Fase 4 (7 tasks) │    │ Fase 3 (13 tasks)│
│ (5 tasks)    │    │ + Fase 1 (paralelo│    │ Fase 2 (9 tasks) │    │ Fase 5 (8 tasks) │    │                  │
└──────────────┘    └──────────────────┘    └──────────────────┘    └──────────────────┘    └──────────────────┘
                                                                                     │
                                                                                     ▼
                                                                            ┌──────────────────┐
                                                                            │ Milestone 5      │
                                                                            │ Benchmarking     │
                                                                            │ Fase 6 (5 tasks) │
                                                                            └────────┬─────────┘
                                                                                     │
                                                                                     ▼
                                                                            ┌──────────────────┐
                                                                            │ Milestone 6      │
                                                                            │ Multi-Node Agent │
                                                                            │ Fase 7 (7 tasks) │
                                                                            └──────────────────┘
```

## Dependências entre fases

```
Fase 0 (Monorepo) ──→ Fase 0b (Go API + SSE) ──→ Fase 2 (Segurança) ──→ Fase 4 (Install) ──→ Fase 3 (Docs)
                          │                        │                      │
                          │                        │                      ├──→ Fase 5 (CI/CD)
                          │                        │                      │
                          │                        │                      └──→ Fase 6 (Benchmark) ──→ Fase 7 (Multi-Node Agent)
                          │                        │
                          └──→ Fase 1 (Legais) ──────────────────────────→ Fase 3 (Docs)
```

- **Fase 0** é pré-requisito absoluto — estrutura de pastas primeiro
- **Fase 0b** é pré-requisito para Fases 2, 3, 4, 5, 6 — evita retrabalho em código Python que seria migrado
- **Fase 1** (legais) pode rodar em paralelo com Fase 0b — não toca no código
- **Fase 2** depende de 0b (security hardening no Go, não no Python)
- **Fase 3** depende de 0 (estrutura), 1 (LICENSE/README) e 2 (env vars)
- **Fase 4** depende de 0b (2 containers: Go API + Python ML) e 2 (secrets)
- **Fase 5** depende de 0b (Go test + lint) e 4 (Dockerfile multi-stage)
- **Fase 6** depende de 0b (Go API para benchmarkar) e 4 (docker-stack.yml)

## Riscos e mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Secrets no git history | Alta | Crítico | `git filter-repo` antes de tornar público |
| Breaking changes para usuários atuais | Média | Médio | Documentar migration guide |
| Docusaurus build complexity | Baixa | Baixo | Começar com template classic |
| Docker Hub rate limits | Baixa | Baixo | Usar GHCR (ilimitado para OSS) |
| Testes iniciais insuficientes | Média | Médio | Começar com testes mínimos, expandir gradualmente |

## Próximos passos após conclusão

1. **v0.1.0 release** — primeiro release público com GitHub Release + tag
2. **Comunidade** — anunciar em fóruns (Reddit r/docker, r/selfhosted, HN)
3. **Docker Hub** — espelhar imagem para Docker Hub além do GHCR
4. **Roadmap público** — criar GitHub Projects board com features futuras
5. **Integrações** — explorar integração com Prometheus, Grafana, Portainer

## Roadmap futuro (pós-open-source)

Features inspiradas na análise competitiva com Swarmpit, Portainer e Cetacean:

| Feature | Inspiração | Prioridade | Sprint alvo |
|---------|-----------|------------|-------------|
| SSE real-time no frontend | Cetacean | **Concluído na Fase 0b** | — |
| Log viewer com live-streaming + search | Cetacean, Swarmpit | Alta | v0.2.0 |
| Topology views (logical + physical) | Cetacean | Média | v0.3.0 |
| Traefik integration com labels | Swarmpit | Média | v0.2.0 |
| Installer interativo via `docker run` | Swarmpit | Média | v0.3.0 |
| MCP server para LLM integration | Swarmpit | Média | v0.3.0 |
| Prometheus + Grafana stack opcional | Portainer | Média | v0.3.0 |
| Alert rules (node down, quorum, OOM, leak) | Portainer | Alta | v0.2.0 |
| Pluggable auth (OIDC, Tailscale, mTLS) | Cetacean | Baixa | v0.4.0 |
| Private registry integration | Swarmpit | Baixa | v0.4.0 |
| Autoredeploy (detectar novas imagens) | Swarmpit | Baixa | v0.4.0 |
| Stack management (CRUD de stacks) | Swarmpit, Portainer | Baixa | v0.5.0 |
| Service CRUD (deploy, scale, rollback) | Swarmpit, Portainer | Baixa | v0.5.0 |

Ver [competitive-analysis.md](./competitive-analysis.md) para detalhes de cada referência.
