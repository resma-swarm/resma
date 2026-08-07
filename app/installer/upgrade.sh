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
#
# Ajustar intervalos de coleta no upgrade (apenas os passados são aplicados;
# os demais permanecem com o valor atual do service):
#   docker run -it --rm \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     -e MODE=upgrade \
#     -e RESMA_VERSION=v0.2.0 \
#     -e RESMA_COLLECT_INTERVAL=10 \
#     -e RESMA_STORAGE_INTERVAL=120 \
#     resmaswarm/resma-install:latest
#
# Intervalos de coleta (opcionais, sobrescrevíveis via -e):
#   RESMA_COLLECT_INTERVAL         (segundos)  — coleta de containers (api + agent)
#   RESMA_CLUSTER_INTERVAL         (segundos)  — info do cluster Swarm (api)
#   RESMA_STORAGE_INTERVAL         (segundos)  — métricas de storage (api)
#   RESMA_AGENT_TASK_POLL_INTERVAL (segundos)  — poll de tasks + health agents (api)
#   RESMA_ROLLBACK_POLL_INTERVAL   (segundos)  — poll do rollback watcher (api)
#   RESMA_SCHEDULER_POLL           (segundos)  — poll do scheduler (api, avançado)
#   RESMA_SSE_KEEPALIVE            (segundos)  — keepalive SSE (api, avançado)
#   RESMA_RETENTION_DAYS           (dias)      — retenção de métricas (api)
#   RESMA_ANALYSIS_WINDOW_DAYS     (dias)      — janela de análise ML (api)
#   RESMA_STALE_SERVICE_DAYS       (dias)      — dias sem heartbeat → stale (api)

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

