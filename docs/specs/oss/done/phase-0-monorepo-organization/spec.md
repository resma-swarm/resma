# Fase 0 — Reorganização do Monorepo

> **Prioridade:** Crítica  
> **Esforço:** Médio (reestruturação + ajustes de paths + CI)  
> **Bloqueador:** Sim — pré-requisito para todas as outras fases  
> **Dependências:** Nenhuma

## Objetivo

Reorganizar a estrutura do repositório RESMA em um monorepo bem definido, preparando a estrutura de pastas para receber o site de documentação (Docusaurus), mantendo o app (API Go + ML sidecar Python + frontend) e o test-env no mesmo repositório.

Esta fase é **pré-requisito para todas as demais fases** — a estrutura de pastas deve estar consolidada antes de qualquer trabalho de documentação, instalação, CI/CD ou benchmarking.

## Decisão: Monorepo único

> **RESMA será um monorepo único** contendo app, test-env e docs-site.

### Justificativa

1. **Docs mudam a cada feature** — PR atômico altera código + docs no mesmo commit. Separar docs cria o problema de "docs desatualizadas". Como a OpsMill documentou: "if you or an AI agent fixes a behavior and doesn't touch the docs in the same PR, the docs are already wrong by the time you merge."

2. **test-env sincroniza com a API do app** — se o endpoint de métricas muda, o test-env precisa ajustar no mesmo PR. Separar cria coordenação desnecessária para um time pequeno.

3. **Onboarding trivial** — uma clone, tudo disponível. Contribuidor novo tem acesso a app, ambiente de teste e docs em um único repo.

4. **CI unificado com path filtering** — GitHub Actions pode rodar apenas jobs relevantes por pasta, evitando builds desnecessários.

5. **Padrão OSS confirmado** — Verdaccio (pnpm monorepo, 50+ packages, docs Docusaurus dentro do repo), Quickwit, React Native, Jest, Cetacean — todos mantêm docs no mesmo repo. Exceções (Portainer, Dyte) são produtos comerciais ou projetos muito grandes.

6. **Splitting é mais fácil que merging** — se test-env crescer e receber muitas contribuições independentes, pode ser extraído depois. Merging repos depois é muito mais difícil.

### O que pegar das referências

- **Verdaccio:** monorepo com pnpm workspaces, docs Docusaurus dentro do repo, deploy via Netlify
- **Quickwit:** repo único com site Docusaurus, architecture diagram, "open and free" section
- **Docusaurus multi-instance:** suporta múltiplas instâncias do plugin-content-docs, permitindo docs de diferentes fontes no mesmo site

### O que NÃO fazer

- Não criar repo separado para docs-site
- Não criar repo separado para test-env (a menos que cresça significativamente)
- Não usar submodules — adicionam complexidade sem benefício para este caso

## Estrutura de pastas proposta

```
resma/
├── app/                        # Aplicação (pós-Fase 0b)
│   ├── api/                    # API Go (net/http + DuckDB + Docker SDK + SSE)
│   │   ├── cmd/server/main.go
│   │   ├── internal/           # config, db, docker, handlers, collector, scheduler, auth, sse, ml
│   │   ├── docs/               # swagger.json + docs.go (gerados por swag init)
│   │   ├── Dockerfile          # Multi-stage: builder/dev/runtime
│   │   └── go.mod
│   └── ml/                     # ML sidecar Python (sklearn/scipy/numpy)
│       ├── main.py
│       ├── requirements.txt
│       ├── tests/
│       └── Dockerfile
├── backend/                    # Python legacy (referência para portar — NÃO recebe alterações após 0b)
│   ├── core/
│   ├── routers/
│   ├── services/
│   ├── models.py
│   ├── config.py
│   └── run.py
├── frontend/                   # App React (Vite + TailwindCSS + shadcn/ui)
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── vite.config.ts
├── test-env/                   # Ambiente de simulação para devs
│   ├── docker-compose.yml
│   ├── docker-stack.yml
│   ├── dotnet-api/
│   ├── drift-service/
│   ├── leaky-service/
│   ├── load-generator/
│   ├── node-api/
│   ├── oom-prone-service/
│   ├── python-api/
│   ├── scenario-simulator/
│   ├── web-frontend/
│   └── README.md
├── docs-site/                  # Docusaurus (site + docs) — Fase 3
│   ├── docs/
│   ├── src/
│   ├── static/
│   ├── docusaurus.config.ts
│   ├── sidebars.ts
│   └── package.json
├── docs/                       # Specs e documentação de design
│   └── specs/
│       └── oss/                # Specs open-source
├── .github/
│   └── workflows/              # CI/CD (Fase 5)
│       ├── ci.yml              # Testes Go API + ML sidecar + frontend
│       ├── docs.yml            # Build + deploy docs-site
│       └── release.yml         # Release + publish 2 imagens (resma-api + resma-ml)
├── docker-compose.yml          # Dev + prod (profiles, usa app/api/Dockerfile)
├── docker-stack.yml            # Produção Swarm (2 serviços) — Fase 4
├── .gitignore
├── .dockerignore
├── LICENSE                     # Fase 1
├── README.md                   # Fase 1
├── CONTRIBUTING.md             # Fase 1
├── CODE_OF_CONDUCT.md          # Fase 1
├── SECURITY.md                 # Fase 1
├── CHANGELOG.md                # Fase 1
└── CODEOWNERS                  # Fase 1
```

### Status atual vs. proposta

