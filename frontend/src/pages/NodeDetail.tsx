import { useState, useEffect } from "react"
import { useQuery } from "@tanstack/react-query"
import { useParams, useNavigate } from "react-router-dom"
import { api } from "@/api/client"
import { useRefreshTimer } from "@/hooks/use-refresh"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { EmptyState } from "@/components/empty-state"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { PageHeader } from "@/components/page-header"
import { formatBytes, formatCPU, formatCores } from "@/lib/utils"
import { ArrowLeft, Cpu, MemoryStick, AlertTriangle, Server, Boxes, Crown, Info, Search, X, Database, Bot } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Sparkline } from "@/components/sparkline"
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, ResponsiveContainer,
} from "recharts"

interface NodeDetail {
  node_id: string
  hostname: string
  role: string
  availability: string
  status: string
  address: string
  cpu_total: number
  mem_total: number
  os: string
  architecture: string
  engine_version: string
  is_leader: boolean
  reachability: string
  labels: Record<string, string>
  tasks_running: number
  cpu_p95: number
  mem_p99: number
  containers: number
  updated_at: string | null
}

interface NodeMetric {
  ts: string
  tasks_running: number
  cpu_total: number
  mem_total: number
}

interface NodeService {
  service: string
  containers: number
  cpu_p95: number
  mem_p99: number
}

// Fase 7 — agent info
interface AgentInfo {
  node_id: string
  hostname: string
  version: string
  containers_count: number
  last_heartbeat: string | null
  status: string
  first_seen: string | null
  updated_at: string | null
}

const STATUS_CONFIG: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline"; dot: string }> = {
  ready: { label: "Ready", variant: "outline", dot: "bg-green-500" },
  down: { label: "Down", variant: "destructive", dot: "bg-red-500" },
  disconnected: { label: "Disconnected", variant: "destructive", dot: "bg-red-500" },
  unknown: { label: "Unknown", variant: "secondary", dot: "bg-muted-foreground" },
}

const chartTooltipStyle = {
  backgroundColor: "var(--popover)",
  border: "1px solid var(--border)",
  borderRadius: "0.5rem",
  fontSize: "12px",
  color: "var(--popover-foreground)",
}

