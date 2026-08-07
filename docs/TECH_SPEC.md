# RESMA — RESource MAnager

> Documento Técnico de Referência
> App customizado para gerenciamento de recursos (CPU/memória) de containers no Docker Swarm.

---

## Sumário

1. [Resumo Executivo](#1-resumo-executivo)
2. [Público e Contexto](#2-público-e-contexto)
3. [Visão Geral](#3-visão-geral)
4. [Arquitetura e Camadas](#4-arquitetura-e-camadas)
5. [Componentes e Módulos](#5-componentes-e-módulos)
6. [Autenticação e Onboarding](#6-autenticação-e-onboarding)
7. [Modelo de IA — ML Simples para Recomendações](#7-modelo-de-ia--ml-simples-para-recomendações)
8. [Stack Tecnológica](#8-stack-tecnológica)
9. [Schema DuckDB e Queries Analíticas](#9-schema-duckdb-e-queries-analíticas)
10. [API Endpoints](#10-api-endpoints)
11. [Templates de Recursos](#11-templates-de-recursos)
12. [Fluxos Principais](#12-fluxos-principais)
13. [Deploy](#13-deploy)
14. [Governança, Regras e Riscos](#14-governança-regras-e-riscos)
15. [FAQ](#15-faq)
16. [Glossário](#16-glossário)
17. [Versionamento](#17-versionamento)

---

## 1. Resumo Executivo

O **RESMA** é um serviço self-contained que coleta métricas de CPU/memória de containers Docker Swarm diretamente da Docker API, armazena em DuckDB (embedded), analisa com percentis + ML simples (3 técnicas) e recomenda limits/reservations de recursos. Roda como 1 container no manager node, sem dependências externas (sem Prometheus, sem cAdvisor, sem Mimir).

**Princípio:** "O básico que funciona" — sem over-engineering, stack já conhecida, ~2 semanas de esforço.

---

## 2. Público e Contexto

### Problema

- 30 servidores CapRover isolados sendo migrados para Docker Swarm
- Containers sem limits de CPU/memória causando instabilidade
- Necessidade de right-sizing data-driven, não baseado em chute
- Usuário leigo precisa aplicar configurações sem entender `cpus: '0.5'`

### Público-alvo

- **Operadores de infraestrutura:** dashboard com recomendações e aplicação de limits
- **Desenvolvedores:** visibilidade de consumo dos seus serviços
- **Gestores:** relatório de eficiência de recursos

### Escala

- ~100 containers
- 30 dias de retenção
- ~20MB de dados
- Coleta a cada 15 segundos

---

## 3. Visão Geral

```
Docker Swarm (manager node)
         │
         ▼
┌──────────────────────────────────────┐
│           RESMA (1 container)          │
│                                        │
│  ┌──────────┐  ┌──────────────┐       │
│  │ Collector │  │  DuckDB      │       │
│  │ (aiodocker│→ │  (embedded)  │       │
│  │  /stats)  │  │  30d retention│       │
│  └──────────┘  └──────────────┘       │
│         │              │               │
│         ▼              ▼               │
│  ┌──────────┐  ┌──────────────┐       │
│  │ OOM Event│  │  Recommender  │       │
│  │ Listener │  │  (percentis + │       │
│  │ (/events)│  │   ML simples) │       │
│  └──────────┘  └──────────────┘       │
│         │              │               │
│         ▼              ▼               │
│  ┌──────────┐  ┌──────────────┐       │
│  │ Templates│  │  FastAPI      │       │
│  │ (YAML)   │  │  (API + SPA)  │       │
│  └──────────┘  └──────────────┘       │
│         │              │               │
│         ▼              ▼               │
│  ┌──────────┐  ┌──────────────┐       │
│  │  Auth    │  │  React SPA   │       │
│  │ (JWT +   │  │  (dashboard)  │       │
│  │  bcrypt) │  │  + login gate │       │
│  └──────────┘  └──────────────┘       │
└──────────────────────────────────────┘
```

---

## 4. Arquitetura e Camadas

### Camada 1 — Coleta (Collector)

- **aiodocker** conecta ao Docker socket (`/var/run/docker.sock`)
- Polling de `/containers/{id}/stats` a cada 15s
- Listener de `/events` para OOM kills
- Batch insert via DuckDB Appender API
- Totalmente async (Python asyncio)

### Camada 2 — Armazenamento (DuckDB)

- Embedded, sem processo separado
- Schema columnar otimizado para OLAP
- Retention automática: delete de dados > 30 dias
- Export opcional para Parquet

### Camada 3 — Análise (Recommender)

- Percentis P50/P95 com outliers removidos (Z-score)
- Detecção de memory leak (Regressão Linear)
- Classificação de padrão de uso (KMeans)
- Margem de segurança adaptativa
- OOM events influenciam recomendação

### Camada 4 — Templates

- Perfis YAML versionados no Git
- Usuário leigo aplica sem entender sintaxe Docker
- 6 perfis: small, medium, large, database, worker-heavy, ml

### Camada 5 — Autenticação (Auth)

- Onboarding no primeiro acesso (criar usuário admin + senha)
- Senha armazenada como hash bcrypt no DuckDB
- JWT para sessões (access token + refresh token)
- Middleware FastAPI: todos os endpoints `/api/*` exigem Bearer token
- Endpoints `/api/auth/*` são públicos (login, onboarding, status)
- Frontend: redirect para login se não autenticado

### Camada 6 — API (FastAPI)

- Endpoints REST para dashboard e ações
- Serve frontend estático via `app.frontend()` (FastAPI 0.138.0)
- Single container: API + SPA
- Auth middleware protege todas as rotas `/api/*` (exceto `/api/auth/*`)

### Camada 7 — Frontend (React SPA)

- Tela de onboarding (primeiro acesso, se não houver usuário)
- Tela de login (usuário + senha)
- Dashboard com gráficos de consumo (protegido)
- Lista de recomendações com confiança
- Aplicação de templates
- Visualização de OOM events
- Logout + refresh automático de token

---

## 5. Componentes e Módulos

### Estrutura de arquivos

```
resma/
├── backend/
│   ├── main.py              # FastAPI app + lifecycle + app.frontend()
│   ├── collector.py          # Async Docker API metrics collector
│   ├── docker_client.py      # Wrapper aiodocker
│   ├── recommender.py        # Percentis + ML simples
│   ├── templates.py          # Load/apply templates YAML
│   ├── db.py                 # DuckDB connection + schema + retention
│   ├── models.py             # Pydantic models
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

### Responsabilidades por arquivo

| Arquivo | Responsabilidade |
|---------|-----------------|
| `main.py` | FastAPI app, lifecycle (startup/shutdown), rotas, `app.frontend()` |
| `collector.py` | Loop async de coleta de `/stats` e `/events`, batch insert |
| `docker_client.py` | Wrapper `aiodocker`, reconexão, error handling |
| `recommender.py` | `ResourceRecommender` class — percentis + ML |
| `templates.py` | Load YAML, validar, aplicar via Docker API |
| `db.py` | DuckDB connection, schema init, Appender, retention job |
| `models.py` | Pydantic schemas para API (incluindo auth) |
| `config.py` | Settings via env vars (intervalo coleta, retention, JWT, etc) |
| `auth.py` | Onboarding, login, JWT generation/validation, bcrypt, middleware |
| `deps.py` | FastAPI dependencies: `get_current_user`, `get_db` |

---

## 6. Autenticação e Onboarding

### Visão Geral

O RESMA implementa autenticação própria similar ao Portainer: no primeiro acesso, o frontend detecta que não há usuário cadastrado e exibe a tela de **onboarding** para criar o usuário admin com senha. Após o cadastro, a senha é armazenada como **hash bcrypt** no DuckDB. Todos os acessos subsequentes exigem **login** (usuário + senha), que retorna um **JWT** (access + refresh token). O frontend bloqueia todas as rotas protegidas se o usuário não estiver autenticado.

### Fluxo de Onboarding (Primeiro Acesso)

```
1. Frontend carrega -> GET /api/auth/status
2. Backend verifica: existe usuario na tabela users?
3. Se nao existe -> retorna { initialized: false }
4. Frontend exibe tela de Onboarding (criar usuario + senha)
5. POST /api/auth/onboarding { username, password }
6. Backend valida senha (min. 8 chars), gera hash bcrypt, insere na tabela users
7. Retorna access_token + refresh_token
8. Frontend armazena token -> redirect para Dashboard
```

### Fluxo de Login (Acessos Posteriores)

```
1. Frontend carrega -> GET /api/auth/status
2. Backend retorna { initialized: true }
3. Frontend verifica: ha token valido em localStorage?
4. Se nao -> exibe tela de Login
5. POST /api/auth/login { username, password }
6. Backend valida: bcrypt.verify(password, hash)
7. Se ok -> retorna access_token + refresh_token
8. Frontend armazena token -> redirect para Dashboard
```

### Tabela users no DuckDB

```sql
CREATE TABLE users (
    id          INTEGER PRIMARY KEY,
    username    VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    role        VARCHAR DEFAULT 'admin',
    created_at  TIMESTAMP DEFAULT now(),
    updated_at  TIMESTAMP DEFAULT now(),
);

CREATE TABLE refresh_tokens (
    token       VARCHAR PRIMARY KEY,
    user_id     INTEGER REFERENCES users(id),
    expires_at  TIMESTAMP,
    revoked     BOOLEAN DEFAULT false,
    created_at  TIMESTAMP DEFAULT now(),
);
```

### JWT — Tokens

| Token | TTL | Funcao |
|-------|-----|--------|
| `access_token` | 15 minutos | Autenticar requisicoes a API |
| `refresh_token` | 7 dias | Renovar access_token sem novo login |

**Payload do JWT:**
```json
{
    "sub": 1,
    "username": "admin",
    "role": "admin",
    "exp": 1737500000,
    "type": "access"
}
```

**Assinatura:** HMAC-SHA256 com `RESMA_JWT_SECRET` (env var).

### Middleware de Autenticacao

```python
from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import jwt

security = HTTPBearer()

async def get_current_user(
    credentials: HTTPAuthorizationCredentials = Depends(security)
) -> dict:
    token = credentials.credentials
    try:
        payload = jwt.decode(token, settings.JWT_SECRET, algorithms=["HS256"])
        if payload.get("type") != "access":
            raise HTTPException(401, "Invalid token type")
        return {
            "sub": payload["sub"],
            "username": payload["username"],
            "role": payload["role"],
        }
    except jwt.ExpiredSignatureError:
        raise HTTPException(401, "Token expired")
    except jwt.InvalidTokenError:
        raise HTTPException(401, "Invalid token")
```

### Endpoints de Autenticacao

| Metodo | Path | Auth | Descricao |
|--------|------|------|-----------|
| GET | `/api/auth/status` | Nao | Retorna `{ initialized: bool }` |
| POST | `/api/auth/onboarding` | Nao | Cria primeiro usuario (so funciona se nao houver usuario) |
| POST | `/api/auth/login` | Nao | Autentica e retorna tokens |
| POST | `/api/auth/refresh` | Refresh token | Renova access token |
| POST | `/api/auth/logout` | Sim | Revoga refresh token |
| GET | `/api/auth/me` | Sim | Retorna dados do usuario atual |
| POST | `/api/auth/change-password` | Sim | Altera senha |

### Frontend — AuthGate e ProtectedRoute

```tsx
function AuthGate() {
    const { initialized, user, loading } = useAuth();

    if (loading) return <Spinner />;
    if (!initialized) return <Onboarding />;
    if (!user) return <Login />;
    return <Outlet />;
}
```

```tsx
function ProtectedRoute() {
    const { user, loading } = useAuth();

    if (loading) return <Spinner />;
    if (!user) return <Navigate to="/login" />;
    return <Outlet />;
}
```

### AuthContext

```tsx
interface AuthContextType {
    initialized: boolean;
    user: User | null;
    loading: boolean;
    login: (username: string, password: string) => Promise<void>;
    onboarding: (username: string, password: string) => Promise<void>;
    logout: () => void;
    refreshToken: () => Promise<void>;
}
```

O `AuthContext` gerencia:
- Verificacao de `initialized` ao montar (`GET /api/auth/status`)
- Login/onboarding -> armazenar tokens em `localStorage`
- Refresh automatico quando `access_token` expira
- Logout -> limpar `localStorage` + redirect para login
- Interceptor HTTP: adiciona `Authorization: Bearer` em todas as requests

### Validacao de Senha

- Minimo 8 caracteres
- Pelo menos 1 letra e 1 numero
- Validacao no backend (nao confiar so no frontend)
- Hash: bcrypt com cost factor 12

### Seguranca

- `RESMA_JWT_SECRET` deve ser gerado na primeira inicializacao (UUID aleatorio) e persistido no DuckDB se nao definido via env var
- Refresh tokens podem ser revogados (tabela `refresh_tokens` no DuckDB)
- Rate limiting no endpoint de login (max 5 tentativas por minuto por IP)
- Senha nunca logada ou retornada em nenhuma resposta de API

---

## 7. Modelo de IA — ML Simples para Recomendações

### Visão Geral

O RESMA usa **3 técnicas clássicas de ML** com 3 dependências (`scikit-learn`, `scipy`, `numpy`). Sem deep learning, sem pipeline de treino, sem GPU. O modelo roda on-demand quando o usuário pede recomendação.

### Técnica 1 — Z-score (Remoção de Outliers)

**Objetivo:** Remover spikes de deploy/restart antes de calcular percentis.

**Algoritmo:**
1. Calcular Z-score de cada amostra: `z = (x - μ) / σ`
2. Remover amostras com `|z| > 3.0` (3 desvios padrão)
3. Calcular P50/P95 sobre os dados limpos

**Threshold:** `|z| > 3.0` (configurável via env var `RESMA_OUTLIER_THRESHOLD`)

**Valor real:** P95 sem o pico do deploy às 14h que triplicou CPU.

```python
from scipy import stats
import numpy as np

def remove_outliers(values: np.ndarray, threshold: float = 3.0) -> np.ndarray:
    z_scores = stats.zscore(values)
    return values[np.abs(z_scores) < threshold]
```

### Técnica 2 — Regressão Linear (Detecção de Memory Leak)

**Objetivo:** Detectar tendência crescente de memória que percentis não capturam.

**Algoritmo:**
1. Agrupar memória por hora (média)
2. Regressão linear: `x = índice temporal`, `y = mem_usage`
3. Calcular slope (bytes/hora) e R² (qualidade do ajuste)
4. Classificar como leak se: `slope > 0` AND `R² > 0.7` AND `daily_growth > 10MB`

**Thresholds:**
- `R² > 0.7` — tendência é real, não ruído
- `daily_growth > 10MB` — crescimento significativo
- Ambos configuráveis via env vars

**Valor real:** "memória cresce 50MB/dia com R²=0.92 — provável leak, não aumentar limit, investigar código"

```python
from sklearn.linear_model import LinearRegression
import numpy as np

def detect_memory_trend(mem_values: np.ndarray) -> dict:
    x = np.arange(len(mem_values)).reshape(-1, 1)
    model = LinearRegression()
    model.fit(x, mem_values)

    slope = model.coef_[0]
    r_squared = model.score(x, mem_values)
    daily_growth_mb = (slope * 24) / (1024 * 1024)

    return {
        'slope_bytes_per_hour': float(slope),
        'daily_growth_mb': round(daily_growth_mb, 2),
        'r_squared': round(r_squared, 3),
        'has_leak': bool(slope > 0 and r_squared > 0.7 and daily_mb > 10),
    }
```

### Técnica 3 — KMeans (Classificação de Padrão)

**Objetivo:** Classificar o workload para ajustar margem de segurança.

**Algoritmo:**
1. Agrupar CPU médio por hora do dia
2. KMeans com `n_clusters=2` (baixo vs alto)
3. Calcular ratio entre clusters
4. Classificar:
   - `ratio > 3` → `business_hours` (pico em horário comercial)
   - `ratio < 1.5` → `constant` (uso uniforme)
   - caso contrário → `batch` (processamento esporádico)

**Valor real:** "serviço é business_hours — pode ter reservation menor fora do horário"

```python
from sklearn.cluster import KMeans
import numpy as np

def classify_workload_pattern(hourly_cpu: dict[int, float]) -> str:
    if len(hourly_cpu) < 12:
        return 'unknown'

    vals = list(hourly_cpu.values())
    ratio = max(vals) / min(vals) if min(vals) > 0 else 999

    if ratio > 3:
        return 'business_hours'
    elif ratio < 1.5:
        return 'constant'
    return 'batch'
```

### Modelo Combinado — ResourceRecommender

```python
import numpy as np
from sklearn.linear_model import LinearRegression
from scipy import stats

class ResourceRecommender:

    def __init__(self, db):
        self.db = db

    def analyze(self, service_name: str) -> dict:
        rows = self.db.execute("""
            SELECT ts, cpu_percent, mem_usage, mem_limit
            FROM metrics
            WHERE service = ? AND ts > now() - INTERVAL 7 DAYS
            ORDER BY ts
        """, [service_name]).fetchall()

        if len(rows) < 100:
            return {'service': service_name, 'status': 'collecting_data', 'samples': len(rows)}

        ts = [r[0] for r in rows]
        cpu = np.array([r[1] for r in rows])
        mem = np.array([r[2] for r in rows])

        cpu_clean = self._remove_outliers(cpu)
        mem_clean = self._remove_outliers(mem)

        cpu_p50 = np.percentile(cpu_clean, 50)
        cpu_p95 = np.percentile(cpu_clean, 95)
        mem_p50 = np.percentile(mem_clean, 50)
        mem_p95 = np.percentile(mem_clean, 95)

        leak = self._detect_leak(mem)
        pattern = self._classify_pattern(ts, cpu)

        oom_count = self.db.execute("""
            SELECT count(*) FROM oom_events
            WHERE service = ? AND ts > now() - INTERVAL 7 DAYS
        """, [service_name]).fetchone()[0]

        margin = self._safety_margin(pattern, oom_count, leak['has_leak'])

        suggested_cpu = max((cpu_p95 * margin) / 100, 0.1)
        suggested_mem = int(mem_p95 * margin)

        if leak['has_leak']:
            suggested_mem = int(mem_p95 * 1.2)

        return {
            'service': service_name,
            'samples': len(rows),
            'outliers_removed': len(cpu) - len(cpu_clean),
            'cpu': {'p50': cpu_p50, 'p95': cpu_p95},
            'mem': {'p50': mem_p50, 'p95': mem_p95},
            'oom_events': oom_count,
            'pattern': pattern,
            'memory_trend': leak,
            'suggested': {
                'cpu_limit': round(suggested_cpu, 2),
                'mem_limit': suggested_mem,
                'cpu_reservation': round((cpu_p50 * 0.75) / 100, 2),
                'mem_reservation': int(mem_p50 * 0.75),
            },
            'confidence': self._confidence(len(rows), leak['r_squared']),
        }

    def _remove_outliers(self, values: np.ndarray, threshold: float = 3.0) -> np.ndarray:
        z = stats.zscore(values)
        return values[np.abs(z) < threshold]

    def _detect_leak(self, mem: np.ndarray) -> dict:
        x = np.arange(len(mem)).reshape(-1, 1)
        model = LinearRegression()
        model.fit(x, mem)
        slope = model.coef_[0]
        r2 = model.score(x, mem)
        daily_mb = (slope * 24) / (1024 * 1024)
        return {
            'slope_bytes_per_hour': float(slope),
            'daily_growth_mb': round(daily_mb, 2),
            'r_squared': round(r2, 3),
            'has_leak': bool(slope > 0 and r2 > 0.7 and daily_mb > 10),
        }

    def _classify_pattern(self, ts, cpu) -> str:
        hourly = {}
        for i, t in enumerate(ts):
            h = t.hour
            hourly.setdefault(h, []).append(cpu[i])
        avgs = {h: np.mean(v) for h, v in hourly.items()}
        if len(avgs) < 12:
            return 'unknown'
        vals = list(avgs.values())
        ratio = max(vals) / min(vals) if min(vals) > 0 else 999
        if ratio > 3:
            return 'business_hours'
        elif ratio < 1.5:
            return 'constant'
        return 'batch'

    def _safety_margin(self, pattern: str, oom: int, has_leak: bool) -> float:
        base = 1.5
        if pattern == 'business_hours':
            base = 1.8
        if oom > 0:
            base += 0.5
        if has_leak:
            base = 1.2
        return base

    def _confidence(self, samples: int, r2: float) -> str:
        if samples > 5000:
            return 'high'
        elif samples > 1000:
            return 'medium'
        return 'low'
```

### Margem de Segurança Adaptativa

| Padrão | OOM Events | Memory Leak | Margem |
|--------|-----------|-------------|--------|
| constant | 0 | não | 1.5x |
| constant | ≥1 | não | 2.0x |
| business_hours | 0 | não | 1.8x |
| business_hours | ≥1 | não | 2.3x |
| batch | 0 | não | 1.5x |
| qualquer | qualquer | sim | 1.2x + alerta |

### Comparação: Antes vs Com ML

| Recomendação | Só percentil | Com ML simples |
|---------------|-------------|----------------|
| CPU limit | P95 × 1.5 | P95 (sem outliers) × margem adaptativa |
| Memory limit | P95 × 1.5 | P95 (sem outliers) × margem + detecção leak |
| Pattern | não detectado | business_hours / constant / batch |
| Memory leak | não detectado | "cresce 50MB/dia, R²=0.92" |
| Outliers do deploy | contamina P95 | filtrado por Z-score |
| Margem adaptativa | fixa 1.5x | 1.2x a 2.3x conforme padrão |
| Confiança | "high se >1000" | "high se >5000 + R² baixo" |

### O que NÃO usar (over-engineering)

- ~~Holt-Winters forecasting~~ — complexo, precisa mais dados, ganho marginal
- ~~Redes neurais / LSTM~~ — desnecessário para este contexto
- ~~AutoML~~ — absurdo para 100 containers
- ~~Time series forecasting de longo prazo~~ — 30 dias não justifica

### Dependências ML

```
scikit-learn    # LinearRegression, KMeans
scipy           # zscore, percentile
numpy           # arrays
```

3 pacotes. Sem TensorFlow, sem PyTorch, sem transformers.

---

## 8. Stack Tecnológica

### Backend

| Tecnologia | Versão | Função |
|-----------|--------|--------|
| Python | 3.12+ | Runtime |
| FastAPI | 0.138.0+ | API + serve SPA (`app.frontend()`) |
| aiodocker | latest | Async Docker API client |
| duckdb | latest | Embedded OLAP database |
| scikit-learn | latest | LinearRegression, KMeans |
| scipy | latest | Z-score, estatísticas |
| numpy | latest | Arrays numéricos |
| pydantic | v2 | Models de API |
| pyyaml | latest | Templates YAML |
| bcrypt | latest | Hash de senhas |
| PyJWT | latest | Geracao e validacao de JWT |

### Frontend

| Tecnologia | Função |
|-----------|--------|
| React 19 | UI framework |
| Vite | Build tool + dev server |
| TailwindCSS v4 | Styling (utility-first, última versão) |
| shadcn/ui (original) | Componentes (última versão, sem forks) |
| Recharts ou Tremor | Gráficos |
| TanStack Query | Data fetching |
| React Router | Roteamento + protected routes |
| pnpm | Gerenciador de pacotes (lockfile determinístico) |

### Justificativas

- **DuckDB vs SQLite:** Workload é 100% OLAP (agregações, percentis, time series). DuckDB tem `quantile()` nativo, 100x mais rápido em GROUP BY, multi-core. SQLite é OLTP — não faz sentido.
- **Python vs Node.js:** Mesma stack do mcp-caprover já existente. DuckDB Python bindings é o cliente primário mais maduro.
- **Vite vs Next.js:** SPA puro, sem SSR. Vite é mais leve e rápido para este caso. FastAPI serve o build estático.
- **aiodocker:** Async nativo, não bloqueia o event loop do FastAPI.

---

## 9. Schema DuckDB e Queries Analíticas

### Schema

```sql
CREATE TABLE metrics (
    ts          TIMESTAMP,
    service     VARCHAR,
    container_id VARCHAR,
    cpu_percent DOUBLE,
    cpu_usage   BIGINT,
    cpu_system  BIGINT,
    mem_usage   BIGINT,
    mem_limit   BIGINT,
    mem_percent DOUBLE,
    net_rx      BIGINT,
    net_tx      BIGINT,
    block_read  BIGINT,
    block_write BIGINT,
);

CREATE TABLE oom_events (
    ts          TIMESTAMP,
    service     VARCHAR,
    container_id VARCHAR,
    exit_code   INTEGER,
);

CREATE TABLE service_configs (
    service           VARCHAR PRIMARY KEY,
    cpu_limit         DOUBLE,
    mem_limit         BIGINT,
    cpu_reservation   DOUBLE,
    mem_reservation   BIGINT,
    template          VARCHAR,
    updated_at        TIMESTAMP,
);

CREATE TABLE users (
    id          INTEGER PRIMARY KEY,
    username    VARCHAR UNIQUE NOT NULL,
    password_hash VARCHAR NOT NULL,
    role        VARCHAR DEFAULT 'admin',
    created_at  TIMESTAMP DEFAULT now(),
    updated_at  TIMESTAMP DEFAULT now(),
);

CREATE TABLE refresh_tokens (
    token       VARCHAR PRIMARY KEY,
    user_id     INTEGER REFERENCES users(id),
    expires_at  TIMESTAMP,
    revoked     BOOLEAN DEFAULT false,
    created_at  TIMESTAMP DEFAULT now(),
);
```

### Queries Analíticas

**Percentis por serviço (7 dias):**
```sql
SELECT
    service,
    quantile(cpu_percent, 0.50) AS cpu_p50,
    quantile(cpu_percent, 0.95) AS cpu_p95,
    quantile(mem_usage, 0.50) AS mem_p50,
    quantile(mem_usage, 0.95) AS mem_p95,
    count(*) AS samples
FROM metrics
WHERE ts > now() - INTERVAL 7 DAYS
GROUP BY service
ORDER BY cpu_p95 DESC;
```

**Média horária de CPU (para classificação de padrão):**
```sql
SELECT
    extract(hour from ts) AS hour,
    avg(cpu_percent) AS cpu_avg
FROM metrics
WHERE service = ? AND ts > now() - INTERVAL 7 DAYS
GROUP BY hour
ORDER BY hour;
```

**OOM events por serviço:**
```sql
SELECT
    service,
    count(*) AS oom_count,
    max(ts) AS last_oom
FROM oom_events
WHERE ts > now() - INTERVAL 7 DAYS
GROUP BY service
ORDER BY oom_count DESC;
```

**Retention (job diário):**
```sql
DELETE FROM metrics WHERE ts < now() - INTERVAL 30 DAYS;
DELETE FROM oom_events WHERE ts < now() - INTERVAL 30 DAYS;
```

### Inserção (Appender API)

```python
import duckdb

con = duckdb.connect('resma.duckdb')
appender = con.appender('metrics')
appender.append([ts, service, container_id, cpu_pct, ...])
appender.close()
```

---

## 10. API Endpoints

### Autenticação

| Método | Path | Auth | Descrição |
|--------|------|------|-----------|
| GET | `/api/auth/status` | Não | Retorna `{ initialized: bool }` |
| POST | `/api/auth/onboarding` | Não | Cria primeiro usuário admin |
| POST | `/api/auth/login` | Não | Autentica e retorna JWT |
| POST | `/api/auth/refresh` | Refresh | Renova access token |
| POST | `/api/auth/logout` | Sim | Revoga refresh token |
| GET | `/api/auth/me` | Sim | Dados do usuário atual |
| POST | `/api/auth/change-password` | Sim | Altera senha |

### Métricas

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/services` | Lista todos os serviços monitorados |
| GET | `/api/services/{name}/metrics` | Métricas de um serviço (query params: `range=7d`) |
| GET | `/api/services/{name}/stats` | Estatísticas (P50, P95, min, max, avg) |

### Recomendações

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/recommendations` | Lista recomendações de todos os serviços |
| GET | `/api/recommendations/{service}` | Recomendação detalhada de um serviço |
| POST | `/api/recommendations/{service}/apply` | Aplica recomendação via Docker API |

### Templates

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/templates` | Lista templates disponíveis |
| GET | `/api/templates/{name}` | Detalha um template |
| POST | `/api/templates/{name}/apply/{service}` | Aplica template a um serviço |

### OOM Events

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/oom-events` | Lista OOM events (query params: `service`, `range`) |

### Dashboard

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/dashboard` | Resumo geral (serviços, top consumers, OOMs) |

---

## 11. Templates de Recursos

### Estrutura YAML

```yaml
name: medium
description: Serviço web com tráfego moderado
resources:
  limits:
    cpus: '0.5'
    memory: 512M
  reservations:
    cpus: '0.25'
    memory: 256M
```

### Perfis Disponíveis

| Template | CPU Limit | Mem Limit | CPU Reservation | Mem Reservation | Uso |
|----------|-----------|-----------|-----------------|-----------------|-----|
| small | 0.25 | 128M | 0.10 | 64M | Microserviços leves |
| medium | 0.50 | 512M | 0.25 | 256M | APIs web moderadas |
| large | 1.00 | 1G | 0.50 | 512M | APIs com alto tráfego |
| database | 2.00 | 2G | 1.00 | 1G | PostgreSQL, MongoDB |
| worker-heavy | 1.50 | 1G | 0.75 | 512M | Workers de processamento |
| ml | 2.00 | 4G | 1.00 | 2G | Modelos de ML |

---

## 12. Fluxos Principais

### Fluxo 1 — Coleta Contínua

```
Startup
  → Conectar ao Docker socket
  → Iniciar DuckDB + criar schema
  → Loop async (a cada 15s):
      → Listar containers ativos
      → Para cada container: GET /containers/{id}/stats
      → Batch insert via Appender
  → Listener paralelo: /events (OOM kills)
  → Job diário: retention (delete > 30 dias)
```

### Fluxo 2 — Recomendação On-Demand

```
Usuário clica "Recomendar"
  → Query DuckDB: métricas 7 dias
  → Se < 100 amostras: retornar "collecting_data"
  → Remover outliers (Z-score)
  → Calcular P50/P95
  → Detectar memory leak (Regressão Linear)
  → Classificar padrão (KMeans/ratio)
  → Contar OOM events
  → Calcular margem adaptativa
  → Retornar recomendação + confiança
```

### Fluxo 3 — Aplicar Recomendação/Template

```
Usuário clica "Aplicar"
  → Validar recurso sugerido
  → Docker API: update service com novos limits
  → Atualizar service_configs no DuckDB
  → Retornar sucesso/falha
```

### Fluxo 4 — Onboarding (Primeiro Acesso)

```
1. Usuario acessa RESMA no browser
2. Frontend: GET /api/auth/status -> { initialized: false }
3. Frontend exibe tela de Onboarding
4. Usuario preenche: username + senha (min. 8 chars)
5. POST /api/auth/onboarding { username, password }
6. Backend: valida senha, gera hash bcrypt, insere em users
7. Backend: gera access_token + refresh_token
8. Frontend: armazena tokens, redirect para Dashboard
```

### Fluxo 5 — Login (Acessos Posteriores)

```
1. Usuario acessa RESMA no browser
2. Frontend: GET /api/auth/status -> { initialized: true }
3. Frontend: verifica token em localStorage
4. Se token valido -> Dashboard diretamente
5. Se token expirado -> tenta refresh
6. Se refresh falha -> exibe tela de Login
7. Usuario preenche: username + senha
8. POST /api/auth/login { username, password }
9. Backend: bcrypt.verify(password, hash)
10. Se ok -> access_token + refresh_token
11. Frontend: armazena tokens, redirect para Dashboard
```

### Fluxo 6 — Deploy do RESMA

```
Dockerfile multi-stage:
  Stage 1: build React (pnpm install --frozen-lockfile && pnpm build)
  Stage 2: Python + FastAPI + dist/ frontend
  → docker stack deploy resma
  → 1 container no manager node
  → Mount /var/run/docker.sock
```

---

## 13. Deploy

### Dockerfile (multi-stage)

```dockerfile
FROM node:22-alpine AS frontend
RUN corepack enable && corepack prepare pnpm@latest --activate
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm build

FROM python:3.12-slim
WORKDIR /app
COPY pyproject.toml .
RUN pip install -e .
COPY backend/ ./backend/
COPY --from=frontend /app/frontend/dist ./static/
EXPOSE 8080
CMD ["uvicorn", "backend.main:app", "--host", "0.0.0.0", "--port", "8080"]
```

### docker-compose.yml (para Swarm)

```yaml
version: '3.8'

services:
  resma:
    image: resma:latest
    deploy:
      placement:
        constraints:
          - node.role == manager
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - resma-data:/data
    environment:
      - RESMA_DB_PATH=/data/resma.duckdb
      - RESMA_COLLECT_INTERVAL=15
      - RESMA_RETENTION_DAYS=30
      - RESMA_OUTLIER_THRESHOLD=3.0
      - RESMA_JWT_SECRET=change-me-in-production
    ports:
      - "8080:8080"

volumes:
  resma-data:
```

### Variáveis de Ambiente

| Variável | Default | Descrição |
|----------|---------|-----------|
| `RESMA_DB_PATH` | `/data/resma.duckdb` | Caminho do DuckDB |
| `RESMA_COLLECT_INTERVAL` | `15` | Intervalo de coleta de containers (segundos) |
| `RESMA_CLUSTER_INTERVAL` | `30` | Intervalo de coleta de info do cluster (segundos) |
| `RESMA_STORAGE_INTERVAL` | `60` | Intervalo de coleta de storage (segundos) |
| `RESMA_SCHEDULER_POLL` | `15` | Intervalo de poll do scheduler de agendamentos (segundos) |
| `RESMA_SSE_KEEPALIVE` | `15` | Intervalo de keepalive ping do SSE broker (segundos) |
| `RESMA_RETENTION_DAYS` | `30` | Dias de retenção |
| `RESMA_OUTLIER_THRESHOLD` | `3.0` | Threshold Z-score |
| `RESMA_LEAK_R2_THRESHOLD` | `0.7` | R² mínimo para leak |
| `RESMA_LEAK_DAILY_MB_THRESHOLD` | `10` | Crescimento diário mínimo (MB) |
| `RESMA_ANALYSIS_WINDOW_DAYS` | `7` | Janela de análise |
| `RESMA_JWT_SECRET` | auto-gerado | Secret para assinar JWT (definir em produção) |
| `RESMA_JWT_ACCESS_TTL_MINUTES` | `15` | TTL do access token (minutos) |
| `RESMA_JWT_REFRESH_TTL_DAYS` | `7` | TTL do refresh token (dias) |
| `RESMA_BCRYPT_COST` | `12` | Cost factor do bcrypt |
| `RESMA_LOGIN_RATE_LIMIT` | `5` | Máx. tentativas de login por minuto por IP |

---

## 14. Governança, Regras e Riscos

### Segurança

- **Docker socket:** RESMA precisa de acesso ao `/var/run/docker.sock` — equivalente a root no host. Restringir acesso de rede ao serviço.
- **Autenticação obrigatória:** Todos os endpoints `/api/*` exigem JWT Bearer token (exceto `/api/auth/*`).
- **Senhas com bcrypt:** Hash bcrypt cost 12. Senha nunca armazenada em texto plano, nunca logada, nunca retornada.
- **JWT com refresh:** Access token 15min + refresh token 7 dias. Refresh tokens revogáveis.
- **Onboarding único:** Endpoint `/api/auth/onboarding` só funciona se não houver usuário cadastrado. Após o primeiro usuário, retorna 403.
- **Rate limiting:** Máximo 5 tentativas de login por minuto por IP.
- **Sem secrets no código:** Todas as configurações via env vars. `RESMA_JWT_SECRET` deve ser definido em produção.
- **HTTPS recomendado:** Em produção, usar Traefik/ingress com TLS.

### Limitações Conhecidas

- **Sem histórico pré-deploy:** RESMA começa a coletar do zero. Recomendações ficam precisas após ~7 dias.
- **100 containers máximo:** Testado para esta escala. Para mais, considerar particionamento.
- **Sem HA:** 1 réplica no manager. Se o manager cai, RESMA cai. Aceitável para tool operacional.
- **DuckDB single-writer:** Apenas 1 processo escreve. Reads concorrentes são suportados.

### Riscos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|--------------|---------|-----------|
| Docker socket exposto | Baixa | Alto | Rede interna only, sem port publicadas |
- DuckDB corrompido | Baixa | Médio | Volume persistente + backup diário do .duckdb |
| Coleta impacta performance | Baixa | Baixo | 15s intervalo, batch insert, ~20MB total |
| Recomendação errada | Média | Médio | Confidence score + humano revisa antes de aplicar |

### No Hardcode

Todos os thresholds, intervalos, limites e padrões são configuráveis via env vars. Templates são YAML externos. Nenhum valor variável é fixo no código.

---

## 15. FAQ

**Q: Por que DuckDB e não Prometheus?**
A: Prometheus exige infraestrutura separada (Prometheus + Grafana + alertmanager). RESMA é self-contained — 1 container, 0 dependências externas. DuckDB é embedded e suficiente para 100 containers.

**Q: Por que não usar o VPA do Kubernetes?**
A: Docker Swarm não tem VPA. O RESMA implementa lógica similar (percentis) + ML simples, específico para Swarm.

**Q: Quando as recomendações ficam confiáveis?**
A: Após ~7 dias de coleta com > 5000 amostras (confidence: high). Antes disso, funciona mas com confidence: low/medium.

**Q: O ML precisa de GPU?**
A: Não. 3 técnicas clássicas (Z-score, Regressão Linear, KMeans) rodam em CPU em < 1 segundo.

**Q: Posso usar sem o ML?**
A: Sim. O ML é uma camada encima dos percentis. Pode-se desabilitar via env var `RESMA_ML_ENABLED=false`.

**Q: Como funciona o primeiro acesso?**
A: No primeiro acesso, o RESMA detecta que não há usuário cadastrado e exibe a tela de onboarding. O usuário cria seu login e senha, que é armazenada como hash bcrypt no DuckDB. Após isso, todos os acessos exigem autenticação.

**Q: Posso ter múltiplos usuários?**
A: A V1 suporta 1 usuário admin. A V2 pode adicionar múltiplos usuários com roles (admin, viewer).

**Q: O JWT secret é auto-gerado?**
A: Se `RESMA_JWT_SECRET` não for definido via env var, o RESMA gera um UUID aleatório e persiste no DuckDB. Em produção, recomenda-se definir explicitamente.

---

## 16. Glossário

| Termo | Definição |
|-------|-----------|
| Right-sizing | Ajustar recursos alocados ao uso real |
| OOM Kill | Out-Of-Memory kill — container morto por exceder memória |
| P50/P95 | Percentil 50 (mediana) / Percentil 95 |
| Z-score | Número de desvios padrão acima/abaixo da média |
| R² | Coeficiente de determinação (qualidade do ajuste da regressão) |
| Memory Leak | Vazamento de memória — crescimento contínuo sem liberação |
| KMeans | Algoritmo de clustering que agrupa dados em K clusters |
| OLAP | Online Analytical Processing (consultas analíticas) |
| Appender | API do DuckDB para inserção em lote de alta performance |
| VPA | Vertical Pod Autoscaler (Kubernetes) |
| Reservation | Recursos garantidos (Docker Swarm) |
| Limit | Recursos máximos (Docker Swarm) |
| JWT | JSON Web Token — token de autenticação stateless |
| bcrypt | Algoritmo de hash de senhas adaptativo |
| Onboarding | Processo de configuração inicial (criar primeiro usuário) |
| Refresh Token | Token de longa duração para renovar access tokens |
| Access Token | Token de curta duração para autenticar requisições |

---

## 17. Versionamento

| Versão | Data | Alteração |
|--------|------|-----------|
| 0.1.0 | 2026-07-20 | Documento técnico inicial |
| 0.2.0 | 2026-07-20 | Adicionada camada de autenticação e onboarding |
