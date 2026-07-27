# SSE — Plano de Execução de Ajustes

> Arquivo de controle para rastrear a execução das correções/otimizações identificadas
> na auditoria das specs em `docs/specs/sse/`.
>
> **Atualizado em:** 2026-07-27
> **Specs auditadas:** 11 (1 padrão + 10 páginas)
> **Specs conformes:** 7 (`dashboard`, `services`, `service-detail`, `alerts`, `tasks`, `schedules`, `pattern`)
> **Specs com ajuste:** 4 (`nodes`, `node-detail`, `container-detail`, `pages-without-sse`)

## Legenda

| Símbolo | Significado |
|---------|-------------|
| ⚠️ | Ajuste necessário (bug de staleness) |
| 🔧 | Otimização opcional (não é bug) |
| ✅ | Concluído |
| ⏳ | Em andamento |
| ⬜ | Pendente |
| ❌ | Bloqueado |

## Priorização

| Prioridade | Spec | Tipo | Esforço | Impacto |
|------------|------|------|---------|---------|
| **P0** | `nodes` | ⚠️ Bug | Baixo (1 linha) | Alto — dados de agent stale em produção |
| **P1** | `pages-without-sse` (Recommendations) | ⚠️ Bug | Médio | Alto — recomendações stale até manual refresh |
| **P2** | `node-detail` | 🔧 Otimização | Médio | Baixo — reduz refetch de storage-summary |
| **P3** | `container-detail` | 🔧 Otimização | Alto (backend + frontend) | Médio — elimina 3 refetchs a cada ~5s |

> P0 e P1 são bugs de staleness (dados desatualizados sem ação do usuário).
> P2 e P3 são otimizações — o sistema funciona, apenas faz mais refetch que o necessário.

---

## Tarefas

### T0 — `nodes.md` — Adicionar `["agents"]` ao invalidateQueries

- **Spec:** `docs/specs/sse/nodes.md`
- **Arquivo:** `frontend/src/pages/Nodes.tsx`
- **Tipo:** ⚠️ Bug (P0)
- **Status:** ✅ Concluído (2026-07-27)

#### Contexto

A query `["agents"]` (linha 78) usa `refetchInterval: fallbackInterval`, que vira `false`
quando SSE está conectado. Como o tópico `nodes` não invalida `["agents"]` (nem no
`TOPIC_QUERY_MAP` nem no `invalidateQueries` da página), os dados de agent na tabela
(status Active/Stale, `containers_count`) ficam stale indefinidamente enquanto SSE ativo.

#### Mudança

```diff
  const { isConnected: sseConnected } = useEventSource({
    topic: "nodes",
-   invalidateQueries: [["nodes"], ["cluster"]],
+   invalidateQueries: [["nodes"], ["cluster"], ["agents"]],
  })
```

#### Checklist

- [x] Aplicar diff em `frontend/src/pages/Nodes.tsx` (linha 64)
- [x] Atualizar spec `docs/specs/sse/nodes.md` — marcar Status como ✅ conforme
- [x] Verificar build: `docker compose exec go-dev go build ./...` (não afeta Go, mas rodar por consistência)
- [x] Verificar frontend build: `pnpm build` (em `frontend/`)
- [ ] Smoke test manual: abrir /nodes, validar que status do agent atualiza com SSE ativo
- [x] Commit: `fix(sse): invalidar ["agents"] na página Nodes quando SSE ativo`

#### Notas

- Alternativa descartada: adicionar `["agents"]` ao `TOPIC_QUERY_MAP["nodes"]` no hook.
  Motivo: nem toda página que assina `nodes` precisa de agents (NodeDetail já invalida
  `["node-agent", nodeId]` separadamente). Manter no `invalidateQueries` da página é
  mais cirúrgico.
- Frequência resultante: `["agents"]` invalidada a cada ~5s (cadência do tópico `nodes`)
  com coalescing de 2s — adequado para dados de agent.

---

### T1 — `pages-without-sse.md` — Adicionar real-time a Recommendations.tsx

- **Spec:** `docs/specs/sse/pages-without-sse.md`
- **Arquivo:** `frontend/src/pages/Recommendations.tsx`
- **Tipo:** ⚠️ Bug (P1)
- **Status:** ⛔ Fora de escopo — tela Recommendations será refatorada (2026-07-27)

#### Contexto

`Recommendations.tsx` mostra recomendações de recursos computadas pelo ML sidecar a
partir de métricas coletadas. As queries `["recommendations"]` (linha 755) e
`["storage-recommendations"]` (linha 760) **não têm `refetchInterval` nem SSE** —
ficam stale até manual refresh. Apenas `["pendingSchedules"]` (linha 765) tem
`refetchInterval: 30000`.

#### Decisão — fora de escopo

