# Fase 6 — Benchmarking Docker Swarm

> **Prioridade:** Média  
> **Esforço:** Médio (scripts shell + documentação)  
> **Bloqueador:** Não  
> **Dependências:** Fase 4 (docker-stack.yml)

## Objetivo

Criar ferramentas de benchmarking nativas para Docker Swarm que permitam medir overhead do orquestrador, performance do cluster e overhead do próprio RESMA. Usar ferramentas nativas do Docker Swarm sem dependências externas pesadas.

## Contexto da pesquisa

Segundo benchmarks de 2026 (TechPlained):
- CPU overhead do Swarm: < 2% em todos os testes
- Memory footprint: ~135 MB no manager idle, ~3.2 MB por container
- Overlay network: 18-22% menos throughput vs host networking
- p99 latency: +0.4-0.7 ms por hop

RESMA deve oferecer scripts que reproduzam esses benchmarks em qualquer cluster.

## Benchmark comparativo vs ferramentas similares

| Ferramenta | Containers | DB externo | Agent | Overhead estimado |
|------------|-----------|------------|-------|-------------------|
| **Swarmpit** | 4 (app + agent + CouchDB + InfluxDB) | Sim (2) | Sim (global) | Alto — 4 containers + 2 DBs |
| **Portainer** | 2+ (server + agent global) | Sim | Sim (global) | Médio — agent em todo node |
| **Cetacean** | 1 | Não (in-memory) | Não | Baixo — single binary |
| **RESMA** | 2 (Go API + ML sidecar) | Não (DuckDB embedded) | Não | Baixo — 2 containers leves |

O `resma-overhead.sh` deve medir e comparar:
- CPU/memória do container RESMA vs limites definidos no stack
- Footprint total (RESMA 2 containers vs Swarmpit 4 containers)
- DuckDB disk I/O vs InfluxDB (se Swarmpit estiver no mesmo cluster para comparação)

## Tarefas

### 6.1 — benchmark/swarm-benchmark.sh

- **Arquivo:** `benchmark/swarm-benchmark.sh`
- **Objetivo:** Medir overhead de CPU/memória do Docker Swarm vs bare metal
- **Ferramentas usadas:**
  - `sysbench` — CPU prime calculation (single e multi-thread)
  - `stress-ng` — memória e I/O
  - `dd` — throughput de disco
  - `docker stats` — coleta de métricas dos containers
- **Funcionamento:**
  1. Deploya stack de benchmark via `docker stack deploy -c benchmark/stack-benchmark.yml benchmark`
  2. Executa testes dentro de containers Swarm
  3. Coleta resultados via `docker service logs`
  4. Compara com baseline bare metal (executado no host)
  5. Gera relatório JSON + tabela no stdout
- **Parâmetros:**
  - `--duration <seconds>` — duração de cada teste (default: 60)
  - `--threads <n>` — threads para CPU test (default: número de CPUs)
  - `--output <file>` — salvar resultados em JSON
- **Métricas coletadas:**

| Métrica | Ferramenta | Descrição |
|---------|-----------|-----------|
| CPU single-thread | sysbench | events/sec |
| CPU multi-thread | sysbench | events/sec |
| Memory throughput | stress-ng | MB/sec |
| Disk I/O write | dd | MB/sec |
| Disk I/O read | dd | MB/sec |
| Network latency | ping | ms p50/p95/p99 |
| Container overhead | docker stats | CPU% e MB do daemon |

### 6.2 — benchmark/resma-overhead.sh

- **Arquivo:** `benchmark/resma-overhead.sh`
- **Objetivo:** Medir overhead do próprio RESMA no cluster
- **Funcionamento:**
  1. Verifica que RESMA está rodando: `docker service inspect resma_resma`
  2. Coleta `docker stats` do container RESMA por N segundos
  3. Calcula: CPU média, CPU p95, memória média, memória p99
  4. Compara com limites definidos no `docker-stack.yml` (0.50 CPU, 512M)
  5. Gera relatório com % de utilização dos limites
- **Saída de exemplo:**
  ```
  RESMA Overhead Report (60s sample)
  ====================================
  CPU avg:    2.3%  (limit: 50.0%)  [4.6% of limit]
  CPU p95:    5.1%  (limit: 50.0%)  [10.2% of limit]
  Memory avg: 78 MB (limit: 512 MB) [15.2% of limit]
  Memory p99: 92 MB (limit: 512 MB) [17.9% of limit]

  Status: HEALTHY — well within resource limits
  ```

