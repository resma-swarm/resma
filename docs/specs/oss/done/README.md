# RESMA Open-Source Specs

> Especificações para transformar o RESMA em um projeto open-source no GitHub com documentação, API pública, site e instalação fácil em Docker Swarm.

## Estrutura

```
docs/specs/oss/
├── README.md              ← este índice
├── ROADMAP.md             ← timeline consolidada
├── competitive-analysis.md ← RESMA vs Swarmpit/Portainer/Cetacean
├── phase-0-monorepo-organization/
│   └── spec.md            ← pré-requisito: reorganização do repo
├── phase-0b-go-migration/
│   └── spec.md            ← pré-requisito: migração API Go + SSE + ML sidecar
├── phase-1-legal-community/
│   └── spec.md            ← spec principal da fase
├── phase-2-security-hardening/
│   └── spec.md
├── phase-3-documentation-site/
│   ├── spec.md
│   └── visual-references.md
├── phase-4-docker-swarm-install/
│   └── spec.md
├── phase-5-cicd-quality/
│   └── spec.md
└── phase-6-benchmarking/
    └── spec.md
```

Cada pasta de fase pode conter múltiplos documentos (specs detalhadas, rascunhos, referências, etc.) além do `spec.md` principal.

| Documento | Fase | Descrição |
|-----------|------|-----------|
| [ROADMAP.md](./ROADMAP.md) | — | Timeline consolidada, milestones e dependências |
| [competitive-analysis.md](./competitive-analysis.md) | — | Análise competitiva vs Swarmpit, Portainer, Cetacean |
| [phase-0-monorepo-organization/](./phase-0-monorepo-organization/) | **0** | **Pré-requisito:** reorganização do monorepo (estrutura de pastas, CI path filtering) |
| [phase-0b-go-migration/](./phase-0b-go-migration/) | **0b** | **Pré-requisito:** migração da API para Go + SSE + Python ML sidecar (evita retrabalho em código Python) |
| [phase-1-legal-community/](./phase-1-legal-community/) | 1 | Fundações legais e comunidade (LICENSE, README, CONTRIBUTING, etc.) |
| [phase-2-security-hardening/](./phase-2-security-hardening/) | 2 | Segurança e hardening (secrets, CORS, env vars, headers) |
| [phase-3-documentation-site/](./phase-3-documentation-site/) | 3 | Documentação e site (Docusaurus + Tailwind + shadcn/ui, OpenAPI, guias, referências visuais) |
| [phase-4-docker-swarm-install/](./phase-4-docker-swarm-install/) | 4 | Instalação fácil Docker Swarm (stack deploy, install.sh, GHCR) |
| [phase-5-cicd-quality/](./phase-5-cicd-quality/) | 5 | CI/CD e qualidade (GitHub Actions, ruff, pytest, pre-commit) |
| [phase-6-benchmarking/](./phase-6-benchmarking/) | 6 | Benchmarking Docker Swarm (scripts, métricas, documentação) |

## Como usar este documento

1. Comece lendo o [ROADMAP.md](./ROADMAP.md) para visão geral
2. **Fase 0 é pré-requisito absoluto** — estrutura de pastas primeiro
3. **Fase 0b é pré-requisito para Fases 2-6** — migra API para Go antes de investir em security/docs/install/CI no Python
4. **Fase 1** (legais) pode rodar em paralelo com Fase 0b — não toca no código
5. Fases 2-6 dependem de 0+0b e têm dependências indicadas entre si

## Decisões do Technical Council

| Decisão | Escolha | Justificativa |
|---------|---------|---------------|
| Estrutura do repo | Monorepo único (app + test-env + docs-site) | Docs atômicas com features, onboarding trivial, padrão OSS (Verdaccio, Quickwit, Jest) |
| Backend language | Go (API) + Python (ML sidecar) | Go resolve gargalos de concorrência/SSE/Docker SDK; Python mantém ecossistema ML (sklearn/scipy) |
| Real-time | SSE (Server-Sent Events) | Go suporta 10x mais conexões que Python; SSE é mais simples que WebSocket para dashboards |
| DuckDB | `marcboeker/go-duckdb` (Go) + `duckdb` (Python) | Mesma engine C, arquivo `.duckdb` compatível entre linguagens |
| Licença | MIT | Máxima permissividade, atrai contribuidores |
| Docs site | Docusaurus 3 | Alinha com stack React, versioning nativo, Algolia DocSearch |
| API surface | **Split público/interno** — `/api/v1/*` (público, API key, OpenAPI) + `/api/*` (interno/UI, JWT) | Evita que endpoints UI-shaped (dashboard, sparklines) e operacionais (auth, config) viriem contrato público estável. Um binário, dois middlewares. Ver [phase-0b spec](./phase-0b-go-migration/spec.md#api-surface-architecture--split-públicointerno) |
| API docs | OpenAPI via `swaggo/swag` (Go) + `docusaurus-plugin-openapi-docs` | Auto-gerado das anotações nos handlers `/api/v1/*`; Swagger UI servido em `/swagger/*` |
| Install | `docker stack deploy` + `install.sh` one-liner | Simples, nativo Swarm |
| CI/CD | GitHub Actions + ruff + pytest | Padrão OSS, sem custo |
| Registry | GitHub Container Registry (ghcr.io) | Integrado ao GitHub, gratuito para OSS |
| Benchmark | Scripts shell nativos (sysbench, stress-ng, docker stats) | Sem dependências externas pesadas |

## Referências competitivas

- **[Swarmpit](https://github.com/swarmpit/swarmpit)** — referência principal: installer via `docker run`, stack deploy, REST API com Swagger, MCP server, Traefik labels, ARM support, roadmap público
- **[Portainer](https://www.portainer.io/)** — referência enterprise: resource limits UI, Prometheus + Grafana integration, alert rules, API programável
- **[Cetacean](https://github.com/Radiergummi/cetacean)** — referência lightweight: single binary, SSE real-time, topology views, log viewer, pluggable auth, zero config

Ver [competitive-analysis.md](./competitive-analysis.md) para comparativo completo de features e recomendações.
