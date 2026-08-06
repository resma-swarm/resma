# Prompt de Implementação — RESMA

> Copie todo o conteúdo abaixo deste bloco e envie em um novo chat que tenha acesso à pasta `.ai/` (AI Engineering Framework).

---

## Missão

Usando a **AI Engineering Framework** (ler `.ai/AGENTS.md` como bootstrap), implementar o app **RESMA — RESource MAnager** conforme especificação técnica completa em `resma/docs/TECH_SPEC.md`.

## Contexto

O RESMA é um gerenciador de recursos para Docker Swarm que coleta métricas de CPU/memória via Docker API, armazena em DuckDB embedded, analisa com percentis + ML simples (3 técnicas) e recomenda limits/reservations. Roda como 1 container no manager node com acesso ao Docker socket. Tem autenticação própria com onboarding no primeiro acesso (similar ao Portainer CE).

## Especificação Técnica de Referência

**Ler obrigatoriamente:** `resma/docs/TECH_SPEC.md` (documento completo com 17 seções)

### Resumo da Stack

**Backend (Python):**
- Python 3.12+ / FastAPI 0.138.0+ (`app.frontend()` para servir SPA)
- aiodocker (async Docker API client)
- DuckDB (embedded OLAP, Appender API, retention 30 dias)
- scikit-learn + scipy + numpy (ML: Z-score, Regressão Linear, KMeans)
- bcrypt + PyJWT (autenticação)
- pydantic v2 / pyyaml

**Frontend (React):**
- React 19 + Vite
- **TailwindCSS v4** (última versão, utility-first)
- **shadcn/ui original** (última versão, sem forks — usar `npx shadcn@latest init` e `npx shadcn@latest add` para cada componente)
- Recharts ou Tremor (gráficos)
- TanStack Query (data fetching)
- React Router (roteamento + protected routes)
- **pnpm** como gerenciador de pacotes (NÃO usar npm/yarn)

**Infra:**
- Dockerfile multi-stage (Node + Python)
- docker-compose.yml para Swarm
- 1 container no manager node
- Mount `/var/run/docker.sock`

### Estrutura do Projeto

```
resma/
├── backend/
│   ├── main.py              # FastAPI app + lifecycle + app.frontend()
│   ├── collector.py          # Async Docker API metrics collector
│   ├── docker_client.py      # Wrapper aiodocker
│   ├── recommender.py        # Percentis + ML simples
│   ├── templates.py          # Load/apply templates YAML
│   ├── db.py                 # DuckDB connection + schema + retention
│   ├── models.py             # Pydantic models (incluindo auth)
│   ├── config.py             # Settings (env vars)
│   ├── auth.py               # Onboarding, login, JWT, bcrypt, middleware
│   └── deps.py               # Dependencies (get_current_user, get_db)
├── frontend/
│   ├── src/
│   │   ├── App.tsx
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Recommendations.tsx
│   │   │   ├── Templates.tsx
│   │   │   ├── Services.tsx
│   │   │   ├── Login.tsx
│   │   │   └── Onboarding.tsx
│   │   ├── components/
│   │   │   ├── ProtectedRoute.tsx
│   │   │   └── AuthGate.tsx
│   │   ├── contexts/
│   │   │   └── AuthContext.tsx
│   │   └── api/
│   ├── package.json
│   ├── pnpm-lock.yaml
│   └── vite.config.ts
├── templates/
│   ├── small.yml
│   ├── medium.yml
│   ├── large.yml
│   ├── database.yml
│   ├── worker-heavy.yml
│   └── ml.yml
├── Dockerfile
├── docker-compose.yml
├── pyproject.toml
└── docs/
    ├── TECH_SPEC.md
    └── TECH_SPEC.html
```

### Requisitos Críticos de UI/UX

1. **TailwindCSS v4** — instalar e configurar na última versão
2. **shadcn/ui original** — usar o CLI oficial (`npx shadcn@latest`), sem forks ou alternativas
3. **pnpm** — gerenciador de pacotes em todo o frontend (instalação, scripts, Dockerfile)
4. UI moderna e responsiva com tema dark (alinhado ao padrão)
5. Telas de auth (Login + Onboarding) devem seguir o estilo do Portainer CE (centradas, limpas, card-based)
6. Dashboard com gráficos de consumo (área/linha), cards de KPI, tabela de serviços
7. Estados de loading, erro e empty em todas as páginas

### Requisitos Críticos de Autenticação

1. **Onboarding no primeiro acesso** — frontend detecta via `GET /api/auth/status` se há usuário cadastrado
2. Se não há usuário → tela de Onboarding (criar username + senha, mín. 8 chars)
3. Senha armazenada como **hash bcrypt** (cost 12) no DuckDB
4. **JWT** com access token (15min) + refresh token (7 dias)
5. Middleware FastAPI: todos os endpoints `/api/*` exigem Bearer token (exceto `/api/auth/*`)
6. Frontend: `AuthGate` bloqueia acesso se não autenticado, `ProtectedRoute` em todas as rotas
7. `AuthContext` gerencia tokens em localStorage, refresh automático, logout
8. Rate limiting no login (máx 5 tentativas/min por IP)

