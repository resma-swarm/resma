#!/usr/bin/env bash
#
# RESMA Installer — roda dentro de um container Docker com docker.sock montado.
# Gera secrets, faz pull das imagens, deploya o stack e aguarda healthcheck.
#
# Uso (one-liner interativo):
#   docker run -it --rm \
#     --name resma-installer \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     resmaswarm/resma-install:latest
#
# Uso com flags (não-interativo, estilo Docker install):
#   docker run -it --rm \
#     --volume /var/run/docker.sock:/var/run/docker.sock \
#     resmaswarm/resma-install:latest \
#     --no-input \
#     --stack-name resma \
#     --port 8080 \
#     --network captain-overlay-network
#
# Flags disponíveis:
#   --no-input                  Desativa prompts interativos (usa defaults ou env vars)
#   --stack-name <name>         Nome do stack (default: resma)
#   --port <port>               Porta publicada (default: 8080)
#   --network <name>            Rede overlay externa adicional (repeatable ou CSV)
#   --image-prefix <prefix>     Prefixo das imagens (default: docker.io/resmaswarm)
#   --version <version>         Versão das imagens (default: latest)
#   --help, -h                  Mostra esta ajuda
#
# Compatibilidade com env vars (flags têm precedência):
#   -e INTERACTIVE=0            Equivalente a --no-input
#   -e STACK_NAME=resma         Equivalente a --stack-name
#   -e APP_PORT=8080            Equivalente a --port
#   -e IMAGE_PREFIX=...         Equivalente a --image-prefix
#
# Intervalos de coleta (via env vars, sobrescrevíveis):
#   RESMA_COLLECT_INTERVAL         (default 15s)  — coleta de containers
#   RESMA_CLUSTER_INTERVAL         (default 30s)  — info do cluster Swarm
#   RESMA_STORAGE_INTERVAL         (default 60s)  — métricas de storage
#   RESMA_AGENT_TASK_POLL_INTERVAL (default 15s)  — poll de tasks + health agents
#   RESMA_ROLLBACK_POLL_INTERVAL   (default 30s)  — poll do rollback watcher
#   RESMA_SCHEDULER_POLL           (default 15s)  — poll do scheduler (avançado)
#   RESMA_SSE_KEEPALIVE            (default 15s)  — keepalive SSE (avançado)
#   RESMA_RETENTION_DAYS           (default 30)   — retenção de métricas (dias)
#   RESMA_ANALYSIS_WINDOW_DAYS     (default 7)    — janela de análise ML (dias)
#   RESMA_STALE_SERVICE_DAYS       (default 7)    — dias sem heartbeat → stale

set -euo pipefail

# ---------- colors ----------
GC='\033[0;32m'; RC='\033[0;31m'; OC='\033[0;33m'; NC='\033[0m'; BC='\033[1m'; UC='\033[4m'
success() { echo -e "${GC}$1${NC}"; }
warning() { echo -e "${OC}$1${NC}"; }
error()   { echo -e "${RC}$1${NC}"; }
title()   { echo -e "${BC}$1${NC}"; }
section() { echo -e "\n${UC}$1${NC}"; }
input()   { printf "$1"; }

# ---------- help ----------
print_help() {
  sed -n '2,40p' "$0" | sed 's/^# \{0,1\}//'
  exit 0
}

# ---------- defaults (env vars, antes do parsing de flags) ----------
INTERACTIVE="${INTERACTIVE:-1}"
STACK_NAME="${STACK_NAME:-resma}"
APP_PORT="${APP_PORT:-8080}"
IMAGE_PREFIX="${IMAGE_PREFIX:-docker.io/resmaswarm}"
VERSION="${VERSION:-latest}"
COMPOSE_FILE="/install/docker-stack.yml"

