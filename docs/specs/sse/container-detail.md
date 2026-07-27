# Spec SSE — ContainerDetail.tsx

> Página: ContainerDetail (detalhes de um container específico)
> Arquivo: `frontend/src/pages/ContainerDetail.tsx`
> Tipo: **Página de Detalhe**

## Status: ✅ Conforme (otimização implementada — tópico container-detail/{id})

## Quadro Resumo

| Critério | Status |
|----------|--------|
| Hook centralizado | ✅ |
| GET inicial | ✅ |
| Polling fallback | ✅ 300_000 (exceção detail page) |
| invalidateQueries | ✅ não necessário — 3 queries via setQueryData |
| Query com fallbackInterval | ✅ |
| Sem ghost cache | ✅ |
| Sem console.debug | ✅ |
| Tópico correto | ✅ `container-detail/{id}` (on-demand, só se subscriber) |

## Otimização implementada (T3)

O tópico `container-detail/{id}` foi criado no backend (similar a
`service-detail/{name}`). O collector publica o payload completo
(`BuildContainerDetailData` = stats + metrics downsampled + network) a cada
coleta (~5s), mas apenas se houver subscriber SSE ativo no tópico
(`SubscribedTopicsByPrefix("container-detail/")`). O `applyContainerDetailPayload`
faz `setQueryData` para as 3 queries (`["container-stats", id]`,
`["container-metrics", id]`, `["container-network-info", id]`) — zero refetch
HTTP com SSE ativo.
