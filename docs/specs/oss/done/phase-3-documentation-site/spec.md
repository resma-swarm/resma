# Fase 3 — Documentação e Site

> **Prioridade:** Alta  
> **Esforço:** Alto (Docusaurus + escrita de conteúdo)  
> **Bloqueador:** Não  
> **Dependências:** Fase 1 (LICENSE, README), Fase 2 (env vars definidas)

## Objetivo

Criar um site de documentação profissional usando Docusaurus 3, com referência de API auto-gerada via OpenAPI, guias de uso, arquitetura e tutoriais. Publicar via GitHub Pages.

## ⚠️ Princípio fundamental — Identidade visual única

> O site do RESMA deve ter sua **própria identidade visual** — reconhecível e memorável. As referências de outros projetos (Swarmpit, Cetacean, Scribe, Verdaccio, Quickwit, Dyte) servem como **inspiração de padrões e estrutura**, não como modelos para copiar.
>
> - Criar paleta de cores própria, consistente com o tema (resource management, monitoring, Swarm)
> - Não replicar cores, fontes, nomes de seções ou layouts de outros projetos
> - Desenvolver componentes que comuniquem os diferenciais únicos do RESMA (ML, leak detection, OOM tracking)
> - Usar as referências para entender o que funciona bem (UX, estrutura, componentes) e **adaptar** — não replicar
> - Ver [visual-references.md](./visual-references.md) para detalhes

## Decisão: Docusaurus 3 + TailwindCSS + shadcn/ui

