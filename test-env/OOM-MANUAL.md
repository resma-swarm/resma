# OOM — Disparo Manual

> Como disparar um OOM manualmente no `oom-prone-service` para testar a
> detecção do RESMA sem esperar o scheduler automático do `scenario-simulator`.

## Serviço

- **Container:** `oom-prone-service` (Swarm service `resma-test_oom-prone-service`)
- **Porta publicada:** `5012` → `8080` interno
- **Limite de memória:** 64M (definido no `docker-compose.yml`)
- **Endpoint:** `GET /oom?mb=<inteiro>` — aloca `<mb>` megabytes e mantém na memória

## Como disparar

### Opção 1 — curl do host (Windows)

```powershell
curl http://localhost:5012/oom?mb=200
```

Use `mb=200` ou maior para garantir OOM imediato. Valores baixos
(`mb=80`) podem não matar o container porque o accounting de memória
do cgroup inclui page cache/buffers — o limite nominal é 64M mas na
prática é preciso alocar bem mais para ultrapassar.

O `curl` retorna exit code 52 (conexão fechada) — isso confirma o OOM
(o container é morto pelo kernel antes de terminar a resposta).

### Opção 2 — curl de dentro do container

```powershell
docker exec (docker ps -q --filter "name=resma-test_oom-prone-service" | Select-Object -First 1) `
  python -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/oom?mb=200')"
```

> O container não tem `curl` instalado, por isso usamos `python urllib`.

### Opção 3 — disparar via API interna do Swarm

```powershell
docker exec (docker ps -q --filter "name=resma-test_scenario-simulator" | Select-Object -First 1) `
  python -c "import requests; requests.get('http://oom-prone-service:8080/oom?mb=200', timeout=15)"
```

## O que esperar

1. O `curl` retorna erro de conexão fechada (o container é morto pelo kernel
   antes de responder) — isso é o comportamento esperado e confirma o OOM.
2. O Swarm reinicia o container automaticamente (`restart: unless-stopped`).
3. O healthcheck leva ~30s para passar após o reinício.
4. O RESMA Agent detecta o OOM via eventos do Docker socket e faz push
   para o Go API → aparece na página `/alerts` e no banco `oom_events`.

## Parâmetros opcionais

| Parâmetro | Default | Descrição |
|-----------|---------|-----------|
| `mb` | 50 | Megabytes a alocar. Use `>= 200` para garantir OOM imediato
|   |   | (valores baixos podem não ultrapassar o limite do cgroup). |

## Resetar memória alocada (sem OOM)

Para limpar a memória acumulada sem matar o container:

```powershell
curl http://localhost:5012/reset
```

## Cadência automática (scenario-simulator)

O `scenario-simulator` dispara OOMs automaticamente com cadência
~2-3 OOMs/hora (configurável via env `OOM_INTERVAL` no `docker-compose.yml`).
Para desabilitar o disparo automático e usar só manual, pare o serviço:

```powershell
docker service scale resma-test_scenario-simulator=0
```

Para reativar:

```powershell
docker service scale resma-test_scenario-simulator=1
```