export default function NodeDetail() {
  const { nodeId } = useParams<{ nodeId: string }>()
  const navigate = useNavigate()
  const nodeIdDecoded = decodeURIComponent(nodeId || "")

  const [chartAnimated, setChartAnimated] = useState(false)
  const [serviceSearch, setServiceSearch] = useState("")

  // SSE: assina nodes (info de nó), metrics (sparklines/node-metrics) e
  // dashboard (storage-summary via setQueryData — evento "storage" ~300s).
  // Antes, ["storage-summary"] era invalidada a cada ~5s pelo tópico metrics
  // (refetch HTTP). Agora o tópico dashboard publica o payload completo de
  // storage (BuildStorageSummary) e o hook aplica setQueryData via
  // EVENT_QUERY_MAP["storage"] → ["storage-summary"] — zero refetch HTTP.
  // O evento "cluster" (~60s) ainda invalida ["storage-summary"] via
  // getUncoveredQueryKeys (safety net), mas é 12x menos refetch que antes.
  useEventSource({
    topic: "nodes",
    invalidateQueries: [
      ["node-detail", nodeIdDecoded],
      ["node-metrics", nodeIdDecoded],
      ["node-services", nodeIdDecoded],
      ["node-agent", nodeIdDecoded],
    ],
    enabled: !!nodeIdDecoded,
  })
  useEventSource({
    topic: "metrics",
    invalidateQueries: [
      ["service-sparklines"],
      ["node-metrics", nodeIdDecoded],
    ],
    enabled: !!nodeIdDecoded,
  })
  useEventSource({
    topic: "dashboard",
    enabled: !!nodeIdDecoded,
  })

  // Reconciliação safety-net controlada pelo dropdown (Frente C).
  // Inclui todas as queries da página para refresh periódico.
  useRefreshTimer([
    ["node-detail", nodeIdDecoded],
    ["node-metrics", nodeIdDecoded],
    ["node-services", nodeIdDecoded],
    ["node-agent", nodeIdDecoded],
    ["storage-summary"],
    ["service-sparklines"],
  ])

  const { data: node, isLoading } = useQuery<NodeDetail>({
    queryKey: ["node-detail", nodeIdDecoded],
    queryFn: () => api.get<NodeDetail>(`/nodes/${encodeURIComponent(nodeIdDecoded)}`),
    enabled: !!nodeIdDecoded,
    refetchInterval: false,
  })

  const { data: metrics } = useQuery<NodeMetric[]>({
    queryKey: ["node-metrics", nodeIdDecoded],
    queryFn: () => api.get<NodeMetric[]>(`/nodes/${encodeURIComponent(nodeIdDecoded)}/metrics`),
    enabled: !!nodeIdDecoded,
    refetchInterval: false,
  })

  const { data: services } = useQuery<NodeService[]>({
    queryKey: ["node-services", nodeIdDecoded],
    queryFn: () => api.get<NodeService[]>(`/nodes/${encodeURIComponent(nodeIdDecoded)}/services`),
    enabled: !!nodeIdDecoded,
    refetchInterval: false,
  })

  const { data: storageData } = useQuery<{
    live: {
      images: { count: number; total_size: number; reclaimable: number }
      containers: { count: number; total_size: number }
      volumes: { count: number; total_size: number; reclaimable: number; orphan_count: number; orphan_size: number }
      top_volumes: Array<{ name: string; size: number; in_use: boolean; driver: string }>
      orphan_volumes: Array<{ name: string; size: number; driver: string }>
    }
  } | null>({
    queryKey: ["storage-summary"],
    queryFn: async () => {
      try { return await api.get("/storage/summary") } catch { return null }
    },
    refetchInterval: false,
  })

  const { data: sparklines } = useQuery<Record<string, Array<{ ts: string; cpu: number; mem: number }>>>({
    queryKey: ["service-sparklines"],
    queryFn: () => api.get<Record<string, Array<{ ts: string; cpu: number; mem: number }>>>("/services/sparklines?points=20"),
    refetchInterval: false,
  })

  // Fase 7 — agent rodando neste node (se houver)
  const { data: agent } = useQuery<AgentInfo | null>({
    queryKey: ["node-agent", nodeIdDecoded],
    queryFn: async () => {
      try { return await api.get<AgentInfo>(`/agents/${encodeURIComponent(nodeIdDecoded)}`) } catch { return null }
    },
    enabled: !!nodeIdDecoded,
    refetchInterval: false,
  })

  const chartData = (metrics || []).map((m) => ({
    ts: new Date(m.ts).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" }),
    tasks: m.tasks_running,
  }))

  useEffect(() => {
    if (chartData.length > 0 && !chartAnimated) {
      const t = setTimeout(() => setChartAnimated(true), 600)
      return () => clearTimeout(t)
    }
  }, [chartData.length, chartAnimated])

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
      </div>
    )
  }

  if (!node) {
    return (
      <div className="space-y-6">
        <PageHeader title="Node" description="Detalhes do node" />
        <Card>
          <EmptyState icon={AlertTriangle} message="Node não encontrado" />
        </Card>
      </div>
    )
  }

  const statusCfg = STATUS_CONFIG[node.status] ?? STATUS_CONFIG.unknown

  const statCards = [
    { label: "CPU Total", value: `${formatCores(node.cpu_total)} cores`, icon: Cpu, iconColor: "text-primary" },
    { label: "Memória Total", value: formatBytes(node.mem_total), icon: MemoryStick, iconColor: "text-chart-2" },
    { label: "Tasks", value: String(node.tasks_running || 0), icon: Boxes, iconColor: "text-chart-5" },
    { label: "Containers", value: String(node.containers || 0), icon: Server, iconColor: "text-chart-3" },
  ]

  const maxCpu = Math.max(...(services || []).map((s) => s.cpu_p95), 1)
  const maxMem = Math.max(...(services || []).map((s) => s.mem_p99), 1)

  const filteredServices = (services || []).filter((s) => {
    if (!serviceSearch) return true
    return s.service.toLowerCase().includes(serviceSearch.toLowerCase())
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

  return (
    <div className="space-y-6">
      <PageHeader title={node.hostname} description={`${node.role} · ${node.address}`}>
        <div className="flex items-center gap-2">
          {node.is_leader && (
            <Badge variant="warning" className="gap-1.5">
              <Crown className="h-3 w-3" />
              Leader
            </Badge>
          )}
          <Badge variant={statusCfg.variant} className="gap-1.5">
            <span className={`h-2 w-2 rounded-full ${statusCfg.dot}`} />
            {statusCfg.label}
          </Badge>
          <Badge variant="outline">{node.role}</Badge>
          <Badge variant="outline">{node.availability}</Badge>
          <button
            className="inline-flex items-center gap-1.5 rounded-md border border-input bg-transparent px-3 py-1.5 text-sm shadow-sm hover:bg-accent transition-colors"
            onClick={() => navigate("/nodes")}
          >
            <ArrowLeft className="h-4 w-4" />
            Voltar
          </button>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-4">
        {statCards.map((stat) => (
          <Card key={stat.label} className="hover:border-primary/40 hover:shadow-md transition-all">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{stat.label}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <stat.icon className={`h-4 w-4 ${stat.iconColor}`} />
                <span className="text-lg font-bold tabular-nums">{stat.value}</span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Visão Geral</TabsTrigger>
          <TabsTrigger value="services">
            <Boxes className="mr-1.5 h-3.5 w-3.5" />
            Services {(services || []).length > 0 && `(${(services || []).length})`}
          </TabsTrigger>
          <TabsTrigger value="info">
            <Info className="mr-1.5 h-3.5 w-3.5" />
            Info
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {/* Fase 7 — Agent card */}
          {agent && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Bot className="h-4 w-4 text-chart-4" />
                  RESMA Agent
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 md:grid-cols-4">
                  <div>
                    <div className="text-xs text-muted-foreground">Status</div>
                    <Badge variant={agent.status === "active" ? "success" : "warning"} className="mt-1 gap-1.5">
                      <span className={`h-2 w-2 rounded-full ${agent.status === "active" ? "bg-green-500" : "bg-orange-500"}`} />
                      {agent.status === "active" ? "Active" : "Stale"}
                    </Badge>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">Containers</div>
                    <div className="text-lg font-bold tabular-nums mt-1">{agent.containers_count}</div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">Version</div>
                    <div className="text-sm font-mono mt-1">{agent.version || "—"}</div>
                  </div>
                  <div>
                    <div className="text-xs text-muted-foreground">Last Heartbeat</div>
                    <div className="text-sm mt-1">{agent.last_heartbeat ? new Date(agent.last_heartbeat).toLocaleString("pt-BR") : "—"}</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Boxes className="h-4 w-4 text-chart-5" />
                  Tasks ao longo do tempo
                </CardTitle>
              </CardHeader>
              <CardContent>
                {chartData.length === 0 ? (
                  <EmptyState icon={Boxes} message="Sem dados de tasks para este período" />
                ) : (
                  <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={chartData} margin={{ top: 5, right: 16, left: 0, bottom: 5 }}>
                        <defs>
                          <linearGradient id="tasksGrad" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-5)" stopOpacity={0.3} />
                            <stop offset="95%" stopColor="var(--chart-5)" stopOpacity={0} />
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                        <XAxis dataKey="ts" tick={{ fill: "var(--muted-foreground)", fontSize: 10 }} axisLine={false} tickLine={false} minTickGap={50} />
                        <YAxis tick={{ fill: "var(--muted-foreground)", fontSize: 11 }} axisLine={false} tickLine={false} allowDecimals={false} />
                        <RTooltip contentStyle={chartTooltipStyle} />
                        <Area type="monotone" dataKey="tasks" stroke="var(--chart-5)" fill="url(#tasksGrad)" strokeWidth={2} name="Tasks" isAnimationActive={!chartAnimated} />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-base">
                  <Cpu className="h-4 w-4 text-primary" />
                  Consumo do Node
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <Cpu className="h-4 w-4 text-primary" />
                    CPU
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">P95 (janela análise)</span>
                    <span className="font-semibold text-primary tabular-nums">{formatCPU(node.cpu_p95)}</span>
                  </div>
                  <Progress value={node.cpu_p95} max={node.cpu_total || 1} indicatorClassName="bg-primary" />
                </div>

                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                    <MemoryStick className="h-4 w-4 text-chart-2" />
                    Memória
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">P99 (janela análise)</span>
                    <span className="font-semibold text-chart-2 tabular-nums">{formatBytes(node.mem_p99)}</span>
                  </div>
                  <Progress value={node.mem_p99} max={node.mem_total || 1} indicatorClassName="bg-chart-2" />
                </div>
              </CardContent>
            </Card>

            {storageData?.live?.volumes && (
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-base">
                    <Database className="h-4 w-4 text-chart-5" />
                    Disk Usage
                    {storageData.live.volumes.orphan_count > 0 && (
                      <Badge variant="warning" className="ml-1 text-xs">{storageData.live.volumes.orphan_count} órfãos</Badge>
                    )}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid grid-cols-3 gap-3">
                    <div className="space-y-0.5">
                      <div className="text-xs text-muted-foreground">Volumes</div>
                      <div className="text-lg font-bold text-chart-5 tabular-nums">{storageData.live.volumes.count}</div>
                      <div className="text-[10px] text-muted-foreground">{formatBytes(storageData.live.volumes.total_size)}</div>
                    </div>
                    <div className="space-y-0.5">
                      <div className="text-xs text-muted-foreground">Imagens</div>
                      <div className="text-lg font-bold text-primary tabular-nums">{storageData.live.images.count}</div>
                      <div className="text-[10px] text-muted-foreground">{formatBytes(storageData.live.images.total_size)}</div>
                    </div>
                    <div className="space-y-0.5">
                      <div className="text-xs text-muted-foreground">Reclaimable</div>
                      <div className="text-lg font-bold text-chart-4 tabular-nums">{formatBytes(storageData.live.volumes.reclaimable + storageData.live.images.reclaimable)}</div>
                      <div className="text-[10px] text-muted-foreground">vol + img</div>
                    </div>
                  </div>
                  {storageData.live.volumes.orphan_count > 0 && (
                    <div className="rounded-md border border-warning/30 bg-warning/5 px-3 py-2 text-xs">
                      <div className="flex items-center gap-1.5">
                        <AlertTriangle className="h-3 w-3 text-warning" />
                        <span className="font-medium">{storageData.live.volumes.orphan_count} volumes órfãos</span>
                        <span className="text-muted-foreground">— {formatBytes(storageData.live.volumes.orphan_size)}</span>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        <TabsContent value="services" className="space-y-4">
          {!services || services.length === 0 ? (
            <Card>
              <EmptyState icon={Boxes} message="Nenhum serviço rodando neste node." />
            </Card>
          ) : (
            <Card>
              <CardContent className="p-0">
                <div className="flex items-center gap-3 flex-wrap px-4 py-3 border-b">
                  <div className="relative flex-1 min-w-48">
                    <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                      value={serviceSearch}
                      onChange={(e) => setServiceSearch(e.target.value)}
                      placeholder="Buscar serviço..."
                      className="pl-9"
                    />
                  </div>
                  {serviceSearch && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="gap-1.5 text-muted-foreground"
                      onClick={() => setServiceSearch("")}
                    >
                      <X className="h-3.5 w-3.5" />
                    </Button>
                  )}
                  <span className="text-xs text-muted-foreground ml-auto">
                    {filteredServices.length} de {services.length} serviços
                  </span>
                </div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Serviço</TableHead>
                      <TableHead className="text-center">Containers</TableHead>
                      <TableHead className="w-48 text-right">CPU P95</TableHead>
                      <TableHead className="w-20 text-center">Trend CPU</TableHead>
                      <TableHead className="w-48 text-right">Memória P99</TableHead>
                      <TableHead className="w-20 text-center">Trend Mem</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredServices.map((s) => (
                      <TableRow
                        key={s.service}
                        className="cursor-pointer hover:bg-accent/40 transition-colors duration-150"
                        onClick={() => navigate(`/services/${encodeURIComponent(s.service)}`)}
                      >
                        <TableCell className="font-medium">{s.service}</TableCell>
                        <TableCell className="text-center">
                          <Badge variant="outline" className="border-chart-2/40 text-chart-2">{s.containers}</Badge>
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
                        <TableCell className="text-center">
                          <Sparkline data={(sparklines?.[s.service] ?? []).map(p => p.cpu)} color="var(--chart-1)" />
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
                          <Sparkline data={(sparklines?.[s.service] ?? []).map(p => p.mem)} color="var(--chart-2)" />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="info" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <Info className="h-4 w-4 text-muted-foreground" />
                Informações do Node
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-3">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Node ID</span>
                    <span className="font-mono text-xs">{node.node_id}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Hostname</span>
                    <span className="font-medium">{node.hostname}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Role</span>
                    <span className="font-medium">{node.role}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Availability</span>
                    <span className="font-medium">{node.availability}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Status</span>
                    <span className="font-medium">{node.status}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Address</span>
                    <span className="font-mono text-xs">{node.address}</span>
                  </div>
                </div>

                <div className="space-y-3">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">OS</span>
                    <span className="font-medium">{node.os}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Architecture</span>
                    <span className="font-medium">{node.architecture}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Engine Version</span>
                    <span className="font-medium">{node.engine_version}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Leader</span>
                    <span className="font-medium">{node.is_leader ? "Sim" : "Não"}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Reachability</span>
                    <span className="font-medium">{node.reachability || "N/A"}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Última atualização</span>
                    <span className="font-medium text-xs">{node.updated_at || "N/A"}</span>
                  </div>
                </div>
              </div>

              {Object.keys(node.labels || {}).length > 0 && (
                <div className="mt-6 space-y-2">
                  <div className="text-sm font-medium text-muted-foreground">Labels</div>
                  <div className="flex flex-wrap gap-2">
                    {Object.entries(node.labels).map(([key, value]) => (
                      <Badge key={key} variant="outline" className="text-xs">
                        {key}={value}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
