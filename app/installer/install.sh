#!/usr/bin/env bash
#
# RESMA Installer — roda dentro de um container Docker com docker.sock montado.
# Gera secrets, faz pull das imagens, deploya o stack e aguarda healthcheck.
#
# Uso (one-liner):
#   docker run -it --rm \
#     --name resma-installer \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     resmaswarm/resma-install:latest
#
# Modo não-interativo:
#   docker run -it --rm \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e INTERACTIVE=0 \
#     -e STACK_NAME=resma \
#     -e APP_PORT=8080 \
#     resmaswarm/resma-install:latest

set -euo pipefail

# ---------- colors ----------
GC='\033[0;32m'; RC='\033[0;31m'; OC='\033[0;33m'; NC='\033[0m'; BC='\033[1m'; UC='\033[4m'
success() { echo -e "${GC}$1${NC}"; }
warning() { echo -e "${OC}$1${NC}"; }
error()   { echo -e "${RC}$1${NC}"; }
title()   { echo -e "${BC}$1${NC}"; }
section() { echo -e "\n${UC}$1${NC}"; }
input()   { printf "$1"; }

# ---------- banner ----------
cat /install/logo.txt
echo
title "Welcome to RESMA — RESource MAnager for Docker Swarm"
echo "Version: ${VERSION:-latest}"
echo

# ---------- defaults ----------
INTERACTIVE="${INTERACTIVE:-1}"
STACK_NAME="${STACK_NAME:-resma}"
APP_PORT="${APP_PORT:-8080}"
IMAGE_PREFIX="${IMAGE_PREFIX:-docker.io/resmaswarm}"
COMPOSE_FILE="/install/docker-stack.yml"

# ---------- 1. validar docker + swarm ----------
section "Checking prerequisites"

if ! docker info >/dev/null 2>&1; then
  error "ERROR: Cannot reach Docker daemon."
  error "Mount the Docker socket: --volume /var/run/docker.sock:/var/run/docker.sock"
  exit 1
fi
success "Docker daemon reachable"

SWARM_STATE=$(docker info --format '{{.Swarm.LocalNodeState}}' 2>/dev/null || echo "inactive")
if [[ "$SWARM_STATE" != "active" ]]; then
  error "ERROR: Docker Swarm is not active on this node (state: $SWARM_STATE)."
  error "Initialize with: docker swarm init"
  exit 1
fi
success "Docker Swarm active"

MANAGER=$(docker info --format '{{.Swarm.ControlAvailable}}' 2>/dev/null || echo "false")
if [[ "$MANAGER" != "true" ]]; then
  error "ERROR: This node is not a Swarm manager."
  error "Run the installer on a manager node."
  exit 1
fi
success "Swarm manager node detected"

# ---------- 2. setup (interativo ou não) ----------
section "Application setup"

if [[ "$INTERACTIVE" == "1" ]]; then
  # Stack name
  while true; do
    input "Enter stack name [$STACK_NAME]: "
    read -r choice
    S="${choice:=$STACK_NAME}"
    if docker stack ps "$S" >/dev/null 2>&1; then
      warning "Stack name [$S] is already taken!"
    else
      STACK_NAME="$S"; break
    fi
  done

  # Port
  input "Enter application port [$APP_PORT]: "
  read -r choice
  APP_PORT="${choice:=$APP_PORT}"
else
  echo "Stack name: $STACK_NAME"
  echo "Application port: $APP_PORT"
  if docker stack ps "$STACK_NAME" >/dev/null 2>&1; then
    warning "Stack [$STACK_NAME] already exists!"
    error "SETUP FAILED — choose a different stack name."
    exit 1
  fi
fi

# ---------- 3. gerar secrets ----------
section "Generating Docker secrets"

# JWT secret
if docker secret inspect resma_jwt_secret >/dev/null 2>&1; then
  warning "Secret resma_jwt_secret already exists — removing..."
  docker secret rm resma_jwt_secret
fi
JWT_SECRET=$(openssl rand -base64 32)
echo -n "$JWT_SECRET" | docker secret create resma_jwt_secret -
success "Secret resma_jwt_secret created"

# Agent token
if docker secret inspect resma_agent_token >/dev/null 2>&1; then
  warning "Secret resma_agent_token already exists — removing..."
  docker secret rm resma_agent_token
fi
AGENT_TOKEN=$(openssl rand -hex 32)
echo -n "$AGENT_TOKEN" | docker secret create resma_agent_token -
success "Secret resma_agent_token created"

# ---------- 4. ajustar compose file (porta) ----------
if [[ "$APP_PORT" != "8080" ]]; then
  sed -i "s|published: 8080|published: $APP_PORT|" "$COMPOSE_FILE"
fi

# ---------- 5. pull imagens ----------
section "Pulling images"
echo "  - ${IMAGE_PREFIX}/resma-api:latest"
docker pull "${IMAGE_PREFIX}/resma-api:latest"
echo "  - ${IMAGE_PREFIX}/resma-ml:latest"
docker pull "${IMAGE_PREFIX}/resma-ml:latest"
echo "  - ${IMAGE_PREFIX}/resma-agent:latest"
docker pull "${IMAGE_PREFIX}/resma-agent:latest"
success "Images pulled"

# ---------- 6. deploy ----------
section "Deploying stack"
RESMA_CORS_ORIGINS="${RESMA_CORS_ORIGINS:-http://localhost:${APP_PORT}}" \
  docker stack deploy -c "$COMPOSE_FILE" "$STACK_NAME"
success "Stack deployed"

# ---------- 7. aguardar healthcheck ----------
section "Waiting for services to be healthy"
printf "  Starting"

MAX_ATTEMPTS=40  # ~3 min
attempt=0
while true; do
  STATUS=$(curl --unix-socket /var/run/docker.sock -sgG \
    -X GET "http:/v1.24/tasks?filters={\"service\":[\"${STACK_NAME}_resma-api\"]}" \
    2>/dev/null | jq -r 'sort_by(.CreatedAt) | .[-1].Status.State' 2>/dev/null || echo "unknown")

  if [[ "$STATUS" == "running" ]]; then
    success "\n  resma-api is healthy"
    break
  fi
  printf "."
  attempt=$((attempt + 1))
  if [[ $attempt -ge $MAX_ATTEMPTS ]]; then
    error "\n  TIMEOUT — resma-api not healthy after ${MAX_ATTEMPTS} attempts."
    warning "Check logs: docker service logs ${STACK_NAME}_resma-api"
    exit 1
  fi
  sleep 5
done

# ---------- 8. done ----------
MANAGER_IP=$(docker info --format '{{.Swarm.NodeAddr}}' 2>/dev/null || echo "localhost")
echo
title "=== RESMA installed successfully! ==="
echo
echo "  URL: http://${MANAGER_IP}:${APP_PORT}"
echo
echo "Next steps:"
echo "  1. Open the URL above in your browser"
echo "  2. Complete onboarding (create admin user)"
echo "  3. Configure API keys if needed"
echo
echo "Logs:     docker service logs ${STACK_NAME}_resma-api"
echo "Logs ML:  docker service logs ${STACK_NAME}_resma-ml"
echo "Remove:   docker stack rm ${STACK_NAME}"
echo
title "Enjoy! :)"
