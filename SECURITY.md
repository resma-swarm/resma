# RESMA — Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in RESMA, please report it responsibly:

1. **DO NOT** open a public GitHub issue
2. Email: security@resma.ai (or open a private security advisory on GitHub)
3. Include: description, steps to reproduce, potential impact
4. You will receive a response within 48 hours

## Security Measures

### Authentication
- **JWT** (access + refresh tokens) for internal/UI endpoints (`/api/*`)
- **API Keys** with scopes (read/write) for public endpoints (`/api/v1/*`)
- **bcrypt** for password hashing (cost 12)
- **bcrypt** for API key hashing (cost 10 — keys are high-entropy)

### Secrets Management
- **JWT secret**: never hardcoded — provided via env var or Docker Swarm secret
- **Production validation**: startup fails if JWT secret is empty or default
- **Docker Swarm secrets**: `RESMA_JWT_SECRET_FILE=/run/secrets/resma_jwt_secret`
- **Admin password**: if `RESMA_DEFAULT_ADMIN_PASSWORD` is empty, a random password is generated

### CORS
- Origins configured via `RESMA_CORS_ORIGINS` (CSV)
- `Access-Control-Allow-Credentials: true` for SSE cookie auth
- Wildcard `*` is never used with credentials

### Security Headers
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Strict-Transport-Security` (HTTPS only)

### SSE Auth
- Cookie `sse_session` HttpOnly (Max-Age=600s, SameSite=Lax)
- Dual auth: cookie OR Authorization bearer header
- Session cleanup every 5 minutes (expired sessions removed)

### ML Sidecar
- Listens only on internal Docker network (port 8081, not exposed externally)
- No external auth needed (network isolation)
- Health check endpoint without auth (internal only)

## API Key Rotation

If an API key is leaked:

1. Create a new API key via UI (`/api/auth/api-keys`)
2. Update your application to use the new key
3. Revoke the old key (sets `revoked_at` timestamp)
4. The old key is immediately rejected by the middleware

## Environment Variables

See `.env.example` for all configuration options. Never commit `.env` with real secrets.
