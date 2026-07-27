# Spec SSE — Dashboard.tsx

> Página: Dashboard (visão geral do cluster)
> Arquivo: `frontend/src/pages/Dashboard.tsx`
> Tipo: **Página de Lista** (multi-tópico)

## Status: ✅ Conforme — nenhum ajuste necessário

## Quadro Resumo

| Critério | Status |
|----------|--------|
| Hook centralizado | ✅ |
| GET inicial | ✅ |
| Polling fallback | ✅ `false` |
| invalidateQueries | ✅ `[["dashboard"], ["storage-summary"]]` |
| Query com fallbackInterval | ✅ |
| Sem ghost cache | ✅ |
| Sem console.debug | ✅ |
| Tópicos corretos | ✅ `dashboard` + `metrics` |