# ---------- defaults de produção (intervalos de coleta) ----------
# Estes valores são os benchmarks alinhados ao Prometheus/Grafana (ver spec
# phase-intervals-refresh seção 4.1.1). O installer só precisa exportá-los
# antes do `docker stack deploy` — o docker-stack.yml referencia via ${VAR:-default}.
RESMA_COLLECT_INTERVAL="${RESMA_COLLECT_INTERVAL:-15}"
RESMA_CLUSTER_INTERVAL="${RESMA_CLUSTER_INTERVAL:-30}"
RESMA_STORAGE_INTERVAL="${RESMA_STORAGE_INTERVAL:-60}"
RESMA_AGENT_TASK_POLL_INTERVAL="${RESMA_AGENT_TASK_POLL_INTERVAL:-15}"
RESMA_ROLLBACK_POLL_INTERVAL="${RESMA_ROLLBACK_POLL_INTERVAL:-30}"
RESMA_SCHEDULER_POLL="${RESMA_SCHEDULER_POLL:-15}"
RESMA_SSE_KEEPALIVE="${RESMA_SSE_KEEPALIVE:-15}"
RESMA_RETENTION_DAYS="${RESMA_RETENTION_DAYS:-30}"
RESMA_ANALYSIS_WINDOW_DAYS="${RESMA_ANALYSIS_WINDOW_DAYS:-7}"
RESMA_STALE_SERVICE_DAYS="${RESMA_STALE_SERVICE_DAYS:-7}"

