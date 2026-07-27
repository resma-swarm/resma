# Fase 4 — Instalação Fácil Docker Swarm

> **Prioridade:** Alta  
> **Esforço:** Médio (Docker + scripts shell)  
> **Bloqueador:** Não  
> **Dependências:** Fase 2 (secrets, env vars)

## Objetivo

Tornar a instalação do RESMA em Docker Swarm o mais simples possível: um comando para deploy, um script one-liner para instalação automatizada, e imagem publicada no GitHub Container Registry.

## Tarefas

### 4.1 — Criar docker-stack.yml

- **Arquivo:** `docker-stack.yml` (raiz)
- **Diferença do `docker-compose.yml`:** Otimizado para `docker stack deploy` (sem `build:`, usa imagem do registry). **2 serviços:** `resma-api` (Go) + `resma-ml` (Python sidecar)
- **Conteúdo:**
  ```yaml
  services:
    resma-api:
      image: docker.io/resmaswarm/resma-api:latest
      ports:
        - target: 8080
          published: 8080
          mode: host
      volumes:
        - /var/run/docker.sock:/var/run/docker.sock:ro
        - resma-data:/data
      environment:
        - RESMA_DB_PATH=/data/resma.duckdb
        - RESMA_ML_URL=http://resma-ml:8081
        - RESMA_JWT_SECRET_FILE=/run/secrets/resma_jwt_secret
        - RESMA_CORS_ORIGINS=${RESMA_CORS_ORIGINS:-http://localhost:8080}
        - RESMA_EXCLUDED_IMAGES=docker.io/resmaswarm/resma-api:latest,docker.io/resmaswarm/resma-ml:latest
      secrets:
        - resma_jwt_secret
      deploy:
        placement:
          constraints:
            - node.role == manager
        resources:
          limits:
            cpus: "0.25"
            memory: 128M
          reservations:
            cpus: "0.05"
            memory: 32M
        restart_policy:
          condition: on-failure
          max_attempts: 3
      healthcheck:
        test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
        interval: 30s
        timeout: 5s
        retries: 3
        start_period: 10s

    resma-ml:
      image: docker.io/resmaswarm/resma-ml:latest
      volumes:
        - resma-data:/data
      environment:
        - RESMA_DB_PATH=/data/resma.duckdb
      deploy:
        resources:
          limits:
            cpus: "0.25"
            memory: 256M
          reservations:
            cpus: "0.05"
            memory: 64M
        restart_policy:
          condition: on-failure
          max_attempts: 3
      healthcheck:
        test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8081/health')"]
        interval: 30s
        timeout: 5s
        retries: 3

  volumes:
    resma-data:

  secrets:
    resma_jwt_secret:
      external: true
  ```
- **Notas:**
  - Usa `image:` em vez de `build:` (imagens do GHCR)
  - Usa Docker secret para JWT
  - Healthcheck nativo do Docker — path `/health` (não `/api/health` — pós-0b a API Go usa `/health`)
  - `resma-ml` não expõe porta externamente (comunicação interna via rede Docker)
  - Placement do `resma-api` no manager node (acesso ao Docker socket); `resma-ml` pode rodar em qualquer node
- **Referência Swarmpit:** O `docker-compose.yml` do Swarmpit usa 4 serviços (app + agent + CouchDB + InfluxDB). RESMA precisa de 2 serviços (API + ML) — ainda diferencial competitivo importante
- **Traefik labels opcionais** (inspirado em Swarmpit/DockerSwarm.rocks):
  ```yaml
  deploy:
    labels:
      - traefik.enable=true
      - traefik.http.routers.resma.rule=Host(${RESMA_DOMAIN:-resma.local})
      - traefik.http.routers.resma.entrypoints=http
      - traefik.http.services.resma.loadbalancer.server.port=8080
  ```
  - Incluir como bloco comentado no `docker-stack.yml` (aplica apenas ao `resma-api`)
  - **SSE via Traefik:** adicionar labels para desabilitar bufferização em `/api/sse/*`:
    ```yaml
    - traefik.http.middlewares.resma-sse.headers.customResponseHeaders.X-Accel-Buffering=no
    - traefik.http.routers.resma-sse.rule=Host(`${RESMA_DOMAIN:-resma.local}`) && PathPrefix(`/api/sse/`)
    - traefik.http.routers.resma-sse.middlewares=resma-sse
    ```
  - **Nginx (se usado em vez de Traefik):** `proxy_buffering off; proxy_read_timeout 24h;` para `/api/sse/`
