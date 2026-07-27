# Configuration

RESMA is configured entirely through environment variables. A complete template
is available in [`.env.example`](https://github.com/resma-swarm/resma/blob/main/.env.example)
in the repository root.

---

## Environment Variables

### Core

| Variable | Default | Description |
|----------|---------|-------------|
| `RESMA_ENV` | `production` | Environment name: `production`, `development`, `test` |
| `RESMA_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `RESMA_API_PORT` | `8080` | Port the Go API listens on |
| `RESMA_ML_SIDECAR_URL` | `http://localhost:8081` | URL of the Python ML sidecar |

### Database (DuckDB)

| Variable | Default | Description |
|----------|---------|-------------|
| `RESMA_DUCKDB_PATH` | `/data/resma.duckdb` | Path to the DuckDB database file |
| `RESMA_DUCKDB_MEMORY_LIMIT` | `1GB` | Memory limit for DuckDB operations |
| `RESMA_DUCKDB_THREADS` | `2` | Number of threads DuckDB can use |

### Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `RESMA_JWT_SECRET` | — | Secret used to sign JWT tokens. **Required in production.** Can also be provided via `RESMA_JWT_SECRET_FILE` (Docker secret). |
| `RESMA_JWT_EXPIRY` | `24h` | JWT token expiration duration |
| `RESMA_ADMIN_PASSWORD` | — | Initial admin password. **Required on first run.** Can also be provided via `RESMA_ADMIN_PASSWORD_FILE` (Docker secret). |
| `RESMA_ADMIN_USERNAME` | `admin` | Initial admin username |
| `RESMA_API_KEY_SALT` | — | Salt for API key hashing. Auto-generated if not set. |

### Collector

| Variable | Default | Description |
|----------|---------|-------------|
| `RESMA_COLLECTOR_INTERVAL` | `15s` | Interval between metric collection cycles |
| `RESMA_COLLECTOR_TIMEOUT` | `10s` | Timeout for Docker API calls during collection |
| `RESMA_COLLECTOR_LABEL_FILTER` | — | Only collect metrics for containers with this Docker label (e.g., `resma.enable=true`) |
| `RESMA_METRICS_RETENTION_DAYS` | `30` | Number of days of metrics data to retain. Older data is purged. |

### ML / Recommendations

| Variable | Default | Description |
|----------|---------|-------------|
| `RESMA_ML_MIN_SAMPLES` | `240` | Minimum number of samples required before generating recommendations (240 samples @ 15s = 1 hour) |
| `RESMA_ML_ANALYSIS_INTERVAL` | `1h` | How often the ML sidecar runs analysis on collected data |
| `RESMA_ML_LEAK_R2_THRESHOLD` | `0.8` | R² threshold for linear regression to flag a memory leak |
| `RESMA_ML_LEAK_MIN_SLOPE_MB` | `0.5` | Minimum memory growth slope (MB/hour) to consider it a leak |
| `RESMA_ML_RECOMMENDATION_PERCENTILE` | `95` | Percentile used for resource limit recommendations (p95 by default) |
| `RESMA_ML_RECOMMENDATION_BUFFER` | `0.15` | Buffer added to percentile value (15% above p95) |

### SSE / Real-time

| Variable | Default | Description |
|----------|---------|-------------|
| `RESMA_SSE_KEEPALIVE_INTERVAL` | `30s` | Interval for SSE keepalive ping messages |
| `RESMA_SSE_MAX_CLIENTS` | `100` | Maximum number of concurrent SSE clients |

### Docker

| Variable | Default | Description |
|----------|---------|-------------|
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon connection URL |
| `RESMA_DOCKER_TLS_VERIFY` | `false` | Enable TLS verification for Docker daemon connection |
| `RESMA_DOCKER_CERT_PATH` | — | Path to Docker TLS certificates (if TLS enabled) |

### Frontend (production build)

| Variable | Default | Description |
|----------|---------|-------------|
| `RESMA_FRONTEND_DIST` | `./frontend/dist` | Path to the built frontend assets (served by the API in production) |

---

## Example `.env` File

```bash
# ── Core ──────────────────────────────────────────────
RESMA_ENV=production
RESMA_LOG_LEVEL=info
RESMA_API_PORT=8080
RESMA_ML_SIDECAR_URL=http://resma-ml:8081

# ── Database ──────────────────────────────────────────
RESMA_DUCKDB_PATH=/data/resma.duckdb
RESMA_DUCKDB_MEMORY_LIMIT=1GB
RESMA_DUCKDB_THREADS=2

# ── Auth ──────────────────────────────────────────────
# IMPORTANT: Change these in production!
RESMA_JWT_SECRET=change-me-to-a-random-64-char-hex-string
RESMA_JWT_EXPIRY=24h
RESMA_ADMIN_USERNAME=admin
RESMA_ADMIN_PASSWORD=ChangeMeInProduction!

# ── Collector ─────────────────────────────────────────
RESMA_COLLECTOR_INTERVAL=15s
RESMA_COLLECTOR_TIMEOUT=10s
RESMA_METRICS_RETENTION_DAYS=30

# ── ML ────────────────────────────────────────────────
RESMA_ML_MIN_SAMPLES=240
RESMA_ML_ANALYSIS_INTERVAL=1h
RESMA_ML_LEAK_R2_THRESHOLD=0.8
RESMA_ML_LEAK_MIN_SLOPE_MB=0.5
RESMA_ML_RECOMMENDATION_PERCENTILE=95
RESMA_ML_RECOMMENDATION_BUFFER=0.15

# ── SSE ───────────────────────────────────────────────
RESMA_SSE_KEEPALIVE_INTERVAL=30s
RESMA_SSE_MAX_CLIENTS=100

# ── Docker ────────────────────────────────────────────
DOCKER_HOST=unix:///var/run/docker.sock
```

---

## Docker Secrets

In production (especially Docker Swarm), prefer file-based secrets over
environment variables for sensitive values. RESMA supports the `_FILE` suffix
for the following variables:

| Env Var | File-based Alternative |
|---------|----------------------|
| `RESMA_JWT_SECRET` | `RESMA_JWT_SECRET_FILE` |
| `RESMA_ADMIN_PASSWORD` | `RESMA_ADMIN_PASSWORD_FILE` |
| `RESMA_API_KEY_SALT` | `RESMA_API_KEY_SALT_FILE` |

When using Docker secrets, the file is read at startup and the value is loaded
into memory. The env var takes precedence if both are set.

Example with Docker Swarm:

```bash
openssl rand -hex 32 | docker secret create resma_jwt_secret -
echo "MySecurePassword!" | docker secret create resma_admin_password -
```

Then in your stack file:

```yaml
environment:
  - RESMA_JWT_SECRET_FILE=/run/secrets/resma_jwt_secret
  - RESMA_ADMIN_PASSWORD_FILE=/run/secrets/resma_admin_password
secrets:
  - resma_jwt_secret
  - resma_admin_password
```

---

## Tuning Guide

### High-frequency collection (more data, more storage)

If you want finer-grained metrics:

```bash
RESMA_COLLECTOR_INTERVAL=5s
RESMA_METRICS_RETENTION_DAYS=14
RESMA_DUCKDB_MEMORY_LIMIT=2GB
```

This increases storage usage ~3x but gives the ML models more data points for
trend detection.

### Large clusters (many services)

For clusters with 50+ services:

```bash
RESMA_COLLECTOR_INTERVAL=30s
RESMA_DUCKDB_THREADS=4
RESMA_DUCKDB_MEMORY_LIMIT=2GB
RESMA_SSE_MAX_CLIENTS=200
```

### Conservative leak detection (fewer false positives)

To reduce false-positive leak alerts:

```bash
RESMA_ML_LEAK_R2_THRESHOLD=0.9
RESMA_ML_LEAK_MIN_SLOPE_MB=2.0
RESMA_ML_MIN_SAMPLES=720   # 3 hours of data at 15s
```
