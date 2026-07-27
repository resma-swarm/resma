---
name: ai
description: >-
  AI Engineering Framework — entry point único. Ativa o framework, permite invocar
  comandos naturais e skills diretamente, e fornece ajuda sobre o framework.
---

# AI Engineering Framework — Skill `/ai`

> Esta skill é o entry point do AI Engineering Framework no Devin.
> Ao ser invocada, ativa o framework e direciona a execução conforme o sub-comando.
>
> **Instalação:** O Devin procura `.devin/` na raiz do projeto. Copie esta pasta para a raiz:
> ```bash
> cp -r .ai/devin .devin
> ```

## Como usar

```
/ai [sub-comando] [descrição da tarefa]
```

| Forma | Ação |
|-------|------|
| `/ai` + descrição | Ativa framework, processa via NLME normal |
| `/ai <comando-natural>` + descrição | Pula NLME, vai direto para o workflow (ex: `/ai analise ...`) |
| `/ai <skill-name>` + descrição | Pula NLME, usa a skill como primária (ex: `/ai backend ...`) |
| `/ai help [tópico]` | Exibe ajuda sobre o framework |

## Comandos naturais disponíveis

Ver `.ai/docs/Natural-Commands.md` para o catálogo completo. Principais:

| Comando | Tipo de missão |
|---------|----------------|
| `analise` | Audit Mission |
| `revise` | Review Mission |
| `corrija` | Bug Mission |
| `implemente` | Development Mission |
| `otimize` | Performance Mission |
| `documente` | Knowledge Mission |
| `automatize` | Automation Mission |
| `configure` | Infrastructure Mission |

## Skills disponíveis

Ver `.ai/workflows/_index.md` para o índice completo. Principais:

| Skill | Uso |
|-------|-----|
| `backend` | Server/API Python |
| `api` | API REST |
| `react` | Frontend React |
| `database` | Banco de dados |
| `devops` | DevOps/infra |
| `qa` | Testes |
| `security-review` | Revisão de segurança |
| `performance` | Performance |

## Help

| Comando | Conteúdo |
|---------|----------|
| `/ai help` | Visão geral do framework |
| `/ai help comandos` | Lista completa de comandos naturais |
| `/ai help skills` | Lista de skills com descrição |
| `/ai help modos` | Explica os 4 modos: Fast, Standard, Review, Technical Council |
| `/ai help <skill-name>` | Detalha uma skill específica |

## Bootstrap

Ao receber `/ai`, seguir estes passos:

1. Ler `.ai/AGENTS.md` (bootstrap do framework)
2. Ler `.ai/skills/orchestrator/SKILL.md` (algoritmo do Orchestrator)
3. Aplicar `rules/direct-invocation.md` para processar o sub-comando
4. Detectar superfície: **Devin** (alta confiança — skill invocada via slash command)
5. Seguir o algoritmo do Orchestrator a partir do step 1

## Referências

- `.ai/AGENTS.md` — bootstrap do framework
- `.ai/rules/direct-invocation.md` — regra de invocação direta
- `.ai/skills/orchestrator/SKILL.md` — algoritmo do Orchestrator
- `.ai/docs/Natural-Commands.md` — catálogo de comandos naturais
- `.ai/workflows/_index.md` — índice de workflows
- `.ai/context/` — contexto do projeto (arquitetura, stack, domínio, padrões)