- **ARM support** (inspirado em Swarmpit `docker-compose.arm.yml`):
  - Criar `docker-stack.arm.yml` com imagem ARM (`docker.io/resmaswarm/resma-api:latest-arm64`, `docker.io/resmaswarm/resma-ml:latest-arm64`)
  - Ou usar multi-arch manifest no GHCR (preferível — 1 imagem para todas as arquiteturas)

### 4.2 — Criar install.sh + Installer interativo

- **Arquivo:** `install.sh` (raiz)
- **One-liner:** `curl -fsSL https://raw.githubusercontent.com/resma-swarm/resma/main/install.sh | bash`
- **Script faz:**
  1. Verifica se Docker está instalado e é manager do Swarm
  2. Gera JWT secret aleatório se não fornecido
  3. Cria Docker secret: `echo "$SECRET" | docker secret create resma_jwt_secret -`
  4. Faz pull das imagens: `docker pull docker.io/resmaswarm/resma-api:latest` e `docker pull docker.io/resmaswarm/resma-ml:latest`
  5. Deploy: `docker stack deploy -c docker-stack.yml resma`
  6. Aguarda healthcheck: `docker service inspect resma_resma-api --format '{{.UpdateStatus.State}}'` e `docker service inspect resma_resma-ml --format '{{.UpdateStatus.State}}'`
  7. Printa URL: `http://<manager-ip>:8080`
  8. Printa instruções para criar usuário admin no first boot
- **Flags:**
  - `--jwt-secret <value>` — fornecer secret próprio
  - `--image <image>` — imagem alternativa
  - `--port <port>` — porta alternativa
  - `--domain <domain>` — configura Traefik labels com domínio
  - `--arm` — usa imagem ARM64
  - `--help` — ajuda

- **Referência Swarmpit — Installer via Docker run:**
  - Swarmpit oferece `docker run -it --rm --volume /var/run/docker.sock:/var/run/docker.sock swarmpit/install:1.10`
  - RESMA pode oferecer installer similar: `docker run -it --rm --volume /var/run/docker.sock:/var/run/docker.sock docker.io/resmaswarm/resma-installer:latest`
  - Installer interativo pergunta: domínio, porta, JWT secret (gerar ou fornecer), Traefik (sim/não)
  - **Decisão:** Implementar como tarefa futura (pós-v0.1.0) — `install.sh` é suficiente para primeira release

### 4.3 — Docker Swarm secrets

- **Setup manual documentado:**
  ```bash
  # Generate secret
  openssl rand -base64 32 | docker secret create resma_jwt_secret -

  # Or use install.sh which does this automatically
  ```
- **Suporte `_FILE` no config:**
  - **Arquivo:** `app/api/internal/config/config.go`
  - Adicionar lógica: se `RESMA_JWT_SECRET_FILE` existir, ler secret do arquivo em vez de `RESMA_JWT_SECRET`
  ```go
  if f := os.Getenv("RESMA_JWT_SECRET_FILE"); f != "" {
      b, err := os.ReadFile(f)
      if err != nil { return nil, err }
      cfg.JWTSecret = strings.TrimSpace(string(b))
  }
  ```

### 4.4 — Healthcheck no Dockerfile

- **Arquivo:** `app/api/Dockerfile` (target `runtime`)
- O runtime `debian:bookworm-slim` não tem curl/wget por padrão. Opções:
  1. Instalar `wget` no runtime stage (leve): `RUN apt-get update && apt-get install -y --no-install-recommends wget && rm -rf /var/lib/apt/lists/*`
  2. Usar healthcheck externo (Docker Swarm service healthcheck via `docker service inspect`)
  3. Compilar um binário tiny de healthcheck no builder stage
- **Recomendado:** instalar `wget` no runtime + `HEALTHCHECK` no Dockerfile:
  ```dockerfile
  RUN apt-get update && apt-get install -y --no-install-recommends wget && rm -rf /var/lib/apt/lists/*
  HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD wget -qO- http://localhost:8080/health || exit 1
  ```
- **Path:** `/health` (não `/api/health` — pós-0b a API Go usa `/health` para liveness, `/ready` para readiness com deps)

### 4.5 — docker-compose.override.yml para desenvolvimento

