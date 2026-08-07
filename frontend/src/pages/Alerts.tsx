import { Fragment, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { useNavigate, Link } from "react-router-dom"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { useRefreshTimer } from "@/hooks/use-refresh"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/empty-state"
import { Badge } from "@/components/ui/badge"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { PageHeader } from "@/components/page-header"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { HelpIcon } from "@/components/help-icon"
import { AlertTriangle, Droplets, Waves, AlertOctagon, Search, Filter, ChevronDown, ChevronRight, X, ShieldAlert, ArrowRight } from "lucide-react"
import { useFilterStore } from "@/stores/filter-store"
import { Input } from "@/components/ui/input"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel } from "@/components/ui/dropdown-menu"

interface AlertItem {
  type: "oom" | "leak" | "drift"
  severity: "critical" | "warning"
  service: string
  message: string
  details: Record<string, unknown>
  ts: string
}

interface AlertsResponse {
  alerts: AlertItem[]
  counts: {
    total: number
    oom: number
    leak: number
    drift: number
    critical: number
    warning: number
  }
}

// Thresholds de classificação de R² (aderência da regressão linear).
// Padrão consagrado em estatística: R²≥0.8 = forte, ≥0.5 = moderado, <0.5 = fraco.
// O backend usa os mesmos critérios implícitos ao decidir o que é leak.
const R2_STRONG = 0.8
const R2_MODERATE = 0.5

const TYPE_CONFIG: Record<
  string,
  { label: string; icon: typeof AlertTriangle; color: string; bg: string; help: { title: string; text: string } }
> = {
  oom: {
    label: "OOM",
    icon: AlertOctagon,
    color: "text-destructive",
    bg: "bg-destructive/15",
    help: {
      title: "OOM (Out of Memory)",
      text: "Evento factual: o kernel matou o container (exit 137) por falta de memória. Persistido em oom_events no DuckDB.",
    },
  },
  leak: {
    label: "Memory Leak",
    icon: Droplets,
    color: "text-chart-4",
    bg: "bg-chart-4/15",
    help: {
      title: "Memory Leak (inferência ML)",
      text: `Regressão linear sobre mem_usage detectou crescimento contínuo. R² indica aderência do ajuste (≥${R2_STRONG} = forte). Estado derivado — some quando a condição para.`,
    },
  },
  drift: {
    label: "Resource Drift",
    icon: Waves,
    color: "text-chart-3",
    bg: "bg-chart-3/15",
    help: {
      title: "Resource Drift (inferência ML)",
      text: "Diferença entre P50 e P95 do consumo indica padrão de uso diferente do baseline. Estado derivado — some quando a condição para.",
    },
  },
}

const SEVERITY_CONFIG: Record<
  string,
  { label: string; variant: "danger" | "warning"; dot: string }
> = {
  critical: { label: "Critical", variant: "danger", dot: "bg-red-500" },
  warning: { label: "Warning", variant: "warning", dot: "bg-orange-500" },
}

function toNum(v: unknown): number | null {
  if (typeof v === "number") return v
  if (typeof v === "string") {
    const n = parseFloat(v)
    return Number.isFinite(n) ? n : null
  }
  return null
}

function formatTimestamp(ts: string): string {
  if (!ts) return "—"
  try {
    const d = new Date(ts)
    return d.toLocaleString("pt-BR", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    })
  } catch {
    return ts
  }
}

// timeAgo — timestamp relativo ("há 5min"), padrão do Studio/RollbackWatches.
function timeAgo(ts: string): string {
  if (!ts) return "—"
  try {
    const d = new Date(ts)
    const diffMs = Date.now() - d.getTime()
    if (diffMs < 0) return formatTimestamp(ts)
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return "agora"
    if (diffMin < 60) return `há ${diffMin}min`
    const diffH = Math.floor(diffMin / 60)
    if (diffH < 24) return `há ${diffH}h`
    const diffD = Math.floor(diffH / 24)
    return `há ${diffD}d`
  } catch {
    return ts
  }
}

