# Fase 8 — Governance, RBAC, Config Two-Tier e Data Retention

> **Prioridade:** Alta
> **Esforço:** Alto (RBAC + UI Settings + Retention expansion + Two-tier config + Prune endpoints)
> **Bloqueador:** Não (Fase 7 concluída; Fase 8 é aditiva)
> **Dependências:** Fase 0b (Go API + Auth + DuckDB), Fase 2 (security hardening), Fase 7 (multi-node agent)
> **Validação:** Technical Council com 4 personas (Security Architect, Frontend Architect, Data/DB Architect, DevOps) + pesquisa web/github/context7

## Contexto do problema

A auditoria pós-Fase 7 (`docs/specs/oss/next-steps.md`) identificou 8 gaps de Prioridade Alta que não foram cobertos pelas Fases 0–7:

1. **RBAC cosmético** — `Claims.Role` existe no JWT mas nenhum middleware checa role. Usuário `user` pode aplicar recomendações, agendar schedules e restartar services.
2. **Sem gestão de usuários** — não há CRUD de usuários; onboarding cria `admin` em vez de `owner`.
3. **API Keys sem UI** — keys são geradas via API mas não há tela para criar/listar/revogar.
4. **Sidebar custom viola regra de ouro** — `Layout.tsx` é custom, não usa `Sidebar` do shadcn. Sem `NavUser`, sem área de Configurações.
5. **Config 100% env vars** — mudar intervalo de coleta ou threshold exige restart do container. Sem persistência operacional.
6. **Retention parcial** — `RunRetention` cobre apenas 3 tabelas (`metrics`, `oom_events`, `node_metrics`). 4 tabelas time-series (`task_history`, `volume_metrics`, `storage_summary`, `change_log`) acumulam indefinidamente.
7. **Sem stale-marking** — services e nodes que somem do Swarm ficam como `active` para sempre.
8. **Sem prune manual** — usuário não pode limpar dados sob demanda; depende do retention automático (job que roda a cada 24h, retendo dados por `RESMA_RETENTION_DAYS` default 30d).

A Fase 8 resolve esses 8 gaps em uma fase coesa de **Governance** — o conjunto mínimo para o RESMA ser production-ready para múltiplos usuários com diferentes níveis de acesso.

## Pesquisa de mercado (benchmark)

Validado pelo Technical Council com pesquisa em repositórios GitHub e docs oficiais:

| Recurso | RESMA (Fase 8) | Swarmpit | Portainer | Grafana | Datadog |
|---------|----------------|----------|-----------|---------|---------|
| RBAC | 3 roles (owner/admin/user) | 3 roles (admin/user/viewer) | 5+ roles granulares | RBAC + teams | RBAC + SSO |
| Onboarding | Web UI (primeiro = owner) | Web UI + YAML | Web UI obrigatório | Web UI + SSO | Web UI + SSO |
| API Keys | OTR (plaintext 1x) + scopes | N/A (JWT) | N/A (JWT) | **Deprecado** → SATs | OTR + scopes |
| Settings UI | Sidebar + nested routes | Menu único | Sidebar + accordions | Sidebar + nested routes | Sidebar + tabs |
| Data Retention | Auto (job 24h, retenção 30d) + manual prune + dry-run | N/A | N/A | Config via UI | Auto |
| Stale-marking | Soft delete (status='stale') | N/A | N/A | N/A | Age-out 24h |
| Two-tier config | Env var (infra) + DB (operacional) | Env var only | Env var only | Env var + DB | YAML + env |

**Decisões chave baseadas no benchmark:**
- **RBAC modelo Swarmpit** (3 roles simples) — Casbin é overkill para Fase 8; middleware custom é suficiente
- **API Keys modelo Datadog** (OTR — One-Time Read) — Grafana deprecou API keys em favor de SATs, mas migração forçada é out-of-scope
- **Settings UI modelo Supabase** (nested routes, não tabs horizontais) — URLs compartilháveis, browser back/forward
- **Stale-marking modelo Netdata** (soft delete, preserva histórico) — Swarm faz scale up/down; service pode voltar
- **Two-tier modelo Grafana** (env var = default/infra, DB = operacional) — padrão consolidado

## Decisão arquitetural

### Modelo: RBAC custom + Two-tier config + Soft-delete stale + Prune auditado

**Justificativa:**
1. **Sem Casbin** — 3 roles com hierarquia simples (owner > admin > user) é resolvido com middleware `RequireRole` de 20 linhas. Casbin adiciona dependência e complexidade desnecessária para o MVP.
2. **Two-tier sem flag de feature** — env var vira default na primeira inicialização; DB prevalece após primeiro boot. Sem `RESMA_USE_DB_CONFIG` flag — migração é unidirecional e transparente.
3. **Soft-delete para stale** — Swarm faz scale up/down constantemente; deletar services/nodes perde histórico. Marcar como `stale` permite recovery.
4. **Prune com dry-run + audit** — toda operação de prune registra em `change_log` (quem, o quê, quantas rows, quando). Dry-run é default para evitar acidentes.

## Arquitetura proposta

