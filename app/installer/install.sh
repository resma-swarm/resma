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
#   --port <port>               Porta publicada da UI/API (default: 8080)
#   --domain <domain>           Domínio público para Traefik + TLS (default: vazio = sem proxy)
#   --tls <auto|internal|none>  Modo TLS (default: auto — ACME se domínio público, internal se .local)
#   --disable-ui                Desabilita o frontend (modo headless para CLI-only)
#   --install-cli               Instala o binário resma-cli em /usr/local/bin do host
#   --network <name>            Rede overlay externa adicional (repeatable ou CSV)
#   --image-prefix <prefix>     Prefixo das imagens (default: docker.io/resmaswarm)
#   --version <version>         Versão das imagens (default: latest)
#   --help, -h                  Mostra esta ajuda
#
# Compatibilidade com env vars (flags têm precedência):
#   -e INTERACTIVE=0            Equivalente a --no-input
#   -e STACK_NAME=resma         Equivalente a --stack-name
#   -e APP_PORT=8080            Equivalente a --port
#   -e RESMA_DOMAIN=foo.com     Equivalente a --domain
#   -e RESMA_TLS_MODE=auto      Equivalente a --tls
#   -e RESMA_DISABLE_UI=1       Equivalente a --disable-ui
#   -e RESMA_INSTALL_CLI=1      Equivalente a --install-cli
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