// alertKey — chave estável para React (não usa índice de array).
function alertKey(a: AlertItem, i: number): string {
  return `${a.type}-${a.service}-${a.ts || i}`
}

// renderDetails — renderiza o conteúdo de `details` por tipo de alerta,
// exibindo os campos que o backend já computa (oom_count, daily_growth_mb,
// r_squared, cpu_drift, mem_drift). Espelha o padrão do ExplainabilityPanel.
function renderDetails(alert: AlertItem): React.ReactNode {
  const d = alert.details ?? {}
  if (alert.type === "oom") {
    const count = toNum(d.oom_count)
    if (count == null) return null
    return (
      <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs">
        <span><span className="text-muted-foreground">OOMs: </span><span className="font-medium tabular-nums">{count}</span></span>
      </div>
    )
  }
  if (alert.type === "leak") {
    const growth = toNum(d.daily_growth_mb)
    const r2 = toNum(d.r_squared)
    return (
      <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs">
        {growth != null && (
          <span>
            <span className="text-muted-foreground">Crescimento: </span>
            <span className="font-medium tabular-nums">+{growth.toFixed(2)} MB/dia</span>
          </span>
        )}
        {r2 != null && (
          <span>
            <span className="text-muted-foreground">R²: </span>
            <span className="font-medium tabular-nums">{r2.toFixed(3)}</span>
            <span className="text-muted-foreground ml-1">({r2 >= R2_STRONG ? "forte" : r2 >= R2_MODERATE ? "moderado" : "fraco"})</span>
          </span>
        )}
      </div>
    )
  }
  if (alert.type === "drift") {
    const cpu = toNum(d.cpu_drift)
    const mem = toNum(d.mem_drift)
    return (
      <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs">
        {cpu != null && (
          <span>
            <span className="text-muted-foreground">CPU drift: </span>
            <span className="font-medium tabular-nums">{(cpu * 100).toFixed(0)}%</span>
          </span>
        )}
        {mem != null && (
          <span>
            <span className="text-muted-foreground">Mem drift: </span>
            <span className="font-medium tabular-nums">{(mem * 100).toFixed(0)}%</span>
          </span>
        )}
      </div>
    )
  }
  return null
}

