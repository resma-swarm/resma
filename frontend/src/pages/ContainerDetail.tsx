import { useState, useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { useParams, useNavigate } from "react-router-dom"
import { api } from "@/api/client"
import { useRefreshInterval } from "@/hooks/use-refresh"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { EmptyState } from "@/components/empty-state"
import { PageHeader } from "@/components/page-header"
import { formatBytes, formatCPU } from "@/lib/utils"
import { ArrowLeft, Cpu, MemoryStick, AlertTriangle, TrendingUp, Network } from "lucide-react"
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, ResponsiveContainer,
} from "recharts"

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
}

interface MetricPoint {
  ts: string
  cpu_percent: number
  mem_usage: number
  mem_limit: number
  mem_percent: number
}

interface ContainerNetworkInfo {
  network: string
  ip_address: string
  ipv6_address: string
  mac_address: string
  gateway: string
  endpoint_id: string
}

const chartTooltipStyle = {
  backgroundColor: "var(--popover)",
  border: "1px solid var(--border)",
  borderRadius: "0.5rem",
  fontSize: "12px",
  color: "var(--popover-foreground)",
}

export default function ContainerDetail() {
  const { name, containerId } = useParams<{ name: string; containerId: string }>()
  const navigate = useNavigate()
  const serviceName = decodeURIComponent(name || "")
  const cid = decodeURIComponent(containerId || "")

  const refreshInterval = useRefreshInterval()

  // SSE: assina o tópico "container-detail/{id}" — o collector publica o
  // payload completo (stats+metrics+network) a cada coleta quando há
  // subscriber ativo. O applySSEPayload faz setQueryData para cada query do
  // ContainerDetail — zero refetch HTTP.
  // Fallback: se SSE cair, polling normal (refreshInterval) + safety net 300s.
  const sseTopic = `container-detail/${cid}`
  const { isConnected: sseConnected } = useEventSource({
    topic: sseTopic,
    enabled: !!cid,
  })
  const fallbackInterval = sseConnected ? 300_000 : refreshInterval

  const { data: stats, isLoading: statsLoading } = useQuery<ContainerStats>({
    queryKey: ["container-stats", cid],
    queryFn: () => api.get<ContainerStats>(`/services/containers/${encodeURIComponent(cid)}/stats`),
    enabled: !!cid,
    refetchInterval: fallbackInterval,
  })

  const [chartAnimated, setChartAnimated] = useState(false)
  const { data: metrics } = useQuery<MetricPoint[]>({
    queryKey: ["container-metrics", cid],
    queryFn: () => api.get<MetricPoint[]>(`/services/containers/${encodeURIComponent(cid)}/metrics`),
    enabled: !!cid,
    refetchInterval: fallbackInterval,
  })

  const { data: networkInfo } = useQuery<ContainerNetworkInfo[]>({
    queryKey: ["container-network-info", cid],
    queryFn: () => api.get<ContainerNetworkInfo[]>(`/services/containers/${encodeURIComponent(cid)}/network-info`),
    enabled: !!cid,
    refetchInterval: fallbackInterval,
  })

  const chartData = (metrics || []).map((m) => ({
    ts: new Date(m.ts).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" }),
    cpu: Math.round(m.cpu_percent * 100) / 100,
    mem: Math.round(m.mem_usage / 1024 / 1024),
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
        <PageHeader title={`Container ${cid.substring(0, 12)}`} description={serviceName} />
        <Card>
          <EmptyState icon={AlertTriangle} message="Nenhum dado disponível para este container" />
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

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Container ${cid.substring(0, 12)}`}
        description={`${serviceName} • ${stats.samples} amostras`}
      >
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate(`/services/${encodeURIComponent(serviceName)}`)}>
            <ArrowLeft className="h-4 w-4" />
            Voltar ao serviço
          </Button>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-4">
        {statCards.map((stat) => (
          <Card key={stat.label} className="hover:border-primary/40 hover:shadow-md transition-all">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{stat.label}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <div className="flex items-center gap-2">
                <Cpu className="h-3.5 w-3.5 text-primary" />
                <span className="text-lg font-bold tabular-nums">{formatCPU(stat.cpu)}</span>
              </div>
              <div className="flex items-center gap-2">
                <MemoryStick className="h-3.5 w-3.5 text-chart-2" />
                <span className="text-lg font-bold tabular-nums">{formatBytes(stat.mem)}</span>
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

      {networkInfo && networkInfo.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Network className="h-4 w-4 text-chart-4" />
              Informações de Rede
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {networkInfo.map((net, i) => (
                <div key={i} className="rounded-lg border p-3 space-y-2">
                  <div className="flex items-center gap-2">
                    <Network className="h-3.5 w-3.5 text-muted-foreground" />
                    <span className="text-sm font-medium">{net.network}</span>
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {net.ip_address && (
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">IP</span>
                        <span className="font-mono text-xs">{net.ip_address}</span>
                      </div>
                    )}
                    {net.gateway && (
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">Gateway</span>
                        <span className="font-mono text-xs">{net.gateway}</span>
                      </div>
                    )}
                    {net.mac_address && (
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">MAC</span>
                        <span className="font-mono text-xs">{net.mac_address}</span>
                      </div>
                    )}
                    {net.ipv6_address && (
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">IPv6</span>
                        <span className="font-mono text-xs">{net.ipv6_address}</span>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
