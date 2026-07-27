# Fase 0b — Migração da API para Go + SSE + Python ML Sidecar

> **Prioridade: CRÍTICA** — Pré-requisito para Fases 2, 3, 4, 5. Qualquer trabalho no backend Python antes desta fase será desperdço.

## Decisão

Migrar a camada de API/infraestrutura do RESMA de Python (FastAPI) para Go, mantendo o ML em Python como sidecar. Adicionar SSE (Server-Sent Events) para real-time.

### Justificativa

1. **Evitar trabalho desperdiçado** — Fases 2 (security), 3 (docs/OpenAPI), 4 (install), 5 (CI/CD) todas tocam no backend. Fazer isso em Python para depois migrar = retrabalho completo
2. **Gargalos são arquiteturais** — DuckDB RLock global, Docker API N+1, ML síncrono no request handler não são resolvíveis sem mudar o runtime
3. **SSE é core, não feature** — RESMA é monitoring tool. Real-time é diferencial competitivo. Go suporta 10x mais conexões SSE que Python
4. **ML fica em Python** — ecossistema scikit-learn/scipy/numpy é irreplaceable. Migração de ML para Go seria esforço alto com benefício zero
5. **Timing ideal** — projeto em Fase 0, ainda não open-source. Migrar agora é mais barato que migrar com comunidade ativa
6. **Footprint** — Go binary ~15MB vs Python image ~300MB. CPU 0.1 + 64M vs 0.5 + 512M

### O que NÃO migra

- **Frontend** (React + Vite + Tailwind + shadcn/ui) — intacto
- **ML/Recommender** (Python com sklearn/scipy/numpy) — vira sidecar
- **test-env** — independente de linguagem
- **docs-site** (Docusaurus) — independente

## Arquitetura alvo

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Swarm                          │
│                                                          │
│  ┌─────────────────────┐     HTTP interno     ┌────────┐│
│  │   resma-api (Go)    │ ────────────────────→ │ resma  ││
│  │                     │                       │ -ml    ││
│  │  • REST endpoints   │  POST /analyze        │ (Py)   ││
│  │  • SSE streaming    │  {service, metrics}   │        ││
│  │  • Docker SDK       │ ←──────────────────── │ sklearn││
│  │  • DuckDB queries   │  {recommendation}     │ scipy  ││
│  │  • Auth (JWT+bcrypt)│                       │ numpy  ││
│  │  • Collector        │                       │        ││
│  │  • Scheduler        │                       │ DuckDB ││
│  └─────────────────────┘                       └────────┘│
│         │              │                          │       │
│    docker.sock     resma-data                 resma-data  │
│    (ro)            (volume)                   (volume)   │
└─────────────────────────────────────────────────────────┘
```

### Comunicação Go ↔ Python

- **Protocolo**: HTTP JSON (simples, debugável) ou gRPC (performance)
- **Latência**: ~1-5ms por chamada (aceitável — análise já leva 100ms+)
- **Python ML expõe**: `POST /analyze` (recebe métricas, devolve recomendação)
- **Go API chama**: apenas durante `/recommendations/{service}/trigger` e `/dashboard`

### Docker compose alvo

```yaml
services:
  resma-api:
    image: docker.io/resmaswarm/resma-api:latest
    ports:
      - target: 8080
        published: 8080
        mode: host
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - resma-data:/data
    environment:
      - RESMA_DB_PATH=/data/resma.duckdb
      - RESMA_ML_URL=http://resma-ml:8081
      # ... demais settings
    deploy:
      placement:
        constraints:
          - node.role == manager
      resources:
        limits:
          cpus: "0.25"
          memory: 128M
        reservations:
          cpus: "0.05"
          memory: 32M

  resma-ml:
    image: docker.io/resmaswarm/resma-ml:latest
    volumes:
      - resma-data:/data
    environment:
      - RESMA_DB_PATH=/data/resma.duckdb
    deploy:
      resources:
        limits:
          cpus: "0.25"
          memory: 256M
        reservations:
          cpus: "0.05"
          memory: 64M

volumes:
  resma-data:
