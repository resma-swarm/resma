# Spec SSE — ServiceDetail.tsx

> Página: ServiceDetail (detalhes de um serviço)
> Arquivo: `frontend/src/pages/ServiceDetail.tsx`
> Tipo: **Página de Detalhe** (multi-query via setQueryData)

## Status: ✅ Conforme — nenhum ajuste necessário

## Quadro Resumo

| Critério | Status |
|----------|--------|
| Hook centralizado | ✅ |
| GET inicial | ✅ |
| Polling fallback | ✅ `false` |
| invalidateQueries | ✅ `[]` (tudo via setQueryData) |
| Query com fallbackInterval | ✅ |
| Sem ghost cache | ✅ |
| Sem console.debug | ✅ |
| Tópico correto | ✅ `service-detail/{name}` |
