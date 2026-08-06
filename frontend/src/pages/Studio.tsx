/**
 * Studio — Right-Sizing Studio (rota /optimizations).
 *
 * v2: cards compactos + Sheet lateral para edição + HelpIcons.
 * Layout: Hero → Filter chips → Cards compactos (botão Configurar abre Sheet).
 * Sheet com 6 modos: over, healthy, leak, unconfigured, collecting, under.
 * Modal bulk mantém como Dialog.
 */
import { useState, useMemo, useEffect } from "react"
import * as yaml from "yaml"
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
  AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle,
  AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction,
} from "@/components/ui/alert-dialog"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select"
import { Checkbox } from "@/components/ui/checkbox"
import { Combobox } from "@/components/combobox"
import { PageHeader } from "@/components/page-header"
import { HelpIcon } from "@/components/help-icon"
import { HeroMetric } from "@/components/right-sizing/HeroMetric"
import { LayerToggle } from "@/components/right-sizing/LayerToggle"
import { ResourceSlider } from "@/components/right-sizing/ResourceSlider"
import { WhatIfPanel } from "@/components/right-sizing/WhatIfPanel"
import { ExplainabilityPanel } from "@/components/right-sizing/ExplainabilityPanel"
import { formatBytes, formatCPU, formatCores, cn } from "@/lib/utils"
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
  Calendar as CalendarIcon, Shield, Zap, Loader2, Package, Eye,
} from "lucide-react"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

// --- tooltip helper ---

