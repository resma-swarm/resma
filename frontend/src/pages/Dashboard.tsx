import { useState, useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { api } from "@/api/client"
import { useRefreshInterval } from "@/hooks/use-refresh"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Button } from "@/components/ui/button"
import { useNavigate } from "react-router-dom"
import { EmptyState } from "@/components/empty-state"
import { PageHeader } from "@/components/page-header"
import { formatBytes, formatCPU } from "@/lib/utils"
import { Server, Cpu, MemoryStick, AlertTriangle, Boxes, ChevronRight, HardDrive, Database, RefreshCw } from "lucide-react"
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, ResponsiveContainer,
} from "recharts"
import { HelpIcon } from "@/components/help-icon"
import { Separator } from "@/components/ui/separator"

interface DashboardData {
  total_services: number
  total_containers: number
  top_cpu_consumers: Array<{ name: string; container_count: number; cpu_p95: number; mem_p99: number }>
  top_mem_consumers: Array<{ name: string; container_count: number; cpu_p95: number; mem_p99: number }>
  alerts_summary: {
    total: number
    oom_count: number
    leak_count: number
    drift_count: number
  }
  cluster?: { nodes_total: number; managers_total: number; workers_total: number; nodes_ready: number; nodes_down: number; quorum_healthy: boolean } | null
  cluster_capacity?: { cpu_total: number; mem_total: number; tasks_total: number; cpu_p95: number; mem_usage: number } | null
  nodes_distribution?: Array<{ hostname: string; node_id: string; role: string; status: string; tasks_running: number }>
}

const triggerHelpText: Record<string, { title: string; body: string }> = {
  oom: {
    title: "OOM (Out of Memory)",
    body: "O container ultrapassou o limite de memória e o kernel o encerrou à força (exit 137). Causas comuns: limite muito baixo, vazamento de memória na aplicação, ou pico de tráfego. Para resolver: aumente o limite de memória ou corrija o uso de memória na aplicação.",
  },
  leak: {
    title: "Memory Leak",
    body: "A memória do serviço cresce continuamente sem ser liberada. Se não tratado, o serviço vai sofrer OOM repetidamente. Para resolver: identifique o vazamento com profiling, adicione limites de cache, remova listeners não usados, ou reinicie o serviço periodicamente como paliativo.",
  },
  drift: {
    title: "Resource Drift",
    body: "O padrão de uso de CPU ou memória mudou significativamente em relação ao período anterior. Isso indica que a carga mudou e os limites atuais podem não ser mais adequados. Para resolver: recalcule os limites do container com base no novo padrão de uso.",
  },
  alerts: {
    title: "Alertas",
    body: "Total de alertas ativos (OOMs, memory leaks e resource drifts) detectados na janela de análise. Clique para ver detalhes na página de Alertas.",
  },
  topCpu: {
    title: "CPU P95 — Multi-core",
    body: "O Docker reporta CPU% somando todos os cores: 100% = 1 core, 1600% = 16 cores. O valor entre parênteses é a % normalizada (0–100%) dividindo pelo total de cores do cluster. Ex: 1311.85% (82%) significa ~13 cores em uso no P95.",
  },
  topMem: {
    title: "Memória P99",
    body: "P99 = percentil 99 (quase o pico). O valor entre parênteses é a % da memória total do cluster usada por este serviço. Útil para identificar serviços que concentram a maior parte da RAM.",
  },
}

const chartTooltipStyle = {
  backgroundColor: "var(--popover)",
  border: "1px solid var(--border)",
  borderRadius: "0.5rem",
  fontSize: "12px",
  color: "var(--popover-foreground)",
}

interface StorageSummary {
  live: {
    images: { count: number; total_size: number; reclaimable: number }
    containers: { count: number; total_size: number }
    volumes: { count: number; total_size: number; reclaimable: number; orphan_count: number; orphan_size: number }
    top_volumes: Array<{ name: string; size: number; in_use: boolean; driver: string }>
    orphan_volumes: Array<{ name: string; size: number; driver: string }>
  }
  latest_snapshot: Record<string, number | string | null> | null
}

