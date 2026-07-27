/**
 * sse-cache-updater — atualiza cache do React Query diretamente com payload SSE.
 *
 * Antes: SSE enviava sinais ({count: N}), o frontend fazia invalidateQueries
 * → GET HTTP → refetch. Cada "ding" do SSE = 1 GET completo.
 *
 * Agora: o backend publica o payload completo (mesmo do GET) no SSE.
 * Este módulo faz setQueryData(queryKey, payload) — zero GETs, UI instantânea.
 *
 * Estratégia:
 * 1. setQueryData para eventos com payload completo (zero refetch)
 * 2. Bail out se old === undefined (query não montada — não criar ghost cache)
 * 3. Reconciliação periódica (30s) via invalidateQueries — safety net para
 *    corrigir drift entre SSE e estado real do DB
 *
 * Mapeamento event.type → queryKey:
 * - metrics / agent_metrics → ["dashboard"]
 * - services → ["services"]
 * - nodes → ["nodes"]
 * - cluster → ["dashboard"] (cluster faz parte do dashboard)
 * - storage → ["storage-summary"]
 * - tasks → ["tasks"]
 * - stale / heartbeat → ["agents"]
 * - oom → ["oom-events"]
 * - schedule → ["schedules"]
 * - change-log → ["change-log"]
 */
import type { QueryClient } from "@tanstack/react-query"

/**
 * Mapeia event.type SSE → queryKey do React Query.
 * O payload do SSE é o mesmo do GET correspondente.
 */
const EVENT_QUERY_MAP: Record<string, string[]> = {
  // Tópico metrics — collector publica buildDashboardData
  metrics: ["dashboard"],
  agent_metrics: ["dashboard"],
  // Tópico services — collector publica buildServicesList
  services: ["services"],
  // Tópico nodes — collector publica buildNodesList
  nodes: ["nodes"],
  // Tópico dashboard — collector publica buildDashboardData ou buildStorageSummary
  cluster: ["dashboard"],
  storage: ["storage-summary"],
  // Tópico tasks — collector publica buildTasksList
  tasks: ["tasks"],
  // Tópico agents — collector/agent_handlers publica buildAgentsList
  stale: ["agents"],
  heartbeat: ["agents"],
  // Tópico events — agent_handlers publica buildOOMEvents
  oom: ["oom-events"],
  // Tópico change-log — scheduler publica buildSchedulesList / buildChangeLog
  schedule: ["schedules"],
  "change-log": ["change-log"],
  schedules: ["schedules"],
  // Tópico service-detail/{name} — collector publica BuildServiceDetailData
  // Payload tem múltiplas queries — tratado separadamente em applySSEPayload
  "service-detail": ["service-detail"],
  // Tópico container-detail/{id} — collector publica BuildContainerDetailData
  // Payload tem múltiplas queries — tratado separadamente em applySSEPayload
  "container-detail": ["container-detail"],
}

/**
 * Aplica payload SSE no cache do React Query via setQueryData.
 *
 * Regras:
 * - Bail out se queryKey não está no mapa (evento não mapeado)
 * - Bail out se old === undefined (query não montada — não criar ghost cache)
 * - Substitui completamente o cache (payload é o mesmo do GET)
 *
 * @returns true se atualizou o cache, false se fez bail out
 */
export function applySSEPayload(
  qc: QueryClient,
  eventType: string,
  payload: unknown,
): boolean {
  const queryKey = EVENT_QUERY_MAP[eventType]
  if (!queryKey) {
    return false
  }

  // Bail out se payload é null/undefined ou parece ser um signal antigo
  if (payload == null || typeof payload !== "object") {
    return false
  }

  // Caso especial: service-detail publica um payload com múltiplas queries
  // (stats, metrics, containers, tasks, health). Aplicar setQueryData para cada uma.
  if (eventType === "service-detail") {
    return applyServiceDetailPayload(qc, payload as Record<string, unknown>)
  }

  // Caso especial: container-detail publica um payload com múltiplas queries
  // (stats, metrics, network). Aplicar setQueryData para cada uma.
  if (eventType === "container-detail") {
    return applyContainerDetailPayload(qc, payload as Record<string, unknown>)
  }

  // Verificar se a query está montada (tem cache ou está loading)
  // Se old === undefined, a query nunca foi feita — não criar ghost cache.
  // A query será montada quando a página carregar e fizer o GET inicial.
  const oldData = qc.getQueryData(queryKey)
  if (oldData === undefined) {
    return false
  }

  // DEBUG: log para validar que setQueryData está atualizando o cache
  // Remover após validação
  console.debug(`[SSE] setQueryData(${JSON.stringify(queryKey)}, ${JSON.stringify(payload).length}bytes)`)

  // Set cache diretamente — zero refetch HTTP
  qc.setQueryData(queryKey, payload)
  return true
}