# Proxy / UI / CLI (novos)
RESMA_DOMAIN="${RESMA_DOMAIN:-}"
RESMA_TLS_MODE="${RESMA_TLS_MODE:-auto}"
RESMA_DISABLE_UI="${RESMA_DISABLE_UI:-0}"
RESMA_INSTALL_CLI="${RESMA_INSTALL_CLI:-0}"

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
    --domain)
      ANY_FLAG_PASSED=1
      shift
      if [[ $# -eq 0 || "$1" == --* ]]; then
        error "ERROR: --domain requires a value"
        exit 1
      fi
      RESMA_DOMAIN="$1"
      shift
      ;;
    --tls)
      ANY_FLAG_PASSED=1
      shift
      if [[ $# -eq 0 || "$1" == --* ]]; then
        error "ERROR: --tls requires a value (auto|internal|none)"
        exit 1
      fi
      case "$1" in
        auto|internal|none) RESMA_TLS_MODE="$1" ;;
        *) error "ERROR: --tls must be auto, internal, or none"; exit 1 ;;
      esac
      shift
      ;;
    --disable-ui)
      ANY_FLAG_PASSED=1
      RESMA_DISABLE_UI=1
      shift
      ;;
    --install-cli)
      ANY_FLAG_PASSED=1
      RESMA_INSTALL_CLI=1
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

  # Disable UI? (modo headless para CLI-only)
  input "Disable web UI (headless mode for CLI-only)? (y/N) [N]: "
  read -r choice
  if [[ "${choice,,}" == "y" || "${choice,,}" == "yes" ]]; then
    RESMA_DISABLE_UI=1
  fi

  # Domain — só perguntado se UI não foi desabilitada (ou até mesmo se foi?
  # Decisão: mesmo com --disable-ui, o domain faz sentido para API com TLS).
  # Mas por simplicidade e alinhado ao acordado: disable-ui ignora Traefik.
  if [[ "$RESMA_DISABLE_UI" != "1" ]]; then
    section "Domain (optional)"
    echo "  Leave blank to access via IP:port (no proxy, no TLS)."
    echo "  Enter a domain to enable Traefik reverse proxy + TLS."
    echo "  Examples: resma.local (self-signed), resma.example.com (Let's Encrypt)."
    input "  Domain [blank]: "
    read -r choice
    RESMA_DOMAIN="${choice:-$RESMA_DOMAIN}"
  fi

  # Port — só perguntada se SEM domain (com domain, Traefik ocupa 80/443).
  # Com --disable-ui, a porta ainda é a da API (acesso CLI via IP:porta).
  if [[ -z "$RESMA_DOMAIN" ]]; then
    input "Enter application port [$APP_PORT]: "
    read -r choice
    APP_PORT="${choice:=$APP_PORT}"
  fi

  # Install CLI?
  input "Install resma-cli to /usr/local/bin on this host? (y/N) [N]: "
  read -r choice
  if [[ "${choice,,}" == "y" || "${choice,,}" == "yes" ]]; then
    RESMA_INSTALL_CLI=1
  fi

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
  echo "Domain: ${RESMA_DOMAIN:-<none>}"
  echo "TLS mode: $RESMA_TLS_MODE"
  echo "Disable UI: $RESMA_DISABLE_UI"
  echo "Install CLI: $RESMA_INSTALL_CLI"
  if [[ ${#EXTRA_NETWORKS[@]} -gt 0 ]]; then
    echo "Extra networks: ${EXTRA_NETWORKS[*]}"
  fi
  if docker stack ps "$STACK_NAME" >/dev/null 2>&1; then
    warning "Stack [$STACK_NAME] already exists!"
    error "SETUP FAILED — choose a different stack name."
    exit 1
  fi
  echo "Collection intervals (production defaults):"
  echo "  COLLECT=${RESMA_COLLECT_INTERVAL}s CLUSTER=${RESMA_CLUSTER_INTERVAL}s STORAGE=${RESMA_STORAGE_INTERVAL}s"
  echo "  AGENT_TASK_POLL=${RESMA_AGENT_TASK_POLL_INTERVAL}s ROLLBACK_POLL=${RESMA_ROLLBACK_POLL_INTERVAL}s"
  echo "  SCHEDULER_POLL=${RESMA_SCHEDULER_POLL}s SSE_KEEPALIVE=${RESMA_SSE_KEEPALIVE}s"
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

# ---------- 4. ajustar compose file (porta + UI) ----------
# Só reescreve a porta publicada se não houver domain (com domain, o Traefik
# ocupa 80/443 e a porta 8080 do api fica interna apenas).
if [[ "$APP_PORT" != "8080" && -z "$RESMA_DOMAIN" ]]; then
  sed -i "s|published: 8080|published: $APP_PORT|" "$COMPOSE_FILE"
fi
# Com domain, remover a publicação da porta 8080 (acesso só via Traefik).
if [[ -n "$RESMA_DOMAIN" ]]; then
  sed -i '/published: 8080/d; /mode: host/d' "$COMPOSE_FILE"
fi
# --disable-ui: setar RESMA_WEB_DIR vazio via sed (docker stack deploy não
# faz merge de env vars individuais via override file — sed é mais confiável).
if [[ "$RESMA_DISABLE_UI" == "1" ]]; then
  sed -i 's|RESMA_WEB_DIR: ${RESMA_WEB_DIR:-/app/web}|RESMA_WEB_DIR: ""|' "$COMPOSE_FILE"
  success "UI disabled (headless mode — RESMA_WEB_DIR empty)"
fi

# ---------- 4b. gerar override de redes externas ----------
# Se --network foi passada, gera um override file que adiciona as redes
# externas APENAS ao service api (não ao ml/agent — esses são internos e
# não precisam ser alcançáveis por proxies externos como CapRover).
# O override inclui resma-net porque Docker Compose substitui (não faz merge)
# a lista de networks.
OVERRIDE_FILE=""
if [[ ${#EXTRA_NETWORKS[@]} -gt 0 ]]; then
  OVERRIDE_FILE="/tmp/resma-networks-override.yml"
  {
    echo "services:"
    echo "  api:"
    echo "    networks:"
    echo "      - resma-net"
    for net in "${EXTRA_NETWORKS[@]}"; do
      echo "      - $net"
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

# ---------- 4c. gerar override de proxy (Traefik) ----------
# Se --domain foi passado, gera um override file que:
#   1. Adiciona o service proxy (Traefik v3) na mesma resma-net
#   2. Adiciona labels Traefik no service api para rotear Host(${DOMAIN}) → api:8080
#   3. Configura TLS (ACME se domínio público, internal se .local, none se --tls=none)
# Tudo na mesma rede resma-net — zero configuração de rede cross-stack.
PROXY_OVERRIDE_FILE=""
if [[ -n "$RESMA_DOMAIN" ]]; then
  PROXY_OVERRIDE_FILE="/tmp/resma-proxy-override.yml"

  # Determinar modo TLS:
  #   - --tls=none        → HTTP-only na porta 80, sem TLS
  #   - --tls=internal    → HTTPS na 443 com cert self-signed do Traefik
  #   - --tls=auto        → ACME TLS-ALPN-01 se domínio NÃO termina em .local
  #                        → internal se termina em .local
  USE_ACME=0
  USE_INTERNAL_TLS=0
  case "$RESMA_TLS_MODE" in
    none)      : ;;  # sem TLS
    internal)  USE_INTERNAL_TLS=1 ;;
    auto)
      if [[ "$RESMA_DOMAIN" == *.local ]]; then
        USE_INTERNAL_TLS=1
      else
        USE_ACME=1
      fi
      ;;
  esac

  # Email para ACME — obrigatório se ACME. Default genérico se não fornecido.
  ACME_EMAIL="${RESMA_ACME_EMAIL:-admin@${RESMA_DOMAIN}}"

  {
    echo "services:"
    echo "  api:"
    echo "    deploy:"
    echo "      labels:"
    echo "        - traefik.enable=true"
    echo "        - traefik.http.routers.resma.rule=Host(\`${RESMA_DOMAIN}\`)"
    if [[ "$RESMA_TLS_MODE" == "none" ]]; then
      echo "        - traefik.http.routers.resma.entrypoints=web"
    else
      echo "        - traefik.http.routers.resma.entrypoints=websecure"
      echo "        - traefik.http.routers.resma.tls=true"
      if [[ "$USE_ACME" == "1" ]]; then
        echo "        - traefik.http.routers.resma.tls.certresolver=letsencrypt"
      fi
    fi
    echo "        - traefik.http.services.resma.loadbalancer.server.port=8080"
    # SSE: desabilitar bufferização para streaming funcionar atrás do proxy
    echo "        - traefik.http.middlewares.resma-sse.headers.customResponseHeaders.X-Accel-Buffering=no"
    echo "        - traefik.http.routers.resma.middlewares=resma-sse"
    echo ""
    echo "  proxy:"
    echo "    image: traefik:v3.7.10"
    echo "    command:"
    echo "      - --entrypoints.web.address=:80"
    echo "      - --providers.swarm.endpoint=unix:///var/run/docker.sock"
    echo "      - --providers.swarm.exposedbydefault=false"
    echo "      - --providers.swarm.network=${STACK_NAME}_resma-net"
    echo "      - --log.level=INFO"
    echo "      - --accesslog=true"
    if [[ "$RESMA_TLS_MODE" != "none" ]]; then
      echo "      - --entrypoints.websecure.address=:443"
      echo "      - --entrypoints.web.http.redirections.entrypoint.to=websecure"
      echo "      - --entrypoints.web.http.redirections.entrypoint.scheme=https"
      echo "      - --entrypoints.web.http.redirections.entrypoint.permanent=true"
      if [[ "$USE_ACME" == "1" ]]; then
        echo "      - --certificatesresolvers.letsencrypt.acme.tlschallenge=true"
        echo "      - --certificatesresolvers.letsencrypt.acme.email=${ACME_EMAIL}"
        echo "      - --certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      fi
    fi
    echo "    volumes:"
    echo "      - /var/run/docker.sock:/var/run/docker.sock:ro"
    if [[ "$USE_ACME" == "1" ]]; then
      echo "      - resma-letsencrypt:/letsencrypt"
    fi
    echo "    ports:"
    echo "      - target: 80"
    echo "        published: 80"
    echo "        protocol: tcp"
    echo "        mode: host"
    if [[ "$RESMA_TLS_MODE" != "none" ]]; then
      echo "      - target: 443"
      echo "        published: 443"
      echo "        protocol: tcp"
      echo "        mode: host"
    fi
    echo "    deploy:"
    echo "      placement:"
    echo "        constraints:"
    echo "          - node.role == manager"
    echo "      resources:"
    echo "        reservations:"
    echo "          cpus: \"0.05\""
    echo "          memory: 32M"
    echo "      restart_policy:"
    echo "        condition: on-failure"
    echo "        max_attempts: 3"
    echo "    networks:"
    echo "      - resma-net"
    echo ""
    if [[ "$USE_ACME" == "1" ]]; then
      echo "volumes:"
      echo "  resma-letsencrypt:"
      echo ""
    fi
  } > "$PROXY_OVERRIDE_FILE"

  if [[ "$USE_ACME" == "1" ]]; then
    success "Proxy override generated: Traefik + ACME (Let's Encrypt) for ${RESMA_DOMAIN}"
  elif [[ "$USE_INTERNAL_TLS" == "1" ]]; then
    success "Proxy override generated: Traefik + internal TLS (self-signed) for ${RESMA_DOMAIN}"
  else
    success "Proxy override generated: Traefik HTTP-only for ${RESMA_DOMAIN}"
  fi
fi

# ---------- 5. pull imagens ----------
section "Pulling images"
echo "  - ${IMAGE_PREFIX}/resma-api:${VERSION}"
docker pull "${IMAGE_PREFIX}/resma-api:${VERSION}"
echo "  - ${IMAGE_PREFIX}/resma-ml:${VERSION}"
docker pull "${IMAGE_PREFIX}/resma-ml:${VERSION}"
echo "  - ${IMAGE_PREFIX}/resma-agent:${VERSION}"
docker pull "${IMAGE_PREFIX}/resma-agent:${VERSION}"
if [[ -n "$RESMA_DOMAIN" ]]; then
  echo "  - traefik:v3.7.10"
  docker pull traefik:v3.7.10
fi
if [[ "$RESMA_INSTALL_CLI" == "1" ]]; then
  echo "  - ${IMAGE_PREFIX}/resma-cli:${VERSION}"
  docker pull "${IMAGE_PREFIX}/resma-cli:${VERSION}"
fi
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
if [[ -n "$PROXY_OVERRIDE_FILE" ]]; then
  DEPLOY_CMD="$DEPLOY_CMD -c \"$PROXY_OVERRIDE_FILE\""
fi
DEPLOY_CMD="$DEPLOY_CMD \"$STACK_NAME\""

# CORS: com domain, incluir a URL pública; sem domain, IP:porta.
if [[ -n "$RESMA_DOMAIN" ]]; then
  if [[ "$RESMA_TLS_MODE" == "none" ]]; then
    CORS_URL="http://${RESMA_DOMAIN}"
  else
    CORS_URL="https://${RESMA_DOMAIN}"
  fi
else
  CORS_URL="http://localhost:${APP_PORT}"
fi
RESMA_CORS_ORIGINS="${RESMA_CORS_ORIGINS:-${CORS_URL}}" \
  eval "$DEPLOY_CMD"
success "Stack deployed"

# ---------- 7. aguardar healthcheck ----------
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

# ---------- 7b. aguardar proxy saudável (se habilitado) ----------
if [[ -n "$RESMA_DOMAIN" ]]; then
  printf "  Waiting for proxy"
  MAX_ATTEMPTS_PROXY=24  # ~2 min
  attempt=0
  while true; do
    STATUS=$(curl --unix-socket /var/run/docker.sock -sgG \
      -X GET "http:/v1.24/tasks?filters={\"service\":[\"${STACK_NAME}_proxy\"]}" \
      2>/dev/null | jq -r 'sort_by(.CreatedAt) | .[-1].Status.State' 2>/dev/null || echo "unknown")
    if [[ "$STATUS" == "running" ]]; then
      success "\n  ${STACK_NAME}_proxy is healthy"
      break
    fi
    printf "."
    attempt=$((attempt + 1))
    if [[ $attempt -ge $MAX_ATTEMPTS_PROXY ]]; then
      warning "\n  ${STACK_NAME}_proxy not healthy after ${MAX_ATTEMPTS_PROXY} attempts (continuing)."
      warning "  Check logs: docker service logs ${STACK_NAME}_proxy"
      break
    fi
    sleep 5
  done
fi

# ---------- 7c. instalar CLI no host (se solicitado) ----------
if [[ "$RESMA_INSTALL_CLI" == "1" ]]; then
  section "Installing resma-cli to /usr/local/bin"
  # Detectar arquitetura do host
  HOST_ARCH=$(uname -m 2>/dev/null || echo "x86_64")
  case "$HOST_ARCH" in
    x86_64|amd64) CLI_IMAGE="${IMAGE_PREFIX}/resma-cli:${VERSION}" ;;
    aarch64|arm64) CLI_IMAGE="${IMAGE_PREFIX}/resma-cli:${VERSION}-arm64" ;;
    *) CLI_IMAGE="${IMAGE_PREFIX}/resma-cli:${VERSION}" ;;
  esac
  # Extrair o binário da imagem distroless para /usr/local/bin no host.
  # A imagem CLI é distroless (sem sh) — usamos docker create + docker cp.
  # O installer roda com docker.sock montado do host.
  CLI_TMP_CONTAINER="resma-cli-extract-$$"
  if docker create --name "$CLI_TMP_CONTAINER" "${CLI_IMAGE}" >/dev/null 2>&1; then
    if docker cp "$CLI_TMP_CONTAINER:/resma" /usr/local/bin/resma 2>/dev/null; then
      chmod +x /usr/local/bin/resma 2>/dev/null || true
      docker rm "$CLI_TMP_CONTAINER" >/dev/null 2>&1
      success "resma-cli installed to /usr/local/bin/resma"
      echo "  Run 'resma --help' to verify."
    else
      docker rm "$CLI_TMP_CONTAINER" >/dev/null 2>&1
      warning "Could not copy to /usr/local/bin (host not Linux or permission denied)."
      echo "  Manual install:"
      echo "    docker create --name cli-extract ${CLI_IMAGE}"
      echo "    docker cp cli-extract:/resma /usr/local/bin/resma"
      echo "    docker rm cli-extract"
      echo "    chmod +x /usr/local/bin/resma"
    fi
  else
    warning "Could not pull/create CLI image ${CLI_IMAGE}."
    echo "  Verify the image exists and try again."
  fi
fi

# ---------- 8. done ----------
MANAGER_IP=$(docker info --format '{{.Swarm.NodeAddr}}' 2>/dev/null || echo "localhost")
echo
title "=== RESMA installed successfully! ==="
echo

# URL de acesso — depende da configuração
if [[ -n "$RESMA_DOMAIN" ]]; then
  if [[ "$RESMA_TLS_MODE" == "none" ]]; then
    echo "  URL: http://${RESMA_DOMAIN}"
  else
    echo "  URL: https://${RESMA_DOMAIN}"
    if [[ "$RESMA_DOMAIN" == *.local ]]; then
      echo "  (self-signed certificate — browser will show a warning)"
    fi
  fi
elif [[ "$RESMA_DISABLE_UI" == "1" ]]; then
  echo "  API: http://${MANAGER_IP}:${APP_PORT} (headless — no web UI)"
  echo "  Use the CLI: resma auth login --server http://${MANAGER_IP}:${APP_PORT}"
else
  echo "  URL: http://${MANAGER_IP}:${APP_PORT}"
fi
echo
echo "Next steps:"
if [[ "$RESMA_DISABLE_UI" != "1" ]]; then
  echo "  1. Open the URL above in your browser"
  echo "  2. Complete onboarding (create admin user)"
  echo "  3. Configure API keys if needed"
else
  echo "  1. Create admin user via CLI: resma auth login --server http://${MANAGER_IP}:${APP_PORT}"
  echo "  2. Use 'resma services list', 'resma agents list', etc."
fi
echo
echo "Logs:       docker service logs ${STACK_NAME}_api"
echo "Logs ML:    docker service logs ${STACK_NAME}_ml"
if [[ -n "$RESMA_DOMAIN" ]]; then
  echo "Logs proxy: docker service logs ${STACK_NAME}_proxy"
fi
echo "Remove:     docker stack rm ${STACK_NAME}"
echo
title "Enjoy! :)"