function InfoTooltip({ content, children, side = "top" }: { content: React.ReactNode; children: React.ReactNode; side?: "top" | "bottom" | "left" | "right" }) {
  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>
          {children as React.ReactElement}
        </TooltipTrigger>
        <TooltipContent side={side}>
          {content}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

// --- status config ---

interface StatusCfg {
  label: string
  icon: typeof AlertTriangle
  variant: "default" | "secondary" | "destructive" | "outline" | "success" | "warning" | "danger"
  borderClass: string
  priority: number
  description: string
}

const statusConfig: Record<string, StatusCfg> = {
  alerted: { label: "Crítico", icon: AlertTriangle, variant: "danger", borderClass: "border-l-destructive", priority: 0, description: "Serviço com OOMs recentes ou sob pressão crítica — ação urgente" },
  under_provisioned: { label: "Insuficiente", icon: AlertTriangle, variant: "danger", borderClass: "border-l-warning", priority: 1, description: "Limites atuais abaixo do necessário — risco de OOM ou throttle" },
  observing: { label: "Em observação", icon: Eye, variant: "secondary", borderClass: "border-l-blue-500", priority: 1.5, description: "Apply realizado — serviço em observação para reavaliação de status" },
  over_provisioned: { label: "Excesso", icon: TrendingDown, variant: "warning", borderClass: "border-l-primary", priority: 2, description: "Limites acima do necessário — recursos podem ser liberados" },
  unconfigured: { label: "Sem config", icon: Settings2, variant: "warning", borderClass: "border-l-warning", priority: 3, description: "Serviço sem limites de CPU/memória configurados" },
  collecting_data: { label: "Coletando", icon: Activity, variant: "secondary", borderClass: "border-l-muted-foreground", priority: 4, description: "Coletando métricas — recomendação disponível em breve" },
  healthy: { label: "Saudável", icon: CheckCircle2, variant: "success", borderClass: "border-l-success", priority: 5, description: "Limites adequados ao uso real — sem ação necessária" },
}

// --- helpers ---

function parseMemoryToBytes(s: string): number {
  if (!s) return 0
  const m = s.match(/^(\d+(?:\.\d+)?)\s*([KMG]?)B?$/i)
  if (!m) return 0
  const val = parseFloat(m[1])
  const unit = m[2].toUpperCase()
  if (unit === "K") return val * 1024
  if (unit === "M") return val * 1024 * 1024
  if (unit === "G") return val * 1024 * 1024 * 1024
  return val
}

function parseTemplateYaml(yamlContent: string): { cpu: number; mem: number } | null {
  try {
    const parsed = yaml.parse(yamlContent) as { limits?: { cpus?: string; memory?: string } }
    if (!parsed?.limits) return null
    return {
      cpu: parseFloat(parsed.limits.cpus ?? "0") || 0,
      mem: parseMemoryToBytes(parsed.limits.memory ?? ""),
    }
  } catch {
    return null
  }
}

function timeAgo(dateStr: string | null | undefined): string | null {
  if (!dateStr) return null
  try {
    const d = new Date(dateStr)
    const diffMs = Date.now() - d.getTime()
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return "agora"
    if (diffMin < 60) return `há ${diffMin}min`
    const diffH = Math.floor(diffMin / 60)
    if (diffH < 24) return `há ${diffH}h`
    const diffD = Math.floor(diffH / 24)
    return `há ${diffD}d`
  } catch {
    return null
  }
}

// --- mode determination ---

function getSheetMode(rec: Recommendation): "over" | "healthy" | "leak" | "unconfigured" | "collecting" | "under" | "observing" {
  if (rec.status === "collecting_data") return "collecting"
  if (rec.status === "unconfigured") return "unconfigured"
  if (rec.status === "alerted" && (rec.memory_trend?.has_leak)) return "leak"
  if (rec.status === "observing") return "observing"
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
  const [selectedTemplateName, setSelectedTemplateName] = useState<string>("")
  const [whatIfCpu, setWhatIfCpu] = useState(0)
  const [whatIfMem, setWhatIfMem] = useState(0)
  const [applying, setApplying] = useState(false)
  const [recalculating, setRecalculating] = useState(false)
  const [quickApplying, setQuickApplying] = useState<string | null>(null)

  const refreshInterval = useRefreshInterval()

  const { data: recs, isLoading } = useQuery<Recommendation[]>({
    queryKey: ["recommendations"],
    queryFn: () => api.get<Recommendation[]>("/recommendations"),
    refetchInterval: refreshInterval,
  })

  const { data: rollbackWatches } = useQuery<{ watches: { id: number; service: string; status: string }[] }>({
    queryKey: ["rollback-watches-active"],
    queryFn: () => api.get("/rollback-watches?status=monitoring"),
    refetchInterval: refreshInterval,
  })
  const watchedServices = useMemo(() => {
    const set = new Set<string>()
    for (const w of rollbackWatches?.watches ?? []) {
      if (w.status === "monitoring") set.add(w.service)
    }
    return set
  }, [rollbackWatches])

  const { data: pendingSchedules } = useQuery<{ id: number; service: string; scheduled_at: string }[]>({
    queryKey: ["schedules", "pending"],
    queryFn: () => api.get("/schedules/pending"),
    refetchInterval: refreshInterval,
  })
  const scheduledServices = useMemo(() => {
    const map = new Map<string, string>()
    for (const s of pendingSchedules ?? []) {
      // API retorna PascalCase (Service/ScheduledAt) — aceitar ambos
      const svc = (s as { Service?: string; service?: string }).Service ?? (s as { service?: string }).service
      const at = (s as { ScheduledAt?: string; scheduled_at?: string }).ScheduledAt ?? (s as { scheduled_at?: string }).scheduled_at
      if (svc && at) map.set(svc, at)
    }
    return map
  }, [pendingSchedules])

  const { data: changeLog } = useQuery<{ Service: string; Action: string; Status: string }[]>({
    queryKey: ["change-log-recent"],
    queryFn: () => api.get("/change-log"),
    refetchInterval: refreshInterval,
  })
  const optimizedCount = useMemo(() => {
    const services = new Set<string>()
    for (const e of changeLog ?? []) {
      if (e.Action === "apply" && e.Status === "completed") services.add(e.Service)
    }
    return services.size
  }, [changeLog])

  const { data: storageRecs } = useQuery<StorageAnalysis>({
    queryKey: ["storage-recommendations"],
    queryFn: () => api.get<StorageAnalysis>("/recommendations/storage"),
  })

  const { data: templates } = useQuery<{ id: number; name: string; description: string; yaml_content: string; stacks: string[] }[]>({
    queryKey: ["templates"],
    queryFn: () => api.get("/templates"),
  })

  // Capacidade total do cluster Swarm (soma dos nodes ready) — usada como max do slider em modo template
  const { data: clusterCapacity } = useQuery<{ cluster_capacity?: { cpu_total: number; mem_total: number } }>({
    queryKey: ["dashboard-cluster-capacity"],
    queryFn: () => api.get("/dashboard"),
    staleTime: 60_000,
  })
  const clusterCpuTotal = clusterCapacity?.cluster_capacity?.cpu_total ?? 16
  const clusterMemTotal = clusterCapacity?.cluster_capacity?.mem_total ?? 16 * 1024 * 1024 * 1024

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
        const tier = r.suggested_tiers["balanced"]
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
      queryClient.invalidateQueries({ queryKey: ["services"] })
      queryClient.invalidateQueries({ queryKey: ["rollback-watches-active"] })
      queryClient.invalidateQueries({ queryKey: ["schedules", "pending"] })
      queryClient.invalidateQueries({ queryKey: ["change-log-recent"] })
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
    if (selectedTier === "template") return null // template não tem tierData ML
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
    if (selectedTier === "template") {
      // Template: alimentar sliders com valores do YAML do template selecionado
      if (!selectedTemplateName) return
      const tmpl = (templates ?? []).find((t) => t.name === selectedTemplateName)
      if (!tmpl) return
      const parsed = parseTemplateYaml(tmpl.yaml_content)
      if (parsed && parsed.cpu > 0 && parsed.mem > 0) {
        setWhatIfCpu(parsed.cpu)
        setWhatIfMem(parsed.mem)
      }
      return
    }
    const tier = getTierData(sheetRec)
    if (tier && tier.cpu_limit != null && tier.mem_limit != null) {
      setWhatIfCpu(tier.cpu_limit)
      setWhatIfMem(tier.mem_limit)
    }
  }, [selectedTier, selectedTemplateName, templates, sheetRec]) // eslint-disable-line react-hooks/exhaustive-deps

  const getResourcesFreed = (rec: Recommendation) => {
    if (selectedTier === "template") return null
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
    if (selectedTier === "template") {
      if (!selectedTemplateName) {
        toast.error("Selecione um template")
        return
      }
      setApplying(true)
      api.post<{ success: boolean; message: string }>(`/templates/${selectedTemplateName}/apply/${sheetRec.service}`)
        .then((data) => {
          if (data.success) {
            toast.success(data.message || `Template ${selectedTemplateName} aplicado em ${sheetRec.service}`)
          } else {
            toast.error(data.message || "Erro ao aplicar template")
          }
          queryClient.invalidateQueries({ queryKey: ["services"] })
          queryClient.invalidateQueries({ queryKey: ["recommendations"] })
          setSheetOpen(false)
        })
        .catch((err) => {
          toast.error(err instanceof Error ? err.message : "Erro ao aplicar template")
        })
        .finally(() => setApplying(false))
      return
    }
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
          <Button
            variant={statusCounts.over_provisioned >= 2 ? "default" : "outline"}
            size="sm"
            onClick={() => setBulkModalOpen(true)}
            className={statusCounts.over_provisioned >= 2 ? "gap-1.5" : ""}
          >
            <Layers className="mr-2 h-4 w-4" />
            Aplicação em Lote
            {statusCounts.over_provisioned >= 2 && (
              <Badge variant="secondary" className="ml-1 text-[10px] py-0">{statusCounts.over_provisioned}</Badge>
            )}
          </Button>
          <Button variant="outline" size="sm" disabled={recalculating} onClick={async () => {
            setRecalculating(true)
            toast.info("Recalculando recomendações...")
            await queryClient.invalidateQueries({ queryKey: ["recommendations"] })
            setRecalculating(false)
            toast.success("Recomendações atualizadas")
          }}>
            {recalculating ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Activity className="mr-2 h-4 w-4" />}
            {recalculating ? "Recalculando..." : "Recalcular"}
          </Button>
          <Button variant="outline" size="sm" asChild>
            <Link to="/optimizations/rollback-watches">
              <RotateCcw className="mr-2 h-4 w-4" />
              Monitoramentos de Rollback
            </Link>
          </Button>
        </div>
      </PageHeader>

      {/* Hero */}
      <HeroMetric data={{ ...calculateHero(recs), optimized_count: optimizedCount }} loading={isLoading} onPendingClick={() => setStatusFilter("over_provisioned")} />

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

      {/* Cards compactos — agrupados por status quando sem filtro */}
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
        ) : statusFilter === "all" && patternFilter === "all" ? (
          (() => {
            const groups: { key: string; label: string; icon: typeof AlertTriangle; recs: Recommendation[] }[] = []
            for (const [key, cfg] of Object.entries(statusConfig)) {
              const groupRecs = filteredRecs.filter(r => r.status === key)
              if (groupRecs.length > 0) groups.push({ key, label: cfg.label, icon: cfg.icon, recs: groupRecs })
            }
            return groups.map(g => {
              const GroupIcon = g.icon
              return (
                <div key={g.key} className="space-y-2">
                  <div className="flex items-center gap-2 px-1">
                    <GroupIcon className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">{g.label}</span>
                    <span className="text-xs text-muted-foreground/50">({g.recs.length})</span>
                  </div>
                  {g.recs.map((rec) => (
                    <CompactCard
                      key={rec.service}
                      rec={rec}
                      freed={getResourcesFreed(rec)}
                      onConfigure={() => openSheet(rec)}
                      isWatched={watchedServices.has(rec.service)}
                      scheduledAt={scheduledServices.get(rec.service)}
                      isQuickApplying={quickApplying === rec.service}
                      onQuickApply={async () => {
                        setQuickApplying(rec.service)
                        const tier = getTierData(rec)
                        if (!tier) { setQuickApplying(null); return }
                        try {
                          await api.post(`/recommendations/${rec.service}/apply`, {
                            cpu_limit: tier.cpu_limit,
                            mem_limit: Math.round(tier.mem_limit),
                            cpu_reservation: tier.cpu_limit * 0.75,
                            mem_reservation: Math.round(tier.mem_limit * 0.75),
                          })
                          queryClient.invalidateQueries({ queryKey: ["recommendations"] })
                          queryClient.invalidateQueries({ queryKey: ["rollback-watches-active"] })
                          queryClient.invalidateQueries({ queryKey: ["change-log-recent"] })
                          toast.success(`Aplicado para ${rec.service} com rollback ativo`)
                        } catch {
                          toast.error(`Erro ao aplicar para ${rec.service}`)
                        } finally {
                          setQuickApplying(null)
                        }
                      }}
                    />
                  ))}
                </div>
              )
            })
          })()
        ) : (
          filteredRecs.map((rec) => (
            <CompactCard
              key={rec.service}
              rec={rec}
              freed={getResourcesFreed(rec)}
              onConfigure={() => openSheet(rec)}
              isWatched={watchedServices.has(rec.service)}
              scheduledAt={scheduledServices.get(rec.service)}
              isQuickApplying={quickApplying === rec.service}
              onQuickApply={async () => {
                setQuickApplying(rec.service)
                const tier = getTierData(rec)
                if (!tier) { setQuickApplying(null); return }
                try {
                  await api.post(`/recommendations/${rec.service}/apply`, {
                    cpu_limit: tier.cpu_limit,
                    mem_limit: Math.round(tier.mem_limit),
                    cpu_reservation: tier.cpu_limit * 0.75,
                    mem_reservation: Math.round(tier.mem_limit * 0.75),
                  })
                  queryClient.invalidateQueries({ queryKey: ["recommendations"] })
                  queryClient.invalidateQueries({ queryKey: ["rollback-watches-active"] })
                  queryClient.invalidateQueries({ queryKey: ["change-log-recent"] })
                  toast.success(`Aplicado para ${rec.service} com rollback ativo`)
                } catch {
                  toast.error(`Erro ao aplicar para ${rec.service}`)
                } finally {
                  setQuickApplying(null)
                }
              }}
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
                  templates={templates ?? []}
                  selectedTemplateName={selectedTemplateName}
                  onTemplateChange={setSelectedTemplateName}
                  clusterCpuTotal={clusterCpuTotal}
                  clusterMemTotal={clusterMemTotal}
                />
              </div>

              {/* Sheet footer */}
              <SheetFooter className="border-t flex-row justify-end gap-2 p-4">
                <Button variant="outline" size="sm" onClick={() => setSheetOpen(false)}>Cancelar</Button>
                <Button variant="outline" size="sm" onClick={() => setScheduleOpen(true)}>
                  <CalendarClock className="mr-2 h-4 w-4" />
                  Agendar
                </Button>
                <Button size="sm" onClick={handleApply} disabled={applying || (selectedTier === "template" && !selectedTemplateName)}>
                  <ShieldCheck className="mr-2 h-4 w-4" />
                  {applying ? "Aplicando..." : selectedTier === "template" ? "Aplicar Template →" : isConfigMode ? "Configurar e monitorar →" : "Aplicar com Rollback →"}
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
            queryClient.invalidateQueries({ queryKey: ["schedules", "pending"] })
            setScheduleOpen(false)
            setSheetOpen(false)
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

function CompactCard({ rec, freed, onConfigure, onQuickApply, isWatched, scheduledAt, isQuickApplying }: {
  rec: Recommendation
  freed: { cpu_cores: number; mem_bytes: number; cpu_pct: number; mem_pct: number } | null
  onConfigure: () => void
  onQuickApply?: () => void
  isWatched: boolean
  scheduledAt?: string
  isQuickApplying: boolean
}) {
  const cfg = statusConfig[rec.status] ?? statusConfig.over_provisioned
  const StatusIcon = cfg.icon
  const hasLeak = rec.memory_trend?.has_leak ?? false
  const isConfig = rec.status === "unconfigured" || rec.status === "collecting_data"
  const hasCurrentConfig = rec.current && (rec.current.cpu_limit > 0 || rec.current.mem_limit > 0)
  const hasSavings = !isConfig && freed && (freed.cpu_cores > 0 || freed.mem_bytes > 0)
  const showLowConfidence = rec.confidence && rec.confidence === "low" && rec.status !== "collecting_data"
  const canQuickApply = !isConfig && rec.confidence === "high" && (rec.status === "over_provisioned" || rec.status === "healthy")
  const updatedAgo = timeAgo(rec.suggested_apply_time)

  // Cor do P95/P99 baseada na utilização vs limite
  // P95 já é porcentagem (7.18 = 7.18% de 1 core); cpu_limit é em cores
  // utilização = p95 / cpu_limit (em % do limite)
  const cpuUtilPct = rec.cpu && rec.current?.cpu_limit ? rec.cpu.p95 / rec.current.cpu_limit : 0
  const memUtilPct = rec.mem && rec.current?.mem_limit ? (rec.mem.p99 / rec.current.mem_limit) * 100 : 0
  const cpuColor = cpuUtilPct > 90 ? "text-destructive" : cpuUtilPct > 75 ? "text-warning" : "text-muted-foreground/70"
  const memColor = memUtilPct > 90 ? "text-destructive" : memUtilPct > 75 ? "text-warning" : "text-muted-foreground/70"

  return (
    <Card className={cn("hover:border-primary/40 transition-colors border-l-2", cfg.borderClass)} role="article" aria-label={`${rec.service} — ${cfg.label}`}>
      <div className="flex items-center gap-3 p-3.5">
        {/* Ícone de status com tooltip explicativo */}
        <InfoTooltip content={<p className="max-w-64"><span className="font-semibold">{cfg.label}</span> — {cfg.description}</p>}>
          <span>
            <StatusIcon className={cn("h-4 w-4 shrink-0 cursor-help", cfg.variant === "danger" ? "text-destructive" : cfg.variant === "warning" ? "text-warning" : cfg.variant === "success" ? "text-success" : "text-muted-foreground")} aria-hidden="true" />
          </span>
        </InfoTooltip>
        <span className="font-semibold text-sm">{rec.service}</span>
        {isWatched && (
          <InfoTooltip content={<p>Rollback watch ativo — monitorando estabilidade pós-apply</p>}>
            <Shield className="h-3.5 w-3.5 text-primary shrink-0 cursor-help" aria-label="Rollback watch ativo" />
          </InfoTooltip>
        )}
        {scheduledAt && (
          <InfoTooltip content={<p>Apply agendado para {format(parseISO(scheduledAt), "dd/MM/yyyy 'às' HH:mm")}</p>}>
            <CalendarClock className="h-3.5 w-3.5 text-chart-5 shrink-0 cursor-help" aria-label="Apply agendado" />
          </InfoTooltip>
        )}
        {showLowConfidence && (
          <InfoTooltip content={<p className="max-w-56">Recomendação baseada em poucas amostras — revise os detalhes antes de aplicar</p>}>
            <Badge variant="danger" className="text-[10px] py-0 cursor-help">Baixa confiança</Badge>
          </InfoTooltip>
        )}
        {/* Configuração atual com tooltip */}
        {rec.status !== "collecting_data" && rec.status !== "unconfigured" && hasCurrentConfig && (
          <InfoTooltip content={<p>Configuração atual de recursos do serviço</p>}>
            <span className="text-xs text-muted-foreground hidden md:inline cursor-help">
              {rec.current!.cpu_limit > 0 ? `${rec.current!.cpu_limit.toFixed(2)} cores` : ""}
              {rec.current!.cpu_limit > 0 && rec.current!.mem_limit > 0 && " · "}
              {rec.current!.mem_limit > 0 ? formatBytes(rec.current!.mem_limit) : ""}
            </span>
          </InfoTooltip>
        )}
        {rec.status === "collecting_data" && (
          <span className="text-xs text-muted-foreground hidden md:inline">{rec.samples} amostras</span>
        )}
        {rec.status === "unconfigured" && (
          <span className="text-xs text-muted-foreground hidden md:inline">Sem limites</span>
        )}
        {/* P95 CPU com tooltip explicativo */}
        {rec.cpu && (
          <InfoTooltip content={<p>P95 CPU: percentil 95 — pico de uso em 5% do tempo. {cpuUtilPct > 90 ? "Crítico: próximo do limite" : cpuUtilPct > 75 ? "Atenção: utilização alta" : "Utilização saudável"} ({cpuUtilPct.toFixed(0)}% do limite)</p>}>
            <span className={cn("text-xs hidden lg:inline tabular-nums cursor-help", cpuColor)}>
              P95 {formatCPU(rec.cpu.p95)}
            </span>
          </InfoTooltip>
        )}
        {rec.cpu && rec.mem && <span className="text-xs text-muted-foreground/40 hidden lg:inline">·</span>}
        {/* P99 Mem com tooltip explicativo */}
        {rec.mem && (
          <InfoTooltip content={<p>P99 Memória: percentil 99 — pico de uso em 1% do tempo. {memUtilPct > 90 ? "Crítico: próximo do limite" : memUtilPct > 75 ? "Atenção: utilização alta" : "Utilização saudável"} ({memUtilPct.toFixed(0)}% do limite)</p>}>
            <span className={cn("text-xs hidden lg:inline tabular-nums cursor-help", memColor)}>
              P99 {formatBytes(rec.mem.p99)}
            </span>
          </InfoTooltip>
        )}
        {/* Timestamp com tooltip */}
        {updatedAgo && (
          <InfoTooltip content={<p>Última atualização da recomendação</p>}>
            <span className="text-[10px] text-muted-foreground/50 hidden xl:inline cursor-help">· {updatedAgo}</span>
          </InfoTooltip>
        )}
        <div className="flex-1" />
        {/* Savings com tooltip */}
        {hasSavings ? (
          <InfoTooltip content={<p>Recursos que podem ser liberados aplicando o tier Equilibrada</p>}>
            <span className="text-sm font-semibold text-success tabular-nums whitespace-nowrap cursor-help">
              {freed!.cpu_cores > 0 && `↓ ${formatCores(freed!.cpu_cores)} cores`}
              {freed!.cpu_cores > 0 && freed!.mem_bytes > 0 && " · "}
              {freed!.mem_bytes > 0 && `↓ ${formatBytes(freed!.mem_bytes)}`}
            </span>
          </InfoTooltip>
        ) : hasLeak ? (
          <InfoTooltip content={<p className="max-w-56">Crescimento de memória detectado — possível leak. Monitorar antes de ajustar limites</p>}>
            <span className="text-sm font-semibold text-warning tabular-nums whitespace-nowrap cursor-help">
              +{formatBytes(Math.abs(rec.memory_trend?.daily_growth_mb ?? 0) * 1e6)}/dia
            </span>
          </InfoTooltip>
        ) : null}
        {canQuickApply && onQuickApply && (
          <Button
            size="sm"
            variant="ghost"
            className="h-7 text-xs shrink-0 text-primary"
            disabled={isQuickApplying}
            onClick={onQuickApply}
            aria-label={`Aplicar recomendação equilibrada para ${rec.service}`}
            title={`Aplicar tier Equilibrada: ${rec.suggested_tiers?.balanced ? `${rec.suggested_tiers.balanced.cpu_limit.toFixed(2)} cores · ${formatBytes(rec.suggested_tiers.balanced.mem_limit)}` : "valores sugeridos"}`}
          >
            {isQuickApplying ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : <Zap className="mr-1 h-3 w-3" />}
            <span className="hidden sm:inline">Aplicar</span>
          </Button>
        )}
        <Button
          size="sm"
          variant="outline"
          className="h-7 text-xs shrink-0"
          onClick={onConfigure}
          aria-label={`Configurar ${rec.service}`}
          title={`Abrir painel de configuração de ${rec.service} — ajustar limites, ver detalhes e simular tiers`}
        >
          <Settings2 className="mr-1 h-3 w-3" />
          <span className="hidden sm:inline">Configurar</span>
        </Button>
      </div>
    </Card>
  )
}

// --- Template selector (combobox apenas) ---
function TemplateSelector({ templates, selectedTemplateName, onTemplateChange }: {
  templates: { id: number; name: string; description: string; yaml_content: string; stacks: string[] }[]
  selectedTemplateName: string
  onTemplateChange: (name: string) => void
}) {
  return (
    <Combobox
      options={templates.map((t) => ({ value: t.name, label: t.name }))}
      value={selectedTemplateName}
      onChange={onTemplateChange}
      placeholder="Escolher template..."
      searchPlaceholder="Buscar template..."
      emptyText="Nenhum template encontrado."
    />
  )
}

// --- Template YAML collapsible (mesmo design do SectionLabel, vai acima dos sliders) ---
function TemplateYamlAccordion({ templates, selectedTemplateName }: {
  templates: { id: number; name: string; description: string; yaml_content: string; stacks: string[] }[]
  selectedTemplateName: string
}) {
  const [open, setOpen] = useState(false)
  const selected = templates.find((t) => t.name === selectedTemplateName)
  if (!selected) return null
  return (
    <div>
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1 text-xs font-semibold text-muted-foreground uppercase tracking-wide hover:text-foreground transition-colors w-full"
      >
        <Package className="h-3.5 w-3.5 text-chart-5" />
        Ver YAML — {selected.name}
        {selected.stacks?.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {selected.stacks.map((s) => (
              <Badge key={s} variant="outline" className="text-[10px] border-chart-5/40 text-chart-5 normal-case tracking-normal">{s}</Badge>
            ))}
          </div>
        )}
        <ChevronDown className={`h-3.5 w-3.5 ml-0.5 transition-transform ${open ? "rotate-180" : ""}`} />
      </button>
      {open && (
        <div className="pt-2 pb-1">
          <p className="text-xs text-muted-foreground mb-2">{selected.description}</p>
          <pre className="text-[10px] font-mono text-muted-foreground bg-background rounded p-2 overflow-auto max-h-40 border">
            {selected.yaml_content}
          </pre>
        </div>
      )}
    </div>
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
  templates: { id: number; name: string; description: string; yaml_content: string; stacks: string[] }[]
  selectedTemplateName: string
  onTemplateChange: (name: string) => void
  clusterCpuTotal: number
  clusterMemTotal: number
}

function SheetBody({ mode, rec, selectedTier, onTierChange, whatIfCpu, whatIfMem, onCpuChange, onMemChange, tierData, templates, selectedTemplateName, onTemplateChange, clusterCpuTotal, clusterMemTotal }: SheetBodyProps) {
  const hasLeak = rec.memory_trend?.has_leak ?? false
  const isTemplate = selectedTier === "template"

  // Slider max = capacidade real do cluster Swarm (soma dos nodes ready)
  // Padrão do mercado (Portainer, Lens, OpenShift): max = recursos disponíveis do ambiente
  const cpuMaxFor = (base: number) => Math.max(base, clusterCpuTotal)
  const memMaxFor = (base: number) => Math.max(base, clusterMemTotal)

  // Helper: renderiza LayerToggle + TemplateSelector (se template)
  const tierSection = (
    <>
      {rec.suggested_tiers && (
        <>
          <SectionLabel>
            Camadas de recomendação
            <HelpIcon title="Camadas" text="Conservadora: 2x P95. Equilibrada: data-driven. Agressiva: 1.1x P95. Template: perfil YAML manual." />
          </SectionLabel>
          <LayerToggle value={selectedTier} onChange={onTierChange} suggestedTiers={rec.suggested_tiers} />
        </>
      )}
      {isTemplate && (
        <>
          <SectionLabel>
            Template YAML
            <HelpIcon title="Template" text="Perfil YAML pré-definido com limits, reservations e margens fixas. Os sliders abaixo refletem os valores do template." />
          </SectionLabel>
          <TemplateSelector
            templates={templates}
            selectedTemplateName={selectedTemplateName}
            onTemplateChange={onTemplateChange}
          />
        </>
      )}
    </>
  )

  // Helper: YAML accordion após sliders (apenas em modo template)
  const yamlAccordion = isTemplate ? (
    <TemplateYamlAccordion templates={templates} selectedTemplateName={selectedTemplateName} />
  ) : null

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

        {yamlAccordion}

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={0.1}
          cpuMax={cpuMaxFor(8)}
          memMin={16 * 1024 * 1024}
          memMax={memMaxFor(8 * 1024 * 1024 * 1024)}
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

        {tierSection}

        {yamlAccordion}

        <SectionLabel>
          Recursos
          <HelpIcon text="Sem limites atuais. Os valores sugeridos são baseados no uso real coletado." title="Primeira Configuração" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.8}
          cpuMax={cpuMaxFor(8)}
          memMin={(rec.mem?.p99 ?? 16e6) * 0.8}
          memMax={memMaxFor(8e9)}
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

        {tierSection}

        {yamlAccordion}

        <SectionLabel>
          Recursos
          <HelpIcon text="Sugestão é AUMENTAR (em laranja). Marcador cinza = atual, azul = sugerido." title="Aumento de Recursos" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.8}
          cpuMax={cpuMaxFor((rec.current?.cpu_limit ?? 4) * 3)}
          memMin={(rec.mem?.p99 ?? 16e6) * 0.8}
          memMax={memMaxFor((rec.current?.mem_limit ?? 8e9) * 3)}
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

  // --- Mode: observing (em observação pós-apply) ---
  if (mode === "observing") {
    const appliedDate = rec.applied_at ? format(parseISO(rec.applied_at), "dd/MM/yyyy HH:mm") : null
    return (
      <>
        <InfoBanner variant="info">
          Apply realizado{appliedDate ? ` em ${appliedDate}` : ""} — serviço em observação.
          O status será reavaliado conforme novas métricas são coletadas.
          {rec.oom_events_since_apply === 0
            ? " Nenhum OOM novo desde o apply."
            : ` ${rec.oom_events_since_apply} OOM(s) novo(s) desde o apply — ação pode ser necessária.`}
        </InfoBanner>

        {tierSection}

        {yamlAccordion}

        <SectionLabel>
          Recursos atuais
          <HelpIcon text="Valores aplicados recentemente. Em observação para confirmar estabilidade." title="Em observação" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.5}
          cpuMax={cpuMaxFor((rec.current?.cpu_limit ?? 4) * 2)}
          memMin={(rec.mem?.p99 ?? 16e6) * 0.5}
          memMax={memMaxFor((rec.current?.mem_limit ?? 8e9) * 2)}
          cpuCurrent={rec.current?.cpu_limit ?? 0}
          memCurrent={rec.current?.mem_limit ?? 0}
          cpuSuggested={tierData?.cpu_limit ?? 0}
          memSuggested={tierData?.mem_limit ?? 0}
          onCpuChange={onCpuChange}
          onMemChange={onMemChange}
        />

        <SectionLabel>
          Painel de Simulação
          <HelpIcon text="Ajuste manual ainda é possível durante o período de observação." title="Simulação" />
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

  // --- Mode: healthy (saudável) ---
  if (mode === "healthy") {
    return (
      <>
        <InfoBanner variant="info">
          Este serviço está saudável — recursos bem ajustados, sem OOMs, sem leak.
          Você ainda pode ajustar manualmente se necessário.
        </InfoBanner>

        {tierSection}

        {yamlAccordion}

        <SectionLabel>
          Recursos atuais
          <HelpIcon text="Valores bem ajustados. Sem sugestão de mudança do ML." title="Saudável" />
        </SectionLabel>

        <ResourceSlider
          cpuCores={whatIfCpu}
          memBytes={whatIfMem}
          cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.5}
          cpuMax={cpuMaxFor((rec.current?.cpu_limit ?? 4) * 2)}
          memMin={(rec.mem?.p99 ?? 16e6) * 0.5}
          memMax={memMaxFor((rec.current?.mem_limit ?? 8e9) * 2)}
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

      {tierSection}

      {yamlAccordion}

      <SectionLabel>
        Recursos
        <HelpIcon text="Arraste os sliders para simular. Azul = sugerido, cinza = atual." title="Sliders" />
      </SectionLabel>

      <ResourceSlider
        cpuCores={whatIfCpu}
        memBytes={whatIfMem}
        cpuMin={(rec.cpu?.p95 ?? 0.1) * 0.8}
        cpuMax={cpuMaxFor((rec.current?.cpu_limit ?? 4) * 1.5)}
        memMin={(rec.mem?.p99 ?? 16e6) * 0.8}
        memMax={memMaxFor((rec.current?.mem_limit ?? 8e9) * 1.5)}
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

// --- Modal: Aplicação em Lote (Dialog) ---

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
  const [selected, setSelected] = useState<Set<string>>(new Set(overProvRecs.map(r => r.service)))
  const [applying, setApplying] = useState(false)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const queryClient = useQueryClient()

  // Reset seleção quando abre
  useEffect(() => {
    if (open) setSelected(new Set(overProvRecs.map(r => r.service)))
  }, [open])

  const toggle = (service: string) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(service)) next.delete(service)
      else next.add(service)
      return next
    })
  }

  const toggleAll = () => {
    if (selected.size === overProvRecs.length) setSelected(new Set())
    else setSelected(new Set(overProvRecs.map(r => r.service)))
  }

  const selectedRecs = overProvRecs.filter(r => selected.has(r.service))

  const totals = selectedRecs.reduce((acc, r) => {
    const freed = getResourcesFreed(r)
    if (freed) { acc.cpu += freed.cpu_cores; acc.mem += freed.mem_bytes }
    return acc
  }, { cpu: 0, mem: 0 })

  const riskCounts = selectedRecs.reduce((acc, r) => {
    const level = r.risk?.color ?? "green"
    acc[level] = (acc[level] ?? 0) + 1
    return acc
  }, {} as Record<string, number>)

  const handleApplyBatch = async () => {
    setApplying(true)
    let success = 0
    let errors = 0
    // Apply sequencial com delay para evitar rollback em cascata
    for (const rec of selectedRecs) {
      const tier = getTierData(rec)
      if (!tier) continue
      try {
        await api.post(`/recommendations/${rec.service}/apply`, {
          cpu_limit: tier.cpu_limit,
          mem_limit: Math.round(tier.mem_limit),
          cpu_reservation: tier.cpu_limit * 0.75,
          mem_reservation: Math.round(tier.mem_limit * 0.75),
        })
        success++
        // Delay de 500ms entre aplicações para não sobrecarregar o cluster
        await new Promise(r => setTimeout(r, 2000))
      } catch {
        errors++
      }
    }
    setApplying(false)
    onOpenChange(false)
    queryClient.invalidateQueries({ queryKey: ["recommendations"] })
    queryClient.invalidateQueries({ queryKey: ["rollback-watches-active"] })
    queryClient.invalidateQueries({ queryKey: ["change-log-recent"] })
    if (errors === 0) {
      toast.success(`${success} serviços aplicados com rollback ativo`)
    } else {
      toast.warning(`${success} aplicados, ${errors} falharam`)
    }
  }

  return (
    <>
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>Aplicação em Lote — Otimização de Recursos</DialogTitle>
          <DialogDescription>
            Tier: {selectedTier === "conservative" ? "Conservadora" : selectedTier === "balanced" ? "Equilibrada" : "Agressiva"}
            {" · "}{selectedRecs.length} de {overProvRecs.length} serviços selecionados
            {" · "}Apply sequencial com rollback automático
          </DialogDescription>
          <p className="text-xs text-muted-foreground">
            Apenas serviços <span className="font-medium text-warning">over-provisioned</span> com sugestões geradas aparecem aqui.
            Serviços críticos, saudáveis ou sem dados suficientes não podem ser otimizados em lote.
          </p>
        </DialogHeader>

        {overProvRecs.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            Nenhum serviço over-provisioned para aplicar.
          </div>
        ) : (
          <>
            <div className="max-h-[400px] overflow-auto rounded-md border">
              <Table>
                <TableHeader className="sticky top-0 bg-background z-10">
                  <TableRow>
                    <TableHead className="w-10">
                      <Checkbox
                        checked={selected.size === overProvRecs.length && overProvRecs.length > 0}
                        onCheckedChange={toggleAll}
                        aria-label="Selecionar todos"
                      />
                    </TableHead>
                    <TableHead>Serviço</TableHead>
                    <TableHead className="hidden sm:table-cell">CPU atual→sug</TableHead>
                    <TableHead className="hidden sm:table-cell">Mem atual→sug</TableHead>
                    <TableHead>Libera</TableHead>
                    <TableHead className="hidden md:table-cell">Risco</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {overProvRecs.map((rec) => {
                    const tier = getTierData(rec)
                    const freed = getResourcesFreed(rec)
                    const riskColor = rec.risk?.color ?? "green"
                    const isSelected = selected.has(rec.service)
                    return (
                      <TableRow key={rec.service} className={!isSelected ? "opacity-50" : ""}>
                        <TableCell>
                          <Checkbox
                            checked={isSelected}
                            onCheckedChange={() => toggle(rec.service)}
                            aria-label={`Selecionar ${rec.service}`}
                          />
                        </TableCell>
                        <TableCell className="font-medium text-sm whitespace-nowrap">{rec.service}</TableCell>
                        <TableCell className="text-xs tabular-nums hidden sm:table-cell whitespace-nowrap">
                          {formatCores(rec.current?.cpu_limit ?? 0)} → {formatCores(tier?.cpu_limit ?? 0)}
                        </TableCell>
                        <TableCell className="text-xs tabular-nums hidden sm:table-cell whitespace-nowrap">
                          {formatBytes(rec.current?.mem_limit ?? 0)} → {formatBytes(tier?.mem_limit ?? 0)}
                        </TableCell>
                        <TableCell className="text-xs tabular-nums text-success font-medium whitespace-nowrap">
                          {freed && (freed.cpu_cores > 0 || freed.mem_bytes > 0) ? (
                            <>
                              {freed.cpu_cores > 0 && `${formatCores(freed.cpu_cores)} cores`}
                              {freed.cpu_cores > 0 && freed.mem_bytes > 0 && " · "}
                              {freed.mem_bytes > 0 && formatBytes(freed.mem_bytes)}
                            </>
                          ) : "—"}
                        </TableCell>
                        <TableCell className="hidden md:table-cell">
                          <InfoTooltip content={<p className="max-w-56">Risco da aplicação: {riskLevelLabel[rec.risk?.level ?? "low"]} — {rec.risk?.color === "green" ? "aplicação segura" : rec.risk?.color === "yellow" ? "monitorar após apply" : "alta chance de rollback"}</p>}>
                            <Badge variant="outline" className={cn(riskColorClasses[riskColor as RiskColor], "cursor-help")}>
                              {riskLevelLabel[rec.risk?.level ?? "low"]}
                            </Badge>
                          </InfoTooltip>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                  <TableRow className="border-t-2 font-medium sticky bottom-0 bg-background">
                    <TableCell colSpan={3} className="hidden sm:table-cell whitespace-nowrap">TOTAL ({selectedRecs.length})</TableCell>
                    <TableCell className="sm:hidden" colSpan={3}>TOTAL ({selectedRecs.length})</TableCell>
                    <TableCell className="text-success tabular-nums whitespace-nowrap">
                      {formatCores(totals.cpu)} cores · {formatBytes(totals.mem)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground hidden md:table-cell whitespace-nowrap">
                      {riskCounts.green ?? 0} safe, {riskCounts.yellow ?? 0} atenção
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>

            {/* Barra de risco agregado */}
            {selectedRecs.length > 0 && (
              <div className="flex gap-0.5 h-3 rounded-full overflow-hidden">
                {selectedRecs.map((rec) => {
                  const color = rec.risk?.color ?? "green"
                  const bg = color === "green" ? "bg-success" : color === "yellow" ? "bg-warning" : color === "orange" ? "bg-warning/70" : "bg-destructive"
                  return <div key={rec.service} className={`flex-1 ${bg}`} />
                })}
              </div>
            )}
          </>
        )}

        <DialogFooter className="gap-2">
          <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>Cancelar</Button>
          <Button
            variant="default"
            size="sm"
            disabled={selectedRecs.length === 0 || applying}
            onClick={() => setConfirmOpen(true)}
            title={`Aplicar ${selectedRecs.length} serviços sequencialmente com rollback ativo (delay 2s entre cada para evitar cascata)`}
          >
            {applying ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Zap className="mr-2 h-4 w-4" />}
            {applying ? "Aplicando..." : `Aplicar ${selectedRecs.length}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    {/* Confirmação double-check antes de aplicar em lote */}
    <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Confirmar aplicação em lote</AlertDialogTitle>
          <AlertDialogDescription>
            Você está prestes a aplicar novos limites de CPU e memória em <strong>{selectedRecs.length} serviços</strong> sequencialmente (delay 2s entre cada).
            Cada serviço terá um rollback watch ativo que reverterá automaticamente se detectar OOM, throttle ou pressão de memória.
            <br /><br />
            Recursos liberados: <strong className="text-success">{formatCores(totals.cpu)} cores · {formatBytes(totals.mem)}</strong>
            <br />
            Risco agregado: {riskCounts.green ?? 0} seguro(s), {riskCounts.yellow ?? 0} com atenção
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction
            onClick={() => { handleApplyBatch(); setConfirmOpen(false) }}
            className="bg-primary text-primary-foreground hover:bg-primary/90"
          >
            <Zap className="mr-2 h-4 w-4" />
            Confirmar e aplicar
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </>
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
              <span className="tabular-nums text-foreground">{current.cpu_limit > 0 ? formatCores(current.cpu_limit) : "—"} → {formatCores(suggested.cpu_limit)}</span>
              <span className="text-muted-foreground">Mem Lim:</span>
              <span className="tabular-nums text-foreground">{current.mem_limit > 0 ? formatBytes(current.mem_limit) : "—"} → {formatBytes(suggested.mem_limit)}</span>
              <span className="text-muted-foreground">CPU Res:</span>
              <span className="tabular-nums text-foreground">{current.cpu_reservation > 0 ? formatCores(current.cpu_reservation) : "—"} → {formatCores(suggested.cpu_reservation)}</span>
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
