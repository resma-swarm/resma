# Spec SSE — Nodes.tsx

> Página: Nodes (lista de nodes do cluster Docker Swarm)
> Arquivo: `frontend/src/pages/Nodes.tsx`
> Tipo: **Página de Lista** (List Page)

## Status: ✅ Conforme

## Implementação atual

- **SSE:** 1 tópico (`nodes`), `invalidateQueries: [["nodes"], ["cluster"], ["agents"]]`
- **Polling:** `sseConnected ? false : refreshInterval`
- **Queries:**
  - `["nodes"]` — `refetchInterval: fallbackInterval` ✅
  - `["agents"]` — `refetchInterval: fallbackInterval` ✅ (invalidado pelo tópico `nodes`)

## Issue: `["agents"]` fica stale quando SSE ativo

### Problema

A query `["agents"]` (linha 77) usa `fallbackInterval` que fica `false` quando SSE está conectado. Mas:

1. O tópico `nodes` publica `buildNodesList` → `setQueryData(["nodes"])` via `EVENT_QUERY_MAP`
2. `TOPIC_QUERY_MAP["nodes"] = [["nodes"], ["cluster"]]` — não inclui `["agents"]`
3. `invalidateQueries` da página = `[["nodes"], ["cluster"]]` — não inclui `["agents"]`
4. `RECONCILE_QUERY_MAP["nodes"]` não existe

Resultado: quando SSE está ativo, `["agents"]` não recebe polling nem invalidação. Os dados de agents na tabela (status Active/Stale, containers_count) ficam stale indefinidamente.

### Fix

Adicionar `["agents"]` ao `invalidateQueries` da página:

```diff
  const { isConnected: sseConnected } = useEventSource({
    topic: "nodes",
-   invalidateQueries: [["nodes"], ["cluster"]],
+   invalidateQueries: [["nodes"], ["cluster"], ["agents"]],
  })
```

### Justificativa

A página Nodes mostra status do agent (Active/Stale) e containers_count na tabela. Esses dados vêm da query `["agents"]`. Sem invalidação, ficam desatualizados quando SSE está ativo. O tópico `nodes` publica a cada ~5s, então `["agents"]` será invalidada a cada ~5s (com coalescing 2s) — frequência adequada para dados de agent.

### Alternativa considerada

Adicionar `["agents"]` ao `TOPIC_QUERY_MAP["nodes"]` no hook. **Descartado** porque nem toda página que assina `nodes` precisa de agents (ex: NodeDetail já invalida `["node-agent", nodeId]` separadamente). Manter no `invalidateQueries` da página é mais cirúrgico.
