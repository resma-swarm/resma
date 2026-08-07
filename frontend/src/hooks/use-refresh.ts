import { useQuery } from "@tanstack/react-query"
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
