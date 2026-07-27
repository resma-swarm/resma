# syntax=docker/dockerfile:1.7
#
# RESMA Agent — Go binary leve (multi-node coleta de stats locais).
#
# Diferenças vs app/api/Dockerfile:
#   - SEM CGO: o agent não usa DuckDB → binário puramente Go → imagem alpine
#   - SEM gcc/libstdc++ → imagem final ~15MB
#   - Build context = raiz do repo (Dockerfile referencia app/agent/...)
#
# Targets:
#   builder — compila o binário Go (CGO_ENABLED=0, alpine-friendly)
#   dev     — herda builder + ferramentas (air, delve). Source montado via volume.
#   runtime — alpine + binário estático. ~15MB final.
#
#   docker build -t resma-agent --target runtime -f app/agent/Dockerfile .
#   docker build -t resma-agent-dev --target dev -f app/agent/Dockerfile .

ARG GO_VERSION=1.26
ARG ALPINE_TAG=3.20

# ---------- builder ----------
FROM golang:${GO_VERSION}-alpine AS builder

# git necessário para go mod download com VCS stamping; ca-certificates para TLS.
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache de módulos
COPY app/agent/go.mod app/agent/go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Código-fonte
COPY app/agent/ ./

# Build estático (CGO desabilitado — sem DuckDB). -trimpath + -ldflags strip.
# Version injetada via -ldflags (sobrescrita no CI).
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /out/resma-agent ./cmd/agent

# ---------- dev (iteração rápida com source montado) ----------
FROM builder AS dev

RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/air-verse/air@latest

WORKDIR /src
CMD ["sleep", "infinity"]

# ---------- runtime (produção) ----------
FROM alpine:${ALPINE_TAG} AS runtime

# ca-certificates para TLS (HTTPS para o manager); wget para healthcheck.
RUN apk add --no-cache ca-certificates wget

COPY --from=builder /out/resma-agent /usr/local/bin/resma-agent

# /data para persistência de OOM events pendentes (events.pending.jsonl)
RUN mkdir -p /data

ENV RESMA_AGENT_HTTP_ADDR=:8082
EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD wget -qO- http://localhost:8082/health || exit 1

ENTRYPOINT ["/usr/local/bin/resma-agent"]
