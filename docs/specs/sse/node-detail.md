# Spec SSE — NodeDetail.tsx

> Página: NodeDetail (detalhes de um node específico)
> Arquivo: `frontend/src/pages/NodeDetail.tsx`
> Tipo: **Página de Detalhe**

## Status: ✅ Conforme (2026-07-27)

## Quadro Resumo

| Critério | Status |
|----------|--------|
| Hook centralizado | ✅ |
| GET inicial | ✅ |
| Polling fallback | ✅ 300_000 (exceção detail page) |
| invalidateQueries | ✅ |
| Query com fallbackInterval | ✅ |
| Sem ghost cache | ✅ |
| Sem console.debug | ✅ |
| Tópicos corretos | ✅ |
| setQueryData para storage-summary | ✅ via tópico dashboard (evento storage) |

## Otimização aplicada (T2 — ajustes.md)

`["storage-summary"]` era invalidada a cada ~5s via tópico `metrics` (refetch HTTP).
Agora recebe `setQueryData` via tópico `dashboard` (evento `storage`, payload completo
de `BuildStorageSummary`, cadência `StorageInterval` = 300s) — zero refetch HTTP para
o caminho feliz.

Mudança aplicada em `frontend/src/pages/NodeDetail.tsx`:
- Removido `["storage-summary"]` do `invalidateQueries` do tópico `metrics`.
- Adicionada assinatura do tópico `dashboard` (sem `invalidateQueries` próprios —
  o hook cuida via `EVENT_QUERY_MAP["storage"]` → `["storage-summary"]`).
- `sseConnected` agora considera `sseDashboard` para o `fallbackInterval`.

Notas:
- O evento `cluster` (~60s, mesmo tópico `dashboard`) ainda invalida
  `["storage-summary"]` via `getUncoveredQueryKeys` (safety net) — 12x menos
  refetch que o invalidate a cada ~5s anterior.
- Reconciliação 30s do hook também invalida `["storage-summary"]` via
  `TOPIC_QUERY_MAP["dashboard"]` — safety net adicional para drift.