### Diagrama de componentes

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Frontend (React 19 + shadcn/ui)                    │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  AppSidebar (shadcn oficial)                                  │   │
│  │  ├── SidebarHeader (logo RESMA)                              │   │
│  │  ├── SidebarContent → NavMain (8 itens operacionais)         │   │
│  │  ├── SidebarFooter → NavSettings (engrenagem) + NavUser      │   │
│  │  └── SidebarRail (collapse)                                  │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  /settings (nested routes)                                    │   │
│  │  ├── /settings/users      → UsersPage    (owner only)        │   │
│  │  ├── /settings/api-keys   → ApiKeysPage  (owner/admin)       │   │
│  │  ├── /settings/parameters → ParametersPage (owner/admin)     │   │
│  │  └── /settings/data       → DataPage     (owner/admin)       │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  /profile → ProfilePage (qualquer role, próprio usuário)            │
│                                                                      │
│  usePermissions() → canEdit(), canDelete(), canManageUsers()        │
│  <RequireRole roles={["owner"]}>...</RequireRole>                   │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                │ HTTP (JWT)
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Go API (app/api/)                                 │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Middleware chain                                             │   │
│  │  JWTMiddleware → RequireRole(owner|admin) → Handler           │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌────────────┐  │
│  │  Auth handlers      │  │  User CRUD handlers │  │  Prune     │  │
│  │  /api/auth/*        │  │  /api/auth/users/*  │  │  handlers  │  │
│  │  + onboarding       │  │  + API keys CRUD    │  │  /api/prune│  │
│  └─────────────────────┘  └─────────────────────┘  └────────────┘  │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Config (two-tier)                                            │   │
│  │  config.Load() → env vars (infra) + defaults (operacional)   │   │
│  │  config.LoadFromDB() → sobrescreve operacional com DB        │   │
│  │  config.SaveToDB() → persiste mudanças via UI                │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Collector loops                                              │   │
│  │  retentionLoop (24h) → RunRetention + CleanupExpiredTokens   │   │
│  │  staleLoop (1h) → MarkStaleServices + MarkStaleNodes         │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                │ DuckDB (exclusive)
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    DuckDB (resma.duckdb)                             │
│                                                                      │
│  users (role: owner|admin|user)  │  api_keys (hash + scopes)        │
│  app_settings (key-value)        │  change_log (audit)              │
│  service_registry (status:active│stale) │  nodes (status:active│stale)│
│  refresh_tokens (expires_at)     │  metrics/oom_events/node_metrics │
│  task_history (ts)               │  volume_metrics/storage_summary  │
└─────────────────────────────────────────────────────────────────────┘
```

## Tarefas

### 8.1 — RBAC: Roles, Middleware e Onboarding

> **Arquivos:** `app/api/internal/auth/roles.go` (novo), `app/api/internal/auth/middleware.go`, `app/api/internal/auth/auth.go`, `app/api/internal/db/schema.go`, `app/api/internal/db/queries.go`, `app/api/internal/server/auth_handlers.go`, `app/api/internal/server/server.go`

#### 8.1.1 — Enum de Roles

Criar `app/api/internal/auth/roles.go`:

```go
package auth

type Role string

const (
    RoleOwner Role = "owner" // Único, primeiro usuário via onboarding
    RoleAdmin Role = "admin" // Mesmo acesso do owner (exceto deletar users)
    RoleUser  Role = "user"  // Somente leitura
)

func IsValidRole(role string) bool {
    switch Role(role) {
    case RoleOwner, RoleAdmin, RoleUser:
        return true
    default:
        return false
    }
}
```

#### 8.1.2 — Middleware `RequireRole` (novo — não existe ainda)

Adicionar em `app/api/internal/auth/middleware.go` (novo middleware, baseado no padrão de `RequireScope` existente):

```go
// RequireRole verifica se o usuário no contexto tem um dos roles permitidos.
// Deve ser usado após JWTMiddleware. Retorna 403 se role insuficiente.
func RequireRole(allowedRoles ...Role) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := UserFromContext(r.Context())
            if user == nil {
                writeAuthError(w, "user not in context", http.StatusUnauthorized)
                return
            }
            userRole := Role(user.Role)
            for _, allowed := range allowedRoles {
                if userRole == allowed {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            writeAuthError(w, "insufficient role", http.StatusForbidden)
        })
    }
}
```

**Semântica de status code:**
- 401 — não autenticado (sem token, token inválido/expirado, user fora do contexto)
- 403 — autenticado mas não autorizado (role insuficiente)

#### 8.1.3 — Onboarding cria `owner` (não `admin`)

Modificar `DoOnboarding` em `app/api/internal/auth/auth.go`:

```go
func (s *Service) DoOnboarding(ctx context.Context, username, password string) (*AuthResult, error) {
    tx, err := s.db.DB().BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()

    var count int32
    if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil {
        return nil, fmt.Errorf("get user count: %w", err)
    }
    if count > 0 {
        return nil, ErrOnboardingCompleted
    }

    hash, err := s.HashPassword(password)
    if err != nil {
        return nil, err
    }

    var userID int32
    err = tx.QueryRowContext(ctx,
        `INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'owner') RETURNING id`,
        username, hash).Scan(&userID)
    if err != nil {
        if strings.Contains(err.Error(), "constraint") {
            return nil, ErrOnboardingCompleted
        }
        return nil, fmt.Errorf("create user: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit: %w", err)
    }

    access, err := s.CreateAccessToken(ctx, userID, username, "owner")
    if err != nil {
        return nil, err
    }
    refresh, err := s.CreateRefreshToken(ctx, userID)
    if err != nil {
        return nil, err
    }
    return &AuthResult{AccessToken: access, RefreshToken: refresh, TokenType: "bearer"}, nil
}
```

**Proteção contra race condition:** transação explícita + unique constraint em `users.username` + catch de constraint violation.

#### 8.1.4 — Migração de schema

Em `app/api/internal/db/schema.go`, adicionar à lista de migrations:

```go
// Fase 8: onboarding cria 'owner' em vez de 'admin'
// Se existe exatamente 1 usuário com role='admin', promover a 'owner'
`UPDATE users SET role = 'owner' WHERE role = 'admin' AND (SELECT count(*) FROM users) = 1`,
// Remover default 'admin' para novos usuários criados via DB.
// Após DROP DEFAULT, inserts devem especificar role explicitamente (CreateUserWithRole já faz isso).
`ALTER TABLE users ALTER COLUMN role DROP DEFAULT`,
```

**Nota sobre audit logging:** O spec usa o método `AddChangeLog` existente em `queries.go` (não criar `LogChange` novo). O campo `DockerResponse` (TEXT, já existente na struct `ChangeLogEntry` e na tabela `change_log`) é reusado para armazenar JSON/string com details da operação (user CRUD, prune). **Não criar coluna `details` nova** — `docker_response` atende ao caso de uso.

#### 8.1.5 — Matriz de permissões

| Endpoint | Método | Owner | Admin | User | Observação |
|----------|--------|:-----:|:-----:|:----:|------------|
| `/api/auth/status` | GET | ✅ | ✅ | ✅ | Público |
| `/api/auth/onboarding` | POST | ✅* | ❌ | ❌ | Apenas se 0 usuários |
| `/api/auth/login` | POST | ✅ | ✅ | ✅ | Público |
| `/api/auth/logout` | POST | ✅ | ✅ | ✅ | Autenticado |
| `/api/auth/me` | GET | ✅ | ✅ | ✅ | Autenticado |
| `/api/auth/change-password` | POST | ✅ | ✅ | ✅ | Próprio usuário |
| `/api/auth/users` | GET | ✅ | ✅ | ❌ | Listar |
| `/api/auth/users` | POST | ✅ | ✅ | ❌ | Criar |
| `/api/auth/users/{id}` | PATCH | ✅ | ✅ | ❌ | Editar role |
| `/api/auth/users/{id}` | DELETE | ✅ | ❌ | ❌ | Apenas owner |
| `/api/auth/api-keys` | GET/POST/PATCH/DELETE | ✅ | ✅ | ❌ | Gestão de keys |
| `/api/schedules` | GET | ✅ | ✅ | ✅ | Listar |
| `/api/schedules` | POST/DELETE | ✅ | ✅ | ❌ | Escrever |
| `/api/templates` | GET | ✅ | ✅ | ✅ | Listar |
| `/api/templates` | POST/PUT/DELETE | ✅ | ✅ | ❌ | Escrever |
| `/api/templates/{name}/apply/{service}` | POST | ✅ | ✅ | ❌ | Aplicar |
| `/api/services/*` | GET | ✅ | ✅ | ✅ | Read-only |
| `/api/services/{name}/archive\|restore` | PATCH | ✅ | ✅ | ❌ | Escrever |
| `/api/prune/*` | POST | ✅ | ✅ | ❌ | Prune manual |
| `/api/settings/*` | GET | ✅ | ✅ | ❌ | Ler config operacional |
| `/api/settings/*` | PATCH | ✅ | ✅ | ❌ | Alterar config |

\* Onboarding apenas se `GET /api/auth/status` retornar `initialized: false`

#### 8.1.6 — Registro de rotas com RequireRole

Em `app/api/internal/server/server.go`, aplicar `RequireRole` nas rotas de escrita:

```go
// Rotas owner/admin apenas (escrita)
ownerAdmin := auth.RequireRole(auth.RoleOwner, auth.RoleAdmin)
mux.Handle("POST /api/auth/users", s.auth.JWTMiddleware(ownerAdmin(s.handleCreateUser)))
mux.Handle("GET /api/auth/users", s.auth.JWTMiddleware(ownerAdmin(s.handleListUsers)))
mux.Handle("PATCH /api/auth/users/{id}", s.auth.JWTMiddleware(ownerAdmin(s.handleUpdateUser)))
mux.Handle("POST /api/schedules", s.auth.JWTMiddleware(ownerAdmin(s.handleCreateSchedule)))
// ... (todas as rotas de escrita)

// Rotas owner apenas (delete users)
ownerOnly := auth.RequireRole(auth.RoleOwner)
mux.Handle("DELETE /api/auth/users/{id}", s.auth.JWTMiddleware(ownerOnly(s.handleDeleteUser)))
```

### 8.2 — User CRUD: Endpoints e DB methods

> **Arquivos:** `app/api/internal/db/queries.go`, `app/api/internal/server/user_handlers.go` (novo)

#### 8.2.1 — DB methods

Adicionar em `app/api/internal/db/queries.go` (métodos novos — `GetUserByID` já existe, não recriar):

```go
type User struct {
    ID        int32     `json:"id"`
    Username  string    `json:"username"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// ListUsers — NOVO
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT id, username, role, created_at, updated_at FROM users ORDER BY created_at`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []User
    for rows.Next() {
        var u User
        if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
            return nil, err
        }
        out = append(out, u)
    }
    return out, rows.Err()
}

// GetOwner — NOVO (obrigatório para validação "apenas 1 owner")
func (s *Store) GetOwner(ctx context.Context) (*User, error) {
    var u User
    err := s.db.QueryRowContext(ctx,
        `SELECT id, username, role, created_at, updated_at FROM users WHERE role = 'owner' LIMIT 1`).
        Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &u.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &u, err
}

// CreateUserWithRole — NOVO
func (s *Store) CreateUserWithRole(ctx context.Context, username, passwordHash, role string) (int32, error) {
    var id int32
    err := s.db.QueryRowContext(ctx,
        `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?) RETURNING id`,
        username, passwordHash, role).Scan(&id)
    return id, err
}

// UpdateUserRole — NOVO
func (s *Store) UpdateUserRole(ctx context.Context, userID int32, role string) error {
    _, err := s.db.ExecContext(ctx,
        `UPDATE users SET role = ?, updated_at = now() WHERE id = ?`, role, userID)
    return err
}

// DeleteUser — NOVO
func (s *Store) DeleteUser(ctx context.Context, userID int32) error {
    // Revogar todos os refresh tokens primeiro
    _, _ = s.db.ExecContext(ctx,
        `UPDATE refresh_tokens SET revoked = true WHERE user_id = ?`, userID)
    _, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
    return err
}
```

**Nota:** `GetUserByID` já existe em `queries.go` (linhas 86-96) — não recriar. Reusar nos handlers.

#### 8.2.2 — Handlers HTTP

Criar `app/api/internal/server/user_handlers.go`:

**Nota sobre helpers JSON do server (já existem em `json.go`):**
- `writeJSON(w, status, v)` — escreve JSON com status code
- `writeOK(w, v)` — atalho para `writeJSON(w, 200, v)`
- `writeError(w, status, msg)` — escreve `{"error": msg}`
- `decodeJSON(w, r, v) bool` — retorna `true` se OK, `false` se erro (já escreve resposta). Padrão: `if !decodeJSON(w, r, &req) { return }`
- `pathValue(r, key) string` — extrai path param (Go 1.22+)

```go
// GET /api/auth/users — lista usuários (sem password_hash)
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
    users, err := s.db.ListUsers(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeOK(w, map[string]any{"users": users})
}

// POST /api/auth/users — cria usuário (owner/admin)
// Body: {"username": "...", "password": "...", "role": "user|admin"}
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Username string `json:"username"`
        Password string `json:"password"`
        Role     string `json:"role"`
    }
    if !decodeJSON(w, r, &req) {
        return
    }
    // Validações
    if len(req.Username) < 3 || len(req.Username) > 50 {
        writeError(w, http.StatusBadRequest, "username must be 3-50 chars")
        return
    }
    if len(req.Password) < 12 {
        writeError(w, http.StatusBadRequest, "password must be at least 12 chars")
        return
    }
    if !auth.IsValidRole(req.Role) || req.Role == "owner" {
        writeError(w, http.StatusBadRequest, "role must be 'admin' or 'user'")
        return
    }
    hash, err := s.auth.HashPassword(req.Password)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    id, err := s.db.CreateUserWithRole(r.Context(), req.Username, hash, req.Role)
    if err != nil {
        writeError(w, http.StatusConflict, "username already exists")
        return
    }
    // Audit log (usa AddChangeLog existente; DockerResponse armazena details)
    user := auth.UserFromContext(r.Context())
    s.db.AddChangeLog(r.Context(), db.ChangeLogEntry{
        Service: "system", Action: "user_created", Source: "manual",
        User: user.Username, Status: "completed",
        DockerResponse: fmt.Sprintf("id=%d, username=%s, role=%s", id, req.Username, req.Role),
    })
    writeJSON(w, http.StatusCreated, map[string]any{"id": id, "username": req.Username, "role": req.Role})
}

// PATCH /api/auth/users/{id} — altera role (owner/admin)
// Body: {"role": "user|admin"}
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid id")
        return
    }
    var req struct{ Role string `json:"role"` }
    if !decodeJSON(w, r, &req) {
        return
    }
    if !auth.IsValidRole(req.Role) || req.Role == "owner" {
        writeError(w, http.StatusBadRequest, "role must be 'admin' or 'user'")
        return
    }
    // Owner não pode ter role alterado
    target, err := s.db.GetUserByID(r.Context(), int32(id))
    if err != nil || target == nil {
        writeError(w, http.StatusNotFound, "user not found")
        return
    }
    if target.Role == "owner" {
        writeError(w, http.StatusBadRequest, "cannot change owner role")
        return
    }
    if err := s.db.UpdateUserRole(r.Context(), int32(id), req.Role); err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    user := auth.UserFromContext(r.Context())
    s.db.AddChangeLog(r.Context(), db.ChangeLogEntry{
        Service: "system", Action: "user_role_changed", Source: "manual",
        User: user.Username, Status: "completed",
        DockerResponse: fmt.Sprintf("id=%d, role=%s→%s", id, target.Role, req.Role),
    })
    writeOK(w, map[string]any{"id": id, "role": req.Role})
}

// DELETE /api/auth/users/{id} — deleta usuário (owner apenas)
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(r.PathValue("id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid id")
        return
    }
    // Owner não pode deletar a si mesmo
    currentUser := auth.UserFromContext(r.Context())
    currentID, _ := strconv.Atoi(currentUser.Sub)
    if currentID == id {
        writeError(w, http.StatusBadRequest, "cannot delete yourself")
        return
    }
    target, err := s.db.GetUserByID(r.Context(), int32(id))
    if err != nil || target == nil {
        writeError(w, http.StatusNotFound, "user not found")
        return
    }
    if target.Role == "owner" {
        writeError(w, http.StatusBadRequest, "cannot delete owner")
        return
    }
    if err := s.db.DeleteUser(r.Context(), int32(id)); err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    s.db.AddChangeLog(r.Context(), db.ChangeLogEntry{
        Service: "system", Action: "user_deleted", Source: "manual",
        User: currentUser.Username, Status: "completed",
        DockerResponse: fmt.Sprintf("id=%d, username=%s", id, target.Username),
    })
    writeOK(w, map[string]string{"message": "user deleted"})
}
```

### 8.3 — API Keys UI (foco 100% frontend — backend já existe)

> **Backend:** já implementado na Fase 0b.4 — `internal/auth/apikey.go` (GenerateAPIKey, ValidateAPIKey, RevokeAPIKey) + `internal/server/apikey_handlers.go` (handlers CRUD: GET/POST/PATCH/DELETE `/api/auth/api-keys`). **Nenhum código backend novo nesta tarefa.**
> **Frontend:** novo `frontend/src/pages/Settings.tsx` com sub-rota `ApiKeysPage` + `ApiKeyCreateDialog`

#### 8.3.1 — Endpoints de API Keys (já existem — documentação para o frontend)

| Endpoint | Método | Role | Descrição |
|----------|--------|------|-----------|
| `/api/auth/api-keys` | GET | owner/admin | Lista keys (sem hash) |
| `/api/auth/api-keys` | POST | owner/admin | Cria key — retorna plaintext **uma vez** |
| `/api/auth/api-keys/{id}` | PATCH | owner/admin | Edita name/scopes |
| `/api/auth/api-keys/{id}` | DELETE | owner/admin | Revoga key (soft delete) |

**Response do POST (201):**
```json
{
  "id": 5,
  "plaintext": "resma_key_abc123...XYZ",
  "prefix": "resma_key_abc...",
  "name": "production-monitoring",
  "scopes": "read",
  "created_at": "2026-07-25T16:00:00Z"
}
```

**Aviso:** `plaintext` é retornado **apenas nesta response**. Listagem retorna apenas `prefix`.

#### 8.3.2 — Frontend: ApiKeysPage

Ver tarefa 8.5 para estrutura completa. Componente `ApiKeyCreateDialog` com fluxo:
1. Form (name + scopes) → POST
2. Dialog de revelação com Alert warning + code block + botão Copy
3. Botão "Concluir" fecha e recarrega listagem

### 8.4 — Two-Tier Config (env var + DB)

> **Arquivos:** `app/api/internal/config/config.go`, `app/api/cmd/server/main.go`, `app/api/internal/server/settings_handlers.go` (novo)

#### 8.4.1 — Parâmetros que migram para `app_settings`

**Operacionais (migram para DB):**

| Env Var | Key em app_settings | Tipo | Default |
|---------|---------------------|------|---------|
| `RESMA_COLLECT_INTERVAL` | `collect_interval` | int (segs) | 1 |
| `RESMA_RETENTION_DAYS` | `retention_days` | int | 30 |
| `RESMA_OUTLIER_THRESHOLD` | `outlier_threshold` | float | 3.0 |
| `RESMA_LEAK_R2_THRESHOLD` | `leak_r2_threshold` | float | 0.7 |
| `RESMA_LEAK_DAILY_MB_THRESHOLD` | `leak_daily_mb_threshold` | float | 10.0 |
| `RESMA_ANALYSIS_WINDOW_DAYS` | `analysis_window_days` | int | 7 |
| `RESMA_CLUSTER_INTERVAL` | `cluster_interval` | int (segs) | 60 |
| `RESMA_STORAGE_INTERVAL` | `storage_interval` | int (segs) | 300 |
| `RESMA_STALE_SERVICE_DAYS` | `stale_service_days` | int | 7 |

**Infra (permanecem em env vars — não mudam sem restart):**
- `RESMA_DB_PATH`, `RESMA_JWT_SECRET`/`_FILE`, `RESMA_HTTP_ADDR`, `RESMA_CORS_ORIGINS`
- `RESMA_AGENT_TOKEN`/`_FILE`, `RESMA_ENV`, `RESMA_DEFAULT_ADMIN_PASSWORD`
- `RESMA_ML_URL`, `RESMA_ML_ENABLED`, `RESMA_EXCLUDED_IMAGES`

#### 8.4.2 — Formato de storage

Manter schema atual `app_settings (key VARCHAR PK, value VARCHAR)`. Sem coluna `value_type` — parsing no Go via `strconv.Atoi`/`ParseFloat`. Simples e compatível com `GetSetting`/`SetSetting` existentes.

#### 8.4.3 — Algoritmo de precedência

```
1. config.Load() lê env vars → Config com defaults de env
2. db.New() inicializa schema
3. initializeDefaultSettings() — se app_settings vazio, persiste defaults de env
4. config.LoadFromDB() — sobrescreve campos operacionais com valores do DB
5. Config final é usada pelo server/collector
```

#### 8.4.4 — Métodos em `config.go`

```go
// LoadFromDB sobrescreve valores operacionais com valores do DB.
// Chamado após initSchema. Se key não existe no DB, mantém valor de env.
func (cfg *Config) LoadFromDB(ctx context.Context, store *db.Store) error {
    cfg.CollectInterval = getDurationFromDB(ctx, store, "collect_interval", cfg.CollectInterval)
    cfg.RetentionDays = getIntFromDB(ctx, store, "retention_days", cfg.RetentionDays)
    cfg.OutlierThreshold = getFloatFromDB(ctx, store, "outlier_threshold", cfg.OutlierThreshold)
    cfg.LeakR2Threshold = getFloatFromDB(ctx, store, "leak_r2_threshold", cfg.LeakR2Threshold)
    cfg.LeakDailyMBThreshold = getFloatFromDB(ctx, store, "leak_daily_mb_threshold", cfg.LeakDailyMBThreshold)
    cfg.AnalysisWindowDays = getIntFromDB(ctx, store, "analysis_window_days", cfg.AnalysisWindowDays)
    cfg.ClusterInterval = getDurationFromDB(ctx, store, "cluster_interval", cfg.ClusterInterval)
    cfg.StorageInterval = getDurationFromDB(ctx, store, "storage_interval", cfg.StorageInterval)
    cfg.StaleServiceDays = getIntFromDB(ctx, store, "stale_service_days", cfg.StaleServiceDays)
    return nil
}

// SaveSettingToDB persiste uma key em app_settings.
func (cfg *Config) SaveSettingToDB(ctx context.Context, store *db.Store, key, value string) error {
    return store.SetSetting(ctx, key, value)
}

// CountSettings retorna o número de keys em app_settings (novo método em queries.go).
func (s *Store) CountSettings(ctx context.Context) (int32, error) {
    var n int32
    err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM app_settings").Scan(&n)
    return n, err
}

// initializeDefaultSettings persiste defaults de env no DB se app_settings estiver vazio.
func initializeDefaultSettings(ctx context.Context, store *db.Store, cfg *Config) error {
    count, err := store.CountSettings(ctx)
    if err != nil { return err }
    if count > 0 { return nil }
    defaults := map[string]string{
        "collect_interval":         fmt.Sprintf("%d", int(cfg.CollectInterval/time.Second)),
        "retention_days":           fmt.Sprintf("%d", cfg.RetentionDays),
        "outlier_threshold":        fmt.Sprintf("%g", cfg.OutlierThreshold),
        "leak_r2_threshold":        fmt.Sprintf("%g", cfg.LeakR2Threshold),
        "leak_daily_mb_threshold":  fmt.Sprintf("%g", cfg.LeakDailyMBThreshold),
        "analysis_window_days":     fmt.Sprintf("%d", cfg.AnalysisWindowDays),
        "cluster_interval":         fmt.Sprintf("%d", int(cfg.ClusterInterval/time.Second)),
        "storage_interval":         fmt.Sprintf("%d", int(cfg.StorageInterval/time.Second)),
        "stale_service_days":       fmt.Sprintf("%d", cfg.StaleServiceDays),
    }
    for k, v := range defaults {
        if err := store.SetSetting(ctx, k, v); err != nil { return err }
    }
    return nil
}
```

#### 8.4.5 — Endpoints de Settings

Criar `app/api/internal/server/settings_handlers.go`:

| Endpoint | Método | Role | Descrição |
|----------|--------|------|-----------|
| `/api/settings` | GET | owner/admin | Lista todas as settings operacionais |
| `/api/settings/{key}` | PATCH | owner/admin | Atualiza uma setting (valida tipo) |

**Decisão de design (conflito com `/api/config` existente):** Já existe `GET /api/config` em `misc_handlers.go` (linha 15) que retorna apenas 3 campos (`collect_interval`, `retention_days`, `analysis_window_days`). **Consolidar tudo em `/api/settings`** (9 campos operacionais) e deprecar `/api/config`:
- `GET /api/settings` substitui `GET /api/config` (mesma response shape + valores do DB)
- `PATCH /api/settings/{key}` é novo (two-tier config)
- `GET /api/config` mantém como alias deprecated por 1 release, depois remove
- Frontend migra de `/api/config` para `/api/settings` na mesma tarefa

**GET response:**
```json
{
  "collect_interval": 1,
  "retention_days": 30,
  "outlier_threshold": 3.0,
  "leak_r2_threshold": 0.7,
  "leak_daily_mb_threshold": 10.0,
  "analysis_window_days": 7,
  "cluster_interval": 60,
  "storage_interval": 300,
  "stale_service_days": 7
}
```

**PATCH request:**
```json
{ "value": "60" }
```

**PATCH response (200):**
```json
{ "key": "cluster_interval", "value": "60" }
```

**Validação de tipo (obrigatória antes de persistir):** cada key tem um tipo esperado. Handler valida antes de salvar:

```go
var settingTypes = map[string]string{
    "collect_interval":        "int",
    "retention_days":          "int",
    "outlier_threshold":       "float",
    "leak_r2_threshold":       "float",
    "leak_daily_mb_threshold": "float",
    "analysis_window_days":    "int",
    "cluster_interval":        "int",
    "storage_interval":        "int",
    "stale_service_days":      "int",
}

func validateSetting(key, value string) error {
    typ, ok := settingTypes[key]
    if !ok {
        return fmt.Errorf("unknown setting key: %s", key)
    }
    switch typ {
    case "int":
        if _, err := strconv.Atoi(value); err != nil {
            return fmt.Errorf("%s must be int", key)
        }
    case "float":
        if _, err := strconv.ParseFloat(value, 64); err != nil {
            return fmt.Errorf("%s must be float", key)
        }
    }
    return nil
}
```

Após persistir, **não aplica em runtime** — mudanças em `collect_interval`/`retention_days` exigem restart do collector (documentar na UI). Settings que podem ser aplicadas em runtime (ex: `outlier_threshold` usado apenas pelo ML sidecar) são lidas sob demanda.

### 8.5 — Frontend: Sidebar shadcn + Settings Area

> **Arquivos:** `frontend/src/components/ui/sidebar.tsx` (novo via shadcn), `frontend/src/components/app-sidebar.tsx` (novo), `frontend/src/components/nav-user.tsx` (novo), `frontend/src/components/nav-main.tsx` (novo), `frontend/src/components/nav-settings.tsx` (novo), `frontend/src/components/Layout.tsx` (refatorar), `frontend/src/pages/Settings.tsx` (novo), `frontend/src/pages/Profile.tsx` (novo), `frontend/src/components/settings/*.tsx` (novos), `frontend/src/hooks/use-permissions.ts` (novo), `frontend/src/hooks/use-mobile.ts` (novo)

#### ⚠️ REGRA DE OURO SHADCN/UI (não negociável)

> **Todo componente de UI DEVE vir do shadcn/ui.** NUNCA criar componentes UI custom quando existe equivalente no shadcn. NUNCA alterar arquivos em `frontend/src/components/ui/` — usar como estão e compor em volta.
>
> Se o shadcn não tem um componente necessário, compor com componentes shadcn existentes (ex: `Button` + `DropdownMenu` em vez de criar `SplitButton`).
>
> **Componentes custom atuais que DEVEM ser substituídos pelos oficiais shadcn nesta fase:**
> - `dropdown-menu.tsx` — implementação custom (React Context puro, sem `DropdownMenuGroup`) → substituir via `npx shadcn@latest add dropdown-menu`
> - `tabs.tsx` — implementação custom (React Context puro, não radix) → substituir via `npx shadcn@latest add tabs`
> - `Layout.tsx` — sidebar custom com `<aside>` → migrar para `Sidebar`/`SidebarProvider`/`SidebarInset` do shadcn
>
> **Componentes novos a adicionar via shadcn (não criar manualmente):**
> - `sidebar`, `form`, `switch`, `alert-dialog`, `checkbox`
>
> **Componentes de composição (não-ui) que SÃO permitidos criar:**
> - `app-sidebar.tsx`, `nav-user.tsx`, `nav-main.tsx`, `nav-settings.tsx` — compõem componentes shadcn (`Sidebar`, `SidebarMenu`, `DropdownMenu`, `Avatar`)
> - `RequireRole.tsx` — wrapper de proteção (lógica, não UI)
> - `settings/*.tsx` — páginas que compõem componentes shadcn (`Table`, `Dialog`, `AlertDialog`, `Form`, `Switch`, `Checkbox`)

#### 8.5.1 — Componentes shadcn a adicionar

```bash
cd frontend
npx shadcn@latest add sidebar
npx shadcn@latest add form
npx shadcn@latest add switch
npx shadcn@latest add alert-dialog
npx shadcn@latest add checkbox
# dropdown-menu: substituir implementação custom pela oficial (para suportar DropdownMenuGroup)
npx shadcn@latest add dropdown-menu
# tabs: substituir implementação custom pela oficial (radix-based) para compatibilidade com shadcn sidebar
npx shadcn@latest add tabs
```

**Regra de ouro:** NUNCA alterar componentes em `frontend/src/components/ui/` — usar como estão. O `dropdown-menu.tsx` e `tabs.tsx` atuais são implementações custom (não radix-based) e serão **substituídos** pelas versões oficiais shadcn (não editados manualmente).

#### 8.5.2 — Hook `use-mobile.ts`

```typescript
import { useState, useEffect } from "react"

export function useIsMobile() {
  const [isMobile, setIsMobile] = useState(false)
  useEffect(() => {
    const check = () => setIsMobile(window.innerWidth < 768)
    check()
    window.addEventListener("resize", check)
    return () => window.removeEventListener("resize", check)
  }, [])
  return isMobile
}
```

#### 8.5.3 — Componente `NavUser` (baseado no shadcn sidebar-07)

```typescript
// frontend/src/components/nav-user.tsx
import { User, Settings, LogOut, Key } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/contexts/AuthContext"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuGroup,
  DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar"
import { ChevronsUpDown } from "lucide-react"

export function NavUser() {
  const { user, logout } = useAuth()
  const { isMobile } = useSidebar()
  const navigate = useNavigate()

  const initials = (user?.username || "U").substring(0, 2).toUpperCase()

  const handleLogout = async () => {
    await logout()
    navigate("/login")
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{user?.username}</span>
                <span className="truncate text-xs text-muted-foreground capitalize">{user?.role}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            {/* DropdownMenuLabel DEVE ser envolvido em DropdownMenuGroup (PR #9321) */}
            <DropdownMenuGroup>
              <DropdownMenuLabel className="p-0 font-normal">
                <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                  <Avatar className="h-8 w-8 rounded-lg">
                    <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
                  </Avatar>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">{user?.username}</span>
                    <span className="truncate text-xs text-muted-foreground capitalize">{user?.role}</span>
                  </div>
                </div>
              </DropdownMenuLabel>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem onClick={() => navigate("/profile")}>
                <User className="mr-2 h-4 w-4" /> Perfil
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => navigate("/profile#change-password")}>
                <Key className="mr-2 h-4 w-4" /> Mudar senha
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => navigate("/settings")}>
                <Settings className="mr-2 h-4 w-4" /> Configurações
              </DropdownMenuItem>
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleLogout} className="text-destructive">
              <LogOut className="mr-2 h-4 w-4" /> Sair
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
```

**Nota crítica (PR #9321):** `DropdownMenuLabel` deve ser envolvido em `DropdownMenuGroup` para evitar erro runtime "MenuGroupRootContext is missing" do Base UI.

#### 8.5.4 — Componente `AppSidebar`

```typescript
// frontend/src/components/app-sidebar.tsx
import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarRail } from "@/components/ui/sidebar"
import { NavMain } from "@/components/nav-main"
import { NavSettings } from "@/components/nav-settings"
import { NavUser } from "@/components/nav-user"
import { Activity } from "lucide-react"

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <div className="flex h-14 items-center gap-2.5 border-b border-sidebar-border px-4">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/15">
            <Activity className="h-4.5 w-4.5 text-primary" />
          </div>
          <div className="flex flex-col">
            <span className="text-sm font-bold tracking-tight">RESMA</span>
            <span className="text-[10px] text-muted-foreground">Otus7 Infrastructure</span>
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <NavMain />
      </SidebarContent>
      <SidebarFooter>
        <NavSettings />
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
```

#### 8.5.5 — `Layout.tsx` migrado

```typescript
// frontend/src/components/Layout.tsx
import { Outlet, useLocation } from "react-router-dom"
import { SidebarProvider, SidebarInset, SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { AppSidebar } from "@/components/app-sidebar"

export function Layout() {
  const location = useLocation()
  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 h-4" />
          {/* Breadcrumbs + refresh mode toggle preservados */}
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}
```

**Preservar:** `useRefreshStore`, breadcrumbs, mobile sidebar (shadcn SidebarProvider já cuida do mobile via offcanvas).

#### 8.5.6 — Routing da área de Settings (nested routes)

```typescript
// frontend/src/App.tsx (rotas novas)
<Route path="/settings" element={<Settings />}>
  <Route index element={<Navigate to="/settings/users" replace />} />
  <Route path="users" element={<RequireRole roles={["owner"]}><UsersPage /></RequireRole>} />
  <Route path="api-keys" element={<RequireRole roles={["owner","admin"]}><ApiKeysPage /></RequireRole>} />
  <Route path="parameters" element={<RequireRole roles={["owner","admin"]}><ParametersPage /></RequireRole>} />
  <Route path="data" element={<RequireRole roles={["owner","admin"]}><DataPage /></RequireRole>} />
</Route>
<Route path="/profile" element={<Profile />} />
```

#### 8.5.7 — Hook `use-permissions.ts`

```typescript
import { useAuth } from "@/contexts/AuthContext"

export type Role = "owner" | "admin" | "user"

export function usePermissions() {
  const { user } = useAuth()
  const role = (user?.role || "user") as Role
  return {
    role,
    canEdit: () => role === "owner" || role === "admin",
    canDelete: () => role === "owner",
    canManageUsers: () => role === "owner",
    canManageApiKeys: () => role === "owner" || role === "admin",
    canSchedule: () => role === "owner" || role === "admin",
    canApplyRecommendations: () => role === "owner" || role === "admin",
    canPrune: () => role === "owner" || role === "admin",
  }
}
```

#### 8.5.8 — Componente `RequireRole`

```typescript
// frontend/src/components/RequireRole.tsx
import { ReactNode } from "react"
import { Navigate } from "react-router-dom"
import { useAuth } from "@/contexts/AuthContext"
import type { Role } from "@/hooks/use-permissions"

export function RequireRole({ children, roles, fallback }: {
  children: ReactNode
  roles: Role[]
  fallback?: ReactNode
}) {
  const { user } = useAuth()
  const role = (user?.role || "user") as Role
  if (!roles.includes(role)) return <>{fallback || <Navigate to="/" replace />}</>
  return <>{children}</>
}
```

#### 8.5.9 — Páginas de Settings

**`Settings.tsx`** (wrapper com sub-navegação via Tabs + NavLink):

```typescript
const settingsTabs = [
  { to: "/settings/users", label: "Usuários", icon: Users, roles: ["owner"] },
  { to: "/settings/api-keys", label: "API Keys", icon: Key, roles: ["owner","admin"] },
  { to: "/settings/parameters", label: "Parâmetros", icon: Settings, roles: ["owner","admin"] },
  { to: "/settings/data", label: "Dados", icon: Database, roles: ["owner","admin"] },
]
// Filtrar tabs por role; renderizar TabsList com NavLink + TabsTrigger; <Outlet /> abaixo
```

**`UsersPage.tsx`** — tabela com colunas (username, role, created_at, ações), dialog de criação, dialog de edição de role, confirm dialog de delete.

**`ApiKeysPage.tsx`** — tabela (prefix, name, scopes, last_used, created_at, status), `ApiKeyCreateDialog` com fluxo OTR (form → reveal → copy → close), botão revoke com confirm.

**`ParametersPage.tsx`** — formulário com 9 campos (collect_interval, retention_days, outlier_threshold, leak_r2_threshold, leak_daily_mb_threshold, analysis_window_days, cluster_interval, storage_interval, stale_service_days). Botão "Salvar" faz PATCH por campo. Aviso: "Mudanças em intervalos exigem restart do collector".

**`DataPage.tsx`** — 6 cards (Services stale, Nodes stale, Tasks órfãs, Métricas antigas, Change log, Volume metrics). Cada card tem botão "Dry-run" (mostra contagem) e botão "Prune" (abre `AlertDialog` com checkbox "Entendo que é irreversível").

**`Profile.tsx`** — mostra username, role, created_at. Form de alterar senha (senha atual + nova + confirmação). POST `/api/auth/change-password`.

### 8.6 — Data Retention Expansion

> **Arquivos:** `app/api/internal/db/queries.go`, `app/api/internal/collector/collector.go`

#### 8.6.1 — Expandir `RunRetention`

```go
// RunRetention deleta dados antigos conforme retentionDays.
// Tabelas time-series: metrics, oom_events, node_metrics, task_history,
// volume_metrics, storage_summary (coluna ts) + change_log (coluna created_at).
func (s *Store) RunRetention(ctx context.Context, retentionDays int) error {
    tsTables := []string{"metrics", "oom_events", "node_metrics", "task_history", "volume_metrics", "storage_summary"}
    for _, table := range tsTables {
        stmt := fmt.Sprintf("DELETE FROM %s WHERE ts < now()::TIMESTAMP - INTERVAL %d DAYS", table, retentionDays)
        if _, err := s.db.ExecContext(ctx, stmt); err != nil {
            return fmt.Errorf("retention %s: %w", table, err)
        }
    }
    // change_log usa created_at
    stmt := fmt.Sprintf("DELETE FROM change_log WHERE created_at < now()::TIMESTAMP - INTERVAL %d DAYS", retentionDays)
    if _, err := s.db.ExecContext(ctx, stmt); err != nil {
        return fmt.Errorf("retention change_log: %w", err)
    }
    return nil
}
```

#### 8.6.2 — Cleanup de refresh tokens expirados

```go
// CleanupExpiredTokens remove tokens expirados ou revogados.
func (s *Store) CleanupExpiredTokens(ctx context.Context) error {
    _, err := s.db.ExecContext(ctx,
        `DELETE FROM refresh_tokens WHERE expires_at < now() OR revoked = true`)
    return err
}
```

#### 8.6.3 — Integração no collector

Modificar `retentionLoop` em `app/api/internal/collector/collector.go`:

```go
func (c *Collector) retentionLoop() {
    defer c.wg.Done()
    c.runRetentionOnce()
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-c.ctx.Done():
            return
        case <-ticker.C:
            c.runRetentionOnce()
        }
    }
}