/**
 * Aplica o payload do service-detail no cache do React Query.
 * O payload tem a estrutura:
 * {
 *   service: "name",
 *   stats: { ... },
 *   metrics: [ ... ],
 *   containers: [ ... ],
 *   tasks: [ ... ],
 *   health: [ ... ]
 * }
 *
 * Faz setQueryData para cada query do ServiceDetail:
 * - ["service-stats", name] ← payload.stats
 * - ["service-metrics", name] ← payload.metrics
 * - ["service-containers", name] ← payload.containers
 * - ["service-tasks", name] ← payload.tasks
 * - ["services-health"] ← payload.health
 *
 * @returns true se atualizou pelo menos uma query, false se todas fizeram bail out
 */
function applyServiceDetailPayload(
  qc: QueryClient,
  payload: Record<string, unknown>,
): boolean {
  const service = payload["service"] as string
  if (!service) {
    return false
  }

  const updates: Array<[string[], unknown]> = [
    [["service-stats", service], payload["stats"]],
    [["service-metrics", service], payload["metrics"]],
    [["service-containers", service], payload["containers"]],
    [["service-tasks", service], payload["tasks"]],
    [["services-health"], payload["health"]],
    [["service-overview", service], {
      status: payload["status"],
      last_seen: payload["last_seen"],
      risk_score: payload["risk_score"],
      risk_factors: payload["risk_factors"],
    }],
  ]

  let applied = false
  for (const [qk, data] of updates) {
    if (data === undefined || data === null) {
      continue
    }
    // Bail out se a query não está montada (não criar ghost cache)
    if (qc.getQueryData(qk) === undefined) {
      continue
    }
    qc.setQueryData(qk, data)
    applied = true
  }

  if (applied) {
    console.debug(`[SSE] service-detail setQueryData for ${service}`)
  }
  return applied
}

/**
 * Aplica o payload do container-detail no cache do React Query.
 * O payload tem a estrutura:
 * {
 *   container_id: "id",
 *   stats: { ... },
 *   metrics: [ ... ],
 *   network: [ ... ]
 * }
 *
 * Faz setQueryData para cada query do ContainerDetail:
 * - ["container-stats", id]       ← payload.stats
 * - ["container-metrics", id]     ← payload.metrics
 * - ["container-network-info", id] ← payload.network
 *
 * @returns true se atualizou pelo menos uma query, false se todas fizeram bail out
 */
function applyContainerDetailPayload(
  qc: QueryClient,
  payload: Record<string, unknown>,
): boolean {
  const containerId = payload["container_id"] as string
  if (!containerId) {
    return false
  }

  const updates: Array<[string[], unknown]> = [
    [["container-stats", containerId], payload["stats"]],
    [["container-metrics", containerId], payload["metrics"]],
    [["container-network-info", containerId], payload["network"]],
  ]

  let applied = false
  for (const [qk, data] of updates) {
    if (data === undefined || data === null) {
      continue
    }
    // Bail out se a query não está montada (não criar ghost cache)
    if (qc.getQueryData(qk) === undefined) {
      continue
    }
    qc.setQueryData(qk, data)
    applied = true
  }

  if (applied) {
    console.debug(`[SSE] container-detail setQueryData for ${containerId}`)
  }
  return applied
}

/**
 * Retorna as queryKeys do tópico que NÃO são cobertas pelo EVENT_QUERY_MAP
 * para o eventType dado. Essas queries devem ser invalidadas via
 * invalidateQueries (refetch HTTP) porque o SSE não envia payload para elas.
 *
 * Exemplo: tópico "services" tem TOPIC_QUERY_MAP = [["services"], ["recs-summary"], ["service-sparklines"]].
 * O evento "services" cobre ["services"] via setQueryData. Mas ["recs-summary"]
 * e ["service-sparklines"] não têm payload no SSE — precisam de invalidate.
 *
 * @param topicQueryKeys QueryKeys associadas ao tópico SSE (TOPIC_QUERY_MAP[topic])
 * @param eventType Tipo do evento SSE recebido
 * @returns QueryKeys que ainda precisam de invalidateQueries
 */
export function getUncoveredQueryKeys(
  topicQueryKeys: string[][],
  eventType: string,
): string[][] {
  const coveredKey = EVENT_QUERY_MAP[eventType]
  if (!coveredKey) {
    // Evento não mapeado — todas as queries do tópico precisam de invalidate
    return topicQueryKeys
  }
  const coveredKeyStr = JSON.stringify(coveredKey)
  return topicQueryKeys.filter(qk => JSON.stringify(qk) !== coveredKeyStr)
}

/**
 * Verifica se um event.type tem mapeamento para setQueryData.
 * Usado pelo use-event-source.ts para decidir entre setQueryData vs invalidate.
 */
export function hasSSEMapping(eventType: string): boolean {
  return eventType in EVENT_QUERY_MAP
}
