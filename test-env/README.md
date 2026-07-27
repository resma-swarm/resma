# RESMA Test Environment

Ambiente simulado para testar coleta de métricas e recomendações do RESMA.

## Estrutura

```
test-env/
├── dotnet-api/              # .NET 8 Minimal API (backend)
│   ├── Program.cs
│   ├── WorkloadSimulator.cs  # Simulador de workload em background
│   └── Dockerfile
├── node-api/                # Node.js Express API (backend)
│   ├── app.js
│   ├── workload_sim.js       # Simulador de workload em background
│   └── Dockerfile
├── python-api/              # Python Flask API (backend)
│   ├── app.py
│   ├── workload_sim.py       # Simulador de workload em background
│   └── Dockerfile
├── web-frontend/            # React + Vite (frontend)
├── load-generator.py        # Gerador de carga com padrões realistas
└── docker-compose.yml
```

## Apps

| App | Stack | Port | Personalidade | Endpoints |
|-----|-------|------|---------------|-----------|
| dotnet-api | .NET 8 | 5001 | business-hours | `/`, `/cpu?loops=N`, `/memory?mb=N`, `/health` |
| node-api | Node.js + Express | 5002 | spike-prone | `/`, `/cpu?loops=N`, `/memory?mb=N`, `/health` |
| python-api | Python + Flask | 5003 | batch-processor | `/`, `/cpu?loops=N`, `/memory?mb=N`, `/health` |
| web-frontend | React + Vite | 5174 | — | Dashboard com status dos serviços |

## Workload Simulator

Cada container roda um **workload simulator** em background que gera oscilações realistas de CPU e memória, mesmo sem requisições externas.

### Personalidades

- **business-hours** (dotnet-api): Ciclo de dia/noite comprimido (10 min = 24h sim). Maior uso durante "horário comercial" (8h-18h sim), baixo à "noite".
- **spike-prone** (node-api): Baseline estável com picos aleatórios (~8% de chance). Simula serviços que recebem bursts esporádicos.
- **batch-processor** (python-api): Bursts periódicos — alto por ~15s, baixo por ~45s. Simula jobs de processamento em lote.

### Técnicas

- **CPU duty-cycle**: Work por X ms, sleep pelo restante do ciclo. X varia com a intensidade calculada.
- **Memory page touching**: Memória alocada é periodicamente "tocada" para manter working set residente.
- **Gradual alloc/dealloc**: Memória cresce/diminui incrementalmente (não alloc-free instantâneo).
- **Noise ±20%**: Variação aleatória em todos os parâmetros para evitar padrões perfeitamente regulares.
- **Sinusoidal base**: Onda lenta (5 min) modula todas as personalidades.

### Magnitudes (leve para notebook)

- CPU: 20-80ms de work por ciclo de ~2.5s = ~1-4% CPU
- Memória: 3-12MB de oscilação

## Load Generator v2

Gerador de carga com padrões realistas que complementa o workload simulator:

- **Simulated business hours**: Dia comprimido em 600s (10 min = 24h)
- **Profile-aware intensity**: Cada serviço recebe carga de acordo com sua personalidade
- **Burst mode**: ~2% de chance de entrar em burst mode por 10-25s (intensidade x2.5)
- **Adaptive interval**: Intervalo entre requests varia inversamente com a intensidade
- **Probabilistic memory**: 60% das requests incluem alocação de memória

## Como usar

```bash
# Subir todos os containers
docker compose up -d --build

# Verificar status
curl http://localhost:5001/health
curl http://localhost:5002/health
curl http://localhost:5003/health

# Gerar carga para o RESMA coletar métricas
pip install requests
python load-generator.py

# Acessar dashboard
open http://localhost:5174

# Parar
docker compose down
```

## Resource Limits

Cada container já tem limits definidos no docker-compose para o RESMA monitorar e recomendar ajustes.
