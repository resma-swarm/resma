/**
 * Tipos do Right-Sizing Studio — extensão do payload ML.
 * Estende a interface Recommendation com os campos novos do payload estendido
 * (ml-payload-schema.md §2): suggested_tiers, risk, explainability, histograms,
 * resources_freed, cpu.p99, mem.p95.
 */

export interface ResourceValues {
  cpu_limit: number
  mem_limit: number
  cpu_reservation: number
  mem_reservation: number
}

export interface ResourcesFreed {
  cpu_cores: number
  mem_bytes: number
  cpu_pct: number
  mem_pct: number
}

export interface SuggestedTier extends ResourceValues {
  margin_cpu: number
  margin_mem: number
  resources_freed: ResourcesFreed
}

export interface SuggestedTiers {
  conservative: SuggestedTier
  balanced: SuggestedTier
  aggressive: SuggestedTier
}

export type RiskLevel = "very_low" | "low" | "attention" | "high" | "critical"
export type RiskColor = "green" | "yellow" | "orange" | "red"

export interface Risk {
  level: RiskLevel
  score: number
  color: RiskColor
  reasons: string[]
  margin_cpu: number
  margin_mem: number
  forecast_vs_limit_pct: number
}

export interface ExplainabilityFactor {
  label: string
  value: string
  detail: string
}

export interface Explainability {
  summary: string
  factors: ExplainabilityFactor[]
}

export interface Histogram {
  buckets: number[]
  counts: number[]
}

export interface MemHistogram {
  buckets_mb: number[]
  counts: number[]
}

export interface Histograms {
  cpu: Histogram
  mem: MemHistogram
}

export interface MemoryTrend {
  slope_bytes_per_hour: number
  daily_growth_mb: number
  r_squared: number
  has_leak: boolean
  projected_mem_7d?: number
}

export interface Forecast {
  days_ahead: number
  projected_mem: number
  projected_mem_p99: number
  slope_bytes_per_hour: number
}

export type TierName = "conservative" | "balanced" | "aggressive"

/**
 * Recommendation estendida — backward compatible com a interface anterior.
 * Campos novos são opcionais (o payload collecting_data não os preenche).
 */
export interface Recommendation {
  service: string
  samples: number
  status: string
  stack?: string
  preset?: string
  current?: ResourceValues
  outliers_removed?: number
  cpu?: { p50: number; p95: number; p99?: number }
  mem?: { p50: number; p99: number; p95?: number }
  oom_events?: number
  has_drift?: boolean
  pattern?: string
  memory_trend?: MemoryTrend
  forecast?: Forecast
  suggested?: ResourceValues
  suggested_apply_time?: string | null
  confidence?: string
  // Right-Sizing Studio — campos novos
  suggested_tiers?: SuggestedTiers | null
  risk?: Risk
  explainability?: Explainability
  histograms?: Histograms | null
  resources_freed?: { balanced: ResourcesFreed } | null
}

// --- Hero Metric ---

export interface HeroData {
  optimized_count: number
  cpu_cores: number
  mem_bytes: number
  delta_optimized_count: number
  pending_count: number
}

/**
 * Calcula o hero metric client-side a partir das recomendações.
 * Soma resources_freed.balanced dos serviços over_provisioned.
 * (spec ml-payload-schema.md §9 — cálculo no frontend)
 */
export function calculateHero(recs: Recommendation[]): HeroData {
  let cpuCores = 0
  let memBytes = 0
  let pending = 0

  for (const rec of recs) {
    if (rec.status !== "over_provisioned") continue
    pending++
    if (!rec.resources_freed) continue
    cpuCores += rec.resources_freed.balanced.cpu_cores
    memBytes += rec.resources_freed.balanced.mem_bytes
  }

  return {
    optimized_count: 0, // requer change_log (R5) — 0 até R5 implementar apply→change_log
    cpu_cores: cpuCores,
    mem_bytes: memBytes,
    delta_optimized_count: 0, // requer histórico (R5)
    pending_count: pending,
  }
}

// --- Risk helpers ---

export const riskColorClasses: Record<RiskColor, string> = {
  green: "bg-green-500/10 text-green-700 border-green-500/30",
  yellow: "bg-yellow-500/10 text-yellow-700 border-yellow-500/30",
  orange: "bg-orange-500/10 text-orange-700 border-orange-500/30",
  red: "bg-red-500/10 text-red-700 border-red-500/30",
}

export const riskLevelLabel: Record<RiskLevel, string> = {
  very_low: "Muito baixo",
  low: "Baixo",
  attention: "Atenção",
  high: "Alto",
  critical: "Crítico",
}

// --- Pattern helpers (alinhado com _pattern_label do ML) ---

export function patternLabel(p?: string): string {
  if (p === "business_hours") return "Horário comercial"
  if (p === "constant") return "Constante"
  if (p === "batch") return "Batch"
  return "Desconhecido"
}
