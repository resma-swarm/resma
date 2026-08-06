# Contribuindo com o RESMA

Obrigado pelo seu interesse em contribuir com o RESMA! Este documento descreve
como configurar seu ambiente de desenvolvimento, os padrões de código que
seguimos e o fluxo para enviar contribuições.

## Pré-requisitos

| Ferramenta    | Versão   | Notas                                              |
|---------------|----------|----------------------------------------------------|
| Docker        | 24+      | Inclui Docker Compose e Docker Swarm               |
| Go            | 1.26     | Para validações rápidas no host (build via Docker) |
| Node.js       | 22       | Para o frontend (pnpm 9+)                          |
| Python        | 3.12     | Para o ML sidecar (opcional — roda em Docker)      |
| pnpm          | 9+       | Gerenciador de pacotes do frontend                 |
| pre-commit    | 4+       | Hooks de pré-commit (`pip install pre-commit`)     |

> **Importante:** O build, test e run do Go acontecem **dentro do container
> `api`** (profile `dev` do docker-compose.yml). O Go instalado no host é
> apenas para validações rápidas (`go vet`, `gofmt`), pois `go-duckdb` requer
> CGO com ABI GNU libstdc++ (Debian).

---

## Setup de Desenvolvimento

### 1. Clone o repositório

```bash
git clone https://github.com/resma-swarm/resma.git
cd resma
```

### 2. Configure as variáveis de ambiente

```bash
cp .env.example .env
# Edite .env conforme necessário (o default funciona para dev)
```

### 3. Instale os hooks de pre-commit

```bash
pip install pre-commit
pre-commit install
```

### 4. Suba o ambiente de desenvolvimento

```bash
# API Go + ML sidecar em modo dev (source montado, hot reload)
docker compose up -d
```

### 5. Inicie o frontend

```bash
cd frontend
pnpm install
pnpm dev
```

### 6. Verifique que tudo está rodando

| Serviço          | URL                      |
|------------------|--------------------------|
| Dashboard (Vite) | http://localhost:5173    |
| API Go           | http://localhost:8080    |
| ML sidecar       | http://localhost:8081    |
| Health check     | http://localhost:8080/health |

---

## Comandos de Desenvolvimento

### API Go (dentro do container `api`)

```bash
# Build
docker compose exec api go build ./...

# Testes
docker compose exec api go test -race -cover -timeout 120s ./...

# Vet
docker compose exec api go vet ./...

# Formatar
docker compose exec api gofmt -l .

# Rodar o servidor
docker compose exec api go run ./cmd/server
```

### ML Sidecar (dentro do container `ml-dev`)

```bash
# Testes
docker compose exec ml-dev pytest tests/ -v

# Lint
docker compose exec ml-dev ruff check app/
docker compose exec ml-dev ruff format --check app/
```

### Frontend (no host)

```bash
cd frontend
pnpm dev      # dev server com hot reload
pnpm build    # build de produção
pnpm lint     # ESLint
```

---

## Padrões de Código

### Go

- **Formatação:** `gofmt` é obrigatório. O CI falha se houver arquivos não formatados.
- **Lint:** Usamos `golangci-lint` com a configuração em `app/api/.golangci.yml`.
- **Imports:** Agrupados em 3 blocos: stdlib, terceiros, locais (`github.com/resma/api`).
- **Erros:** Sempre verificar erros (`if err != nil`). Não usar `_` para ignorar erros.
- **Context:** Handlers devem aceitar `context.Context` para cancelamento graceful.
- **Nomes:** Seguir convenções Go — exported = `PascalCase`, unexported = `camelCase`.

### Python (ML sidecar)

- **Lint:** `ruff` para lint e formatação (configurado no `pre-commit-config.yaml`).
- **Tipos:** Type hints são obrigatórios em funções públicas.
- **Estrutura:** Seguir o padrão FastAPI com routers e serviços separados.

### Frontend

- **Lint:** ESLint + Prettier.
- **Componentes:** Usar shadcn/ui quando possível. Componentes custom em `src/components/`.
- **Hooks:** Custom hooks em `src/hooks/`. Nomear com prefixo `use`.
- **Tipos:** TypeScript estrito. Sem `any` sem justificativa.

### Pre-commit

Os hooks rodam automaticamente a cada commit. Para rodar manualmente:

```bash
pre-commit run --all-files
```

Hooks configurados:
- `go-fmt`, `go-imports`, `go-vet` (Go)
- `ruff`, `ruff-format` (Python)
- `trailing-whitespace`, `end-of-file-fixer`, `check-yaml`, `check-added-large-files`, `detect-private-key` (geral)

---

## Fluxo de Pull Request

### 1. Fork e clone

```bash
# Fork o repositório no GitHub, depois:
git clone https://github.com/SEU_USUARIO/resma.git
cd resma
git remote add upstream https://github.com/resma-swarm/resma.git
```

### 2. Crie uma branch

```bash
git checkout -b feat/minha-feature
```

Use o padrão `tipo/descrição-breve` (ex: `feat/oom-tracking`, `fix/sse-reconnect`,
`docs/install-guide`).

### 3. Desenvolva e commite

