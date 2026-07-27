# Spec SSE — Páginas sem SSE

> Páginas que não usam Server-Sent Events
> Tipo: **Sem SSE**

## Status: ⛔ Recommendations fora de escopo (tela será refatorada)

## Quadro Resumo

| Página | SSE? | Polling? | Necessita SSE? | Status |
|--------|------|----------|----------------|--------|
| Login.tsx | Não | Não | ❌ Não | ✅ N/A |
| Onboarding.tsx | Não | Não | ❌ Não | ✅ N/A |
| Profile.tsx | Não | Não | ❌ Não | ✅ N/A |
| Settings.tsx | Não | Não | ❌ Não | ✅ N/A |
| settings/UsersPage.tsx | Não | Não | ❌ Não | ✅ N/A |
| settings/ApiKeysPage.tsx | Não | Não | ❌ Não | ✅ N/A |
| settings/ParametersPage.tsx | Não | Não | ❌ Não | ✅ N/A |
| settings/DataPage.tsx | Não | Não | ❌ Não | ✅ N/A |
| Templates.tsx | Não | Não | ❌ Não | ✅ N/A |
| **Recommendations.tsx** | **Não** | **Não** | **⚠️ Sim** | **⛔ Fora de escopo (refatoração)** |

## Issue: Recommendations.tsx sem real-time

Recommendations.tsx mostra recomendações de recursos que são computadas pelo ML sidecar a partir de métricas coletadas. Não assina SSE nem tem polling. Os dados ficam stale até manual refresh.

### Decisão: fora de escopo

A tela `Recommendations.tsx` será refatorada em uma fase futura. Adicionar
real-time (SSE ou polling) agora seria trabalho descartado pela refatoração.
Reabrir quando a refatoração for planejada.

### Fix recomendado (referência futura)

Adicionar `useEventSource` com tópico `services` e invalidar a query de recomendações, ou no mínimo adicionar `refetchInterval` como fallback.
