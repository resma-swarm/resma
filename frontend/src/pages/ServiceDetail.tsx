import { useState, useEffect } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useParams, useNavigate } from "react-router-dom"
import { api } from "@/api/client"
import { useRefreshInterval } from "@/hooks/use-refresh"
import { useFilterStore } from "@/stores/filter-store"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { EmptyState } from "@/components/empty-state"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { PageHeader } from "@/components/page-header"
import { formatBytes, formatCPU } from "@/lib/utils"
import { ArrowLeft, Cpu, MemoryStick, AlertTriangle, TrendingUp, Box, ChevronRight, Filter, Zap, CalendarClock, History, XCircle, ListTodo } from "lucide-react"
import { RecommendationCard, type Recommendation, type ResourceValues } from "./Recommendations"
import { StatusBadge } from "@/components/status-badge"
import { RiskScoreGauge, type RiskFactors } from "@/components/risk-score-gauge"
import { toast } from "sonner"
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, ResponsiveContainer,
} from "recharts"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem } from "@/components/ui/dropdown-menu"

interface ServiceStats {
  service: string
  samples: number
  cpu_p50: number
  cpu_p95: number
  cpu_min: number
  cpu_max: number
  cpu_avg: number
  mem_p50: number
  mem_p99: number
  mem_min: number
  mem_max: number
  mem_avg: number
}

interface MetricPoint {
  ts: string
  cpu_percent: number
  mem_usage: number
  mem_limit: number
  mem_percent: number
}

interface ContainerStats {
  container_id: string
  samples: number
  cpu_p50: number
  cpu_p95: number
  cpu_min: number
  cpu_max: number
  cpu_avg: number
  mem_p50: number
  mem_p99: number
  mem_min: number
  mem_max: number
  mem_avg: number
  last_seen: string | null
  status: "online" | "offline" | "legado"
  networks: Record<string, string>
}


interface ServiceSchedule {
  id: number
  service: string
  cpu_limit: number | null
  mem_limit: number | null
  cpu_reservation: number | null
  mem_reservation: number | null
  scheduled_at: string
  status: string
  applied_at: string | null
  error: string | null
  attempts: number
  created_at: string
}

interface ServiceChangeLog {
  id: number
  service: string
  action: string
  source: string
  schedule_id: number | null
  cpu_limit_before: number | null
  mem_limit_before: number | null
  cpu_reservation_before: number | null
  mem_reservation_before: number | null
  cpu_limit_after: number | null
  mem_limit_after: number | null
  cpu_reservation_after: number | null
  mem_reservation_after: number | null
  user: string | null
  status: string
  error: string | null
  docker_response: string | null
  created_at: string
}

// Fase 7 — Task lifecycle
interface TaskRow {
  task_id: string
  service: string
  node_id: string
  node_hostname: string
  slot: number
  status: string
  desired_state: string
  created_at: string | null
  updated_at: string | null
}

interface ServiceHealthRow {
  service: string
  tasks_running: number
  tasks_failed: number
  tasks_pending: number
  restarts: number
  last_restart_at: string | null
}

const taskStatusConfig: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline" | "success" | "warning"; dot: string }> = {
  running: { label: "Running", variant: "success", dot: "bg-green-500" },
  pending: { label: "Pending", variant: "secondary", dot: "bg-yellow-500" },
  failed: { label: "Failed", variant: "destructive", dot: "bg-red-500" },
  rejected: { label: "Rejected", variant: "destructive", dot: "bg-red-500" },
  complete: { label: "Complete", variant: "outline", dot: "bg-green-500" },
  shutdown: { label: "Shutdown", variant: "outline", dot: "bg-muted-foreground" },
}

const scheduleStatusConfig: Record<string, { label: string; variant: "success" | "secondary" | "destructive" | "outline" | "warning" }> = {
  pending: { label: "Pendente", variant: "warning" },
  running: { label: "Executando", variant: "secondary" },
  completed: { label: "Concluído", variant: "success" },
  failed: { label: "Falhou", variant: "destructive" },
  cancelled: { label: "Cancelado", variant: "outline" },
}