A tela `Recommendations.tsx` será refatorada em uma fase futura. Adicionar
real-time (SSE ou polling) agora seria trabalho descartado pela refatoração.
Portanto, esta tarefa foi removida do escopo do plano de ajustes SSE.

#### Checklist

- [x] Confirmar com usuário: tela será refatorada → fora de escopo
- [ ] ~~Verificar no backend se `change-log` publica evento quando recomendações mudam~~
- [ ] ~~Implementar opção escolhida em `frontend/src/pages/Recommendations.tsx`~~
- [ ] ~~Adicionar `refetchInterval: fallbackInterval` (ou safety net 300_000) nas duas queries~~
- [ ] ~~Atualizar spec `docs/specs/sse/pages-without-sse.md` — marcar Recommendations como ✅~~
- [ ] ~~Verificar frontend build: `pnpm build` (em `frontend/`)~~
- [ ] ~~Smoke test: abrir /recommendations, validar atualização sem manual refresh~~
- [ ] ~~Commit: `fix(sse): adicionar real-time a Recommendations.tsx`~~

#### Notas

- A query `["recs-summary"]` já é invalidada pelo tópico `services` via
  `RECONCILE_QUERY_MAP["services"]` (refresh 30s). Verificar se Recommendations.tsx
  usa `["recs-summary"]` ou se precisa de invalidação própria.
- Recomendações são computadas pelo ML sidecar — cadência de atualização depende do
  scheduler do ML, não do coletor de métricas. SSE de `services` (~5s) pode ser
  excessivo.
- **Reabrir esta tarefa quando a refatoração da tela Recommendations for planejada.**

---

### T2 — `node-detail.md` — Otimizar `["storage-summary"]` via setQueryData

- **Spec:** `docs/specs/sse/node-detail.md`
- **Arquivo:** `frontend/src/pages/NodeDetail.tsx` + `frontend/src/hooks/use-event-source.ts`
- **Tipo:** 🔧 Otimização (P2)
- **Status:** ✅ Concluído (2026-07-27)

#### Contexto

`["storage-summary"]` é invalidada a cada ~5s via tópico `metrics` (invalidate). O
tópico `dashboard` publica evento `storage` (~60s) com `buildStorageSummary` —
mesmo payload de `GET /api/storage/summary`. Poderia usar `setQueryData` via
`EVENT_QUERY_MAP["storage"]` em vez de invalidate, reduzindo refetch de 5s → 60s.

#### Mudança

1. Confirmar que `EVENT_QUERY_MAP["storage"]` já mapeia para `["storage-summary"]`
   (ver `pattern.md` seção 4 — sim, mapeia).
2. Remover `["storage-summary"]` do `invalidateQueries` do tópico `metrics` em
   NodeDetail.tsx (se presente).
3. Garantir que NodeDetail.tsx assina o tópico `dashboard` (para receber evento
   `storage` e aplicar `setQueryData`).

#### Checklist

- [x] Auditar `NodeDetail.tsx` — confirmar quais tópicos assina e o `invalidateQueries` de cada
- [x] Confirmar que `EVENT_QUERY_MAP["storage"]` mapeia para `["storage-summary"]` no hook
- [x] Ajustar `invalidateQueries` em NodeDetail.tsx (remover `["storage-summary"]` do tópico `metrics`)
- [x] Garantir assinatura do tópico `dashboard` em NodeDetail.tsx
- [x] Atualizar spec `docs/specs/sse/node-detail.md` — marcar Status como ✅ conforme
- [x] Verificar frontend build: `pnpm build` (em `frontend/`) — validado em T3
- [ ] Smoke test: abrir /nodes/:id, validar storage card atualiza a cada ~60s
- [x] Commit: `perf(sse): usar setQueryData para storage-summary em NodeDetail`

#### Notas

- Não é bug — funciona corretamente via invalidate. Apenas reduz carga (refetch 5s → 60s).
- Validar que `buildStorageSummary` é o payload completo (não parcial) — senão `setQueryData`
  causaria dados incompletos.

---

### T3 — `container-detail.md` — Criar tópico `container-detail/{id}` no backend

- **Spec:** `docs/specs/sse/container-detail.md`
- **Arquivo:** `app/api/internal/server/*` (broker + handler) + `frontend/src/pages/ContainerDetail.tsx` + `frontend/src/hooks/use-event-source.ts`
- **Tipo:** 🔧 Otimização (P3)
- **Status:** ✅ Concluído (2026-07-27)

#### Contexto

`ContainerDetail.tsx` assina tópico `metrics` (único disponível). O tópico `metrics`
publica `buildDashboardData`, não dados de container. As 3 queries
(`["container-stats", id]`, `["container-metrics", id]`, `["container-network", id]`)
fazem refetch HTTP a cada ~5s via invalidate. Criar um tópico `container-detail/{id}`
no backend (similar a `service-detail/{name}`) eliminaria os refetchs via `setQueryData`.