### 6.3 — benchmark/stack-benchmark.yml

- **Arquivo:** `benchmark/stack-benchmark.yml`
- **Stack de teste com serviços de carga controlada:**
  ```yaml
  services:
    cpu-benchmark:
      image: sysbench/sysbench:latest
      command: sysbench cpu --time=60 run
      deploy:
        replicas: 1
        placement:
          constraints: [node.role == manager]
        resources:
          limits:
            cpus: "2"
            memory: 512M

    memory-benchmark:
      image: alpine:latest
      command: sh -c "apk add stress-ng && stress-ng --vm 1 --vm-bytes 256M --timeout 60s --metrics-brief"
      deploy:
        replicas: 1
        placement:
          constraints: [node.role == manager]
        resources:
          limits:
            cpus: "1"
            memory: 512M

    io-benchmark:
      image: alpine:latest
      command: sh -c "dd if=/dev/zero of=/tmp/test bs=1M count=1024 oflag=direct && dd if=/tmp/test of=/dev/null bs=1M"
      deploy:
        replicas: 1
        placement:
          constraints: [node.role == manager]
        resources:
          limits:
            cpus: "1"
            memory: 256M

    nginx-load:
      image: nginx:alpine
      deploy:
        replicas: 3
        resources:
          limits:
            cpus: "0.5"
            memory: 128M
  ```
- **Nota:** Serviços usam imagens leves e disponíveis publicamente

### 6.4 — Documentar benchmarking

- **Arquivo:** `docs-site/docs/guides/benchmarking.md`
- **Seções:**
  1. Visão geral — o que medir e por quê
  2. Pré-requisitos — Swarm ativo, manager node
  3. Executar swarm-benchmark.sh — passo a passo
  4. Executar resma-overhead.sh — passo a passo
  5. Interpretar resultados — tabela de referência com baselines
  6. Comparar com bare metal — como rodar baseline no host
  7. Tuning — recomendações baseadas em resultados (overlay network, resource limits)
- **Baseline de referência (2026):**

| Métrica | Bare Metal | Swarm (1 node) | Swarm (3 nodes) |
|---------|-----------|----------------|-----------------|
| CPU 1-thread | 3,847 ev/s | 3,831 (0.4%) | 3,829 (0.5%) |
| CPU 64-thread | 198,412 ev/s | 196,890 (0.8%) | 196,744 (0.8%) |
| Manager memory (idle) | — | 135 MB | 135 MB |
| Per-container overhead | — | 3.2 MB | 3.2 MB |

### 6.5 — Integrar com swarm-hpa (referência)

- **Arquivo:** `docs-site/docs/guides/auto-scaling.md`
- **Conteúdo:**
  - Documentar `swarm-hpa` (github.com/Aleksey512/swarm-hpa) como ferramenta complementar
  - Como RESMA e swarm-hpa se complementam:
    - RESMA: coleta métricas, recomenda limites, detecta leaks
    - swarm-hpa: auto-scaling de réplicas baseado em métricas
  - Guia de integração: instalar swarm-hpa no cluster, usar recomendações do RESMA como base para regras de auto-scaling
  - Exemplo de labels do swarm-hpa usando dados do RESMA

## Critérios de aceite

- [ ] `benchmark/swarm-benchmark.sh` executa sem erros em cluster Swarm
- [ ] `benchmark/resma-overhead.sh` mede overhead do RESMA corretamente
- [ ] `benchmark/stack-benchmark.yml` deploya via `docker stack deploy`
- [ ] Resultados em JSON + tabela legível no stdout
- [ ] Documentação de benchmarking no site Docusaurus
- [ ] Baseline de referência documentado
- [ ] Guia de integração com swarm-hpa documentado

## Estrutura de arquivos resultante

```
benchmark/
├── README.md                  ← visão geral
├── swarm-benchmark.sh         ← overhead do Swarm
├── resma-overhead.sh          ← overhead do RESMA
├── stack-benchmark.yml        ← stack de teste
└── results/                   ← exemplos de resultados (gitignored)
```