# ---------- validação de ranges (igual ao install.sh) ----------
# Ranges definidos na spec phase-intervals-refresh seção 8.3.6.
validate_range() {
  local var="$1" val="$2" min="$3" max="$4" unit="$5"
  if ! [[ "$val" =~ ^[0-9]+$ ]]; then
    error "ERROR: $var='$val' não é um número inteiro válido."
    exit 1
  fi
  if (( 10#$val < min || 10#$val > max )); then
    error "ERROR: $var=$val fora do range permitido [$min..$max] $unit."
    error "       Use um valor entre $min e $max. Override via -e $var=<valor>."
    exit 1
  fi
}

# validate_passed_intervals valida apenas as env vars que foram explicitamente
# passadas (não vazias). As não-passadas permanecem com o valor atual do service.
validate_passed_intervals() {
  [[ -n "${RESMA_COLLECT_INTERVAL:-}" ]]         && validate_range RESMA_COLLECT_INTERVAL         "$RESMA_COLLECT_INTERVAL"         5  3600 "segundos"
  [[ -n "${RESMA_CLUSTER_INTERVAL:-}" ]]         && validate_range RESMA_CLUSTER_INTERVAL         "$RESMA_CLUSTER_INTERVAL"         5  3600 "segundos"
  [[ -n "${RESMA_STORAGE_INTERVAL:-}" ]]         && validate_range RESMA_STORAGE_INTERVAL         "$RESMA_STORAGE_INTERVAL"        10  3600 "segundos"
  [[ -n "${RESMA_AGENT_TASK_POLL_INTERVAL:-}" ]] && validate_range RESMA_AGENT_TASK_POLL_INTERVAL "$RESMA_AGENT_TASK_POLL_INTERVAL" 5   300 "segundos"
  [[ -n "${RESMA_ROLLBACK_POLL_INTERVAL:-}" ]]   && validate_range RESMA_ROLLBACK_POLL_INTERVAL   "$RESMA_ROLLBACK_POLL_INTERVAL"  10   300 "segundos"
  [[ -n "${RESMA_SCHEDULER_POLL:-}" ]]           && validate_range RESMA_SCHEDULER_POLL           "$RESMA_SCHEDULER_POLL"           5   300 "segundos"
  [[ -n "${RESMA_SSE_KEEPALIVE:-}" ]]            && validate_range RESMA_SSE_KEEPALIVE            "$RESMA_SSE_KEEPALIVE"            5    60 "segundos"
  [[ -n "${RESMA_RETENTION_DAYS:-}" ]]           && validate_range RESMA_RETENTION_DAYS           "$RESMA_RETENTION_DAYS"           1  3650 "dias"
  [[ -n "${RESMA_ANALYSIS_WINDOW_DAYS:-}" ]]     && validate_range RESMA_ANALYSIS_WINDOW_DAYS     "$RESMA_ANALYSIS_WINDOW_DAYS"     1   365 "dias"
  [[ -n "${RESMA_STALE_SERVICE_DAYS:-}" ]]       && validate_range RESMA_STALE_SERVICE_DAYS       "$RESMA_STALE_SERVICE_DAYS"       1   365 "dias"
  return 0  # sempre retorna 0 se nenhuma var foi passada (não trigga set -e)
}

# update_env aplica uma env var em um service via `docker service update --env-add`.
# Segue o padrão Docker: --env-add sobrescreve apenas a var passada, preservando as demais.
update_env() {
  local svc="$1"      # ex: api
  local var="$2"      # ex: RESMA_COLLECT_INTERVAL
  local val="$3"      # ex: 10
  local full="${STACK_NAME}_${svc}"
  echo "  docker service update --env-add ${var}=${val} ${full}"
  if docker service update --env-add "${var}=${val}" "$full"; then
    success "  ${var}=${val} applied to ${full}"
  else
    error "  ERROR: Failed to apply ${var}=${val} to ${full}"
    return 1
  fi
}

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

  # Optional: ajustar intervalos de coleta durante o upgrade.
  # Pressionar Enter em todos = manter valores atuais (não toca nos services).
  section "Collection intervals (press Enter to keep current values)"
  input "  Collect interval in seconds [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_COLLECT_INTERVAL="$choice"
  input "  Cluster interval in seconds [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_CLUSTER_INTERVAL="$choice"
  input "  Storage interval in seconds [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_STORAGE_INTERVAL="$choice"
  input "  Agent task poll interval in seconds [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_AGENT_TASK_POLL_INTERVAL="$choice"
  input "  Rollback poll interval in seconds [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_ROLLBACK_POLL_INTERVAL="$choice"
  input "  Retention days [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_RETENTION_DAYS="$choice"
  input "  Analysis window days [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_ANALYSIS_WINDOW_DAYS="$choice"
  input "  Stale service days [skip]: "
  read -r choice; [[ -n "$choice" ]] && RESMA_STALE_SERVICE_DAYS="$choice"

  input "  Customize advanced intervals? (y/N) [N]: "
  read -r choice
  if [[ "${choice,,}" == "y" || "${choice,,}" == "yes" ]]; then
    input "    Scheduler poll interval in seconds [skip]: "
    read -r choice; [[ -n "$choice" ]] && RESMA_SCHEDULER_POLL="$choice"
    input "    SSE keepalive interval in seconds [skip]: "
    read -r choice; [[ -n "$choice" ]] && RESMA_SSE_KEEPALIVE="$choice"
  fi
fi

echo "  Stack:    $STACK_NAME"
echo "  Version:  $RESMA_VERSION"
echo "  Prefix:   $IMAGE_PREFIX"
echo

# ---------- 2b. validar ranges das env vars passadas ----------
validate_passed_intervals
success "Interval values validated"

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

# ---------- 5b. aplicar env vars de intervalo passadas (opcional) ----------
# Aplicar apenas as env vars explicitamente passadas (não vazias). As demais
# permanecem com o valor atual do service (padrão `docker service update --env-add`).
# RESMA_COLLECT_INTERVAL é compartilhado entre api e agent (ambos coletam).
APPLIED_ANY=0
apply_env_if_set() {
  local var="$1" svc="$2"
  local val="${!var:-}"
  if [[ -n "$val" ]]; then
    update_env "$svc" "$var" "$val"
    APPLIED_ANY=1
  fi
}

if [[ -n "${RESMA_COLLECT_INTERVAL:-}" ]] \
  || [[ -n "${RESMA_CLUSTER_INTERVAL:-}" ]] \
  || [[ -n "${RESMA_STORAGE_INTERVAL:-}" ]] \
  || [[ -n "${RESMA_AGENT_TASK_POLL_INTERVAL:-}" ]] \
  || [[ -n "${RESMA_ROLLBACK_POLL_INTERVAL:-}" ]] \
  || [[ -n "${RESMA_SCHEDULER_POLL:-}" ]] \
  || [[ -n "${RESMA_SSE_KEEPALIVE:-}" ]] \
  || [[ -n "${RESMA_RETENTION_DAYS:-}" ]] \
  || [[ -n "${RESMA_ANALYSIS_WINDOW_DAYS:-}" ]] \
  || [[ -n "${RESMA_STALE_SERVICE_DAYS:-}" ]]; then
  section "Applying collection interval overrides"
  # API service — todas as env vars de intervalo
  apply_env_if_set RESMA_COLLECT_INTERVAL         api
  apply_env_if_set RESMA_CLUSTER_INTERVAL         api
  apply_env_if_set RESMA_STORAGE_INTERVAL         api
  apply_env_if_set RESMA_AGENT_TASK_POLL_INTERVAL api
  apply_env_if_set RESMA_ROLLBACK_POLL_INTERVAL   api
  apply_env_if_set RESMA_SCHEDULER_POLL           api
  apply_env_if_set RESMA_SSE_KEEPALIVE            api
  apply_env_if_set RESMA_RETENTION_DAYS           api
  apply_env_if_set RESMA_ANALYSIS_WINDOW_DAYS     api
  apply_env_if_set RESMA_STALE_SERVICE_DAYS       api
  # Agent service — apenas RESMA_COLLECT_INTERVAL (compartilhado com api)
  apply_env_if_set RESMA_COLLECT_INTERVAL         agent
  if [[ "$APPLIED_ANY" == "1" ]]; then
    success "Interval overrides applied"
  fi
else
  echo "  No collection interval overrides passed — keeping current values."
fi

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
