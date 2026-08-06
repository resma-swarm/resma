# RESMA — Schedules Redesign (UX/Produto)

> **Status:** Implementação em andamento
> **Domínio:** Product & Design · Frontend
> **Fase:** pós-Right-Sizing Studio (ajustes de consistência UI/UX)
> **Baseado em:** Análise do Conselho de Revisão (AI Engineering Framework)
> **Alvo:** `frontend/src/pages/Schedules.tsx` + backend `schedule_handlers.go`/`misc_handlers.go`

---

## Contexto

A tela de Agendamentos (`/schedules`) foi identificada como inconsistente com os
padrões evoluídos no Studio (Optimizations), Alerts e RollbackWatches. A análise
do conselho identificou 9 problemas (P1-P9) agrupados em 2 fases de correção.

O problema mais crítico (P1) é de **separação semântica**: a aba "Agendamentos"
mostra TODOS os schedules (pending + completed + failed + cancelled), criando
confusão — o usuário vê agendamentos já passados na aba que deveria ser acionável.

---

## Problemas Identificados

### P1 — Crítico: separação semântica incorreta entre abas

| Aba | Endpoint atual | O que mostra | Problema |
|-----|---------------|-------------|----------|
| Agendamentos | `GET /schedules` (sem filtro) | TODOS os schedules | Mostra completed/failed junto com pending |
| Histórico | `GET /change-log` | Audit log de todas mudanças | Conceito diferente de "histórico de schedules" |

**Root cause:** `GET /api/schedules` retorna todos os status. O default do filtro
dropdown é `"all"`. A aba "Agendamentos" deveria mostrar apenas o acionável (pending).

### P2 — Timestamps absolutos em vez de relativos
Schedules usa `formatDate` (absoluto: "06/08, 10:58"). Padrão evoluído: `timeAgo`
relativo ("há 5min") + tooltip com absoluto (Studio, Alerts, RollbackWatches).

### P3 — Sem tooltips/HelpIcon
Schedules não tem nenhum tooltip. Studio tem tooltips extensivos. Alerts tem
HelpIcon em cada tipo. Conceitos como "running", "attempts", "scheduler vs manual"
merecem explicação contextual.

### P4 — Empty state sem CTA
Schedules: "Nenhum agendamento encontrado" — sem link. Alerts (após fix): CTA →
`/optimizations`. Studio: CTA "Limpar filtros" / "Recalcular".

### P5 — Sem busca textual
Tasks, Nodes, Services, RollbackWatches — todas têm busca. Schedules não. Com 37
entries no change-log, busca por serviço é necessária.

### P6 — Sem paginação
RollbackWatches tem paginação (PAGE_SIZE=20). Schedules não. Com growth do
change-log, a tabela fica pesada.

### P7 — Stat cards não são clicáveis
RollbackWatches tem stat cards clicáveis que filtram. Schedules tem 4 cards
informativos que não filtram — perdem a oportunidade de serem atalhos.

### P8 — Filtros não persistem
Tasks/Nodes/Services usam `useFilterStore` (zustand persist). Schedules usa
`useState` local — filtros se perdem ao navegar fora e voltar.

### P9 — Aba "Histórico" mistura conceitos
A aba "Histórico de Alterações" mostra o `change_log` (audit de TODAS mudanças:
manual, scheduler, auto-rollback). Mas o usuário espera ver também os schedules
que já executaram. Hoje, para ver um schedule que falhou, precisa filtrar na aba
"Agendamentos" — mas o default é "Todos".

---

## Solção Proposta

### Reestruturação em 3 abas (resolve P1 + P9)

| Aba | Endpoint | O que mostra | Ação principal |
|-----|----------|-------------|----------------|
| **Pendentes** | `GET /schedules?status=pending` | Schedules aguardando execução | Cancelar (X) |
| **Histórico** | `GET /schedules` (filtrar completed/failed/cancelled no frontend) | Schedules já executados | Ver erro, re-agendar |
| **Auditoria** | `GET /change-log` | Log de todas mudanças (manual/scheduler/rollback) | Somente leitura |

**Racional:** separa o acionável (pendentes) do passado (histórico de schedules)
do audit (change_log). O usuário sabe exatamente onde olhar.

---

## Fase 1 — Correção de fluxo (P0)

### Ajuste A — Reestruturar para 3 abas
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - Tab 1: "Pendentes" — filtra `schedules.filter(s => s.status === "pending")`
  - Tab 2: "Histórico" — filtra `schedules.filter(s => ["completed","failed","cancelled"].includes(s.status))`
  - Tab 3: "Auditoria" — changeLog entries
