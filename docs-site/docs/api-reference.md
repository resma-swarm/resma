# API Reference

RESMA exposes two API surfaces: an **internal API** (`/api/*`) consumed by the
frontend dashboard (JWT auth), and a **public API** (`/api/v1/*`) for automation
and integrations (API key + scopes).

---

## Authentication

### Internal API (`/api/*`)

Uses **JWT bearer tokens** obtained via the login endpoint. Tokens expire after
`RESMA_JWT_EXPIRY` (default 24h). Include the token in the `Authorization`
header:

```
Authorization: Bearer <jwt-token>
```

For SSE endpoints, the JWT is set as an `HttpOnly` cookie by the login endpoint
and sent automatically by browsers. Non-browser clients can use the
`Authorization` header instead.

### Public API (`/api/v1/*`)

Uses **API keys** with scoped permissions. API keys are managed via the
dashboard or the internal API. Include the key in the `Authorization` header:

```
Authorization: Bearer <api-key>
```

#### API Key Scopes

| Scope | Description |
|-------|-------------|
| `metrics:read` | Read metrics for all services |
| `recommendations:read` | Read ML recommendations |
| `alerts:read` | Read leak detection alerts |
| `services:read` | List services and containers |
| `apikeys:manage` | Create and revoke API keys |
| `admin` | Full access (all scopes) |

---

## Internal API (`/api/*`)

### Auth

#### `POST /api/auth/login`

Authenticate and obtain a JWT token.

**Request:**

```json
{
  "username": "admin",
  "password": "your-password"
}
```

**Response (200):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2025-01-15T18:30:00Z"
}
```

Sets an `HttpOnly` cookie `resma_session` with the JWT for SSE consumption.

#### `POST /api/auth/logout`

Invalidate the current session. Clears the `resma_session` cookie.

**Response (204):** No content.

#### `POST /api/auth/refresh`

Refresh the current JWT token (requires a valid token).

**Response (200):**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2025-01-15T19:30:00Z"
}
```

### API Keys

#### `GET /api/apikeys`

List all API keys (without revealing the key value).

**Response (200):**

```json
{
  "keys": [
    {
      "id": "key_abc123",
      "name": "Grafana integration",
      "scopes": ["metrics:read"],
      "created_at": "2025-01-10T12:00:00Z",
      "last_used_at": "2025-01-14T08:15:00Z"
    }
  ]
}
```

#### `POST /api/apikeys`

Create a new API key.

**Request:**

```json
{
  "name": "Grafana integration",
  "scopes": ["metrics:read", "recommendations:read"]
}
```

**Response (201):**

```json
{
  "id": "key_abc123",
  "key": "rsk_live_xxxxxxxxxxxxx",
  "name": "Grafana integration",
  "scopes": ["metrics:read", "recommendations:read"],
  "created_at": "2025-01-15T12:00:00Z"
}
```

:::warning
The `key` value is only shown once at creation time. Store it securely.
:::

#### `DELETE /api/apikeys/:id`

Revoke an API key by its ID.

**Response (204):** No content.

### SSE Streaming

#### `GET /api/sse/metrics`

Stream real-time metric updates via Server-Sent Events.

**Events:**

| Event | Data | Description |
|-------|------|-------------|
| `metric` | `{service, container, cpu, memory, timestamp}` | Per-container metric update |
| `alert` | `{service, type, severity, message, timestamp}` | Leak detection alert |
| `ping` | — | Keepalive (every `RESMA_SSE_KEEPALIVE_INTERVAL`) |

**Example (JavaScript):**

```javascript
const es = new EventSource('/api/sse/metrics');

es.addEventListener('metric', (e) => {
  const data = JSON.parse(e.data);
  console.log(`${data.service}: CPU ${data.cpu}% MEM ${data.memory}MB`);
});

es.addEventListener('alert', (e) => {
  const alert = JSON.parse(e.data);
  console.warn(`ALERT: ${alert.message}`);
});
```

---

## Public API (`/api/v1/*`)

All public API endpoints require an API key with the appropriate scope.

### Services

#### `GET /api/v1/services`

List all monitored Docker Swarm services.

**Required scope:** `services:read`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `limit` | int | 100 | Maximum number of services to return |
| `offset` | int | 0 | Pagination offset |

**Response (200):**

```json
{
  "services": [
    {
      "id": "swarm-service-id",
      "name": "api",
      "image": "ghcr.io/user/api:latest",
      "replicas": 3,
      "containers": [
        {
          "id": "container-id",
          "node": "node-1",
          "status": "running"
        }
      ]
    }
  ],
  "total": 12
}
```

#### `GET /api/v1/services/:id`

Get details for a specific service.

**Required scope:** `services:read`

**Response (200):**

```json
{
  "id": "swarm-service-id",
  "name": "api",
  "image": "ghcr.io/user/api:latest",
  "replicas": 3,
  "resource_limits": {
    "cpus": "0.50",
    "memory": "512MB"
  },
  "containers": [
    {
      "id": "container-id",
      "node": "node-1",
      "status": "running",
      "cpu_percent": 12.5,
      "memory_mb": 340.2
    }
  ]
}
```

### Metrics

#### `GET /api/v1/metrics/:service`

Get time-series metrics for a service.

