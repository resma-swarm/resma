/**
 * Studio — Right-Sizing Studio (rota /studio).
 *
 * Página principal do Right-Sizing Studio seguindo o wireframe (mockup.html).
 * Layout: Hero North Star → Filter chips → Cards compactos (expand/collapse).
 * Card expandido: LayerToggle + ResourceSlider + WhatIfPanel + ExplainabilityPanel + botões.
 * Modais: Aplicar com Rollback + Simulação em Lote.
 * Botão no header: Rollback Watches (link para /rollback-watches).
 *
 * Segue padrões de componentes do RESMA: PageHeader, Card, Badge, Dialog, Button.
 */
import { useState, useMemo, useRef } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link } from "react-router-dom"
import { api } from "@/api/client"
import { toast } from "sonner"
import {
  Card, CardContent, CardHeader,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "@/components/ui/dialog"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Collapsible, CollapsibleContent, CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { PageHeader } from "@/components/page-header"
import { HelpIcon } from "@/components/help-icon"
import { HeroMetric } from "@/components/right-sizing/HeroMetric"
import { LayerToggle } from "@/components/right-sizing/LayerToggle"
import { ResourceSlider } from "@/components/right-sizing/ResourceSlider"
import { WhatIfPanel } from "@/components/right-sizing/WhatIfPanel"
import { ExplainabilityPanel } from "@/components/right-sizing/ExplainabilityPanel"
import { ExportYamlButton } from "@/components/right-sizing/ExportYamlButton"
import { formatBytes, formatCPU } from "@/lib/utils"
import {
  calculateHero,
  patternLabel, riskColorClasses, riskLevelLabel,
  type Recommendation, type ResourceValues, type TierName,
} from "@/components/right-sizing/types"
import type { RiskColor } from "@/components/right-sizing/types"
import {
  ChevronRight, RotateCcw, Download, CalendarClock, Layers,
  ShieldCheck, AlertTriangle, CheckCircle2, Settings2, TrendingDown,
  Cpu, MemoryStick, Zap, Activity, Filter, X,
} from "lucide-react"

// --- status config ---

interface StatusCfg {
  label: string
  icon: typeof AlertTriangle
  variant: "default" | "secondary" | "destructive" | "outline" | "success" | "warning" | "danger"
}

const statusConfig: Record<string, StatusCfg> = {
  over_provisioned: { label: "Over-provisioned", icon: TrendingDown, variant: "secondary" },
  under_provisioned: { label: "Atenção", icon: AlertTriangle, variant: "warning" },
  healthy: { label: "Saudável", icon: CheckCircle2, variant: "success" },
  alerted: { label: "Crítico", icon: AlertTriangle, variant: "danger" },
  unconfigured: { label: "Sem config", icon: Settings2, variant: "outline" },
}

const patternIcons: Record<string, string> = {
  constant: "Web",
  business_hours: "Horário",
  batch: "Batch",
}

export default function Studio() {
  const queryClient = useQueryClient()
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [patternFilter, setPatternFilter] = useState<string>("all")
  const [windowFilter, setWindowFilter] = useState<string>("30")
  const [confidenceFilter, setConfidenceFilter] = useState<string>("medium")
  const [expandedCard, setExpandedCard] = useState<string | null>(null)
  const [selectedTier, setSelectedTier] = useState<TierName>("balanced")
  const [whatIfCpu, setWhatIfCpu] = useState(0)
  const [whatIfMem, setWhatIfMem] = useState(0)
  const [applying, setApplying] = useState<string | null>(null)

  // Modais
  const [applyModalService, setApplyModalService] = useState<Recommendation | null>(null)
  const [bulkModalOpen, setBulkModalOpen] = useState(false)

  // Apply com Rollback — form state
  const [applyStrategy, setApplyStrategy] = useState<"immediate" | "deferred" | "canary">("immediate")
  const [applyWindow, setApplyWindow] = useState("24")
  const [criteriaOOM, setCriteriaOOM] = useState(true)
  const [criteriaThrottle, setCriteriaThrottle] = useState(true)
  const [criteriaMemPressure, setCriteriaMemPressure] = useState(true)

  const { data: recs, isLoading } = useQuery<Recommendation[]>({
    queryKey: ["recommendations"],
    queryFn: () => api.get<Recommendation[]>("/recommendations"),
  })

  // --- counts por status ---
  const statusCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const r of recs ?? []) {
      counts[r.status] = (counts[r.status] ?? 0) + 1
    }
    return counts
  }, [recs])

  const patternCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const r of recs ?? []) {
      const p = r.pattern ?? "unknown"
      counts[p] = (counts[p] ?? 0) + 1
    }
    return counts
  }, [recs])

  // --- filtros ---
  const filteredRecs = useMemo(() => {
    if (!recs) return []
    return recs.filter((r) => {
      if (statusFilter !== "all" && r.status !== statusFilter) return false
      if (patternFilter !== "all" && (r.pattern ?? "unknown") !== patternFilter) return false
      if (confidenceFilter === "high" && r.confidence !== "high") return false
      if (confidenceFilter === "medium" && r.confidence === "low") return false
      return true
    })
  }, [recs, statusFilter, patternFilter, confidenceFilter])

  // --- apply mutation ---
  const applyMutation = useMutation({
    mutationFn: ({ service, values }: { service: string; values: ResourceValues }) =>
      api.post(`/recommendations/${service}/apply`, values),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ["recommendations"] })
      setApplying(null)
      toast.success(`Recomendação aplicada para ${vars.service} com sucesso`)
    },
    onError: (_error, vars) => {
      setApplying(null)
      toast.error(`Erro ao aplicar recomendação para ${vars.service}`)
    },
  })

  const handleApply = (service: string, values: ResourceValues) => {
    setApplying(service)
    applyMutation.mutate({ service, values })
  }

  // --- helpers ---
  const getTierData = (rec: Recommendation): ResourceValues | null => {
    if (!rec.suggested_tiers) return rec.suggested ?? null
    const tier = rec.suggested_tiers[selectedTier as TierName]
    if (!tier) return rec.suggested ?? null
    return {
      cpu_limit: tier.cpu_limit,
      mem_limit: tier.mem_limit,
      cpu_reservation: tier.cpu_reservation,
      mem_reservation: tier.mem_reservation,
    }
  }

  const getResourcesFreed = (rec: Recommendation) => {
    if (!rec.suggested_tiers) return rec.resources_freed?.balanced ?? null
    const tier = rec.suggested_tiers[selectedTier as TierName]
    return tier?.resources_freed ?? null
  }

  const toggleCard = (service: string) => {
    if (expandedCard === service) {
      setExpandedCard(null)
    } else {
      setExpandedCard(service)
      // Inicializar what-if com valores suggested do tier atual
      const rec = recs?.find((r) => r.service === service)
      if (rec) {
        const tier = getTierData(rec)
        if (tier) {
          setWhatIfCpu(tier.cpu_limit)
          setWhatIfMem(tier.mem_limit)
        }
      }
    }
  }

  // --- loading ---
  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-12 w-full" />
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      </div>
    )
  }

  if (!recs || recs.length === 0) {
    return (
      <div className="space-y-6">
        <PageHeader title="Right-Sizing Studio" description="Sugestões de limites baseadas em dados" />
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <Layers className="h-10 w-10 text-muted-foreground/50" />
            <p className="mt-3 text-sm text-muted-foreground">
              Sem recomendações ainda. Aguarde a coleta de métricas ou recalcule.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <PageHeader title="Right-Sizing Studio" description="Sugestões de limites baseadas em dados">
        <div className="flex gap-2 flex-wrap items-center">
          <Button variant="outline" size="sm" onClick={() => setBulkModalOpen(true)}>
            <Layers className="mr-2 h-4 w-4" />
            Simulação em Lote
          </Button>
          <Button variant="outline" size="sm" onClick={() => {
            queryClient.invalidateQueries({ queryKey: ["recommendations"] })
            toast.info("Recalculando recomendações...")
          }}>
            <Activity className="mr-2 h-4 w-4" />
            Recalcular
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to="/rollback-watches">
              <RotateCcw className="mr-2 h-4 w-4" />
              Rollback Watches
            </Link>
          </Button>
        </div>
      </PageHeader>

      {/* Hero North Star */}
      <HeroMetric data={calculateHero(recs)} loading={isLoading} />

      {/* Filter chips */}
      <Card>
        <CardContent className="flex gap-2 flex-wrap items-center p-3">
          {/* Status chips */}
          <FilterChip
            active={statusFilter === "all"}
            onClick={() => setStatusFilter("all")}
            label="Todos"
            count={recs.length}
          />
          {Object.entries(statusConfig).map(([key, cfg]) => (
            <FilterChip
              key={key}
              active={statusFilter === key}
              onClick={() => setStatusFilter(key)}
              label={cfg.label}
              count={statusCounts[key] ?? 0}
            />
          ))}

          <Separator orientation="vertical" className="h-5 mx-1" />

          {/* Pattern chips */}
          {Object.entries(patternCounts).map(([pat, count]) => (
            <FilterChip
              key={pat}
              active={patternFilter === pat}
              onClick={() => setPatternFilter(patternFilter === pat ? "all" : pat)}
              label={patternLabel(pat)}
              count={count}
            />
          ))}

          <Separator orientation="vertical" className="h-5 mx-1" />

          {/* Janela */}
          <div className="flex items-center gap-1.5">
            <span className="text-xs text-muted-foreground">Janela:</span>
            <Select value={windowFilter} onValueChange={setWindowFilter}>
              <SelectTrigger className="h-7 w-24 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="7">7 dias</SelectItem>
                <SelectItem value="30">30 dias</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Confiança */}
          <div className="flex items-center gap-1.5">
            <span className="text-xs text-muted-foreground">Confiança:</span>
            <Select value={confidenceFilter} onValueChange={setConfidenceFilter}>
              <SelectTrigger className="h-7 w-28 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="low">≥ Baixa</SelectItem>
                <SelectItem value="medium">≥ Média</SelectItem>
                <SelectItem value="high">≥ Alta</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {(statusFilter !== "all" || patternFilter !== "all") && (
            <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => {
              setStatusFilter("all")
              setPatternFilter("all")
            }}>
              <X className="mr-1 h-3 w-3" />
              Limpar
            </Button>
          )}
        </CardContent>
      </Card>

      {/* Cards */}
      <div className="space-y-3">
        {filteredRecs.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center text-sm text-muted-foreground">
              Nenhum serviço corresponde aos filtros selecionados.
            </CardContent>
          </Card>
        ) : (
          filteredRecs.map((rec) => (
            <StudioCard
              key={rec.service}
              rec={rec}
              expanded={expandedCard === rec.service}
              onToggle={() => toggleCard(rec.service)}
              selectedTier={selectedTier}
              onTierChange={setSelectedTier}
              whatIfCpu={whatIfCpu}
              whatIfMem={whatIfMem}
              onCpuChange={setWhatIfCpu}
              onMemChange={setWhatIfMem}
              getTierData={getTierData}
              getResourcesFreed={getResourcesFreed}
              onApply={() => setApplyModalService(rec)}
              applying={applying === rec.service}
            />
          ))
        )}
      </div>

      {/* Modal: Aplicar com Rollback */}
      <ApplyRollbackModal
        rec={applyModalService}
        open={!!applyModalService}
        onOpenChange={(open) => !open && setApplyModalService(null)}
        strategy={applyStrategy}
        onStrategyChange={setApplyStrategy}
        window={applyWindow}
        onWindowChange={setApplyWindow}
        criteriaOOM={criteriaOOM}
        onCriteriaOOMChange={setCriteriaOOM}
        criteriaThrottle={criteriaThrottle}
        onCriteriaThrottleChange={setCriteriaThrottle}
        criteriaMemPressure={criteriaMemPressure}
        onCriteriaMemPressureChange={setCriteriaMemPressure}
        tierData={applyModalService ? getTierData(applyModalService) : null}
        onConfirm={() => {
          if (!applyModalService || !getTierData(applyModalService)) return
          const values = getTierData(applyModalService)!
          handleApply(applyModalService.service, values)
          setApplyModalService(null)
        }}
        applying={!!applying}
      />

      {/* Modal: Simulação em Lote */}
      <BulkSimulateModal
        open={bulkModalOpen}
        onOpenChange={setBulkModalOpen}
        recs={filteredRecs}
        selectedTier={selectedTier}
        getTierData={getTierData}
        getResourcesFreed={getResourcesFreed}
      />
    </div>
  )
}

