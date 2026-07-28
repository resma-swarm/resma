/**
 * ExplainabilityPanel — painel collapsible com explicação da recomendação.
 *
 * Spec: explainability-rendering.md
 * 3 zonas: texto natural (summary), fatores estruturados (chips com tooltip),
 * gráficos Recharts (histograma CPU, P50/P95/P99, timeline memória).
 *
 * shadcn: Collapsible, Button, Badge, Tooltip
 * Recharts: BarChart, AreaChart, LineChart, ReferenceLine
 * Reusa: HelpIcon (não duplica Tooltip+HelpCircle)
 */
import { useState } from "react"
import { ChevronDown, ChevronRight } from "lucide-react"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  BarChart, Bar, AreaChart, Area, Line,
  ReferenceLine, XAxis, YAxis, ResponsiveContainer, Cell,
} from "recharts"
import { formatBytes } from "@/lib/utils"
import type {
  Explainability,
  Histograms,
  MemoryTrend,
  Forecast,
  TierName,
} from "./types"

interface ExplainabilityPanelProps {
  explainability: Explainability
  histograms: Histograms | null
  cpuStats: { p50: number; p95: number; p99?: number }
  memStats: { p50: number; p95?: number; p99: number }
  memoryTrend: MemoryTrend
  forecast: Forecast
  suggestedMemLimit: number
  selectedTier: TierName
}