export default function Dashboard() {
  const navigate = useNavigate()
  const refreshInterval = useRefreshInterval()
  const [chartAnimated, setChartAnimated] = useState(false)

  // SSE: invalida queries do dashboard quando receber evento.
  // Gap D: assina tanto "dashboard" (cluster info, ~30s) quanto "metrics"
  // (coleta de métricas, ~5s) para atualização em tempo real de métricas.
  const { isConnected: sseDashboard } = useEventSource({
    topic: "dashboard",
    invalidateQueries: [["dashboard"], ["storage-summary"]],
  })
  const { isConnected: sseMetrics } = useEventSource({
    topic: "metrics",
    invalidateQueries: [["dashboard"], ["storage-summary"]],
  })
  const sseConnected = sseDashboard || sseMetrics

  // SSE ativo = zero polling (SSE publica payload completo + reconciliação 30s)
  // SSE inativo = polling normal como fallback
  const fallbackInterval = sseConnected ? false : refreshInterval

  const { data, isLoading, isError, refetch } = useQuery<DashboardData>({
    queryKey: ["dashboard"],
    queryFn: () => api.get<DashboardData>("/dashboard"),
    refetchInterval: fallbackInterval,
  })

  const cpuChartData = (data?.top_cpu_consumers ?? []).map((s) => ({
    name: s.name.length > 12 ? s.name.slice(0, 11) + "…" : s.name,
    cpu: Math.round(s.cpu_p95 * 100) / 100,
  }))

  const memChartData = (data?.top_mem_consumers ?? []).map((s) => ({
    name: s.name.length > 12 ? s.name.slice(0, 11) + "…" : s.name,
    mem: Math.round(s.mem_p99 / 1024 / 1024),
  }))

  useEffect(() => {
    if (cpuChartData.length > 0 && !chartAnimated) {
      const t = setTimeout(() => setChartAnimated(true), 600)
      return () => clearTimeout(t)
    }
  }, [cpuChartData.length, chartAnimated])

  const { data: storageData } = useQuery<StorageSummary>({
    queryKey: ["storage-summary"],
    queryFn: () => api.get<StorageSummary>("/storage/summary"),
    refetchInterval: fallbackInterval,
  })

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
        <div className="grid gap-4 md:grid-cols-2">
          <Skeleton className="h-80" />
          <Skeleton className="h-80" />
        </div>
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="space-y-6">
        <PageHeader title="Dashboard" description="Visão geral dos recursos do cluster" />
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-20 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10 mb-4">
              <AlertTriangle className="h-7 w-7 text-destructive" />
            </div>
            <h3 className="text-lg font-semibold">Não foi possível carregar o dashboard</h3>
            <p className="text-sm text-muted-foreground mt-1 max-w-md">
              Ocorreu um erro ao buscar os dados do dashboard. Verifique a conexão com a API e tente novamente.
            </p>
            <Button variant="outline" size="sm" className="mt-5 gap-2" onClick={() => refetch()}>
              <RefreshCw className="h-4 w-4" />
              Tentar novamente
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  const maxCpu = Math.max(...(data.top_cpu_consumers ?? []).map((s) => s.cpu_p95), 1)
  const maxMem = Math.max(...(data.top_mem_consumers ?? []).map((s) => s.mem_p99), 1)

  const cluster = data.cluster
  const clusterCapacity = data.cluster_capacity
  const nodesDist = data.nodes_distribution ?? []

  // Normalização: CPU% do Docker é multi-core (100% = 1 core).
  // Para obter % normalizada (0–100%): cpu_p95 / num_cores.
  // Ex: 1311.85% / 16 cores = 81.99% normalizado.
  // Memória: divide pelo total de RAM do cluster.
  const cpuCores = clusterCapacity?.cpu_total || 1
  const memTotal = clusterCapacity?.mem_total || 1
  const normCpu = (v: number) => Math.min(v / cpuCores, 100)
  const normMem = (v: number) => (v / memTotal) * 100

  const nodeChartData = nodesDist.map((n) => ({
    name: n.hostname.length > 12 ? n.hostname.slice(0, 11) + "\u2026" : n.hostname,
    tasks: n.tasks_running,
  }))

  const stats = [
    {
      label: "Serviços",
      value: data.total_services,
      icon: Server,
      iconBg: "bg-primary/15",
      iconColor: "text-primary",
      valueColor: "text-primary",
      clickable: true,
      to: "/services",
    },
    {
      label: "Containers",
      value: data.total_containers,
      icon: Boxes,
      iconBg: "bg-chart-2/15",
      iconColor: "text-chart-2",
      valueColor: "text-chart-2",
      clickable: false,
    },
    {
      label: "Nodes",
      value: cluster?.nodes_total ?? 0,
      icon: HardDrive,
      iconBg: "bg-chart-3/15",
      iconColor: "text-chart-3",
      valueColor: "text-chart-3",
      clickable: true,
      to: "/nodes",
    },
    {
      label: "Alertas",
      value: data.alerts_summary?.total ?? 0,
      icon: AlertTriangle,
      iconBg: (data.alerts_summary?.total ?? 0) > 0 ? "bg-destructive/15" : "bg-muted",
      iconColor: (data.alerts_summary?.total ?? 0) > 0 ? "text-destructive" : "text-muted-foreground",
      valueColor: (data.alerts_summary?.total ?? 0) > 0 ? "text-destructive" : "text-foreground",
      clickable: true,
      to: "/alerts",
      helpKey: "alerts",
    },
  ]

  return (
    <div className="space-y-6">
      <PageHeader
        title="Dashboard"
        description="Visão geral dos recursos do cluster"
      />

      <div className="grid gap-4 md:grid-cols-4">
        {stats.map((stat) => (
          <Card
            key={stat.label}
            className={stat.clickable ? "cursor-pointer hover:border-primary/50 hover:shadow-lg hover:-translate-y-0.5 transition-all group" : "hover:bg-accent/30 transition-colors"}
            onClick={stat.clickable && stat.to ? () => navigate(stat.to) : undefined}
          >
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <div className="flex items-center gap-1.5">
                <CardTitle className="text-sm font-medium text-muted-foreground">
                  {stat.label}
                </CardTitle>
                {stat.helpKey && (
                  <HelpIcon
                    title={triggerHelpText[stat.helpKey].title}
                    text={triggerHelpText[stat.helpKey].body}
                    side="bottom"
                  />
                )}
              </div>
              <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${stat.iconBg}`}>
                <stat.icon className={`h-4.5 w-4.5 ${stat.iconColor}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div className={`text-3xl font-bold tabular-nums ${stat.valueColor}`}>{stat.value}</div>
                {stat.clickable && (
                  <ChevronRight className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors" />
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Cpu className="h-4 w-4 text-primary" />
              Top CPU Consumers
              <HelpIcon
                title={triggerHelpText.topCpu.title}
                text={triggerHelpText.topCpu.body}
                side="bottom"
              />
            </CardTitle>
          </CardHeader>
          <CardContent>
            {(data.top_cpu_consumers?.length ?? 0) === 0 ? (
              <EmptyState icon={Cpu} message="Nenhum dado de CPU disponível" />
            ) : (
              <>
                <div className="h-48 mb-4">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={cpuChartData} layout="vertical" margin={{ left: 0, right: 16 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" horizontal={false} />
                      <XAxis type="number" tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} />
                      <YAxis type="category" dataKey="name" tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} width={90} />
                      <RTooltip contentStyle={chartTooltipStyle} cursor={{ fill: "var(--muted)" }} />
                      <Bar dataKey="cpu" fill="var(--chart-1)" radius={[0, 4, 4, 0]} name="CPU P95 (%)" isAnimationActive={!chartAnimated} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
                <div className="space-y-2">
                  {data.top_cpu_consumers.slice(0, 5).map((s) => (
                    <div key={s.name} className="flex items-center gap-3 rounded-md px-2 py-1 -mx-2 hover:bg-accent/40 transition-colors cursor-default">
                      <span className="text-sm font-medium min-w-0 flex-1 truncate text-left">{s.name}</span>
                      <Progress
                        value={s.cpu_p95}
                        max={maxCpu}
                        indicatorClassName="bg-chart-1"
                        className="w-24 shrink-0"
                      />
                      <span className="text-sm tabular-nums text-muted-foreground w-32 shrink-0 text-right whitespace-nowrap">
                        {formatCPU(s.cpu_p95)}{" "}
                        <span className="text-muted-foreground/70">({formatCPU(normCpu(s.cpu_p95))})</span>
                      </span>
                    </div>
                  ))}
                </div>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <MemoryStick className="h-4 w-4 text-chart-2" />
              Top Memory Consumers
              <HelpIcon
                title={triggerHelpText.topMem.title}
                text={triggerHelpText.topMem.body}
                side="bottom"
              />
            </CardTitle>
          </CardHeader>
          <CardContent>
            {(data.top_mem_consumers?.length ?? 0) === 0 ? (
              <EmptyState icon={MemoryStick} message="Nenhum dado de memória disponível" />
            ) : (
              <>
                <div className="h-48 mb-4">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={memChartData} layout="vertical" margin={{ left: 0, right: 16 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" horizontal={false} />
                      <XAxis type="number" tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} unit=" MB" />
                      <YAxis type="category" dataKey="name" tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} width={90} />
                      <RTooltip contentStyle={chartTooltipStyle} cursor={{ fill: "var(--muted)" }} />
                      <Bar dataKey="mem" fill="var(--chart-2)" radius={[0, 4, 4, 0]} name="Mem P99 (MB)" isAnimationActive={!chartAnimated} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
                <div className="space-y-2">
                  {data.top_mem_consumers.slice(0, 5).map((s) => (
                    <div key={s.name} className="flex items-center gap-3 rounded-md px-2 py-1 -mx-2 hover:bg-accent/40 transition-colors cursor-default">
                      <span className="text-sm font-medium min-w-0 flex-1 truncate text-left">{s.name}</span>
                      <Progress
                        value={s.mem_p99}
                        max={maxMem}
                        indicatorClassName="bg-chart-2"
                        className="w-24 shrink-0"
                      />
                      <span className="text-sm tabular-nums text-muted-foreground w-36 shrink-0 text-right whitespace-nowrap">
                        {formatBytes(s.mem_p99)}{" "}
                        <span className="text-muted-foreground/70">({normMem(s.mem_p99).toFixed(1)}%)</span>
                      </span>
                    </div>
                  ))}
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {storageData?.live?.volumes && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Database className="h-4 w-4 text-chart-5" />
              Storage Overview
              {storageData.live.volumes.orphan_count > 0 && (
                <Badge variant="warning" className="ml-1">{storageData.live.volumes.orphan_count} órfãos</Badge>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3 mb-4">
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground">Volumes</div>
                <div className="text-2xl font-bold text-chart-5 tabular-nums">{storageData.live.volumes.count}</div>
                <div className="text-xs text-muted-foreground">{formatBytes(storageData.live.volumes.total_size)} total</div>
              </div>
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground">Imagens</div>
                <div className="text-2xl font-bold text-primary tabular-nums">{storageData.live.images.count}</div>
                <div className="text-xs text-muted-foreground">{formatBytes(storageData.live.images.total_size)} total</div>
              </div>
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground">Reclaimable</div>
                <div className="text-2xl font-bold text-chart-4 tabular-nums">{formatBytes(storageData.live.volumes.reclaimable + storageData.live.images.reclaimable)}</div>
                <div className="text-xs text-muted-foreground">
                  {formatBytes(storageData.live.volumes.reclaimable)} volumes · {formatBytes(storageData.live.images.reclaimable)} imagens
                </div>
              </div>
            </div>
            {storageData.live.volumes.orphan_count > 0 && (
              <div className="rounded-lg border border-warning/30 bg-warning/5 p-3 mb-3">
                <div className="flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 text-warning" />
                  <span className="text-sm font-medium">{storageData.live.volumes.orphan_count} volumes órfãos</span>
                  <span className="text-xs text-muted-foreground">— {formatBytes(storageData.live.volumes.orphan_size)} desperdiçados</span>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {cluster && clusterCapacity && (
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <HardDrive className="h-4 w-4 text-chart-3" />
                Distribuição de Tasks por Node
              </CardTitle>
            </CardHeader>
            <CardContent>
              {nodeChartData.length === 0 ? (
                <EmptyState icon={HardDrive} message="Sem dados de nodes disponíveis" />
              ) : (
                <div className="h-56">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={nodeChartData} margin={{ top: 5, right: 16, left: 0, bottom: 5 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                      <XAxis dataKey="name" tick={{ fill: "var(--muted-foreground)", fontSize: 10 }} axisLine={false} tickLine={false} minTickGap={20} />
                      <YAxis tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} allowDecimals={false} />
                      <RTooltip contentStyle={chartTooltipStyle} cursor={{ fill: "var(--muted)" }} />
                      <Bar dataKey="tasks" fill="var(--chart-5)" radius={[4, 4, 0, 0]} name="Tasks" isAnimationActive={!chartAnimated} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <Server className="h-4 w-4 text-primary" />
                Capacidade do Cluster
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <div className="text-sm text-muted-foreground">Nodes</div>
                  <div className="text-2xl font-bold text-chart-3 tabular-nums">{cluster.nodes_total}</div>
                  <div className="flex gap-2 text-xs text-muted-foreground">
                    <span>{cluster.managers_total} managers</span>
                    <span>·</span>
                    <span>{cluster.workers_total} workers</span>
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-sm text-muted-foreground">Status</div>
                  <div className="flex items-center gap-2">
                    <Badge variant={cluster.quorum_healthy ? "outline" : "destructive"} className="gap-1.5">
                      <span className={`h-2 w-2 rounded-full ${cluster.quorum_healthy ? "bg-green-500" : "bg-red-500"}`} />
                      {cluster.quorum_healthy ? "Quorum OK" : "Quorum Down"}
                    </Badge>
                  </div>
                  <div className="flex gap-2 text-xs text-muted-foreground">
                    <span className="text-green-500">{cluster.nodes_ready} ready</span>
                    {cluster.nodes_down > 0 && <span className="text-destructive">{cluster.nodes_down} down</span>}
                  </div>
                </div>
              </div>
              <Separator />
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <div className="text-sm text-muted-foreground">CPU Total</div>
                  <div className="text-xl font-bold text-primary tabular-nums">{formatCPU(clusterCapacity.cpu_total)}</div>
                  <div className="text-xs text-muted-foreground">P95: {formatCPU(clusterCapacity.cpu_p95)}</div>
                </div>
                <div className="space-y-1">
                  <div className="text-sm text-muted-foreground">Memória Total</div>
                  <div className="text-xl font-bold text-chart-2 tabular-nums">{formatBytes(clusterCapacity.mem_total)}</div>
                  <div className="text-xs text-muted-foreground">Uso: {formatBytes(clusterCapacity.mem_usage)}</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

    </div>
  )
}
