/**
 * Studio — Right-Sizing Studio (rota /studio).
 *
 * v2: cards compactos + Sheet lateral para edição + HelpIcons.
 * Layout: Hero → Filter chips → Cards compactos (botão Configurar abre Sheet).
 * Sheet com 6 modos: over, healthy, leak, unconfigured, collecting, under.
 * Modal bulk mantém como Dialog.
 */
import { useState, useMemo, useEffect } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link } from "react-router-dom"
import { api } from "@/api/client"
import { useRefreshInterval } from "@/hooks/use-refresh"
import { toast } from "sonner"
import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Separator } from "@/components/ui/separator"
import {
  Sheet, SheetContent, SheetHeader, SheetTitle, SheetFooter,
} from "@/components/ui/sheet"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "@/components/ui/dialog"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { PageHeader } from "@/components/page-header"
import { HelpIcon } from "@/components/help-icon"
import { HeroMetric } from "@/components/right-sizing/HeroMetric"
import { LayerToggle } from "@/components/right-sizing/LayerToggle"
import { ResourceSlider } from "@/components/right-sizing/ResourceSlider"
import { WhatIfPanel } from "@/components/right-sizing/WhatIfPanel"
import { ExplainabilityPanel } from "@/components/right-sizing/ExplainabilityPanel"
import { ExportYamlButton } from "@/components/right-sizing/ExportYamlButton"
import { formatBytes, formatCPU, cn } from "@/lib/utils"
import { Calendar } from "@/components/ui/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Input } from "@/components/ui/input"
import { format, parseISO } from "date-fns"
import {
  calculateHero,
  patternLabel, riskColorClasses, riskLevelLabel,
  type Recommendation, type ResourceValues, type TierName,
} from "@/components/right-sizing/types"
import type { RiskColor } from "@/components/right-sizing/types"
import {
  RotateCcw, Layers, Activity, Settings2, AlertTriangle,
  CheckCircle2, TrendingDown, X, ShieldCheck, CalendarClock,
  Database, HardDrive, TrendingUp, ChevronDown, AlertCircle,
  Calendar as CalendarIcon,
} from "lucide-react"

// --- status config ---

interface StatusCfg {
  label: string
  icon: typeof AlertTriangle
  variant: "default" | "secondary" | "destructive" | "outline" | "success" | "warning" | "danger"
  borderClass: string
  priority: number
}

const statusConfig: Record<string, StatusCfg> = {
  alerted: { label: "Crítico", icon: AlertTriangle, variant: "danger", borderClass: "border-l-destructive", priority: 0 },
  under_provisioned: { label: "Insuficiente", icon: AlertTriangle, variant: "danger", borderClass: "border-l-warning", priority: 1 },
  over_provisioned: { label: "Excesso", icon: TrendingDown, variant: "warning", borderClass: "border-l-primary", priority: 2 },
  unconfigured: { label: "Sem config", icon: Settings2, variant: "warning", borderClass: "border-l-warning", priority: 3 },
  collecting_data: { label: "Coletando", icon: Activity, variant: "secondary", borderClass: "border-l-muted-foreground", priority: 4 },
  healthy: { label: "Saudável", icon: CheckCircle2, variant: "success", borderClass: "border-l-success", priority: 5 },
}

const confidenceConfig: Record<string, { label: string; variant: "success" | "warning" | "danger" }> = {
  high: { label: "Alta", variant: "success" },
  medium: { label: "Média", variant: "warning" },
  low: { label: "Baixa", variant: "danger" },
}

// --- mode determination ---

function getSheetMode(rec: Recommendation): "over" | "healthy" | "leak" | "unconfigured" | "collecting" | "under" {
  if (rec.status === "collecting_data") return "collecting"
  if (rec.status === "unconfigured") return "unconfigured"
  if (rec.status === "alerted" && (rec.memory_trend?.has_leak)) return "leak"
  if (rec.status === "under_provisioned" || rec.status === "alerted") return "under"
  if (rec.status === "over_provisioned") return "over"
  return "healthy"
}

