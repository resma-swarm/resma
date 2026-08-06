#!/usr/bin/env bash
#
# RESMA Upgrader — roda dentro de um container Docker com docker.sock montado.
# Faz pull das novas imagens e atualiza os services do Swarm in-place, preservando
# dados (volumes) e secrets. Não recria o stack — apenas atualiza as imagens.
#
# Uso (one-liner):
#   docker run -it --rm \
#     --name resma-upgrader \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e MODE=upgrade \
#     resmaswarm/resma-install:latest
#
# Modo não-interativo:
#   docker run -it --rm \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e MODE=upgrade \
#     -e INTERACTIVE=0 \
#     -e STACK_NAME=resma \
#     resmaswarm/resma-install:latest
#
# Especificar versão (default: latest):
#   docker run -it --rm \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e MODE=upgrade \
#     -e RESMA_VERSION=v0.2.0 \
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
title "RESMA Upgrader"
echo "Version: ${RESMA_VERSION:-latest}"
echo

# ---------- defaults ----------
INTERACTIVE="${INTERACTIVE:-1}"
STACK_NAME="${STACK_NAME:-resma}"
IMAGE_PREFIX="${IMAGE_PREFIX:-docker.io/resmaswarm}"
RESMA_VERSION="${RESMA_VERSION:-latest}"

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
  exit 1
fi
success "Docker Swarm active"

MANAGER=$(docker info --format '{{.Swarm.ControlAvailable}}' 2>/dev/null || echo "false")
if [[ "$MANAGER" != "true" ]]; then
  error "ERROR: This node is not a Swarm manager."
  error "Run the upgrader on a manager node."
  exit 1
fi
success "Swarm manager node detected"

# ---------- 2. configurar (interativo ou não) ----------
section "Upgrade configuration"

if [[ "$INTERACTIVE" == "1" ]]; then
  input "Enter stack name [$STACK_NAME]: "
  read -r choice
  STACK_NAME="${choice:=$STACK_NAME}"

  input "Enter version to upgrade to [$RESMA_VERSION]: "
  read -r choice
  RESMA_VERSION="${choice:=$RESMA_VERSION}"
fi

echo "  Stack:    $STACK_NAME"
echo "  Version:  $RESMA_VERSION"
echo "  Prefix:   $IMAGE_PREFIX"
echo

# ---------- 3. verificar se o stack existe ----------
if ! docker stack ls --format '{{.Name}}' | grep -qx "$STACK_NAME"; then
  error "ERROR: Stack '$STACK_NAME' not found."
  echo
  echo "Available stacks:"
  docker stack ls --format '  {{.Name}} ({{.Services}} services)' 2>/dev/null || echo "  (none)"
  echo
  echo "To install RESMA from scratch, run without -e MODE=upgrade:"
  echo "  docker run -it --rm --volume /var/run/docker.sock:/var/run/docker.sock resmaswarm/resma-install:latest"
  exit 1
fi
success "Stack '$STACK_NAME' found"

# ---------- 4. pull das novas imagens ----------
section "Pulling images"
echo "  - ${IMAGE_PREFIX}/resma-api:${RESMA_VERSION}"
docker pull "${IMAGE_PREFIX}/resma-api:${RESMA_VERSION}"
echo "  - ${IMAGE_PREFIX}/resma-ml:${RESMA_VERSION}"
docker pull "${IMAGE_PREFIX}/resma-ml:${RESMA_VERSION}"
echo "  - ${IMAGE_PREFIX}/resma-agent:${RESMA_VERSION}"
docker pull "${IMAGE_PREFIX}/resma-agent:${RESMA_VERSION}"
success "Images pulled"

# ---------- 5. atualizar services ----------
section "Updating services"

update_service() {
  local svc="$1"
  local image="$2"
  local full="${STACK_NAME}_${svc}"
  echo "  docker service update --image $image $full"
  if docker service update --image "$image" "$full"; then
    success "  $full updated"
  else
    error "  ERROR: Failed to update $full"
    return 1
  fi
}

update_service "api"    "${IMAGE_PREFIX}/resma-api:${RESMA_VERSION}"
update_service "ml"     "${IMAGE_PREFIX}/resma-ml:${RESMA_VERSION}"
update_service "agent"  "${IMAGE_PREFIX}/resma-agent:${RESMA_VERSION}"

# ---------- 6. aguardar healthcheck ----------
section "Waiting for services to be healthy"
printf "  Starting"

MAX_ATTEMPTS=40  # ~3 min
attempt=0
while true; do
  STATUS=$(curl --unix-socket /var/run/docker.sock -sgG \
    -X GET "http:/v1.24/tasks?filters={\"service\":[\"${STACK_NAME}_api\"]}" \
    2>/dev/null | jq -r 'sort_by(.CreatedAt) | .[-1].Status.State' 2>/dev/null || echo "unknown")

  if [[ "$STATUS" == "running" ]]; then
    success "\n  ${STACK_NAME}_api is healthy"
    break
  fi
  printf "."
  attempt=$((attempt + 1))
  if [[ $attempt -ge $MAX_ATTEMPTS ]]; then
    error "\n  TIMEOUT — ${STACK_NAME}_api not healthy after ${MAX_ATTEMPTS} attempts."
    warning "Check logs: docker service logs ${STACK_NAME}_api"
    exit 1
  fi
  sleep 5
done

# ---------- 7. done ----------
MANAGER_IP=$(docker info --format '{{.Swarm.NodeAddr}}' 2>/dev/null || echo "localhost")
echo
title "=== RESMA upgraded successfully! ==="
echo
echo "  Version: $RESMA_VERSION"
echo "  URL:     http://${MANAGER_IP}:8080"
echo
echo "Notes:"
echo "  - Data (DuckDB) and secrets were preserved."
echo "  - If you upgraded to a tagged version (e.g. v0.2.0) and want to"
echo "    track 'latest' again, re-run with -e RESMA_VERSION=latest."
echo
echo "Logs:     docker service logs ${STACK_NAME}_api"
echo "Logs ML:  docker service logs ${STACK_NAME}_ml"
echo "Logs Ag:  docker service logs ${STACK_NAME}_agent"
echo
title "Enjoy! :)"