func (c *Collector) runRetentionOnce() {
    if err := c.db.RunRetention(c.ctx, c.cfg.RetentionDays); err != nil {
        c.log.Error("retention erro", "err", err)
    }
    if err := c.db.CleanupExpiredTokens(c.ctx); err != nil {
        c.log.Error("cleanup tokens erro", "err", err)
    }
    c.log.Info("retention job completed")
}
```

### 8.7 — Stale-Marking

> **Arquivos:** `app/api/internal/db/queries.go`, `app/api/internal/collector/collector.go`, `app/api/internal/config/config.go`

#### 8.7.1 — Nova config

Adicionar em `app/api/internal/config/config.go`:

```go
StaleServiceDays int // RESMA_STALE_SERVICE_DAYS (default 7)
```

```go
StaleServiceDays: getInt("RESMA_STALE_SERVICE_DAYS", 7),
```

#### 8.7.2 — Funções de stale-marking

```go
// MarkStaleServices marca services sem métricas recentes como 'stale'.
// Usa last_seen em service_registry. Retorna count de rows afetadas.
func (s *Store) MarkStaleServices(ctx context.Context, thresholdDays int) (int64, error) {
    stmt := fmt.Sprintf(`
        UPDATE service_registry SET status = 'stale', updated_at = now()
        WHERE status = 'active' AND last_seen < now()::TIMESTAMP - INTERVAL %d DAYS`, thresholdDays)
    res, err := s.db.ExecContext(ctx, stmt)
    if err != nil { return 0, err }
    n, _ := res.RowsAffected()
    return n, nil
}

