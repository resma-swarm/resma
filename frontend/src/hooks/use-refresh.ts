import { useEffect, useRef } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useRefreshStore, getIntervalMs } from "@/stores/refresh-store"
import { api } from "@/api/client"

interface AppConfig {
  collect_interval: number
  retention_days: number
  analysis_window_days: number
}

let cachedCollectInterval = 15

export function useRefreshInterval(): number | false {
  const mode = useRefreshStore((s) => s.mode)
  return getIntervalMs(mode)
}

export function useCollectInterval(): number {
  const { data: config } = useQuery<AppConfig>({
    queryKey: ["app-config"],
    queryFn: () => api.get<AppConfig>("/config"),
    staleTime: 60_000,
  })

  if (config?.collect_interval) {
    cachedCollectInterval = config.collect_interval
  }

  return cachedCollectInterval
}

/**
 * useRefreshTimer — timer periódico que invalida queries na cadência
 * definida pelo dropdown de refresh (modo auto/5s/30s/1m/5m/off).
 *
 * Substitui (Frente C — spec phase-intervals-refresh):
 * - A reconciliação 30s hardcoded de use-event-source.ts
 * - O padrão fallbackInterval (sseConnected ? false : refreshInterval)
 * - Valores hardcoded 30000/300_000 em páginas
 *
 * Comportamento por cenário:
 * - SSE ativo + dropdown "5s"  → reconciliação safety-net a cada 5s
 * - SSE ativo + dropdown "off" → sem reconciliação (só SSE push)
 * - SSE inativo + dropdown "5s" → polling puro a cada 5s (fallback)
 * - SSE inativo + dropdown "off" → sem atualização automática (só manual)
 *
 * Pausa quando a aba está oculta (visibilitychange) — alinhado com o
 * comportamento do Grafana (SceneRefreshPicker) e economia de requests.
 */
export function useRefreshTimer(queryKeys: string[][]) {
  const queryClient = useQueryClient()
  const mode = useRefreshStore((s) => s.mode)
  const intervalMs = getIntervalMs(mode)

  // queryKeys é passado como array literal pelas páginas, mudando identidade
  // a cada render. O ref mantém o valor mais recente sem trocar a identidade
  // do effect — evita tear down/recreate do interval a cada render.
  const queryKeysRef = useRef(queryKeys)
  queryKeysRef.current = queryKeys

  useEffect(() => {
    if (!intervalMs) return

    let intervalId: ReturnType<typeof setInterval> | null = null

    const start = () => {
      if (intervalId) return
      intervalId = setInterval(() => {
        queryClient.invalidateQueries({ queryKey: queryKeysRef.current })
      }, intervalMs)
    }

    const stop = () => {
      if (intervalId) {
        clearInterval(intervalId)
        intervalId = null
      }
    }

    const onVisibilityChange = () => {
      if (document.hidden) {
        stop()
      } else {
        start()
      }
    }

    // Só inicia se a aba estiver visível — não desperdiça requests em aba oculta
    if (!document.hidden) {
      start()
    }

    document.addEventListener("visibilitychange", onVisibilityChange)

    return () => {
      stop()
      document.removeEventListener("visibilitychange", onVisibilityChange)
    }
  }, [intervalMs, queryClient])
}
