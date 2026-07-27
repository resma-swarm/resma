# Spec SSE — Alerts.tsx

> Página: Alerts (alertas de OOM, memory leaks e resource drifts)
> Arquivo: `frontend/src/pages/Alerts.tsx`
> Tipo: **Página Derivada** (dados derivados de métricas, sem payload SSE próprio)

## Status: ✅ Conforme — nenhum ajuste necessário

## Padrão aplicado

Seção 8.3 do `pattern.md` — Página Derivada.

## Implementação atual

- **SSE:** 2 tópicos (`dashboard` + `metrics`), `invalidateQueries: [["alerts"]]`
- **Polling:** `sseConnected ? false : refreshInterval`
- **Query:** `["alerts"]` com `refetchInterval: fallbackInterval`

## Justificativa

Alertas são computados server-side a partir de métricas (OOMs, leaks, drifts). Não há payload SSE direto para alertas — a página assina os tópicos cujas mudanças podem afetar o estado dos alertas e invalida a query `["alerts"]` para refetch.