// MarkStaleNodes marca nodes sem atualização recente como 'stale'.
// Usa updated_at em nodes. Retorna count.
func (s *Store) MarkStaleNodes(ctx context.Context, thresholdDays int) (int64, error) {
    stmt := fmt.Sprintf(`
        UPDATE nodes SET status = 'stale', updated_at = now()
        WHERE status = 'active' AND updated_at < now()::TIMESTAMP - INTERVAL %d DAYS`, thresholdDays)
    res, err := s.db.ExecContext(ctx, stmt)
    if err != nil { return 0, err }
    n, _ := res.RowsAffected()
    return n, nil
}

// PruneContainerMap remove mapeamentos de containers que não existem mais.
// Recebe lista de container_ids ativos do Docker API.
func (s *Store) PruneContainerMap(ctx context.Context, activeContainerIDs []string) (int64, error) {
    if len(activeContainerIDs) == 0 {
        res, err := s.db.ExecContext(ctx, "DELETE FROM container_node_map")
        if err != nil { return 0, err }
        n, _ := res.RowsAffected()
        return n, nil
    }
    placeholders := make([]string, len(activeContainerIDs))
    args := make([]any, len(activeContainerIDs))
    for i, id := range activeContainerIDs {
        placeholders[i] = "?"
        args[i] = id
    }
    stmt := fmt.Sprintf("DELETE FROM container_node_map WHERE container_id NOT IN (%s)",
        strings.Join(placeholders, ","))
    res, err := s.db.ExecContext(ctx, stmt, args...)
    if err != nil { return 0, err }
    n, _ := res.RowsAffected()
    return n, nil
}
```

#### 8.7.3 — Loop de stale-marking no collector

```go
func (c *Collector) staleLoop() {
    defer c.wg.Done()
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-c.ctx.Done():
            return
        case <-ticker.C:
            if n, err := c.db.MarkStaleServices(c.ctx, c.cfg.StaleServiceDays); err != nil {
                c.log.Error("mark stale services", "err", err)
            } else if n > 0 {
                c.log.Info("services marcados stale", "count", n)
            }
            if n, err := c.db.MarkStaleNodes(c.ctx, c.cfg.StaleServiceDays); err != nil {
                c.log.Error("mark stale nodes", "err", err)
            } else if n > 0 {
                c.log.Info("nodes marcados stale", "count", n)
            }
        }
    }
}
```

Iniciar em `collector.Start()`: `c.wg.Add(1); go c.staleLoop()`.

### 8.8 — Data Prune Endpoints

> **Arquivos:** `app/api/internal/server/prune_handlers.go` (novo), `app/api/internal/db/queries.go`

#### 8.8.1 — Endpoints

| Endpoint | Método | Role | Descrição |
|----------|--------|------|-----------|
| `/api/prune/services-stale` | POST | owner/admin | Remove services stale + métricas órfãs |
| `/api/prune/nodes-stale` | POST | owner/admin | Remove nodes stale + node_metrics órfãs + container_node_map |
| `/api/prune/tasks-orphan` | POST | owner/admin | Remove tasks cujo service não existe mais |
| `/api/prune/metrics?days=N` | POST | owner/admin | Remove métricas antigas |
| `/api/prune/change-log?days=N` | POST | owner/admin | Remove change_log antigo |
| `/api/prune/volumes?days=N` | POST | owner/admin | Remove volume_metrics + storage_summary antigos |

Todos suportam `?dry-run=true` (retorna contagem sem deletar). Default `days` = `RESMA_RETENTION_DAYS`.

#### 8.8.2 — Response shape

```json
{
  "dry_run": false,
  "deleted": {
    "metrics": 12450,
    "oom_events": 8,
    "service_registry": 3
  },
  "message": "Prune concluído",
  "timestamp": "2026-07-25T16:00:00Z"
}
```

#### 8.8.3 — Audit logging

Toda operação de prune registra em `change_log` usando `AddChangeLog` existente (campo `DockerResponse` armazena JSON com contagem):

```go
// LogPruneOperation registra uma operação de prune em change_log.
// Usa AddChangeLog existente; DockerResponse armazena JSON com contagem de rows.
func (s *Store) LogPruneOperation(ctx context.Context, operation, user string, deleted map[string]int64) error {
    deletedJSON, _ := json.Marshal(deleted)
    return s.AddChangeLog(ctx, ChangeLogEntry{
        Service:        "system",
        Action:         operation,
        Source:         "manual",
        User:           user,
        Status:         "completed",
        DockerResponse: string(deletedJSON),
    })
}
```

#### 8.8.4 — DB helpers necessários

Adicionar em `queries.go`: `GetStaleServices`, `GetStaleNodes`, `CountMetricsByService`, `DeleteMetricsByService`, `DeleteOOMEventsByService`, `DeleteServiceConfig`, `DeleteStaleServices`, `CountNodeMetricsByNode`, `DeleteNodeMetricsByNode`, `DeleteStaleNodes`, `CountOrphanTasks`, `DeleteOrphanTasks`, `CountMetricsOlderThan`, `DeleteMetricsOlderThan`, `CountChangeLogOlderThan`, `DeleteChangeLogOlderThan`, `CountVolumeMetricsOlderThan`, `DeleteVolumeMetricsOlderThan`, `CountStorageSummaryOlderThan`, `DeleteStorageSummaryOlderThan`.

### 8.9 — Documentação e arquivos legais (subset da Fase 1)

> **Arquivos:** `CHANGELOG.md` (novo), `.github/CODEOWNERS` (novo), `.github/ISSUE_TEMPLATE/*.yml` (converter de .md)

#### 8.9.1 — CHANGELOG.md (formato Keep a Changelog)

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- RBAC with 3 roles (owner/admin/user) and RequireRole middleware
- User management UI (CRUD) with onboarding flow
- API Keys management UI with One-Time Read (OTR)
- Settings area with nested routes (users, api-keys, parameters, data)
- Two-tier config (env var infra + DB operacional)
- Data retention expansion (task_history, volume_metrics, storage_summary, change_log)
- Stale-marking for services and nodes (soft delete)
- Data prune endpoints with dry-run and audit logging
- shadcn official Sidebar migration (NavUser, NavSettings, NavMain)
- Profile page with change password
### Changed
- Onboarding now creates 'owner' instead of 'admin'
- Layout migrated from custom to shadcn Sidebar
### Security
- RequireRole middleware protects all write endpoints
- API keys plaintext shown only once at creation
```

#### 8.9.2 — CODEOWNERS

```github
* @resma/tech-leads
/frontend/ @resma/frontend-team
/app/api/ @resma/backend-team
/app/ml/ @resma/ml-team
/app/agent/ @resma/backend-team
/.github/workflows/ @resma/devops-team
/docker-compose*.yml @resma/devops-team
/docker-stack.yml @resma/devops-team
/docs/ @resma/docs-team
/app/api/internal/auth/ @resma/security-lead @resma/backend-team
```

#### 8.9.3 — Issue Forms (.yml)

Converter `ISSUE_TEMPLATE/bug_report.md` e `feature_request.md` para formato GitHub Issue Forms (`.github/ISSUE_TEMPLATE/bug_report.yml`, `feature_request.yml`). Estrutura: markdown header + input/textarea/dropdown/checkboxes fields.

## Critérios de aceite

### RBAC
- [ ] `RequireRole` middleware implementado e aplicado em todas as rotas de escrita
- [ ] Onboarding cria usuário com role `owner` (não `admin`)
- [ ] Usuário existente único com role `admin` é promovido a `owner` na migração
- [ ] Usuário `user` recebe 403 ao tentar POST/PATCH/DELETE em rotas protegidas
- [ ] Owner não pode deletar a si mesmo
- [ ] Apenas 1 owner permitido (validação no POST/PATCH de users)
- [ ] Role no JWT tem TTL curto (15min access, 7d refresh) — window de stale role máximo 15min

### User CRUD
- [ ] `GET /api/auth/users` retorna lista sem `password_hash`
- [ ] `POST /api/auth/users` valida username (3-50), password (min 12), role (admin|user)
- [ ] `PATCH /api/auth/users/{id}` não permite alterar owner
- [ ] `DELETE /api/auth/users/{id}` revoga refresh tokens antes de deletar
- [ ] Toda mutação registra em `change_log`

### API Keys UI
- [ ] Página cria key com dialog OTR (plaintext uma vez)
- [ ] Listagem mostra prefix, name, scopes, last_used, created_at, status
- [ ] Revogação com confirm dialog
- [ ] Edição de name/scopes

### Two-Tier Config
- [ ] `config.LoadFromDB()` sobrescreve 9 parâmetros operacionais
- [ ] `initializeDefaultSettings()` persiste defaults no primeiro boot
- [ ] `PATCH /api/settings/{key}` valida tipo antes de persistir
- [ ] Env vars permanecem para infra (JWT secret, DB path, porta, CORS, agent token)

### Data Retention
- [ ] `RunRetention` cobre 7 tabelas (3 originais + task_history + volume_metrics + storage_summary + change_log)
- [ ] **Nota:** `task_errors` (mencionada no `next-steps.md`) não existe no schema atual — usar `task_history` com filtro `status IN ('failed','rejected')` como proxy. Criar `task_errors` é out-of-scope (fase futura se necessário)
- [ ] `CleanupExpiredTokens` roda junto com retention (24h)
- [ ] Stale-marking roda a cada 1h
- [ ] `RESMA_STALE_SERVICE_DAYS` default 7

### Data Prune
- [ ] 6 endpoints implementados com `?dry-run=true`
- [ ] Toda operação registra em `change_log`
- [ ] Confirm dialog duplo no frontend para prune de services/nodes
- [ ] Toast com resumo pós-prune

### Frontend
- [ ] `sidebar.tsx` do shadcn adicionado (via `npx shadcn@latest add`)
- [ ] `Layout.tsx` migrado para `SidebarProvider` + `AppSidebar`
- [ ] `NavUser` segue padrão sidebar-07 com `DropdownMenuGroup` (PR #9321)
- [ ] `NavSettings` (engrenagem) no `SidebarFooter` acima do `NavUser`
- [ ] `/settings` com 4 sub-rotas (users, api-keys, parameters, data)
- [ ] `/profile` com form de alterar senha
- [ ] `usePermissions()` e `<RequireRole>` implementados
- [ ] Ações de escrita escondidas para role `user`
- [ ] Refresh mode toggle preservado na migração

### Documentação
- [ ] `CHANGELOG.md` na raiz (Keep a Changelog)
- [ ] `.github/CODEOWNERS` configurado
- [ ] Issue Templates convertidos para `.yml`

### Testes automatizados
- [ ] Unit test para `RequireRole` middleware (cobre 401, 403, role permitida)
- [ ] Unit test para `IsValidRole` (cobre owner/admin/user/inválido)
- [ ] Integration test para User CRUD (create/list/update/delete + audit em change_log)
- [ ] Integration test para onboarding (race condition: 2 requests simultâneos → 1 owner)
- [ ] Integration test para prune endpoints (dry-run não deleta, real deleta + audita)
- [ ] Unit test para `MarkStaleServices`/`MarkStaleNodes` (threshold boundary)
- [ ] Unit test para `validateSetting` (int/float/unknown key)

## Validação obrigatória após cada tarefa

1. `docker compose exec go-dev go build ./...` — compila sem erros
2. `docker compose exec go-dev go vet ./...` — sem warnings
3. `docker compose exec go-dev gofmt -l .` — sem arquivos não formatados
4. `docker compose exec go-dev go test ./...` — testes automatizados passam
5. Smoke test dos endpoints (curl dentro do container)
6. `pnpm build` (em `frontend/`) — build sem erros
7. **Testes de UI com Playwright** — navegar nas telas e confirmar via HTML retornado (ver seção abaixo)
8. Atualizar checkbox da tarefa neste spec

## Testes (automatizados + UI com Playwright)

### Testes automatizados (Go)

Criar arquivos de teste ao longo da implementação (não como fase separada):

- `app/api/internal/auth/middleware_test.go` — `RequireRole` (401 sem user, 403 role insuficiente, 200 role permitida)
- `app/api/internal/auth/roles_test.go` — `IsValidRole` (owner/admin/user/inválido)
- `app/api/internal/db/queries_test.go` — `ListUsers`, `CreateUserWithRole`, `UpdateUserRole`, `DeleteUser`, `GetOwner`, `CountSettings`
- `app/api/internal/server/user_handlers_test.go` — integration test (create/list/update/delete + audit em change_log)
- `app/api/internal/server/settings_handlers_test.go` — `validateSetting` (int/float/unknown key)
- `app/api/internal/server/prune_handlers_test.go` — dry-run não deleta, real deleta + audita
- `app/api/internal/db/retention_test.go` — `MarkStaleServices`/`MarkStaleNodes` (threshold boundary)

**Comando:** `docker compose exec go-dev go test -race -cover -timeout 120s ./...`

### Testes de UI com Playwright

Usar o MCP server `playwright` (já disponível) para navegar nas telas e confirmar via HTML retornado.

**Setup:**
1. Subir ambiente dev: `docker compose up -d` (API + frontend)
2. Fazer onboarding inicial (se DB vazio): POST `/api/auth/onboarding` com owner
3. Login via UI ou API para obter JWT

**Fluxos de UI a validar (após tarefa 8.5):**

1. **Login + Onboarding**
   - Navegar para `http://localhost:5173`
   - Se não inicializado: confirmar tela de onboarding com label "Create Owner Account"
   - Após onboarding: confirmar redirect para Dashboard
   - Verificar HTML contém `data-testid="dashboard"` ou título "Dashboard"

2. **Sidebar shadcn**
   - Confirmar `<aside>` com classes `sidebar` (não custom)
   - Confirmar `SidebarHeader` com logo RESMA
   - Confirmar `SidebarFooter` com `NavUser` (avatar + username + role)
   - Confirmar `SidebarRail` para collapse
   - Clicar em collapse, confirmar sidebar recolhe

3. **NavUser dropdown**
   - Clicar no avatar no footer da sidebar
   - Confirmar dropdown abre com itens: Perfil, Mudar senha, Configurações, Sair
   - Confirmar `DropdownMenuGroup` presente no HTML (não erro MenuGroupRootContext)

4. **Área de Settings**
   - Navegar para `/settings`
   - Confirmar 4 tabs/sub-rotas: Usuários, API Keys, Parâmetros, Dados
   - Como owner: confirmar todas as 4 tabs visíveis
   - Como user: confirmar redirect para `/` (sem permissão)

5. **UsersPage (owner)**
   - Confirmar tabela com colunas: username, role, created_at, ações
   - Clicar "Criar Usuário" → confirmar dialog abre
   - Criar user com role "user" → confirmar aparece na tabela
   - Tentar criar owner → confirmar erro "role must be 'admin' or 'user'"

6. **ApiKeysPage (owner/admin)**
   - Clicar "Criar API Key" → confirmar dialog com form (name + scopes)
   - Criar key → confirmar dialog de revelação com alert warning
   - Confirmar plaintext visível uma vez
   - Clicar "Concluir" → confirmar key na tabela com prefix (não plaintext)

7. **ParametersPage (owner/admin)**
   - Confirmar 9 campos com valores atuais do DB
   - Alterar `retention_days` para 60 → clicar Salvar
   - Confirmar toast de sucesso
   - Recarregar página → confirmar valor persiste (60)

8. **DataPage (owner/admin)**
   - Confirmar 6 cards (Services stale, Nodes stale, Tasks órfãs, Métricas, Change log, Volumes)
   - Clicar "Dry-run" em Services stale → confirmar contagem exibida
   - Clicar "Prune" → confirmar AlertDialog com checkbox "irreversível"
   - Confirmar checkbox obrigatório antes de habilitar botão

9. **ProfilePage**
   - Navegar para `/profile`
   - Confirmar username, role, created_at exibidos
   - Form de alterar senha: senha atual + nova + confirmação
   - Submeter → confirmar toast de sucesso

10. **RBAC enforcement na UI**
    - Login como user (role=user)
    - Confirmar botões de escrita escondidos (Criar, Editar, Deletar)
    - Navegar para `/settings/users` → confirmar redirect para `/`
    - Navegar para `/settings/api-keys` → confirmar redirect para `/`

**Comando Playwright (via MCP):** usar `mcp_call_tool` com server `playwright` para `browser_navigate`, `browser_click`, `browser_snapshot` (retorna HTML/acessibilidade tree para confirmação).

## Riscos e mitigações

| Risco | Prob | Impacto | Mitigação |
|-------|------|---------|-----------|
| Race condition no onboarding (múltiplos owners) | Baixa | Alto | Transação + unique constraint + catch de constraint violation |
| Escalada de privilégio (admin cria owner) | Média | Alto | Validação no POST/PATCH: role `owner` rejeitado; apenas 1 owner |
| JWT stale com role antiga após role change | Média | Médio | TTL curto (15min access); documentar token version counter para Fase 9 |
| Breaking change na Layout (sidebar custom → shadcn) | Alta | Médio | Branch `feature/sidebar-migration`; preservar `Layout.legacy.tsx` como backup; testar todas as páginas |
| Perda do refresh mode toggle na migração | Média | Baixo | Mover para header após `SidebarTrigger`; preservar `useRefreshStore` |
| DropdownMenu custom vs oficial (sem DropdownMenuGroup) | Alta | Médio | Substituir pela oficial via `npx shadcn@latest add dropdown-menu` |
| Prune acidental sem dry-run | Média | Alto | Dry-run default true; confirm dialog duplo; audit em change_log |
| Lock contention DuckDB em DELETE massivo | Baixa | Médio | DELETE por idade usa zonemaps (skip row groups); não fazer batch pequeno |
| Two-tier config inconsistência após upgrade | Baixa | Médio | Env var = default no primeiro boot; DB prevalece depois; endpoint reset to defaults |

## Ordem de implementação (dependências)

```
8.1 (RBAC: roles + middleware + onboarding) ← fundação
  ↓
8.2 (User CRUD endpoints) ← depende de 8.1
  ↓
8.3 (API Keys UI — backend já existe, foco frontend) ← independente
  ↓
8.4 (Two-tier config) ← independente
  ↓
8.6 (Retention expansion) ← independente
  ↓
8.7 (Stale-marking) ← depende de 8.4 (StaleServiceDays)
  ↓
8.8 (Prune endpoints) ← depende de 8.7 (GetStaleServices/Nodes)
  ↓
8.5 (Frontend: sidebar + settings) ← depende de 8.1, 8.2, 8.3, 8.4, 8.8
  ↓
8.9 (Docs legais) ← independente, pode ser paralelo
```

**Paralelizável:** 8.3, 8.4, 8.6, 8.9 podem rodar em paralelo após 8.1. 8.5 (frontend) pode começar em paralelo com 8.2-8.8 usando mocks.

## Gaps não cobertos nesta fase (out-of-scope)

A Fase 8 cobre os 8 gaps de **Governance** identificados no `next-steps.md`. Os seguintes itens de Prioridade Alta/Média **não são cobertos** e serão tratados em fases separadas:

| Gap | Prioridade | Fase futura | Justificativa |
|-----|------------|-------------|---------------|
| **SSE bug crítico** (real-time quebrado — nenhum tópico tem publisher E consumer) | Alta | Fase 9 (SSE Fix) | Bug de SSE é independente de Governance; correção exige injetar SSE handler no Collector + fazer frontend assinar tópicos. Escopo diferente, não misturar com RBAC. |
| Fase 7 — Resiliência do Agent (gzip, rate limiting, replay protection) | Alta | Fase 9 ou 10 | Reforço do agent binary, não relacionado a Governance |
| Fase 7 — Dashboard (card Service Health) | Alta | Fase 9 | Frontend aditivo, pode ser feito após sidebar migration |
| Fase 7 — Frontend (pie chart, restart history, config agent) | Média | Fase 9 | Frontend aditivo |
| Fase 3 — Docusaurus content (guides, contributing) | Média | Fase 3 (continuação) | Documentação, independente |
| Fase 5 — Testes (Go suite, ML suite, swag-check) | Média | Fase 5 (continuação) | QA, independente |
| Fase 6 — Benchmarking | Baixa | Fase 6 | Performance, independente |

**Recomendação:** Após Fase 8, priorizar **Fase 9 (SSE Fix)** por ser bug crítico de Prioridade Alta que afeta experiência do usuário em todas as telas.

## Esforço estimado

| Tarefa | Esforço |
|--------|---------|
| 8.1 RBAC (roles + middleware + onboarding + migrations) | 4h |
| 8.2 User CRUD (DB methods + handlers + audit) | 4h |
| 8.3 API Keys UI (backend já existe — foco 100% frontend: ApiKeysPage + OTR dialog) | 1h |
| 8.4 Two-tier config (LoadFromDB + SaveToDB + endpoints + validação de tipo) | 5h |
| 8.5 Frontend (sidebar + nav-user + settings area + 4 sub-páginas + profile) | 18h |
| 8.6 Retention expansion (RunRetention + CleanupExpiredTokens) | 2h |
| 8.7 Stale-marking (3 funções + staleLoop) | 3h |
| 8.8 Prune endpoints (6 handlers + DB helpers + audit) | 6h |
| 8.9 Docs legais (CHANGELOG + CODEOWNERS + Issue Forms) | 2h |
| **Total** | **~45h** |

## Referências

### Pesquisa Technical Council
- **Casbin RBAC:** https://github.com/casbin/casbin-website-v3/blob/master/content/docs/model/rbac/rbac.mdx
- **Go RBAC patterns:** https://reliasoftware.com/blog/golang-rbac
- **shadcn PR #9321 (DropdownMenuGroup):** https://github.com/shadcn-ui/ui/pull/9321
- **shadcn Issue #9117 (MenuGroupRootContext):** https://github.com/shadcn-ui/ui/issues/9117
- **shadcn nav-user.tsx (sidebar-07):** https://github.com/shadcn-ui/ui/blob/main/apps/v4/registry/new-york-v4/blocks/sidebar-07/components/nav-user.tsx
- **shadcn sidebar.tsx source:** https://github.com/shadcn-ui/ui/blob/main/apps/v4/registry/new-york-v4/ui/sidebar.tsx
- **Swarmpit RBAC:** https://github.com/swarmpit/swarmpit/blob/master/doc/USER_CONFIG.md
- **Portainer RBAC:** https://docs.portainer.io/admin/user/roles
- **Datadog API Keys (OTR):** https://docs.datadoghq.com/account_management/api-app-keys.md
- **Grafana API Keys deprecation:** https://grafana.com/docs/grafana/latest/administration/service-accounts/migrate-api-keys/
- **Supabase SettingsLayout:** https://github.com/supabase/supabase/blob/main/apps/studio/components/layouts/ProjectSettingsLayout/SettingsLayout.tsx
- **Prometheus retention:** https://prometheus.io/docs/prometheus/latest/storage/
- **TimescaleDB retention policy:** https://github.com/timescale/docs/blob/latest/api/data-retention/add_retention_policy.md
- **InfluxDB retention:** https://docs.influxdata.com/influxdb/v2/reference/internals/data-retention/
- **Netdata node lifecycle:** https://learn.netdata.cloud/docs/netdata-parents/node-types-and-lifecycle-reference
- **Prometheus staleness:** https://promcon.io/2017-munich/slides/staleness-in-prometheus-2-0.pdf
- **Grafana runtime settings:** https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/settings-updates-at-runtime/
- **DuckDB reclaiming space:** https://duckdb.org/docs/current/operations_manual/footprint_of_duckdb/reclaiming_space
- **DuckDB DELETE partial vacuum:** https://github.com/duckdb/duckdb/pull/9931
- **DuckDB indexing:** https://www.duckdb.org/docs/current/guides/performance/indexing
- **JWT is not a session:** https://umurinan.com/pages/posts/your-jwt-is-not-a-session.html
- **DB-backed authorization:** https://antonraphaelcables.medium.com/fixing-stale-roles-in-jwts-db-backed-authorization-for-multi-tenant-apis-e5dc6a04aa23
- **Zuplo API key best practices:** https://zuplo.com/docs/articles/api-key-best-practices
- **Keep a Changelog:** https://keepachangelog.com/en/1.1.0/
- **GitHub CODEOWNERS:** https://docs.github.com/en/repositories/managing-your-repositorys-settings/about-code-owners
- **GitHub Issue Forms:** https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms
- **Docker multi-platform:** https://docs.docker.com/build/building/multi-platform

### Specs relacionadas
- Fase 0b (Go API + Auth): `done/phase-0b-go-migration/spec.md`
- Fase 2 (Security hardening): `done/phase-2-security-hardening/spec.md`
- Fase 7 (Multi-node agent): `done/phase-7-multi-node-agent/spec.md`
- Próximos passos: `../next-steps.md`

### Arquivos do projeto relevantes
- `app/api/internal/auth/auth.go` — Claims, UserContext, DoOnboarding, API key generation
- `app/api/internal/auth/middleware.go` — JWTMiddleware, APIKeyMiddleware, RequireScope (base para RequireRole)
- `app/api/internal/db/schema.go` — users, api_keys, app_settings, change_log, service_registry, nodes
- `app/api/internal/db/queries.go` — GetSetting/SetSetting, RunRetention (atual — 3 tabelas)
- `app/api/internal/config/config.go` — Config struct com todas as env vars
- `app/api/internal/collector/collector.go` — retentionLoop (24h)
- `app/api/internal/server/server.go` — registro de rotas com middlewares
- `frontend/src/components/Layout.tsx` — Layout custom (a ser migrado)
- `frontend/src/contexts/AuthContext.tsx` — user.role já exposto
- `app/api/internal/server/json.go` — helpers writeJSON/writeOK/writeError/decodeJSON/pathValue

## Validação contra código atual (pré-implementação)

Validação executada com 3 subagents em paralelo contra o código real do repo. Resultado consolidado:

### Backend Auth + DB (25 itens verificados)
- ✅ `Claims.Role` existe (auth.go:54)
- ✅ `UserContext.Role` existe (auth.go:71)
- ✅ `UserFromContext` existe (middleware.go:29)
- ✅ `JWTMiddleware`, `APIKeyMiddleware`, `RequireScope`, `writeAuthError` existem
- ✅ `GenerateAPIKey` retorna (plaintext, hash, prefix, err) (auth.go:369)
- ✅ `users` table: role DEFAULT 'admin', username UNIQUE (schema.go:58-65)
- ✅ `api_keys` table: key_hash, key_prefix, scopes, revoked_at (schema.go:210-219)
- ✅ `app_settings` table: key, value (schema.go:75-78)
- ✅ `change_log` table: docker_response TEXT, sem coluna details (schema.go:166-185)
- ✅ `service_registry`: status, last_seen (schema.go:90-95)
- ✅ `nodes`: status, updated_at (schema.go:97-114)
- ✅ `refresh_tokens`: expires_at, revoked (schema.go:67-73)
- ✅ `GetSetting`/`SetSetting` existem (queries.go:22-36)
- ✅ `AddChangeLog` existe, recebe `ChangeLogEntry` (queries.go:845)
- ✅ `ChangeLogEntry.DockerResponse` existe (sql.NullString), sem campo Details (queries.go:840)
- ✅ `GetUserByID` existe (queries.go:87)
- ✅ `RunRetention` cobra 3 tabelas: metrics, oom_events, node_metrics (queries.go:43-53)
- ✅ `task_history`, `volume_metrics`, `storage_summary` existem com coluna `ts`
- ✅ `container_node_map` existe com container_id PK (schema.go:141-146)
- ✅ `ListUsers`, `GetOwner`, `CreateUserWithRole`, `UpdateUserRole`, `DeleteUser`, `CountSettings` — NÃO existem (spec propõe criar)
- ⚠️ `DoOnboarding` usa role `'admin'` hardcoded (auth.go:220) — spec propõe mudar para `'owner'`

### Server + Config + Collector (15 itens verificados)
- ✅ Go 1.22+ pattern routing: `mux.HandleFunc("METHOD /path", handler)` (server.go:72)
- ✅ `s.auth.JWTMiddleware` existe e envolve handlers (server.go:111)
- ✅ Todas as rotas `/api/*` usam apenas JWTMiddleware (sem RequireRole) — spec propõe adicionar
- ✅ `handleConfig` existe, retorna 3 campos (misc_handlers.go:15)
- ✅ `handleOnboarding`/`handleAuthStatus` existem (auth_handlers.go)
- ✅ Handlers CRUD de API keys existem: GET/POST/PATCH/DELETE (apikey_handlers.go:17-20)
- ✅ Config struct tem todos os campos operacionais (config.go:17-62)
- ✅ Helpers `getInt`, `getFloat`, `getDurationSecs`, `getenv` existem (config.go)
- ✅ `retentionLoop` existe, roda a cada 24h, chama `RunRetention` (collector.go:333-353)
- ✅ `staleLoop` NÃO existe (spec propõe criar)
- ✅ `s.db` é `*db.Store`, `s.auth` é `*auth.Service`, `s.cfg` é `*config.Config` (server.go:34-43)
- ✅ Helpers JSON: `writeJSON(w, status, v)`, `writeOK(w, v)`, `writeError(w, status, msg)`, `decodeJSON(w, r, v) bool`, `pathValue(r, key)` (json.go)
- ⚠️ `StaleServiceDays` NÃO existe em Config (spec propõe adicionar)
- ⚠️ `LoadFromDB`/`initializeDefaultSettings` NÃO existem em main.go (spec propõe adicionar após InitSchema)

### Frontend (20 itens verificados)
- ✅ `Layout.tsx` é custom (não usa Sidebar do shadcn) — spec propõe migrar
- ✅ `useRefreshStore` com mode/setMode existe (refresh-store.ts, zustand)
- ✅ Breadcrumbs no Layout, PageHeader como componente separado
- ✅ `User.role` existe na interface (AuthContext.tsx:5-9)
- ✅ `useAuth()` retorna `{ user, logout, ... }` (AuthContext.tsx:11-18)
- ✅ `checkAuth` chama `/auth/status` e retorna `initialized` (AuthContext.tsx:27-50)
- ✅ React Router com `<Route element={<Layout />}>` e rotas aninhadas (App.tsx:47-63)
- ✅ `react-router-dom` v7, React 19, Vite 6, radix-ui, lucide-react, sonner, zustand (package.json)
- ✅ `dialog.tsx` é radix-based (oficial shadcn)
- ✅ `alert.tsx` tem variants warning e destructive
- ✅ `refresh-store.ts` é zustand com `mode` e `setMode`
- ⚠️ `dropdown-menu.tsx` é CUSTOM (não exporta DropdownMenuGroup) — spec propõe substituir
- ⚠️ `tabs.tsx` é CUSTOM (não radix) — spec propõe substituir
- ⚠️ Componentes faltantes: sidebar, form, switch, alert-dialog, checkbox (spec propõe adicionar via shadcn)
- ⚠️ Hooks faltantes: use-mobile.ts, use-permissions.ts (spec propõe criar)
- ⚠️ Páginas faltantes: Settings.tsx, Profile.tsx (spec propõe criar)
- ⚠️ Componentes faltantes: NavUser.tsx, app-sidebar.tsx, nav-main.tsx, nav-settings.tsx (spec propõe criar)
- ⚠️ Rotas faltantes: /settings/*, /profile (spec propõe adicionar)

### Correções aplicadas no spec após validação
1. `respondJSON` → `writeOK` (200) ou `writeJSON` (outros status) — alinhado com json.go
2. `decodeJSON(r, &req)` returning error → `if !decodeJSON(w, r, &req) { return }` — alinhado com json.go (retorna bool)
3. `tabs.tsx` adicionado à lista de componentes a substituir (era custom, não radix)
4. Helpers JSON documentados na seção 8.2.2 (writeJSON/writeOK/writeError/decodeJSON/pathValue)
5. `handleConfig` atual retorna apenas 3 campos — documentado na seção 8.4.5

**Conclusão:** Spec está compatível com o código atual. Todas as divergências de API foram corrigidas.
