import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { useNavigate } from "react-router-dom"
import { api } from "@/api/client"
import { Button } from "@/components/ui/button"
import { useRefreshInterval } from "@/hooks/use-refresh"
import { useEventSource } from "@/hooks/use-event-source"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { EmptyState } from "@/components/empty-state"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table"
import { PageHeader } from "@/components/page-header"
import { formatBytes, formatCPU, formatCores } from "@/lib/utils"
import { Server, Cpu, Boxes, ChevronRight, Crown, Search, Filter, ChevronDown, X, Bot } from "lucide-react"
import { useFilterStore } from "@/stores/filter-store"
import { Input } from "@/components/ui/input"
import { DropdownMenu, DropdownMenuTrigger, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel } from "@/components/ui/dropdown-menu"

interface Node {
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

const STATUS_CONFIG: Record<string, { label: string; variant: "default" | "secondary" | "destructive" | "outline"; dot: string }> = {
  ready: { label: "Ready", variant: "outline", dot: "bg-green-500" },
  down: { label: "Down", variant: "destructive", dot: "bg-red-500" },
  disconnected: { label: "Disconnected", variant: "destructive", dot: "bg-red-500" },
  unknown: { label: "Unknown", variant: "secondary", dot: "bg-muted-foreground" },
}

const ROLE_CONFIG: Record<string, { label: string; variant: "default" | "secondary" | "outline" }> = {
  manager: { label: "Manager", variant: "default" },
  worker: { label: "Worker", variant: "secondary" },
}

export default function Nodes() {
  const navigate = useNavigate()
  const refreshInterval = useRefreshInterval()
  const [search, setSearch] = useState("")
  const { nodesRole: roleFilter, nodesStatus: statusFilter, setNodesRole: setRoleFilter, setNodesStatus: setStatusFilter } = useFilterStore()

  // SSE: invalida queries de nodes quando receber evento
  // Nota: ["agents"] incluído porque a tabela mostra status do agent (Active/Stale,
  // containers_count) que vem da query ["agents"] — sem invalidação fica stale com SSE ativo.
  const { isConnected: sseConnected } = useEventSource({
    topic: "nodes",
    invalidateQueries: [["nodes"], ["cluster"], ["agents"]],
  })

  // SSE ativo = zero polling (SSE publica payload completo + reconciliação 30s)
  const fallbackInterval = sseConnected ? false : refreshInterval

  const { data: nodes, isLoading } = useQuery<Node[]>({
    queryKey: ["nodes"],
    queryFn: () => api.get<Node[]>("/nodes"),
    refetchInterval: fallbackInterval,
  })

  // Fase 7 — agents para mostrar status na tabela de nodes
  const { data: agents } = useQuery<Array<{ node_id: string; status: string; containers_count: number }>>({
    queryKey: ["agents"],
    queryFn: () => api.get<Array<{ node_id: string; status: string; containers_count: number }>>("/agents"),
    refetchInterval: fallbackInterval,
  })

  // Mapa node_id -> agent status
  const agentByNode = new Map((agents || []).map((a) => [a.node_id, a]))

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

  if (!nodes || nodes.length === 0) {
    return (
      <div className="space-y-6">
        <PageHeader title="Nodes" description="Nodes do cluster Docker Swarm" />
        <Card>
          <EmptyState icon={Server} message="Nenhum node encontrado. Aguardando coleta de dados..." />
        </Card>
      </div>
    )
  }

  const managers = nodes.filter((n) => n.role === "manager").length
  const workers = nodes.filter((n) => n.role === "worker").length
  const ready = nodes.filter((n) => n.status === "ready").length
  const totalTasks = nodes.reduce((sum, n) => sum + (n.tasks_running || 0), 0)
  const totalCpu = nodes.filter((n) => n.status === "ready").reduce((sum, n) => sum + n.cpu_total, 0)
  const totalMem = nodes.filter((n) => n.status === "ready").reduce((sum, n) => sum + n.mem_total, 0)

  const statCards = [
    { label: "Nodes", value: nodes.length, icon: Server, iconBg: "bg-chart-3/15", iconColor: "text-chart-3", valueColor: "text-chart-3" },
    { label: "Managers", value: managers, icon: Crown, iconBg: "bg-primary/15", iconColor: "text-primary", valueColor: "text-primary" },
    { label: "Workers", value: workers, icon: Boxes, iconBg: "bg-chart-2/15", iconColor: "text-chart-2", valueColor: "text-chart-2" },
    { label: "Tasks", value: totalTasks, icon: Cpu, iconBg: "bg-chart-5/15", iconColor: "text-chart-5", valueColor: "text-chart-5" },
  ]

  const filteredNodes = nodes.filter((n) => {
    if (search) {
      const q = search.toLowerCase()
      if (!n.hostname.toLowerCase().includes(q) && !n.address.toLowerCase().includes(q)) return false
    }
    if (roleFilter !== "all" && n.role !== roleFilter) return false
    if (statusFilter !== "all" && n.status !== statusFilter) return false
    return true
  })

  const maxCpu = Math.max(...nodes.map((n) => n.cpu_p95), 1)
  const maxMem = Math.max(...nodes.map((n) => n.mem_p99), 1)

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
      <PageHeader title="Nodes" description="Nodes do cluster Docker Swarm">
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="gap-1.5">
            <span className="relative flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full rounded-full bg-success animate-pulse-dot" />
            </span>
            {ready}/{nodes.length} ready
          </Badge>
        </div>
      </PageHeader>

      <div className="grid gap-4 md:grid-cols-4">
        {statCards.map((stat) => (
          <Card key={stat.label} className="hover:bg-accent/30 transition-colors">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{stat.label}</CardTitle>
              <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${stat.iconBg}`}>
                <stat.icon className={`h-4.5 w-4.5 ${stat.iconColor}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className={`text-3xl font-bold tabular-nums ${stat.valueColor}`}>{stat.value}</div>
              {stat.label === "Nodes" && (
                <div className="text-xs text-muted-foreground mt-1">
                  {formatCores(totalCpu)} cores CPU · {formatBytes(totalMem)} RAM
                </div>
              )}
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
                placeholder="Buscar hostname ou IP..."
                className="pl-9"
              />
            </div>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="gap-1.5">
                  <Filter className="h-3.5 w-3.5" />
                  Role
                  {roleFilter !== "all" && (
                    <Badge variant="secondary" className="text-[10px] px-1 py-0">{roleFilter === "manager" ? "Manager" : "Worker"}</Badge>
                  )}
                  <ChevronDown className="h-3.5 w-3.5 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start">
                <DropdownMenuLabel>Role</DropdownMenuLabel>
                <DropdownMenuItem onClick={() => setRoleFilter("all")}>
                  {roleFilter === "all" && "✓ "}Todos
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setRoleFilter("manager")}>
                  {roleFilter === "manager" && "✓ "}Manager
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setRoleFilter("worker")}>
                  {roleFilter === "worker" && "✓ "}Worker
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

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
                <DropdownMenuItem onClick={() => setStatusFilter("ready")}>
                  {statusFilter === "ready" && "✓ "}Ready
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("down")}>
                  {statusFilter === "down" && "✓ "}Down
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setStatusFilter("disconnected")}>
                  {statusFilter === "disconnected" && "✓ "}Disconnected
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {(search || roleFilter !== "all" || statusFilter !== "all") && (
              <Button
                variant="ghost"
                size="sm"
                className="gap-1.5 text-muted-foreground"
                onClick={() => { setSearch(""); setRoleFilter("all"); setStatusFilter("all") }}
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            )}