# ---------- parse flags ----------
EXTRA_NETWORKS=()
ANY_FLAG_PASSED=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-input)
      INTERACTIVE=0
      ANY_FLAG_PASSED=1
      shift
      ;;
    --network)
      ANY_FLAG_PASSED=1
      shift
      if [[ $# -eq 0 || "$1" == --* ]]; then
        error "ERROR: --network requires a value"
        exit 1
      fi
      IFS=',' read -ra NETS <<< "$1"
      for net in "${NETS[@]}"; do
        EXTRA_NETWORKS+=("$(echo "$net" | xargs)")  # trim whitespace
      done
      shift
      ;;
    --stack-name)
      ANY_FLAG_PASSED=1
      shift
      if [[ $# -eq 0 || "$1" == --* ]]; then
        error "ERROR: --stack-name requires a value"
        exit 1
      fi
      STACK_NAME="$1"
      shift
      ;;
    --port)
      ANY_FLAG_PASSED=1
      shift
      if [[ $# -eq 0 || "$1" == --* ]]; then
        error "ERROR: --port requires a value"
        exit 1
      fi
      APP_PORT="$1"
      shift
      ;;
    --image-prefix)
      ANY_FLAG_PASSED=1
      shift
      if [[ $# -eq 0 || "$1" == --* ]]; then
        error "ERROR: --image-prefix requires a value"
        exit 1
      fi
      IMAGE_PREFIX="$1"
      shift
      ;;
    --version)
      ANY_FLAG_PASSED=1
      shift
      if [[ $# -eq 0 || "$1" == --* ]]; then
        error "ERROR: --version requires a value"
        exit 1
      fi
      VERSION="$1"
      shift
      ;;
    --help|-h)
      print_help
      ;;
    *)
      error "ERROR: Unknown option: $1"
      error "  Run with --help for usage."
      exit 1
      ;;
  esac
done

# Auto-detect: se qualquer flag foi passada, desliga modo interativo.
# Convenção de mercado: flags na linha de comando implicam automação (CI/CD).
if [[ "$ANY_FLAG_PASSED" == "1" && "$INTERACTIVE" == "1" ]]; then
  INTERACTIVE=0
fi

# ---------- banner ----------
cat /install/logo.txt
echo
title "Welcome to RESMA — RESource MAnager for Docker Swarm"
echo "Version: ${VERSION}"
echo

# ---------- validação de ranges ----------
# Evita misconfiguration (ex: COLLECT_INTERVAL=0 que causaria busy loop).
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

validate_intervals() {
  validate_range RESMA_COLLECT_INTERVAL         "$RESMA_COLLECT_INTERVAL"         5  3600 "segundos"
  validate_range RESMA_CLUSTER_INTERVAL         "$RESMA_CLUSTER_INTERVAL"         5  3600 "segundos"
  validate_range RESMA_STORAGE_INTERVAL         "$RESMA_STORAGE_INTERVAL"        10  3600 "segundos"
  validate_range RESMA_AGENT_TASK_POLL_INTERVAL "$RESMA_AGENT_TASK_POLL_INTERVAL" 5   300 "segundos"
  validate_range RESMA_ROLLBACK_POLL_INTERVAL   "$RESMA_ROLLBACK_POLL_INTERVAL"  10   300 "segundos"
  validate_range RESMA_SCHEDULER_POLL           "$RESMA_SCHEDULER_POLL"           5   300 "segundos"
  validate_range RESMA_SSE_KEEPALIVE            "$RESMA_SSE_KEEPALIVE"            5    60 "segundos"
  validate_range RESMA_RETENTION_DAYS           "$RESMA_RETENTION_DAYS"           1  3650 "dias"
  validate_range RESMA_ANALYSIS_WINDOW_DAYS     "$RESMA_ANALYSIS_WINDOW_DAYS"     1   365 "dias"
  validate_range RESMA_STALE_SERVICE_DAYS       "$RESMA_STALE_SERVICE_DAYS"       1   365 "dias"
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

  # Collection intervals — defaults de produção sugeridos entre colchetes.
  # Pressionar Enter aceita o default (não sobrecarrega o usuário comum).
  section "Collection intervals (press Enter for production defaults)"
  input "  Collect interval in seconds [$RESMA_COLLECT_INTERVAL]: "
  read -r choice; RESMA_COLLECT_INTERVAL="${choice:=$RESMA_COLLECT_INTERVAL}"
  input "  Cluster interval in seconds [$RESMA_CLUSTER_INTERVAL]: "
  read -r choice; RESMA_CLUSTER_INTERVAL="${choice:=$RESMA_CLUSTER_INTERVAL}"
  input "  Storage interval in seconds [$RESMA_STORAGE_INTERVAL]: "
  read -r choice; RESMA_STORAGE_INTERVAL="${choice:=$RESMA_STORAGE_INTERVAL}"
  input "  Agent task poll interval in seconds [$RESMA_AGENT_TASK_POLL_INTERVAL]: "
  read -r choice; RESMA_AGENT_TASK_POLL_INTERVAL="${choice:=$RESMA_AGENT_TASK_POLL_INTERVAL}"
  input "  Rollback poll interval in seconds [$RESMA_ROLLBACK_POLL_INTERVAL]: "
  read -r choice; RESMA_ROLLBACK_POLL_INTERVAL="${choice:=$RESMA_ROLLBACK_POLL_INTERVAL}"
  input "  Retention days [$RESMA_RETENTION_DAYS]: "
  read -r choice; RESMA_RETENTION_DAYS="${choice:=$RESMA_RETENTION_DAYS}"
  input "  Analysis window days [$RESMA_ANALYSIS_WINDOW_DAYS]: "
  read -r choice; RESMA_ANALYSIS_WINDOW_DAYS="${choice:=$RESMA_ANALYSIS_WINDOW_DAYS}"
  input "  Stale service days [$RESMA_STALE_SERVICE_DAYS]: "
  read -r choice; RESMA_STALE_SERVICE_DAYS="${choice:=$RESMA_STALE_SERVICE_DAYS}"

  # Intervalos avançados ficam atrás de um sub-prompt para não sobrecarregar.
  input "  Customize advanced intervals? (y/N) [N]: "
  read -r choice
  if [[ "${choice,,}" == "y" || "${choice,,}" == "yes" ]]; then
    input "    Scheduler poll interval in seconds [$RESMA_SCHEDULER_POLL]: "
    read -r choice; RESMA_SCHEDULER_POLL="${choice:=$RESMA_SCHEDULER_POLL}"
    input "    SSE keepalive interval in seconds [$RESMA_SSE_KEEPALIVE]: "
    read -r choice; RESMA_SSE_KEEPALIVE="${choice:=$RESMA_SSE_KEEPALIVE}"
  fi
else
  echo "Stack name: $STACK_NAME"
  echo "Application port: $APP_PORT"
  if [[ ${#EXTRA_NETWORKS[@]} -gt 0 ]]; then
    echo "Extra networks: ${EXTRA_NETWORKS[*]}"
  fi
  if docker stack ps "$STACK_NAME" >/dev/null 2>&1; then
    warning "Stack [$STACK_NAME] already exists!"
    error "SETUP FAILED — choose a different stack name."
    exit 1
  fi
  echo "Collection intervals (production defaults):"
  echo "  COLLECT=$RESMA_COLLECT_INTERVALs CLUSTER=$RESMA_CLUSTER_INTERVALs STORAGE=$RESMA_STORAGE_INTERVALs"
  echo "  AGENT_TASK_POLL=$RESMA_AGENT_TASK_POLL_INTERVALs ROLLBACK_POLL=$RESMA_ROLLBACK_POLL_INTERVALs"
  echo "  SCHEDULER_POLL=$RESMA_SCHEDULER_POLLs SSE_KEEPALIVE=$RESMA_SSE_KEEPALIVEs"
  echo "  RETENTION=${RESMA_RETENTION_DAYS}d ANALYSIS_WINDOW=${RESMA_ANALYSIS_WINDOW_DAYS}d STALE_SERVICE=${RESMA_STALE_SERVICE_DAYS}d"
fi

# ---------- 2b. validar ranges dos intervalos ----------
validate_intervals
success "Interval values validated"

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

# ---------- 4b. gerar override de redes externas ----------
# Se --network foi passada, gera um override file que adiciona as redes
# externas a TODOS os serviços (api, ml, agent). O override inclui resma-net
# porque Docker Compose substitui (não faz merge) a lista de networks.
OVERRIDE_FILE=""
if [[ ${#EXTRA_NETWORKS[@]} -gt 0 ]]; then
  OVERRIDE_FILE="/tmp/resma-networks-override.yml"
  {
    echo "services:"
    for svc in api ml agent; do
      echo "  $svc:"
      echo "    networks:"
      echo "      - resma-net"
      for net in "${EXTRA_NETWORKS[@]}"; do
        echo "      - $net"
      done
    done
    echo ""
    echo "networks:"
    for net in "${EXTRA_NETWORKS[@]}"; do
      echo "  $net:"
      echo "    external: true"
    done
  } > "$OVERRIDE_FILE"
  success "Networks override generated: ${EXTRA_NETWORKS[*]}"
fi

# ---------- 5. pull imagens ----------
section "Pulling images"
echo "  - ${IMAGE_PREFIX}/resma-api:${VERSION}"
docker pull "${IMAGE_PREFIX}/resma-api:${VERSION}"
echo "  - ${IMAGE_PREFIX}/resma-ml:${VERSION}"
docker pull "${IMAGE_PREFIX}/resma-ml:${VERSION}"
echo "  - ${IMAGE_PREFIX}/resma-agent:${VERSION}"
docker pull "${IMAGE_PREFIX}/resma-agent:${VERSION}"
success "Images pulled"

# ---------- 6. deploy ----------
section "Deploying stack"
# Exportar todas as env vars de intervalo — o docker-stack.yml referencia via
# ${VAR:-default}, e `docker stack deploy` herda o env do shell.
export RESMA_COLLECT_INTERVAL RESMA_CLUSTER_INTERVAL RESMA_STORAGE_INTERVAL \
       RESMA_AGENT_TASK_POLL_INTERVAL RESMA_ROLLBACK_POLL_INTERVAL \
       RESMA_SCHEDULER_POLL RESMA_SSE_KEEPALIVE \
       RESMA_RETENTION_DAYS RESMA_ANALYSIS_WINDOW_DAYS RESMA_STALE_SERVICE_DAYS

DEPLOY_CMD="docker stack deploy -c \"$COMPOSE_FILE\""
if [[ -n "$OVERRIDE_FILE" ]]; then
  DEPLOY_CMD="$DEPLOY_CMD -c \"$OVERRIDE_FILE\""
fi
DEPLOY_CMD="$DEPLOY_CMD \"$STACK_NAME\""

RESMA_CORS_ORIGINS="${RESMA_CORS_ORIGINS:-http://localhost:${APP_PORT}}" \
  eval "$DEPLOY_CMD"
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
