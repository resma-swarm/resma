# Installation

RESMA can be deployed in three ways: as a **Docker Swarm service** (recommended
for production), via **Docker Compose** (for single-node or testing), or
**locally** for development.

---

## Prerequisites

- **Docker** 24.0+ with the Docker CLI
- **Docker Swarm** initialized (for Swarm deployment) — `docker swarm init`
- **Go 1.26+** (only for local development)
- **Python 3.12+** (only for local ML sidecar development)
- **Node.js 20+** and **pnpm** (only for local frontend development)

---

## Method 1: Docker Swarm (Production)

This is the recommended deployment for production clusters.

### 1. Initialize Swarm (if not already)

```bash
docker swarm init
```

### 2. Create the RESMA network

```bash
docker network create --driver overlay resma-net
```

### 3. Create required secrets and configs

RESMA needs a JWT secret and an admin password. Create them as Docker secrets:

```bash
# Generate a random JWT secret
openssl rand -hex 32 | docker secret create resma_jwt_secret -

# Set the admin password (change this!)
echo "ChangeMeInProduction!" | docker secret create resma_admin_password -
```

### 4. Deploy the stack

Create a `resma-stack.yml` file:

```yaml
version: "3.8"

services:
  resma-api:
    image: ghcr.io/user/resma:latest
    environment:
      - RESMA_JWT_SECRET_FILE=/run/secrets/resma_jwt_secret
      - RESMA_ADMIN_PASSWORD_FILE=/run/secrets/resma_admin_password
      - RESMA_DUCKDB_PATH=/data/resma.duckdb
      - RESMA_COLLECTOR_INTERVAL=15s
      - RESMA_ML_SIDECAR_URL=http://resma-ml:8081
    secrets:
      - resma_jwt_secret
      - resma_admin_password
    volumes:
      - resma-data:/data
    ports:
      - target: 8080
        published: 8080
        mode: ingress
    networks:
      - resma-net
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.role == manager

  resma-ml:
    image: ghcr.io/user/resma-ml:latest
    networks:
      - resma-net
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.role == manager

secrets:
  resma_jwt_secret:
    external: true
  resma_admin_password:
    external: true

volumes:
  resma-data:

networks:
  resma-net:
    external: true
```

Deploy:

```bash
docker stack deploy -c resma-stack.yml resma
```

### 5. Verify

```bash
docker service ls
curl http://localhost:8080/health
```

The API should respond with `{"status":"ok"}`. The dashboard is available at
`http://localhost:8080/` (served by the API in production mode).

---

## Method 2: Docker Compose (Single-node / Testing)

For single-node deployments or local testing without Swarm.

### 1. Clone the repository

```bash
git clone https://github.com/USER/resma.git
cd resma
```

### 2. Configure environment

Copy the example env file and edit it:

```bash
cp .env.example .env
```

Edit `.env` with your settings (see [Configuration](./configuration) for all
variables).

### 3. Deploy via Docker Swarm (production)

```bash
.\scripts\deploy-swarm.ps1
```

This builds the images and deploys via `docker stack deploy`:
- `resma-api` on port **8080** (manager node only)
- `resma-ml` on port **8081** (internal, not exposed)
- `resma-agent` as a global service (1 per node)

### 4. Verify

```bash
docker stack services resma
curl http://localhost:8080/health
```

### Stopping

```bash
docker stack rm resma
```

---

## Method 3: Local Development

For contributing to RESMA or running it outside Docker.

### 1. Clone the repository

```bash
git clone https://github.com/USER/resma.git
cd resma
```

### 2. Start the Go API in dev mode (Docker)

The Go API is built and run inside a Docker container to ensure the correct
CGO toolchain for `go-duckdb`:

```bash
docker compose up -d go-dev
docker compose exec go-dev go run ./cmd/server
```

The API is available at `http://localhost:8080`.

### 3. Start the ML sidecar (Python)

```bash
cd app/ml
python -m venv .venv
source .venv/bin/activate  # Windows: .venv\Scripts\activate
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8081 --reload
```

The ML sidecar runs at `http://localhost:8081` (internal only).

### 4. Start the frontend (React + Vite)

```bash
cd frontend
pnpm install
pnpm dev
```

The frontend dev server runs at `http://localhost:5173` and proxies `/api` to
`http://localhost:8080`.

### 5. Verify

Open `http://localhost:5173` in your browser. You should see the RESMA
dashboard. Check the API health:

```bash
curl http://localhost:8080/health
```

---

## Post-Install Setup

Regardless of the deployment method, after RESMA is running:

1. **Log in** — Navigate to the dashboard and log in with the admin credentials
   (default: `admin` / password from secret/env). **Change the password
   immediately.**

2. **Generate an API key** — For programmatic access to `/api/v1/*`, create an
   API key with appropriate scopes via the dashboard or the internal API.

3. **Verify collection** — After 1–2 minutes, check that metrics are being
   collected:
   ```bash
   curl -H "Authorization: Bearer <your-api-key>" \
     http://localhost:8080/api/v1/services
   ```

4. **Wait for recommendations** — ML recommendations require at least 1 hour of
   collected data to produce meaningful results. Leak detection requires
   sustained data (typically 6+ hours).

---

## Troubleshooting

### API won't start: DuckDB error

DuckDB requires a writable directory for its database file. Ensure the volume
mount (`/data`) is writable by the container. On Swarm, check that the node
running RESMA has sufficient disk space.

### ML sidecar unreachable

The ML sidecar listens on port 8081 and should only be accessible within the
Docker network. Verify the `RESMA_ML_SIDECAR_URL` env var points to the correct
service name (`http://resma-ml:8081` in Swarm, `http://resma-ml:8081` in
Compose).

### No metrics appearing

Ensure the Docker socket is accessible. In Swarm/Compose, the API container
needs access to `/var/run/docker.sock`. The compose file mounts it read-only by
default. Verify the Docker daemon is running and the API container can reach it.
