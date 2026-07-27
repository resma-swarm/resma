# Análise Competitiva — RESMA vs Ferramentas Similares

> Research consolidada pelo Technical Council com base em pesquisa web (Jul 2026).

## Ferramentas analisadas

| Ferramenta | Stack | DB | Agent | Deploy | Porta | Stars/Releases |
|------------|-------|-----|-------|--------|-------|----------------|
| **Swarmpit** | Clojure + React | CouchDB + InfluxDB | Sim (global) | 4 serviços | 888 | v1.10 (Abr 2026) |
| **Portainer** | Go + React | Interna | Sim (global) | Server + Agent | 9443 | CE limitado a 5 nodes |
| **Cetacean** | Go + React | Nenhuma (in-memory) | Não | Single container | 8080 | v0.11.2 (Mai 2026) |
| **RESMA** | Go (API) + Python (ML sidecar) + React | DuckDB (embedded) | Não | 2 containers | 8080 | v0.1.0 |

## Comparativo de features

| Feature | Swarmpit | Portainer | Cetacean | RESMA |
|---------|----------|-----------|----------|-------|
| **Stack management** | ✅ | ✅ | ✅ | ❌ |
| **Service CRUD** | ✅ | ✅ | ✅ | ❌ |
| **Resource monitoring** | ✅ InfluxDB | ✅ Built-in | ✅ Prometheus | ✅ DuckDB |
| **ML recommendations** | ❌ | ❌ | ❌ | ✅ **Único** |
| **Memory leak detection** | ❌ | ❌ | ❌ | ✅ **Único** |
| **OOM tracking** | ❌ | ❌ | ❌ | ✅ **Único** |
| **Scheduled changes** | ❌ | ❌ | ❌ | ✅ **Único** |
| **Templates YAML** | ❌ | ✅ | ❌ | ✅ |
| **Log viewer** | ✅ | ✅ | ✅ SSE | ❌ |
| **Topology views** | ❌ | ❌ | ✅ Logical + Physical | ❌ |
| **Real-time updates** | Polling | Polling | ✅ SSE | Polling |
| **REST API** | ✅ Swagger | ✅ | ✅ OpenAPI | ✅ OpenAPI |
| **MCP server** | ✅ | ❌ | ❌ | ❌ |
| **Multi-user** | ✅ | ✅ | ✅ | ✅ |
| **Private registry** | ✅ | ✅ | ❌ | ❌ |
| **Autoredeploy** | ✅ | ❌ | ❌ | ❌ |
| **ARM support** | ✅ | ✅ | ✅ | ✅ (Go + Python) |
| **Single container** | ❌ | ❌ | ✅ | ❌ (2 containers leves) |
| **No external DB** | ❌ | ❌ | ✅ | ✅ DuckDB embedded |
| **No agent needed** | ❌ | ❌ | ✅ | ✅ |
| **Healthcheck nativo** | ❌ | ✅ | ✅ | ✅ (Fase 4) |
| **Traefik integration** | ✅ Labels | ✅ Labels | ❌ | ❌ |

## Análise por ferramenta

### Swarmpit — Referência principal

**O que pegar de referência:**
1. **Installer via Docker run** — `docker run -it --rm --volume /var/run/docker.sock:/var/run/docker.sock swarmpit/install:1.10` — guia interativo de setup
2. **Manual install via stack deploy** — `git clone` + `docker stack deploy -c docker-compose.yml swarmpit`
3. **docker-compose.yml com 4 serviços** — app + agent (global) + db + influxdb, com resource limits e placement constraints
4. **Traefik labels no deploy** — integração nativa com Traefik para HTTPS automático
5. **REST API com Swagger** — tudo que a UI faz é exposto via API
6. **MCP server** — integração com LLMs (Claude Code, opencode)
7. **User config via YAML** — `users.yaml` via docker config para automação
8. **Roadmap público** — `ROADMAP.md` no repo com what's done + what's planned
9. **ARM support** — compose file separado `docker-compose.arm.yml`
10. **Resource monitoring** — InfluxDB para stats históricas, agente global para coleta

**O que RESMA já faz melhor:**
- DuckDB embedded (sem InfluxDB externo)
- 2 containers leves (sem 4 serviços)
- Sem agent (acessa Docker socket diretamente)
- ML para recomendações (Swarmpit não tem)
- Detecção de memory leaks (Swarmpit não tem)
- OOM tracking (Swarmpit não tem)
- Scheduled changes (Swarmpit não tem)