Justificativa do Technical Council:
- Stack React (alinha com frontend do RESMA)
- Versioning nativo para múltiplas versões de docs
- Algolia DocSearch gratuito para projetos OSS
- MDX para conteúdo interativo
- Deploy nativo no GitHub Pages
- 64.4k stars, mantido pela Meta
- **TailwindCSS + shadcn/ui** — mesma stack do frontend RESMA, componentes reutilizáveis
- **Template base:** [docusaurus-tailwind-shadcn-template](https://github.com/namnguyenthanhwork/docusaurus-tailwind-shadcn-template) — já integra Tailwind + shadcn + OpenAPI plugin + dark mode

**Referências visuais:** Ver [visual-references.md](./visual-references.md) — análise detalhada de 7 sites visitados com padrões extraídos, wireframe da landing page, paleta de cores e componentes MDX recomendados.

## Tarefas

### 3.1 — Setup Docusaurus

- **Pasta:** `docs-site/` (raiz do repo)
- **Comando:** `npx create-docusaurus@latest docs-site classic --typescript`
- **Node:** >= 20
- **Configurar:** `docusaurus.config.ts` com:
  - Title: "RESMA"
  - Tagline: "RESource MAnager for Docker Swarm"
  - URL: `https://USER.github.io` (ou domínio próprio)
  - baseUrl: `/resma/`
  - Theme: dark mode default
  - Navbar com: Docs, API Reference, GitHub
  - Footer com: links, copyright, licença MIT

### 3.2 — Homepage

- **Arquivo:** `docs-site/src/pages/index.tsx`
- **Seções:**
  - Hero section: título, tagline, CTA "Get Started" + "GitHub"
  - Features grid: 6 cards (Metrics, ML Recommendations, Leak Detection, Dashboard, API, Open Source)
  - Quick install snippet (1 comando)
  - Screenshot do dashboard

### 3.3 — docs/introduction.md

- O que é RESMA
- Caso de uso principal
- Arquitetura em alto nível (diagrama Mermaid)
- Stack tecnológica

### 3.4 — docs/installation.md

- 3 métodos de instalação:
  1. **Docker Swarm (recomendado):** `docker stack deploy -c docker-stack.yml resma`
  2. **Docker Compose (dev):** `docker compose --profile prod up --build`
  3. **Desenvolvimento local:** `docker compose --profile dev up -d` + `docker compose --profile dev exec go-dev go run ./cmd/server` + `pnpm dev` (em `frontend/`)
- Requisitos para cada método
- First boot: criar usuário admin via UI onboarding
- Verificação: `curl http://localhost:8080/health`

### 3.5 — docs/configuration.md

- Todas as env vars com prefixo `RESMA_` (referenciar `.env.example`)
- Tabela: variável, tipo, default, descrição
- Seções: Database, Collection, ML/Analysis, Auth, Security, Docker
- Exemplos de configuração para diferentes cenários (small cluster, large cluster, dev)

### 3.6 — docs/api-reference.md (OpenAPI integration)

- Instalar `docusaurus-plugin-openapi-docs` (PaloAlto Networks)
- Apontar para `swagger.json` gerado por `swaggo/swag` a partir das anotações nos handlers Go `/api/v1/*`
- **Apenas API pública `/api/v1/*` é documentada** — endpoints internos `/api/*` (dashboard, sparklines, auth, config, templates CRUD, schedules CRUD) não entram no OpenAPI (ver [phase-0b spec — API Surface Architecture](../phase-0b-go-migration/spec.md#api-surface-architecture--split-públicointerno))
- Configurar para auto-gerar páginas de referência por endpoint com try-it-out, exemplos e schemas
- Endpoints agrupados por tag: Services, Recommendations, Nodes, Storage, OOM, Change Log
- **Pipeline:** anotações swaggo nos handlers Go → `swag init` (gera `swagger.json` + `docs.go`) → commit `swagger.json` no repo → Docusaurus consome no build
- **Auth na docs:** documentar API key auth (`Authorization: Bearer resma_key_...` ou `X-API-Key`) com exemplo de key de teste (sandbox)

### 3.7 — docs/architecture.md

- Diagrama de componentes (Mermaid):
  - Docker Swarm → moby/moby SDK (Go) → Collector (goroutines) → DuckDB
  - DuckDB → Recommender (Python ML sidecar via HTTP) → Go API → Frontend (React)
  - Scheduler (goroutine) → Docker API → Apply recommendations
  - **API split:** `/api/v1/*` (público, API key) vs `/api/*` (interno, JWT) vs `/api/sse/*` (streaming)
- Fluxo de métricas: coleta → armazenamento → análise → recomendação
- ML pipeline: outlier removal → percentile → regression → recommendation
- Detecção de memory leaks: regressão linear + R² threshold

### 3.8 — docs/guides/

- `getting-started.md` — primeiro deploy, criar usuário, ver dashboard
- `recommendations.md` — como interpretar recomendações de CPU/memória
- `memory-leaks.md` — como detectar e confirmar memory leaks
- `apply-limits.md` — como aplicar recomendações manualmente
- `schedules.md` — como agendar mudanças de recursos
- `templates.md` — como usar templates de configuração YAML

### 3.9 — docs/contributing/

- `development-setup.md` — setup detalhado (Docker dev container Go, Node, pnpm)
- `code-style.md` — `gofmt`/`goimports`, `golangci-lint`, conventional commits, branch naming
- `adding-handlers.md` — como adicionar novos endpoints (decidir público `/api/v1/*` vs interno `/api/*`, anotações swaggo se público)
- `testing.md` — como escrever e rodar testes Go (`go test ./...`) + pytest ML sidecar
- `documentation.md` — como atualizar docs e o site Docusaurus

### 3.10 — Deploy GitHub Pages

- **Arquivo:** `.github/workflows/docs.yml`
- Build Docusaurus no push para main
- Deploy para `gh-pages` branch
- Workflow:
  ```yaml
  name: Deploy Docs
  on:
    push:
      branches: [main]
      paths: ['docs-site/**']
  jobs:
    deploy:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - uses: actions/setup-node@v4
          with:
            node-version: '20'
        - run: cd docs-site && npm install && npm run build
        - uses: peaceiris/actions-gh-pages@v4
          with:
            github_token: ${{ secrets.GITHUB_TOKEN }}
            publish_dir: docs-site/build
  ```

### 3.11 — Metadados OpenAPI via swaggo (Go)

- **Arquivo:** `app/api/cmd/server/main.go` — declarar metadados gerais via comentários swaggo
- **Arquivo:** `app/api/internal/handlers/*.go` — anotações swaggo em cada handler `/api/v1/*`
- `swag init` gera `docs/docs.go` + `docs/swagger.json` + `docs/swagger.yaml`
- **Apenas endpoints públicos `/api/v1/*` são documentados** — endpoints internos `/api/*` (dashboard, sparklines, auth, config, templates CRUD, schedules CRUD) não entram no OpenAPI (ver [phase-0b spec — API Surface Architecture](../phase-0b-go-migration/spec.md#api-surface-architecture--split-públicointerno))
- Metadados gerais (comentário no main.go):
  ```go
  // @title RESMA API
  // @version 1.0
  // @description RESource MAnager for Docker Swarm
  // @contact.name RESMA
  // @contact.url https://github.com/resma-swarm/resma
  // @license.name MIT
  // @license.url https://opensource.org/licenses/MIT
  // @host localhost:8080
  // @BasePath /api/v1
  // @securityDefinitions.apikey ApiKeyAuth
  // @in header
  // @name Authorization
  ```
- Tags por grupo (apenas públicos): `services`, `recommendations`, `nodes`, `storage`, `oom`, `change_log`

### 3.12 — Anotações swaggo nos handlers públicos

- **Arquivos:** `app/api/internal/handlers/*.go` (apenas handlers `/api/v1/*`)
- Adicionar em cada endpoint:
  - `@Summary` — título curto (1 linha)
  - `@Description` — descrição detalhada
  - `@Tags` — grupo
  - `@Param` — parâmetros (path, query, body)
  - `@Success` — código de sucesso + schema
  - `@Failure` — códigos de erro + schema
  - `@Router` — path (relativo a `/api/v1`)
  - `@Security ApiKeyAuth` — todos os endpoints públicos exigem API key
- Exemplos de request/response via `@Example`

### 3.13 — docs/api-keys.md (novo)

- Como criar API key via UI (`/api/auth/api-keys` CRUD interno)
- Scopes: `read` (GET em `/api/v1/*`), `write` (mutações em `/api/v1/*` — v1.1+)
- Como usar: header `Authorization: Bearer resma_key_...` ou `X-API-Key`
- Rate limits por key
- Revogação e rotação
- Exemplos em curl, Python (requests), Go (net/http), JavaScript (fetch)

## Critérios de aceite

- [ ] `docs-site/` com Docusaurus 3 funcional (`npm run start` funciona)
- [ ] Homepage com hero, features e CTA
- [ ] Páginas: introduction, installation, configuration, architecture, api-keys
- [ ] API reference auto-gerada via `docusaurus-plugin-openapi-docs` consumindo `swagger.json` do swaggo
- [ ] **OpenAPI cobre apenas `/api/v1/*`** (endpoints públicos com API key auth)
- [ ] Guias: getting-started, recommendations, memory-leaks, apply-limits, schedules, templates
- [ ] Seção contributing com development-setup, code-style, adding-handlers, testing, documentation
- [ ] GitHub Actions deploy para GitHub Pages
- [ ] `swag init` gera `swagger.json` válido a partir das anotações nos handlers Go
- [ ] Todos os handlers `/api/v1/*` com anotações swaggo completas
- [ ] Build do Docusaurus sem warnings

## Estrutura de pastas resultante

```
docs-site/
├── docusaurus.config.ts
├── sidebars.ts
├── package.json
├── src/
│   ├── pages/
│   │   └── index.tsx          ← homepage
│   ├── components/
│   └── css/
│       └── custom.css
├── docs/
│   ├── introduction.md
│   ├── installation.md
│   ├── configuration.md
│   ├── architecture.md
│   ├── api-reference.md       ← auto-gerado (swagger.json do swaggo)
│   ├── api-keys.md            ← guia de API keys públicas
│   ├── guides/
│   │   ├── getting-started.md
│   │   ├── recommendations.md
│   │   ├── memory-leaks.md
│   │   ├── apply-limits.md
│   │   ├── schedules.md
│   │   └── templates.md
│   └── contributing/
│       ├── development-setup.md
│       ├── code-style.md
│       ├── adding-handlers.md
│       ├── testing.md
│       └── documentation.md
├── static/
│   └── img/
│       └── dashboard.png      ← screenshots
└── build/                     ← output (gitignored)
```
