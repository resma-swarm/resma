# RESMA — Deploy Swarm (PowerShell)
#
# Faz o build das imagens e deploya via `docker stack deploy` usando
# docker-compose.swarm.yml (arquivo dedicado para Swarm, sem profiles/build).
#
# Pré-requisitos:
#   - Docker Swarm ativo no node manager: `docker swarm init`
#   - Variáveis de ambiente (opcional, tem defaults de dev):
#       RESMA_JWT_SECRET, RESMA_AGENT_TOKEN
#
# Uso:
#   .\scripts\deploy-swarm.ps1              # build + deploy
#   .\scripts\deploy-swarm.ps1 -NoBuild     # deploy sem rebuild
#   .\scripts\deploy-swarm.ps1 -Remove      # remove o stack
#   .\scripts\deploy-swarm.ps1 -StackName resma-prod  # nome customizado

param(
    [switch]$NoBuild,
    [switch]$Remove,
    [string]$StackName = "resma"
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) { Write-Host "`n[DEPLOY] $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "[OK]     $msg" -ForegroundColor Green }
function Write-Err($msg)  { Write-Host "[ERROR]  $msg" -ForegroundColor Red; exit 1 }

# 0. Validar Swarm ativo
Write-Step "Validando Docker Swarm..."
$swarmState = docker info --format "{{.Swarm.LocalNodeState}}" 2>$null
if ($swarmState -ne "active") {
    Write-Err "Docker Swarm nao esta ativo (estado: $swarmState). Rode: docker swarm init"
}
Write-Ok "Swarm ativo"

# 1. Remover stack (se -Remove)
if ($Remove) {
    Write-Step "Removendo stack '$StackName'..."
    docker stack rm $StackName
    Write-Ok "Stack removido. Aguarde os services pararem (docker service ls)."
    exit 0
}

# 2. Build das imagens (se nao -NoBuild)
#    docker stack deploy nao suporta build: — as imagens devem ser pre-buildadas.
#    O docker-compose.yml (dev) tem build: com target dev; aqui usamos target runtime.
if (-not $NoBuild) {
    Write-Step "Buildando imagens (target runtime)..."
    docker build -t resma-api:latest -f app/api/Dockerfile --target runtime .
    if ($LASTEXITCODE -ne 0) { Write-Err "Build resma-api falhou" }
    docker build -t resma-ml:latest -f app/ml/Dockerfile .
    if ($LASTEXITCODE -ne 0) { Write-Err "Build resma-ml falhou" }
    docker build -t resma-agent:latest -f app/agent/Dockerfile --target runtime .
    if ($LASTEXITCODE -ne 0) { Write-Err "Build resma-agent falhou" }
    Write-Ok "Imagens buildadas"
}

# 3. Deploy do stack via docker-compose.swarm.yml
Write-Step "Deployando stack '$StackName' via docker stack deploy..."
docker stack deploy -c docker-compose.swarm.yml --with-registry-auth $StackName
if ($LASTEXITCODE -ne 0) { Write-Err "docker stack deploy falhou" }
Write-Ok "Stack deployado"

# 4. Verificar
Write-Step "Services do stack:"
Start-Sleep -Seconds 3
docker service ls | Where-Object { $_ -match $StackName }

Write-Host "`n[DEPLOY] Concluido. Comandos uteis:" -ForegroundColor Cyan
Write-Host "  docker service ls                              # ver todos services"
Write-Host "  docker stack services $StackName               # ver services do stack"
Write-Host "  docker service logs ${StackName}_api --tail 50 -f   # logs da API"
Write-Host "  docker service logs ${StackName}_resma-agent --tail 50 -f  # logs do agent"
Write-Host "  docker stack rm $StackName                     # remover stack"
