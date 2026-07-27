# Spec SSE — Schedules.tsx

> Página: Schedules (agendamentos e histórico de alterações)
> Arquivo: `frontend/src/pages/Schedules.tsx`
> Tipo: **Página de Lista**

## Status: ✅ Conforme — nenhum ajuste necessário

## Quadro Resumo

| Critério | Status |
|----------|--------|
| Hook centralizado | ✅ |
| GET inicial | ✅ |
| Polling fallback | ✅ `false` / 30000 |
| invalidateQueries | ✅ `[["schedules"], ["change-log"]]` |
| Query com fallbackInterval | ✅ |
| Sem ghost cache | ✅ |
| Sem console.debug | ✅ |
| Tópico correto | ✅ `change-log` |
