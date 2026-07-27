# Fase 5 — CI/CD e Qualidade

> **Prioridade:** Alta  
> **Esforço:** Médio (GitHub Actions + tooling Go + Python ML)  
> **Bloqueador:** Não  
> **Dependências:** Fase 2 (env vars), Fase 4 (Dockerfiles com healthcheck)

## Objetivo

Estabelecer pipeline de CI/CD completo no GitHub Actions: lint, testes, build Docker, push para GHCR (2 imagens), deploy de docs e releases automatizados com semantic versioning. CI cobre **Go API** + **Python ML sidecar** + **frontend React**.

## Tarefas

### 5.1 — .github/workflows/ci.yml

- **Trigger:** push para main, pull request
- **Jobs:**
  1. **go-lint** — `golangci-lint run ./...` (em `app/api/`)
  2. **go-test** — `go test -race -cover ./...` (em `app/api/`, CGO_ENABLED=1 — DuckDB exige)
  3. **go-vet** — `go vet ./...`
  4. **swag-check** — `swag init -g cmd/server/main.go` e verifica se `docs/swagger.json` está atualizado (diff contra commit)
  5. **ml-test** — `pip install -r app/ml/requirements.txt && pytest app/ml/tests/`
  6. **frontend-build** — `cd frontend && pnpm install && pnpm build`
  7. **docker-build** — `docker build -t resma-api:ci -f app/api/Dockerfile --target runtime .` + `docker build -t resma-ml:ci -f app/ml/Dockerfile .` (valida que ambos Dockerfiles buildam)
  8. **push** — só em push para main: push 2 imagens para GHCR
- **Matrix:** Go 1.26 (única versão suportada), Python 3.12 (ML sidecar), Node 22 (frontend)
- **Cache:** `actions/setup-go` com cache de módulos, `actions/setup-python` com cache de pip, `docker/build-push-action` com cache

### 5.2 — .github/workflows/docs.yml

- **Trigger:** push para main com paths `docs-site/**` ou `app/api/docs/swagger.json`
- **Jobs:**
  1. Build Docusaurus: `cd docs-site && npm install && npm run build`
  2. Deploy para GitHub Pages: `peaceiris/actions-gh-pages@v4`
- **Configurar:** GitHub Pages settings → Source: GitHub Actions

### 5.3 — .github/workflows/release.yml

- **Trigger:** push de tag `v*.*.*`
- **Jobs:**
  1. Build Docker multi-arch (amd64 + arm64) para **2 imagens**:
     - `docker.io/resmaswarm/resma-api:latest`, `:vX.Y.Z`, `:vX.Y`, `:vX`
     - `docker.io/resmaswarm/resma-ml:latest`, `:vX.Y.Z`, `:vX.Y`, `:vX`
  2. Criar GitHub Release com notes auto-geradas
  3. Atualizar CHANGELOG.md (opcional — via semantic-release)

### 5.4 — Adicionar golangci-lint como linter

- **Arquivo:** `app/api/.golangci.yml`
- **Config:**
  ```yaml
  run:
    timeout: 5m
    go: "1.26"
  linters:
    enable:
      - errcheck
      - gosimple
      - govet
      - ineffassign
      - staticcheck
      - unused
      - gofmt
      - goimports
      - misspell
      - revive
      - gocritic
  linters-settings:
    goimports:
      local-prefixes: github.com/resma/api
  ```
- **Comandos:** `golangci-lint run ./...` (lint), `gofmt -w .` (format)

### 5.5 — Suite de testes Go + pytest ML

- **Pasta Go:** `app/api/internal/*/` (testes ao lado do código, padrão Go)
- **Estrutura:**
  ```
  app/api/
  ├── internal/
  │   ├── config/config_test.go       ← env vars, defaults, _FILE support
  │   ├── db/db_test.go               ← DuckDB open/ping/close, schema init
  │   ├── docker/docker_test.go       ← client init, health (mock daemon)
  │   ├── auth/auth_test.go           ← JWT issue/verify, bcrypt, API key gen/verify
  │   ├── handlers/handlers_test.go   ← HTTP handlers com httptest
  │   └── sse/sse_test.go             ← SSE broker, fan-out, disconnect
  ```
- **Mínimo inicial:**
  - `config_test.go`: carrega de env vars, defaults corretos, `_FILE` lê de arquivo
  - `auth_test.go`: JWT issue/verify/expire, bcrypt hash/compare, API key gen com prefixo `resma_key_`, scopes validados
  - `handlers_test.go`: `/health` 200, `/ready` 200/degraded, `/api/v1/services` 401 sem key, 200 com key
  - `db_test.go`: DuckDB open, ping, schema init (CREATE TABLE IF NOT EXISTS)