#### Escopo (maior esforço)

1. **Backend (Go):**
   - Adicionar builder `BuildContainerDetailData(containerID)` em `app/api/internal/`
   - Adicionar tópico `container-detail/{id}` no broker (publish on-demand ou ~5s só se subscriber)
   - Adicionar handler de subscribe em `app/api/internal/server/sse_handlers.go`
2. **Hook (frontend):**
   - Adicionar `container-detail` ao `EVENT_QUERY_MAP` (com lógica de múltiplas queries,
     similar a `applyServiceDetailPayload`)
3. **Página (frontend):**
   - Trocar tópico `metrics` por `container-detail/${containerId}` em ContainerDetail.tsx
   - Remover `invalidateQueries` das 3 queries (passam a ser via `setQueryData`)

#### Decisão de cadência

**On-demand (~5s só se subscriber)** — mesmo padrão do `service-detail/{name}`.
O collector usa `SubscribedTopicsByPrefix("container-detail/")` para descobrir
apenas os containers que alguém está vendo, sem iterar sobre todos os containers
conhecidos. Zero trabalho quando ninguém abre `/containers/:id`.

#### Checklist

- [x] Decidir cadência: on-demand (só se subscriber) vs ~5s fixo
- [x] Implementar `BuildContainerDetailData` no Go API
- [x] Registrar tópico `container-detail/{id}` no broker (via `SubscribedTopicsByPrefix`)
- [x] Adicionar handler de subscribe (`handleSSEContainerDetail` + rota)
- [x] Build Go: `docker compose exec go-dev go build ./...`
- [x] Test Go: `docker compose exec go-dev go test ./...`
- [x] Adicionar `container-detail` ao `EVENT_QUERY_MAP` no hook
- [x] Ajustar ContainerDetail.tsx — trocar tópico e remover invalidateQueries
- [x] Atualizar spec `docs/specs/sse/container-detail.md` — marcar Status como ✅ conforme
- [x] Atualizar `docs/specs/sse/pattern.md` seção 3 (adicionar tópico à tabela)
- [x] Verificar frontend build: `pnpm build` (em `frontend/`)
- [ ] Smoke test: abrir /containers/:id, validar atualização via SSE sem refetch HTTP
- [x] Commit: `feat(sse): tópico container-detail/{id} com setQueryData`

#### Notas

- **Maior esforço** — envolve backend Go + frontend. Decidido prosseguir pois elimina
  3 refetchs HTTP a cada ~5s por container aberto e segue o padrão já estabelecido
  do `service-detail/{name}`.
- O payload de `metrics` é downsampled (~300 pontos) como o service-detail — o GET
  retorna resolução completa, mas o SSE envia downsampled para limitar o payload.
  Tradeoff aceito (mesmo do service-detail).
- O `network` usa o mesmo shape do GET (`[]docker.ContainerNetwork` — PascalCase
  sem json tags). Há um bug pré-existente de serialização network (frontend lê
  snake_case mas o backend emite PascalCase) — fora do escopo do T3.

---

## Resumo de Progresso

| ID | Spec | Tipo | Prioridade | Status |
|----|------|------|------------|--------|
| T0 | `nodes` | ⚠️ Bug | P0 | ✅ |
| T1 | `pages-without-sse` | ⚠️ Bug | P1 | ⛔ Fora de escopo (tela será refatorada) |
| T2 | `node-detail` | 🔧 Otimização | P2 | ✅ |
| T3 | `container-detail` | 🔧 Otimização | P3 | ✅ |

**Total:** 4 tarefas — 2 bugs (P0/P1) + 2 otimizações (P2/P3)
- ✅ Concluídas: T0, T2, T3
- ⛔ Fora de escopo: T1 (tela Recommendations será refatorada)

## Ordem recomendada de execução

1. **T0** (P0, baixo esforço, alto impacto) — ✅ concluído
2. **T1** (P1, médio esforço, alto impacto) — ⛔ fora de escopo (tela será refatorada)
3. **T2** (P2, médio esforço, baixo impacto) — ✅ concluído
4. **T3** (P3, alto esforço, médio impacto) — ✅ concluído

## Verificação final (após todas as tarefas)

- [x] `docker compose exec go-dev go build ./...` sem erros
- [x] `docker compose exec go-dev go vet ./...` sem warnings
- [x] `docker compose exec go-dev gofmt -l .` sem arquivos não formatados
- [x] `pnpm build` (em `frontend/`) sem erros
- [ ] Smoke test completo: percorrer /nodes, /nodes/:id, /containers/:id, /recommendations
- [x] Atualizar `docs/specs/sse/pattern.md` seção 3 se novos tópicos foram adicionados
- [x] Commit final consolidando ajustes (commits individuais por tarefa: T0, T2, T3)