- **Arquivo:** `docker-compose.override.yml` (raiz) — opcional, sobrescreve o `docker-compose.yml` principal
- **Conteúdo:**
  ```yaml
  services:
    api:
      build:
        target: dev
      volumes:
        - ./app/api:/src
        - go-mod-cache:/go/pkg/mod
        - go-build-cache:/root/.cache/go-build
      environment:
        - RESMA_JWT_SECRET=dev-secret-change-me
        - RESMA_CORS_ORIGINS=http://localhost:5173,http://localhost:8080
      command: ["air"]
  ```
- **Nota:** Docker Compose mescla override automaticamente com `docker-compose.yml` quando ambos existem

### 4.6 — Documentar instalação no README

- **Seção no README.md (raiz):**
  ```markdown
  ## Quick Start (Docker Swarm)

  ```bash
  # 1. Initialize Swarm (if not already)
  docker swarm init

  # 2. Deploy RESMA
  curl -fsSL https://raw.githubusercontent.com/resma-swarm/resma/main/install.sh | bash

  # 3. Access
  open http://localhost:8080
  ```
  ```
- Também documentar instalação manual sem script
- Documentar que 2 containers são criados: `resma-api` (Go) + `resma-ml` (Python ML sidecar)

### 4.7 — Publicar imagens no GitHub Container Registry

- **Arquivo:** `.github/workflows/release.yml` (ver Fase 5)
- **2 imagens:** `docker.io/resmaswarm/resma-api:latest` (Go) + `docker.io/resmaswarm/resma-ml:latest` (Python)
- **Workflow:**
  ```yaml
  - name: Login to GHCR
    uses: docker/login-action@v3
    with:
      registry: ghcr.io
      username: ${{ github.actor }}
      password: ${{ secrets.GITHUB_TOKEN }}

  - name: Build and push API (Go)
    uses: docker/build-push-action@v6
    with:
      context: .
      file: app/api/Dockerfile
      target: runtime
      push: true
      tags: ghcr.io/${{ github.repository }}-api:latest,ghcr.io/${{ github.repository }}-api:${{ github.ref_name }}

  - name: Build and push ML (Python)
    uses: docker/build-push-action@v6
    with:
      context: .
      file: app/ml/Dockerfile
      push: true
      tags: ghcr.io/${{ github.repository }}-ml:latest,ghcr.io/${{ github.repository }}-ml:${{ github.ref_name }}
  ```
- **Configurar:** Package visibility = public no GitHub para ambos os packages
- **Documentar:** `docker pull docker.io/resmaswarm/resma-api:latest` e `docker pull docker.io/resmaswarm/resma-ml:latest`

## Critérios de aceite

- [ ] `docker-stack.yml` funcional com `docker stack deploy` (2 serviços: resma-api + resma-ml)
- [ ] `install.sh` funciona com one-liner `curl | bash`
- [ ] Docker Swarm secrets implementado e documentado (suporte `_FILE` no config Go)
- [ ] Healthcheck no Dockerfile (target runtime) com path `/health`
- [ ] `docker-compose.override.yml` para dev (hot reload via air)
- [ ] README com quick start de 1 comando
- [ ] 2 imagens publicadas no GHCR: `resma-api` (Go) + `resma-ml` (Python)
- [ ] `docker service inspect resma_resma-api` mostra status healthy
- [ ] `docker service inspect resma_resma-ml` mostra status healthy

## Estrutura de arquivos resultante

```
raiz/
├── docker-compose.yml           ← dev + prod (profiles, usa app/api/Dockerfile)
├── docker-compose.override.yml  ← dev overrides (hot reload via air) — opcional
├── docker-stack.yml             ← produção (2 serviços, image: do GHCR, Traefik opcional)
├── docker-stack.arm.yml         ← ARM64 (se não usar multi-arch manifest)
├── app/api/Dockerfile           ← Go API (multi-stage: builder/dev/runtime) com healthcheck
├── app/ml/Dockerfile            ← Python ML sidecar com healthcheck
├── install.sh                   ← one-liner installer
└── .env.example                 ← todas as env vars
```

## Referências competitivas

- **Swarmpit:** Installer via `docker run` interativo, `docker stack deploy` manual, 4 serviços (app+agent+db+influx), Traefik labels, ARM compose separado, ROADMAP.md público
- **Cetacean:** Single container sem DB nem agent, deploy em segundos, zero config
- **Portainer:** Server + Agent global, resource limits UI, stack deploy via UI, Prometheus integration

Ver [competitive-analysis.md](../competitive-analysis.md) para comparativo completo.
