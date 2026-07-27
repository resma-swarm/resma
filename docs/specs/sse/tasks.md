# Spec SSE — Tasks.tsx

> Página: Tasks (tasks do Docker Swarm)
> Arquivo: `frontend/src/pages/Tasks.tsx`
> Tipo: **Página de Lista**

## Status: ✅ Conforme — nenhum ajuste necessário

## Quadro Resumo

| Critério | Status |
|----------|--------|
| Hook centralizado | ✅ |
| GET inicial | ✅ |
| Polling fallback | ✅ `false` |
| invalidateQueries | ✅ `[["tasks"], ["services-health"]]` |
| Query com fallbackInterval | ✅ |
| Sem ghost cache | ✅ |
| Sem console.debug | ✅ |
| Tópico correto | ✅ `tasks` |
