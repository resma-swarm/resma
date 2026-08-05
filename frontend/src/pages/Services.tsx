import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "react-router-dom"
import { api } from "@/api/client"
import { useRefreshInterval } from "@/hooks/use-refresh"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/empty-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { Input } from "@/components/ui/input"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator } from "@/components/ui/dropdown-menu"
import { RiskScoreGauge, type RiskFactors } from "@/components/risk-score-gauge"
import { StatusBadge, STATUS_CONFIG } from "@/components/status-badge"
import { PageHeader } from "@/components/page-header"
import { formatBytes, formatCPU } from "@/lib/utils"
import { toast } from "sonner"
import { Boxes, ChevronRight, MoreVertical, Archive, Search, Filter, ChevronDown, X } from "lucide-react"
import { useFilterStore } from "@/stores/filter-store"

interface ServiceResources {
  cpu_limit: number
  mem_limit: number
  cpu_reservation: number
  mem_reservation: number
}

interface Service {
  name: string
  container_count: number
  active_containers: number
  cpu_p95: number
  mem_p99: number
  last_seen: string | null
  status: string
  current?: ServiceResources
  risk_score?: number
  risk_factors?: RiskFactors
}

function hasConfig(r?: ServiceResources): boolean {
  return !!(r && (r.cpu_limit > 0 || r.mem_limit > 0 || r.cpu_reservation > 0 || r.mem_reservation > 0))
}

function ServiceConfigCell({ resources }: { resources?: ServiceResources }) {
  if (!hasConfig(resources)) {
    return <span className="text-sm text-muted-foreground">—</span>
  }
  const r = resources!
  return (
    <div className="space-y-0.5">
      <div className="text-[10px] text-muted-foreground/70 uppercase tracking-wide">
        CPU / Mem
      </div>
      <div className="flex items-center gap-1.5 text-[11px]">
        <span className="text-muted-foreground">Lim:</span>
        <span className="tabular-nums text-foreground">
          {r.cpu_limit > 0 ? formatCPU(r.cpu_limit) : "—"}
          <span className="text-muted-foreground mx-0.5">/</span>
          {r.mem_limit > 0 ? formatBytes(r.mem_limit) : "—"}
        </span>
      </div>
      <div className="flex items-center gap-1.5 text-[11px]">
        <span className="text-muted-foreground">Res:</span>
        <span className="tabular-nums text-foreground">
          {r.cpu_reservation > 0 ? formatCPU(r.cpu_reservation) : "—"}
          <span className="text-muted-foreground mx-0.5">/</span>
          {r.mem_reservation > 0 ? formatBytes(r.mem_reservation) : "—"}
        </span>
      </div>
    </div>
  )
}

