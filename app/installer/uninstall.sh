#!/usr/bin/env bash
#
# RESMA Uninstaller — roda dentro de um container Docker com docker.sock montado.
# Remove o stack, os secrets e opcionalmente os volumes.
#
# Uso (one-liner):
#   docker run -it --rm \
#     --name resma-uninstaller \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e MODE=uninstall \
#     resmaswarm/resma-install:latest
#
# Modo não-interativo (sem prompts, remove volumes também):
#   docker run -it --rm \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e MODE=uninstall \
#     -e INTERACTIVE=0 \
#     -e REMOVE_VOLUMES=1 \
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
title "RESMA Uninstaller"
echo

# ---------- defaults ----------
INTERACTIVE="${INTERACTIVE:-1}"
STACK_NAME="${STACK_NAME:-resma}"
REMOVE_VOLUMES="${REMOVE_VOLUMES:-0}"

# ---------- 1. validar docker ----------
section "Checking prerequisites"

if ! docker info >/dev/null 2>&1; then
  error "ERROR: Cannot reach Docker daemon."
  error "Mount the Docker socket: --volume /var/run/docker.sock:/var/run/docker.sock"
  exit 1
fi
success "Docker daemon reachable"

# ---------- 2. confirmar ----------
section "Uninstall configuration"

if [[ "$INTERACTIVE" == "1" ]]; then
  input "Enter stack name to remove [$STACK_NAME]: "
  read -r choice
  STACK_NAME="${choice:=$STACK_NAME}"

  input "Also remove Docker volumes (data will be lost)? [y/N]: "
  read -r choice
  if [[ "${choice,,}" == "y" || "${choice,,}" == "yes" ]]; then
    REMOVE_VOLUMES=1
  fi
fi

echo "  Stack:    $STACK_NAME"
echo "  Volumes:  $([ "$REMOVE_VOLUMES" == "1" ] && echo 'YES (data lost)' || echo 'NO (preserved)')"
echo

# ---------- 3. verificar se o stack existe ----------
if ! docker stack ls --format '{{.Name}}' | grep -qx "$STACK_NAME"; then
  warning "Stack '$STACK_NAME' not found — nothing to remove."
  echo
  echo "Available stacks:"
  docker stack ls --format '  {{.Name}} ({{.Services}} services)' 2>/dev/null || echo "  (none)"
  exit 0
fi

# ---------- 4. remover stack ----------
section "Removing stack"
echo "  docker stack rm $STACK_NAME"
docker stack rm "$STACK_NAME"
success "Stack removal requested"

# ---------- 5. aguardar serviços pararem ----------
section "Waiting for services to stop"
printf "  Stopping"
MAX_ATTEMPTS=30
attempt=0
while true; do
  REMAINING=$(docker stack ps "$STACK_NAME" --format '{{.ID}}' 2>/dev/null | wc -l)
  if [[ "$REMAINING" -eq 0 ]]; then
    success "\n  All services stopped"
    break
  fi
  printf "."
  attempt=$((attempt + 1))
  if [[ $attempt -ge $MAX_ATTEMPTS ]]; then
    warning "\n  Some tasks still shutting down after ${MAX_ATTEMPTS} attempts."
    warning "  Check with: docker stack ps $STACK_NAME"
    break
  fi
  sleep 2
done

# ---------- 6. remover secrets ----------
section "Removing Docker secrets"
for secret in resma_jwt_secret resma_agent_token; do
  if docker secret inspect "$secret" >/dev/null 2>&1; then
    echo "  docker secret rm $secret"
    docker secret rm "$secret"
    success "  Secret $secret removed"
  else
    echo "  Secret $secret not found — skipping"
  fi
done

# ---------- 7. remover volumes (opcional) ----------
if [[ "$REMOVE_VOLUMES" == "1" ]]; then
  section "Removing Docker volumes"
  for vol in "${STACK_NAME}_resma-data" "${STACK_NAME}_resma-agent-data" resma-data resma-agent-data; do
    if docker volume inspect "$vol" >/dev/null 2>&1; then
      echo "  docker volume rm $vol"
      docker volume rm "$vol" 2>/dev/null && success "  Volume $vol removed" || warning "  Volume $vol in use — skipped"
    fi
  done
else
  section "Preserving Docker volumes"
  echo "  Volumes preserved. To remove manually:"
  echo "    docker volume rm ${STACK_NAME}_resma-data ${STACK_NAME}_resma-agent-data"
fi

# ---------- 8. remover network (se criada) ----------
section "Cleaning up network"
NETWORK="${STACK_NAME}_resma-net"
if docker network inspect "$NETWORK" >/dev/null 2>&1; then
  echo "  docker network rm $NETWORK"
  docker network rm "$NETWORK" 2>/dev/null && success "  Network $NETWORK removed" || warning "  Network $NETWORK in use — skipped"
fi

# ---------- 9. done ----------
echo
title "=== RESMA uninstalled successfully! ==="
echo
echo "The RESMA stack, secrets, and network have been removed."
if [[ "$REMOVE_VOLUMES" == "1" ]]; then
  echo "Volumes were also removed (data lost)."
else
  echo "Volumes were preserved — remove manually if no longer needed."
fi
echo
echo "Docker images are still on this node. To remove:"
echo "  docker rmi docker.io/resmaswarm/resma-api:latest"
echo "  docker rmi docker.io/resmaswarm/resma-ml:latest"
echo "  docker rmi docker.io/resmaswarm/resma-agent:latest"
echo
title "Done! :)"