```

## Mapeamento de componentes

| Componente Python | Destino | Tecnologia Go | Esforço |
|---|---|---|---|
| `main.py` (69 linhas) | Go API | `net/http` ou Gin | Baixo |
| `docker_client.py` (714 linhas) | Go API | Docker SDK oficial (`moby/moby/client`) | Médio |
| `db.py` (861 linhas) | Go API | `marcboeker/go-duckdb` + `database/sql` | Médio |
| `collector.py` (345 linhas) | Go API | Goroutines + ticker | Médio |
| `scheduler.py` (153 linhas) | Go API | Goroutine + ticker | Baixo |
| `auth.py` (201 linhas) | Go API | `golang-jwt/jwt` + `golang.org/x/crypto/bcrypt` + **API key com scopes** | Médio |
| 11 routers (~1,000 linhas) | Go API | HTTP handlers em 2 mux trees (`/api/v1/*` público + `/api/*` interno) | Médio |
| `recommender.py` (560 linhas) | **Python sidecar** | Mantém sklearn/scipy/numpy | — |
| `templates.py` (147 linhas) | Go API | `gopkg.in/yaml.v3` | Baixo |
| SSE broker (novo) | Go API | stdlib `net/http` + `http.Flusher` | Baixo |
| API key model (novo) | Go API | tabela `api_keys` (DuckDB) + middleware de scopes | Médio |

## API Surface Architecture — Split público/interno

> **Decisão do Technical Council:** um binário Go, duas superfícies de rota com políticas de auth, versionamento e estabilidade diferentes. Não dois serviços — a divisão é de routing + middleware, não de deployment.

### Justificativa (baseada em pesquisa web)

| Fonte | Posição |
|-------|---------|
| [INNOQ — Indecent Exposure](https://www.innoq.com/en/blog/2026/06/indecent-exposure-your-domain-model-is-showing/) | SPA APIs vazam internals de domínio. Entregar endpoints de frontend a integradores = contrato de estabilidade implícito que você vai quebrar. Split. |
| [Alanta — Your frontend deserves its own backend](https://alanta.nl/posts/2026/03/your-frontend-deserves-its-own-backend) | Endpoints operacionais/sensíveis acabam na superfície pública por descuido quando tudo vive no mesmo router. Split por audiência. |
| [Latch — Separate Public APIs from Internal](https://latchworkflow.com/blog/public-api-internal-endpoint-separation/) | API pública = superfície de produto. Endpoint interno = superfície de operação. Misturar = blast radius grande em incidente/pen-test. |
| [SE Stack Exchange — consensus](https://softwareengineering.stackexchange.com/questions/461192/) | Não split só porque big company faz. Single API é fine no início. Adicione complexidade depois se necessário. |
| [James Ross — API-First](https://www.jamesrossjr.com/blog/api-first-architecture) | Uma API, UI e externos consomem o mesmo. Força a API ser boa. Single, mas com API keys scoped. |
| [Versioning consensus](https://apistatuscheck.com/blog/api-versioning-strategies) | Para API pública: URL path versioning `/v1/` (Stripe, Twilio, GitHub). Cacheable, discoverable, curl-friendly. |

**Síntese:** Single API pura trava mudanças de contrato depois que comunidade OSS existir (dashboard é blob UI-shaped). Dois serviços é overkill nesta escala. **Um binário, duas superfícies de rota** resolve: público versionado e estável, interno não-versionado e ágil.

### Estrutura de rotas

```
resma-api (Go, um binário, um router tree)
├── /api/v1/*          ← PÚBLICO: estável, versionado, OpenAPI, API key + scopes
├── /api/*             ← INTERNO/UI: não-versionado, JWT, muda rápido
├── /api/sse/*         ← STREAMING: JWT (UI) ou API key (integração real-time)
├── /health, /ready    ← INFRA: sem auth, sem rate limit
└── /swagger/*         ← OpenAPI docs (só /api/v1/*)
```

### Classificação dos 40 endpoints

**Público `/api/v1/`** (resource-oriented, estável, leitura no v1):
- `GET /v1/services`, `/v1/services/{name}/metrics`, `/v1/services/{name}/stats`, `/v1/services/{name}/containers`
- `GET /v1/containers/{id}/metrics`, `/v1/containers/{id}/stats`
- `GET /v1/nodes`, `/v1/nodes/{id}`, `/v1/nodes/{id}/metrics`, `/v1/nodes/{id}/services`, `/v1/cluster`
- `GET /v1/oom-events`
- `GET /v1/storage/trend`, `/v1/storage/volumes/growth`, `/v1/storage/volumes/{name}/growth`
- `GET /v1/recommendations`, `/v1/recommendations/{service}`, `/v1/recommendations/storage`
- `GET /v1/change-log`

**Interno/UI `/api/`** (não-versionado, JWT, muda rápido):
- `GET /api/dashboard` (blob agregado UI — 7 conceitos de domínio)
- `GET /api/services/sparklines` (widget)
- `GET /api/config` (operacional)
- `GET /api/storage/summary` (live Docker DF — operacional)
- `GET /api/containers/{id}/network-info` (painel de detalhe UI)
- `POST /api/recommendations/{service}/apply` (mutação — admin via UI)
- `POST /api/recommendations/recalculate`
- `/api/templates/*` (CRUD + apply — admin via UI)
- `/api/schedules/*` (CRUD — admin via UI)
- `PATCH /api/services/{name}/archive|restore` (admin)
- `/api/auth/*` (fluxos humanos: onboarding, login, refresh, logout, me, change-password)

**Mutações no público** (só quando API keys scoped existirem — v1.1+):
- `POST /v1/recommendations/{service}/apply` (scope: `write`)
- `POST /v1/templates/{name}/apply/{service}` (scope: `write`)
- `POST /v1/schedules`, `DELETE /v1/schedules/{id}` (scope: `write`)

### Auth model

| Superfície | Auth | Rate limit |
|-----------|------|------------|
| `/api/v1/*` | API key + scopes (`read`, `write`) | Generoso, por key |
| `/api/*` | JWT (cookie/bearer) | Normal, por user |
| `/api/sse/*` | Cookie HttpOnly `sse_session` (browser/EventSource) OU Authorization header (clients não-browser) | Conexões |
| `/health`, `/ready` | Nenhuma | Sem limite |

### Versionamento

URL path `/v1/` — consenso unânime da pesquisa para APIs públicas. Cacheable, discoverable, curl-friendly, usado por Stripe/Twilio/GitHub. Header versioning é só para microserviços internos.

### Impacto nas tarefas da Fase 0b

| Tarefa | Ajuste |
|--------|--------|
| 0b.4 (auth) | Adicionar API key model (tabela `api_keys` com scopes `read`/`write`) além de JWT. Middleware que valida API key no header `Authorization: Bearer resma_key_...` ou `X-API-Key`. |
| 0b.5 (routers) | Handlers registrados em dois mux trees: `/api/v1/*` (público) e `/api/*` (interno). Handlers compartilhados onde shape é idêntico; handlers divergentes para dashboard/sparklines. swaggo anotações apenas em `/api/v1/*`. |
| 0b.7 (SSE) | Suportar **cookie HttpOnly** (browser via EventSource) **+ Authorization header** (clients não-browser via fetch). Ver seção "Frontend Impact — SSE Auth" abaixo. |
| Phase 3 (docs) | Docusaurus consome `swagger.json` do `/v1/` apenas. UI interna não precisa de OpenAPI. |

## Frontend Impact — O que muda no React

> **Prioridade:** Esta seção documenta o impacto da migração no frontend. **Toda alteração no frontend acontece DEPOIS da API Go estar funcional (0b.1–0b.10)**. Durante 0b.1–0b.10 o frontend continua consumindo a API Python sem mudanças. A tarefa 0b.11 é a única que toca no frontend.

### Decisão: Frontend fica em `/api/*` (interno)

O frontend **não migra para `/api/v1/*`** nesta fase. Razões:

1. **Auth flow quebra totalmente** — login/onboarding/refresh/logout usam JWT bearer via `Authorization` header. API key não suporta login com username/password, não tem refresh tokens, não tem `/auth/me`. Migrar para API key exigiria redesenhar todo o `AuthContext`.
2. **Endpoints internos-only** — `/dashboard`, `/services/sparklines`, `/config`, `/storage/summary`, `/containers/{id}/network-info`, `/templates/*`, `/schedules/*`, `/auth/*` não existem em `/api/v1/*`. O frontend usa todos eles.
3. **Sem benefício imediato** — `/api/v1/*` é para integradores externos. O frontend é o consumidor interno por excelência.
4. **Migração futura opcional** — se comunidade OSS pedir uma SPA que consome só a API pública, pode ser feita gradualmente em versão posterior (v0.2+).

**Estratégia:** `API_BASE = "/api"` mantido. Todos os 40 endpoints atuais continuam funcionando com JWT. O Vite proxy `/api` → `:8080` cobre tanto `/api/*` quanto `/api/v1/*` automaticamente (proxy é por prefixo).

### Mapa de endpoints consumidos pelo frontend (40 endpoints)

Mapeamento completo feito a partir do código em `frontend/src/`. Classificação confirma que o frontend consome majoritariamente endpoints internos:

| Categoria | Endpoints | Classificação |
|-----------|-----------|---------------|
| **Auth** (6) | `/auth/status`, `/auth/me`, `/auth/login`, `/auth/onboarding`, `/auth/logout`, `/auth/refresh` | Interno-only |
| **Dashboard** (3) | `/dashboard`, `/config`, `/storage/summary` | Interno-only |
| **Services** (9) | `/services`, `/services/{name}/stats`, `/services/{name}/metrics`, `/services/{name}/containers`, `/services/sparklines`, `/services/containers/{id}/stats`, `/services/containers/{id}/metrics`, `/services/containers/{id}/network-info`, `/services/{name}/archive` | Mix (7 públicos, 2 internos) |
| **Nodes** (4) | `/nodes`, `/nodes/{id}`, `/nodes/{id}/metrics`, `/nodes/{id}/services` | Público |
| **Recommendations** (5) | `/recommendations`, `/recommendations/{service}`, `/recommendations/storage`, `/recommendations/{service}/apply`, `/recommendations/{service}/recalculate` | Mix (3 públicos leitura, 2 internos mutação) |
| **Schedules** (5) | `/schedules`, `/schedules?service=`, `/schedules/pending`, `POST /schedules`, `DELETE /schedules/{id}` | Interno-only |
| **Change Log** (2) | `/change-log`, `/change-log/{service}` | Público (leitura) |
| **Templates** (5) | `/templates` (CRUD), `/templates/{name}/apply/{service}` | Interno-only |

**Conclusão:** 26 dos 40 endpoints que o frontend consome são interno-only. Migrar para `/api/v1/*` não é viável sem redesenhar o frontend.

### SSE Auth — Decisão: Cookie HttpOnly

**Problema:** `EventSource` nativo do browser **não suporta headers custom**. Não é possível enviar `Authorization: Bearer <jwt>` via EventSource. O frontend usa JWT bearer para todas as chamadas REST.

**Pesquisa web (6 fontes):** gptme migrou de query param → cookie HttpOnly por segurança; Centrifugo tem feature request aberta para o mesmo; server-sent-events.com recomenda cookie; Strait.dev usa short-lived ticket.

| Opção | Segurança | Compatibilidade JWT | Esforço | Veredito |
|-------|-----------|---------------------|---------|----------|
| Cookie HttpOnly | Alta | Alta (coexiste) | Médio | ✅ **Escolhida** |
| JWT via query param | Baixa (logs, history, referrer) | Alta | Baixo | ❌ Anti-padrão |
| Short-lived ticket | Média-Alta | Alta | Alto | ❌ Overkill |
| Polyfill (`@microsoft/fetch-event-source`) | Alta | Perfeita | Baixo | ⚠️ Alternativa |

**Decisão: Cookie HttpOnly com troca de token.**

Implementação no backend Go:
```go
// POST /api/sse/session — troca JWT bearer por cookie de sessão SSE
// Header: Authorization: Bearer <jwt>
// Set-Cookie: sse_session=<signed_jwt>; HttpOnly; Secure; SameSite=Lax; Path=/api/sse/; Max-Age=600

// GET /api/sse/* — valida cookie sse_session OU Authorization header (para clients não-browser)
```

Implementação no frontend:
```typescript
// 1. Após login, obter cookie de sessão SSE
await fetch('/api/sse/session', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${jwt}` },
  credentials: 'include',  // necessário para cookies cross-origin em dev
})

// 2. Abrir EventSource com cookie (enviado automaticamente)
const es = new EventSource('/api/sse/metrics', { withCredentials: true })
```

**Impacto no AuthContext:**
- `login()` e `onboarding()` chamam `POST /api/sse/session` após receber tokens
- `logout()` chama `DELETE /api/sse/session` para invalidar cookie
- Refresh de cookie: quando JWT for renovado (401 → refresh), re-chamar `POST /api/sse/session`
- Cookie TTL curto (10 min) — se expirar, SSE reconecta e reobtém cookie

**Alternativa de fallback:** Se cookie HttpOnly provar complexo em dev (CORS cross-origin entre Vite :5173 e Go :8080), usar `@microsoft/fetch-event-source` (polyfill que suporta headers custom via Fetch API). Mantém JWT bearer sem mudança no backend. Trade-off: +15KB no bundle.

### Vite Proxy — Ajuste para SSE

O proxy atual (`vite.config.ts:13-17`) é:
```typescript
proxy: { "/api": "http://localhost:8080" }
```

**Problema:** O proxy do Vite (baseado em `http-proxy`) pode bufferizar respostas SSE, quebrando o streaming. SSE exige flush imediato de cada evento.

**Solução:** Configurar proxy para SSE com headers corretos:
```typescript
proxy: {
  "/api/sse": {
    target: "http://localhost:8080",
    changeOrigin: true,
    // SSE: não bufferizar, manter conexão aberta
    configure: (proxy) => {
      proxy.on('proxyRes', (proxyRes) => {
        proxyRes.headers['x-accel-buffering'] = 'no'
      })
    },
  },
  "/api": "http://localhost:8080",
}
```

**Ordem importa:** `/api/sse` deve vir antes de `/api` para match correto.

**Produção:** Nginx/Traefik precisa de `proxy_buffering off` e `proxy_read_timeout 24h` para SSE. Documentar em Fase 4 (docker-stack.yml com Traefik labels).

### Tipos TypeScript — Contrato a manter

A API Go deve retornar **exatamente os mesmos shapes** que o frontend espera hoje. Os tipos TypeScript abaixo (extraídos do código) são o contrato de compatibilidade:

| Tipo | Arquivo | Campos principais |
|------|---------|-------------------|
| `User` | AuthContext.tsx | `id, username, role` |
| `DashboardData` | Dashboard.tsx | `total_services, top_cpu_consumers[], recent_ooms[], leak_alerts[], cluster, cluster_capacity, nodes_distribution[]` |
| `StorageSummary` | Dashboard.tsx | `live{images, containers, volumes, top_volumes[], orphan_volumes[]}, latest_snapshot` |
| `Service` | Services.tsx | `name, container_count, active_containers, cpu_p95, mem_p99, last_seen, status, current?` |
| `ServiceStats` | ServiceDetail.tsx | `service, samples, cpu_p50/p95/min/max/avg, mem_p50/p99/min/max/avg` |
| `MetricPoint` | ServiceDetail.tsx | `ts, cpu_percent, mem_usage, mem_limit, mem_percent` |
| `ContainerStats` | ServiceDetail.tsx | `container_id, samples, cpu/mem stats, last_seen, status, networks?` |
| `ContainerNetworkInfo` | ContainerDetail.tsx | `network, ip_address, ipv6_address, mac_address, gateway, endpoint_id` |
| `Node` / `NodeDetail` | Nodes.tsx | `node_id, hostname, role, availability, status, address, cpu_total, mem_total, os, architecture, engine_version, is_leader, reachability, labels, tasks_running, cpu_p95, mem_p99, containers, updated_at` |
| `NodeMetric` | NodeDetail.tsx | `ts, tasks_running, cpu_total, mem_total` |
| `NodeService` | NodeDetail.tsx | `service, containers, cpu_p95, mem_p99` |
| `Recommendation` | Recommendations.tsx | `service, samples, status, stack?, current?, outliers_removed?, cpu{p50,p95}, mem{p50,p99}, oom_events?, has_drift?, pattern?, memory_trend{slope, daily_growth_mb, r_squared, has_leak}, forecast{days_ahead, projected_mem, projected_mem_p99, slope}, suggested?, suggested_apply_time?, confidence?` |
| `StorageAnalysis` | Recommendations.tsx | `total_reclaimable, recommendations[]{service, volume_name, reclaimable_size, reclaimable_percent, last_used}` |
| `Schedule` | Schedules.tsx | `id, service, cpu_limit, mem_limit, cpu_reservation, mem_reservation, scheduled_at, status, applied_at, error, attempts, created_at` |
| `ChangeLogEntry` | Schedules.tsx | `id, service, action, source, schedule_id, cpu/mem_before/after, user, status, error, docker_response, created_at` |
| `Template` | Templates.tsx | `id, name, description, yaml_content, stacks[], created_at?, updated_at?` |
| `AppConfig` | use-refresh.ts | `collect_interval, retention_days, analysis_window_days` |

**Validação:** A tarefa 0b.12 (testes de equivalência) deve incluir comparação de shapes entre respostas Python e Go para estes tipos. Considerar gerar tipos TypeScript do OpenAPI do swaggo em versão futura (v0.2+) para garantir drift detection.

### Páginas candidatas a SSE (priorização)

| Prioridade | Página | Endpoint(s) atual(es) | Intervalo | Endpoint SSE alvo |
|-----------|--------|----------------------|-----------|-------------------|
| Alta | Dashboard | `/dashboard`, `/storage/summary` | 15s dinâmico | `/api/sse/dashboard` |
| Alta | Services | `/services`, `/recommendations`, `/services/sparklines` | 15s dinâmico | `/api/sse/metrics` |
| Alta | ServiceDetail | `/services/{name}/metrics`, `/containers` | 15s dinâmico | `/api/sse/services/{service}` |
| Alta | Nodes | `/nodes` | 15s dinâmico | `/api/sse/nodes` |
| Média | NodeDetail | `/nodes/{id}/metrics`, `/services` | 15s dinâmico | `/api/sse/nodes` |
| Média | Schedules | `/schedules`, `/change-log` | 30s fixo | `/api/sse/events` |
| Baixa | ContainerDetail | `/containers/{id}/stats`, `/metrics` | 15s dinâmico | `/api/sse/services/{service}` |
| Baixa | Recommendations | `/recommendations`, `/schedules/pending` | 15s/30s | `/api/sse/events` |
| Mantém polling | Templates | — | Sem polling | — |
| Mantém polling | Login/Onboarding | — | — | — |

## SSE — Endpoints planejados

| Endpoint | Eventos | Frequência |
|---|---|---|
| `GET /api/sse/metrics` | `metric_update` por service | A cada coleta (1s) |
| `GET /api/sse/dashboard` | Snapshot agregado (top CPU, top mem, OOMs, leaks) | A cada 5s |
| `GET /api/sse/events` | Docker events (container start/stop/die, OOM) | Tempo real |
| `GET /api/sse/services/{service}` | Métricas de um service específico | A cada 1s |
| `GET /api/sse/nodes` | Status de nodes do cluster | A cada 5s |

### Padrão SSE em Go (stdlib)

```go
func (h *Handler) SSEMetrics(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", 500)
        return
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-r.Context().Done():
            return
        case <-ticker.C:
            snapshot := h.collector.Snapshot()
            data, _ := json.Marshal(snapshot)
            fmt.Fprintf(w, "event: metric_update\ndata: %s\n\n", data)
            flusher.Flush()
        }
    }
}
```

## DuckDB em Go — validação

| Aspecto | Status |
|---|---|
| Driver | `github.com/marcboeker/go-duckdb` — ativo, production-ready |
| API | `database/sql` compatível — padrão Go |
| Arquivo `.duckdb` | Compatível entre Python e Go (mesma engine C) |
| Appender API | Inserção em lote 4x mais rápida que batch insert |
| Concorrência | Goroutines + `sql.DB` pool — sem GIL, sem RLock |
| CGO | Necessário (DuckDB é C embarcado). Build com `CGO_ENABLED=1` |

### Schema migration

**Nenhuma migração necessária.** O arquivo `resma.duckdb` existente é aberto diretamente pelo Go. Todas as tabelas, sequences e dados são preservados.

## ML Sidecar — API

### Python ML service (FastAPI minimal)

```python
# resma-ml/main.py
from fastapi import FastAPI
from pydantic import BaseModel
from backend.services.recommender import ResourceRecommender

app = FastAPI(title="RESMA ML")
recommender = ResourceRecommender()

class AnalyzeRequest(BaseModel):
    service: str
    metrics: list[dict]  # rows from DuckDB

@app.post("/analyze")
async def analyze(req: AnalyzeRequest):
    return await recommender.analyze_with_data(req.service, req.metrics)

@app.post("/forecast")
async def forecast(req: AnalyzeRequest):
    return await recommender.forecast_with_data(req.service, req.metrics)

@app.get("/health")
async def health():
    return {"status": "ok"}
```

### Go API chamando ML

```go
func (s *MLService) Analyze(service string, metrics []Metric) (*Recommendation, error) {
    body, _ := json.Marshal(map[string]interface{}{
        "service": service,
        "metrics": metrics,
    })
    resp, err := http.Post(s.baseURL+"/analyze", "application/json", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    var rec Recommendation
    json.NewDecoder(resp.Body).Decode(&rec)
    return &rec, nil
}
```

## Estrutura de pastas alvo

```
resma/
├── app/
│   ├── api/                    # Go API (novo)
│   │   ├── cmd/
│   │   │   └── server/
│   │   │       └── main.go
│   │   ├── internal/
│   │   │   ├── handlers/       # HTTP handlers (11 routers)
│   │   │   ├── docker/         # Docker SDK wrapper
│   │   │   ├── db/             # DuckDB layer
│   │   │   ├── collector/      # Metrics collector (goroutines)
│   │   │   ├── scheduler/      # Scheduler (goroutine)
│   │   │   ├── auth/           # JWT + bcrypt
│   │   │   ├── sse/            # SSE broker
│   │   │   ├── ml/             # HTTP client para Python sidecar
│   │   │   └── config/         # Config (env vars)
│   │   ├── go.mod
│   │   ├── go.sum
│   │   └── Dockerfile
│   ├── ml/                     # Python ML sidecar (novo)
│   │   ├── main.py             # FastAPI minimal
│   │   ├── recommender.py      # Lógica ML (de backend/services/recommender.py)
│   │   ├── requirements.txt
│   │   └── Dockerfile
│   └── frontend/               # React (mantém)
│       ├── src/
│       ├── package.json
│       └── ...
├── test-env/
├── docs-site/
└── docker-compose.yml          # Atualizado com 2 serviços
```

## Tarefas

### 0b.1 — Setup Go + estrutura base
- Criar `app/api/` com `go.mod`, `cmd/server/main.go`
- Configurar `database/sql` + `go-duckdb`
- Configurar Docker SDK
- **Esforço:** 2h

### 0b.2 — Migrar DuckDB layer
- Portar schema init de `db.py`
- Portar queries de todos os routers
- Implementar Appender API para batch insert
- Validar com arquivo `.duckdb` existente
- **Corrigir cálculo de memory working set:** `working_set = memory.usage - inactive_file` (cgroup v2) ou `memory.usage - cache` (cgroup v1). O Docker stats API retorna ambos os campos. Usar working set (não raw usage) para recomendações de memória — é o valor que o kernel considera não-reclaimable e o que Kubernetes usa para OOM kill
- **Adicionar CPU throttling metrics:** coletar `cpu_stats.throttling_data.throttled_periods` e `throttled_time` do Docker stats API. CPU throttling é um dos problemas mais comuns em Swarm — container pode usar "50% CPU" mas ser throttled 50% do tempo
- **Esforço:** 4h

### 0b.3 — Migrar Docker client
- Portar `docker_client.py` usando SDK oficial `moby/moby/client`
- Mapear: list containers, stats stream, services, nodes, tasks, events
- **Eliminar N+1 de `get_all_service_resources()`:** usar `cli.ServiceList(ctx, types.ServiceListOptions{})` que retorna todos os serviços com `Spec.Resources` em uma única chamada. No Python atual, cada serviço requer um `inspect()` separado. Em Go, uma chamada resolve
- Manter stats streaming por container (padrão correto, confirmado por cAdvisor, docker-exporter, Cetacean)
- Manter event-driven container discovery via Docker events (padrão correto)
- **Esforço:** 4h

### 0b.4 — Migrar Auth + API Key model
- JWT com `golang-jwt/jwt/v5`
- bcrypt com `golang.org/x/crypto/bcrypt`
- Portar onboarding, login, refresh, logout, change-password
- **API key model (novo):** tabela `api_keys` no DuckDB com campos `id`, `key_hash`, `name`, `scopes` (`read`/`write`), `created_at`, `last_used_at`, `revoked_at`. Middleware que valida `Authorization: Bearer resma_key_...` ou `X-API-Key` header. Endpoints internos `/api/auth/api-keys` (CRUD) para admin gerenciar keys via UI.
- **Esforço:** 3h

### 0b.5 — Migrar Routers (11 routers, ~40 endpoints) — Split público/interno
- Portar todos os handlers HTTP em **dois mux trees**:
  - `/api/v1/*` — público, versionado, API key + scopes, OpenAPI via swaggo
  - `/api/*` — interno/UI, JWT, não-versionado
- Handlers compartilhados onde shape é idêntico (services, nodes, oom, storage, recommendations leitura, change-log)
- Handlers divergentes para dashboard (blob UI-shaped fica em `/api/`) e sparklines
- Manter mesmos contratos de resposta para compatibilidade com frontend atual (frontend continua consumindo `/api/*` durante 0b; migração para `/api/v1/` é opcional e gradual)
- **Anotar apenas handlers `/api/v1/*` com `swaggo/swag`:** anotações `@Summary`, `@Tags`, `@Param`, `@Success`, `@Router` nos comentários Go. `swag init` gera `swagger.json` + `docs.go` automaticamente.
- **Servir Swagger UI embutido:** rota `/swagger/*` com `swaggo/http-swagger` (compatível com `net/http` stdlib). Swagger UI acessível na própria API Go, documentando apenas `/api/v1/*`
- **Esforço:** 7h

### 0b.6 — Migrar Collector + Scheduler
- Collector: 7 goroutines + ticker (stats, events, OOM, retention, nodes, cluster, storage)
- Scheduler: 1 goroutine + polling
- **Esforço:** 3h

### 0b.7 — Implementar SSE broker
- Broker com fan-out via channels
- 5 endpoints SSE (metrics, dashboard, events, services, nodes)
- Keepalive ping a cada 15s
- Disconnect detection via `r.Context().Done()`
- **Auth dual (ver seção "Frontend Impact — SSE Auth"):**
  - `POST /api/sse/session` — troca JWT bearer por cookie `sse_session` HttpOnly (Max-Age=600, Path=/api/sse/, SameSite=Lax, Secure em produção)
  - `DELETE /api/sse/session` — invalida cookie
  - Endpoints `/api/sse/*` validam cookie `sse_session` OU header `Authorization: Bearer` (para clients não-browser como curl, scripts)
  - CORS: `Access-Control-Allow-Credentials: true` + origens específicas (não `*`) quando cookie está em uso
- **Esforço:** 4h (aumentou de 3h para refletir endpoint de sessão + cookie + CORS com credentials)

### 0b.8 — Criar Python ML sidecar
- FastAPI minimal com `/analyze`, `/forecast`, `/health`
- Mover `recommender.py` para `app/ml/`
- Dockerfile Python slim
- **Esforço:** 2h

### 0b.9 — Integrar Go ↔ Python ML
- HTTP client no Go para chamar ML sidecar
- Fallback graceful se ML indisponível
- Health check entre serviços
- **Esforço:** 1h

### 0b.10 — Atualizar Dockerfile + docker-compose
- Dockerfile multi-stage Go (build + scratch/distroless)
- Dockerfile Python ML slim
- docker-compose.yml com 2 serviços
- Validar deployment no Swarm
- **Esforço:** 2h

### 0b.11 — Atualizar frontend para SSE
- **Pré-requisito:** 0b.1–0b.10 completos (API Go funcional com SSE endpoints)
- **Estratégia híbrida:** SSE para invalidação de cache + TanStack Query para fetch. O servidor envia sinal "dados mudaram", o cliente faz `queryClient.invalidateQueries()`. Padrão confirmado por pesquisa: -90% API calls, -77% server load
- **Auth via cookie HttpOnly** (ver seção "Frontend Impact — SSE Auth" acima):
  - `AuthContext.login()` e `onboarding()`: após receber JWT, chamar `POST /api/sse/session` para obter cookie
  - `AuthContext.logout()`: chamar `DELETE /api/sse/session` para invalidar cookie
  - Refresh de cookie: quando JWT for renovado (401 → refresh), re-chamar `POST /api/sse/session`
  - `credentials: 'include'` no fetch do cookie (necessário para cross-origin em dev Vite :5173 → Go :8080)
- **Vite proxy ajustado** para SSE (ver seção "Vite Proxy" acima): `/api/sse` com `x-accel-buffering: no` antes de `/api`
- **Criar hook `useEventSource`** centralizado:
  - Owns lifecycle do `EventSource` (open, message, error, close)
  - Cleanup correto no `useEffect` return (React StrictMode dobra effects em dev)
  - Coalescing de eventos (múltiplos eventos → 1 invalidation)
  - Exponential backoff manual para reconexão (controle fino vs reconexão nativa)
  - Integração com `queryClient.invalidateQueries()` por queryKey
- **Páginas migradas (prioridade alta):** Dashboard, Services, ServiceDetail, Nodes
- **Páginas mantidas com polling:** Templates (sem polling hoje), Login/Onboarding, ContainerDetail (migração gradual pós-0b)
- **Manter polling como fallback:** se SSE falhar (EventSource error), voltar para `refetchInterval` atual. Páginas secundárias podem continuar com polling até migração gradual
- **Cuidados:**
  - HTTP/1.1 limita 6 conexões por origem (HTTP/2 resolve — configurar Go server para HTTP/2)
  - React StrictMode dobra effects em dev (cleanup obrigatório para evitar EventSource leak)
  - Cookie `Max-Age=600` (10 min) — se expirar, SSE reconecta e reobtém cookie automaticamente
  - `withCredentials: true` no EventSource para enviar cookie cross-origin
  - CORS: `Access-Control-Allow-Credentials: true` + origem específica (não `*`) no Go
- **Fallback alternativo:** Se cookie HttpOnly provar complexo em dev (CORS cross-origin), usar `@microsoft/fetch-event-source` (polyfill que suporta headers custom via Fetch API). Mantém JWT bearer sem mudança no backend. Trade-off: +15KB no bundle. Decidir durante implementação.
- **Esforço:** 5h (aumentou de 3h para refletir auth cookie + Vite proxy + hook centralizado)

### 0b.12 — Testes de equivalência
- Validar que todos os endpoints retornam mesmos contratos
- Testar com `test-env` (serviços simulados)
- Benchmark comparativo Go vs Python (latência, RAM, conexões SSE)
- **Esforço:** 3h

## Benchmarks esperados

| Métrica | Python (atual) | Go (esperado) | Melhoria |
|---|---|---|---|
| RAM idle | ~95MB | ~18MB | 5x |
| RAM @ 100 SSE clients | ~400MB | ~40MB | 10x |
| p50 latency (dashboard) | ~2-5s | ~50-100ms | 20-50x |
| p99 latency (list services) | ~300ms | ~50ms | 6x |
| Max SSE conexões | ~200-500 | ~10,000+ | 20-50x |
| Docker image size | ~300MB | ~15MB (API) + ~80MB (ML) | 2x |
| CPU usage (coletor) | ~0.5 CPU | ~0.05 CPU | 10x |

## Riscos

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| DuckDB Go driver menos maduro | Baixa | Médio | `marcboeker/go-duckdb` é ativo, compatível com `database/sql` |
| CGO complica build | Média | Baixo | Multi-stage Dockerfile com `CGO_ENABLED=1` |
| Perda de OpenAPI auto-gerado | Resolvido | Baixo | `swaggo/swag` gera Swagger 2.0 das anotações nos handlers Go. `docusaurus-plugin-openapi-docs` (PaloAlto Networks) consome o `swagger.json` no Docusaurus — gera MDX com try-it-out, exemplos e schemas. Pipeline: anotações → `swag init` → Swagger UI na API + Docusaurus API reference |
| Latência Go→Python ML | Média | Baixo | Cache de resultados, ML roda em background |
| Frontend quebra com mudanças | Baixa | Médio | Manter mesmos contratos de resposta, testar com test-env |

## Critérios de aceite

- [ ] Todos os 40+ endpoints retornam mesmos contratos de resposta
- [x] Arquivo `resma.duckdb` existente abre sem migração
- [ ] SSE streaming funciona com 100+ clientes simultâneos
- [ ] ML sidecar responde em < 200ms para análise de 1 service
- [ ] Docker image total (API + ML) < 150MB
- [ ] RAM total < 200MB com 50 SSE clients
- [ ] Frontend funciona sem mudanças de contrato (exceto SSE opt-in)
- [ ] `docker compose up` funciona com 2 serviços
- [ ] **API split implementado:** `/api/v1/*` (público, API key) e `/api/*` (interno, JWT) com middlewares distintos
- [ ] **API key model:** tabela `api_keys` com scopes `read`/`write`, CRUD via `/api/auth/api-keys`
- [ ] **OpenAPI:** `swagger.json` gerado por `swag init` cobre apenas `/api/v1/*`; Swagger UI servido em `/swagger/*`
- [ ] **SSE auth:** cookie HttpOnly para browser (EventSource) + Authorization header para clients não-browser
- [ ] **Frontend compatibilidade:** todos os 40 endpoints retornam mesmos shapes TypeScript (ver tabela "Tipos TypeScript — Contrato a manter")
- [ ] **Vite proxy:** configurado para SSE sem bufferização
- [ ] **Frontend SSE:** Dashboard, Services, ServiceDetail, Nodes migrados de polling para SSE com fallback

## Validação arquitetural — Coleta de métricas Docker

### O que está correto (confirmado por pesquisa de mercado)

- **Stats streaming por container** — `container.stats(stream=True)` mantém conexão persistente por container. Padrão usado por cAdvisor, docker-exporter e Cetacean
- **Event-driven container discovery** — Docker events listener adiciona/remove containers do cache automaticamente. Mesmo padrão de OneUptime e Cetacean
- **Container cache em memória** — evita re-listar containers a cada ciclo de coleta
- **Batch insert no DuckDB** — inserção em lote, não linha-a-linha
- **Exclusão de imagens** — filtro para não coletar métricas do próprio RESMA

### Gaps corrigidos nesta fase

| Gap | Ação | Tarefa |
|---|---|---|
| Memory working set não calculado | Calcular `usage - inactive_file` (v2) ou `usage - cache` (v1) | 0b.2 |
| CPU throttling não coletado | Adicionar `throttled_periods` e `throttled_time` ao schema | 0b.2 |
| N+1 em `get_all_service_resources()` | Usar `ServiceList()` — 1 chamada retorna todos com `Spec.Resources` | 0b.3 |

### Gaps futuros (pós-0b)

| Gap | Descrição | Prioridade |
|---|---|---|
| Swarm task monitoring | Swarm opera em tasks (pending/accepted/running/failed). Container só existe quando task está running. Task stuck em pending = serviço degradado invisível para RESMA. Adicionar poll de `service.tasks()` periodicamente | Média — feature futura |

### Comparação com referências do mercado

| Aspecto | RESMA (atual) | Cetacean | OneUptime | docker-exporter |
|---|---|---|---|---|
| Stats collection | Stream por container | In-memory cache | OTel collector | On-demand scrape |
| Event-driven | Docker events | Docker events | Inventory poller | N/A |
| Working set | **Corrigir em 0b.2** | N/A (Prometheus) | Via docker_stats | Calcula corretamente |
| CPU throttling | **Corrigir em 0b.2** | Via Prometheus | Via docker_stats | Não coleta |
| Task monitoring | Não | Sim | Sim | Não |
| N+1 services | Possível | N/A (cache) | N/A | N/A |

## Dependências

- **Depende de:** Fase 0 (monorepo reorg) — precisa da estrutura de pastas consolidada
- **Bloqueia:** Fase 2 (security — JWT, CORS, headers no Go), Fase 3 (docs — OpenAPI do Go via swaggo/swag + Docusaurus), Fase 4 (install — 2 containers), Fase 5 (CI/CD — Go test + lint)
- **Não bloqueia:** Fase 1 (legal/community — arquivos estáticos)