export default function Services() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState("")
  const { servicesStatus: statusFilter, servicesAlert: alertFilter, servicesConfig: configFilter, setServicesStatus: setStatusFilter, setServicesAlert: setAlertFilter, setServicesConfig: setConfigFilter } = useFilterStore()

  const refreshInterval = useRefreshInterval()

  // SSE: invalida queries de services quando receber evento
  const { isConnected: sseConnected } = useEventSource({
    topic: "services",
    invalidateQueries: [["services"], ["recs-summary"]],
  })

  // SSE ativo = zero polling (SSE publica payload completo + reconciliação 30s)
  const fallbackInterval = sseConnected ? false : refreshInterval

  const { data: services, isLoading } = useQuery<Service[]>({
    queryKey: ["services"],
    queryFn: () => api.get<Service[]>("/services"),
    refetchInterval: fallbackInterval,
  })

  const { data: recs } = useQuery<Record<string, { oom_events?: number; has_drift?: boolean; memory_trend?: { has_leak: boolean } }>>({
    queryKey: ["recs-summary"],
    queryFn: async () => {
      const all = await api.get<Array<{ service: string; oom_events?: number; has_drift?: boolean; memory_trend?: { has_leak: boolean } }>>("/recommendations")
      const map: Record<string, { oom_events?: number; has_drift?: boolean; memory_trend?: { has_leak: boolean } }> = {}
      for (const r of all || []) map[r.service] = r
      return map
    },
    refetchInterval: fallbackInterval,
  })

  const archiveMutation = useMutation({
    mutationFn: (name: string) => api.patch(`/services/${encodeURIComponent(name)}/archive`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["services"] })
      toast.success("Serviço arquivado")
    },
    onError: () => toast.error("Erro ao arquivar serviço"),
  })

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-96" />
      </div>
    )
  }

  if (!services || services.length === 0) {
    return (
      <div className="space-y-6">
        <PageHeader title="Serviços" description="Serviços monitorados pelo RESMA" />
        <Card>
          <EmptyState icon={Boxes} message="Nenhum serviço monitorado. Aguardando coleta de dados..." />
        </Card>
      </div>
    )
  }

  const maxCpu = Math.max(...services.map((s) => s.cpu_p95), 1)
  const maxMem = Math.max(...services.map((s) => s.mem_p99), 1)

  const getAlertType = (name: string) => {
    const r = recs?.[name]
    if (!r) return "none"
    if ((r.oom_events ?? 0) > 0) return "oom"
    if (r.has_drift) return "drift"
    if (r.memory_trend?.has_leak) return "leak"
    return "none"
  }

  const filteredServices = services.filter((s) => {
    if (search) {
      const q = search.toLowerCase()
      if (!s.name.toLowerCase().includes(q)) return false
    }
    if (statusFilter !== "all" && s.status !== statusFilter) return false
    if (configFilter === "configured" && !hasConfig(s.current)) return false
    if (configFilter === "unconfigured" && hasConfig(s.current)) return false
    if (alertFilter !== "all") {
      const alertType = getAlertType(s.name)
      if (alertFilter === "none" && alertType !== "none") return false
      if (alertFilter !== "none" && alertType !== alertFilter) return false
    }
    return true
  })

  const getCpuColor = (val: number) => {
    if (val > 80) return "bg-destructive"
    if (val > 50) return "bg-warning"
    return "bg-primary"
  }

  const getMemColor = (val: number, max: number) => {
    const pct = (val / max) * 100
    if (pct > 80) return "bg-destructive"
    if (pct > 50) return "bg-warning"
    return "bg-chart-2"
  }

  const getContainerBadgeClass = (count: number) => {
    if (count > 5) return "border-destructive/40 text-destructive"
    if (count > 2) return "border-warning/40 text-warning"
    return "border-chart-2/40 text-chart-2"
  }

  return (
    <div className="space-y-6">
      <PageHeader title="Serviços" description="Serviços monitorados pelo RESMA">
        <Badge variant="outline">{services.length} serviços</Badge>
      </PageHeader>

      <Card>
        <CardContent className="p-0">
          <div className="flex items-center gap-3 flex-wrap px-4 py-3 border-b">
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
                    <Badge variant="secondary" className="text-[10px] px-1 py-0">{STATUS_CONFIG[statusFilter]?.label ?? statusFilter}</Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Status</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setStatusFilter("all")}>
                  {statusFilter === "all" && "✓ "}Todos
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("online")}>
                  {statusFilter === "online" && "✓ "}Online
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("offline")}>
                  {statusFilter === "offline" && "✓ "}Offline
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("legado")}>
                  {statusFilter === "legado" && "✓ "}Legado
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  Alertas
                  {alertFilter !== "all" && (
                    <Badge variant="secondary" className="text-[10px] px-1 py-0">{alertFilter === "none" ? "Sem alertas" : alertFilter === "oom" ? "OOM" : alertFilter === "drift" ? "Drift" : "Leak"}</Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Alertas</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setAlertFilter("all")}>
                  {alertFilter === "all" && "✓ "}Todos
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setAlertFilter("oom")}>
                  {alertFilter === "oom" && "✓ "}Com OOM
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setAlertFilter("drift")}>
                  {alertFilter === "drift" && "✓ "}Com Drift
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setAlertFilter("leak")}>
                  {alertFilter === "leak" && "✓ "}Com Leak
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => setAlertFilter("none")}>
                  {alertFilter === "none" && "✓ "}Sem alertas
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  Config
                  {configFilter !== "all" && (
                    <Badge variant="secondary" className="text-[10px] px-1 py-0">{configFilter === "configured" ? "Configurado" : "Sem config"}</Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Configuração</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setConfigFilter("all")}>
                  {configFilter === "all" && "✓ "}Todos
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setConfigFilter("configured")}>
                  {configFilter === "configured" && "✓ "}Configurado
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setConfigFilter("unconfigured")}>
                  {configFilter === "unconfigured" && "✓ "}Sem configuração
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {(search || statusFilter !== "all" || alertFilter !== "all" || configFilter !== "all") && (
              <Button
                variant="ghost"
                size="sm"
                className="gap-1.5 text-muted-foreground"
                onClick={() => { setSearch(""); setStatusFilter("all"); setAlertFilter("all"); setConfigFilter("all") }}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            )}

            <span className="text-xs text-muted-foreground ml-auto">
              {filteredServices.length} de {services.length} serviços
            </span>
          </div>

          <Table className="[&_th:not(:first-child)]:border-l [&_td:not(:first-child)]:border-l [&_th:not(:first-child)]:border-border/40 [&_td:not(:first-child)]:border-border/40">
            <TableHeader>
              <TableRow>
                <TableHead>Serviço</TableHead>
                <TableHead className="text-center">Status</TableHead>
                <TableHead className="text-center">Containers (ativos/total)</TableHead>
                <TableHead className="w-48 text-right">CPU P95</TableHead>
                <TableHead className="w-48 text-right">Memória P99</TableHead>
                <TableHead className="w-28 text-center">Risco</TableHead>
                <TableHead className="w-40">Config. Atual</TableHead>
                <TableHead className="w-20"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredServices.map((s) => (
                <TableRow
                  key={s.name}
                  className="cursor-pointer hover:bg-accent/40 transition-colors duration-150"
                  onClick={() => navigate(`/services/${encodeURIComponent(s.name)}`)}
                >
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-1.5">
                      {s.name}
                      {(() => {
                        const r = recs?.[s.name]
                        if (!r) return null
                        const hasOom = (r.oom_events ?? 0) > 0
                        const hasDrift = r.has_drift
                        const hasLeak = r.memory_trend?.has_leak
                        if (!hasOom && !hasDrift && !hasLeak) return null
                        return (
                          <span className="inline-flex items-center gap-0.5">
                            {hasOom && <Badge variant="destructive" className="text-[9px] px-1 py-0">OOM</Badge>}
                            {hasDrift && <Badge variant="warning" className="text-[9px] px-1 py-0">Drift</Badge>}
                            {hasLeak && <Badge variant="danger" className="text-[9px] px-1 py-0">Leak</Badge>}
                          </span>
                        )
                      })()}
                    </div>
                  </TableCell>
                  <TableCell className="text-center">
                    <StatusBadge status={s.status} lastSeen={s.last_seen} />
                  </TableCell>
                  <TableCell className="text-center">
                    <div className="flex items-center justify-center gap-1.5">
                      <Badge variant="outline" className={getContainerBadgeClass(s.active_containers)}>{s.active_containers}</Badge>
                      {s.container_count !== s.active_containers && (
                        <span className="text-xs text-muted-foreground">/ {s.container_count}</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-3">
                      <Progress
                        value={s.cpu_p95}
                        max={maxCpu}
                        indicatorClassName={getCpuColor(s.cpu_p95)}
                        className="w-24 shrink-0"
                      />
                      <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap shrink-0 w-14 text-right">
                        {formatCPU(s.cpu_p95)}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-3">
                      <Progress
                        value={s.mem_p99}
                        max={maxMem}
                        indicatorClassName={getMemColor(s.mem_p99, maxMem)}
                        className="w-24 shrink-0"
                      />
                      <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap shrink-0 w-20 text-right">
                        {formatBytes(s.mem_p99)}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-center">
                    <RiskScoreGauge score={s.risk_score ?? 0} factors={s.risk_factors} orientation="horizontal" />
                  </TableCell>
                  <TableCell>
                    <ServiceConfigCell resources={s.current} />
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <div className="flex items-center gap-1">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                            <MoreVertical className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => archiveMutation.mutate(s.name)}>
                            <Archive className="h-4 w-4" />
                            Arquivar
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
