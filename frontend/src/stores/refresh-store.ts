import { create } from "zustand"
import { persist } from "zustand/middleware"

export type RefreshMode = "auto" | "5s" | "30s" | "1m" | "5m" | "off"

const INTERVAL_MAP: Record<RefreshMode, number | false> = {
  "5s": 5_000,
  "30s": 30_000,
  "1m": 60_000,
  "5m": 300_000,
  off: false,
  auto: 30_000,
}

interface RefreshState {
  mode: RefreshMode
  setMode: (mode: RefreshMode) => void
}

export const useRefreshStore = create<RefreshState>()(
  persist(
    (set) => ({
      mode: "auto",
      setMode: (mode) => set({ mode }),
    }),
    { name: "resma-refresh" }
  )
)

export function getIntervalMs(mode: RefreshMode, collectInterval: number): number | false {
  if (mode === "auto") return collectInterval * 1000
  return INTERVAL_MAP[mode]
}