function fmtDate(ts: string | null): string {
  if (!ts) return "—"
  return new Date(ts).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" })
}

function fmtVal(v: number | null, fmt: "cpu" | "mem"): string {
  if (v === null || v === undefined) return "—"
  return fmt === "cpu" ? formatCPU(v) : formatBytes(v)
}


const statusBadge: Record<string, { label: string; variant: "success" | "secondary" | "outline" }> = {
  online: { label: "Online", variant: "success" },
  offline: { label: "Offline", variant: "secondary" },
  legado: { label: "Legado", variant: "outline" },
}

const chartTooltipStyle = {
  backgroundColor: "var(--popover)",
  border: "1px solid var(--border)",
  borderRadius: "0.5rem",
  fontSize: "12px",
  color: "var(--popover-foreground)",
}

export default function ServiceDetail() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const serviceName = decodeURIComponent(name || "")

  const refreshInterval = useRefreshInterval()

  // SSE: assina o tópico "service-detail/{name}" — o collector publica o
  // payload completo (stats+metrics+containers+tasks+health) a cada coleta
  // quando há subscriber ativo. O applySSEPayload faz setQueryData para
  // cada query do ServiceDetail — zero refetch HTTP.
  // Fallback: se SSE cair, polling normal (refreshInterval).
  const sseTopic = `service-detail/${serviceName}`
  const { isConnected: sseConnected } = useEventSource({
    topic: sseTopic,
    enabled: !!serviceName,
  })
  // Quando SSE está ativo, não precisamos de polling — o SSE pusha os dados.
  // Fallback para polling apenas se SSE cair.
  const fallbackInterval = sseConnected ? false : refreshInterval

  const { data: stats, isLoading: statsLoading } = useQuery<ServiceStats>({
    queryKey: ["service-stats", serviceName],
    queryFn: () => api.get<ServiceStats>(`/services/${encodeURIComponent(serviceName)}/stats`),
    enabled: !!serviceName,
    refetchInterval: fallbackInterval,
  })

  const { data: serviceSchedules } = useQuery<ServiceSchedule[]>({
    queryKey: ["schedules", serviceName],
    queryFn: () => api.get<ServiceSchedule[]>(`/schedules?service=${encodeURIComponent(serviceName)}`),
    enabled: !!serviceName,
    refetchInterval: 30000,
  })

  const { data: serviceChangeLog } = useQuery<ServiceChangeLog[]>({
    queryKey: ["change-log", serviceName],
    queryFn: () => api.get<ServiceChangeLog[]>(`/change-log/${encodeURIComponent(serviceName)}`),
    enabled: !!serviceName,
    refetchInterval: 30000,
  })

  // Fase 7 — tasks do serviço (Swarm task lifecycle)
  const { data: serviceTasks } = useQuery<TaskRow[]>({
    queryKey: ["service-tasks", serviceName],
    queryFn: () => api.get<TaskRow[]>(`/tasks/${encodeURIComponent(serviceName)}`),
    enabled: !!serviceName,
    refetchInterval: fallbackInterval,
  })

  // Fase 7 — health do serviço (restarts, tasks running/failed)
  const { data: serviceHealth } = useQuery<ServiceHealthRow[]>({
    queryKey: ["services-health"],
    queryFn: () => api.get<ServiceHealthRow[]>("/services/health"),
    refetchInterval: fallbackInterval,
  })

  const [chartAnimated, setChartAnimated] = useState(false)
  const { serviceContainerFilter: containerFilter, setServiceContainerFilter: setContainerFilter } = useFilterStore()

  const { data: metrics } = useQuery<MetricPoint[]>({
    queryKey: ["service-metrics", serviceName],
    queryFn: () => api.get<MetricPoint[]>(`/services/${encodeURIComponent(serviceName)}/metrics`),
    enabled: !!serviceName,
    refetchInterval: fallbackInterval,
  })

  const { data: containers } = useQuery<ContainerStats[]>({
    queryKey: ["service-containers", serviceName],
    queryFn: () => api.get<ContainerStats[]>(`/services/${encodeURIComponent(serviceName)}/containers`),
    enabled: !!serviceName,
    refetchInterval: fallbackInterval,
  })

  const queryClient = useQueryClient()

  const cancelScheduleMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/schedules/${id}`),
    onSuccess: () => {
      toast.success("Agendamento cancelado")
      queryClient.invalidateQueries({ queryKey: ["schedules"] })
    },
    onError: () => toast.error("Erro ao cancelar agendamento"),
  })

  const { data: recommendation, isLoading: recLoading } = useQuery<Recommendation | null>({
    queryKey: ["service-recommendation", serviceName],
    queryFn: async () => {
      try {
        return await api.get<Recommendation>(`/recommendations/${encodeURIComponent(serviceName)}`)
      } catch {
        return null
      }
    },
    enabled: !!serviceName,
    refetchInterval: fallbackInterval,
  })

  // Overview (status + risk_score) — alimentado via SSE pelo collector
  const { data: overview } = useQuery<{
    status: string
    last_seen: string | null
    risk_score: number
    risk_factors: RiskFactors
  }>({
    queryKey: ["service-overview", serviceName],
    queryFn: async () => {
      try {
        // Tenta obter da lista de serviços (fallback inicial antes do SSE)
        const services = await api.get<Array<{ name: string; status: string; last_seen: string | null; risk_score?: number; risk_factors?: RiskFactors }>>("/services")
        const found = services.find((s) => s.name === serviceName)
        if (found) {
          return {
            status: found.status,
            last_seen: found.last_seen,
            risk_score: found.risk_score ?? 0,
            risk_factors: found.risk_factors ?? { oom_count: 0, has_leak: false, has_drift: false, mem_limit: 0 },
          }
        }
      } catch {
        // ignore
      }
      return { status: "legado", last_seen: null, risk_score: 0, risk_factors: { oom_count: 0, has_leak: false, has_drift: false, mem_limit: 0 } }
    },
    enabled: !!serviceName,
    refetchInterval: fallbackInterval,
  })

  const [applying, setApplying] = useState<string | null>(null)
  const [recalculating, setRecalculating] = useState<string | null>(null)

  const applyMutation = useMutation({
    mutationFn: ({ service, values }: { service: string; values: ResourceValues }) =>
      api.post(`/recommendations/${encodeURIComponent(service)}/apply`, values),
    onSuccess: () => {
      toast.success("Recursos aplicados com sucesso")
      setApplying(null)
      queryClient.invalidateQueries({ queryKey: ["service-recommendation", serviceName] })
    },
    onError: () => {
      toast.error("Erro ao aplicar recursos")
      setApplying(null)
    },
  })

  const recalcMutation = useMutation({
    mutationFn: (service: string) =>
      api.post(`/recommendations/${encodeURIComponent(service)}/recalculate`),
    onSuccess: () => {
      toast.success("Recomendação recalculada")
      setRecalculating(null)
      queryClient.invalidateQueries({ queryKey: ["service-recommendation", serviceName] })
    },
    onError: () => {
      toast.error("Erro ao recalcular")
      setRecalculating(null)
    },
  })

  const handleApply = (service: string, values: ResourceValues) => {
    setApplying(service)
    applyMutation.mutate({ service, values })
  }

  const handleRecalculate = (service: string) => {
    setRecalculating(service)
    recalcMutation.mutate(service)
  }

  const chartData = (metrics || []).map((m) => ({
    ts: new Date(m.ts).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" }),
    cpu: Math.round(m.cpu_percent * 100) / 100,
    mem: Math.round(m.mem_usage / 1024 / 1024),
    memPct: Math.round(m.mem_percent * 100) / 100,
  }))

  useEffect(() => {
    if (chartData.length > 0 && !chartAnimated) {
      const t = setTimeout(() => setChartAnimated(true), 600)
      return () => clearTimeout(t)
    }
  }, [chartData.length, chartAnimated])

  if (statsLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid gap-4 md:grid-cols-2">
          <Skeleton className="h-80" />
          <Skeleton className="h-80" />
        </div>
      </div>
    )
  }

  if (!stats) {
    return (
      <div className="space-y-6">
        <PageHeader title={serviceName} description="Detalhes do serviço" />
        <Card>
          <EmptyState icon={AlertTriangle} message="Nenhum dado disponível para este serviço" />
        </Card>
      </div>
    )
  }

  const statCards = [
    { label: "P50", cpu: stats.cpu_p50, mem: stats.mem_p50 },
    { label: "P95/P99", cpu: stats.cpu_p95, mem: stats.mem_p99 },
    { label: "Máx", cpu: stats.cpu_max, mem: stats.mem_max },
    { label: "Média", cpu: stats.cpu_avg, mem: stats.mem_avg },
  ]

  const maxCpu = Math.max(...(containers || []).map((c) => c.cpu_p95), 1)
  const maxMem = Math.max(...(containers || []).map((c) => c.mem_p99), 1)

  const filteredContainers = (containers || []).filter(
    (c) => containerFilter === "all" || c.status === containerFilter
  )

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

  return (
    <div className="space-y-6">
      <PageHeader title={serviceName} description={`${stats.samples} amostras coletadas`}>
        <div className="flex items-center gap-3">
          {overview && <StatusBadge status={overview.status} lastSeen={overview.last_seen} />}
          {overview && <RiskScoreGauge score={overview.risk_score} factors={overview.risk_factors} orientation="horizontal" border />}
          <div className="h-5 w-px bg-border" />
          {recommendation && (() => {
            const hasOom = (recommendation.oom_events ?? 0) > 0
            const hasDrift = recommendation.has_drift
            const hasLeak = recommendation.memory_trend?.has_leak
            if (!hasOom && !hasDrift && !hasLeak) return null
            return (
              <span className="inline-flex items-center gap-1">
                {hasOom && <Badge variant="destructive" className="text-[10px]">OOM</Badge>}
                {hasDrift && <Badge variant="warning" className="text-[10px]">Drift</Badge>}
                {hasLeak && <Badge variant="danger" className="text-[10px]">Leak</Badge>}
              </span>
            )
          })()}
          <Button variant="outline" size="sm" onClick={() => navigate("/services")}>
            <ArrowLeft className="h-4 w-4" />
            Voltar
          </Button>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-2">
        {/* Card unificado: 4 indicadores (P50, P95/P99, Máx, Média) */}
        <Card className="hover:border-primary/40 hover:shadow-md transition-all">
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <TrendingUp className="h-4 w-4" />
              Estatísticas de Recursos
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-4 gap-3">
              {statCards.map((stat) => (
                <div key={stat.label} className="space-y-2">
                  <span className="text-xs font-medium text-muted-foreground">{stat.label}</span>
                  <div className="space-y-1.5">
                    <div className="flex items-center gap-1.5">
                      <Cpu className="h-3 w-3 text-primary shrink-0" />
                      <span className="text-sm font-bold tabular-nums">{formatCPU(stat.cpu)}</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <MemoryStick className="h-3 w-3 text-chart-2 shrink-0" />
                      <span className="text-sm font-bold tabular-nums">{formatBytes(stat.mem)}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Card de configuração atual de recursos — sempre visível, com loading */}
        <Card className="hover:border-primary/40 hover:shadow-md transition-all">
          <CardHeader className="pb-3">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Box className="h-4 w-4" />
              Configuração Atual de Recursos
            </CardTitle>
          </CardHeader>
          <CardContent>
            {recLoading ? (
              <div className="space-y-3">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Skeleton className="h-3 w-24" />
                    <Skeleton className="h-5 w-20" />
                    <Skeleton className="h-5 w-20" />
                  </div>
                  <div className="space-y-2">
                    <Skeleton className="h-3 w-24" />
                    <Skeleton className="h-5 w-20" />
                    <Skeleton className="h-5 w-20" />
                  </div>
                </div>
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1.5">
                  <span className="text-xs font-medium text-muted-foreground">Limites (Caps)</span>
                  <div className="flex items-center gap-2">
                    <Cpu className="h-3.5 w-3.5 text-primary" />
                    <span className="text-sm tabular-nums">
                      {recommendation?.current?.cpu_limit ? formatCPU(recommendation.current.cpu_limit) : "—"}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <MemoryStick className="h-3.5 w-3.5 text-chart-2" />
                    <span className="text-sm tabular-nums">
                      {recommendation?.current?.mem_limit ? formatBytes(recommendation.current.mem_limit) : "—"}
                    </span>
                  </div>
                </div>
                <div className="space-y-1.5">
                  <span className="text-xs font-medium text-muted-foreground">Reservas (Garantias)</span>
                  <div className="flex items-center gap-2">
                    <Cpu className="h-3.5 w-3.5 text-primary" />
                    <span className="text-sm tabular-nums">
                      {recommendation?.current?.cpu_reservation ? formatCPU(recommendation.current.cpu_reservation) : "—"}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <MemoryStick className="h-3.5 w-3.5 text-chart-2" />
                    <span className="text-sm tabular-nums">
                      {recommendation?.current?.mem_reservation ? formatBytes(recommendation.current.mem_reservation) : "—"}
                    </span>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Visão Geral</TabsTrigger>
          <TabsTrigger value="containers">
            <Box className="mr-1.5 h-3.5 w-3.5" />
            Containers {(containers || []).length > 0 && `(${(containers || []).length})`}
          </TabsTrigger>
          <TabsTrigger value="recommendation">
            <Zap className="mr-1.5 h-3.5 w-3.5" />
            Recomendação
          </TabsTrigger>
          <TabsTrigger value="schedules">
            <CalendarClock className="mr-1.5 h-3.5 w-3.5" />
            Agendamentos {((serviceSchedules || []).length > 0 || (serviceChangeLog || []).length > 0) && `(${(serviceSchedules || []).length + (serviceChangeLog || []).length})`}
          </TabsTrigger>
          {(serviceTasks || []).length > 0 && (
            <TabsTrigger value="tasks">
              <ListTodo className="mr-1.5 h-3.5 w-3.5" />
              Tasks ({(serviceTasks || []).length})
            </TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Cpu className="h-4 w-4 text-primary" />
                  CPU ao longo do tempo
                </CardTitle>
              </CardHeader>
              <CardContent>
                {chartData.length === 0 ? (
                  <EmptyState icon={Cpu} message="Sem dados de CPU para este período" />
                ) : (
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={chartData} margin={{ top: 5, right: 16, left: 0, bottom: 5 }}>
                        <defs>
                          <linearGradient id="cpuGrad" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-1)" stopOpacity={0.3} />
                            <stop offset="95%" stopColor="var(--chart-1)" stopOpacity={0} />
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                        <XAxis dataKey="ts" tick={{ fill: "var(--muted-foreground)", fontSize: 10 }} axisLine={false} tickLine={false} minTickGap={50} />
                        <YAxis tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} unit="%" />
                        <RTooltip contentStyle={chartTooltipStyle} />
                        <Area type="monotone" dataKey="cpu" stroke="var(--chart-1)" fill="url(#cpuGrad)" strokeWidth={2} name="CPU %" isAnimationActive={!chartAnimated} />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <MemoryStick className="h-4 w-4 text-chart-2" />
                  Memória ao longo do tempo
                </CardTitle>
              </CardHeader>
              <CardContent>
                {chartData.length === 0 ? (
                  <EmptyState icon={MemoryStick} message="Sem dados de memória para este período" />
                ) : (
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={chartData} margin={{ top: 5, right: 16, left: 0, bottom: 5 }}>
                        <defs>
                          <linearGradient id="memGrad" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-2)" stopOpacity={0.3} />
                            <stop offset="95%" stopColor="var(--chart-2)" stopOpacity={0} />
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                        <XAxis dataKey="ts" tick={{ fill: "var(--muted-foreground)", fontSize: 10 }} axisLine={false} tickLine={false} minTickGap={50} />
                        <YAxis tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} unit=" MB" />
                        <RTooltip contentStyle={chartTooltipStyle} />
                        <Area type="monotone" dataKey="mem" stroke="var(--chart-2)" fill="url(#memGrad)" strokeWidth={2} name="Mem (MB)" isAnimationActive={!chartAnimated} />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <TrendingUp className="h-4 w-4 text-chart-5" />
                Estatísticas Detalhadas
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-6 md:grid-cols-2">
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <Cpu className="h-4 w-4 text-primary" />
                    CPU
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Mínimo</span>
                      <span className="font-medium tabular-nums">{formatCPU(stats.cpu_min)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">P50 (Mediana)</span>
                      <span className="font-medium tabular-nums">{formatCPU(stats.cpu_p50)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">P95</span>
                      <span className="font-semibold text-primary tabular-nums">{formatCPU(stats.cpu_p95)}</span>
                    </div>
                    <Progress value={stats.cpu_p95} max={stats.cpu_max || 1} indicatorClassName="bg-primary" />
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Máximo</span>
                      <span className="font-medium tabular-nums">{formatCPU(stats.cpu_max)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Média</span>
                      <span className="font-medium tabular-nums">{formatCPU(stats.cpu_avg)}</span>
                    </div>
                  </div>
                </div>

                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <MemoryStick className="h-4 w-4 text-chart-2" />
                    Memória
                  </div>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Mínimo</span>
                      <span className="font-medium tabular-nums">{formatBytes(stats.mem_min)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">P50 (Mediana)</span>
                      <span className="font-medium tabular-nums">{formatBytes(stats.mem_p50)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">P99</span>
                      <span className="font-semibold text-chart-2 tabular-nums">{formatBytes(stats.mem_p99)}</span>
                    </div>
                    <Progress value={stats.mem_p99} max={stats.mem_max || 1} indicatorClassName="bg-chart-2" />
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Máximo</span>
                      <span className="font-medium tabular-nums">{formatBytes(stats.mem_max)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Média</span>
                      <span className="font-medium tabular-nums">{formatBytes(stats.mem_avg)}</span>
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="containers" className="space-y-4">
          {!containers || containers.length === 0 ? (
            <Card>
              <EmptyState icon={Box} message="Nenhum container individual encontrado para este serviço." />
            </Card>
          ) : (
            <Card>
              <CardContent className="p-0">
                <div className="flex items-center justify-between px-4 py-3 border-b">
                  <span className="text-sm text-muted-foreground">
                    {filteredContainers.length} de {containers.length} containers
                  </span>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="outline" size="sm" className="gap-1.5">
                        <Filter className="h-3.5 w-3.5" />
                        {containerFilter === "all" ? "Todos" : statusBadge[containerFilter]?.label ?? containerFilter}
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => setContainerFilter("all")}>
                        Todos ({containers.length})
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setContainerFilter("online")}>
                        Online ({containers.filter(c => c.status === "online").length})
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setContainerFilter("offline")}>
                        Offline ({containers.filter(c => c.status === "offline").length})
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => setContainerFilter("legado")}>
                        Legado ({containers.filter(c => c.status === "legado").length})
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Container ID</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Rede / IP</TableHead>
                      <TableHead className="text-center">Amostras</TableHead>
                      <TableHead className="w-48 text-right">CPU P95</TableHead>
                      <TableHead className="w-48 text-right">Memória P99</TableHead>
                      <TableHead className="w-12"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredContainers.map((c) => (
                      <TableRow
                        key={c.container_id}
                        className="cursor-pointer hover:bg-accent/40 transition-colors duration-150"
                        onClick={() => navigate(`/services/${encodeURIComponent(serviceName)}/containers/${encodeURIComponent(c.container_id)}`)}
                      >
                        <TableCell className="font-mono text-xs">{c.container_id.substring(0, 12)}</TableCell>
                        <TableCell>
                          <Badge variant={statusBadge[c.status]?.variant ?? "outline"}>
                            {statusBadge[c.status]?.label ?? c.status}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {c.networks && Object.keys(c.networks).length > 0 ? (
                            <div className="space-y-0.5">
                              {Object.entries(c.networks).map(([net, ip]) => (
                                <div key={net} className="flex items-center gap-1.5">
                                  <span className="text-xs text-muted-foreground truncate max-w-28">{net}</span>
                                  <span className="font-mono text-xs">{ip}</span>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <span className="text-xs text-muted-foreground">—</span>
                          )}
                        </TableCell>
                        <TableCell className="text-center">
                          <Badge variant="outline" className="border-chart-2/40 text-chart-2">{c.samples}</Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center justify-end gap-3">
                            <Progress
                              value={c.cpu_p95}
                              max={maxCpu}
                              indicatorClassName={getCpuColor(c.cpu_p95)}
                              className="w-24 shrink-0"
                            />
                            <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap shrink-0 w-14 text-right">
                              {formatCPU(c.cpu_p95)}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center justify-end gap-3">
                            <Progress
                              value={c.mem_p99}
                              max={maxMem}
                              indicatorClassName={getMemColor(c.mem_p99, maxMem)}
                              className="w-24 shrink-0"
                            />
                            <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap shrink-0 w-20 text-right">
                              {formatBytes(c.mem_p99)}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <ChevronRight className="h-4 w-4 text-muted-foreground" />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="recommendation" className="space-y-4">
          {recommendation ? (
            <div className="grid gap-4 md:grid-cols-2">
              <RecommendationCard
                rec={recommendation}
                onApply={handleApply}
                applying={applying}
                error={null}
                onRecalculate={handleRecalculate}
                recalculating={recalculating}
                onSchedule={() => {}}
                onCancelSchedule={() => {}}
              />
            </div>
          ) : (
            <Card>
              <EmptyState icon={Zap} message="Nenhuma recomendação disponível para este serviço." />
            </Card>
          )}
        </TabsContent>

        <TabsContent value="schedules" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <CalendarClock className="h-4 w-4 text-warning" />
                Agendamentos do Serviço
              </CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {(!serviceSchedules || serviceSchedules.length === 0) ? (
                <EmptyState icon={CalendarClock} message="Nenhum agendamento para este serviço" />
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Status</TableHead>
                      <TableHead className="text-right">CPU Lim</TableHead>
                      <TableHead className="text-right">Mem Lim</TableHead>
                      <TableHead>Agendado para</TableHead>
                      <TableHead>Aplicado em</TableHead>
                      <TableHead className="text-center">Tentativas</TableHead>
                      <TableHead className="w-12"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {serviceSchedules.map((s) => {
                      const cfg = scheduleStatusConfig[s.status] ?? { label: s.status, variant: "outline" as const }
                      return (
                        <TableRow key={s.id}>
                          <TableCell>
                            <Badge variant={cfg.variant} className="text-xs">{cfg.label}</Badge>
                          </TableCell>
                          <TableCell className="text-right tabular-nums text-muted-foreground">{fmtVal(s.cpu_limit, "cpu")}</TableCell>
                          <TableCell className="text-right tabular-nums text-muted-foreground">{fmtVal(s.mem_limit, "mem")}</TableCell>
                          <TableCell className="text-sm text-muted-foreground">{fmtDate(s.scheduled_at)}</TableCell>
                          <TableCell className="text-sm text-muted-foreground">{fmtDate(s.applied_at)}</TableCell>
                          <TableCell className="text-center">
                            <Badge variant="outline" className="tabular-nums">{s.attempts}</Badge>
                          </TableCell>
                          <TableCell>
                            {s.status === "pending" && (
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-7 w-7 text-destructive hover:bg-destructive/10"
                                onClick={() => cancelScheduleMutation.mutate(s.id)}
                              >
                                <XCircle className="h-3.5 w-3.5" />
                              </Button>
                            )}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <History className="h-4 w-4 text-chart-5" />
                Histórico de Alterações
              </CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {(!serviceChangeLog || serviceChangeLog.length === 0) ? (
                <EmptyState icon={History} message="Nenhuma alteração registrada para este serviço" />
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Ação</TableHead>
                      <TableHead>Fonte</TableHead>
                      <TableHead>Usuário</TableHead>
                      <TableHead className="text-right">CPU Antes→Depois</TableHead>
                      <TableHead className="text-right">Mem Antes→Depois</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Data</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {serviceChangeLog.map((e) => (
                      <TableRow key={e.id}>
                        <TableCell>
                          <Badge variant="outline" className="text-xs">
                            {e.action === "apply" ? "Aplicar" : e.action === "scheduled_apply" ? "Agendado" : e.action}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={e.source === "manual" ? "secondary" : "outline"} className="text-xs">
                            {e.source === "manual" ? "Manual" : "Scheduler"}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">{e.user ?? "—"}</TableCell>
                        <TableCell className="text-right text-sm tabular-nums text-muted-foreground">
                          {fmtVal(e.cpu_limit_before, "cpu")} → {fmtVal(e.cpu_limit_after, "cpu")}
                        </TableCell>
                        <TableCell className="text-right text-sm tabular-nums text-muted-foreground">
                          {fmtVal(e.mem_limit_before, "mem")} → {fmtVal(e.mem_limit_after, "mem")}
                        </TableCell>
                        <TableCell>
                          <Badge variant={e.status === "completed" ? "success" : "destructive"} className="text-xs">
                            {e.status === "completed" ? "OK" : "Erro"}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-sm text-muted-foreground">{fmtDate(e.created_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Fase 7 — Tasks tab (Swarm task lifecycle) */}
        {(serviceTasks || []).length > 0 && (
          <TabsContent value="tasks" className="space-y-4">
            {/* Service health summary */}
            {(() => {
              const health = (serviceHealth || []).find((h) => h.service === serviceName)
              if (!health) return null
              return (
                <div className="grid gap-4 md:grid-cols-4">
                  <Card>
                    <CardContent className="pb-2 pt-6">
                      <div className="text-sm text-muted-foreground">Tasks Running</div>
                      <div className="text-2xl font-bold text-success tabular-nums">{health.tasks_running}</div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pb-2 pt-6">
                      <div className="text-sm text-muted-foreground">Tasks Failed</div>
                      <div className="text-2xl font-bold text-destructive tabular-nums">{health.tasks_failed}</div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pb-2 pt-6">
                      <div className="text-sm text-muted-foreground">Tasks Pending</div>
                      <div className="text-2xl font-bold text-warning tabular-nums">{health.tasks_pending}</div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pb-2 pt-6">
                      <div className="text-sm text-muted-foreground">Restarts (7d)</div>
                      <div className="text-2xl font-bold text-chart-1 tabular-nums">{health.restarts}</div>
                    </CardContent>
                  </Card>
                </div>
              )
            })()}

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <ListTodo className="h-4 w-4 text-chart-1" />
                  Tasks do Swarm
                </CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="text-center">Slot</TableHead>
                      <TableHead className="text-center">Status</TableHead>
                      <TableHead className="text-center">Desired</TableHead>
                      <TableHead>Node</TableHead>
                      <TableHead className="font-mono text-xs">Task ID</TableHead>
                      <TableHead>Updated</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(serviceTasks || []).map((t) => {
                      const cfg = taskStatusConfig[t.status] ?? { label: t.status, variant: "outline" as const, dot: "bg-muted-foreground" }
                      return (
                        <TableRow key={t.task_id}>
                          <TableCell className="text-center">
                            <Badge variant="outline" className="tabular-nums">{t.slot}</Badge>
                          </TableCell>
                          <TableCell className="text-center">
                            <Badge variant={cfg.variant} className="gap-1.5">
                              <span className={`h-2 w-2 rounded-full ${cfg.dot}`} />
                              {cfg.label}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-center">
                            <Badge variant="outline">{t.desired_state || "—"}</Badge>
                          </TableCell>
                          <TableCell className="text-sm">{t.node_hostname || t.node_id?.substring(0, 12) || "—"}</TableCell>
                          <TableCell className="font-mono text-xs text-muted-foreground">{t.task_id.substring(0, 12)}</TableCell>
                          <TableCell className="text-sm text-muted-foreground">{t.updated_at ? fmtDate(t.updated_at) : "—"}</TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          </TabsContent>
        )}
      </Tabs>
    </div>
  )
}