export function ExplainabilityPanel({
  explainability,
  histograms,
  cpuStats,
  memStats,
  memoryTrend,
  forecast,
  suggestedMemLimit,
  selectedTier,
}: ExplainabilityPanelProps) {
  const [expanded, setExpanded] = useState(false)

  // Dados para gráfico de P50/P95/P99 CPU
  const cpuPercentileData = [
    { label: "P50", value: cpuStats.p50, fill: "hsl(var(--chart-2))" },
    { label: "P95", value: cpuStats.p95, fill: "hsl(var(--chart-3))" },
    { label: "P99", value: cpuStats.p99 ?? 0, fill: "hsl(var(--chart-5))" },
  ]

  // Dados para histograma de CPU
  const cpuHistogramData = histograms
    ? histograms.cpu.buckets.slice(0, -1).map((bucket, i) => ({
        label: `${bucket}%`,
        count: histograms.cpu.counts[i] ?? 0,
        bucketEnd: histograms.cpu.buckets[i + 1],
      }))
    : []

  // Dados para histograma de memória
  const memHistogramData = histograms
    ? histograms.mem.buckets_mb.slice(0, -1).map((bucket, i) => ({
        label: bucket >= 1000 ? `${(bucket / 1000).toFixed(0)}GB` : `${bucket}MB`,
        count: histograms.mem.counts[i] ?? 0,
      }))
    : []

  // Dados para timeline de memória + forecast (simplificado: pontos do forecast)
  const memForecastData = [
    { label: "Atual", usage: memStats.p99, forecast: null as number | null },
    { label: `+${forecast.days_ahead}d`, usage: null as number | null, forecast: forecast.projected_mem_p99 },
  ]

  const tierLabel = selectedTier === "conservative" ? "Conservadora" : selectedTier === "aggressive" ? "Agressiva" : "Equilibrada"

  return (
    <Collapsible open={expanded} onOpenChange={setExpanded}>
      <CollapsibleTrigger asChild>
        <Button variant="ghost" size="sm" className="w-full justify-between text-xs">
          <span className="text-muted-foreground">Por que estes valores? ({tierLabel})</span>
          {expanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
        </Button>
      </CollapsibleTrigger>
      <CollapsibleContent className="space-y-3 pt-2">
        {/* ZONA 1 — Texto natural */}
        <div className="p-3 bg-muted/50 rounded-lg">
          <p className="text-xs leading-relaxed text-foreground">{explainability.summary}</p>
        </div>

        {/* ZONA 2 — Fatores estruturados (chips com tooltip) */}
        <div className="flex flex-wrap gap-1.5">
          {explainability.factors.map((f) => (
            <TooltipProvider key={f.label}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge variant="outline" className="cursor-help text-[10px] gap-1">
                    <span className="text-muted-foreground">{f.label}:</span>
                    {f.value}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent side="top" className="max-w-48 text-xs">
                  {f.detail}
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ))}
        </div>

        {/* ZONA 3 — Gráficos (2x2 grid) */}
        {histograms && (
          <div className="grid grid-cols-2 gap-3">
            {/* Gráfico 1: Histograma CPU */}
            <div className="rounded-lg border p-2">
              <p className="text-[10px] text-muted-foreground mb-1">Distribuição CPU (%)</p>
              <ResponsiveContainer width="100%" height={100}>
                <BarChart data={cpuHistogramData}>
                  <XAxis dataKey="label" fontSize={8} interval={1} />
                  <YAxis fontSize={8} />
                  <Bar dataKey="count" fill="hsl(var(--chart-2))" radius={[2, 2, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Gráfico 2: Histograma Memória */}
            <div className="rounded-lg border p-2">
              <p className="text-[10px] text-muted-foreground mb-1">Distribuição Memória</p>
              <ResponsiveContainer width="100%" height={100}>
                <BarChart data={memHistogramData}>
                  <XAxis dataKey="label" fontSize={8} interval={2} />
                  <YAxis fontSize={8} />
                  <Bar dataKey="count" fill="hsl(var(--chart-3))" radius={[2, 2, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Gráfico 3: P50/P95/P99 CPU */}
            <div className="rounded-lg border p-2">
              <p className="text-[10px] text-muted-foreground mb-1">P50 / P95 / P99 CPU (%)</p>
              <ResponsiveContainer width="100%" height={100}>
                <BarChart data={cpuPercentileData}>
                  <XAxis dataKey="label" fontSize={9} />
                  <YAxis fontSize={8} />
                  <Bar dataKey="value" radius={[2, 2, 0, 0]}>
                    {cpuPercentileData.map((entry, i) => (
                      <Cell key={i} fill={entry.fill} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Gráfico 4: Timeline Memória + Forecast */}
            <div className="rounded-lg border p-2">
              <p className="text-[10px] text-muted-foreground mb-1">
                Memória P99 + Forecast {forecast.days_ahead}d
              </p>
              <ResponsiveContainer width="100%" height={100}>
                <AreaChart data={memForecastData}>
                  <defs>
                    <linearGradient id="memGrad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="hsl(var(--chart-3))" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="hsl(var(--chart-3))" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <XAxis dataKey="label" fontSize={9} />
                  <YAxis fontSize={8} tickFormatter={(v) => formatBytes(v)} />
                  <Area
                    dataKey="usage"
                    stroke="hsl(var(--chart-3))"
                    fill="url(#memGrad)"
                    connectNulls
                  />
                  <Line
                    dataKey="forecast"
                    stroke="hsl(var(--chart-5))"
                    strokeDasharray="4 4"
                    dot={false}
                    connectNulls
                  />
                  {suggestedMemLimit > 0 && (
                    <ReferenceLine
                      y={suggestedMemLimit}
                      stroke="hsl(var(--success))"
                      strokeDasharray="2 2"
                      label={{ value: "lim", fontSize: 8, position: "right" }}
                    />
                  )}
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>
        )}

        {/* Leak warning */}
        {memoryTrend.has_leak && (
          <div className="flex items-center gap-1.5 text-[10px] text-orange-700 bg-orange-500/10 rounded px-2 py-1">
            <span>
              Leak detectado: +{memoryTrend.daily_growth_mb} MB/dia (R²={memoryTrend.r_squared}).
              Forecast {forecast.days_ahead}d: {formatBytes(forecast.projected_mem_p99)}.
            </span>
          </div>
        )}
      </CollapsibleContent>
    </Collapsible>
  )
}
