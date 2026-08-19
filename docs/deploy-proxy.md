# RESMA — Deploy com Proxy (Traefik)

> Como o RESMA expõe o frontend e a API publicamente com domínio + TLS,
> sem precisar de um reverse proxy pré-existente no cluster.

## Visão geral

O RESMA oferece 4 modos de acesso público, escolhidos na instalação:

| Modo | Acesso | TLS | Como ativar |
|------|--------|-----|-------------|
| **IP:porta** | `http://<ip>:8080` | Não | Default (sem flags) |
| **Domínio + TLS** | `https://<domain>` | Let's Encrypt (ACME) | `--domain foo.com` |
| **Domínio + self-signed** | `https://<domain>.local` | Self-signed (Traefik internal) | `--domain resma.local` |
| **Headless (CLI-only)** | `http://<ip>:8080` (sem UI) | Não | `--disable-ui` |

## Arquitetura

```
                    ┌─────────────────────────────────────────┐
                    │              Docker Swarm                │
                    │                                          │
   :80/:443 ──────► │  proxy (Traefik v3.7.10)                │
                    │    │                                     │
                    │    │ Host(resma.example.com)              │
                    │    ▼                                     │
                    │  api (Go :8080)  ◄──►  ml (Python :8081) │
                    │    │                                     │
                    │    └──► /api/*  (JSON)                   │
                    │    └──► /        (SPA React)             │
                    │                                          │
                    │  agent (global, 1 por node)              │
                    └─────────────────────────────────────────┘
```

O Traefik roda na **mesma overlay network** (`resma-net`) do API — zero
configuração de rede cross-stack. Ele descobre o service `api` via Docker
Swarm labels e roteia por `Host(<domain>)`.

## Instalação

### Interativa (recomendado)

```bash
docker run -it --rm \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest
```

O installer pergunta:
1. Stack name (default: `resma`)
2. Disable UI? (default: N) — se sim, modo headless
3. Domain (default: vazio) — se preenchido, sobe Traefik + TLS
4. Port (default: 8080) — só se sem domain
5. Install CLI? (default: N)
6. Collection intervals (defaults de produção)

### Não-interativa (CI/CD)

```bash
# IP:porta com UI (default)
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest --no-input

# Domínio público com Let's Encrypt
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest --no-input --domain resma.example.com

# Domínio .local com self-signed (teste local)
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest --no-input --domain resma.local

# Headless (sem UI) + CLI
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest --no-input --disable-ui --install-cli

# Porta customizada
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest --no-input --port 8173
```

## Flags

| Flag | Default | Descrição |
|------|---------|-----------|
| `--domain <domain>` | vazio | Domínio para Traefik + TLS |
| `--tls <auto\|internal\|none>` | `auto` | Modo TLS (`auto`: ACME se público, internal se `.local`) |
| `--disable-ui` | off | Desabilita o frontend (modo headless para CLI) |
| `--install-cli` | off | Instala `resma-cli` em `/usr/local/bin` do host |
| `--port <port>` | `8080` | Porta publicada (ignorada com `--domain`) |
| `--network <name>` | — | Rede overlay externa (para proxy externo existente) |
| `--stack-name <name>` | `resma` | Nome do stack |
| `--no-input` | off | Modo não-interativo |

## TLS

### Let's Encrypt (ACME TLS-ALPN-01)

Ativado automaticamente quando `--domain` é um domínio público (não `.local`).

Requisitos:
- Porta 443 reachável de fora (firewall/port forward)
- Domínio apontando para o IP do manager node

O Traefik emite o certificado automaticamente no primeiro acesso. O email
ACME é `admin@<domain>` por default — override via `RESMA_ACME_EMAIL`.

### Self-signed (Traefik internal)

Ativado automaticamente quando `--domain` termina em `.local`. O browser
mostra warning de certificado (esperado). Útil para testes locais.

### Sem TLS (HTTP-only)

`--tls none` — Traefik roteia HTTP na porta 80, sem HTTPS.

## Proxy externo existente (CapRover, Traefik separado)

Se o cluster já tem um reverse proxy, use `--network` para anexar o API à
rede desse proxy (em vez de subir outro Traefik):

```bash
docker run -it --rm -v /var/run/docker.sock:/var/run/docker.sock \
  resmaswarm/resma-install:latest --no-input --network captain-overlay-network
```

Isso anexa **apenas o service `api`** à rede externa (ml e agent continuam
internos). Configure as labels do proxy externo manualmente no `api`.

## CLI

O `resma-cli` é um binário Go estático (~8MB) que fala com a API via HTTP.

Instalação (durante o install):
```bash
--install-cli
```

Instalação manual:
```bash
docker create --name cli-extract resmaswarm/resma-cli:latest
docker cp cli-extract:/resma /usr/local/bin/resma
docker rm cli-extract
chmod +x /usr/local/bin/resma
```

Uso:
```bash
resma auth login --server http://<ip>:8080
resma services list
resma agents list
resma recommendations show <service>
```

## Troubleshooting

### Traefik retorna 404

- Verificar se o `api` está na mesma network do Traefik: `docker network inspect resma_resma-net`
- Verificar labels do `api`: `docker service inspect resma_api --format '{{json .Spec.Labels}}'`
- Logs do Traefik: `docker service logs resma_proxy`

### Certificado Let's Encrypt não emite

- Porta 443 deve estar reachável de fora
- Domínio deve apontar para o IP do manager
- Verificar logs ACME: `docker service logs resma_proxy | grep acme`

### Agent não conecta ao API

- Verificar se o secret `resma_agent_token` existe: `docker secret inspect resma_agent_token`
- Verificar logs do agent: `docker service logs resma_agent`
- O agent lê o token via `RESMA_AGENT_TOKEN_FILE` (Docker Swarm secret)
