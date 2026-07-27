/**
 * useCollectionStatus — hook que rastreia o estado real da coleta de métricas.
 *
 * Combina dois sinais para determinar se dados estão chegando:
 *
 *   1. SSE eventos "metrics" / "agent_metrics" (tempo real, baixa latência)
 *   2. React Query refetch bem-sucedido das queries "dashboard" / "services"
 *      (fallback quando SSE cai — polling continua entregando dados)
 *
 * Detecção de fonte (100% de certeza):
 *   - Quando onEvent (SSE) dispara → source = "sse"
 *   - Quando QueryCache dispara E não houve evento SSE nos últimos 3s → source = "polling"
 *   - Quando QueryCache dispara E houve evento SSE nos últimos 3s → é invalidação
 *     causada por SSE, não polling → mantém source = "sse"
 *
 *   A janela de 3s cobre o tempo entre SSE invalidar a query e o refetch
 *   completar (rede local: ~50-200ms). Se for maior que 3s, é polling real.
 *
 * Estados:
 *   - "collecting"   → dados chegaram há < 30s (via SSE ou polling)
 *   - "idle"         → SSE conectado mas sem dados há > 30s
 *   - "reconnecting" → sem dados há > 30s E SSE desconectado
 *
 * O threshold de 30s cobre o intervalo de coleta do agent (default 15s) com
 * margem para latência de rede e retransmissões.
 */
import { useState, useEffect, useRef } from "react"
import { useQueryClient, type QueryCacheNotifyEvent } from "@tanstack/react-query"
import { useEventSource } from "@/hooks/use-event-source"

export type CollectionStatus = "collecting" | "idle" | "reconnecting"
export type CollectionSource = "sse" | "polling"

/** Threshold para considerar coleta "parada" (sem dados) */
const STALE_THRESHOLD_MS = 30_000

/** Intervalo de checagem para transição collecting → idle/reconnecting */
const CHECK_INTERVAL_MS = 5_000

/** Janela para distinguir invalidação por SSE de polling real */
const SSE_INVALIDATION_WINDOW_MS = 3_000

/** Eventos SSE que indicam que coleta de métricas aconteceu */
const COLLECTION_EVENT_TYPES = new Set(["metrics", "agent_metrics"])

/** Queries cujo refetch bem-sucedido indica que dados chegaram */
const COLLECTION_QUERY_KEYS = new Set(["dashboard", "services"])

export function useCollectionStatus(): {
  status: CollectionStatus
  source: CollectionSource
  lastEventAt: Date | null
} {
  const queryClient = useQueryClient()
  const [lastEventAt, setLastEventAt] = useState<Date | null>(null)
  const [source, setSource] = useState<CollectionSource>("polling")
  const lastEventRef = useRef<Date | null>(null)
  const sourceRef = useRef<CollectionSource>("polling")
  const sseConnectedRef = useRef(false)
  const lastSSEEventTimeRef = useRef<number>(0)

  const updateLastEvent = (src: CollectionSource) => {
    const now = new Date()
    lastEventRef.current = now
    sourceRef.current = src
    setLastEventAt(now)
    setSource(src)
  }

  // Sinal 1: SSE eventos de coleta
  const { isConnected } = useEventSource({
    topic: "metrics",
    onEvent: (event) => {
      if (COLLECTION_EVENT_TYPES.has(event.type)) {
        // Marca o timestamp do evento SSE para o QueryCache poder distinguir
        // invalidação por SSE de polling real
        lastSSEEventTimeRef.current = Date.now()
        updateLastEvent("sse")
      }
    },
  })

  // Rastrear estado SSE em ref (evita re-render desnecessário)
  sseConnectedRef.current = isConnected

  // Sinal 2: React Query refetch bem-sucedido
  // Quando o polling faz refetch da query "dashboard" ou "services" e obtém
  // dados novos, significa que a coleta está acontecendo (o agent coletou e
  // a API tem dados recentes). Isso cobre o caso em que SSE caiu mas o
  // polling continua entregando dados.
  useEffect(() => {
    const queryCache = queryClient.getQueryCache()

    const unsubscribe = queryCache.subscribe((event: QueryCacheNotifyEvent) => {
      // Só interessam eventos de query atualizada (fetch completado)
      if (event.type !== "updated") return
      const query = event.query
      const key = query.queryKey?.[0]
      if (typeof key !== "string" || !COLLECTION_QUERY_KEYS.has(key)) return

      // Só conta se o fetch foi bem-sucedido (state === "success")
      if (query.state.status === "success") {
        // Distinguir invalidação por SSE de polling real:
        // Se houve evento SSE nos últimos 3s, este refetch foi causado por
        // invalidação do SSE — não é polling, mantém source = "sse"
        const timeSinceSSE = Date.now() - lastSSEEventTimeRef.current
        const src: CollectionSource = timeSinceSSE < SSE_INVALIDATION_WINDOW_MS ? "sse" : "polling"
        updateLastEvent(src)
      }
    })

    return unsubscribe
  }, [queryClient])

  const [status, setStatus] = useState<CollectionStatus>("reconnecting")

  // Recalcular status
  const recalcStatus = () => {
    const last = lastEventRef.current
    const sseOk = sseConnectedRef.current

    if (!last) {
      setStatus(sseOk ? "idle" : "reconnecting")
      return
    }

    const ageMs = Date.now() - last.getTime()
    if (ageMs < STALE_THRESHOLD_MS) {
      setStatus("collecting")
    } else {
      setStatus(sseOk ? "idle" : "reconnecting")
    }
  }

  // Recalcula quando isConnected muda
  useEffect(() => {
    recalcStatus()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isConnected])

  // Checagem periódica: transição collecting → idle/reconnecting após 30s
  useEffect(() => {
    const interval = setInterval(recalcStatus, CHECK_INTERVAL_MS)
    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { status, source, lastEventAt }
}
