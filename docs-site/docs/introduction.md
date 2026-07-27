# Introduction

**RESMA** (RESource MAnager) is an open-source tool for managing and optimizing
CPU and memory resources of containers running in **Docker Swarm**. It collects
metrics continuously, applies statistical analysis and machine learning to
recommend optimal resource limits, and detects memory leaks before they cause
out-of-memory kills.

---

## What is RESMA?

Docker Swarm lets you set per-container resource reservations and limits
(`cpus`, `memory`). In practice, teams either leave these unset (causing noisy
neighbors and unpredictable scheduling) or set them once and never revisit them.
RESMA automates the feedback loop:

1. **Collect** — A Go collector polls the Docker API for per-container CPU and
   memory stats at a configurable interval (default 15s) and stores them as
   time-series in an embedded DuckDB database.
2. **Analyze** — A Python ML sidecar (scikit-learn + scipy) runs statistical
   models on the collected data: percentile analysis, linear regression for
   trend detection, and clustering for usage pattern classification.
3. **Recommend** — Based on the analysis, RESMA generates concrete
   recommendations: "Service `api` should set `memory_limit` to 512MB (current
   p95: 420MB, trend: +2.1%/week)".
4. **Alert** — When a memory leak signature is detected (sustained linear growth
   with R² > 0.8), RESMA emits an alert via SSE and the API.

---

## Use Case

Consider a Docker Swarm cluster running 12 microservices across 5 nodes. Without
resource limits, a single service experiencing a memory leak can consume all
available RAM on a node, causing the kernel OOM-killer to terminate random
processes — including unrelated services.

With RESMA deployed as a Swarm service:

- The collector gathers memory usage every 15 seconds for every container.
- After 24 hours of data, the ML sidecar identifies that `payment-service` has a
  memory growth trend of +5MB/hour with R² = 0.94 — a classic leak signature.
- RESMA sends an SSE alert to the dashboard and exposes it via the API.
- The operator sees the alert, sets a temporary `memory_limit` based on RESMA's
  recommendation (e.g., 768MB), and schedules a fix for the leaking service.
- Over time, RESMA's recommendations help right-size all services, freeing
  cluster capacity for additional workloads.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Docker Swarm                          │
│                                                          │
│  ┌──────────┐    metrics     ┌──────────────────────┐   │
│  │  Docker   │◄──────────────│  RESMA API (Go)      │   │
│  │  Engine   │   Docker SDK   │  :8080               │   │
│  │  (nodes)  │                │                      │   │
│  └──────────┘                │  ┌────────────────┐  │   │
│                               │  │  DuckDB (embed)│  │   │
│                               │  └────────────────┘  │   │
│                               │  ┌────────────────┐  │   │
│                               │  │  SSE Broker    │  │   │
│                               │  └────────────────┘  │   │
│                               └──────────┬───────────┘   │
│                                          │ HTTP          │
│                               ┌──────────▼───────────┐   │
│                               │  ML Sidecar (Python) │   │
│                               │  :8081 (internal)    │   │
│                               │  scikit-learn+scipy  │   │
│                               └──────────────────────┘   │
│                                          │               │
│  ┌──────────────────────────┐            │               │
│  │  Frontend (React+Vite)   │◄─── SSE ───┘               │
│  │  :5173 (dev) / built     │    /api/* (JWT)            │
│  └──────────────────────────┘                            │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Components

| Component | Technology | Port | Description |
|-----------|-----------|------|-------------|
| **API** | Go 1.26 + net/http + DuckDB | 8080 | REST API, metrics collection, SSE streaming, auth |
| **ML Sidecar** | Python 3.12 + FastAPI + scikit-learn | 8081 | Statistical analysis, ML models, leak detection (internal only) |
| **Frontend** | React 19 + Vite 6 + TailwindCSS 4 | 5173 (dev) | Dashboard with real-time SSE updates |
| **Database** | DuckDB (embedded) | — | Time-series storage, no external DB needed |

### API Surface

RESMA exposes two distinct API surfaces:

- **`/api/*`** — Internal/UI endpoints. Authenticated via JWT. The frontend
  consumes these. Not versioned.
- **`/api/v1/*`** — Public API. Authenticated via API key with scopes. Documented
  with OpenAPI (swaggo). Intended for automation, integrations, and external
  consumers.
- **`/api/sse/*`** — Server-Sent Events streaming. Authenticated via HttpOnly
  cookie (browser) or `Authorization` header (non-browser clients).
- **`/health`, `/ready`** — Infrastructure endpoints, no auth.

---

## Tech Stack

### API (Go)

- **Go 1.26** with the standard library `net/http` server (no framework)
- **DuckDB** via `go-duckdb` (CGO) for embedded time-series storage
- **Docker SDK** (`moby/moby`) for container inspection and stats collection
- **golang-jwt** for JWT issuance and validation
- **bcrypt** for password hashing
- **swaggo** for OpenAPI documentation generation

### ML Sidecar (Python)

- **Python 3.12** with **FastAPI** (minimal, internal-only)
- **scikit-learn** for regression and clustering models
- **scipy** for statistical tests (Mann-Kendall, linear regression)
- **numpy** for numerical operations

### Frontend

- **React 19** with **Vite 6** for development and bundling
- **TailwindCSS 4** for styling
- **shadcn/ui** for component primitives
- **EventSource API** for SSE consumption

### Infrastructure

- **Docker** + **Docker Swarm** for deployment
- **docker-compose.yml** with `dev` and `prod` profiles for local development
- **DuckDB** embedded (no external database server required)
- **MIT License** — fully open source

---

## Next Steps

- [Installation](./installation) — Deploy RESMA on Docker Swarm, Compose, or run locally
- [Configuration](./configuration) — Environment variables and tuning
- [API Reference](./api-reference) — Full endpoint documentation
