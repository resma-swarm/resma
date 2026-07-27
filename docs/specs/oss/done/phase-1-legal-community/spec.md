# Fase 1 — Fundações Legais e Comunidade

> **Prioridade:** Crítica  
> **Esforço:** Baixo (arquivos de texto)  
> **Bloqueador:** Sim — deve ser concluída antes de tornar o repo público  
> **Dependências:** Nenhuma

## Objetivo

Estabelecer as fundações legais e de comunidade que todo projeto open-source precisa no GitHub. Estes arquivos são prerequisite para que contribuidores possam usar, modificar e contribuir com o projeto de forma legal e segura.

## Tarefas

### 1.1 — LICENSE

- **Arquivo:** `LICENSE` (raiz do repo)
- **Licença:** MIT
- **Justificativa:** Máxima permissividade, sem copyleft, atrai contribuidores e adoção empresarial
- **Conteúdo:** Texto integral da licença MIT com copyright year e holder
- **Exemplo:**
  ```
  MIT License

  Copyright (c) 2026 RESMA Contributors

  Permission is hereby granted, free of charge, to any person obtaining a copy
  ...
  ```

### 1.2 — README.md (raiz)

- **Arquivo:** `README.md` (raiz do repo)
- **Seções obrigatórias:**
  - Badges (CI status, license, Docker pulls, GitHub stars)
  - Título + tagline (1 frase)
  - Descrição (2-3 parágrafos)
  - Features (bullet list com ícones)
  - Screenshots/GIF do dashboard
  - Quickstart (1 comando: `docker stack deploy -c docker-stack.yml resma`)
  - Links para documentação completa
  - Requisitos (Docker Swarm, manager node)
  - Configuração (env vars principais)
  - Contribuindo (link para CONTRIBUTING.md)
  - Licença (link para LICENSE)
- **Referências:** Ver `docs/README.md` existente como base, mas o README da raiz deve ser mais completo
- **Tabela comparativa** (inspirada em Cetacean — diferencial competitivo):
  ```markdown
  | Feature | Swarmpit | Portainer | Cetacean | RESMA |
  |---------|----------|-----------|----------|-------|
  | ML Recommendations | ❌ | ❌ | ❌ | ✅ |
  | Memory Leak Detection | ❌ | ❌ | ❌ | ✅ |
  | OOM Tracking | ❌ | ❌ | ❌ | ✅ |
  | Scheduled Changes | ❌ | ❌ | ❌ | ✅ |
  | Single Container | ❌ | ❌ | ✅ | ❌ (2 containers leves) |
  | No External DB | ❌ | ❌ | ✅ | ✅ (DuckDB) |
  | No Agent Needed | ❌ | ❌ | ✅ | ✅ |
  | Resource Monitoring | ✅ | ✅ | ✅ Prometheus | ✅ DuckDB |
  ```
- **Roadmap público** no README (inspirado em Swarmpit — link para `docs/specs/oss/ROADMAP.md`)

### 1.3 — CONTRIBUTING.md

- **Arquivo:** `CONTRIBUTING.md` (raiz)
- **Seções:**
  - Pré-requisitos (Docker, Node 22, pnpm; Go 1.26 opcional para dev local sem Docker)
  - Setup de desenvolvimento (clone, `docker compose --profile dev up -d`, `docker compose --profile dev exec go-dev bash`, `pnpm install` em `frontend/`)
  - Comandos: `docker compose --profile dev exec go-dev go run ./cmd/server`, `pnpm dev` (em `frontend/`), `docker compose --profile prod up --build`
  - Workflow: fork → branch (`feat/`, `fix/`, `docs/`) → commit → PR
  - Conventional commits: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
  - Onde adicionar testes (Go: `*_test.go` ao lado do código; ML: `app/ml/tests/`)
  - Expectativa de documentação (atualizar docs junto com código; swaggo annotations se endpoint público `/api/v1/*`)
  - Processo de review (1 approval mínimo, CI verde)
  - Link para CODE_OF_CONDUCT.md
  - Link para SECURITY.md

### 1.4 — CODE_OF_CONDUCT.md

- **Arquivo:** `CODE_OF_CONDUCT.md` (raiz)
- **Padrão:** Contributor Covenant 2.1
- **Formato:** Link para canonical (não embed completo)
  ```markdown
  # Code of Conduct

  Este projeto adota o [Contributor Covenant 2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).

  Para reportar violações, abra uma issue privada ou contate via SECURITY.md.
  ```

### 1.5 — SECURITY.md