export default function Alerts() {
  const navigate = useNavigate()
  const [search, setSearch] = useState("")
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set())
  const { alertsType: typeFilter, alertsSeverity: severityFilter, setAlertsType: setTypeFilter, setAlertsSeverity: setSeverityFilter } = useFilterStore()

  // SSE: invalida queries de alerts quando receber evento de events (OOMs
  // em tempo real + rollbacks/otimizações que mudam o estado de alertas) ou
  // metrics (coleta nova — leaks/drifts são inferências on-demand do ML).
  //
  // Antes assinávamos "dashboard" + "metrics", mas "dashboard" publica o
  // mesmo payload de "metrics" (buildDashboardData) — era redundante e abria
  // 2 conexões EventSource à toa. O tópico "events" publica OOMs pontuais
  // (collector.go) e rollbacks (execute.go / rollback_watch_handlers.go),
  // que são exatamente os eventos que mudam o feed de alertas críticos.
  useEventSource({
    topic: "events",
    invalidateQueries: [["alerts"]],
  })
  useEventSource({
    topic: "metrics",
    invalidateQueries: [["alerts"]],
  })

  // Reconciliação safety-net controlada pelo dropdown (Frente C).
  useRefreshTimer([["alerts"]])

  const { data, isLoading } = useQuery<AlertsResponse>({
    queryKey: ["alerts"],
    queryFn: () => api.get<AlertsResponse>("/alerts"),
    refetchInterval: false,
  })

  const toggleRow = (key: string) => {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid gap-4 md:grid-cols-4">
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
        </div>
        <Skeleton className="h-96" />
      </div>
    )
  }

  const alerts = data?.alerts ?? []
  const counts = data?.counts ?? { total: 0, oom: 0, leak: 0, drift: 0, critical: 0, warning: 0 }

  if (alerts.length === 0) {
    return (
      <div className="space-y-6">
        <PageHeader title="Alertas" description="OOMs, memory leaks e resource drifts detectados" />
        <Card>
          <EmptyState
            icon={ShieldAlert}
            message="Nenhum alerta ativo. Tudo sob controle."
            action={
              <Button asChild variant="outline" size="sm" className="gap-1.5">
                <Link to="/optimizations">
                  Ver recomendações
                  <ArrowRight className="h-3.5 w-3.5" />
                </Link>
              </Button>
            }
          />
        </Card>
      </div>
    )
  }

  const filteredAlerts = alerts.filter((a) => {
    if (search) {
      const q = search.toLowerCase()
      if (!a.service.toLowerCase().includes(q) && !a.message.toLowerCase().includes(q)) return false
    }
    if (typeFilter !== "all" && a.type !== typeFilter) return false
    if (severityFilter !== "all" && a.severity !== severityFilter) return false
    return true
  })

  // Stat cards por tipo (OOM/Leak/Drift) — alinha com os filtros de Tipo e
  // com o que o backend computa em counts. Antes misturávamos tipo (OOMs)
  // com severidade (Critical/Warning), quebrando a consistência.
  const statCards = [
    { label: "Total", value: counts.total, icon: AlertTriangle, iconBg: "bg-primary/15", iconColor: "text-primary", valueColor: "text-primary" },
    { label: "OOMs", value: counts.oom, icon: AlertOctagon, iconBg: "bg-destructive/15", iconColor: "text-destructive", valueColor: "text-destructive" },
    { label: "Leaks", value: counts.leak, icon: Droplets, iconBg: "bg-chart-4/15", iconColor: "text-chart-4", valueColor: "text-chart-4" },
    { label: "Drifts", value: counts.drift, icon: Waves, iconBg: "bg-chart-3/15", iconColor: "text-chart-3", valueColor: "text-chart-3" },
  ]

  return (
    <div className="space-y-6">
      <PageHeader title="Alertas" description="OOMs, memory leaks e resource drifts detectados">
        <div className="flex items-center gap-2">
          <Badge variant={counts.critical > 0 ? "danger" : "outline"} className="gap-1.5">
            <span className={`h-2 w-2 rounded-full ${counts.critical > 0 ? "bg-red-500" : "bg-muted-foreground"}`} />
            {counts.critical} critical
          </Badge>
          <Badge variant={counts.warning > 0 ? "warning" : "outline"} className="gap-1.5">
            <span className={`h-2 w-2 rounded-full ${counts.warning > 0 ? "bg-orange-500" : "bg-muted-foreground"}`} />
            {counts.warning} warning
          </Badge>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-4">
        {statCards.map((stat) => (
          <Card key={stat.label} className="hover:bg-accent/30 transition-colors">
            <CardContent className="flex flex-row items-center justify-between pb-2 pt-6">
              <div>
                <div className="text-sm font-medium text-muted-foreground">{stat.label}</div>
                <div className={`text-3xl font-bold tabular-nums ${stat.valueColor}`}>{stat.value}</div>
              </div>
              <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${stat.iconBg}`}>
                <stat.icon className={`h-4.5 w-4.5 ${stat.iconColor}`} />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="flex items-center gap-3 flex-wrap px-4 py-3 border-b">
            <div className="relative flex-1 min-w-48">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Buscar serviço ou mensagem..."
                className="pl-9"
              />
            </div>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  Tipo
                  {typeFilter !== "all" && (
                    <Badge variant="secondary" className="text-[10px] px-1 py-0">
                      {TYPE_CONFIG[typeFilter]?.label ?? typeFilter}
                    </Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Tipo</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setTypeFilter("all")}>
                  {typeFilter === "all" && "✓ "}Todos
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTypeFilter("oom")}>
                  {typeFilter === "oom" && "✓ "}OOM
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTypeFilter("leak")}>
                  {typeFilter === "leak" && "✓ "}Memory Leak
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTypeFilter("drift")}>
                  {typeFilter === "drift" && "✓ "}Resource Drift
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  Severidade
                  {severityFilter !== "all" && (
                    <Badge variant="secondary" className="text-[10px] px-1 py-0">
                      {SEVERITY_CONFIG[severityFilter]?.label ?? severityFilter}
                    </Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Severidade</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setSeverityFilter("all")}>
                  {severityFilter === "all" && "✓ "}Todas
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSeverityFilter("critical")}>
                  {severityFilter === "critical" && "✓ "}Critical
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setSeverityFilter("warning")}>
                  {severityFilter === "warning" && "✓ "}Warning
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {(search || typeFilter !== "all" || severityFilter !== "all") && (
              <Button
                variant="ghost"
                size="sm"
                className="gap-1.5 text-muted-foreground"
                onClick={() => { setSearch(""); setTypeFilter("all"); setSeverityFilter("all") }}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            )}

            <span className="text-xs text-muted-foreground ml-auto">
              {filteredAlerts.length} de {alerts.length} alertas
            </span>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8" />
                <TableHead className="w-28">Tipo</TableHead>
                <TableHead className="w-28">Severidade</TableHead>
                <TableHead>Serviço</TableHead>
                <TableHead>Mensagem</TableHead>
                <TableHead className="text-right">Timestamp</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredAlerts.map((alert, i) => {
                const typeCfg = TYPE_CONFIG[alert.type] ?? { label: alert.type, icon: AlertTriangle, color: "text-muted-foreground", bg: "bg-muted", help: { title: alert.type, text: "" } }
                const sevCfg = SEVERITY_CONFIG[alert.severity] ?? { label: alert.severity, variant: "outline" as const, dot: "bg-muted-foreground" }
                const key = alertKey(alert, i)
                const isExpanded = expandedRows.has(key)
                const details = renderDetails(alert)
                return (
                  <Fragment key={key}>
                    <TableRow
                      className="cursor-pointer hover:bg-accent/40 transition-colors duration-150"
                      onClick={() => details && toggleRow(key)}
                    >
                      <TableCell className="w-8">
                        {details ? (
                          isExpanded
                            ? <ChevronDown className="h-4 w-4 text-muted-foreground" />
                            : <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        ) : null}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <div className={`flex h-7 w-7 items-center justify-center rounded-md ${typeCfg.bg}`}>
                            <typeCfg.icon className={`h-3.5 w-3.5 ${typeCfg.color}`} />
                          </div>
                          <span className="text-xs font-medium">{typeCfg.label}</span>
                          <HelpIcon title={typeCfg.help.title} text={typeCfg.help.text} />
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={sevCfg.variant} className="gap-1.5">
                          <span className={`h-2 w-2 rounded-full ${sevCfg.dot}`} />
                          {sevCfg.label}
                        </Badge>
                      </TableCell>
                      <TableCell
                        className="font-medium"
                        onClick={(e) => { e.stopPropagation(); alert.service && navigate(`/services/${encodeURIComponent(alert.service)}`) }}
                      >
                        <span className="hover:underline">{alert.service || "—"}</span>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{alert.message}</TableCell>
                      <TableCell className="text-right text-xs text-muted-foreground font-mono whitespace-nowrap">
                        <TooltipProvider delayDuration={200}>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="cursor-help">{timeAgo(alert.ts)}</span>
                            </TooltipTrigger>
                            <TooltipContent side="left">
                              <p>{formatTimestamp(alert.ts)}</p>
                            </TooltipContent>
                          </Tooltip>
                        </TooltipProvider>
                      </TableCell>
                    </TableRow>
                    {isExpanded && details && (
                      <TableRow key={`${key}-detail`} className="bg-muted/30">
                        <TableCell colSpan={6} className="py-3 px-6">
                          {details}
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                )
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