export default function Studio() {
  const queryClient = useQueryClient()
  const [statusFilter, setStatusFilter] = useState<string>("all")
  const [patternFilter, setPatternFilter] = useState<string>("all")
  const [confidenceFilter, setConfidenceFilter] = useState<string>("medium")
  const [bulkModalOpen, setBulkModalOpen] = useState(false)

  // Sheet state
  const [sheetRec, setSheetRec] = useState<Recommendation | null>(null)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [scheduleOpen, setScheduleOpen] = useState(false)
  const [showStorage, setShowStorage] = useState(true)
  const [selectedTier, setSelectedTier] = useState<TierName>("balanced")
  const [whatIfCpu, setWhatIfCpu] = useState(0)
  const [whatIfMem, setWhatIfMem] = useState(0)
  const [applying, setApplying] = useState(false)

  const refreshInterval = useRefreshInterval()

  const { data: recs, isLoading } = useQuery<Recommendation[]>({
    queryKey: ["recommendations"],
    queryFn: () => api.get<Recommendation[]>("/recommendations"),
    refetchInterval: refreshInterval,
  })

  const { data: storageRecs } = useQuery<StorageAnalysis>({
    queryKey: ["storage-recommendations"],
    queryFn: () => api.get<StorageAnalysis>("/recommendations/storage"),
  })

  // --- counts ---
  const statusCounts = useMemo(() => {
    const c: Record<string, number> = {}
    for (const r of recs ?? []) c[r.status] = (c[r.status] ?? 0) + 1
    return c
  }, [recs])

  const patternCounts = useMemo(() => {
    const c: Record<string, number> = {}
    for (const r of recs ?? []) {
      const p = r.pattern ?? "unknown"
      c[p] = (c[p] ?? 0) + 1
    }
    return c
  }, [recs])

  // --- filters + sorting (prioridade: severidade → savings → OOMs) ---
  const filteredRecs = useMemo(() => {
    if (!recs) return []
    const filtered = recs.filter((r) => {
      if (statusFilter !== "all" && r.status !== statusFilter) return false
      if (patternFilter !== "all" && (r.pattern ?? "unknown") !== patternFilter) return false
      if (confidenceFilter === "high" && r.confidence !== "high") return false
      if (confidenceFilter === "medium" && r.confidence === "low") return false
      return true
    })
    const getFreed = (r: Recommendation) => {
      if (r.suggested_tiers) {
        const tier = r.suggested_tiers["balanced" as TierName]
        return tier?.resources_freed ?? null
      }
      return r.resources_freed?.balanced ?? null
    }
    return filtered.sort((a, b) => {
      const pa = statusConfig[a.status]?.priority ?? 99
      const pb = statusConfig[b.status]?.priority ?? 99
      if (pa !== pb) return pa - pb
      const fa = getFreed(a)
      const fb = getFreed(b)
      const sa = (fa?.cpu_cores ?? 0) + (fa?.mem_bytes ?? 0) / 1e9
      const sb = (fb?.cpu_cores ?? 0) + (fb?.mem_bytes ?? 0) / 1e9
      if (sa !== sb) return sb - sa
      return (b.oom_events ?? 0) - (a.oom_events ?? 0)
    })
  }, [recs, statusFilter, patternFilter, confidenceFilter])

  // --- apply mutation ---
  const applyMutation = useMutation({
    mutationFn: ({ service, values }: { service: string; values: ResourceValues }) =>
      api.post(`/recommendations/${service}/apply`, values),
    onSuccess: (_data, vars) => {
      queryClient.invalidateQueries({ queryKey: ["recommendations"] })
      setApplying(false)
      setSheetOpen(false)
      toast.success(`Recomendação aplicada para ${vars.service} com sucesso`)
    },
    onError: (_error, vars) => {
      setApplying(false)
      toast.error(`Erro ao aplicar recomendação para ${vars.service}`)
    },
  })

  // --- helpers ---
  const getTierData = (rec: Recommendation): ResourceValues | null => {
    if (!rec.suggested_tiers) return rec.suggested ?? null
    const tier = rec.suggested_tiers[selectedTier]
    if (!tier) return rec.suggested ?? null
    return {
      cpu_limit: tier.cpu_limit,
      mem_limit: tier.mem_limit,
      cpu_reservation: tier.cpu_reservation,
      mem_reservation: tier.mem_reservation,
    }
  }

  // Quando o tier muda, atualiza os sliders para os valores do tier selecionado
  useEffect(() => {
    if (!sheetRec) return
    const tier = getTierData(sheetRec)
    if (tier && tier.cpu_limit != null && tier.mem_limit != null) {
      setWhatIfCpu(tier.cpu_limit)
      setWhatIfMem(tier.mem_limit)
    }
  }, [selectedTier]) // eslint-disable-line react-hooks/exhaustive-deps

  const getResourcesFreed = (rec: Recommendation) => {
    if (!rec.suggested_tiers) return rec.resources_freed?.balanced ?? null
    const tier = rec.suggested_tiers[selectedTier]
    return tier?.resources_freed ?? null
  }

  const openSheet = (rec: Recommendation) => {
    setSheetRec(rec)
    setSheetOpen(true)
    const tier = getTierData(rec)
    if (tier && tier.cpu_limit != null && tier.mem_limit != null) {
      setWhatIfCpu(tier.cpu_limit)
      setWhatIfMem(tier.mem_limit)
    } else {
      // Manual mode — default values
      setWhatIfCpu(rec.cpu?.p95 ? rec.cpu.p95 * 1.3 : 1)
      setWhatIfMem(rec.mem?.p99 ? rec.mem.p99 * 1.3 : 512 * 1024 * 1024)
    }
  }

  const handleApply = () => {
    if (!sheetRec) return
    const values: ResourceValues = {
      cpu_limit: whatIfCpu,
      mem_limit: Math.round(whatIfMem),
      cpu_reservation: whatIfCpu * 0.75,
      mem_reservation: Math.round(whatIfMem * 0.75),
    }
    setApplying(true)
    applyMutation.mutate({ service: sheetRec.service, values })
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
        <PageHeader title="Otimização de Recursos" description="Sugestões de limites baseadas em dados" />
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

  const sheetMode = sheetRec ? getSheetMode(sheetRec) : "over"
  const isConfigMode = sheetMode === "collecting" || sheetMode === "unconfigured"

  return (
    <div className="space-y-4">
      {/* Header */}
      <PageHeader title="Otimização de Recursos" description="Sugestões de limites baseadas em dados">
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
            <Link to="/studio/rollback-watches">
              <RotateCcw className="mr-2 h-4 w-4" />
              Monitoramentos de Rollback
            </Link>
          </Button>
        </div>
      </PageHeader>

      {/* Hero */}
      <HeroMetric data={calculateHero(recs)} loading={isLoading} />

      {/* Filters */}
      <Card>
        <CardContent className="flex gap-2 flex-wrap items-center p-3">
          <FilterChip active={statusFilter === "all"} onClick={() => setStatusFilter("all")} label="Todos" count={recs.length} />
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
          {Object.entries(patternCounts)
            .filter(([pat]) => pat !== "unknown")
            .map(([pat, count]) => (
              <FilterChip
                key={pat}
                active={patternFilter === pat}
                onClick={() => setPatternFilter(patternFilter === pat ? "all" : pat)}
                label={patternLabel(pat)}
                count={count}
              />
            ))}
          {Object.keys(patternCounts).some(p => p !== "unknown") && (
            <Separator orientation="vertical" className="h-5 mx-1" />
          )}
          <div className="flex items-center gap-1.5">
            <Select value={confidenceFilter} onValueChange={setConfidenceFilter}>
              <SelectTrigger className="h-7 w-32 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="low">Todas</SelectItem>
                <SelectItem value="medium">Média e Alta</SelectItem>
                <SelectItem value="high">Apenas Alta</SelectItem>
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

      {/* Cards compactos */}
      <div className="space-y-3">
        {filteredRecs.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center text-sm text-muted-foreground space-y-3">
              <p>Nenhum serviço corresponde aos filtros selecionados.</p>
              <Button variant="outline" size="sm" onClick={() => { setStatusFilter("all"); setPatternFilter("all") }}>
                <X className="mr-1 h-3 w-3" />
                Limpar filtros
              </Button>
            </CardContent>
          </Card>
        ) : (
          filteredRecs.map((rec) => (
            <CompactCard
              key={rec.service}
              rec={rec}
              freed={getResourcesFreed(rec)}
              onConfigure={() => openSheet(rec)}
            />
          ))
        )}
      </div>

      {/* Storage recommendations */}
      {storageRecs && storageRecs.recommendations.length > 0 && (
        <div className="space-y-3">
          <button
            className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
            onClick={() => setShowStorage(!showStorage)}
          >
            <ChevronDown className={cn("h-4 w-4 transition-transform", !showStorage && "-rotate-90")} />
            <Database className="h-4 w-4 text-chart-5" />
            <span className="font-medium">Storage ({storageRecs.recommendations.length})</span>
            <span className="text-xs text-muted-foreground/70">
              {showStorage ? "ocultar" : "ver recomendações"}
            </span>
          </button>
          {showStorage && (
            <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
              {storageRecs.recommendations.map((rec, i) => {
                const sc = severityConfig[rec.severity] ?? severityConfig.info
                const SevIcon = sc.icon
                const TypeIcon = storageTypeIcon[rec.type] ?? Database
                return (
                  <Card key={i} className={cn("border-l-2", sc.border)}>
                    <CardContent className="space-y-2 py-3">
                      <div className="flex items-start gap-2">
                        <div className={cn("flex h-7 w-7 items-center justify-center rounded-lg shrink-0", sc.bg)}>
                          <TypeIcon className={cn("h-3.5 w-3.5", sc.color)} />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-1.5">
                            <SevIcon className={cn("h-3 w-3 shrink-0", sc.color)} />
                            <span className={cn("text-[11px] font-medium uppercase tracking-wide", sc.color)}>
                              {rec.severity}
                            </span>
                          </div>
                          <p className="text-xs text-foreground mt-1">{rec.message}</p>
                        </div>
                      </div>
                      {rec.action && (
                        <div className="rounded-md bg-muted/50 px-2 py-1.5 font-mono text-[10px] text-muted-foreground break-all">
                          {rec.action}
                        </div>
                      )}
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* Sheet — painel lateral de configuração */}
      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent side="right" className="w-full sm:max-w-[560px] p-0 overflow-y-auto">
          {sheetRec && (
            <>
              <SheetHeader className="border-b">
                <SheetTitle className="text-lg">Configurar — {sheetRec.service}</SheetTitle>
                <div className="flex items-center gap-2 flex-wrap text-sm text-muted-foreground">
                  <Badge variant={statusConfig[sheetRec.status]?.variant ?? "outline"} className="text-[10px]">
                    {statusConfig[sheetRec.status]?.label ?? sheetRec.status}
                  </Badge>
                  <span className="text-xs">
                    {patternLabel(sheetRec.pattern)} ·
                    {" "}P95 CPU {sheetRec.cpu ? formatCPU(sheetRec.cpu.p95) : "—"} ·
                    {" "}P99 Mem {sheetRec.mem ? formatBytes(sheetRec.mem.p99) : "—"}
                    {sheetRec.oom_events ? ` · ${sheetRec.oom_events} OOMs` : ""}
                  </span>
                </div>
              </SheetHeader>

              {/* Sheet body — mode-specific */}
              <div className="flex-1 overflow-y-auto p-6 space-y-4">
                <SheetBody
                  mode={sheetMode}
                  rec={sheetRec}
                  selectedTier={selectedTier}
                  onTierChange={setSelectedTier}
                  whatIfCpu={whatIfCpu}
                  whatIfMem={whatIfMem}
                  onCpuChange={setWhatIfCpu}
                  onMemChange={setWhatIfMem}
                  tierData={getTierData(sheetRec)}
                />
              </div>

              {/* Sheet footer */}
              <SheetFooter className="border-t flex-row justify-end gap-2 p-4">
                <Button variant="outline" size="sm" onClick={() => setSheetOpen(false)}>Cancelar</Button>
                <Button variant="outline" size="sm" onClick={() => setScheduleOpen(true)}>
                  <CalendarClock className="mr-2 h-4 w-4" />
                  Agendar
                </Button>
                <Button size="sm" onClick={handleApply} disabled={applying}>
                  <ShieldCheck className="mr-2 h-4 w-4" />
                  {applying ? "Aplicando..." : isConfigMode ? "Configurar e monitorar →" : "Aplicar com Rollback →"}
                </Button>
              </SheetFooter>
            </>
          )}
        </SheetContent>
      </Sheet>

      {/* Modal: Simulação em Lote (Dialog) */}
      <BulkSimulateModal
        open={bulkModalOpen}
        onOpenChange={setBulkModalOpen}
        recs={filteredRecs}
        selectedTier={selectedTier}
        getTierData={getTierData}
        getResourcesFreed={getResourcesFreed}
      />

      {/* Dialog: Agendar aplicação */}
      {sheetRec && (
        <ScheduleDialog
          rec={sheetRec}
          open={scheduleOpen}
          onOpenChange={setScheduleOpen}
          onScheduled={() => {
            queryClient.invalidateQueries({ queryKey: ["schedules"] })
            toast.success(`Agendamento criado para ${sheetRec.service}`)
          }}
        />
      )}
    </div>
  )
}

// --- Filter Chip ---

function FilterChip({ active, onClick, label, count }: {
  active: boolean; onClick: () => void; label: string; count: number
}) {
  return (
    <button
      onClick={onClick}
      className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs transition-colors ${
        active ? "border-primary bg-primary text-primary-foreground" : "border-border bg-secondary text-muted-foreground hover:border-primary"
      }`}
    >
      {label}
      <span className={`tabular-nums ${active ? "opacity-80" : "opacity-60"}`}>{count}</span>
    </button>
  )
}

// --- Compact Card ---

function CompactCard({ rec, freed, onConfigure }: {
  rec: Recommendation
  freed: { cpu_cores: number; mem_bytes: number; cpu_pct: number; mem_pct: number } | null
  onConfigure: () => void
}) {
  const cfg = statusConfig[rec.status] ?? statusConfig.over_provisioned
  const StatusIcon = cfg.icon
  const hasLeak = rec.memory_trend?.has_leak ?? false
  const isConfig = rec.status === "unconfigured" || rec.status === "collecting_data"
  const hasCurrentConfig = rec.current && (rec.current.cpu_limit > 0 || rec.current.mem_limit > 0)
  const hasPattern = rec.pattern && rec.pattern !== "unknown"
  const hasSavings = !isConfig && freed && (freed.cpu_cores > 0 || freed.mem_bytes > 0)
  const showLowConfidence = rec.confidence && rec.confidence === "low" && rec.status !== "collecting_data"

  return (
    <Card className={cn("hover:border-primary/40 transition-colors border-l-2", cfg.borderClass)}>
      <div className="flex items-center gap-3 p-3.5">
        {/* Linha 1 essencial: nome + status + ação */}
        <StatusIcon className={cn("h-4 w-4 shrink-0", cfg.variant === "danger" ? "text-destructive" : cfg.variant === "warning" ? "text-warning" : cfg.variant === "success" ? "text-success" : "text-muted-foreground")} />
        <span className="font-semibold text-sm">{rec.service}</span>
        {showLowConfidence && (
          <Badge variant="danger" className="text-[10px] py-0">Baixa confiança</Badge>
        )}
        {/* Métricas secundárias em muted */}
        <span className="text-xs text-muted-foreground hidden md:inline">
          {rec.status === "collecting_data"
            ? `${rec.samples} amostras`
            : rec.status === "unconfigured"
            ? "Sem limites"
            : hasCurrentConfig
              ? `${rec.current!.cpu_limit > 0 ? `${rec.current!.cpu_limit.toFixed(2)} cores` : ""}${rec.current!.cpu_limit > 0 && rec.current!.mem_limit > 0 ? " · " : ""}${rec.current!.mem_limit > 0 ? formatBytes(rec.current!.mem_limit) : ""}`
              : ""}
        </span>
        <span className="text-xs text-muted-foreground/70 hidden lg:inline">
          {rec.cpu ? `P95 ${formatCPU(rec.cpu.p95)}` : ""}{rec.cpu && rec.mem ? " · " : ""}{rec.mem ? `P99 ${formatBytes(rec.mem.p99)}` : ""}
        </span>
        <div className="flex-1" />
        {/* Savings ou leak — só se relevante */}
        {hasSavings ? (
          <span className="text-sm font-semibold text-success tabular-nums whitespace-nowrap">
            {freed!.cpu_cores > 0 && `↓ ${formatCPU(freed!.cpu_cores)} cores`}
            {freed!.cpu_cores > 0 && freed!.mem_bytes > 0 && " · "}
            {freed!.mem_bytes > 0 && `↓ ${formatBytes(freed!.mem_bytes)}`}
          </span>
        ) : hasLeak ? (
          <span className="text-sm font-semibold text-warning tabular-nums whitespace-nowrap">
            +{formatBytes(Math.abs(rec.memory_trend?.daily_growth_mb ?? 0) * 1e6)}/dia
          </span>
        ) : null}
        <Button size="sm" variant="outline" className="h-7 text-xs shrink-0" onClick={onConfigure}>
          <Settings2 className="mr-1 h-3 w-3" />
          Configurar
        </Button>
      </div>
    </Card>
  )
}

// --- Sheet Body (mode-specific) ---

interface SheetBodyProps {
  mode: string
  rec: Recommendation
  selectedTier: TierName
  onTierChange: (t: TierName) => void
  whatIfCpu: number
  whatIfMem: number
  onCpuChange: (v: number) => void
  onMemChange: (v: number) => void
  tierData: ResourceValues | null
}

function SheetBody({ mode, rec, selectedTier, onTierChange, whatIfCpu, whatIfMem, onCpuChange, onMemChange, tierData }: SheetBodyProps) {
  const hasLeak = rec.memory_trend?.has_leak ?? false

  // --- Mode: collecting_data (manual, sem ML) ---
  if (mode === "collecting") {
    return (
      <>
        <InfoBanner variant="warning">
          <strong>Dados insuficientes</strong> para sugestões automáticas (mínimo 100 amostras, atual: {rec.samples}).
          Defina os limites manualmente abaixo.
        </InfoBanner>

        <SectionLabel>
          Configuração manual
          <HelpIcon text="Sem sugestão ML — você define os valores. Volte depois da coleta mínima para sugestões automáticas." title="Configuração Manual" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={0.1}
          cpuMax={8}
          memMin={16 * 1024 * 1024}
          memMax={8 * 1024 * 1024 * 1024}
          cpuCurrent={0}
          memCurrent={0}
          cpuSuggested={0}
          memSuggested={0}
          onCpuChange={onCpuChange}
          onMemChange={onMemChange}
        />

        <InfoBanner variant="info">
          <strong>Dica:</strong> comece com valores conservadores e ajuste após a coleta atingir 100 amostras.
        </InfoBanner>
      </>
    )
  }

  // --- Mode: unconfigured (primeira config com sugestão ML) ---
  if (mode === "unconfigured") {
    return (
      <>
        <InfoBanner variant="info">
          Este serviço não tem limites configurados. Defina os valores abaixo —
          as sugestões já consideram o uso real (P95/P99 dos últimos 7 dias).
        </InfoBanner>

        {rec.suggested_tiers && (
          <>
            <SectionLabel>
              Camadas de recomendação
              <HelpIcon title="Camadas" text="Conservadora: margem 2x P95 (mais seguro). Equilibrada: data-driven (recomendada). Agressiva: 1.1x P95 (máxima liberação)." />
            </SectionLabel>
            <LayerToggle value={selectedTier} onChange={onTierChange} suggestedTiers={rec.suggested_tiers} />
          </>
        )}

        <SectionLabel>
          Recursos
          <HelpIcon text="Sem limites atuais. Os valores sugeridos são baseados no uso real coletado." title="Primeira Configuração" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.8}
          cpuMax={8}
          memMin={(rec.mem?.p99 ?? 16e6) * 0.8}
          memMax={8e9}
          cpuCurrent={0}
          memCurrent={0}
          cpuSuggested={tierData?.cpu_limit ?? 0}
          memSuggested={tierData?.mem_limit ?? 0}
          onCpuChange={onCpuChange}
          onMemChange={onMemChange}
        />

        <SectionLabel>
          Painel de Simulação
          <HelpIcon text="Simulação da primeira configuração baseada nos dados coletados." title="Simulação" />
        </SectionLabel>
        <WhatIfPanel
          service={rec.service}
          whatIfCpu={whatIfCpu}
          whatIfMem={whatIfMem}
          currentCpu={0}
          currentMem={0}
          p95Cpu={rec.cpu?.p95 ?? 0}
          p99Mem={rec.mem?.p99 ?? 0}
          forecastP99={rec.forecast?.projected_mem_p99 ?? 0}
          oomEvents={rec.oom_events ?? 0}
          hasLeak={hasLeak}
        />
      </>
    )
  }

  // --- Mode: under / leak (precisa de MAIS recurso) ---
  if (mode === "under" || mode === "leak") {
    return (
      <>
        <InfoBanner variant="danger">
          {mode === "leak" ? (
            <>
              <strong>Memory leak detectado</strong>: R²={rec.memory_trend?.r_squared.toFixed(2)},
              crescimento de ~{rec.memory_trend?.daily_growth_mb}MB/dia.
              A sugestão é aumentar memória + investigar o código.
            </>
          ) : (
            <>
              Este serviço está estressado: {rec.oom_events ?? 0} OOMs nos últimos 30 dias,
              {" "}CPU P95 em {rec.cpu && rec.current?.cpu_limit ? Math.round(rec.cpu.p95 / rec.current.cpu_limit * 100) : "?"}% do limite.
              A sugestão é <strong>aumentar</strong> os recursos.
            </>
          )}
        </InfoBanner>

        {rec.suggested_tiers && (
          <>
            <SectionLabel>
              Camadas de recomendação
              <HelpIcon title="Camadas" text="Para under-provisioned, todas as camadas sugerem AUMENTAR recursos. Conservadora = maior aumento." />
            </SectionLabel>
            <LayerToggle value={selectedTier} onChange={onTierChange} suggestedTiers={rec.suggested_tiers} />
          </>
        )}

        <SectionLabel>
          Recursos
          <HelpIcon text="Sugestão é AUMENTAR (em laranja). Marcador cinza = atual, azul = sugerido." title="Aumento de Recursos" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.8}
          cpuMax={(rec.current?.cpu_limit ?? 4) * 3}
          memMin={(rec.mem?.p99 ?? 16e6) * 0.8}
          memMax={(rec.current?.mem_limit ?? 8e9) * 3}
          cpuCurrent={rec.current?.cpu_limit ?? 0}
          memCurrent={rec.current?.mem_limit ?? 0}
          cpuSuggested={tierData?.cpu_limit ?? 0}
          memSuggested={tierData?.mem_limit ?? 0}
          onCpuChange={onCpuChange}
          onMemChange={onMemChange}
        />

        <SectionLabel>
          Painel de Simulação
          <HelpIcon text="Impacto do aumento de recursos. Risco baixa porque OOMs esperados = 0 com novo limite." title="Simulação" />
        </SectionLabel>
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
      </>
    )
  }

  // --- Mode: healthy (saudável) ---
  if (mode === "healthy") {
    return (
      <>
        <InfoBanner variant="info">
          Este serviço está saudável — recursos bem ajustados, sem OOMs, sem leak.
          Você ainda pode ajustar manualmente se necessário.
        </InfoBanner>

        {rec.suggested_tiers && (
          <>
            <SectionLabel>
              Camadas de recomendação
              <HelpIcon title="Camadas" text="Conservadora: 2x P95. Equilibrada: data-driven. Agressiva: 1.1x P95." />
            </SectionLabel>
            <LayerToggle value={selectedTier} onChange={onTierChange} suggestedTiers={rec.suggested_tiers} />
          </>
        )}

        <SectionLabel>
          Recursos atuais
          <HelpIcon text="Valores bem ajustados. Sem sugestão de mudança do ML." title="Saudável" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.5}
          cpuMax={(rec.current?.cpu_limit ?? 4) * 2}
          memMin={(rec.mem?.p99 ?? 16e6) * 0.5}
          memMax={(rec.current?.mem_limit ?? 8e9) * 2}
          cpuCurrent={rec.current?.cpu_limit ?? 0}
          memCurrent={rec.current?.mem_limit ?? 0}
          cpuSuggested={tierData?.cpu_limit ?? 0}
          memSuggested={tierData?.mem_limit ?? 0}
          onCpuChange={onCpuChange}
          onMemChange={onMemChange}
        />

        <SectionLabel>
          Painel de Simulação
          <HelpIcon text="Tudo dentro do esperado. Sem ação necessária." title="Simulação" />
        </SectionLabel>
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
      </>
    )
  }

  // --- Mode: over (otimização com sugestões ML completas) ---
  return (
    <>
      <InfoBanner variant="info">
        Este serviço está over-provisioned. A sugestão é <strong>reduzir</strong> recursos
        baseada em P95/P99 reais.
      </InfoBanner>

      {rec.suggested_tiers && (
        <>
          <SectionLabel>
            Camadas de recomendação
            <HelpIcon title="Camadas" text="Conservadora: 2x P95 (seguro). Equilibrada: data-driven (recomendada). Agressiva: 1.1x P95 (máxima liberação)." />
          </SectionLabel>
          <LayerToggle value={selectedTier} onChange={onTierChange} suggestedTiers={rec.suggested_tiers} />
        </>
      )}

      <SectionLabel>
        Recursos
        <HelpIcon text="Arraste os sliders para simular. Azul = sugerido, cinza = atual." title="Sliders" />
      </SectionLabel>

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

      <SectionLabel>
        Painel de Simulação
        <HelpIcon text="Simulação em tempo real: mostra o impacto dos valores escolhidos. Atualiza ao arrastar." title="Simulação" />
      </SectionLabel>
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

      <div className="flex gap-2 flex-wrap pt-2">
        <ExportYamlButton services={[rec.service]} tier={selectedTier} />
      </div>
    </>
  )
}

// --- Info Banner ---

function InfoBanner({ variant, children }: { variant: "info" | "warning" | "danger"; children: React.ReactNode }) {
  const styles = {
    info: "bg-primary/8 border-primary/20 text-foreground",
    warning: "bg-warning/8 border-warning/20 text-foreground",
    danger: "bg-destructive/8 border-destructive/20 text-foreground",
  }
  return (
    <div className={`flex items-start gap-2 rounded-lg border p-3 text-xs leading-relaxed ${styles[variant]}`}>
      <div className="flex-1 [&_strong]:text-primary">{children}</div>
    </div>
  )
}

// --- Section Label ---

function SectionLabel({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-1 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
      {children}
    </div>
  )
}

// --- Modal: Simulação em Lote (Dialog) ---

function BulkSimulateModal({
  open, onOpenChange, recs, selectedTier, getTierData, getResourcesFreed,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  recs: Recommendation[]
  selectedTier: TierName
  getTierData: (rec: Recommendation) => ResourceValues | null
  getResourcesFreed: (rec: Recommendation) => { cpu_cores: number; mem_bytes: number; cpu_pct: number; mem_pct: number } | null
}) {
  const overProvRecs = recs.filter((r) => r.status === "over_provisioned" && r.suggested_tiers)

  const totals = overProvRecs.reduce((acc, r) => {
    const freed = getResourcesFreed(r)
    if (freed) { acc.cpu += freed.cpu_cores; acc.mem += freed.mem_bytes }
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
          <DialogTitle>Simulação em Lote — Otimização de Recursos</DialogTitle>
          <DialogDescription>
            Cenário: {selectedTier === "conservative" ? "Conservadora" : selectedTier === "balanced" ? "Equilibrada" : "Agressiva"}
            {" · "}{overProvRecs.length} serviços selecionados
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

            {overProvRecs.length > 0 && (
              <div className="flex gap-0.5 h-2 rounded-full overflow-hidden">
                {overProvRecs.map((rec) => {
                  const color = rec.risk?.color ?? "green"
                  const bg = color === "green" ? "bg-success" : color === "yellow" ? "bg-warning" : color === "orange" ? "bg-orange-500" : "bg-destructive"
                  return <div key={rec.service} className={`flex-1 ${bg}`} />
                })}
              </div>
            )}

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

// --- Storage types & config ---

interface StorageRec {
  type: string
  severity: string
  message: string
  action?: string
}

interface StorageAnalysis {
  summary: {
    total_volumes: number
    total_volume_size: number
    orphan_count: number
    orphan_size: number
    reclaimable_volumes: number
    reclaimable_images: number
    recommendation_count: number
  }
  recommendations: StorageRec[]
}

const severityConfig: Record<string, { icon: typeof AlertTriangle; color: string; bg: string; border: string }> = {
  critical: { icon: AlertTriangle, color: "text-destructive", bg: "bg-destructive/10", border: "border-destructive/30" },
  warning: { icon: AlertTriangle, color: "text-warning", bg: "bg-warning/10", border: "border-warning/30" },
  info: { icon: CheckCircle2, color: "text-primary", bg: "bg-primary/10", border: "border-primary/30" },
}

const storageTypeIcon: Record<string, typeof Database> = {
  orphan_volume: Database,
  reclaimable_space: HardDrive,
  image_reclaimable: HardDrive,
  volume_growth: TrendingUp,
  large_volume: Database,
}

// --- Schedule Dialog ---

function ScheduleDialog({ rec, open, onOpenChange, onScheduled }: {
  rec: Recommendation
  open: boolean
  onOpenChange: (open: boolean) => void
  onScheduled: () => void
}) {
  const suggested = rec.suggested ?? { cpu_limit: 0, mem_limit: 0, cpu_reservation: 0, mem_reservation: 0 }
  const current = rec.current ?? { cpu_limit: 0, mem_limit: 0, cpu_reservation: 0, mem_reservation: 0 }
  const [mode, setMode] = useState<"suggest" | "custom">("suggest")
  const [selectedDate, setSelectedDate] = useState<Date | undefined>(undefined)
  const [timeHour, setTimeHour] = useState("02")
  const [timeMin, setTimeMin] = useState("00")
  const [popoverOpen, setPopoverOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const suggestedTime = rec.suggested_apply_time ? parseISO(rec.suggested_apply_time) : null

  const getFinalDate = (): Date | null => {
    if (mode === "suggest" && suggestedTime) return suggestedTime
    if (mode === "custom" && selectedDate) {
      const d = new Date(selectedDate)
      d.setHours(parseInt(timeHour) || 0, parseInt(timeMin) || 0, 0, 0)
      return d
    }
    return null
  }

  const finalDate = getFinalDate()

  const handleSubmit = async () => {
    const date = getFinalDate()
    if (!date) {
      setError("Selecione uma data e horário")
      return
    }
    if (date <= new Date()) {
      setError("A data deve ser no futuro")
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      await api.post("/schedules", {
        service: rec.service,
        cpu_limit: suggested.cpu_limit,
        mem_limit: suggested.mem_limit,
        cpu_reservation: suggested.cpu_reservation,
        mem_reservation: suggested.mem_reservation,
        scheduled_at: date.toISOString(),
      })
      onScheduled()
      onOpenChange(false)
    } catch (e: any) {
      setError(e?.response?.data?.detail ?? "Erro ao criar agendamento")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CalendarClock className="h-4 w-4 text-warning" />
            Agendar aplicação — {rec.service}
          </DialogTitle>
          <DialogDescription>
            Aplicar configurações em um horário agendado para evitar indisponibilidade.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-md border border-border/60 bg-muted/30 p-3 space-y-1">
            <div className="text-[10px] text-muted-foreground/70 uppercase tracking-wide">Configuração a ser aplicada</div>
            <div className="grid grid-cols-2 gap-x-3 gap-y-0.5 text-[11px]">
              <span className="text-muted-foreground">CPU Lim:</span>
              <span className="tabular-nums text-foreground">{current.cpu_limit > 0 ? formatCPU(current.cpu_limit) : "—"} → {formatCPU(suggested.cpu_limit)}</span>
              <span className="text-muted-foreground">Mem Lim:</span>
              <span className="tabular-nums text-foreground">{current.mem_limit > 0 ? formatBytes(current.mem_limit) : "—"} → {formatBytes(suggested.mem_limit)}</span>
              <span className="text-muted-foreground">CPU Res:</span>
              <span className="tabular-nums text-foreground">{current.cpu_reservation > 0 ? formatCPU(current.cpu_reservation) : "—"} → {formatCPU(suggested.cpu_reservation)}</span>
              <span className="text-muted-foreground">Mem Res:</span>
              <span className="tabular-nums text-foreground">{current.mem_reservation > 0 ? formatBytes(current.mem_reservation) : "—"} → {formatBytes(suggested.mem_reservation)}</span>
            </div>
          </div>

          <div className="space-y-2">
            <div className="text-xs font-medium text-muted-foreground">Quando aplicar?</div>
            <div className="grid grid-cols-2 gap-2">
              <button
                className={cn(
                  "rounded-md border px-3 py-2 text-xs text-left transition-all",
                  mode === "suggest"
                    ? "border-warning/50 bg-warning/10 text-foreground"
                    : "border-border/60 text-muted-foreground hover:border-border"
                )}
                onClick={() => setMode("suggest")}
              >
                <div className="flex items-center gap-1.5 font-medium">
                  <CalendarClock className="h-3 w-3" />
                  Sugerir horário
                </div>
                {suggestedTime && (
                  <div className="mt-1 text-[10px] text-muted-foreground">
                    {format(suggestedTime, "EEE dd/MM 'às' HH:mm")}
                    {rec.pattern && rec.pattern !== "unknown" && ` · ${patternLabel(rec.pattern)}`}
                  </div>
                )}
              </button>
              <button
                className={cn(
                  "rounded-md border px-3 py-2 text-xs text-left transition-all",
                  mode === "custom"
                    ? "border-primary/50 bg-primary/10 text-foreground"
                    : "border-border/60 text-muted-foreground hover:border-border"
                )}
                onClick={() => setMode("custom")}
              >
                <div className="flex items-center gap-1.5 font-medium">
                  <CalendarIcon className="h-3 w-3" />
                  Escolher data/hora
                </div>
                <div className="mt-1 text-[10px] text-muted-foreground">Selecionar manualmente</div>
              </button>
            </div>
          </div>

          {mode === "custom" && (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
                  <PopoverTrigger asChild>
                    <Button variant="outline" size="sm" className="gap-1.5 justify-start font-normal" data-empty={!selectedDate}>
                      <CalendarIcon className="h-3.5 w-3.5" />
                      {selectedDate ? format(selectedDate, "dd/MM/yyyy") : "Selecionar data"}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-auto p-0" align="start">
                    <Calendar
                      mode="single"
                      selected={selectedDate}
                      onSelect={(d) => { setSelectedDate(d); setPopoverOpen(false) }}
                      disabled={(d) => d < new Date(new Date().setHours(0, 0, 0, 0))}
                    />
                  </PopoverContent>
                </Popover>
                <div className="flex items-center gap-1">
                  <Input
                    type="text"
                    value={timeHour}
                    onChange={(e) => setTimeHour(e.target.value.slice(0, 2))}
                    className="w-12 text-center tabular-nums text-xs"
                    placeholder="HH"
                  />
                  <span className="text-muted-foreground text-xs">:</span>
                  <Input
                    type="text"
                    value={timeMin}
                    onChange={(e) => setTimeMin(e.target.value.slice(0, 2))}
                    className="w-12 text-center tabular-nums text-xs"
                    placeholder="MM"
                  />
                </div>
              </div>
            </div>
          )}

          {finalDate && (
            <div className="rounded-md bg-warning/10 border border-warning/20 px-3 py-2 text-[11px] text-warning flex items-center gap-1.5">
              <AlertTriangle className="h-3 w-3 shrink-0" />
              <span>
                O serviço será reiniciado em <strong>{format(finalDate, "dd/MM 'às' HH:mm")}</strong> (alguns segundos de indisponibilidade)
              </span>
            </div>
          )}

          {error && (
            <div className="text-[11px] text-destructive bg-destructive/10 rounded px-2 py-1 flex items-center gap-1.5">
              <AlertCircle className="h-3 w-3" />
              {error}
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
            Cancelar
          </Button>
          <Button size="sm" onClick={handleSubmit} disabled={!finalDate || submitting}>
            <CalendarClock className="h-3.5 w-3.5" />
            {submitting ? "Agendando..." : "Confirmar agendamento"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
