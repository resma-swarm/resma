# RESMA — RESource MAnager

> App customizado para gerenciar recursos (CPU/memória) de containers no Docker Swarm.

## Documentação

- [Documento Técnico Completo (MD)](./TECH_SPEC.md)
- [Documento Técnico Completo (HTML)](./TECH_SPEC.html)

## Credenciais de Desenvolvimento

O onboarding cria o primeiro usuário com role **owner** (acesso total, único).
Usuários adicionais são criados via UI em Configurações → Usuários ou via API.

| Usuário | Senha      | Role  | Descrição                        |
|---------|------------|-------|----------------------------------|
| `owner` | `owner123` | owner | Criado via onboarding (primeiro) |
| `admin` | `admin123` | admin | Administrador (gestão completa)  |
| `user`  | `user123`  | user  | Usuário read-only (sem Config)   |

> **Nota:** O usuário `admin` com senha `admin123` (Fase 2) foi substituído
> pelo role `owner` na Fase 8 (RBAC). O onboarding agora cria `owner` em vez
> de `admin`. Para resetar o DB e fazer onboarding fresh, pare o go-dev,
> apague `data/resma.duckdb` e `data/resma.duckdb.wal`, e reinicie.

## Stack

- **Backend:** Python + FastAPI
- **Frontend:** React + Vite + TailwindCSS + shadcn/ui
- **Banco:** DuckDB (embedded, OLAP)
- **ML:** scikit-learn + scipy + numpy (3 técnicas simples)
- **Docker API:** aiodocker (async)
