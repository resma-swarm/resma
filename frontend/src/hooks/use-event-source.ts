/**
 * useEventSource — hook centralizado para SSE (Server-Sent Events).
 *
 * Estratégia híbrida: SSE envia payload completo (mesmo do GET), o frontend
 * usa setQueryData para atualizar o cache instantaneamente — zero refetch HTTP.
 * Se o payload não for aplicável (query não montada), faz fallback para
 * invalidateQueries (comportamento anterior).
 *
 * Features:
 * - Lifecycle completo do EventSource (open, message, error, close)
 * - Cleanup correto no useEffect return (React StrictMode dobra effects)
 * - setQueryData direto para eventos com payload completo (zero GET)
 * - Fallback para invalidateQueries quando setQueryData faz bail out
 * - Exponential backoff manual para reconexão
 * - withCredentials: true para enviar cookie cross-origin
 */
import { useEffect, useRef, useCallback, useState } from "react"
import { useQueryClient, type QueryClient } from "@tanstack/react-query"
import { sseSessionManager } from "@/hooks/sse-session-manager"
import { applySSEPayload, getUncoveredQueryKeys } from "@/hooks/sse-cache-updater"

interface SSEEvent {
  type: string
  payload: unknown
}

/**
 * Coalescing global singleton — agrupa invalidações de TODOS os hooks
 * useEventSource em uma única flush a cada COALESCE_INTERVAL_MS.
 *
 * Antes, cada hook tinha seu próprio coalescing de 500ms. Com 3 hooks
 * (Dashboard metrics, Dashboard dashboard, CollectingBadge metrics),
 * isso gerava até 6 invalidações/s, cada uma disparando refetch de
 * "dashboard" e "storage-summary" → 80 invalidações/10s no QueryCache.
 *
 * Com coalescing global, mesmo com 3 hooks recebendo 5 eventos/s do SSE,
 * as invalidações são agrupadas em 1 flush a cada 2s = 0.5 invalidações/s.
 */
const COALESCE_INTERVAL_MS = 2000

const globalCoalesce = {
  pending: new Set<string>(),
  timer: null as ReturnType<typeof setTimeout> | null,
  qc: null as QueryClient | null,
  flush() {
    if (!this.qc) return
    for (const key of this.pending) {
      try {
        this.qc.invalidateQueries({ queryKey: [key] })
      } catch {
        // ignore
      }
    }
    this.pending.clear()
    this.timer = null
  },
  schedule(queryKeys: string[][]) {
    for (const qk of queryKeys) {
      if (qk[0]) this.pending.add(qk[0] as string)
    }
    if (this.timer) return
    this.timer = setTimeout(() => this.flush(), COALESCE_INTERVAL_MS)
  },
}

interface UseEventSourceOptions {
  /** Tópico SSE (metrics, dashboard, events, services, nodes) */
  topic: string
  /** Query keys para invalidar quando receber evento */
  invalidateQueries?: string[][]
  /** Callback chamado quando receber evento */
  onEvent?: (event: SSEEvent) => void
  /** Habilitar/desabilitar SSE */
  enabled?: boolean
}

const SSE_BASE = "/api/sse"

// Mapeamento de tópicos SSE para query keys que devem ser invalidadas.
// IMPORTANTE: cada tópico só deve listar queries cujos dados são
// relevantes para esse tópico. Queries cobertas por setQueryData
// (EVENT_QUERY_MAP) de OUTRO tópico não devem aparecer aqui, senão
// getUncoveredQueryKeys vai invalidá-las a cada evento do tópico errado.
//
// Exemplo: "metrics" envia buildDashboardData (payload do dashboard).
// NÃO incluir ["services"] aqui — o tópico "services" cuida disso.
const TOPIC_QUERY_MAP: Record<string, string[][]> = {
  // Tópico metrics: payload = buildDashboardData. Só dashboard.
  metrics: [["dashboard"]],
  // Tópico dashboard: payload = buildDashboardData ou buildStorageSummary.
  dashboard: [["dashboard"], ["storage-summary"]],
  // Tópico events: payload = buildOOMEvents. oom-events + change-log.
  events: [["oom-events"], ["change-log"]],
  // Tópico services: payload = buildServicesList. services via setQueryData.
  // recs-summary e service-sparklines NÃO são invalidados aqui — são dados
  // derivados/lentos que só precisam da reconciliação 30s. Invalidar a cada
  // evento "services" (~1/s) causaria refetch excessivo.
  services: [["services"]],
  // Tópico nodes: payload = buildNodesList. nodes via setQueryData,
  // cluster via invalidate (não coberto pelo payload).
  nodes: [["nodes"], ["cluster"]],
  // Fase 7 — multi-node
  tasks: [["tasks"], ["services-health"]],
  agents: [["agents"], ["nodes"]],
  // Scheduler publica quando um schedule executa/falha/pula
  "change-log": [["schedules"], ["change-log"]],
  // Tópico service-detail/{name} — payload via setQueryData (múltiplas queries)
  // Não precisa invalidar nada aqui — o applySSEPayload cuida de todas as queries.
  "service-detail": [],
  // Tópico container-detail/{id} — payload via setQueryData (múltiplas queries)
  // Não precisa invalidar nada aqui — o applySSEPayload cuida de todas as queries.
  "container-detail": [],
}