### Requisitos Críticos de Backend

1. Coleta async a cada 15s via `aiodocker` (`/containers/{id}/stats`)
2. Listener de `/events` para OOM kills
3. DuckDB Appender API para batch insert
4. Retention job diário (delete > 30 dias)
5. ML on-demand: Z-score (outliers) → Percentis P50/P95 → Regressão Linear (leak) → KMeans (padrão)
6. Margem de segurança adaptativa (1.2x a 2.3x conforme padrão + OOM + leak)
7. Templates YAML: small, medium, large, database, worker-heavy, ml
8. Aplicação de limits via Docker API (`update service`)
9. Confidence score baseado em amostras e R²

### Variáveis de Ambiente

```
RESMA_DB_PATH=/data/resma.duckdb
RESMA_COLLECT_INTERVAL=15
RESMA_RETENTION_DAYS=30
RESMA_OUTLIER_THRESHOLD=3.0
RESMA_LEAK_R2_THRESHOLD=0.7
RESMA_LEAK_DAILY_MB_THRESHOLD=10
RESMA_ANALYSIS_WINDOW_DAYS=7
RESMA_JWT_SECRET=<definir em produção>
RESMA_JWT_ACCESS_TTL_MINUTES=15
RESMA_JWT_REFRESH_TTL_DAYS=7
RESMA_BCRYPT_COST=12
RESMA_LOGIN_RATE_LIMIT=5
```

## Recomendações sobre a Framework

1. **Seguir o bootstrap completo** — ler `.ai/AGENTS.md`, iniciar pelo Orchestrator
2. **Modo Review** — esta é uma implementação com auth, DB, API e frontend → modo Review (Context → Especialistas → Critic → Validator)
3. **Pre-Execution Gate** — apresentar Execution Target Map, plano, modelo recomendado e pedir aprovação antes de executar
4. **No Hardcode** — todos os thresholds, intervalos e limites via env vars (regra `rules/no-hardcode.md`)
5. **Frontend Runtime Validation** — validar porta, cache, console, network e DOM após implementar frontend (`rules/frontend-runtime-validation.md`)
6. **Frontend Anti-Patterns** — aplicar `rules/design/frontend-anti-patterns.md` para evitar UI genérica
7. **Security Intelligence** — acionar automaticamente para auth, JWT, bcrypt, rate limiting (`docs/SECURITY_INTELLIGENCE.md`)
8. **Execution Memory** — consultar aprendizados anteriores antes de iniciar
9. **Evidence Anchoring** — separar observado, inferido e hipótese durante implementação
10. **Post-Mission Evaluation** — registrar aprendizado reutilizável ao fim

## Ordem Sugerida de Implementação

1. **Estrutura base** — `pyproject.toml`, `frontend/` com Vite + pnpm + TailwindCSS v4 + shadcn/ui
2. **Backend core** — `config.py`, `db.py` (schema + Appender), `docker_client.py`
3. **Coleta** — `collector.py` (loop async + OOM listener)
4. **ML** — `recommender.py` (Z-score, Regressão Linear, KMeans, margem adaptativa)
5. **Templates** — `templates.py` + 6 arquivos YAML
6. **Auth** — `auth.py` (bcrypt, JWT, onboarding, login, middleware), `deps.py`
7. **API** — `main.py` (todos os endpoints, `app.frontend()`)
8. **Frontend auth** — `AuthContext`, `AuthGate`, `ProtectedRoute`, `Login.tsx`, `Onboarding.tsx`
9. **Frontend dashboard** — `Dashboard.tsx`, `Recommendations.tsx`, `Templates.tsx`, `Services.tsx`
10. **Docker** — `Dockerfile` (multi-stage com pnpm), `docker-compose.yml`
11. **Validação** — testar build, subir container, validar fluxo completo

## Critérios de Aceite

- [ ] Backend coleta métricas a cada 15s e armazena em DuckDB
- [ ] OOM events capturados via listener `/events`
- [ ] ML gera recomendações com confidence score
- [ ] Templates YAML funcionais (6 perfis)
- [ ] Onboarding funciona no primeiro acesso
- [ ] Login retorna JWT (access + refresh)
- [ ] Middleware bloqueia acesso sem token
- [ ] Frontend bloqueia rotas sem auth
- [ ] Dashboard exibe gráficos e KPIs
- [ ] Apply de limits/templates via Docker API
- [ ] Dockerfile build com pnpm (multi-stage)
- [ ] docker-compose.yml pronto para Swarm
- [ ] Sem hardcode (tudo via env vars)
- [ ] UI com TailwindCSS v4 + shadcn/ui original
