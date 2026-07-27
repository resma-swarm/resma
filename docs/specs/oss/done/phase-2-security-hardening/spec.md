# Fase 2 — Segurança e Hardening

> **Prioridade:** Crítica  
> **Esforço:** Médio (hardening no Go API + ML sidecar)  
> **Bloqueador:** Sim — deve ser concluída antes de tornar o repo público  
> **Dependências:** Fase 0b (API Go implementada — security hardening acontece no Go, não no Python)

## Objetivo

Remover todos os secrets hardcoded, fechar vulnerabilidades de segurança e tornar a configuração production-ready antes de expor o código publicamente. Todo o hardening acontece na **API Go** (pós-Fase 0b) e no **ML sidecar Python** — o backend Python original não recebe alterações de segurança (será removido ao final da migração).

## Estado atual (problemas identificados)

| Problema | Localização | Severidade |
|----------|-------------|------------|
| JWT secret hardcoded | `docker-compose.yml` (legacy, já substituído) | **Crítica** |
| CORS `allow_origins=["*"]` | `backend/main.py:41` (Python legacy — migrar para Go) | Alta |
| Senha padrão `admin123` hardcoded | `backend/services/auth.py` (Python legacy — migrar para Go) | Alta |
| Sem validação de JWT secret em produção | `backend/core/config.py:17` (migrar para Go `internal/config`) | Média |
| Sem headers de segurança | `backend/main.py` (migrar para Go middleware) | Média |
| Sem `.env.example` | — | Média |
| API keys sem rotação/revogação | Novo (Fase 0b introduz API key model) | Média |

## Tarefas

### 2.1 — Remover JWT secret hardcoded do docker-compose.yml

- **Arquivo:** `docker-compose.yml` ( já usa `${RESMA_JWT_SECRET:-dev-secret-change-me}` pós-0b.1)
- **Produção:** `RESMA_JWT_SECRET=${RESMA_JWT_SECRET:?RESMA_JWT_SECRET is required}` no `docker-stack.yml` (Fase 4)
- O `:?` faz o Docker falhar se a variável não estiver definida
- Default `dev-secret-change-me` permitido apenas no profile `dev`/`prod` local

### 2.2 — Criar .env.example

- **Arquivo:** `.env.example` (raiz)
- Todas as env vars com prefixo `RESMA_` e comentários explicativos
- Incluir instrução para gerar JWT secret: `openssl rand -base64 32` ou `python -c "import secrets; print(secrets.token_urlsafe(32))"`
- Adicionar `.env` ao `.gitignore`
- **Novas vars (API key):** `RESMA_API_KEY_DEFAULT_SCOPES=read` (scopes padrão ao criar key via UI)

### 2.3 — Restringir CORS

- **Arquivo:** `app/api/internal/config/config.go` — adicionar `CORSOrigins []string` (env `RESMA_CORS_ORIGINS`, CSV)
- **Arquivo:** `app/api/cmd/server/main.go` — adicionar middleware CORS que aplica apenas às rotas `/api/*` e `/api/v1/*` (não em `/health`, `/ready`)
- Default dev: `http://localhost:5173,http://localhost:8080`
- Produção: deve ser explicitamente definido via env
- **SSE + cookie HttpOnly:** CORS para `/api/sse/*` deve incluir `Access-Control-Allow-Credentials: true` + origens específicas (nunca `*` quando credentials=true). Necessário porque o frontend usa cookie `sse_session` para autenticar EventSource (ver [phase-0b — SSE Auth](../phase-0b-go-migration/spec.md#sse-auth--decisão-cookie-httponly))

### 2.4 — Validar JWT secret em produção

- **Arquivo:** `app/api/internal/config/config.go`
- Se `RESMA_JWT_SECRET` estiver vazio OU igual a `dev-secret-change-me` E ambiente for produção (detectar via `RESMA_ENV=production`), falhar startup com erro claro
- Mensagem deve incluir comando para gerar secret

### 2.5 — Usar Docker Swarm secrets

- **Arquivo:** `docker-stack.yml` (Fase 4)
- Estrutura: `secrets:` com `external: true` + `RESMA_JWT_SECRET_FILE=/run/secrets/resma_jwt_secret`
- Setup: `echo "secret" | docker secret create resma_jwt_secret -`
- **Arquivo:** `app/api/internal/config/config.go` — adicionar suporte a `_FILE` suffix: se `RESMA_JWT_SECRET_FILE` existir, ler secret do arquivo

### 2.6 — Parametrizar senha de admin

- **Arquivo:** `app/api/internal/auth/auth.go` (pós-0b.4)
- Usar `RESMA_DEFAULT_ADMIN_PASSWORD` env var em vez de hardcoded `admin123`
- Se não definida, gerar senha aleatória e logar uma vez no startup
- **Arquivo:** `app/api/internal/config/config.go` — adicionar `DefaultAdminPassword string`

### 2.7 — Headers de segurança

- **Arquivo:** `app/api/cmd/server/main.go` — adicionar middleware que seta headers em todas as respostas:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `X-XSS-Protection: 1; mode=block`
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains` (apenas em HTTPS)
  - `Referrer-Policy: strict-origin-when-cross-origin`

### 2.8 — API key security (novo)

- **Arquivo:** `app/api/internal/auth/apikey.go` (pós-0b.4)
- API keys geradas com `crypto/rand` (32 bytes, base64url, prefixo `resma_key_`)
- Hash armazenado no DuckDB (bcrypt cost 10 — menor que user password pois key é high-entropy)
- Scopes validados por endpoint: `read` (GET), `write` (POST/PUT/PATCH/DELETE em endpoints públicos `/api/v1/*`)
- Rate limit por key: configurável via `RESMA_API_KEY_RATE_LIMIT` (default 100 req/min)
- Revogação: `revoked_at` timestamp; middleware rejeita keys revogadas
- Rotação: admin cria nova key, revoga antiga (sem downtime)
- **Logs:** toda mutação via API key loga `key_id` + `action` no change-log

### 2.9 — ML sidecar security (novo)

- **Arquivo:** `app/ml/main.py` (pós-0b.8)
- ML sidecar escuta apenas em `0.0.0.0:8081` dentro da rede Docker interna — não exposto externamente
- Validação: aceita requests apenas do `resma-api` (validar via Docker network — sem auth adicional necessário em rede interna isolada)
- Health check endpoint `/health` sem auth (interno)
- Rate limit interno: rejeitar se > 50 req/s (proteção contra loop)

## Critérios de aceite

- [ ] Nenhum secret hardcoded em nenhum arquivo versionado
- [ ] `.env.example` existe com todas as env vars documentadas (incluindo API key vars)
- [ ] `.env` está no `.gitignore`
- [ ] CORS configurável via env var, aplicado apenas em rotas `/api/*` e `/api/v1/*`
- [ ] JWT secret validado em startup (produção rejeita default/dev secret)
- [ ] Senha de admin parametrizada
- [ ] Headers de segurança presentes em todas as respostas
- [ ] API keys: geração segura, hash bcrypt, scopes validados, revogação funcional
- [ ] ML sidecar: não exposto externamente, rate limit interno
- [ ] `git log --all -p | grep -i "secret\|password\|key"` não retorna secrets reais

## Riscos

- **Git history:** Se secrets já foram committed, considerar `git filter-repo` ou BFG para limpar histórico antes de tornar público. **Nota:** o usuário já removeu `.git` para recriar antes do push público — histórico limpo por design
- **Breaking change:** Usuários existentes podem ter configs com `*` no CORS — documentar migration
- **API key leakage:** Se key vazada, revogação imediata via UI. Documentar procedimento em SECURITY.md
