#!/usr/bin/env bash
#
# RESMA — Docker Swarm Installer
#
# Uso:
#   curl -fsSL https://raw.githubusercontent.com/USER/resma/main/install.sh | bash
#   ou
#   ./install.sh [flags]
#
# Flags:
#   --jwt-secret <value>  Fornecer JWT secret próprio (default: gerar aleatório)
#   --image <image>       Imagem alternativa (default: ghcr.io/user/resma-api:latest)
#   --port <port>         Porta alternativa (default: 8080)
#   --domain <domain>     Configura Traefik labels com domínio
#   --arm                 Usa imagem ARM64
#   --help                Exibe ajuda

set -euo pipefail

# Defaults
JWT_SECRET=""
IMAGE_PREFIX="ghcr.io/user"
PORT="8080"
DOMAIN=""
ARM=false
STACK_NAME="resma"

# Parse args
while [[ $# -gt 0 ]]; do
  case $1 in
    --jwt-secret)
      JWT_SECRET="$2"; shift 2 ;;
    --image)
      IMAGE_PREFIX="$2"; shift 2 ;;
    --port)
      PORT="$2"; shift 2 ;;
    --domain)
      DOMAIN="$2"; shift 2 ;;
    --arm)
      ARM=true; shift ;;
    --help)
      echo "RESMA Docker Swarm Installer"
      echo ""
      echo "Uso: ./install.sh [flags]"
      echo ""
      echo "Flags:"
      echo "  --jwt-secret <value>  Fornecer JWT secret próprio (default: gerar aleatório)"
      echo "  --image <prefix>      Prefixo de imagem (default: ghcr.io/user)"
      echo "  --port <port>         Porta alternativa (default: 8080)"
      echo "  --domain <domain>     Configura Traefik labels com domínio"
      echo "  --arm                 Usa imagem ARM64"
      echo "  --help                Exibe esta ajuda"
      exit 0 ;;
    *)
      echo "Flag desconhecida: $1"; exit 1 ;;
  esac
done

echo "=== RESMA Docker Swarm Installer ==="
echo ""

# 1. Verificar Docker
if ! command -v docker &>/dev/null; then
  echo "ERROR: Docker não está instalado."
  echo "Instale em: https://docs.docker.com/engine/install/"
  exit 1
fi

# 2. Verificar Swarm
if ! docker info --format '{{.Swarm.LocalNodeState}}' | grep -q "active"; then
  echo "ERROR: Docker Swarm não está ativo neste node."
  echo "Inicie com: docker swarm init"
  exit 1
fi

# Verificar se é manager
if ! docker info --format '{{.Swarm.ControlAvailable}}' | grep -q "true"; then
  echo "ERROR: Este node não é um Swarm manager."
  echo "Execute em um manager node."
  exit 1
fi

echo "[OK] Docker Swarm manager detectado"

# 3. Gerar JWT secret se não fornecido
if [[ -z "$JWT_SECRET" ]]; then
  echo "[INFO] Gerando JWT secret aleatório..."
  JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || python3 -c "import secrets; print(secrets.token_urlsafe(32))")
  if [[ -z "$JWT_SECRET" ]]; then
    echo "ERROR: Falha ao gerar JWT secret. Forneça via --jwt-secret"
    exit 1
  fi
  echo "[OK] JWT secret gerado"
else
  echo "[OK] JWT secret fornecido via flag"
fi

# 4. Criar Docker secret
echo "[INFO] Criando Docker secret resma_jwt_secret..."
if docker secret inspect resma_jwt_secret &>/dev/null; then
  echo "[WARN] Secret resma_jwt_secret já existe — removendo..."
  docker secret rm resma_jwt_secret
fi
echo -n "$JWT_SECRET" | docker secret create resma_jwt_secret -
echo "[OK] Docker secret criado"

# 5. Suffix ARM
SUFFIX=""
if $ARM; then
  SUFFIX="-arm64"
  echo "[INFO] Usando imagens ARM64"
fi

# 6. Pull imagens
echo "[INFO] Baixando imagens..."
docker pull "${IMAGE_PREFIX}/resma-api:latest${SUFFIX}"
docker pull "${IMAGE_PREFIX}/resma-ml:latest${SUFFIX}"
echo "[OK] Imagens baixadas"

# 7. Deploy
echo "[INFO] Deployando stack resma..."
RESMA_REGISTRY="${IMAGE_PREFIX#ghcr.io/}" \
RESMA_CORS_ORIGINS="${RESMA_CORS_ORIGINS:-http://localhost:${PORT}}" \
  docker stack deploy -c docker-stack.yml "$STACK_NAME"
echo "[OK] Stack deployado"

# 8. Aguardar healthcheck
echo "[INFO] Aguardando serviços ficarem healthy..."
sleep 5

for svc in resma-api resma-ml; do
  for i in $(seq 1 30); do
    state=$(docker service inspect "${STACK_NAME}_${svc}" --format '{{.UpdateStatus.State}}' 2>/dev/null || echo "unknown")
    if [[ "$state" == "completed" ]] || [[ "$state" == "running" ]]; then
      echo "[OK] ${svc} está rodando"
      break
    fi
    if [[ $i -eq 30 ]]; then
      echo "[WARN] ${svc} ainda inicializando (state: ${state})"
    fi
    sleep 2
  done
done

# 9. Print URL
MANAGER_IP=$(docker info --format '{{.Swarm.NodeAddr}}' 2>/dev/null || echo "localhost")
echo ""
echo "=== RESMA instalado com sucesso! ==="
echo ""
echo "URL: http://${MANAGER_IP}:${PORT}"
echo ""
echo "Próximos passos:"
echo "  1. Acesse a URL acima no navegador"
echo "  2. Complete o onboarding (criar usuário admin)"
echo "  3. Configure API keys se necessário"
echo ""
echo "Logs: docker service logs ${STACK_NAME}_resma-api"
echo "Logs ML: docker service logs ${STACK_NAME}_resma-ml"
echo ""
echo "Para remover: docker stack rm ${STACK_NAME}"
