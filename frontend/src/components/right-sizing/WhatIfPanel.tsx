/**
 * WhatIfPanel — painel que atualiza em tempo real quando o usuário
 * arrasta o ResourceSlider. Mostra o impacto da configuração what-if.
 *
 * Spec: react-components.md §7 + spec.md §4.2 (recalcWhatIf)
 * 4 métricas em grid: Recursos Liberados, Risco, Margem, Forecast.
 *
 * shadcn: Card, CardContent, Badge
 * Reusa: HelpIcon
 */
import { Card, CardContent } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { HelpIcon } from "@/components/help-icon"
import { TrendingDown, TrendingUp, Shield, Gauge, Activity } from "lucide-react"
import { formatBytes, cn } from "@/lib/utils"

interface WhatIfPanelProps {
  service: string
  whatIfCpu: number
  whatIfMem: number
  currentCpu: number
  currentMem: number
  p95Cpu: number
  p99Mem: number
  forecastP99: number
  oomEvents: number
  hasLeak: boolean
}

function marginColor(pct: number): string {
  if (pct >= 20) return "text-green-600"
  if (pct >= 10) return "text-yellow-600"
  return "text-red-600"
}

function riskLabel(score: number): { label: string; color: string } {
  if (score >= 80) return { label: "Crítico", color: "bg-red-500/10 text-red-700 border-red-500/30" }
  if (score >= 60) return { label: "Alto", color: "bg-orange-500/10 text-orange-700 border-orange-500/30" }
  if (score >= 40) return { label: "Atenção", color: "bg-yellow-500/10 text-yellow-700 border-yellow-500/30" }
  if (score >= 20) return { label: "Baixo", color: "bg-green-500/10 text-green-700 border-green-500/30" }
  return { label: "Muito baixo", color: "bg-green-500/10 text-green-700 border-green-500/30" }
}

function scoreRisk(cpuMargin: number, memMargin: number, oomEvents: number, hasLeak: boolean, forecastVsLimit: number): number {
  let score = 0
  // Margem CPU
  if (cpuMargin < 1.1) score += 40
  else if (cpuMargin < 1.3) score += 20
  else if (cpuMargin < 1.5) score += 5
  // Margem Mem
  if (memMargin < 1.1) score += 40
  else if (memMargin < 1.3) score += 20
  else if (memMargin < 1.5) score += 5
  // OOM
  if (oomEvents > 0) score += 15
  // Leak
  if (hasLeak) score += 15
  // Forecast vs limite
  if (forecastVsLimit > 0.9) score += 20
  else if (forecastVsLimit > 0.75) score += 10
  return Math.min(score, 100)
}

export function WhatIfPanel({
  service: _service,
  whatIfCpu, whatIfMem,
  currentCpu, currentMem,
  p95Cpu, p99Mem,
  forecastP99,
  oomEvents, hasLeak,
}: WhatIfPanelProps) {
  // Cálculos what-if (spec.md §4.2 — recalcWhatIf client-side)
  const cpuFreed = currentCpu - whatIfCpu
  const memFreed = currentMem - whatIfMem
  const cpuMargin = p95Cpu > 0 ? whatIfCpu / (p95Cpu / 100) : 99
  const memMargin = p99Mem > 0 ? whatIfMem / p99Mem : 99
  const forecastVsLimit = whatIfMem > 0 ? forecastP99 / whatIfMem : 0
  const riskScore = scoreRisk(cpuMargin, memMargin, oomEvents, hasLeak, forecastVsLimit)
  const risk = riskLabel(riskScore)
  const cpuMarginPct = whatIfCpu > 0 ? ((whatIfCpu - p95Cpu / 100) / whatIfCpu) * 100 : 0
  const memMarginPct = whatIfMem > 0 ? ((whatIfMem - p99Mem) / whatIfMem) * 100 : 0

  return (
    <Card className="border-dashed">
      <CardContent className="grid grid-cols-2 gap-3 p-3">
        {/* 1. Recursos Liberados */}
        <div className="space-y-1">
          <div className="flex items-center gap-1.5">
            {cpuFreed > 0 || memFreed > 0 ? (
              <TrendingDown className="h-3.5 w-3.5 text-green-600" />
            ) : cpuFreed < 0 || memFreed < 0 ? (
              <TrendingUp className="h-3.5 w-3.5 text-red-600" />
            ) : (
              <Activity className="h-3.5 w-3.5 text-muted-foreground" />
            )}
            <span className="text-xs font-medium text-muted-foreground">Libera</span>
            <HelpIcon text="Delta = atual - what-if. Verde: libera recursos. Vermelho: precisa mais recursos." className="h-3 w-3 text-muted-foreground" />
          </div>
          <div className="space-y-0.5">
            <p className={cn("text-sm font-bold tabular-nums", cpuFreed > 0 ? "text-green-600" : cpuFreed < 0 ? "text-red-600" : "text-muted-foreground")}>
              {cpuFreed > 0 ? "+" : ""}{cpuFreed.toFixed(2)} cores
            </p>
            <p className={cn("text-sm font-bold tabular-nums", memFreed > 0 ? "text-green-600" : memFreed < 0 ? "text-red-600" : "text-muted-foreground")}>
              {memFreed > 0 ? "+" : ""}{formatBytes(Math.abs(memFreed))}
            </p>
          </div>
        </div>

        {/* 2. Risco */}
        <div className="space-y-1">
          <div className="flex items-center gap-1.5">
            <Shield className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-xs font-medium text-muted-foreground">Risco</span>
            <HelpIcon text="Score 0-100 baseado em margem CPU/mem, OOMs, leak e forecast vs limite." className="h-3 w-3 text-muted-foreground" />
          </div>
          <Badge variant="outline" className={cn("text-xs", risk.color)}>
            {risk.label} ({riskScore})
          </Badge>
        </div>

        {/* 3. Margem de Segurança */}
        <div className="space-y-1">
          <div className="flex items-center gap-1.5">
            <Gauge className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-xs font-medium text-muted-foreground">Margem</span>
            <HelpIcon text="(what-if - P95/P99) / what-if * 100. Verde > 20%, amarelo 10-20%, vermelho < 10%." className="h-3 w-3 text-muted-foreground" />
          </div>
          <div className="space-y-0.5">
            <p className={cn("text-sm font-bold tabular-nums", marginColor(cpuMarginPct))}>
              CPU: {cpuMarginPct.toFixed(0)}%
            </p>
            <p className={cn("text-sm font-bold tabular-nums", marginColor(memMarginPct))}>
              Mem: {memMarginPct.toFixed(0)}%
            </p>
          </div>
        </div>

        {/* 4. Forecast vs limite */}
        <div className="space-y-1">
          <div className="flex items-center gap-1.5">
            <Activity className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-xs font-medium text-muted-foreground">Forecast 7d</span>
            <HelpIcon text="Projeção P99 de memória em 7 dias vs novo limite. < 75% = seguro, > 90% = risco." className="h-3 w-3 text-muted-foreground" />
          </div>
          <div className="space-y-0.5">
            <p className={cn("text-sm font-bold tabular-nums", forecastVsLimit > 0.9 ? "text-red-600" : forecastVsLimit > 0.75 ? "text-yellow-600" : "text-green-600")}>
              {(forecastVsLimit * 100).toFixed(0)}% do limite
            </p>
            <p className="text-[10px] text-muted-foreground tabular-nums">
              {formatBytes(forecastP99)} projetado
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