Faça commits atômicos seguindo a [convenção de commits](#convenção-de-commits).

### 4. Mantenha sincronizado com upstream

```bash
git fetch upstream
git rebase upstream/main
```

### 5. Rode os testes localmente

```bash
# Go
docker compose exec api go test -race -cover ./...
docker compose exec api go vet ./...
docker compose exec api gofmt -l .

# Frontend
cd frontend && pnpm build

# Pre-commit
pre-commit run --all-files
```

### 6. Abra o Pull Request

- Push para seu fork: `git push origin feat/minha-feature`
- Abra o PR no GitHub apontando para `main`
- Preencha o template de PR
- Aguarde o CI passar e a revisão

---

## Convenção de Commits

Seguimos o padrão [Conventional Commits](https://www.conventionalcommits.org/):

```
tipo(escopo): descrição breve

Corpo opcional com mais detalhes.
```

### Tipos

| Tipo       | Uso                                                        |
|------------|------------------------------------------------------------|
| `feat`     | Nova feature ou funcionalidade                             |
| `fix`      | Correção de bug                                            |
| `docs`     | Mudanças em documentação                                   |
| `refactor` | Refatoração que não muda comportamento                     |
| `test`     | Adição ou correção de testes                               |
| `chore`    | Tarefas de manutenção (deps, configs, CI)                  |
| `perf`     | Melhorias de performance                                   |
| `style`    | Formatação, ponto e vírgula, etc (sem mudança de lógica)   |
| `ci`       | Mudanças no CI/CD                                          |
| `build`    | Mudanças no sistema de build ou dependências               |

### Escopos comuns

`api`, `ml`, `frontend`, `docker`, `auth`, `sse`, `collector`, `db`, `docs`, `ci`

### Exemplos

```
feat(sse): adicionar reconexão automática com backoff exponencial
fix(collector): corrigir cálculo de CPU delta quando container reinicia
docs(install): adicionar seção de troubleshooting do Swarm
refactor(auth): extrair validação de API key para middleware dedicado
test(db): adicionar testes de integridade do schema DuckDB
chore(deps): bump golang-jwt para v5.2.1
```

---

## Estrutura do Projeto

```
resma/
├── app/
│   ├── api/                    # API Go (Go 1.26)
│   │   ├── cmd/
│   │   │   ├── server/         # Entry point do servidor
│   │   │   └── smoke-test/     # Smoke test CLI
│   │   ├── internal/
│   │   │   ├── auth/           # JWT, API key, middleware de auth
│   │   │   ├── collector/      # Coleta de métricas do Docker
│   │   │   ├── config/         # Config via env vars
│   │   │   ├── db/             # DuckDB layer (schema, queries, appender)
│   │   │   ├── docker/         # Docker SDK client (containers, nodes, services, stats, events)
│   │   │   ├── mlclient/       # Cliente HTTP para ML sidecar
│   │   │   ├── scheduler/      # Scheduler de coleta e análise
│   │   │   ├── server/         # HTTP handlers e middleware
│   │   │   └── sse/            # SSE broker e handlers
│   │   ├── Dockerfile          # Multi-stage: dev + runtime
│   │   ├── go.mod
│   │   └── .golangci.yml
│   └── ml/                     # ML sidecar (Python 3.12 + FastAPI)
│       ├── app/
│       ├── tests/
│       ├── Dockerfile
│       └── requirements.txt
├── frontend/                   # Dashboard (React 19 + Vite + TailwindCSS)
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── docs-site/                  # Documentação (Docusaurus)
├── docs/                       # Specs e documentação técnica
├── test-env/                   # Ambiente de testes (services de exemplo)
├── .github/
│   ├── workflows/              # CI/CD (ci.yml, docs.yml, release.yml)
│   ├── ISSUE_TEMPLATE/         # Templates de issue
│   └── PULL_REQUEST_TEMPLATE.md
├── docker-compose.yml          # Profiles dev/prod
├── docker-stack.yml            # Docker Swarm stack
├── install.sh                  # Installer para Docker Swarm
├── .env.example                # Template de variáveis de ambiente
├── .pre-commit-config.yaml     # Hooks de pre-commit
├── LICENSE                     # MIT
├── CONTRIBUTING.md             # Este arquivo
├── CODE_OF_CONDUCT.md          # Código de Conduta
└── README.md                   # README principal
```

---

## Testes

### Go

```bash
# Todos os testes com race detector e coverage
docker compose exec api go test -race -cover -timeout 120s ./...

# Teste de um pacote específico
docker compose exec api go test -v ./internal/sse/...

# Benchmark
docker compose exec api go test -bench=. ./internal/db/...
```

### Python (ML sidecar)

```bash
docker compose exec ml-dev pytest tests/ -v --tb=short
```

### Frontend

```bash
cd frontend
pnpm build    # type-check + build
pnpm lint     # ESLint
```

### Smoke test end-to-end

```bash
# Com a API rodando em dev:
docker compose exec api go run ./cmd/smoke-test
```

---

## Dúvidas?

- Abra uma [issue](https://github.com/resma-swarm/resma/issues) com a label `question`
- Leia a [documentação](docs-site/)
- Consulte o [Código de Conduta](CODE_OF_CONDUCT.md)

Obrigado por contribuir! 🚀
