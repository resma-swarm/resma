# Installation

RESMA can be deployed in three ways: via the **installer container** (recommended
for production Docker Swarm), via **Docker Compose** (for single-node or testing),
or **locally** for development.

---

## Prerequisites

- **Docker** 24.0+ with the Docker CLI
- **Docker Swarm** initialized (for Swarm deployment) — `docker swarm init`
- **Go 1.26+** (only for local development)
- **Python 3.12+** (only for local ML sidecar development)
- **Node.js 20+** and **pnpm** (only for local frontend development)

---

## Method 1: Installer Container (Production — Recommended)

The installer container is the simplest way to deploy RESMA on a Docker Swarm
cluster. It runs isolated in a container with the Docker socket mounted — no
`curl | bash`, no dependencies on the host beyond Docker itself.

### One-liner

Run this on any Swarm **manager** node:

```bash
docker run -it --rm \
  --name resma-installer \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest
```

The installer will:

1. Validate that Docker Swarm is active and the node is a manager
2. Generate the `resma_jwt_secret` and `resma_agent_token` Docker secrets
3. Pull the 3 runtime images (`resma-api`, `resma-ml`, `resma-agent`)
4. Deploy the stack via `docker stack deploy`
5. Wait for the API healthcheck to pass
6. Print the dashboard URL

### Non-interactive mode

For automation or CI, use environment variables instead of prompts:

```bash
docker run -it --rm \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  -e INTERACTIVE=0 \
  -e STACK_NAME=resma \
  -e APP_PORT=8080 \
  resmaswarm/resma-install:latest
```

| Variable | Default | Description |
|----------|---------|-------------|
| `INTERACTIVE` | `1` | Set to `0` for non-interactive mode |
| `STACK_NAME` | `resma` | Docker Swarm stack name |
| `APP_PORT` | `8080` | Published port for the dashboard |
| `IMAGE_PREFIX` | `docker.io/resmaswarm` | Image namespace (for custom registries) |

### Verify

```bash
docker service ls
curl http://localhost:8080/health
```

The API should respond with `{"status":"ok"}`. The dashboard is available at
`http://localhost:8080/`.

### Stopping / Removing

```bash
docker stack rm resma
docker secret rm resma_jwt_secret resma_agent_token
```

---

## Method 2: Manual Docker Swarm (Production)

If you prefer to control each step manually instead of using the installer
container.

### 1. Initialize Swarm (if not already)

```bash
docker swarm init
```

### 2. Create required secrets

```bash
# JWT secret (for auth tokens)
openssl rand -base64 32 | docker secret create resma_jwt_secret -

# Agent token (for multi-node agents to push metrics)
openssl rand -hex 32 | docker secret create resma_agent_token -
```

### 3. Pull the images

```bash
docker pull docker.io/resmaswarm/resma-api:latest
docker pull docker.io/resmaswarm/resma-ml:latest
docker pull docker.io/resmaswarm/resma-agent:latest
```

### 4. Deploy the stack

Download the stack file and deploy:

```bash
curl -fsSL https://raw.githubusercontent.com/resma-swarm/resma/main/docker-stack.yml -o docker-stack.yml
docker stack deploy -c docker-stack.yml resma
```

### 5. Verify

```bash
docker service ls
curl http://localhost:8080/health
```

### Stopping

```bash
docker stack rm resma
```

---

## Method 3: Docker Compose (Dev / Single-node)

For development or single-node testing with local image builds.

### 1. Clone the repository

```bash
git clone https://github.com/resma-swarm/resma.git
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

## Method 4: Local Development

For contributing to RESMA or running it outside Docker.

### 1. Clone the repository

```bash
git clone https://github.com/resma-swarm/resma.git
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