| Pasta | Status atual | Ação necessária |
|-------|-------------|-----------------|
| `app/api/` | ✅ Já existe (scaffold 0b.1) | Manter — Go API |
| `app/ml/` | ❌ Não existe | Criar na Fase 0b (tarefa 0b.8) |
| `backend/` | ✅ Já existe (Python legacy) | Manter como referência para portar — NÃO recebe alterações após 0b |
| `frontend/` | ✅ Já existe | Manter |
| `test-env/` | ✅ Já existe | Manter |
| `docs-site/` | ❌ Não existe | Criar na Fase 3 |
| `docs/` | ✅ Já existe (specs) | Manter |
| `.github/workflows/` | ❌ Não existe | Criar na Fase 5 |
| `docker-compose.yml` | ✅ Já existe (pós-0b.1) | Manter (profiles dev/prod) |
| `app/api/Dockerfile` | ✅ Já existe (pós-0b.1) | Manter (multi-stage) |
| `templates/` | ✅ Já existe | Avaliar se permanece ou move para `app/api/templates/` após migração Go |
| `data/` | ✅ Já existe | Manter (DuckDB), adicionar ao `.gitignore` se não estiver |
| `resma.egg-info/` | ✅ Existe | Remover, adicionar ao `.gitignore` |

### Decisões sobre pastas existentes

- **`templates/`** — avaliar se é usado pelo backend (legacy). Se sim, mover para `app/api/templates/` após migração Go. Se não, remover.
- **`data/`** — diretório de dados do DuckDB. Deve estar no `.gitignore` (dados locais, não versionados).
- **`resma.egg-info/`** — artefato de build Python. Remover e adicionar ao `.gitignore`.
- **`.ai/`** e **`.devin/`** — frameworks de AI. Manter, já no `.gitignore` ou adicionar.
- **`backend/`** — Python legacy. Mantido como referência para portar durante a Fase 0b. Remover do repo apenas após 0b.12 (testes de equivalência) confirmar paridade total.

## Tarefas

### 0.1 — Limpeza de artefatos

- [ ] Remover `resma.egg-info/` do repo
- [ ] Adicionar `resma.egg-info/` ao `.gitignore`
- [ ] Verificar se `data/` está no `.gitignore` (DuckDB local)
- [ ] Verificar se `.ai/` e `.devin/` estão no `.gitignore`
- [ ] Avaliar `templates/` — mover para `app/api/templates/` se usado pela API Go (após migração)

### 0.2 — Criar estrutura de pastas base

- [ ] Criar pasta `.github/workflows/` (vazia, preenchida na Fase 5)
- [ ] Criar `docker-compose.dev.yml` com override para dev (usa test-env)
- [ ] Garantir que `docs/` está estruturado corretamente para specs

### 0.3 — Documentar estrutura no README

- [ ] Adicionar seção "Estrutura do projeto" no README.md raiz (Fase 1) com a árvore de pastas
- [ ] Documentar que `docs-site/` será criado na Fase 3
- [ ] Documentar que `.github/workflows/` será criado na Fase 5

### 0.4 — Preparar CI com path filtering

- [ ] Criar `.github/workflows/ci.yml` com jobs condicionais por path:
  - `app/api/**` → Go test + lint (golangci-lint) + swag-check
  - `app/ml/**` → pytest (Python ML sidecar)
  - `frontend/**` → build + lint frontend
  - `test-env/**` → validação docker-compose
  - `docs-site/**` → build Docusaurus (quando existir)
  - `docs/**` → sem CI (apenas docs)
  - `backend/**` → sem CI (legacy, não recebe alterações)
- [ ] Configurar `paths` filter no workflow

### 0.5 — Validar docker-compose após reorganização

- [ ] Garantir que `docker-compose.yml` funciona após mudanças de paths
- [ ] Garantir que `docker-compose.dev.yml` sobe test-env junto
- [ ] Testar `docker compose up` e `docker compose -f docker-compose.yml -f docker-compose.dev.yml up`

## Critérios de aceitação

- [ ] Estrutura de pastas segue o layout proposto
- [ ] Artefatos de build (`resma.egg-info/`) removidos e no `.gitignore`
- [ ] `docker-compose.yml` e `docker-compose.dev.yml` funcionais
- [ ] CI com path filtering configurado
- [ ] README documenta a estrutura do projeto
- [ ] Repo pronto para receber Fase 1 (LICENSE, CONTRIBUTING, etc.)

## Riscos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Quebrar docker-compose ao mover pastas | Baixa | Médio | Testar após cada mudança |
| Quebrar imports Go ao mover templates | Baixa | Baixo | Buscar referências antes de mover |
| CI muito complexo para time pequeno | Baixa | Baixo | Começar simples, expandir gradualmente |

## Referências

- [Monorepo vs Multi-repo — Masst Docs](https://docs.masst.dev/sd/fundamentals/monorepo)
- [Monorepos vs Polyrepos — Vercel Academy](https://vercel.com/academy/production-monorepos/monorepos-vs-polyrepos)
- [Verdaccio Monorepo Structure](https://deepwiki.com/verdaccio/verdaccio/2-monorepo-structure)
- [Verdaccio Documentation Website](https://deepwiki.com/verdaccio/verdaccio/11.3-documentation-website)
- [One Docs Site, Many Repos — OpsMill](https://opsmill.com/blog/one-docs-site-many-repos/)
- [Docusaurus Multi-instance](https://docusaurus.io/docs/docs-multi-instance)
- [Docusaurus GitHub Pages Deploy](https://docusaurus.io/docs/next/deployment/github-pages)
