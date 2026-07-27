import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/api/client"
import { Card, CardHeader, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/empty-state"
import { PageHeader } from "@/components/page-header"
import { Slider } from "@/components/ui/slider"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "@/components/ui/dialog"
import { Calendar } from "@/components/ui/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn, formatBytes, formatCPU } from "@/lib/utils"
import {
  CheckCircle, AlertCircle, Clock, Zap, Cpu, MemoryStick, RotateCcw,
  TrendingUp, TrendingDown, AlertTriangle, Pencil, BrainCircuit,
  Search, Filter, ChevronDown, X, MoreVertical, Settings, ChevronRight,
  CalendarClock, Calendar as CalendarIcon, Trash2, Database, HardDrive,
} from "lucide-react"
import { toast } from "sonner"
import { useState, useMemo } from "react"
import { useNavigate } from "react-router-dom"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuLabel } from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { useFilterStore } from "@/stores/filter-store"
import { format, parseISO } from "date-fns"

export interface ResourceValues {
  cpu_limit: number
  mem_limit: number
  cpu_reservation: number
  mem_reservation: number
}

interface MemoryTrend {
  slope_bytes_per_hour: number
  daily_growth_mb: number
  r_squared: number
  has_leak: boolean
}

interface Forecast {
  days_ahead: number
  projected_mem: number
  projected_mem_p99: number
  slope_bytes_per_hour: number
}

export interface Recommendation {
  service: string
  samples: number
  status: string
  stack?: string
  preset?: string
  current?: ResourceValues
  outliers_removed?: number
  cpu?: { p50: number; p95: number }
  mem?: { p50: number; p99: number }
  oom_events?: number
  has_drift?: boolean
  pattern?: string
  memory_trend?: MemoryTrend
  forecast?: Forecast
  suggested?: ResourceValues
  suggested_apply_time?: string | null
  confidence?: string
}

interface Schedule {
  id: number
  service: string
  cpu_limit: number
  mem_limit: number
  cpu_reservation: number
  mem_reservation: number
  scheduled_at: string
  status: string
  applied_at: string | null
  error: string | null
  attempts: number
  created_at: string
}

const confidenceLabel = (conf?: string) => {
  if (conf === "high") return "Alta"
  if (conf === "medium") return "Média"
  return "Baixa"
}

const patternLabel = (p?: string) => {
  if (p === "business_hours") return "Business Hours"
  if (p === "constant") return "Constante"
  if (p === "batch") return "Batch"
  return "Desconhecido"
}

const statusConfig: Record<string, { icon: typeof AlertTriangle; border: string; label: string; labelClass: string }> = {
  unconfigured: { icon: Settings, border: "border-l-warning", label: "Sem configuração", labelClass: "text-warning" },
  alerted: { icon: AlertTriangle, border: "border-l-destructive", label: "Crítico", labelClass: "text-destructive" },
  under_provisioned: { icon: TrendingUp, border: "border-l-warning", label: "Atenção", labelClass: "text-warning" },
  over_provisioned: { icon: TrendingDown, border: "border-l-primary", label: "Otimizar", labelClass: "text-primary" },
  healthy: { icon: CheckCircle, border: "border-l-success", label: "Saudável", labelClass: "text-success" },
  collecting_data: { icon: Clock, border: "border-l-muted", label: "Coletando", labelClass: "text-muted-foreground" },
}

const getStatusConfig = (status: string) => statusConfig[status] ?? statusConfig.collecting_data