// --- Filter Chip ---

function FilterChip({ active, onClick, label, count }: {
  active: boolean
  onClick: () => void
  label: string
  count: number
}) {
  return (
    <button
      onClick={onClick}
      className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs transition-colors ${
        active
          ? "border-primary bg-primary text-primary-foreground"
          : "border-border bg-secondary text-muted-foreground hover:border-primary"
      }`}
    >
      {label}
      <span className={`tabular-nums ${active ? "opacity-80" : "opacity-60"}`}>{count}</span>
    </button>
  )
}

// --- Studio Card ---

interface StudioCardProps {
  rec: Recommendation
  expanded: boolean
  onToggle: () => void
  selectedTier: TierName
  onTierChange: (tier: TierName) => void
  whatIfCpu: number
  whatIfMem: number
  onCpuChange: (cpu: number) => void
  onMemChange: (mem: number) => void
  getTierData: (rec: Recommendation) => ResourceValues | null
  getResourcesFreed: (rec: Recommendation) => { cpu_cores: number; mem_bytes: number; cpu_pct: number; mem_pct: number } | null
  onApply: () => void
  applying: boolean
}

function StudioCard({
  rec, expanded, onToggle, selectedTier, onTierChange,
  whatIfCpu, whatIfMem, onCpuChange, onMemChange,
  getTierData, getResourcesFreed, onApply, applying,
}: StudioCardProps) {
  const cfg = statusConfig[rec.status] ?? statusConfig.over_provisioned
  const StatusIcon = cfg.icon
  const freed = getResourcesFreed(rec)
  const tierData = getTierData(rec)
  const hasLeak = rec.memory_trend?.has_leak ?? false

  return (
    <Card className={expanded ? "border-primary" : ""}>
      {/* Card header (compacto) */}
      <div
        className="flex items-center gap-3 p-3.5 cursor-pointer hover:bg-accent/50 transition-colors"
        onClick={onToggle}
      >
        <StatusIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
        <span className="font-medium text-sm">{rec.service}</span>
        <Badge variant="outline" className="text-[10px] py-0">
          {patternLabel(rec.pattern)}
        </Badge>
        <span className="text-xs text-muted-foreground hidden sm:inline">
          P95 CPU {rec.cpu ? formatCPU(rec.cpu.p95) : "—"} · P99 Mem {rec.mem ? formatBytes(rec.mem.p99) : "—"}
        </span>
        <div className="flex-1" />
        <Badge variant={cfg.variant} className="text-[10px]">{cfg.label}</Badge>
        {rec.confidence && (
          <Badge variant="outline" className="text-[10px] py-0">
            Confiança: {rec.confidence === "high" ? "Alta" : rec.confidence === "medium" ? "Média" : "Baixa"}
          </Badge>
        )}
        {freed && (freed.cpu_cores > 0 || freed.mem_bytes > 0) ? (
          <span className="text-sm font-semibold text-success tabular-nums">
            {freed.cpu_cores > 0 && `${formatCPU(freed.cpu_cores)} cores`}
            {freed.cpu_cores > 0 && freed.mem_bytes > 0 && " · "}
            {freed.mem_bytes > 0 && formatBytes(freed.mem_bytes)}
          </span>
        ) : hasLeak ? (
          <span className="text-sm font-semibold text-warning tabular-nums">
            +{formatBytes(Math.abs(rec.memory_trend?.daily_growth_mb ?? 0) * 1e6)}/dia
          </span>
        ) : (
          <span className="text-sm text-muted-foreground">0 liberado</span>
        )}
        <ChevronRight className={`h-4 w-4 text-muted-foreground transition-transform ${expanded ? "rotate-90" : ""}`} />
      </div>

      {/* Card body (expandido) */}
      {expanded && rec.suggested_tiers && (
        <CardContent className="pt-0 pb-4 px-4 space-y-4">
          <Separator />

          {/* Layer Toggle */}
          <LayerToggle
            value={selectedTier}
            onChange={onTierChange}
            suggestedTiers={rec.suggested_tiers}
          />

          {/* Sliders */}
          <ResourceSlider
            cpuCores={whatIfCpu}
            memBytes={whatIfMem}
            cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.8}
            cpuMax={(rec.current?.cpu_limit ?? 4) * 1.5}
            memMin={(rec.mem?.p99 ?? 16e6) * 0.8}
            memMax={(rec.current?.mem_limit ?? 8e9) * 1.5}
            cpuCurrent={rec.current?.cpu_limit ?? 0}
            memCurrent={rec.current?.mem_limit ?? 0}
            cpuSuggested={tierData?.cpu_limit ?? 0}
            memSuggested={tierData?.mem_limit ?? 0}
            onCpuChange={onCpuChange}
            onMemChange={onMemChange}
          />

          {/* What-If Panel */}
          <WhatIfPanel
            service={rec.service}
            whatIfCpu={whatIfCpu}
            whatIfMem={whatIfMem}
            currentCpu={rec.current?.cpu_limit ?? 0}
            currentMem={rec.current?.mem_limit ?? 0}
            p95Cpu={rec.cpu?.p95 ?? 0}
            p99Mem={rec.mem?.p99 ?? 0}
            forecastP99={rec.forecast?.projected_mem_p99 ?? 0}
            oomEvents={rec.oom_events ?? 0}
            hasLeak={hasLeak}
          />

          {/* Explainability */}
          {rec.explainability && (
            <ExplainabilityPanel
              explainability={rec.explainability}
              histograms={rec.histograms ?? null}
              cpuStats={rec.cpu!}
              memStats={rec.mem!}
              memoryTrend={rec.memory_trend!}
              forecast={rec.forecast!}
              suggestedMemLimit={tierData?.mem_limit ?? 0}
              selectedTier={selectedTier}
            />
          )}

          {/* Action buttons */}
          <div className="flex gap-2 flex-wrap pt-2">
            <Button size="sm" onClick={onApply} disabled={applying}>
              <ShieldCheck className="mr-2 h-4 w-4" />
              {applying ? "Aplicando..." : "Aplicar com Rollback"}
            </Button>
            <Button size="sm" variant="outline">
              <CalendarClock className="mr-2 h-4 w-4" />
              Agendar
            </Button>
            <ExportYamlButton services={[rec.service]} tier={selectedTier} />
          </div>
        </CardContent>
      )}
    </Card>
  )
}

// --- Modal: Aplicar com Rollback ---

interface ApplyRollbackModalProps {
  rec: Recommendation | null
  open: boolean
  onOpenChange: (open: boolean) => void
  strategy: "immediate" | "deferred" | "canary"
  onStrategyChange: (s: "immediate" | "deferred" | "canary") => void
  window: string
  onWindowChange: (w: string) => void
  criteriaOOM: boolean
  onCriteriaOOMChange: (v: boolean) => void
  criteriaThrottle: boolean
  onCriteriaThrottleChange: (v: boolean) => void
  criteriaMemPressure: boolean
  onCriteriaMemPressureChange: (v: boolean) => void
  tierData: ResourceValues | null
  onConfirm: () => void
  applying: boolean
}

function ApplyRollbackModal({
  rec, open, onOpenChange, strategy, onStrategyChange, window, onWindowChange,
  criteriaOOM, onCriteriaOOMChange, criteriaThrottle, onCriteriaThrottleChange,
  criteriaMemPressure, onCriteriaMemPressureChange, tierData, onConfirm, applying,
}: ApplyRollbackModalProps) {
  if (!rec) return null

  const cpuDelta = rec.current?.cpu_limit && tierData
    ? ((rec.current.cpu_limit - tierData.cpu_limit) / rec.current.cpu_limit) * 100
    : 0
  const memDelta = rec.current?.mem_limit && tierData
    ? ((rec.current.mem_limit - tierData.mem_limit) / rec.current.mem_limit) * 100
    : 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Aplicar com Rollback Automático — {rec.service}</DialogTitle>
          <DialogDescription>
            Valores: CPU {formatCPU(rec.current?.cpu_limit ?? 0)} → {formatCPU(tierData?.cpu_limit ?? 0)}
            {" "}({cpuDelta > 0 ? "-" : "+"}{Math.abs(cpuDelta).toFixed(0)}%) · {" "}
            Mem {formatBytes(rec.current?.mem_limit ?? 0)} → {formatBytes(tierData?.mem_limit ?? 0)}
            {" "}({memDelta > 0 ? "-" : "+"}{Math.abs(memDelta).toFixed(0)}%)
          </DialogDescription>
        </DialogHeader>

        {/* Estratégia */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Estratégia de Apply</Label>
          <div className="space-y-2">
            <StrategyRadio
              selected={strategy === "immediate"}
              onClick={() => onStrategyChange("immediate")}
              title="Imediato"
              desc="1 serviço, rollback em 15min se problema"
            />
            <StrategyRadio
              selected={strategy === "deferred"}
              onClick={() => onStrategyChange("deferred")}
              title="Diferido"
              desc="Agenda para janela de manutenção"
            />
            <StrategyRadio
              selected={strategy === "canary"}
              onClick={() => onStrategyChange("canary")}
              title="Híbrido (canário)"
              desc="1 task primeiro → resto após 30min sem incidente"
            />
          </div>
        </div>

        {/* Critérios de Rollback */}
        <div className="space-y-2">
          <Label className="text-sm font-medium">Critérios de Rollback Automático</Label>
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <Checkbox checked={criteriaOOM} onCheckedChange={(v) => onCriteriaOOMChange(!!v)} />
              <Label className="text-xs cursor-pointer" onClick={() => onCriteriaOOMChange(!criteriaOOM)}>
                OOM kill em qualquer task
              </Label>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox checked={criteriaThrottle} onCheckedChange={(v) => onCriteriaThrottleChange(!!v)} />
              <Label className="text-xs cursor-pointer" onClick={() => onCriteriaThrottleChange(!criteriaThrottle)}>
                CPU throttling {'>'} 10% por {'>'} 5min
              </Label>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox checked={criteriaMemPressure} onCheckedChange={(v) => onCriteriaMemPressureChange(!!v)} />
              <Label className="text-xs cursor-pointer" onClick={() => onCriteriaMemPressureChange(!criteriaMemPressure)}>
                Memória {'>'} 95% do limite por {'>'} 5min
              </Label>
            </div>
          </div>
          <div className="flex items-center gap-2 pt-1">
            <span className="text-xs text-muted-foreground">Janela de observação:</span>
            <Select value={window} onValueChange={onWindowChange}>
              <SelectTrigger className="h-7 w-20 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="6">6h</SelectItem>
                <SelectItem value="24">24h</SelectItem>
                <SelectItem value="168">7d</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {/* Snapshot */}
        <div className="rounded-lg border bg-muted/50 p-3 text-xs">
          <div className="font-medium mb-1">Snapshot de Rollback</div>
          <div className="text-muted-foreground">
            Estado atual salvo: CPU {formatCPU(rec.current?.cpu_limit ?? 0)}, Mem {formatBytes(rec.current?.mem_limit ?? 0)}
            <br />
            Será revertido automaticamente se algum critério disparar.
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancelar</Button>
          <Button onClick={onConfirm} disabled={applying}>
            {applying ? "Aplicando..." : "Aplicar e monitorar →"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function StrategyRadio({ selected, onClick, title, desc }: {
  selected: boolean
  onClick: () => void
  title: string
  desc: string
}) {
  return (
    <div
      onClick={onClick}
      className={`flex items-center gap-3 rounded-lg border p-2.5 cursor-pointer transition-colors ${
        selected ? "border-primary bg-primary/5" : "border-border hover:bg-accent/50"
      }`}
    >
      <div className={`h-4 w-4 rounded-full border-2 shrink-0 ${
        selected ? "border-primary bg-primary" : "border-muted-foreground"
      }`}>
        {selected && <div className="h-1.5 w-1.5 rounded-full bg-primary-foreground m-auto mt-[3px]" />}
      </div>
      <div>
        <div className="text-sm font-medium">{title}</div>
        <div className="text-xs text-muted-foreground">{desc}</div>
      </div>
    </div>
  )
}

// --- Modal: Simulação em Lote ---

interface BulkSimulateModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  recs: Recommendation[]
  selectedTier: TierName
  getTierData: (rec: Recommendation) => ResourceValues | null
  getResourcesFreed: (rec: Recommendation) => { cpu_cores: number; mem_bytes: number; cpu_pct: number; mem_pct: number } | null
}

function BulkSimulateModal({
  open, onOpenChange, recs, selectedTier, getTierData, getResourcesFreed,
}: BulkSimulateModalProps) {
  const overProvRecs = recs.filter((r) => r.status === "over_provisioned" && r.suggested_tiers)

  const totals = overProvRecs.reduce((acc, r) => {
    const freed = getResourcesFreed(r)
    if (freed) {
      acc.cpu += freed.cpu_cores
      acc.mem += freed.mem_bytes
    }
    return acc
  }, { cpu: 0, mem: 0 })

  const riskCounts = overProvRecs.reduce((acc, r) => {
    const level = r.risk?.color ?? "green"
    acc[level] = (acc[level] ?? 0) + 1
    return acc
  }, {} as Record<string, number>)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Simulação em Lote — Right-Sizing Studio</DialogTitle>
          <DialogDescription>
            Cenário: {selectedTier === "conservative" ? "Conservadora" : selectedTier === "balanced" ? "Equilibrada" : "Agressiva"}
            {" · "}
            {overProvRecs.length} serviços selecionados
          </DialogDescription>
        </DialogHeader>

        {overProvRecs.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            Nenhum serviço over-provisioned para simular.
          </div>
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Serviço</TableHead>
                  <TableHead>CPU atual→sug</TableHead>
                  <TableHead>Mem atual→sug</TableHead>
                  <TableHead>Libera</TableHead>
                  <TableHead>Risco</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {overProvRecs.map((rec) => {
                  const tier = getTierData(rec)
                  const freed = getResourcesFreed(rec)
                  const riskColor = rec.risk?.color ?? "green"
                  return (
                    <TableRow key={rec.service}>
                      <TableCell className="font-medium text-sm">{rec.service}</TableCell>
                      <TableCell className="text-xs tabular-nums">
                        {formatCPU(rec.current?.cpu_limit ?? 0)} → {formatCPU(tier?.cpu_limit ?? 0)}
                      </TableCell>
                      <TableCell className="text-xs tabular-nums">
                        {formatBytes(rec.current?.mem_limit ?? 0)} → {formatBytes(tier?.mem_limit ?? 0)}
                      </TableCell>
                      <TableCell className="text-xs tabular-nums text-success font-medium">
                        {freed && (freed.cpu_cores > 0 || freed.mem_bytes > 0) ? (
                          <>
                            {freed.cpu_cores > 0 && `${formatCPU(freed.cpu_cores)} cores`}
                            {freed.cpu_cores > 0 && freed.mem_bytes > 0 && " · "}
                            {freed.mem_bytes > 0 && formatBytes(freed.mem_bytes)}
                          </>
                        ) : "—"}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={riskColorClasses[riskColor as RiskColor]}>
                          {riskLevelLabel[rec.risk?.level ?? "low"]}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  )
                })}
                <TableRow className="border-t-2 font-medium">
                  <TableCell>TOTAL</TableCell>
                  <TableCell colSpan={2} />
                  <TableCell className="text-success tabular-nums">
                    {formatCPU(totals.cpu)} cores · {formatBytes(totals.mem)}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {riskCounts.green ?? 0} safe, {riskCounts.yellow ?? 0} atenção
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>

            {/* Risk bar */}
            {overProvRecs.length > 0 && (
              <div className="flex gap-0.5 h-2 rounded-full overflow-hidden">
                {overProvRecs.map((rec) => {
                  const color = rec.risk?.color ?? "green"
                  const bg = color === "green" ? "bg-success" : color === "yellow" ? "bg-warning" : color === "orange" ? "bg-orange-500" : "bg-destructive"
                  return <div key={rec.service} className={`flex-1 ${bg}`} />
                })}
              </div>
            )}

            {/* Summary */}
            <div className="grid grid-cols-3 gap-3">
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">Recursos liberados</div>
                <div className="text-lg font-bold text-success tabular-nums">
                  {formatCPU(totals.cpu)} · {formatBytes(totals.mem)}
                </div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">Risco agregado</div>
                <div className="text-lg font-bold">
                  {(riskCounts.green ?? 0) > overProvRecs.length / 2 ? "Baixo" : "Atenção"}
                </div>
              </div>
              <div className="rounded-lg border p-3">
                <div className="text-xs text-muted-foreground">Maior risco</div>
                <div className="text-sm font-medium">
                  {overProvRecs.find((r) => r.risk?.color === "yellow" || r.risk?.color === "orange")?.service ?? "—"}
                </div>
              </div>
            </div>
          </>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancelar</Button>
          <ExportYamlButton
            services={overProvRecs.map((r) => r.service)}
            tier={selectedTier}
            disabled={overProvRecs.length === 0}
          />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
