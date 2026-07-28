# syntax=docker/dockerfile:1.7
#
# RESMA API — Go binary + frontend estático (multi-stage, dev + prod).
#
# Targets:
#   frontend  — builda o React app (pnpm) → /out/web (dist estático)
#   builder   — compila o binário Go com CGO (go-duckdb)
#   dev       — herda builder + ferramentas de desenvolvimento (air, delve)
#   runtime   — produção: debian-slim + binário + frontend + libs C mínimas
#
# CGO é obrigatório: o driver go-duckdb linka contra a engine C do DuckDB
# (libduckdb.a prebuilt exige gcc + libstdc++ GNU ABI). O builder usa
# golang:1.26-bookworm (Debian) com gcc nativo compatível.
#
# Build context deve ser a raiz do repo (Dockerfile referencia app/api/...).
#   docker build -t resma-api --target runtime -f app/api/Dockerfile .
#   docker build -t resma-api-dev --target dev -f app/api/Dockerfile .

ARG GO_VERSION=1.26
ARG NODE_VERSION=22
ARG DEBIAN_TAG=bookworm-slim

# ---------- frontend (builda React com pnpm) ----------
FROM node:${NODE_VERSION}-slim AS frontend
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate
WORKDIR /app/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile || pnpm install
COPY frontend/ ./
RUN pnpm build
# dist/ fica em /app/frontend/dist

# ---------- builder (base compartilhada Go) ----------
FROM golang:${GO_VERSION}-bookworm AS builder

# gcc já vem na imagem golang:*-bookworm; garantimos apenas libstdc++ dev
# para o link do libduckdb.a.
RUN apt-get update && apt-get install -y --no-install-recommends \
        gcc \
        g++ \
        libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Cache de módulos: copiamos go.mod/go.sum primeiro.
COPY app/api/go.mod app/api/go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Código-fonte
COPY app/api/ ./

# Build com CGO habilitado. -trimpath remove paths absolutos do binário;
# -ldflags="-s -w" stripa símbolos e DWARF para imagem menor.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/resma-api ./cmd/server

# ---------- dev (iteração rápida com source montado) ----------
# Herda o builder com toolchain + módulos já baixados. O compose monta o
# source em /src via volume, permitindo `go build`/`go test`/`air` sem
# rebuildar imagem. Útil para contribuidores OSS sem Go/gcc locais.
FROM builder AS dev

# air: hot reload (recompila e reinicia on-save). Opcional — o compose
# pode sobrescrever o CMD com `sleep infinity` para shell iterativo.
RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/air-verse/air@latest

# .air.toml será criado em app/api/ quando hot reload for ativado (0b.6+).
# Por padrão mantemos o container ocioso para `docker compose exec`.
CMD ["sleep", "infinity"]

# ---------- runtime (produção) ----------
FROM debian:${DEBIAN_TAG} AS runtime

# DuckDB precisa de libstdc++ e libgcc em runtime (link dinâmico do CGO).
# wget para healthcheck nativo do Docker.
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        libstdc++6 \
        libgcc-s1 \
        wget \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/resma-api /usr/local/bin/resma-api

# Frontend estático buildado (React + Vite)
COPY --from=frontend /app/frontend/dist /app/web

ENV RESMA_HTTP_ADDR=:8080
ENV RESMA_WEB_DIR=/app/web
EXPOSE 8080

# Healthcheck via /health (liveness probe — não requer deps)
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/resma-api"]