- **Arquivo:** `SECURITY.md` (raiz)
- **Conteúdo:**
  - Política de disclosure responsável
  - Escopo (apenas vulnerabilidades do RESMA, não de dependências)
  - Como reportar (email ou GitHub Security Advisories)
  - Tempo de resposta esperado (72h confirmação, 30d patch)
  - O que incluir no report (steps to reproduce, impacto, versão)
  - Agradecimentos

### 1.6 — CHANGELOG.md

- **Arquivo:** `CHANGELOG.md` (raiz)
- **Formato:** [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/)
- **Seções:** Added, Changed, Deprecated, Removed, Fixed, Security
- **Versão inicial:**
  ```markdown
  # Changelog

  ## [Unreleased]
  ### Added
  - Suporte open-source inicial (LICENSE, CONTRIBUTING, etc.)

  ## [0.1.0] - 2026-07-24
  ### Added
  - API Go com SSE (Server-Sent Events) para real-time
  - Coleta de métricas CPU/memória de containers Docker Swarm
  - Recomendações de limites via ML sidecar Python (scikit-learn)
  - Detecção de memory leaks via regressão linear
  - Dashboard com sparklines e métricas em tempo real
  - API split: `/api/v1/*` público (API key + OpenAPI) + `/api/*` interno (JWT)
  - Frontend React + Vite + TailwindCSS + shadcn/ui
  - DuckDB compartilhado entre API Go e ML sidecar Python
  ```

### 1.7 — .github/CODEOWNERS

- **Arquivo:** `.github/CODEOWNERS`
- **Conteúdo:**
  ```
  # Default owners
  * @USER_GITHUB

  # Go API
  /app/api/ @USER_GITHUB

  # Python ML sidecar
  /app/ml/ @USER_GITHUB

  # Frontend
  /frontend/ @USER_GITHUB

  # Docs
  /docs/ @USER_GITHUB
  /docs-site/ @USER_GITHUB
  ```
- **Nota:** Substituir `@USER_GITHUB` pelo handle real

### 1.8 — .github/pull_request_template.md

- **Arquivo:** `.github/pull_request_template.md`
- **Conteúdo:**
  ```markdown
  ## Descrição

  [Descreva o que este PR faz]

  ## Tipo de mudança

  - [ ] Bug fix
  - [ ] Nova feature
  - [ ] Breaking change
  - [ ] Documentação
  - [ ] Refatoração

  ## Checklist

  - [ ] Código segue o estilo do projeto
  - [ ] Testes adicionados/atualizados
  - [ ] Documentação atualizada
  - [ ] CHANGELOG.md atualizado
  - [ ] Nenhum secret exposto
  ```

### 1.9 — .github/ISSUE_TEMPLATE/

- **Arquivos:**
  - `.github/ISSUE_TEMPLATE/bug_report.yml` — template estruturado (GitHub Issue Forms)
  - `.github/ISSUE_TEMPLATE/feature_request.yml` — template estruturado
  - `.github/ISSUE_TEMPLATE/config.yml` — desabilitar blank issues, adicionar links

- **bug_report.yml campos:**
  - Descrição do bug
  - Passos para reproduzir
  - Comportamento esperado vs atual
  - Ambiente (OS, Docker version, Swarm size, RESMA version)
  - Logs relevantes

- **feature_request.yml campos:**
  - Problema que resolve
  - Solução proposta
  - Alternativas consideradas
  - Contexto adicional

### 1.10 — .github/FUNDING.yml (opcional)

- **Arquivo:** `.github/FUNDING.yml`
- **Conteúdo:**
  ```yaml
  github: [USER_GITHUB]
  ```
- **Nota:** Opcional — habilita botão "Sponsor" no GitHub

## Critérios de aceite

- [ ] LICENSE existe na raiz com texto integral da MIT
- [ ] README.md na raiz com todas as seções obrigatórias
- [ ] CONTRIBUTING.md com workflow completo
- [ ] CODE_OF_CONDUCT.md linkando Contributor Covenant 2.1
- [ ] SECURITY.md com política de disclosure
- [ ] CHANGELOG.md no formato Keep a Changelog
- [ ] .github/CODEOWNERS configurado
- [ ] .github/pull_request_template.md existe
- [ ] .github/ISSUE_TEMPLATE/ com bug_report.yml e feature_request.yml
- [ ] Nenhum secret ou credencial em nenhum arquivo

## Referências

- [OSS_SPEC.md](https://github.com/niclaslindstedt/oss-spec) — especificação prescritiva para bootstrap OSS
- [Contributor Covenant 2.1](https://www.contributor-covenant.org/version/2/1/code_of_conduct/)
- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
- [GitHub Issue Forms](https://docs.github.com/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)