export function EditDialog({ rec, open, onOpenChange, onApply, applying }: {
  rec: Recommendation
  open: boolean
  onOpenChange: (open: boolean) => void
  onApply: (service: string, values: ResourceValues) => void
  applying: string | null
}) {
  const suggested = rec.suggested ?? { cpu_limit: 0, mem_limit: 0, cpu_reservation: 0, mem_reservation: 0 }
  const current = rec.current ?? { cpu_limit: 0, mem_limit: 0, cpu_reservation: 0, mem_reservation: 0 }
  const [edited, setEdited] = useState<ResourceValues>(suggested)

  const cpuMax = Math.max(suggested.cpu_limit * 2, current.cpu_limit * 2, 4)
  const memMax = Math.max(suggested.mem_limit * 2, current.mem_limit * 2, 512 * 1024 * 1024)
  const cpuResMax = Math.max(suggested.cpu_reservation * 2, current.cpu_reservation * 2, 2)
  const memResMax = Math.max(suggested.mem_reservation * 2, current.mem_reservation * 2, 256 * 1024 * 1024)

  const isModified = useMemo(() =>
    edited.cpu_limit !== suggested.cpu_limit ||
    edited.mem_limit !== suggested.mem_limit ||
    edited.cpu_reservation !== suggested.cpu_reservation ||
    edited.mem_reservation !== suggested.mem_reservation
  , [edited, suggested])

  const handleApply = () => {
    onApply(rec.service, edited)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Pencil className="h-4 w-4 text-primary" />
            {rec.service}
          </DialogTitle>
          <DialogDescription>
            Ajuste os valores antes de aplicar. Use os sliders para modificar.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-5">
          <div className="space-y-3">
            <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
              <Cpu className="h-3 w-3 text-primary" />
              CPU
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Limite</span>
                <span className="tabular-nums font-medium text-primary">
                  {edited.cpu_limit.toFixed(2)} cores
                </span>
              </div>
              <Slider
                value={[edited.cpu_limit]}
                min={0.1}
                max={cpuMax}
                step={0.05}
                onValueChange={(v) => setEdited({ ...edited, cpu_limit: v[0] })}
              />
              <div className="flex justify-between text-[10px] text-muted-foreground">
                <span>0.10</span>
                <span>{cpuMax.toFixed(2)}</span>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Reserva</span>
                <span className="tabular-nums font-medium text-primary">
                  {edited.cpu_reservation.toFixed(2)} cores
                </span>
              </div>
              <Slider
                value={[edited.cpu_reservation]}
                min={0.05}
                max={cpuResMax}
                step={0.05}
                onValueChange={(v) => setEdited({ ...edited, cpu_reservation: v[0] })}
              />
              <div className="flex justify-between text-[10px] text-muted-foreground">
                <span>0.05</span>
                <span>{cpuResMax.toFixed(2)}</span>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
              <MemoryStick className="h-3 w-3 text-chart-2" />
              Memória
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Limite</span>
                <span className="tabular-nums font-medium text-chart-2">
                  {formatBytes(edited.mem_limit)}
                </span>
              </div>
              <Slider
                value={[edited.mem_limit]}
                min={16 * 1024 * 1024}
                max={memMax}
                step={16 * 1024 * 1024}
                onValueChange={(v) => setEdited({ ...edited, mem_limit: v[0] })}
              />
              <div className="flex justify-between text-[10px] text-muted-foreground">
                <span>16 MB</span>
                <span>{formatBytes(memMax)}</span>
              </div>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Reserva</span>
                <span className="tabular-nums font-medium text-chart-2">
                  {formatBytes(edited.mem_reservation)}
                </span>
              </div>
              <Slider
                value={[edited.mem_reservation]}
                min={8 * 1024 * 1024}
                max={memResMax}
                step={8 * 1024 * 1024}
                onValueChange={(v) => setEdited({ ...edited, mem_reservation: v[0] })}
              />
              <div className="flex justify-between text-[10px] text-muted-foreground">
                <span>8 MB</span>
                <span>{formatBytes(memResMax)}</span>
              </div>
            </div>
          </div>
        </div>

        <DialogFooter className="gap-2">
          <Button variant="ghost" size="sm" onClick={() => setEdited(suggested)} disabled={!isModified}>
            <RotateCcw className="h-3 w-3" />
            Resetar
          </Button>
          <Button size="sm" onClick={handleApply} disabled={applying === rec.service}>
            <Zap className="h-3.5 w-3.5" />
            {applying === rec.service ? "Aplicando..." : "Aplicar"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function ScheduleDialog({ rec, open, onOpenChange, onScheduled }: {
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

function CompareRow({ label, currentVal, suggestedVal, formatVal }: {
  label: string
  currentVal: number
  suggestedVal: number
  formatVal: (v: number) => string
}) {
  const hasDiff = currentVal !== suggestedVal
  const pct = currentVal > 0 ? Math.round((suggestedVal / currentVal - 1) * 100) : null
  return (
    <div className="flex items-center justify-between py-1 text-xs">
      <span className="text-muted-foreground w-20">{label}</span>
      <span className="tabular-nums text-muted-foreground w-24 text-right">
        {currentVal > 0 ? formatVal(currentVal) : "—"}
      </span>
      <span className="text-muted-foreground/50">→</span>
      <span className="tabular-nums font-medium text-foreground w-24 text-right">
        {suggestedVal > 0 ? formatVal(suggestedVal) : "—"}
      </span>
      {hasDiff && pct !== null && (
        <span className={cn("text-[10px] font-medium w-12 text-right", pct > 0 ? "text-warning" : "text-chart-2")}>
          {pct > 0 ? "+" : ""}{pct}%
        </span>
      )}
      {!hasDiff && <span className="w-12" />}
    </div>
  )
}

export function RecommendationCard({ rec, onApply, applying, error, onRecalculate, recalculating, pendingSchedule, onSchedule, onCancelSchedule }: {
  rec: Recommendation
  onApply: (service: string, values: ResourceValues) => void
  applying: string | null
  error: string | null
  onRecalculate: (service: string) => void
  recalculating: string | null
  pendingSchedule?: Schedule | null
  onSchedule: () => void
  onCancelSchedule: (id: number) => void
}) {
  const navigate = useNavigate()
  const [editOpen, setEditOpen] = useState(false)
  const [scheduleOpen, setScheduleOpen] = useState(false)
  const suggested = rec.suggested ?? { cpu_limit: 0, mem_limit: 0, cpu_reservation: 0, mem_reservation: 0 }
  const current = rec.current ?? { cpu_limit: 0, mem_limit: 0, cpu_reservation: 0, mem_reservation: 0 }
  const sc = getStatusConfig(rec.status)
  const StatusIcon = sc.icon

  if (rec.status === "collecting_data" || !rec.suggested) {
    return (
      <Card className={cn("hover:bg-accent/30 transition-colors border-l-2", sc.border)}>
        <CardContent className="flex items-center gap-2 py-3">
          <StatusIcon className={cn("h-3.5 w-3.5", sc.labelClass)} />
          <button
            className="text-sm font-medium hover:text-primary transition-colors"
            onClick={() => navigate(`/services/${encodeURIComponent(rec.service)}`)}
          >
            {rec.service}
          </button>
          <span className="text-xs text-muted-foreground">Coletando ({rec.samples} amostras)</span>
        </CardContent>
      </Card>
    )
  }

  const hasOom = (rec.oom_events ?? 0) > 0
  const hasLeak = rec.memory_trend?.has_leak
  const hasDrift = rec.has_drift

  return (
    <>
      <Card className={cn(
        "border-l-2 hover:shadow-md transition-all",
        sc.border,
        hasOom && "animate-pulse-border"
      )}>
        <CardHeader className="pb-2 pt-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2 min-w-0">
              <StatusIcon className={cn("h-4 w-4 shrink-0", sc.labelClass)} />
              <button
                className="text-lg font-bold truncate hover:text-primary transition-colors"
                onClick={() => navigate(`/services/${encodeURIComponent(rec.service)}`)}
              >
                {rec.service}
              </button>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
                  <MoreVertical className="h-4 w-4 text-muted-foreground" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setEditOpen(true)} className="gap-2">
                  <Zap className="h-3.5 w-3.5" />
                  Aplicar agora
                </DropdownMenuItem>
                {pendingSchedule ? (
                  <DropdownMenuItem
                    onClick={() => onCancelSchedule(pendingSchedule.id)}
                    className="gap-2 text-destructive"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                    Cancelar agendamento
                  </DropdownMenuItem>
                ) : (
                  <DropdownMenuItem onClick={() => setScheduleOpen(true)} className="gap-2">
                    <CalendarClock className="h-3.5 w-3.5" />
                    Agendar
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  onClick={() => recalculating !== rec.service && onRecalculate(rec.service)}
                  className={cn("gap-2", recalculating === rec.service && "opacity-50 pointer-events-none")}
                >
                  <BrainCircuit className="h-3.5 w-3.5" />
                  {recalculating === rec.service ? "Gerando..." : "Recalcular"}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => navigate(`/services/${encodeURIComponent(rec.service)}`)}
                  className="gap-2"
                >
                  <ChevronRight className="h-3.5 w-3.5" />
                  Ver serviço
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <div className="flex items-center gap-2 text-[11px]">
            <span className={cn("font-medium", sc.labelClass)}>{sc.label}</span>
            <span className="text-muted-foreground/50">·</span>
            <span className="text-muted-foreground">{confidenceLabel(rec.confidence)}</span>
            {rec.pattern && rec.pattern !== "unknown" && (
              <>
                <span className="text-muted-foreground/50">·</span>
                <span className="text-muted-foreground">{patternLabel(rec.pattern)}</span>
              </>
            )}
            {pendingSchedule && (
              <>
                <span className="text-muted-foreground/50">·</span>
                <span className="text-warning flex items-center gap-0.5">
                  <CalendarClock className="h-3 w-3" />
                  Agendado: {format(parseISO(pendingSchedule.scheduled_at), "dd/MM HH:mm")}
                </span>
              </>
            )}
          </div>
        </CardHeader>

        <CardContent className="space-y-2 pt-0">
          {(hasOom || hasLeak || hasDrift) && (
            <div className="space-y-1">
              {hasOom && (
                <div className="flex items-center gap-1.5 text-[11px] text-destructive bg-destructive/10 rounded px-2 py-1">
                  <AlertTriangle className="h-3 w-3 shrink-0" />
                  <span>{rec.oom_events} OOM events nos últimos {settings_label}</span>
                </div>
              )}
              {hasLeak && (
                <div className="flex items-center gap-1.5 text-[11px] text-destructive bg-destructive/10 rounded px-2 py-1">
                  <AlertTriangle className="h-3 w-3 shrink-0" />
                  <span>
                    Memory leak: +{rec.memory_trend!.daily_growth_mb} MB/dia (R²={rec.memory_trend!.r_squared})
                    {rec.forecast && ` · Projeção ${rec.forecast.days_ahead}d: ${formatBytes(rec.forecast.projected_mem_p99)}`}
                  </span>
                </div>
              )}
              {hasDrift && !hasOom && !hasLeak && (
                <div className="flex items-center gap-1.5 text-[11px] text-warning bg-warning/10 rounded px-2 py-1">
                  <AlertTriangle className="h-3 w-3 shrink-0" />
                  <span>Resource drift detectado</span>
                </div>
              )}
            </div>
          )}

          <div className="grid grid-cols-2 gap-x-4 gap-y-0.5 py-1">
            <div className="text-[10px] text-muted-foreground/70 uppercase tracking-wide pb-1">Atual</div>
            <div className="text-[10px] text-muted-foreground/70 uppercase tracking-wide pb-1 text-right">Sugerido</div>
            <CompareRow
              label="CPU Lim"
              currentVal={current.cpu_limit}
              suggestedVal={suggested.cpu_limit}
              formatVal={(v) => v.toFixed(2)}
            />
            <CompareRow
              label="Mem Lim"
              currentVal={current.mem_limit}
              suggestedVal={suggested.mem_limit}
              formatVal={formatBytes}
            />
            <CompareRow
              label="CPU Res"
              currentVal={current.cpu_reservation}
              suggestedVal={suggested.cpu_reservation}
              formatVal={(v) => v.toFixed(2)}
            />
            <CompareRow
              label="Mem Res"
              currentVal={current.mem_reservation}
              suggestedVal={suggested.mem_reservation}
              formatVal={formatBytes}
            />
          </div>

          <div className="flex items-center gap-3 text-[10px] text-muted-foreground border-t pt-2">
            <span><span className="font-medium text-foreground">{rec.samples.toLocaleString()}</span> amostras</span>
            {rec.cpu && (
              <span>CPU P95 <span className="font-medium text-foreground tabular-nums">{formatCPU(rec.cpu.p95)}</span></span>
            )}
            {rec.mem && (
              <span>Mem P99 <span className="font-medium text-foreground tabular-nums">{formatBytes(rec.mem.p99)}</span></span>
            )}
          </div>

          {error && applying === rec.service && (
            <div className="flex items-center gap-1.5 text-[11px] text-destructive bg-destructive/10 rounded px-2 py-1">
              <AlertCircle className="h-3 w-3" />
              <span>Erro ao aplicar recomendação</span>
            </div>
          )}
        </CardContent>
      </Card>

      <EditDialog
        rec={rec}
        open={editOpen}
        onOpenChange={setEditOpen}
        onApply={onApply}
        applying={applying}
      />
      <ScheduleDialog
        rec={rec}
        open={scheduleOpen}
        onOpenChange={setScheduleOpen}
        onScheduled={onSchedule}
      />
    </>
  )
}

const settings_label = "7 dias"

interface StorageRec {
  type: string
  severity: string
  volume?: string
  size_bytes?: number
  reclaimable_bytes?: number
  message: string
  action?: string | null
  action_label?: string | null
  daily_growth_mb?: number
  r_squared?: number
  days_to_double?: number
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
  info: { icon: CheckCircle, color: "text-primary", bg: "bg-primary/10", border: "border-primary/30" },
}

const storageTypeIcon: Record<string, typeof Database> = {
  orphan_volume: Database,
  reclaimable_space: HardDrive,
  image_reclaimable: HardDrive,
  volume_growth: TrendingUp,
  large_volume: Database,
}

export default function Recommendations() {
  const queryClient = useQueryClient()
  const [applying, setApplying] = useState<string | null>(null)
  const [showHealthy, setShowHealthy] = useState(false)
  const [showStorage, setShowStorage] = useState(true)

  const { data: recs, isLoading } = useQuery<Recommendation[]>({
    queryKey: ["recommendations"],
    queryFn: () => api.get<Recommendation[]>("/recommendations"),
  })

  const { data: storageRecs } = useQuery<StorageAnalysis>({
    queryKey: ["storage-recommendations"],
    queryFn: () => api.get<StorageAnalysis>("/recommendations/storage"),
  })

  const { data: pendingSchedules } = useQuery<Schedule[]>({
    queryKey: ["schedules", "pending"],
    queryFn: () => api.get<Schedule[]>("/schedules/pending"),
    refetchInterval: 30000,
  })

  const recalculateMutation = useMutation({
    mutationFn: (service: string) =>
      api.post<Recommendation>(`/recommendations/${encodeURIComponent(service)}/recalculate`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recommendations"] })
      queryClient.invalidateQueries({ queryKey: ["triggers"] })
    },
  })

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

  const cancelScheduleMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/schedules/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedules"] })
    },
  })

  const handleApply = (service: string, values: ResourceValues) => {
    setApplying(service)
    applyMutation.mutate({ service, values })
  }

  const handleScheduled = () => {
    queryClient.invalidateQueries({ queryKey: ["schedules"] })
    toast.success("Agendamento criado com sucesso")
  }

  const handleCancelSchedule = (id: number) => {
    cancelScheduleMutation.mutate(id)
  }

  const pendingScheduleMap = useMemo(() => {
    const map: Record<string, Schedule> = {}
    for (const s of pendingSchedules ?? []) {
      map[s.service] = s
    }
    return map
  }, [pendingSchedules])

  const { recConfidence: confFilter, recEvents: eventFilterList, recStatus: statusFilter, setRecConfidence: setConfFilter, setRecEvents: setRecEventsList, setRecStatus: setStatusFilter } = useFilterStore()
  const eventFilters = new Set(eventFilterList)
  const [search, setSearch] = useState("")

  const toggleEvent = (event: string) => {
    const next = new Set(eventFilterList)
    if (next.has(event)) next.delete(event)
    else next.add(event)
    setRecEventsList(Array.from(next))
  }

  const setEventFilters = (s: Set<string>) => setRecEventsList(Array.from(s))

  const statusPriority = (r: Recommendation) => {
    if (r.status === "alerted") return 4
    if (r.status === "unconfigured") return 3
    if (r.status === "under_provisioned") return 2
    if (r.status === "over_provisioned") return 1
    if (r.status === "healthy") return 0
    return 0
  }

  const confidenceRank = (conf?: string) => conf === "high" ? 0 : conf === "medium" ? 1 : 2

  const statusCounts = (recs ?? []).reduce((acc: Record<string, number>, r) => {
    acc[r.status] = (acc[r.status] ?? 0) + 1
    return acc
  }, {} as Record<string, number>)

  const unconfiguredCount = statusCounts["unconfigured"] ?? 0
  const alertedCount = statusCounts["alerted"] ?? 0
  const underProvCount = statusCounts["under_provisioned"] ?? 0
  const overProvCount = statusCounts["over_provisioned"] ?? 0
  const healthyCount = statusCounts["healthy"] ?? 0
  const leakCount = (recs ?? []).filter((r) => r.memory_trend?.has_leak).length
  const oomCount = (recs ?? []).filter((r) => r.oom_events && r.oom_events > 0).length
  const driftCount = (recs ?? []).filter((r) => r.has_drift).length

  const filteredRecs = useMemo(() => {
    if (!recs) return []
    return recs.filter((r) => {
      if (search && !r.service.toLowerCase().includes(search.toLowerCase())) return false
      if (confFilter !== "all" && r.confidence !== confFilter) return false
      if (statusFilter !== "all" && r.status !== statusFilter) return false
      if (eventFilters.size > 0) {
        const matches: string[] = []
        if ((r.oom_events ?? 0) > 0) matches.push("oom")
        if (r.has_drift) matches.push("drift")
        if (r.memory_trend?.has_leak) matches.push("leak")
        if (!matches.some(m => eventFilters.has(m))) return false
      }
      return true
    })
  }, [recs, confFilter, statusFilter, eventFilterList, search])

  const sortedRecs = [...filteredRecs].sort((a, b) => {
    const aPri = statusPriority(a)
    const bPri = statusPriority(b)
    if (aPri !== bPri) return bPri - aPri
    return confidenceRank(a.confidence) - confidenceRank(b.confidence)
  })

  const confLabel = (c: string) => c === "high" ? "Alta" : c === "medium" ? "Média" : c === "low" ? "Baixa" : "Todas"
  const statusLabelMap: Record<string, string> = {
    all: "Todos",
    unconfigured: "Sem config",
    alerted: "Crítico",
    under_provisioned: "Atenção",
    over_provisioned: "Otimizar",
    healthy: "Saudável",
    collecting_data: "Coletando",
  }
  const activeEventFilters = Array.from(eventFilters)

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
      </div>
    )
  }

  if (!recs || recs.length === 0) {
    return (
      <div className="space-y-6">
        <PageHeader title="Recomendações" description="Sugestões de limites baseadas em dados" />
        <Card>
          <EmptyState icon={Clock} message="Nenhuma recomendação disponível. Aguardando coleta de dados..." />
        </Card>
      </div>
    )
  }

  const actionRecs = sortedRecs.filter(r => r.status !== "healthy" && r.status !== "collecting_data")
  const healthyRecs = sortedRecs.filter(r => r.status === "healthy")
  const collectingRecs = sortedRecs.filter(r => r.status === "collecting_data")

  return (
    <div className="space-y-6">
      <PageHeader title="Recomendações" description="Sugestões de limites baseadas em dados">
        <div className="flex gap-2 flex-wrap">
          {unconfiguredCount > 0 && <Badge variant="warning">{unconfiguredCount} sem config</Badge>}
          {alertedCount > 0 && <Badge variant="danger">{alertedCount} críticos</Badge>}
          {underProvCount > 0 && <Badge variant="warning">{underProvCount} atenção</Badge>}
          {overProvCount > 0 && <Badge variant="secondary">{overProvCount} otimizar</Badge>
          }
          {healthyCount > 0 && <Badge variant="success">{healthyCount} saudáveis</Badge>}
        </div>
      </PageHeader>

      {pendingSchedules && pendingSchedules.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CalendarClock className="h-4 w-4 text-warning" />
            <span className="font-medium">Agendamentos pendentes ({pendingSchedules.length})</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {pendingSchedules.map((s) => (
              <div key={s.id} className="flex items-center gap-2 rounded-md border border-warning/20 bg-warning/5 px-3 py-1.5 text-xs">
                <CalendarClock className="h-3 w-3 text-warning" />
                <span className="font-medium text-foreground">{s.service}</span>
                <span className="text-muted-foreground">{format(parseISO(s.scheduled_at), "dd/MM 'às' HH:mm")}</span>
                <button
                  className="text-muted-foreground hover:text-destructive transition-colors"
                  onClick={() => handleCancelSchedule(s.id)}
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="flex items-center gap-3 flex-wrap">
        <div className="relative flex-1 min-w-48">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar serviço..."
            className="pl-9"
          />
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-1.5">
              <Filter className="h-3.5 w-3.5" />
              Status
              {statusFilter !== "all" && (
                <Badge variant="secondary" className="text-[10px] px-1 py-0">{statusLabelMap[statusFilter] ?? statusFilter}</Badge>
              )}
              <ChevronDown className="h-3.5 w-3.5 opacity-50" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel>Status</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => setStatusFilter("all")}>
              {statusFilter === "all" && "✓ "}Todos
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => setStatusFilter("unconfigured")}>
              {statusFilter === "unconfigured" && "✓ "}Sem configuração ({unconfiguredCount})
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setStatusFilter("alerted")}>
              {statusFilter === "alerted" && "✓ "}Crítico ({alertedCount})
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setStatusFilter("under_provisioned")}>
              {statusFilter === "under_provisioned" && "✓ "}Atenção ({underProvCount})
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setStatusFilter("over_provisioned")}>
              {statusFilter === "over_provisioned" && "✓ "}Otimizar ({overProvCount})
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setStatusFilter("healthy")}>
              {statusFilter === "healthy" && "✓ "}Saudável ({healthyCount})
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-1.5">
              <Filter className="h-3.5 w-3.5" />
              Confiabilidade
              {confFilter !== "all" && (
                <Badge variant="secondary" className="text-[10px] px-1 py-0">{confLabel(confFilter)}</Badge>
              )}
              <ChevronDown className="h-3.5 w-3.5 opacity-50" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel>Confiabilidade</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => setConfFilter("all")}>
              {confFilter === "all" && "✓ "}Todas
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setConfFilter("high")}>
              {confFilter === "high" && "✓ "}Alta
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setConfFilter("medium")}>
              {confFilter === "medium" && "✓ "}Média
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setConfFilter("low")}>
              {confFilter === "low" && "✓ "}Baixa
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="gap-1.5">
              <Filter className="h-3.5 w-3.5" />
              Eventos
              {activeEventFilters.length > 0 && (
                <div className="flex items-center gap-1">
                  {activeEventFilters.map((ev) => (
                    <Badge key={ev} variant="secondary" className="text-[10px] px-1 py-0 capitalize">{ev}</Badge>
                  ))}
                </div>
              )}
              <ChevronDown className="h-3.5 w-3.5 opacity-50" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel>Eventos (múltipla seleção)</DropdownMenuLabel>
            <DropdownMenuItem onClick={() => toggleEvent("oom")}>
              {eventFilters.has("oom") && "✓ "}OOM ({oomCount})
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => toggleEvent("drift")}>
              {eventFilters.has("drift") && "✓ "}Drift ({driftCount})
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => toggleEvent("leak")}>
              {eventFilters.has("leak") && "✓ "}Leak ({leakCount})
            </DropdownMenuItem>
            {activeEventFilters.length > 0 && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setEventFilters(new Set())}>
                  Limpar filtros
                </DropdownMenuItem>
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>

        {(search || confFilter !== "all" || eventFilters.size > 0 || statusFilter !== "all") && (
          <Button
            variant="ghost"
            size="sm"
            className="gap-1.5 text-muted-foreground"
            onClick={() => { setSearch(""); setConfFilter("all"); setEventFilters(new Set()); setStatusFilter("all") }}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        )}

        <span className="text-xs text-muted-foreground ml-auto">
          {sortedRecs.length} de {recs.length} recomendações
        </span>
      </div>

      {actionRecs.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {actionRecs.map((rec) => (
            <RecommendationCard
              key={rec.service}
              rec={rec}
              onApply={handleApply}
              applying={applying}
              error={applyMutation.isError ? rec.service : null}
              onRecalculate={(service) => recalculateMutation.mutate(service)}
              recalculating={recalculateMutation.isPending ? recalculateMutation.variables ?? null : null}
              pendingSchedule={pendingScheduleMap[rec.service] ?? null}
              onSchedule={handleScheduled}
              onCancelSchedule={handleCancelSchedule}
            />
          ))}
        </div>
      )}

      {collectingRecs.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs text-muted-foreground font-medium">
            Aguardando coleta ({collectingRecs.length})
          </p>
          <div className="grid gap-2 md:grid-cols-3 lg:grid-cols-4">
            {collectingRecs.map((rec) => (
              <RecommendationCard
                key={rec.service}
                rec={rec}
                onApply={handleApply}
                applying={applying}
                error={null}
                onRecalculate={(service) => recalculateMutation.mutate(service)}
                recalculating={null}
                pendingSchedule={pendingScheduleMap[rec.service] ?? null}
                onSchedule={handleScheduled}
                onCancelSchedule={handleCancelSchedule}
              />
            ))}
          </div>
        </div>
      )}

      {healthyRecs.length > 0 && (
        <div className="space-y-3">
          <button
            className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
            onClick={() => setShowHealthy(!showHealthy)}
          >
            <ChevronDown className={cn("h-4 w-4 transition-transform", !showHealthy && "-rotate-90")} />
            <CheckCircle className="h-4 w-4 text-success" />
            <span className="font-medium">{healthyCount} serviços saudáveis</span>
            <span className="text-xs text-muted-foreground/70">
              {showHealthy ? "ocultar" : "ver recomendações"}
            </span>
          </button>
          {showHealthy && (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {healthyRecs.map((rec) => (
                <RecommendationCard
                  key={rec.service}
                  rec={rec}
                  onApply={handleApply}
                  applying={applying}
                  error={applyMutation.isError ? rec.service : null}
                  onRecalculate={(service) => recalculateMutation.mutate(service)}
                  recalculating={recalculateMutation.isPending ? recalculateMutation.variables ?? null : null}
                  pendingSchedule={pendingScheduleMap[rec.service] ?? null}
                  onSchedule={handleScheduled}
                  onCancelSchedule={handleCancelSchedule}
                />
              ))}
            </div>
          )}
        </div>
      )}

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

      {actionRecs.length === 0 && collectingRecs.length === 0 && healthyRecs.length === 0 && (!storageRecs || storageRecs.recommendations.length === 0) && (
        <Card>
          <EmptyState icon={CheckCircle} message="Nenhuma recomendação encontrada com os filtros atuais." />
        </Card>
      )}
    </div>
  )
}