            <span className="text-xs text-muted-foreground ml-auto">
              {filteredNodes.length} de {nodes.length} nodes
            </span>
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Hostname</TableHead>
                <TableHead className="text-center">Role</TableHead>
                <TableHead className="text-center">Status</TableHead>
                <TableHead className="text-center">Tasks</TableHead>
                <TableHead className="text-center">Agent</TableHead>
                <TableHead className="text-center">Containers</TableHead>
                <TableHead className="w-48 text-right">CPU P95</TableHead>
                <TableHead className="w-48 text-right">Memória P99</TableHead>
                <TableHead className="w-12"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredNodes.map((n) => {
                const statusCfg = STATUS_CONFIG[n.status] ?? STATUS_CONFIG.unknown
                const roleCfg = ROLE_CONFIG[n.role] ?? { label: n.role, variant: "outline" as const }
                return (
                  <TableRow
                    key={n.node_id}
                    className="cursor-pointer hover:bg-accent/40 transition-colors duration-150"
                    onClick={() => navigate(`/nodes/${encodeURIComponent(n.node_id)}`)}
                  >
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-1.5">
                        {n.is_leader && <Crown className="h-3.5 w-3.5 text-warning" />}
                        {n.hostname}
                      </div>
                      <div className="text-xs text-muted-foreground font-mono">{n.address}</div>
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge variant={roleCfg.variant}>{roleCfg.label}</Badge>
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge variant={statusCfg.variant} className="gap-1.5">
                        <span className={`h-2 w-2 rounded-full ${statusCfg.dot}`} />
                        {statusCfg.label}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge variant="outline" className="border-chart-5/40 text-chart-5">{n.tasks_running || 0}</Badge>
                    </TableCell>
                    <TableCell className="text-center">
                      {(() => {
                        const agent = agentByNode.get(n.node_id)
                        if (!agent) return <span className="text-xs text-muted-foreground">—</span>
                        const isActive = agent.status === "active"
                        return (
                          <Badge variant={isActive ? "success" : "warning"} className="gap-1.5">
                            <Bot className="h-3 w-3" />
                            {isActive ? "Active" : "Stale"}
                          </Badge>
                        )
                      })()}
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge variant="outline" className="border-chart-2/40 text-chart-2">{n.containers || 0}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center justify-end gap-3">
                        <Progress
                          value={n.cpu_p95}
                          max={maxCpu}
                          indicatorClassName={getCpuColor(n.cpu_p95)}
                          className="w-24 shrink-0"
                        />
                        <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap shrink-0 w-14 text-right">
                          {formatCPU(n.cpu_p95)}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center justify-end gap-3">
                        <Progress
                          value={n.mem_p99}
                          max={maxMem}
                          indicatorClassName={getMemColor(n.mem_p99, maxMem)}
                          className="w-24 shrink-0"
                        />
                        <span className="text-sm tabular-nums text-muted-foreground whitespace-nowrap shrink-0 w-20 text-right">
                          {formatBytes(n.mem_p99)}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