// RECONCILE_QUERY_MAP: queries adicionais para a reconciliação 30s.
// Inclui queries derivadas/lentas que NÃO são invalidadas a cada evento SSE
// mas precisam de refresh periódico (recs-summary, etc).
// A reconciliação 30s invalida TOPIC_QUERY_MAP[topic] + RECONCILE_QUERY_MAP[topic].
const RECONCILE_QUERY_MAP: Record<string, string[][]> = {
  // recs-summary muda lentamente — refresh 30s é suficiente.
  // service-sparklines foi removido (substituído por risk_score no buildServicesList).
  services: [["recs-summary"]],
}

export function useEventSource({
  topic,
  invalidateQueries,
  onEvent,
  enabled = true,
}: UseEventSourceOptions) {
  const queryClient = useQueryClient()
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const coalesceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const pendingInvalidationsRef = useRef<Set<string[]>>(new Set())
  const [isConnected, setIsConnected] = useState(false)

  // Refs para estabilizar callbacks: invalidateQueries e onEvent são passados
  // como arrays literais / funções inline pelas páginas, mudando de identidade
  // a cada render. Se entrarem como deps do useCallback (flushInvalidations →
  // connect → useEffect), o EventSource é fechado e reaberto a cada render,
  // gerando um storm de reconnects (especialmente quando eventos SSE causam
  // refetch → re-render). Os refs mantêm o valor mais recente sem trocar a
  // identidade dos callbacks.
  const invalidateQueriesRef = useRef(invalidateQueries)
  invalidateQueriesRef.current = invalidateQueries
  const onEventRef = useRef(onEvent)
  onEventRef.current = onEvent

  const clearReconnectTimer = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
  }, [])

  const flushInvalidations = useCallback(() => {
    // Coalescing global: agrupa invalidações de todos os hooks em 1 flush
    // a cada COALESCE_INTERVAL_MS (2s). Antes, cada hook tinha seu próprio
    // coalescing de 500ms, gerando até 80 invalidações/10s com 3 hooks.
    globalCoalesce.qc = queryClient

    const iq = invalidateQueriesRef.current
    if (iq) {
      globalCoalesce.schedule(iq)
    }

    const topicQueries = TOPIC_QUERY_MAP[topic] || []
    globalCoalesce.schedule(topicQueries)

    const pending = Array.from(pendingInvalidationsRef.current)
    pendingInvalidationsRef.current.clear()
    globalCoalesce.schedule(pending)
  }, [queryClient, topic])

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    const url = `${SSE_BASE}/${topic}`
    const es = new EventSource(url, { withCredentials: true })
    eventSourceRef.current = es

    es.onopen = () => {
      reconnectAttemptsRef.current = 0
      setIsConnected(true)
    }

    es.onerror = () => {
      setIsConnected(false)
      es.close()
      eventSourceRef.current = null

      // Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s
      const attempts = reconnectAttemptsRef.current
      const delay = Math.min(1000 * Math.pow(2, attempts), 30000)
      reconnectAttemptsRef.current++

      clearReconnectTimer()
      reconnectTimeoutRef.current = setTimeout(async () => {
        // Antes de reconectar, renovar a sessão SSE (cookie pode ter expirado).
        // EventSource.onerror não expõe o HTTP status code, então não dá para
        // diferenciar 401 (cookie expirado) de network error. Renovar sempre é
        // mais seguro — o custo de um POST extra é insignificante vs. loop
        // infinito de 401 que deixaria o SSE morto até re-login.
        try {
          await sseSessionManager.refreshForReconnect()
        } catch {
          // SSE session é opcional — fallback para polling
        }
        connect()
      }, delay)
    }

    es.addEventListener("message", (e) => {
      try {
        const payload = JSON.parse(e.data)
        const cb = onEventRef.current
        if (cb) {
          cb({ type: e.type, payload })
        }

        // Eventos "message" (sem tipo) não têm mapeamento para setQueryData.
        // Fallback para invalidateQueries.
        if (e.type && TOPIC_QUERY_MAP[topic]) {
          flushInvalidations()
        }
      } catch {
        // ignore parse errors
      }
    })

    // O servidor envia eventos tipados (event: <type>).
    // EventSource só dispara listeners para o tipo exato do evento — eventos
    // sem listener registrado são silenciosamente descartados. Por isso
    // registramos listeners para TODOS os tipos que os publishers emitem:
    //   collector: metrics, services, nodes, tasks, cluster, storage, oom, stale
    //   agent_handlers: agent_metrics, oom, heartbeat
    //   scheduler: schedule
    //   broker: connected (evento inicial)
    const handleTypedEvent = (e: MessageEvent) => {
      try {
        const payload = JSON.parse(e.data)
        const cb = onEventRef.current
        if (cb) {
          // e.type é o nome do evento SSE (ex: "metrics", "agent_metrics").
          // O payload JSON do backend não inclui o tipo — ele vem do cabeçalho
          // "event: <type>" do SSE, acessível via e.type no MessageEvent.
          cb({ type: e.type, payload })
        }
        // Tentar setQueryData primeiro (zero refetch HTTP para a query principal).
        // Se applySSEPayload retornar true, o cache da query principal foi
        // atualizado diretamente. Mas pode haver OUTRAS queries no tópico
        // (TOPIC_QUERY_MAP) que não são cobertas pelo payload SSE — essas
        // ainda precisam de invalidateQueries (refetch HTTP).
        // Exemplo: tópico "services" cobre ["services"] via setQueryData,
        // mas ["recs-summary"] e ["service-sparklines"] precisam de invalidate.
        //
        // Evento "connected" é handshake do broker — não carrega dados,
        // não deve invalidar queries (senão causa 1 refetch desnecessário
        // a cada reconnect).
        if (e.type === "connected") {
          return
        }
        const applied = applySSEPayload(queryClient, e.type, payload)
        if (!applied) {
          // Evento não mapeado ou bail out — invalidar tudo do tópico
          flushInvalidations()
        } else {
          // Query principal atualizada via setQueryData. Invalidar apenas
          // as queries do tópico que NÃO são cobertas pelo EVENT_QUERY_MAP.
          const topicQueries = TOPIC_QUERY_MAP[topic] || []
          const uncovered = getUncoveredQueryKeys(topicQueries, e.type)
          if (uncovered.length > 0) {
            globalCoalesce.qc = queryClient
            globalCoalesce.schedule(uncovered)
          }
        }
      } catch {
        // ignore
      }
    }

    // Registrar listener genérico para todos os tipos de evento emitidos pelos publishers
    const EVENT_TYPES = [
      "metrics", "services", "nodes", "tasks", "cluster", "storage", "oom", "stale",
      "agent_metrics", "heartbeat", "schedule", "connected", "service-detail",
    ]
    for (const type of EVENT_TYPES) {
      es.addEventListener(type, handleTypedEvent as EventListener)
    }
  }, [topic, flushInvalidations, clearReconnectTimer])

  useEffect(() => {
    if (!enabled) {
      return
    }

    connect()

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
      clearReconnectTimer()
      if (coalesceTimeoutRef.current) {
        clearTimeout(coalesceTimeoutRef.current)
        coalesceTimeoutRef.current = null
      }
      setIsConnected(false)
    }
  }, [connect, enabled, clearReconnectTimer])

  // Reconciliação periódica (30s) — safety net para corrigir drift entre
  // SSE e estado real do DB. Mesmo com setQueryData, pode haver drift se:
  // - Um evento SSE for perdido (reconnect)
  // - O payload SSE for stale no momento do publish
  // - Múltiplos eventos chegarem fora de ordem
  // A cada 30s, invalida as queries do tópico (TOPIC_QUERY_MAP) + queries
  // derivadas/lentas (RECONCILE_QUERY_MAP) para forçar 1 refetch e garantir
  // que o cache está sincronizado com o DB.
  useEffect(() => {
    if (!enabled) return
    const RECONCILE_INTERVAL_MS = 30000
    const interval = setInterval(() => {
      const topicQueries = TOPIC_QUERY_MAP[topic] || []
      const reconcileQueries = RECONCILE_QUERY_MAP[topic] || []
      const allQueries = [...topicQueries, ...reconcileQueries]
      if (allQueries.length > 0) {
        globalCoalesce.qc = queryClient
        globalCoalesce.schedule(allQueries)
      }
    }, RECONCILE_INTERVAL_MS)
    return () => clearInterval(interval)
  }, [enabled, topic, queryClient])

  const reconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0
    connect()
  }, [connect])

  return { isConnected, reconnect }
}