- **Fixtures:** `httptest.NewServer` para mock Docker daemon, temp dir para DuckDB test
- **Pasta ML:** `app/ml/tests/`
  ```
  app/ml/tests/
  ├── conftest.py          ← fixtures compartilhadas (TestClient FastAPI)
  ├── test_health.py       ← GET /health
  ├── test_analyze.py      ← POST /analyze com métricas mock
  └── test_forecast.py     ← POST /forecast
  ```

### 5.6 — Pre-commit hooks

- **Arquivo:** `.pre-commit-config.yaml` (raiz)
- **Hooks:**
  ```yaml
  repos:
    - repo: https://github.com/dnephin/pre-commit-golang
      rev: v0.5.1
      hooks:
        - id: go-fmt
        - id: go-imports
        - id: go-vet
        - id: go-build
    - repo: https://github.com/astral-sh/ruff-pre-commit
      rev: v0.6.0
      hooks:
        - id: ruff
          args: [--fix]
          files: ^app/ml/
        - id: ruff-format
          files: ^app/ml/
    - repo: https://github.com/pre-commit/pre-commit-hooks
      rev: v4.6.0
      hooks:
        - id: trailing-whitespace
        - id: end-of-file-fixer
        - id: check-yaml
        - id: check-added-large-files
        - id: check-merge-conflict
  ```
- **Instrução no CONTRIBUTING.md:** `pip install pre-commit && pre-commit install`

### 5.7 — Dependabot

- **Arquivo:** `.github/dependabot.yml`
- **Config:**
  ```yaml
  version: 2
  updates:
    - package-ecosystem: "gomod"
      directory: "/app/api"
      schedule:
        interval: "weekly"
    - package-ecosystem: "pip"
      directory: "/app/ml"
      schedule:
        interval: "weekly"
    - package-ecosystem: "npm"
      directory: "/frontend"
      schedule:
        interval: "weekly"
    - package-ecosystem: "npm"
      directory: "/docs-site"
      schedule:
        interval: "weekly"
    - package-ecosystem: "docker"
      directory: "/"
      schedule:
        interval: "weekly"
    - package-ecosystem: "github-actions"
      directory: "/"
      schedule:
        interval: "weekly"
  ```

### 5.8 — Melhorar .dockerignore

- **Arquivo:** `.dockerignore` (raiz)
- **Conteúdo:**
  ```
  .git/
  .gitignore
  .ai/
  .devin/
  .github/
  .env
  .env.example
  .pre-commit-config.yaml
  docs/
  docs-site/
  test-env/
  *.md
  !README.md
  data/
  backend/
  resma.egg-info/
  __pycache__/
  *.pyc
  .venv/
  node_modules/
  frontend/node_modules/
  app/api/bin/
  ```

## Critérios de aceite

- [ ] CI roda em todo PR: go-lint + go-test + go-vet + swag-check + ml-test + frontend-build + docker-build
- [ ] CI faz push de 2 imagens (resma-api + resma-ml) para GHCR em push para main
- [ ] Docs deploy automático para GitHub Pages
- [ ] Release workflow cria GitHub Release em tag push (2 imagens multi-arch)
- [ ] `golangci-lint run ./...` passa sem erros
- [ ] `go test ./...` passa com no mínimo testes de config, auth, handlers, db
- [ ] `pytest app/ml/tests/` passa com testes de health, analyze, forecast
- [ ] `swag init` gera `swagger.json` consistente com anotações (swag-check job passa)
- [ ] `pre-commit install` funciona
- [ ] Dependabot configurado para gomod, pip, npm, docker, github-actions
- [ ] `.dockerignore` exclui arquivos não necessários no build

## Estrutura de arquivos resultante

```
raiz/
├── .github/
│   ├── workflows/
│   │   ├── ci.yml
│   │   ├── docs.yml
│   │   └── release.yml
│   ├── dependabot.yml
│   ├── CODEOWNERS
│   ├── pull_request_template.md
│   └── ISSUE_TEMPLATE/
├── app/api/
│   ├── .golangci.yml
│   └── internal/*/..._test.go
├── app/ml/tests/
│   ├── conftest.py
│   ├── test_health.py
│   ├── test_analyze.py
│   └── test_forecast.py
├── .pre-commit-config.yaml
└── .dockerignore           ← melhorado
```