**O que RESMA pode aprender:**
- Installer interativo via `docker run`
- Traefik labels no stack file
- MCP server para LLM integration
- ARM compose file separado
- Roadmap público no repo
- User config via YAML para automação

### Portainer — Referência enterprise

**O que pegar de referência:**
1. **Resource limits UI** — interface visual para CPU/memória limits e reservations
2. **Service resource management** — editar limits diretamente na UI
3. **Stack deploy via UI** — upload de compose file e deploy
4. **Prometheus + Grafana integration** — stack de monitoring completo
5. **Alert rules** — node down, high CPU, high memory, quorum risk
6. **API programável** — `curl` com Bearer token para automação

**O que RESMA pode aprender:**
- Integração com Prometheus + Grafana como complemento
- Alert rules para condições críticas (node down, quorum, OOM)
- API token sem expiração opcional para automação

### Cetacean — Referência lightweight

**O que pegar de referência:**
1. **Single binary, zero config** — deploy 1 container, sem DB, sem agent
2. **SSE real-time** — updates push em vez de polling
3. **Topology views** — logical (service-to-service via overlay) + physical (task-to-node)
4. **Log viewer** — live-streaming com regex search e JSON formatting
5. **Pluggable auth** — anonymous, OIDC, Tailscale, mTLS, proxy headers
6. **OpenAPI spec** — API REST com search, filtering, pagination, SSE streaming
7. **Tabela comparativa no README** — compara com Portainer e Swarmpit

**O que RESMA pode aprender:**
- SSE para real-time em vez de polling (frontend)
- Topology views (logical + physical)
- Log viewer com search
- Tabela comparativa no README (diferencial competitivo)
- Pluggable auth (além de JWT)

## Diferenciais do RESMA (unique selling points)

| Diferencial | Descrição | Nenhuma ferramenta tem |
|-------------|-----------|----------------------|
| **ML Recommendations** | Recomendações de CPU/memória baseadas em percentis + outliers + forecast | ✅ |
| **Memory Leak Detection** | Regressão linear + R² para detectar leaks antes do OOM | ✅ |
| **OOM Tracking** | Histórico de eventos OOM por serviço | ✅ |
| **Scheduled Changes** | Agendar mudanças de recursos para horários de baixo tráfego | ✅ |
| **DuckDB Embedded** | OLAP sem servidor externo (vs InfluxDB do Swarmpit) | Parcial (Cetacean não tem DB) |
| **Change Log / Audit** | Log de quem mudou o quê e quando | ❌ (Portainer tem parcial) |

## Recomendações para as specs

### Atualizações na Fase 1 (Legal/Comunidade)
- **README.md** deve incluir tabela comparativa (estilo Cetacean) vs Swarmpit/Portainer
- **ROADMAP.md** no repo (estilo Swarmpit) com "what we did" + "what's planned"

### Atualizações na Fase 4 (Docker Swarm Install)
- **Installer interativo** via `docker run` (estilo Swarmpit) além do `install.sh`
- **Traefik labels** opcionais no `docker-stack.yml` para HTTPS automático
- **ARM compose file** separado (`docker-stack.arm.yml`)
- **Healthcheck nativo** (Swarmpit não tem, Cetacean tem — RESMA deve ter)

### Atualizações na Fase 6 (Benchmarking)
- **Comparativo de overhead** vs Swarmpit (4 serviços + InfluxDB) e Cetacean (single binary)
- RESMA usa 2 containers leves (Go API + ML sidecar) vs Swarmpit 4 containers — benchmark deve mostrar isso como vantagem
- Medir overhead comparativo: RESMA (DuckDB) vs Swarmpit (CouchDB + InfluxDB)

### Novas features para roadmap futuro (pós-open-source)
- **SSE real-time** no frontend (estilo Cetacean) em vez de polling
- **Log viewer** com live-streaming e search
- **Topology views** (logical + physical)
- **MCP server** para integração com LLMs (estilo Swarmpit)
- **Traefik integration** com labels no deploy
- **Prometheus + Grafana** como complemento opcional
- **Alert rules** para node down, quorum, OOM, memory leak detectado
- **Pluggable auth** — OIDC, Tailscale, mTLS além de JWT
- **Private registry** integration
- **Autoredeploy** — detectar novas imagens e atualizar serviços
