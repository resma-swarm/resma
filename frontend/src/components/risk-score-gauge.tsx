import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

export interface RiskFactors {
  oom_count: number
  has_leak: boolean
  has_drift: boolean
  mem_limit: number
}

function getRiskLabel(score: number): string {
  if (score <= 25) return "Saudável"
  if (score <= 50) return "Atenção"
  if (score <= 75) return "Risco"
  return "Crítico"
}

function getRiskColor(score: number): string {
  if (score <= 25) return "#22c55e"
  if (score <= 50) return "#f59e0b"
  if (score <= 75) return "#f97316"
  return "#ef4444"
}

// Gera dicas de melhoria baseadas nos fatores de risco presentes
function getRiskTips(factors?: RiskFactors): string[] {
  if (!factors) return []
  const tips: string[] = []
  if (factors.mem_limit === 0) {
    tips.push("Definir limite de memória (--mem-limit) para evitar OOM e consumo ilimitado")
  }
  if (factors.oom_count > 0) {
    tips.push(`Aumentar mem_limit: ${factors.oom_count} OOM${factors.oom_count > 1 ? "s" : ""} recente${factors.oom_count > 1 ? "s" : ""} — serviço foi morto pelo kernel`)
  }
  if (factors.has_leak) {
    tips.push("Investigar memory leak: aplicar recomendação de limite ou corrigir vazamento no código")
  }
  if (factors.has_drift) {
    tips.push("Config drift detectado: config atual não bate com o sugerido pelo ML — reanalisar")
  }
  if (tips.length === 0) {
    tips.push("Serviço saudável — manter config atual e monitorar tendência")
  }
  return tips
}

interface RiskScoreGaugeProps {
  score: number
  factors?: RiskFactors
  /** Orientação do layout: "horizontal" (barra + número inline) ou
   *  "vertical" (barra em cima, número embaixo). Default: "horizontal". */
  orientation?: "horizontal" | "vertical"
  /** Adiciona borda estilo Badge (rounded-md border px-2.5 py-0.5).
   *  Útil para headers onde o gauge precisa ter aparência de badge.
   *  Default: false. */
  border?: boolean
}

// RiskScoreGauge — barra gradiente (verde→vermelho) com marcador do score.
// Inspirado no Bar Gauge do Grafana (display mode: Gradient).
// Compacto o suficiente para célula de tabela, mas visualmente intuitivo:
// o usuário vê instantaneamente onde o score está no espectro de risco.
//
// Score calculado no backend (buildServicesList) por fórmula determinística
// baseada em: memory pressure (40%), OOM history (30%), leak (20%), drift (10%).
export function RiskScoreGauge({
  score,
  factors,
  orientation = "horizontal",
  border = false,
}: RiskScoreGaugeProps) {
  const label = getRiskLabel(score)
  const color = getRiskColor(score)
  const tips = getRiskTips(factors)
  const markerPos = `${Math.min(Math.max(score, 0), 100)}%`

  // Fatores ativos para exibir no tooltip
  const activeFactors: string[] = []
  if (factors) {
    if (factors.oom_count > 0) activeFactors.push(`OOM: ${factors.oom_count}`)
    if (factors.has_leak) activeFactors.push("Memory leak")
    if (factors.has_drift) activeFactors.push("Config drift")
    if (factors.mem_limit === 0) activeFactors.push("Sem limite de memória")
  }

  // Container: orientação + borda opcional (mesmo estilo do Badge outline)
  // py-1.5 (6px) + barra h-1.5 (6px) = ~18px conteúdo + 12px padding = 30px
  // para equiparar com StatusBadge (text-xs + py-0.5 = ~22px) mantendo alinhado.
  const orientCls = orientation === "vertical"
    ? "inline-flex flex-col items-center gap-0.5 cursor-help py-0.5"
    : "inline-flex items-center gap-1.5 cursor-help"
  const borderCls = border
    ? "rounded-md border px-2.5 py-1"
    : ""

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className={`${orientCls} ${borderCls}`}>
            {/* Barra gradiente com marcador */}
            <div className="relative h-1.5 w-20 rounded-full overflow-visible"
                 style={{ background: "linear-gradient(to right, #22c55e 0%, #22c55e 25%, #f59e0b 25%, #f59e0b 50%, #f97316 50%, #f97316 75%, #ef4444 75%, #ef4444 100%)" }}>
              {/* Marcador vertical — linha branca com borda escura para contraste */}
              <div className="absolute top-1/2 -translate-y-1/2 -translate-x-1/2"
                   style={{ left: markerPos }}>
                <div className="h-3.5 w-0.5 rounded-full bg-white shadow-sm ring-1 ring-black/20" />
              </div>
            </div>
            {/* Score numérico */}
            <span className="text-xs font-semibold tabular-nums leading-none" style={{ color }}>{score}</span>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top" className="text-xs max-w-xs">
          <div className="font-semibold" style={{ color }}>{label} — {score}/100</div>
          {activeFactors.length > 0 && (
            <div className="mt-1 text-muted-foreground">
              <span className="font-medium">Fatores:</span> {activeFactors.join(" • ")}
            </div>
          )}
          <div className="mt-1.5 pt-1.5 border-t border-border">
            <span className="font-medium">Como melhorar:</span>
            <ul className="mt-0.5 space-y-0.5 text-muted-foreground">
              {tips.map((tip, i) => (
                <li key={i} className="flex gap-1">
                  <span className="text-muted-foreground/50">→</span>
                  <span>{tip}</span>
                </li>
              ))}
            </ul>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