- **Stat cards:** permanecem no topo (visíveis em todas as abas), mas clicáveis
- **Default:** aba "Pendentes" selecionada

### Ajuste C — Timestamps relativos + tooltip
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - Substituir `formatDate` por `timeAgo` (relativo: "há 5min", "agora")
  - Envolver timestamp em `<Tooltip>` com absoluto (`formatDate` no tooltip)
  - Aplicar em: `scheduled_at`, `applied_at`, `created_at`
  - Reutilizar `timeAgo` do Alerts.tsx (ou extrair para `lib/time.ts`)

### Ajuste E — Empty state com CTA
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - Aba Pendentes vazia → "Nenhum agendamento pendente" + CTA → `/optimizations`
  - Aba Histórico vazia → "Nenhum agendamento executado ainda"
  - Aba Auditoria vazia → "Nenhuma alteração registrada"
  - Usar prop `action` do `EmptyState` (já estendido no fix do Alerts)

### Ajuste F — Busca textual
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - Input com ícone `Search` (mesmo padrão de Alerts/Tasks/Nodes)
  - Placeholder: "Buscar serviço ou mensagem..."
  - Filtra por `service` e `message`/`error` em todas as abas
  - Posição: barra de filtros acima da tabela (mesmo padrão Alerts)

### Ajuste H — Stat cards clicáveis
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - 4 cards: Pendentes / Concluídos / Falhas / Cancelados
  - Click em card → troca para aba relevante + filtra por status
  - Card ativo: `ring-2 ring-primary` (mesmo padrão RollbackWatches)
  - Card "Pendentes" → aba Pendentes
  - Card "Concluídos" / "Falhas" / "Cancelados" → aba Histórico + filtro status

---

## Fase 2 — Consistência (P1/P2)

### Ajuste D — HelpIcon em conceitos não-óbvios
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - Status "running" → HelpIcon: "Schedule em execução no momento"
  - Coluna "Tentativas" → HelpIcon: "Número de tentativas de aplicação (retry automático em caso de falha)"
  - Source "scheduler" vs "manual" (aba Auditoria) → HelpIcon explicativo
  - Action "apply" vs "rollback" (aba Auditoria) → HelpIcon explicativo
  - Usar componente `HelpIcon` existente (`@/components/help-icon`)

### Ajuste I — Filtros persistentes
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - Migrar `useState` para `useFilterStore` (zustand persist)
  - Persistir: `statusFilter`, `logFilter`, `search`, `activeTab`
  - Mesmo padrão de Tasks/Nodes/Services

### Ajuste G — Paginação
- **Arquivo:** `frontend/src/pages/Schedules.tsx`
- **Mudança:**
  - Client-side PAGE_SIZE=20 (mesmo padrão RollbackWatches)
  - Aplicar nas abas Histórico e Auditoria (onde volume cresce)
  - Botões Anterior/Próxima + contador "Página X de Y · Z entries"
  - Aba Pendentes não precisa (volume baixo)

---

## Validação

### Por fase
1. **Type check:** `pnpm exec tsc --noEmit` em `frontend/` (não rodar build completo)
2. **Revisão de código:** verificar padrões, imports, DS tokens
3. **Validação visual:** Playwright MCP — navegar, interagir, confirmar DOM

### Final (após Fase 2)
1. Rebuild imagem `resma-api:latest` (inclui frontend buildado)
2. Redeploy Swarm (`docker stack deploy`)
3. Playwright completo:
   - 3 abas renderizam corretamente
   - Stat cards clicáveis filtram
   - Timestamps relativos + tooltip absoluto
   - Busca funcional
   - Empty states com CTA
   - HelpIcon tooltips
   - Paginação
   - Filtros persistem após navegar fora e voltar

---

## Regras de Implementação

1. **NÃO alterar design tokens** — cores, typography, spacing permanecem
2. **Seguir padrões evoluídos** — Alerts, Studio, RollbackWatches como referência
3. **Type check por fase** — `pnpm exec tsc --noEmit` (não build)
4. **Build apenas no final** — para deploy do Docker
5. **Frontend embarcado na API** — rebuild `resma-api` image inclui frontend
6. **Docker Swarm** — deploy via `docker stack deploy`, não `docker compose up`
7. **Playwright por fase** — validar visualmente após cada fase
8. **Commits por fase** — commitar após cada fase validada