**Required scope:** `metrics:read`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `start` | ISO8601 | 24h ago | Start of the time range |
| `end` | ISO8601 | now | End of the time range |
| `interval` | string | `auto` | Aggregation interval: `auto`, `1m`, `5m`, `15m`, `1h` |
| `metric` | string | `all` | Filter: `cpu`, `memory`, `all` |

**Response (200):**

```json
{
  "service": "api",
  "start": "2025-01-14T12:00:00Z",
  "end": "2025-01-15T12:00:00Z",
  "interval": "5m",
  "series": [
    {
      "timestamp": "2025-01-14T12:00:00Z",
      "cpu_avg": 15.3,
      "cpu_p95": 22.1,
      "memory_avg_mb": 340.5,
      "memory_p95_mb": 380.2
    },
    {
      "timestamp": "2025-01-14T12:05:00Z",
      "cpu_avg": 14.8,
      "cpu_p95": 20.5,
      "memory_avg_mb": 342.1,
      "memory_p95_mb": 381.0
    }
  ]
}
```

#### `GET /api/v1/metrics/:service/summary`

Get aggregated metric statistics for a service over a time range.

**Required scope:** `metrics:read`

**Query parameters:** Same as `GET /api/v1/metrics/:service`

**Response (200):**

```json
{
  "service": "api",
  "period": "24h",
  "cpu": {
    "avg": 15.1,
    "p50": 14.0,
    "p95": 22.3,
    "p99": 28.7,
    "max": 35.2
  },
  "memory": {
    "avg_mb": 341.0,
    "p50_mb": 338.0,
    "p95_mb": 382.0,
    "p99_mb": 395.0,
    "max_mb": 410.5
  }
}
```

### Recommendations

#### `GET /api/v1/recommendations`

List ML-generated resource recommendations for all services.

**Required scope:** `recommendations:read`

**Response (200):**

```json
{
  "recommendations": [
    {
      "service": "api",
      "type": "memory_limit",
      "current": "512MB",
      "recommended": "440MB",
      "reason": "p95 memory usage is 382MB over the last 24h. 15% buffer applied.",
      "confidence": 0.92,
      "potential_savings": "72MB per replica",
      "generated_at": "2025-01-15T12:00:00Z"
    },
    {
      "service": "worker",
      "type": "cpu_limit",
      "current": null,
      "recommended": "0.25",
      "reason": "p95 CPU usage is 18% (0.18 cores). 15% buffer applied.",
      "confidence": 0.88,
      "potential_savings": null,
      "generated_at": "2025-01-15T12:00:00Z"
    }
  ]
}
```

#### `GET /api/v1/recommendations/:service`

Get recommendations for a specific service.

**Required scope:** `recommendations:read`

**Response:** Same as above, filtered to a single service.

### Alerts

#### `GET /api/v1/alerts`

List leak detection alerts.

**Required scope:** `alerts:read`

**Query parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `status` | string | `all` | Filter: `all`, `active`, `resolved` |
| `severity` | string | `all` | Filter: `all`, `warning`, `critical` |

**Response (200):**

```json
{
  "alerts": [
    {
      "id": "alert_001",
      "service": "payment-service",
      "type": "memory_leak",
      "severity": "critical",
      "status": "active",
      "message": "Memory growing at 5.2 MB/hour (R²=0.94). Estimated OOM in 48 hours.",
      "slope_mb_hour": 5.2,
      "r_squared": 0.94,
      "estimated_oom_hours": 48,
      "detected_at": "2025-01-15T06:00:00Z"
    }
  ]
}
```

#### `GET /api/v1/alerts/:id`

Get details for a specific alert.

**Required scope:** `alerts:read`

#### `POST /api/v1/alerts/:id/resolve`

Mark an alert as resolved (requires `admin` scope).

**Required scope:** `admin`

**Response (200):**

```json
{
  "id": "alert_001",
  "status": "resolved",
  "resolved_at": "2025-01-15T14:30:00Z"
}
```

---

## Infrastructure Endpoints

These endpoints require no authentication and are meant for health checks and
load balancers.

### `GET /health`

Liveness probe. Returns `200` if the process is running.

```json
{
  "status": "ok"
}
```

### `GET /ready`

Readiness probe. Returns `200` if the API is ready to serve requests (DuckDB
initialized, Docker daemon reachable, ML sidecar reachable).

```json
{
  "status": "ready",
  "checks": {
    "duckdb": "ok",
    "docker": "ok",
    "ml_sidecar": "ok"
  }
}
```

If any check fails, returns `503` with the failing check indicated:

```json
{
  "status": "not_ready",
  "checks": {
    "duckdb": "ok",
    "docker": "ok",
    "ml_sidecar": "unreachable"
  }
}
```

---

## Error Responses

All API endpoints use standard HTTP status codes and return errors in a
consistent JSON format:

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired token",
    "details": {}
  }
}
```

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `BAD_REQUEST` | Malformed request body or query params |
| 401 | `UNAUTHORIZED` | Missing or invalid authentication |
| 403 | `FORBIDDEN` | Authenticated but lacking required scope |
| 404 | `NOT_FOUND` | Resource not found |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Unexpected server error |
| 503 | `NOT_READY` | Service not ready (health check failure) |

---

## Rate Limiting

The public API (`/api/v1/*`) applies rate limiting per API key:

- **Default:** 100 requests per minute
- **Burst:** 20 requests

Rate limit headers are included in every response:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1736944860
```

When rate limited, the API returns `429` with `Retry-After` header indicating
seconds to wait.
